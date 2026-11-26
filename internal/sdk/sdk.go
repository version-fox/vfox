/*
 *    Copyright 2026 Han Li and contributors
 *
 *    Licensed under the Apache License, Version 2.0 (the "License");
 *    you may not use this file except in compliance with the License.
 *    You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 *    Unless required by applicable law or agreed to in writing, software
 *    distributed under the License is distributed on an "AS IS" BASIS,
 *    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *    See the License for the specific language governing permissions and
 *    limitations under the License.
 */

package sdk

import (
	"cmp"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/pterm/pterm"
	"github.com/schollz/progressbar/v3"

	"github.com/version-fox/vfox/internal/env"
	"github.com/version-fox/vfox/internal/pathmeta"
	"github.com/version-fox/vfox/internal/plugin"
	"github.com/version-fox/vfox/internal/shared/cache"
	"github.com/version-fox/vfox/internal/shared/logger"
	"github.com/version-fox/vfox/internal/shared/util"
	"github.com/version-fox/vfox/internal/shell"
)

const (
	packageInstalledPrefix = "v-"   // Prefix of path for installed SDK packages
	additionRuntimePrefix  = "add-" // Prefix of path for additional runtime packages
)

// InstallOptions controls optional behavior for InstallResolvedWithOptions.
type InstallOptions struct {
	Reinstall   bool
	NoClipboard bool
}

// Sdk interface defines the methods for managing software development kits (SDKs).
type Sdk interface {
	Install(version Version) error                                                    // Install a specific runtime of the SDK
	Reinstall(version Version) error                                                  // Replace an installed runtime with a fresh copy
	InstallResolved(version Version, reinstall bool) (Version, error)                 // Install and report the version that ended up on disk
	InstallResolvedWithOptions(version Version, opts InstallOptions) (Version, error) // InstallResolved with explicit behavior options
	InstallAs(version Version, name Version) (Version, error)                         // Install version as a second copy kept under name
	Uninstall(version Version, fallbacks ...Version) error                            // Uninstall a runtime; the first usable fallback takes over as global
	Available(args []string) ([]*AvailableRuntimePackage, error)                      // List available runtime of the SDK
	EnvKeys(runtimePackage *RuntimePackage) (*env.Envs, error)                        // Get environment variables for a specific runtime of the SDK
	Use(version Version, scope env.UseScope) error                                    // Use a specific runtime in a given scope
	UseWithConfig(version Version, scope env.UseScope, unlink bool) error             // Use with link configuration
	Unuse(scopes ...env.UseScope) error                                               // Unuse the current runtime in a given scope
	GetRuntimePackage(version Version) (*RuntimePackage, error)                       // Get the runtime package for a specific version
	CheckRuntimeExist(version Version) bool                                           // Check if a specific runtime version is installed
	ResolveInstalledVersion(version Version) (Version, error)                         // The installed version a typed one stands for, as `vfox use` selects it
	InstalledList() []Version

	// HasVersion reports whether anything is on disk under the version's name, whole or not: an
	// installation, a link whose target is gone, an interrupted installation. CheckRuntimeExist is
	// the stricter question, whether the version can be used.
	HasVersion(version Version) bool
	// HasLeftovers reports whether files of a version that is not installed any more, or global
	// links into it, are left, because an uninstall could not remove them. Uninstall removes them.
	HasLeftovers(version Version) bool
	// HasVersionDirs reports whether the directory of the SDK holds a version, usable or not, or an
	// interrupted installation of one.
	HasVersionDirs() bool

	// Installation describes an installed version: who owns it and what vfox recorded about it.
	Installation(version Version) (*Installation, error)
	// Link registers an existing SDK directory as a version, without copying or downloading anything.
	Link(version Version, target string, opts LinkOptions) error
	// Adopt writes a receipt for a version that has none. It reports whether it wrote one.
	Adopt(version Version) (bool, error)
	// Diagnose checks installed versions and reports what looks wrong.
	Diagnose(opts DiagnoseOptions) []Finding
	// Detect lists SDKs on this machine that the plugin knows how to find and vfox did not install.
	Detect() ([]DetectedRuntime, error)
	ParseLegacyFile(path string) (Version, error) // Parse legacy version file to get the runtime version
	Current() Version
	Metadata() *Metadata // Get the metadata of the SDK

	// CreateSymlinksForScope creates symlinks for a specific version in the given scope
	CreateSymlinksForScope(version Version, scope env.UseScope) error
	// EnvKeysForScope returns environment variables for a version in the given scope
	// It creates symlinks if needed and returns env vars with paths pointing to symlinks
	EnvKeysForScope(version Version, scope env.UseScope) (*env.Envs, error)

	Close()
}

// impl represents a software development kit managed by the SDK manager.
type impl struct {
	// TODO: check this name what is it
	Name        string                 // Name of the SDK
	envContext  *env.RuntimeEnvContext // Environment context
	plugin      *plugin.Wrapper        // Plugin wrapper
	InstallPath string                 // Installation path of the SDK
}

func (b *impl) Metadata() *Metadata {
	return &Metadata{
		Name:                b.Name,
		PluginMetadata:      b.plugin.Metadata,
		SdkInstalledPath:    b.InstallPath,
		PluginInstalledPath: b.plugin.InstalledPath,
	}
}

func (b *impl) invokeAvailable(args []string) ([]*AvailableRuntimePackage, error) {
	ctx := &plugin.AvailableHookCtx{
		Args: args,
	}
	available, err := b.plugin.Available(ctx)
	if b.plugin.IsNoResultProvided(err) {
		return nil, fmt.Errorf("no available version provided")
	}
	if err != nil {
		return nil, fmt.Errorf("plugin [Available] method error: %w", err)
	}
	return convertAvailableHookResultItem2AvailableRuntimePackage(b.Name, available), nil
}

func (b *impl) Available(args []string) ([]*AvailableRuntimePackage, error) {
	p := b.plugin
	envCtx := b.envContext
	cachePath := filepath.Join(p.InstalledPath, ".available.cache")
	cacheDuration := envCtx.UserConfig.Cache.AvailableHookDuration
	logger.Debugf("Available hook cache duration: %v\n", cacheDuration)
	// Cache is disabled
	if cacheDuration == 0 {
		return b.invokeAvailable(args)
	}

	cacheKey := strings.Join(args, "##")
	if cacheKey == "" {
		cacheKey = "empty"
	}

	fileCache, err := cache.NewFileCache(cachePath)
	if err == nil {
		cacheValue, ok := fileCache.Get(cacheKey)
		logger.Debugf("Available hook cache key: %s, hit: %+v \n", cacheKey, ok)
		if ok {
			var hookResult []*AvailableRuntimePackage
			if err = cacheValue.Unmarshal(&hookResult); err == nil {
				return hookResult, nil
			}
		}
	}
	result, err := b.invokeAvailable(args)
	if err != nil {
		return result, err
	}
	if result == nil {
		fileCache.Set(cacheKey, nil, cache.ExpireTime(cacheDuration))
	}
	if value, err := cache.NewValue(result); err == nil {
		logger.Debugf("Available hook cache set\n")
		fileCache.Set(cacheKey, value, cache.ExpireTime(cacheDuration))
		_ = fileCache.Close()
	}

	return result, nil
}

// Install installs a specific version of the SDK.
// For main runtime, it will be installed to {InstallPath}/v-{main_version}/{main_name}-{main_version}
// For additional runtimes, it will be installed to {InstallPath}/v-{main_version}/add-{addition_name}-{addition_version}
func (b *impl) Install(version Version) error {
	_, err := b.InstallResolved(version, false)
	return err
}

// Reinstall replaces an installed version with a fresh copy, for example after
// the SDK modified itself in place. The version must be given exactly as it is
// installed, and a linked version cannot be reinstalled because vfox does not
// own its files. If the new installation fails the old one is gone, as with any
// failed installation.
func (b *impl) Reinstall(version Version) error {
	_, err := b.InstallResolved(version, true)
	return err
}

// InstallResolved installs version, or with reinstall replaces it, and returns the
// version that is on disk afterwards. That is the version PreInstall resolved the
// request to, which can differ from the one asked for (`lts` becoming `2.0.0`); if
// the version as given is already installed it is returned as is and PreInstall is
// not called. The version is only meaningful when err is nil. Install and Reinstall
// wrap this for callers that do not need it.
func (b *impl) InstallResolved(version Version, reinstall bool) (Version, error) {
	return b.install(version, "", InstallOptions{Reinstall: reinstall})
}

// InstallResolvedWithOptions installs version according to opts and returns the
// resolved version on disk.
func (b *impl) InstallResolvedWithOptions(version Version, opts InstallOptions) (Version, error) {
	return b.install(version, "", opts)
}

// InstallAs installs version as a second copy of it, kept under name, which is what
// `vfox install --as` does. The copy is used, listed and uninstalled by that name, and
// the installation under the version it resolves to is left alone, so both can be on
// disk at once. PreInstall is asked for version, as usual. Nothing installed or linked
// may have name yet, and name cannot be the version the request resolves to. A copy is
// selected by its name only: no typed version reaches it, by prefix or through what a
// plugin is given in PreUse and ParseLegacyFile.
func (b *impl) InstallAs(version Version, name Version) (Version, error) {
	if name == "" {
		return "", fmt.Errorf("a copy of %s needs a name", version)
	}
	return b.install(version, name, InstallOptions{})
}

// install is InstallResolved and InstallAs. Without a name the installation is kept
// under the version PreInstall resolves the request to; with one, under name, and the
// version it resolves to is recorded in the receipt.
func (b *impl) install(version, name Version, opts InstallOptions) (installed Version, installErr error) {
	reinstall := opts.Reinstall
	// PreInstall does not know the name of a copy, so a copy is reinstalled from the
	// version it was asked for, and the reinstall has to resolve to the version it has.
	exact := version
	if reinstall && name == "" {
		if receipt := b.copyReceipt(version); receipt != nil {
			version, name, exact = Version(receipt.Requested), version, Version(receipt.Resolved)
		} else if b.isCopy(version) {
			// Marked as a copy, but the receipt that says what it is a copy of
			// cannot be read, so it cannot be reinstalled as one or as a version.
			_, err := readReceipt(b.packagePath(version))
			if err == nil {
				err = errors.New("its receipt does not say what it is a copy of")
			}
			return "", fmt.Errorf("cannot reinstall %s, a copy made with `vfox install --as`: %w; fix the permissions of %s or remove it", b.Label(version), err, b.packagePath(version))
		}
	}
	named := name != ""
	label := b.Label(version)
	if named {
		label = b.Label(name)
	}
	logger.Debugf("Installing SDK: %s (reinstall: %v)\n", label, reinstall)

	if named {
		if err := ValidCopyName(string(name)); err != nil {
			return "", err
		}
		if !reinstall && b.nameTaken(name) {
			return "", b.nameTakenError(name)
		}
	} else if reinstall {
		if target, linked := b.linkedTarget(version); linked {
			return "", fmt.Errorf("%s is linked to %s and was not installed by vfox; `vfox uninstall %s` removes the link", label, target, label)
		}
	} else if b.CheckRuntimeExist(version) {
		fmt.Printf("%s is already installed\n", label)
		logger.Debugf("SDK %s already exists\n", label)
		return version, nil
	}

	// An installation of this very version that is running is reported at once, as
	// before the SDK had a lock, rather than waited for.
	typed := version
	if named {
		typed = name
	}
	if b.hasPendingInstall(typed) {
		lock, err := acquireInstallLock(b.versionLockPath(typed))
		if err != nil {
			return "", fmt.Errorf("cannot install %s: %w", label, err)
		}
		if err := lock.Close(); err != nil {
			return "", fmt.Errorf("close installation lock: %w", err)
		}
	}

	// PreInstall runs without a lock, so plugin code can take as long as it needs
	// and can install other versions of the SDK, as on main. The SDK lock is taken
	// after it, only to decide on the name the request resolved to and claim it, that
	// is, until its installation marker is written. If the list the plugin was given
	// changed meanwhile, another installation installed or claimed a version, which the
	// plugin may have mapped the request to, so it is asked again with the new list.
	// The lock is released before the download and PostInstall, so other versions of
	// the SDK, including one that a PostInstall installs, install side by side.
	var (
		sdkLock     *os.File
		installInfo *plugin.PreInstallHookResult
		mainSdk     *plugin.PreInstallPackageItem
		sdkVersion  Version
		key         Version
		lock        *os.File
	)
	releaseSDKLock := func() error {
		if sdkLock == nil {
			return nil
		}
		err := sdkLock.Close()
		sdkLock = nil
		if err != nil {
			return fmt.Errorf("close installation lock: %w", err)
		}
		return nil
	}
	defer func() {
		if err := releaseSDKLock(); err != nil {
			installErr = errors.Join(installErr, err)
		}
	}()
	// Only a plain install is given the installed versions. A reinstall replaces the
	// version as it was typed, and --as makes a second copy on purpose, so for both
	// the plugin resolves the version as it would without the list.
	listed := !reinstall && !named
	requestLabel := label
	for {
		label = requestLabel
		installedSdks := map[string]*plugin.InstalledPackageItem{}
		var installing map[Version]bool
		if listed {
			installedSdks, installing = b.preInstallList()
		}
		ctx := &plugin.PreInstallHookCtx{
			Version:       string(version),
			InstalledSdks: installedSdks,
		}
		logger.Debugf("Calling PreInstall hook for %s\n", label)
		var err error
		installInfo, err = b.plugin.PreInstall(ctx)
		if b.plugin.IsNoResultProvided(err) {
			return "", fmt.Errorf("no installable runtime provided")
		}
		if err != nil {
			logger.Debugf("PreInstall hook failed for %s: %v\n", label, err)
			return "", fmt.Errorf("plugin [PreInstall] method error: %w", err)
		}
		if installInfo == nil || installInfo.PreInstallPackageItem == nil {
			return "", fmt.Errorf("no information about the current version")
		}

		mainSdk = installInfo.PreInstallPackageItem
		mainSdk.Name = b.plugin.Name
		if mainSdk.Version == "" {
			// Without a version there is nothing to name the installation after: it
			// would land in v- and be recorded with an empty version.
			return "", fmt.Errorf("plugin [PreInstall] method error: no version number provided")
		}

		sdkVersion = Version(mainSdk.Version)
		// The installation is kept under its name, or under the version it resolved to.
		key = sdkVersion
		if named {
			key = name
		}
		// A second check is required because the plug-in may change the version number,
		// for example, latest is resolved to a specific version number.
		label = b.Label(key)
		logger.Debugf("Resolved version: %s\n", sdkVersion)
		if reinstall && sdkVersion != exact {
			if named {
				return "", fmt.Errorf("%s is a copy of %s, which now resolves to %s, not the %s it was installed as; uninstall it and install it again with --as to get %s", label, version, sdkVersion, exact, sdkVersion)
			}
			return "", fmt.Errorf("%s resolves to %s; give the exact installed version to reinstall it", b.Label(version), label)
		}
		if named && sdkVersion == name {
			return "", fmt.Errorf("%s resolves to %s, so a copy of it cannot be named %s: name the copy differently", b.Label(version), sdkVersion, name)
		}

		sdkLock, err = waitForInstallLock(filepath.Join(b.InstallPath, sdkInstallLockName), func() {
			pterm.Printf("Waiting for another installation of %s to finish...\n", b.Name)
		})
		if errors.Is(err, errInstallLockStillHeld) {
			return "", fmt.Errorf("cannot install %s: another installation of %s is still running after %s; try again when it is done", label, b.Name, installLockWait)
		}
		if err != nil {
			return "", fmt.Errorf("cannot install %s: %w", label, err)
		}
		if listed {
			// A version that went from installed to being reinstalled keeps its
			// key, so the versions being installed are compared as well.
			if now, nowInstalling := b.preInstallList(); !sameVersions(installedSdks, now) || !sameInstalling(installing, nowInstalling) {
				logger.Debugf("Installed versions changed during PreInstall of %s; asking again\n", requestLabel)
				if err := releaseSDKLock(); err != nil {
					return "", err
				}
				continue
			}
		}
		// The list given to PreInstall leaves copies out, but a plugin can return any
		// name. A copy is selected only by its own name, so it cannot stand for the
		// version asked for.
		if !named && (b.isCopy(sdkVersion) || b.copyBeingMade(sdkVersion)) {
			return "", fmt.Errorf("plugin [PreInstall] method error: %s was resolved to %s, but %s is a copy made with `vfox install --as` and is selected only by its own name", b.Label(version), label, label)
		}
		if err := b.inspectError(key); err != nil {
			return "", fmt.Errorf("cannot tell whether %s is installed: %w; fix the permissions of %s or remove it", label, err, b.packagePath(key))
		}
		// A request mapped to a version that another installation is still
		// installing waits for that installation without the SDK lock, so the
		// installation it waits for never waits on it, and is decided again after.
		if installing[key] && key != typed {
			if err := releaseSDKLock(); err != nil {
				return "", err
			}
			if err := b.waitForInstallation(requestLabel, key); err != nil {
				return "", err
			}
			continue
		}
		lock, err = acquireInstallLock(b.versionLockPath(key))
		if err != nil {
			return "", fmt.Errorf("cannot install %s: %w", label, err)
		}
		break
	}
	defer func() {
		if err := lock.Close(); err != nil {
			installErr = errors.Join(installErr, fmt.Errorf("close installation lock: %w", err))
		}
	}()
	newDirPath := b.packagePath(key)
	// Recheck while holding the lock, before cleaning an interrupted attempt.
	if named {
		if !reinstall && b.nameTaken(key) {
			return "", b.nameTakenError(key)
		}
	} else if !reinstall && b.CheckRuntimeExist(key) {
		fmt.Printf("%s is already installed\n", label)
		logger.Debugf("SDK %s already exists after version resolution\n", label)
		return key, nil
	}
	success := false
	logger.Debugf("Installing to path: %s\n", newDirPath)
	// Keep the marker outside the payload so failed cleanup cannot erase it
	// before removing files still held open by a child process on Windows.
	markerPath := b.installMarkerPath(key)
	if err := os.WriteFile(markerPath, nil, 0600); err != nil {
		return "", fmt.Errorf("mark installation in progress: %w", err)
	}
	// Ordinary errors clean up immediately. Process termination leaves the
	// marker behind and releases the OS lock; the next attempt can retry.
	defer func() {
		if !success {
			if err := os.RemoveAll(newDirPath); err != nil {
				installErr = errors.Join(installErr, fmt.Errorf("clean incomplete installation: %w", err))
			} else if err := os.Remove(b.copyMarkerPath(key)); err != nil && !os.IsNotExist(err) {
				installErr = errors.Join(installErr, fmt.Errorf("remove copy marker: %w", err))
			} else if err := os.Remove(markerPath); err != nil && !os.IsNotExist(err) {
				installErr = errors.Join(installErr, fmt.Errorf("remove installation marker: %w", err))
			}
		}
	}()
	// A copy is marked beside its directory, where the mark can be read when the
	// directory cannot be entered and while the copy is being installed. Installing
	// a version under the name of a copy that is gone drops a mark it left.
	if named {
		if err := os.WriteFile(b.copyMarkerPath(key), nil, 0644); err != nil {
			return "", fmt.Errorf("mark copy: %w", err)
		}
	} else if err := os.Remove(b.copyMarkerPath(key)); err != nil && !os.IsNotExist(err) {
		return "", fmt.Errorf("remove copy marker: %w", err)
	}
	// The name is claimed, so other installations of the SDK can go ahead.
	if err := releaseSDKLock(); err != nil {
		return "", err
	}
	if err := os.RemoveAll(newDirPath); err != nil {
		return "", fmt.Errorf("remove interrupted installation: %w", err)
	}
	installedPackage := make(map[string]*plugin.InstalledPackageItem)

	// The main runtime is where mainRuntimePath looks for it, so it is named after the
	// installation, which for a copy is not the version the plugin resolved to.
	path, err := b.preInstallSdk(mainSdk, b.mainRuntimePath(key))

	if err != nil {
		return "", err
	}
	installedPackage[mainSdk.Name] = &plugin.InstalledPackageItem{
		Name:    mainSdk.Name,
		Version: string(key),
		Path:    path,
		Note:    mainSdk.Note,
	}
	if len(installInfo.Addition) > 0 {
		pterm.Printf("There are %d additional files that need to be downloaded...\n", len(installInfo.Addition))
		for _, oSdk := range installInfo.Addition {
			path, err = b.preInstallSdk(oSdk, filepath.Join(newDirPath, additionDirName(oSdk)))
			if err != nil {
				return "", err
			}
			installedPackage[oSdk.Name] = &plugin.InstalledPackageItem{
				Name:    oSdk.Name,
				Version: oSdk.Version,
				Path:    path,
				Note:    oSdk.Note,
			}
		}
	}
	postCtx := &plugin.PostInstallHookCtx{
		RootPath: newDirPath,
		SdkInfo:  installedPackage,
	}
	if b.plugin.HasFunction("PostInstall") {
		logger.Debugf("Running post-installation steps...\n")
		err = withPostInstallLock(b.versionLockPath(key), func() error {
			return b.plugin.PostInstall(postCtx)
		})
		if err != nil {
			return "", fmt.Errorf("plugin [PostInstall] method error: %w", err)
		}
	}
	// The receipt goes in before the marker comes off, so a finished
	// installation always has one.
	b.recordInstall(version, key, installInfo, installedPackage, newDirPath)
	if err := os.Remove(markerPath); err != nil {
		return "", fmt.Errorf("finish installation: %w", err)
	}
	success = true
	pterm.Printf("Install %s success! \n", pterm.LightGreen(label))
	logger.Debugf("SDK %s installed successfully\n", label)
	useCommand := fmt.Sprintf("vfox use %s", label)
	pterm.Printf("Please use `%s` to use it.\n", pterm.LightBlue(useCommand))

	// Copy command to clipboard in interactive TTY mode unless suppressed (e.g. non-interactive exec).
	if !opts.NoClipboard && util.IsTTY() {
		if err := util.CopyToClipboard(useCommand); err == nil {
			pterm.Printf("%s\n", pterm.LightYellow("Copied to clipboard, you can paste it now."))
		}
		// Silently ignore clipboard errors (not supported, utility not found, etc.)
	}

	return key, nil
}

// copyReceipt returns the receipt of a copy made by InstallAs, and nil for anything
// else: only a copy records the version its request resolved to.
func (b *impl) copyReceipt(name Version) *Receipt {
	receipt, err := readReceipt(b.packagePath(name))
	if err != nil {
		logger.Debugf("Ignoring unreadable receipt of %s: %v\n", b.Label(name), err)
		return nil
	}
	if !receipt.IsCopy() {
		return nil
	}
	return receipt
}

// isCopy reports whether an installed version is a copy made with InstallAs.
// Its mark beside the directory is read first, so that a copy whose directory
// cannot be entered is still known as one.
func (b *impl) isCopy(version Version) bool {
	// What an interrupted or running `install --as` left is not a copy yet: a
	// plain install of the name replaces it, or waits for it, as on main.
	if b.hasPendingInstall(version) {
		return false
	}
	if _, err := os.Lstat(b.copyMarkerPath(version)); err == nil {
		if _, err := os.Lstat(b.packagePath(version)); err == nil {
			return true
		}
	}
	return b.copyReceipt(version) != nil
}

// preInstallList is what a plain install gives PreInstall: the installed versions
// without the copies, those whose directory cannot be entered (a request resolved to
// one is refused), and the versions another installation is installing now, which
// it returns too (a request mapped to one waits for it).
func (b *impl) preInstallList() (map[string]*plugin.InstalledPackageItem, map[Version]bool) {
	installedSdks := b.matchableInstalledItems()
	for _, version := range b.uninspectableVersions() {
		installedSdks[string(version)] = b.installedItem(version)
	}
	installing := b.installingVersions()
	for version := range installing {
		installedSdks[string(version)] = b.installedItem(version)
	}
	return installedSdks, installing
}

// sameVersions reports whether two lists from preInstallList name the same versions.
func sameVersions(a, b map[string]*plugin.InstalledPackageItem) bool {
	if len(a) != len(b) {
		return false
	}
	for version := range a {
		if _, ok := b[version]; !ok {
			return false
		}
	}
	return true
}

// sameInstalling reports whether two sets of versions being installed, from
// preInstallList, are the same.
func sameInstalling(a, b map[Version]bool) bool {
	if len(a) != len(b) {
		return false
	}
	for version := range a {
		if !b[version] {
			return false
		}
	}
	return true
}

// copyBeingMade reports whether another vfox is making a copy under version with
// `install --as` now. isCopy does not count it yet, and installingVersions leaves it
// out, so a plain install resolved to its name is refused as it is once the copy is
// made, rather than failing on the lock of the copy.
func (b *impl) copyBeingMade(version Version) bool {
	if _, err := os.Lstat(b.copyMarkerPath(version)); err != nil {
		return false
	}
	return b.hasPendingInstall(version) && b.installationRunning(version)
}

// copyMarkerPath is the mark InstallAs leaves beside the directory of a copy.
func (b *impl) copyMarkerPath(version Version) string {
	return filepath.Join(b.InstallPath, ".v-"+string(version)+".copy")
}

// installingVersions returns the versions that another installation is installing
// now: those with an installation marker whose lock is held. Copies are left out,
// as installed ones are. With the SDK lock held it sees every name claimed so far,
// because a name is claimed under that lock.
func (b *impl) installingVersions() map[Version]bool {
	entries, err := os.ReadDir(b.InstallPath)
	if err != nil {
		return nil
	}
	installing := make(map[Version]bool)
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasPrefix(name, ".v-") || !strings.HasSuffix(name, ".installing") {
			continue
		}
		version := Version(strings.TrimSuffix(strings.TrimPrefix(name, ".v-"), ".installing"))
		if _, err := os.Lstat(b.copyMarkerPath(version)); err == nil {
			continue
		}
		if b.installationRunning(version) {
			installing[version] = true
		}
	}
	return installing
}

// installationRunning reports whether another installation holds the lock of
// version. A marker whose lock is free was left by an interrupted installation.
func (b *impl) installationRunning(version Version) bool {
	file, err := openInstallLock(b.versionLockPath(version))
	if err != nil {
		return false
	}
	defer func() { _ = file.Close() }() // Closing releases the lock if it was taken here.
	err = lockInstallFile(file)
	return err != nil && installLockHeld(err)
}

// waitForInstallation waits, up to installLockWait, for the installation of version
// that another vfox is running. requested names the install that waits.
func (b *impl) waitForInstallation(requested string, version Version) error {
	label := b.Label(version)
	path := b.versionLockPath(version)
	if inPostInstallOf(path) {
		return fmt.Errorf("cannot install %s: it resolves to %s, and this install runs in the PostInstall of %s, so it would wait for itself", requested, label, label)
	}
	lock, err := waitForInstallLock(path, func() {
		pterm.Printf("Waiting for the installation of %s to finish...\n", label)
	})
	if errors.Is(err, errInstallLockStillHeld) {
		return fmt.Errorf("cannot install %s: it resolves to %s, and %s is still being installed after %s; try again when it is done", requested, label, label, installLockWait)
	}
	if err != nil {
		return fmt.Errorf("cannot install %s: %w", requested, err)
	}
	if err := lock.Close(); err != nil {
		return fmt.Errorf("close installation lock: %w", err)
	}
	return nil
}

// matchableVersions is InstalledList without the copies: the installed versions a
// typed version can stand for. A copy is selected only by its own name.
func (b *impl) matchableVersions() []Version {
	var versions []Version
	for _, version := range b.InstalledList() {
		if !b.isCopy(version) {
			versions = append(versions, version)
		}
	}
	return versions
}

// nameTaken reports whether a version is installed or linked under name, a link
// whose target has gone included. What an interrupted installation left behind does
// not count: installing under the name replaces it.
func (b *impl) nameTaken(name Version) bool {
	return b.CheckRuntimeExist(name) || b.brokenLink(name)
}

// nameTakenError says what holds name and how to free it.
func (b *impl) nameTakenError(name Version) error {
	label := b.Label(name)
	if target, linked := b.linkedTarget(name); linked {
		return fmt.Errorf("%s is already linked to %s; choose another name for the copy, or `vfox uninstall %s` removes the link", label, target, label)
	}
	return fmt.Errorf("%s is already installed; choose another name for the copy, or `vfox uninstall %s` removes it first", label, label)
}

// recordInstall leaves the receipt of a finished installation in its version
// directory. A receipt only informs later checks, so failing to write one is
// logged and never fails the installation.
func (b *impl) recordInstall(requested, version Version, info *plugin.PreInstallHookResult, installed map[string]*plugin.InstalledPackageItem, packagePath string) {
	receipt := b.newReceipt(version, OriginInstalled)
	receipt.Requested = string(requested)
	// A copy kept under a name records the version the request resolved to.
	if resolved := Version(info.PreInstallPackageItem.Version); resolved != version {
		receipt.Resolved = string(resolved)
	}
	receipt.Sources = append(receipt.Sources, receiptSource(info.PreInstallPackageItem))
	for _, addition := range info.Addition {
		receipt.Sources = append(receipt.Sources, receiptSource(addition))
	}
	mainPath := ""
	if main := installed[b.sdkName()]; main != nil {
		mainPath = main.Path
	}
	facts, err := b.fingerprint(version, installed, mainPath)
	if err != nil {
		logger.Debugf("No facts recorded for %s: %v\n", b.Label(version), err)
	}
	receipt.Facts = facts
	if err := writeReceipt(packagePath, receipt); err != nil {
		logger.Debugf("Failed to write the receipt of %s: %v\n", b.Label(version), err)
	}
}

func (b *impl) moveLocalFile(info *plugin.PreInstallPackageItem, targetPath string) error {
	pterm.Printf("Moving %s to %s...\n", util.RedactSourceURL(info.Path), targetPath)
	if err := util.MoveFiles(info.Path, targetPath); err != nil {
		var pathErr *fs.PathError
		if errors.As(err, &pathErr) {
			// A PreInstall source can be a URL with credentials, and the
			// PathError prints its Path, so the copy the user sees is redacted.
			redacted := *pathErr
			redacted.Path = util.RedactSourceURL(pathErr.Path)
			err = &redacted
		}
		return fmt.Errorf("failed to move file, err:%w", err)
	}
	return nil
}

func (b *impl) moveRemoteFile(info *plugin.PreInstallPackageItem, targetPath string) error {
	u, err := url.Parse(info.Path)
	label := info.Label()
	if err != nil {
		return util.RedactError(err)
	}
	filePath, err := b.Download(u, info.Headers)
	if err != nil {
		return fmt.Errorf("failed to download %s file, err:%w", label, err)
	}
	defer func() {
		// del cache file
		_ = os.Remove(filePath)
	}()
	checksum := info.Checksum()
	pterm.Printf("Verifying checksum %s...\n", checksum.Value)
	verify := checksum.Verify(filePath)
	if !verify {
		fmt.Printf("Checksum error, file: %s\n", filePath)
		return errors.New("checksum error")
	}
	decompressor := util.NewDecompressor(filePath)
	if decompressor == nil {
		// If it is not a compressed file, move file to the corresponding sdk directory,
		// and the rest be handled by the PostInstall function.
		if err = util.MoveFiles(filePath, targetPath); err != nil {
			return fmt.Errorf("failed to move file, err:%w", err)
		}
		return nil
	}
	pterm.Printf("Unpacking %s...\n", filePath)
	err = decompressor.Decompress(targetPath)
	if err != nil {
		return fmt.Errorf("unpack failed, err:%w", err)
	}
	return nil
}

// additionDirName names the directory of an additional runtime inside its version
// directory: add-{name}-{version}, or add-{name} when it has no version. The main
// runtime is named by mainRuntimePath.
func additionDirName(info *plugin.PreInstallPackageItem) string {
	if info.Version == "" {
		return additionRuntimePrefix + info.Name
	}
	return additionRuntimePrefix + info.Name + "-" + info.Version
}

func (b *impl) preInstallSdk(info *plugin.PreInstallPackageItem, sdkDestPath string) (string, error) {
	pterm.Printf("Preinstalling %s...\n", info.Name+"@"+info.Version)

	path := sdkDestPath
	if !util.FileExists(path) {
		if err := os.MkdirAll(path, pathmeta.ReadWriteAuth); err != nil {
			return "", fmt.Errorf("failed to create directory, err:%w", err)
		}
	}
	if info.Path == "" {
		return path, nil
	}
	if strings.HasPrefix(info.Path, "https://") || strings.HasPrefix(info.Path, "http://") {
		if err := b.moveRemoteFile(info, path); err != nil {
			return "", err
		}
		return path, nil
	} else {
		if err := b.moveLocalFile(info, path); err != nil {
			return "", err
		}
		return path, nil
	}
}

// Uninstall removes version. If it is the global version, the first of
// fallbacks that can be used takes over, or with no fallbacks the global setting
// is removed. The global switch runs before PreUninstall, so the hook never runs
// for a version the global setting cannot leave.
//
// If the global version is another one that is linked into the directory of
// version, nothing is removed. Other versions linked into it stop working, and a
// warning names each.
//
// The removal takes effect in one step, by renaming the directory of the
// version. If anything fails up to and including that step, the version stays
// installed, and the global entry and links are set back to what they were. If
// that fails too, the error says what the global version is. After the rename,
// the version is gone, and what can still fail is removing the global links into
// it (ErrLinksLeft) and deleting its files (ErrFilesLeft). The next Uninstall of
// the same version does what is left, whether it was installed again or not.
func (b *impl) Uninstall(version Version, fallbacks ...Version) (err error) {
	label := b.Label(version)
	logger.Debugf("Uninstalling SDK: %s\n", label)

	// Without the global config there is no telling whether it names the version, or
	// one that is linked into its directory.
	globalEntry, err := b.globalEntry()
	if err != nil {
		return fmt.Errorf("could not read the global version, so nothing was removed: %w", err)
	}

	installed := b.CheckRuntimeExist(version)
	// A link whose target has gone away no longer counts as installed, but it
	// is still there to be removed.
	if !installed && !b.brokenLink(version) {
		linksLeft, err := b.linksLeft(version, globalEntry)
		if err != nil {
			return fmt.Errorf("could not read the global links: %w", err)
		}
		files := b.leftovers(version)
		if linksLeft || len(files) > 0 {
			if linksLeft {
				err = b.removeLinksTo(version)
			}
			err = errors.Join(err, b.deleteLeftovers(version, files))
			if err == nil {
				pterm.Printf("Removed what was left of %s.\n", label)
			}
			return err
		}
		logger.Debugf("SDK %s not installed\n", label)
		return fmt.Errorf("%s is not installed", pterm.Red(label))
	}
	path := b.packagePath(version)
	logger.Debugf("Uninstalling from path: %s\n", path)

	linkedTo, linked := b.linkedTarget(version)
	if linked {
		if err = b.checkLink(version); err != nil {
			return err
		}
	}
	var sdkPackage *RuntimePackage
	if installed {
		sdkPackage, err = b.GetRuntimePackage(version)
		if err != nil {
			logger.Debugf("Failed to get runtime package for %s: %v\n", label, err)
			return
		}
	}

	// Versions linked into the directory of this one break when it goes. The global
	// version must not.
	dependents := b.dependents(version)
	if globalEntry != nil {
		if global := Version(globalEntry.Version); slices.Contains(dependents, global) {
			return fmt.Errorf("the global version %s is linked into the directory of %s, so nothing was removed: switch or unset the global version first (vfox unuse -g %s)", b.Label(global), label, strings.ToLower(b.Name))
		}
	}

	// If the global scope asks for the version to be uninstalled, it has to stop
	// doing so. A version only a project or session asks for leaves the global
	// setting alone.
	isGlobal := globalEntry != nil && Version(globalEntry.Version) == version
	var before globalState
	var takesOver Version
	if isGlobal {
		before.entry = globalEntry
		if before.links, err = b.linksOfScope(env.Global); err != nil {
			return fmt.Errorf("could not read the global links, so nothing was removed: %w", err)
		}
		if takesOver, err = b.releaseGlobal(version, fallbacks); err != nil {
			return err
		}
	}
	// undo puts the global setting back for a version that stays installed.
	undo := func(cause error) error {
		if !isGlobal {
			return cause
		}
		if undoErr := b.restoreGlobal(version, takesOver, before); undoErr != nil {
			now := "there is no global version now"
			if current := b.globalVersion(); current != "" {
				now = "the global version is " + b.Label(current) + " now"
			}
			return errors.Join(cause, fmt.Errorf("%s is still installed, but %s: it could not be set back, `vfox use -g %s` does it: %w", label, now, label, undoErr))
		}
		return cause
	}

	// vfox does not own the files of a linked version, so the plugin has
	// nothing of its own to clean up in them.
	hookRan := ""
	if installed && !linked && b.plugin.HasFunction("PreUninstall") {
		logger.Debugf("Calling PreUninstall hook for %s\n", label)
		// Give the plugin a chance before actually uninstalling targeted version.
		main := convertRuntime2InstalledPackageItem(sdkPackage.Runtime)
		sdkInfo := map[string]*plugin.InstalledPackageItem{}
		for _, addition := range sdkPackage.Additions {
			sdkInfo[addition.Name] = convertRuntime2InstalledPackageItem(addition)
		}
		sdkInfo[main.Name] = main
		preUninstallCtx := &plugin.PreUninstallHookCtx{
			Main:    main,
			SdkInfo: sdkInfo,
		}
		err = b.plugin.PreUninstall(preUninstallCtx)
		if err != nil {
			logger.Debugf("PreUninstall hook failed for %s: %v\n", label, err)
			return undo(err)
		}
		hookRan = ", but its PreUninstall hook has already run"
	}

	removing := b.freeLeftoverPath(version)
	logger.Debugf("Moving %s to %s\n", path, removing)
	if err = os.Rename(path, removing); err != nil {
		return undo(fmt.Errorf("could not remove %s, it is still installed%s: %w", label, hookRan, err))
	}

	// The global links that pointed into the version now dangle: those of the
	// version itself, or of additions the global version does not have. The version
	// is gone, so if one cannot be removed there is nothing to go back to: the rest
	// is done all the same, and the next uninstall tries that link again.
	linkErr := b.removeLinksTo(version)
	// The mark of a copy goes with it. One left behind is dropped by the next plain
	// install under the name.
	if err := os.Remove(b.copyMarkerPath(version)); err != nil && !os.IsNotExist(err) {
		linkErr = errors.Join(linkErr, fmt.Errorf("remove copy marker: %w", err))
	}

	// Files that an earlier uninstall of the version left go as well.
	paths := []string{removing}
	for _, left := range b.leftovers(version) {
		if left != removing {
			paths = append(paths, left)
		}
	}
	leftErr := errors.Join(linkErr, b.deleteLeftovers(version, paths))
	switch {
	case leftErr != nil:
		pterm.Printf("%s is no longer installed.\n", label)
	case linked:
		pterm.Printf("Unlinked %s successfully! %s was left untouched.\n", label, linkedTo)
	default:
		pterm.Printf("Uninstalled %s successfully!\n", label)
	}
	for _, dependent := range dependents {
		pterm.Warning.Printf("%s no longer works: it is linked into the directory of %s.\n", b.Label(dependent), label)
	}
	if takesOver != "" {
		pterm.Printf("Auto switch to the other version: %s.\n", b.Label(takesOver))
		b.reportFindings(takesOver, env.IsHookEnv())
	} else if isGlobal {
		b.printUnset()
	}
	return leftErr
}

// globalState is what the global scope holds for the SDK: its config entry and
// the links of the scope.
type globalState struct {
	entry *pathmeta.ToolConfig
	links map[string]string
}

// globalEntry returns the global entry of the SDK as it is, attributes included,
// or nil if it has none.
func (b *impl) globalEntry() (*pathmeta.ToolConfig, error) {
	vfoxToml, err := b.envContext.LoadVfoxTomlByScope(env.Global)
	if err != nil {
		return nil, err
	}
	if entry, ok := vfoxToml.GetToolConfig(b.Name); ok {
		return entry, nil
	}
	return nil, nil
}

// releaseGlobal makes the global setting, which names version, stop doing so: the
// first of fallbacks that can be used takes over and is returned, or with no
// fallbacks the setting is removed and "" is returned. If it fails, the global
// setting is as it was.
func (b *impl) releaseGlobal(version Version, fallbacks []Version) (Version, error) {
	if len(fallbacks) > 0 {
		return b.switchGlobal(version, fallbacks)
	}
	var registryEnvs *env.Envs
	if runtime.GOOS == "windows" {
		var err error
		// Best effort: a broken version has no environment to look up, and its
		// registry entries are only stale.
		if registryEnvs, err = b.EnvKeysForScope(version, env.Global); err != nil {
			logger.Debugf("Failed to get the registry environment of %s: %v\n", b.Label(version), err)
		}
	}
	vfoxToml, err := b.envContext.LoadVfoxTomlByScope(env.Global)
	if err == nil {
		vfoxToml.RemoveTool(b.Name)
		err = vfoxToml.Save()
	}
	if err != nil {
		return "", fmt.Errorf("the global version could not be unset, so nothing was removed: %w", err)
	}
	// Best effort, as in unuse: the entries are only stale, and the config
	// decides which version is in use.
	if err := env.RemoveEnvsFromRegistry(registryEnvs); err != nil {
		pterm.Warning.Printf("Failed to remove the global registry environment of %s: %v\n", b.Label(version), err)
	}
	return "", nil
}

// restoreGlobal undoes releaseGlobal for version, which tookOver had replaced: the
// links and the entry become what they were before. As in apply, the order is
// links, config, registry, and if the links or the config fail, the links go to
// what the config still asks for.
func (b *impl) restoreGlobal(version, tookOver Version, before globalState) error {
	after, err := b.linksOfScope(env.Global)
	if err != nil {
		return err
	}
	if err = b.restoreLinks(before.links, env.Global); err == nil {
		err = b.restoreEntry(before.entry)
	}
	if err != nil {
		// Without a version that took over, the config asks for none.
		if tookOver == "" {
			return errors.Join(err, b.removeLinksInto(b.packagePath(version), env.Global))
		}
		return errors.Join(err, b.restoreLinks(after, env.Global))
	}
	if runtime.GOOS == "windows" {
		// Best effort, as in apply: the registry only mirrors the config for
		// programs that do not start from a shell with the hook.
		if tookOver != "" {
			envs, err := b.EnvKeysForScope(tookOver, env.Global)
			if err == nil {
				err = env.RemoveEnvsFromRegistry(envs)
			}
			if err != nil {
				logger.Debugf("Failed to remove the registry environment of %s: %v\n", b.Label(tookOver), err)
			}
		}
		envs, err := b.EnvKeysForScope(version, env.Global)
		if err != nil {
			// A broken version has no environment, so it had no entries to restore.
			logger.Debugf("Failed to get the registry environment of %s: %v\n", b.Label(version), err)
		} else if err := env.ApplyEnvsToRegistry(envs); err != nil {
			pterm.Warning.Printf("Failed to apply global registry environment: %v\n", err)
		}
	}
	return nil
}

// restoreEntry writes entry back as the global entry of the SDK.
func (b *impl) restoreEntry(entry *pathmeta.ToolConfig) error {
	vfoxToml, err := b.envContext.LoadVfoxTomlByScope(env.Global)
	if err != nil {
		return err
	}
	vfoxToml.SetToolWithAttr(b.Name, entry.Version, entry.Attr)
	return vfoxToml.Save()
}

// leftoverPath returns the path that the directory of version is moved to before
// its files are deleted. It is outside the names of versions, and n tells it from
// the files that are left of an earlier removal.
func (b *impl) leftoverPath(version Version, n int) string {
	name := ".v-" + string(version) + ".removing"
	if n > 0 {
		name += "." + strconv.Itoa(n)
	}
	return filepath.Join(b.InstallPath, name)
}

func (b *impl) freeLeftoverPath(version Version) string {
	for n := 0; ; n++ {
		path := b.leftoverPath(version, n)
		if _, err := os.Lstat(path); err != nil {
			return path
		}
	}
}

// leftovers returns the directories with the files of version that an uninstall
// could not delete.
func (b *impl) leftovers(version Version) []string {
	entries, err := os.ReadDir(b.InstallPath)
	if err != nil {
		return nil
	}
	first := filepath.Base(b.leftoverPath(version, 0))
	var paths []string
	for _, entry := range entries {
		n, numbered := strings.CutPrefix(entry.Name(), first+".")
		if _, err := strconv.Atoi(n); entry.Name() == first || numbered && err == nil {
			paths = append(paths, filepath.Join(b.InstallPath, entry.Name()))
		}
	}
	return paths
}

// deleteLeftovers deletes paths, the directories that version was moved to. It
// goes on after one that cannot be deleted, and returns the errors of all of them.
func (b *impl) deleteLeftovers(version Version, paths []string) error {
	var errs []error
	for _, path := range paths {
		if err := deletePackage(path); err != nil {
			errs = append(errs, filesLeftError(b.Label(version), path, err))
		}
	}
	return errors.Join(errs...)
}

// linksLeft reports whether an uninstall of version left global links that point
// into it. Such links are reported only when the directory of the version is gone
// and the global config, whose entry is entry, does not name the version. A version
// it does name was not uninstalled: its directory went some other way.
func (b *impl) linksLeft(version Version, entry *pathmeta.ToolConfig) (bool, error) {
	path := b.packagePath(version)
	if _, err := os.Lstat(path); !os.IsNotExist(err) || entry != nil && Version(entry.Version) == version {
		return false, nil
	}
	links, err := b.linksInto(path, env.Global)
	return len(links) > 0, err
}

// removeLinksTo removes the global links that point into the directory of version,
// which is gone.
func (b *impl) removeLinksTo(version Version) error {
	if err := b.removeLinksInto(b.packagePath(version), env.Global); err != nil {
		label := b.Label(version)
		return fmt.Errorf("%w %s: %w (uninstall %s again to remove them)", ErrLinksLeft, label, err, label)
	}
	return nil
}

func filesLeftError(label, path string, err error) error {
	return fmt.Errorf("%w %s: %w (they are in %s, uninstall %s again to delete them)", ErrFilesLeft, label, err, path, label)
}

// deletePackage deletes the directory of a version. The links in it go first, so
// that nothing behind them is deleted.
func deletePackage(path string) error {
	if env.IsDirSymlink(path) {
		return env.RemoveDirSymlink(path)
	}
	entries, err := os.ReadDir(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	for _, entry := range entries {
		link := filepath.Join(path, entry.Name())
		if !env.IsDirSymlink(link) {
			continue
		}
		if err := env.RemoveDirSymlink(link); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove link %s: %w", link, err)
		}
	}
	return os.RemoveAll(path)
}

// switchGlobal makes the first of candidates that can be used the global version,
// in the place of removed, and returns it. It warns of each one it had to skip. If
// none can be used, the global version stays as it was, and the error names each
// candidate.
func (b *impl) switchGlobal(removed Version, candidates []Version) (Version, error) {
	var errs []error
	for _, candidate := range candidates {
		v, err := b.resolveInstalled(candidate, env.Global)
		if err == nil && v == removed {
			err = errors.New("it resolves to the version that is being removed")
		}
		if err == nil && b.dependsOn(v, removed) {
			err = fmt.Errorf("it is linked into the directory of %s, which is being removed", b.Label(removed))
		}
		if err == nil {
			_, err = b.apply(v, env.Global, false)
		}
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", b.Label(candidate), err))
			continue
		}
		for _, skipped := range errs {
			pterm.Warning.Printf("Could not switch to %v\n", skipped)
		}
		return v, nil
	}
	return "", noSwitchError(errs)
}

// dependents returns the other installed versions that break when the directory of
// version is moved.
func (b *impl) dependents(version Version) []Version {
	var dependents []Version
	for _, v := range b.InstalledList() {
		if v != version && b.dependsOn(v, version) {
			dependents = append(dependents, v)
		}
	}
	return dependents
}

// dependsOn reports whether the main runtime or an addition of version is reached
// through the directory of other, so that it breaks when that directory is moved.
func (b *impl) dependsOn(version, other Version) bool {
	runtimePackage, err := b.GetRuntimePackage(version)
	if err != nil {
		return false // apply returns the error
	}
	dir := b.packagePath(other)
	for _, r := range append([]*Runtime{runtimePackage.Runtime}, runtimePackage.Additions...) {
		if leadsThrough(r.Path, dir) {
			return true
		}
	}
	return false
}

// leadsThrough reports whether path passes dir, by its own directories or by those
// of the links on it. The end of the links is not enough to tell: a link in dir may
// lead out of it again, and is gone with dir all the same.
func leadsThrough(path, dir string) bool {
	moved, err := os.Lstat(dir)
	if err != nil {
		return false
	}
	for links := 0; links < 255; links++ {
		link, rest, through := firstLink(path, moved)
		if through || link == "" {
			return through
		}
		target, ok := linkTarget(link)
		if !ok {
			return false
		}
		if !filepath.IsAbs(target) {
			target = filepath.Join(filepath.Dir(link), target)
		}
		path = filepath.Join(target, rest)
	}
	return false
}

// firstLink goes down path until it finds a link, and returns the link and the part
// of path after it, or until it comes to the directory moved.
func firstLink(path string, moved os.FileInfo) (link, rest string, through bool) {
	root := filepath.VolumeName(path) + string(filepath.Separator)
	names := strings.Split(strings.TrimPrefix(filepath.Clean(path), root), string(filepath.Separator))
	current := root
	for i, name := range names {
		current = filepath.Join(current, name)
		info, err := os.Lstat(current)
		if err != nil {
			return "", "", false
		}
		if os.SameFile(info, moved) {
			return "", "", true
		}
		if env.IsDirSymlink(current) {
			return current, filepath.Join(names[i+1:]...), false
		}
	}
	return "", "", false
}

func noSwitchError(errs []error) error {
	return fmt.Errorf("could not switch the global version to another version, so nothing was removed: %w", errors.Join(errs...))
}

// EnvKeys Only return the really installed path of this SDK
func (b *impl) EnvKeys(runtimePackage *RuntimePackage) (*env.Envs, error) {
	sdkInfos := make(map[string]*plugin.InstalledPackageItem)
	mainSdk := convertRuntime2InstalledPackageItem(runtimePackage.Runtime)
	sdkInfos[mainSdk.Name] = mainSdk

	for _, addition := range runtimePackage.Additions {
		sdkInfos[addition.Name] = convertRuntime2InstalledPackageItem(addition)
	}

	envKeysCtx := &plugin.EnvKeysHookCtx{
		Main:    mainSdk,
		SdkInfo: sdkInfos,
		Path:    runtimePackage.Path,
	}
	envKeysHookResultItems, err := b.plugin.EnvKeys(envKeysCtx)
	if b.plugin.IsNoResultProvided(err) {
		return nil, fmt.Errorf("no environment variables provided")
	}
	if err != nil {
		return nil, fmt.Errorf("plugin [EnvKeys] error: err:%w", err)
	}

	envKeys := &env.Envs{
		Variables: make(env.Vars),
		Paths:     env.NewPaths(env.EmptyPaths),
	}
	for _, item := range envKeysHookResultItems {
		if item.Key == "PATH" {
			envKeys.Paths.Add(item.Value)
		} else {
			envKeys.Variables[item.Key] = &item.Value
		}
	}
	logger.Debugf("EnvKeysHookResult: %+v \n", envKeys)
	return envKeys, nil
}

// matchableInstalledItems returns the installed versions a typed version can stand
// for, as PreInstall and PreUse are given them, keyed by version. They are built from
// what InstalledList already checked, without listing each version directory, so
// that one installation that cannot be read is neither left out nor able to empty
// the list.
func (b *impl) matchableInstalledItems() map[string]*plugin.InstalledPackageItem {
	versions := b.matchableVersions()
	sdks := make(map[string]*plugin.InstalledPackageItem, len(versions))
	for _, version := range versions {
		sdks[string(version)] = b.installedItem(version)
	}
	return sdks
}

func (b *impl) installedItem(version Version) *plugin.InstalledPackageItem {
	return &plugin.InstalledPackageItem{
		Name:    b.sdkName(),
		Version: string(version),
		Path:    b.packagePath(version),
	}
}

// inspectError returns why vfox cannot tell whether version is installed: its
// directory is there but cannot be entered, as one owned by another user can be.
// CheckRuntimeExist takes such a version as absent. It is nil otherwise.
func (b *impl) inspectError(version Version) error {
	if b.hasPendingInstall(version) {
		return nil
	}
	if _, err := os.Stat(b.mainRuntimePath(version)); errors.Is(err, fs.ErrPermission) {
		return err
	}
	return nil
}

// uninspectableVersions returns the versions whose directory cannot be entered (see
// inspectError), copies left out. InstalledList leaves them out.
func (b *impl) uninspectableVersions() []Version {
	entries, err := os.ReadDir(b.InstallPath)
	if err != nil {
		return nil
	}
	var versions []Version
	for _, d := range entries {
		if d.IsDir() && strings.HasPrefix(d.Name(), "v-") {
			if version := Version(strings.TrimPrefix(d.Name(), "v-")); b.inspectError(version) != nil && !b.isCopy(version) {
				versions = append(versions, version)
			}
		}
	}
	return versions
}

func (b *impl) preUse(version Version, scope env.UseScope) (Version, error) {
	// A copy is selected only by its own name, so the plugin is not given it and the
	// prefix matching below does not reach it.
	installedSdks := b.matchableInstalledItems()
	preUseCtx := &plugin.PreUseHookCtx{
		Cwd:             b.envContext.PathMeta.Working.Directory,
		PreviousVersion: string(b.Current()),
		Scope:           scope.String(),
		Version:         string(version),
		InstalledSdks:   installedSdks,
	}

	var newVersion string
	if b.plugin.HasFunction("PreUse") {
		preUseResult, err := b.plugin.PreUse(preUseCtx)
		if b.plugin.IsNoResultProvided(err) {
			// No result provided, continue with fuzzy matching
		} else if err != nil {
			return "", fmt.Errorf("plugin [preUse] error: err:%w", err)
		} else {
			newVersion = preUseResult.Version
		}
	}

	// If the plugin does not return a version (or doesn't have PreUse),
	// fuzzy match the version based on the input version.
	if newVersion == "" {
		// Before fuzzy matching, perform exact matching first.
		if b.CheckRuntimeExist(version) {
			return version, nil
		}
		installedVersions := make([]Version, 0, len(installedSdks))
		for v := range installedSdks {
			installedVersions = append(installedVersions, Version(v))
		}
		sortNewestFirst(installedVersions)
		version := string(version)
		prefix := version + "."
		for _, v := range installedVersions {
			if name := string(v); name == version || strings.HasPrefix(name, prefix) {
				newVersion = name
				break
			}
		}
	}
	if newVersion == "" {
		return version, nil
	}

	return Version(newVersion), nil
}

// ResolveInstalledVersion returns the installed version that version stands for,
// selected the way `vfox use` does: what the plugin's PreUse hook answers, else an
// exact match, else the newest installed version that extends it (4.5.0 for 4.5).
// The version comes back as given when nothing installed matches.
func (b *impl) ResolveInstalledVersion(version Version) (Version, error) {
	return b.preUse(version, env.Session)
}

type VersionNotExistsError struct {
	Label string
}

func (e *VersionNotExistsError) Error() string {
	return fmt.Sprintf("%s is not installed", e.Label)
}

func (b *impl) Use(version Version, scope env.UseScope) error {
	// Default behavior: project scope uses link=true
	return b.UseWithConfig(version, scope, false)
}

// UseWithConfig uses a version with custom link configuration
// unlink: if true, disables link for project scope (downgrade to session scope)
func (b *impl) UseWithConfig(version Version, scope env.UseScope, unlink bool) error {
	logger.Debugf("Use SDK version: %s, scope: %v, unlink: %v\n", string(version), scope, unlink)

	// Verify hook environment is available
	hookEnv := env.IsHookEnv()
	if !hookEnv {
		if runtime.GOOS == "windows" {
			// On Windows without hook, force global scope
			if scope != env.Global {
				pterm.Printf("Warning: The current shell lacks hook support. Switching to global scope automatically.\n")
				scope = env.Global
			}
		} else {
			logger.Debugf("Hook environment not available\n")
			return fmt.Errorf("vfox requires hook support. Please ensure vfox is properly initialized with 'vfox activate'")
		}
	}

	resolvedVersion, err := b.resolveInstalled(version, scope)
	if err != nil {
		return err
	}

	if err := b.useInHook(resolvedVersion, scope, unlink, hookEnv); err != nil {
		return err
	}

	// In non-hook environment, spawn a new shell so the user can use the new version immediately
	if !hookEnv {
		return openShell(os.Getppid())
	}
	return nil
}

// openShell runs a new shell as a child of vfox and waits for it to exit. It is
// a variable so that a test can see whether it is called.
var openShell = shell.Open

// resolveInstalled resolves version through the plugin's PreUse hook and checks
// that the result is installed.
func (b *impl) resolveInstalled(version Version, scope env.UseScope) (Version, error) {
	logger.Debugf("Resolving version with preUse hook for %s\n", version)
	resolvedVersion, err := b.preUse(version, scope)
	if err != nil {
		logger.Debugf("Failed to resolve version: %v\n", err)
		return "", err
	}
	logger.Debugf("Resolved version: %s\n", resolvedVersion)

	label := b.Label(resolvedVersion)
	if !b.CheckRuntimeExist(resolvedVersion) {
		logger.Debugf("SDK %s not installed\n", label)
		return "", &VersionNotExistsError{Label: label}
	}
	return resolvedVersion, nil
}

func (b *impl) useInHook(version Version, scope env.UseScope, unlink bool, hookEnv bool) error {
	logger.Debugf("Using SDK in hook: version=%s, scope=%v, unlink=%v\n", version, scope, unlink)

	inRegistry, err := b.apply(version, scope, unlink)
	if err != nil {
		return err
	}
	if inRegistry {
		pterm.Println(pterm.LightCyan("Applied global environment variables to Windows registry."))
	}

	pterm.Printf("Now using %s.\n", pterm.LightGreen(b.Label(version)))
	logger.Debugf("Successfully using SDK: %s\n", b.Label(version))

	// Say so when the shell is not going to run what was just selected. A caller
	// that hands off to a new shell does so after this returns, and that blocks.
	b.reportFindings(version, hookEnv)

	return nil
}

// apply makes version the one scope asks for: it points the links of the scope
// at it, writes the config and, for the global scope on Windows, updates the
// registry, in that order. If the links or the config fail, the links go back to
// what they were. It reports whether the registry was updated,
// and leaves it to the caller to say what happened.
func (b *impl) apply(version Version, scope env.UseScope, unlink bool) (inRegistry bool, err error) {
	runtimePackage, err := b.GetRuntimePackage(version)
	if err != nil {
		logger.Debugf("Failed to get runtime package for %s: %v\n", b.Label(version), err)
		return false, ErrRuntimeNotFound
	}

	// Gather env keys early so we can update the registry if needed
	envs, err := b.EnvKeysForScope(version, scope)
	if err != nil {
		logger.Debugf("Failed to generate env keys for registry update: %v\n", err)
		return false, err
	}

	// Ensure .vfox directory exists for project scope
	if env.Project == scope {
		if !util.FileExists(b.envContext.PathMeta.Working.ProjectSdkDir) {
			logger.Debugf("Creating project SDK directory: %s\n", b.envContext.PathMeta.Working.ProjectSdkDir)
			err := os.MkdirAll(b.envContext.PathMeta.Working.ProjectSdkDir, pathmeta.ReadWriteAuth)
			if err != nil {
				logger.Debugf("Failed to create project SDK directory: %v\n", err)
				return false, fmt.Errorf("failed to create .vfox directory: %w", err)
			}
		}

		// Add .vfox/ to .gitignore if needed
		if b.envContext.UserConfig.Gitignore.Enable {
			added, err := ensureVfoxInGitignore(b.envContext.PathMeta.Working.Directory)
			if err != nil {
				logger.Debugf("Failed to update .gitignore: %v\n", err)
			}
			if added {
				pterm.Printf("Added '.vfox/' to .gitignore\n")
				logger.Debugf("Added .vfox/ to .gitignore\n")
			}
		}
	}

	// Load config for the specified scope
	logger.Debugf("Loading config for scope: %v\n", scope)
	vfoxToml, err := b.envContext.LoadVfoxTomlByScope(scope)
	if err != nil {
		logger.Debugf("Failed to load config for scope %v: %v\n", scope, err)
		return false, err
	}

	logger.Debugf("Load %v tool versions: %v\n", scope, vfoxToml.GetAllTools())

	previous, hasPrevious := vfoxToml.GetToolVersion(b.Name)
	links, err := b.linksOfScope(scope)
	if err != nil {
		logger.Debugf("Failed to read the links of scope %v: %v\n", scope, err)
		return false, fmt.Errorf("failed to read symlinks, err:%w", err)
	}

	// Create symlinks for the specified scope
	logger.Debugf("Creating symlinks for scope: %v\n", scope)
	if err := b.createSymlinksForScope(runtimePackage, scope); err != nil {
		logger.Debugf("Failed to create symlinks: %v\n", err)
		return false, errors.Join(fmt.Errorf("failed to create symlinks, err:%w", err), b.revertLinks(links, scope))
	}

	// Determine link flag based on scope and unlink parameter
	if scope == env.Project && unlink {
		logger.Debugf("Setting unlink flag for project scope\n")
		attr := pathmeta.Attr{
			UnLinkAttrFlag: BoolYes,
		}
		// Update version record
		vfoxToml.SetToolWithAttr(b.Name, string(version), attr)
	} else {
		// Update version record
		vfoxToml.SetTool(b.Name, string(version))
	}
	if err := vfoxToml.Save(); err != nil {
		logger.Debugf("Failed to save tool versions: %v\n", err)
		return false, errors.Join(fmt.Errorf("failed to save tool versions, err:%w", err), b.revertLinks(links, scope))
	}

	// On Windows, always update registry for global scope so GUI apps can pick up the PATH
	if runtime.GOOS == "windows" && scope == env.Global {
		// Remove old version's registry entries before adding new ones
		if hasPrevious && previous != string(version) {
			oldEnvs, err := b.EnvKeysForScope(Version(previous), scope)
			if err == nil && oldEnvs != nil {
				logger.Debugf("Removing old version %s registry entries\n", previous)
				_ = env.RemoveEnvsFromRegistry(oldEnvs)
			}
		}
		if err := env.ApplyEnvsToRegistry(envs); err != nil {
			logger.Debugf("Failed to apply registry envs: %v\n", err)
			pterm.Warning.Printf("Failed to apply global registry environment: %v\n", err)
		} else {
			inRegistry = true
		}
	}
	return inRegistry, nil
}

// revertLinks undoes what apply did to the links of scope, which were links before.
func (b *impl) revertLinks(links map[string]string, scope env.UseScope) error {
	if err := b.restoreLinks(links, scope); err != nil {
		return fmt.Errorf("could not restore the links: %w", err)
	}
	return nil
}

// linksOfScope returns the links of scope, of all SDKs, and what they point at.
func (b *impl) linksOfScope(scope env.UseScope) (map[string]string, error) {
	linkDir := b.envContext.GetLinkDirPathByScope(scope)
	entries, err := os.ReadDir(linkDir)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	links := make(map[string]string)
	for _, entry := range entries {
		if target, ok := linkTarget(filepath.Join(linkDir, entry.Name())); ok {
			links[entry.Name()] = target
		}
	}
	return links, nil
}

// restoreLinks makes the links of this SDK in scope what they were when
// linksOfScope returned links. It needs nothing of the versions they point at, so
// a link to a version that is broken comes back too. A link of another SDK is left
// alone, unless it was replaced by one of this SDK. It goes on after a link that
// fails, and returns the errors of all of them.
func (b *impl) restoreLinks(links map[string]string, scope env.UseScope) error {
	current, err := b.linksOfScope(scope)
	if err != nil {
		return err
	}
	names := make(map[string]struct{})
	for name := range links {
		names[name] = struct{}{}
	}
	for name := range current {
		names[name] = struct{}{}
	}
	linkDir := b.envContext.GetLinkDirPathByScope(scope)
	var errs []error
	for name := range names {
		was, had := links[name]
		is, has := current[name]
		if had == has && was == is {
			continue
		}
		mine := name == b.sdkName() || had && isInside(b.InstallPath, was) || has && isInside(b.InstallPath, is)
		if !mine {
			continue
		}
		link := filepath.Join(linkDir, name)
		if had {
			err = env.CreateDirSymlink(was, link)
		} else {
			err = env.RemoveDirSymlink(link)
		}
		if err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// isInside reports whether path is dir or something in it.
func isInside(dir, path string) bool {
	rel, err := filepath.Rel(dir, path)
	return err == nil && filepath.IsLocal(rel)
}

func (b *impl) InstalledList() []Version {
	if !util.FileExists(b.InstallPath) {
		return make([]Version, 0)
	}
	var versions []Version
	dir, err := os.ReadDir(b.InstallPath)
	if err != nil {
		return make([]Version, 0)
	}
	for _, d := range dir {
		if d.IsDir() && strings.HasPrefix(d.Name(), "v-") {
			version := Version(strings.TrimPrefix(d.Name(), "v-"))
			if b.CheckRuntimeExist(version) {
				versions = append(versions, version)
			}
		}
	}
	sortNewestFirst(versions)
	return versions
}

// sortNewestFirst sorts versions by version number: 10.0.0 comes before 9.0.0.
func sortNewestFirst(versions []Version) {
	sort.Slice(versions, func(i, j int) bool {
		if c := compareInstalled(string(versions[i]), string(versions[j])); c != 0 {
			return c > 0
		}
		return versions[i] > versions[j]
	})
}

// compareInstalled orders installed version names for sortNewestFirst. A leading
// "v" is ignored (v20.1.0 is 20.1.0), a pre-release or other suffix comes
// before its release (2.0.0-rc1 < 2.0.0), the numbers in a suffix compare as
// numbers (rc9 < rc10), and a name that does not start with a number (dev,
// latest) is lower than every numbered one, so sortNewestFirst puts it last.
// Missing parts count as 0, so 1.0 and 1.0.0 compare equal.
func compareInstalled(a, b string) int {
	na, sa, oka := splitVersionName(a)
	nb, sb, okb := splitVersionName(b)
	switch {
	case oka != okb:
		if oka {
			return 1
		}
		return -1
	case !oka:
		return 0
	}
	for i := 0; i < len(na) || i < len(nb); i++ {
		var x, y int
		if i < len(na) {
			x = na[i]
		}
		if i < len(nb) {
			y = nb[i]
		}
		if x != y {
			if x > y {
				return 1
			}
			return -1
		}
	}
	switch {
	case sa == sb:
		return 0
	case sa == "":
		return 1
	case sb == "":
		return -1
	}
	return compareSuffix(sa, sb)
}

// suffixParts matches the runs of digits and the runs of anything else.
var suffixParts = regexp.MustCompile(`[0-9]+|[^0-9]+`)

// compareSuffix compares the runs of digits in two suffixes as numbers and the
// rest as text. A number comes before text, and a suffix before a longer one
// that starts with it.
func compareSuffix(a, b string) int {
	pa, pb := suffixParts.FindAllString(a, -1), suffixParts.FindAllString(b, -1)
	for i := 0; i < len(pa) && i < len(pb); i++ {
		x, y := pa[i], pb[i]
		if isDigit(x[0]) && isDigit(y[0]) {
			// Without leading zeros, the longer number is the larger one.
			x, y = strings.TrimLeft(x, "0"), strings.TrimLeft(y, "0")
			if c := cmp.Compare(len(x), len(y)); c != 0 {
				return c
			}
		}
		if c := strings.Compare(x, y); c != 0 {
			return c
		}
	}
	return cmp.Compare(len(pa), len(pb))
}

func isDigit(c byte) bool { return c >= '0' && c <= '9' }

// splitVersionName splits a name such as v2.0.0-rc1 into its numbers [2 0 0]
// and the suffix "-rc1". ok is false when the name does not start with a number.
func splitVersionName(name string) (nums []int, suffix string, ok bool) {
	if len(name) > 1 && (name[0] == 'v' || name[0] == 'V') && name[1] >= '0' && name[1] <= '9' {
		name = name[1:]
	}
	rest := name
	for {
		i := 0
		for i < len(rest) && rest[i] >= '0' && rest[i] <= '9' {
			i++
		}
		if i == 0 {
			break
		}
		n, err := strconv.Atoi(rest[:i])
		if err != nil {
			break
		}
		nums = append(nums, n)
		rest = rest[i:]
		if len(rest) < 2 || rest[0] != '.' || rest[1] < '0' || rest[1] > '9' {
			break
		}
		rest = rest[1:]
	}
	return nums, rest, len(nums) > 0
}

// globalVersion returns the version the global config asks for, or "".
func (b *impl) globalVersion() Version {
	global, err := b.envContext.LoadVfoxTomlByScope(env.Global)
	if err != nil {
		logger.Debugf("Failed to load the global config: %v", err)
		return ""
	}
	if version, ok := global.GetToolVersion(b.Name); ok {
		return Version(version)
	}
	return ""
}

// Current returns the current version of the SDK.
// Lookup priority is: project > session > global
func (b *impl) Current() Version {
	// Load configs from all scopes with priority: Global < Session < Project
	chain, err := b.envContext.LoadVfoxTomlChainByScopes(env.Global, env.Session, env.Project)
	if err != nil {
		logger.Debugf("Failed to load config chain: %v", err)
		return ""
	}

	// Search for current version (with priority)
	if version, _, ok := chain.GetToolVersion(b.Name); ok && b.CheckRuntimeExist(Version(version)) {
		return Version(version)
	}
	return ""
}

// ParseLegacyFile tries to parse legacy version files to get the Runtime version.
// It returns the first valid version found.
func (b *impl) ParseLegacyFile(path string) (Version, error) {
	legacyFilenames := b.plugin.Metadata.LegacyFilenames
	if len(legacyFilenames) == 0 {
		return "", nil
	}
	if !b.plugin.HasFunction("ParseLegacyFile") {
		return "", nil
	}
	for _, filename := range legacyFilenames {
		legacyFilePath := filepath.Join(path, filename)
		if !util.FileExists(legacyFilePath) {
			continue
		}
		logger.Debugf("Parsing legacy file %s \n", legacyFilePath)
		ctx := &plugin.ParseLegacyFileHookCtx{
			Filepath: legacyFilePath,
			Filename: filename,
			GetInstalledVersions: func() []string {
				versions := b.matchableVersions()
				logger.Debugf("Invoking GetInstalledVersions result: %+v \n", versions)
				convertVersions := make([]string, 0, len(versions))
				for _, v := range versions {
					convertVersions = append(convertVersions, string(v))
				}
				return convertVersions
			},
			Strategy: b.envContext.UserConfig.LegacyVersionFile.Strategy,
		}

		result, err := b.plugin.ParseLegacyFile(ctx)
		if err != nil {
			logger.Debugf("Parsing legacy file failed:%s , error:%v\n", legacyFilePath, err)
			return "", err
		}
		if result.Version != "" {
			return Version(result.Version), nil
		}
	}
	return "", nil
}

func (b *impl) Close() {
	b.plugin.Close()
}

// createSymlinksForScope creates symlinks in the appropriate SDK directory for the scope
func (b *impl) createSymlinksForScope(runtimePackage *RuntimePackage, scope env.UseScope) error {

	// Determine target SDK directory based on scope
	sdkDir := b.envContext.GetLinkDirPathByScope(scope)
	// Create symlinks for main runtime
	if err := b.createDirSymlinks(runtimePackage.Runtime, sdkDir); err != nil {
		return err
	}
	// Create symlinks for all additions
	for _, addition := range runtimePackage.Additions {
		if err := b.createDirSymlinks(addition, sdkDir); err != nil {
			// Log but continue with other additions
			logger.Debugf("Failed to create symlink for addition %s: %v", addition.Name, err)
		}
	}
	return nil
}

// CreateSymlinksForScope creates symlinks for a specific version in the given scope
func (b *impl) CreateSymlinksForScope(version Version, scope env.UseScope) error {
	runtimePackage, err := b.GetRuntimePackage(version)
	if err != nil {
		return err
	}
	return b.createSymlinksForScope(runtimePackage, scope)
}

// EnvKeysForScope returns environment variables for a version in the given scope.
// It returns env vars with paths pointing to symlinks (does NOT create symlinks).
func (b *impl) EnvKeysForScope(version Version, scope env.UseScope) (*env.Envs, error) {
	// 1. Get the real runtime package
	runtimePackage, err := b.GetRuntimePackage(version)
	if err != nil {
		return nil, err
	}

	// 2. Get the symlink directory for this scope
	linkDir := b.envContext.GetLinkDirPathByScope(scope)
	// 3. Return env keys with paths pointing to symlinks
	symlinkPackage := runtimePackage.ReplacePath(linkDir)
	return b.EnvKeys(symlinkPackage)
}

// removeSymlinksForScope removes symlinks for the specified scope and version
func (b *impl) removeSymlinksForScope(version Version, scope env.UseScope) error {
	// Determine target SDK directory based on scope
	sdkDir := b.envContext.GetLinkDirPathByScope(scope)
	runtimePackage, err := b.GetRuntimePackage(version)
	if err != nil {
		return err
	}
	symlinkRuntimePackage := runtimePackage.ReplacePath(sdkDir)

	mainRuntime := symlinkRuntimePackage.Runtime
	_ = env.RemoveDirSymlink(mainRuntime.Path)
	for _, addition := range symlinkRuntimePackage.Additions {
		_ = env.RemoveDirSymlink(addition.Path)
	}
	return nil
}

// linksInto returns the links of scope that point into dir.
func (b *impl) linksInto(dir string, scope env.UseScope) ([]string, error) {
	targets, err := b.linksOfScope(scope)
	if err != nil {
		return nil, err
	}
	linkDir := b.envContext.GetLinkDirPathByScope(scope)
	var links []string
	for name, target := range targets {
		if isInside(dir, target) {
			links = append(links, filepath.Join(linkDir, name))
		}
	}
	sort.Strings(links)
	return links, nil
}

// removeLinksInto removes the links of scope that point into dir. It goes on
// after a link that cannot be removed, and returns the errors of all of them.
func (b *impl) removeLinksInto(dir string, scope env.UseScope) error {
	links, err := b.linksInto(dir, scope)
	if err != nil {
		return err
	}
	var errs []error
	for _, link := range links {
		if err := env.RemoveDirSymlink(link); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// createDirSymlinks creates symlinks for runtime in the given paths
// Checks if symlink already exists and points to the correct target before creating
func (b *impl) createDirSymlinks(runtime *Runtime, targetDir string) error {
	// Full path to the symlink
	linkPath := filepath.Join(targetDir, runtime.Name)

	// Check if symlink already exists and points to the correct target
	if env.IsDirSymlink(linkPath) {
		currentTarget, err := env.ReadDirSymlink(linkPath)
		if err == nil {
			// Normalize paths for comparison (handles path separators, etc.)
			// Convert both paths to absolute paths for accurate comparison
			currentTargetAbs, err1 := filepath.Abs(currentTarget)
			expectedTargetAbs, err2 := filepath.Abs(runtime.Path)

			if err1 == nil && err2 == nil && currentTargetAbs == expectedTargetAbs {
				// Symlink already exists and points to the correct target, skip creation
				logger.Debugf("Symlink already exists and is correct: %s -> %s \n", linkPath, currentTarget)
				return nil
			}
			// Symlink exists but points to wrong target or path resolution failed, need to recreate
			logger.Debugf("Symlink exists but points to wrong target, recreating: %s (current: %s, expected: %s) \n",
				linkPath, currentTarget, runtime.Path)
		}
	}

	// Ensure parent directory exists
	if err := os.MkdirAll(targetDir, pathmeta.ReadWriteAuth); err != nil {
		return fmt.Errorf("failed to create SDK directory %s, err:%w", targetDir, err)
	}

	// Create the symlink (env.CreateDirSymlink will handle removing old symlink if needed)
	return env.CreateDirSymlink(runtime.Path, linkPath)
}

func (b *impl) GetRuntimePackage(version Version) (*RuntimePackage, error) {
	if b.hasPendingInstall(version) {
		return nil, ErrRuntimeNotFound
	}
	versionPath := b.packagePath(version)
	items := make(map[string]*Runtime)
	dir, err := os.ReadDir(versionPath)
	if err != nil {
		return nil, err
	}
	for _, d := range dir {
		// DirEntry.IsDir() is false for a symlink or junction, which is how a
		// linked runtime is stored, so ask the file system about the target.
		if fi, err := os.Stat(filepath.Join(versionPath, d.Name())); err == nil && fi.IsDir() {
			if strings.HasSuffix(d.Name(), string(version)) {
				name := strings.TrimSuffix(d.Name(), "-"+string(version))
				if name == "" {
					continue
				}
				logger.Debugf("Load SDK package item: name:%s, version: %s \n", name, version)
				items[name] = &Runtime{
					Name:    name,
					Version: version,
					Path:    filepath.Join(versionPath, d.Name()),
				}
			}
		}
	}
	main, ok := items[b.plugin.Name]
	if !ok {
		return nil, ErrRuntimeNotFound
	}
	delete(items, b.plugin.Name)
	if main.Path == "" {
		return nil, ErrRuntimeNotFound
	}
	var additions []*Runtime
	for _, v := range items {
		additions = append(additions, v)
	}
	p2 := &RuntimePackage{
		Runtime:     main,
		Additions:   additions,
		PackagePath: versionPath,
	}
	if b.hasPendingInstall(version) {
		return nil, ErrRuntimeNotFound
	}
	return p2, nil
}

func (b *impl) CheckRuntimeExist(version Version) bool {
	if b.hasPendingInstall(version) {
		return false
	}
	// Older installations have no marker. Keep those valid, but do not accept
	// an empty version directory without the SDK's main payload directory.
	// os.Stat follows links, so a linked runtime counts while its target exists.
	info, err := os.Stat(b.mainRuntimePath(version))
	return err == nil && info.IsDir() && !b.hasPendingInstall(version)
}

func (b *impl) HasVersion(version Version) bool {
	if version == "" {
		return false
	}
	for _, path := range []string{b.packagePath(version), b.installMarkerPath(version)} {
		if _, err := os.Lstat(path); err == nil {
			return true
		}
	}
	return false
}

// HasLeftovers reports whether an uninstall of version left files or links of it.
func (b *impl) HasLeftovers(version Version) bool {
	if version == "" {
		return false
	}
	if len(b.leftovers(version)) > 0 {
		return true
	}
	// Uninstall returns the errors.
	entry, err := b.globalEntry()
	if err != nil {
		return false
	}
	left, err := b.linksLeft(version, entry)
	return err == nil && left
}

// HasVersionDirs reports whether the directory of the SDK holds a version, usable
// or not, or an interrupted installation of one.
func (b *impl) HasVersionDirs() bool {
	entries, err := os.ReadDir(b.InstallPath)
	if err != nil {
		return !os.IsNotExist(err)
	}
	return slices.ContainsFunc(entries, func(entry os.DirEntry) bool {
		name := entry.Name()
		return strings.HasPrefix(name, packageInstalledPrefix) ||
			strings.HasPrefix(name, ".v-") && strings.HasSuffix(name, ".installing")
	})
}

func (b *impl) installMarkerPath(version Version) string {
	return filepath.Join(b.InstallPath, ".v-"+string(version)+".installing")
}

func (b *impl) hasPendingInstall(version Version) bool {
	_, err := os.Lstat(b.installMarkerPath(version))
	return !os.IsNotExist(err)
}

func (b *impl) packagePath(version Version) string {
	return filepath.Join(b.InstallPath, packageInstalledPrefix+string(version))
}

func (b *impl) Download(u *url.URL, headers map[string]string) (string, error) {
	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return "", util.RedactError(err)
	}
	for key, value := range headers {
		req.Header.Add(key, value)
	}
	// The archive is asked for with the agent of the plugin's own requests,
	// unless PreInstall returned one in its headers.
	if req.Header.Get("User-Agent") == "" && b.plugin != nil && b.plugin.Metadata != nil {
		req.Header.Set("User-Agent", util.UserAgent(b.envContext.RuntimeVersion, b.plugin.Name, b.plugin.Version))
	}
	client, err := b.envContext.HttpClient()
	if err != nil {
		return "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		var urlErr *url.Error
		if errors.As(err, &urlErr) {
			var netErr net.Error
			if errors.As(urlErr.Err, &netErr) && netErr.Timeout() {
				return "", errors.New("request timeout")
			}
		}
		return "", util.RedactError(err)
	}

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return "", errors.New("source file not found")
	}

	if err = util.UnexpectedStatusError(resp, u.String()); err != nil {
		return "", err
	}

	err = os.MkdirAll(b.InstallPath, 0755)
	if err != nil {
		return "", err
	}

	fileName := filepath.Base(u.Path)
	if strings.HasPrefix(u.Fragment, "/") && strings.Contains(u.Fragment, ".") {
		fileName = strings.Trim(u.Fragment, "/")
	} else if !strings.Contains(fileName, ".") {
		finalURL, err := url.Parse(resp.Request.URL.String())
		if err != nil {
			return "", err
		}
		fileName = filepath.Base(finalURL.Path)
	}
	path := filepath.Join(b.InstallPath, fileName)

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return "", err
	}

	defer f.Close()

	bar := progressbar.NewOptions64(
		resp.ContentLength,
		progressbar.OptionSetWriter(os.Stderr),
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionShowBytes(true),
		progressbar.OptionFullWidth(),
		progressbar.OptionOnCompletion(func() {
			fmt.Fprintf(os.Stderr, "\n")
		}),
		progressbar.OptionSetDescription("Downloading..."),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "[green]=[reset]",
			SaucerHead:    "[green]>[reset]",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}),
	)
	defer bar.Close()
	_, err = io.Copy(io.MultiWriter(f, bar), resp.Body)
	if err != nil {
		return "", err
	}
	return path, nil
}

func (b *impl) Label(version Version) string {
	return fmt.Sprintf("%s@%s", strings.ToLower(b.Name), version)
}

// Unuse removes the version setting for the SDK from the specified scope
func (b *impl) Unuse(scopes ...env.UseScope) error {
	if err := b.unuse(scopes...); err != nil {
		return err
	}
	b.printUnset()
	return nil
}

func (b *impl) printUnset() {
	pterm.Printf("Unset %s successfully.\n", pterm.LightGreen(b.Name))
}

// unuse is Unuse without the message.
func (b *impl) unuse(scopes ...env.UseScope) error {
	for _, scope := range scopes {
		vfoxtoml, err := b.envContext.LoadVfoxTomlByScope(scope)
		if err != nil {
			return fmt.Errorf("failed to read %s config: %w", scope, err)
		}
		// Check if the SDK is currently set globally
		if version, ok := vfoxtoml.GetToolVersion(b.Name); ok {
			var registryEnv *env.Envs
			if runtime.GOOS == "windows" && scope == env.Global {
				registryEnv, _ = b.EnvKeysForScope(Version(version), scope)
			}
			// Remove shims for the current version
			if err := b.removeSymlinksForScope(Version(version), scope); err != nil {
				logger.Debugf("Failed to remove global symlinks: %v\n", err)
			}
			if registryEnv != nil {
				if err := env.RemoveEnvsFromRegistry(registryEnv); err != nil {
					logger.Debugf("Failed to remove registry envs: %v\n", err)
				}
			}
		}
		vfoxtoml.RemoveTool(b.Name)
		// Save all modified configs
		if err = vfoxtoml.Save(); err != nil {
			return fmt.Errorf("failed to save configs: %w", err)
		}
	}
	return nil
}

// NewSdk creates a new SDK instance based on the provided plugin path and runtime environment context.
func NewSdk(runtimeEnvContext *env.RuntimeEnvContext, pluginPath string) (Sdk, error) {
	sdkName := filepath.Base(pluginPath)
	plugin, err := plugin.CreatePlugin(pluginPath, runtimeEnvContext)
	if err != nil {
		return nil, fmt.Errorf("failed to create plugin: %w", err)
	}

	// SDK install path fixed to shared root (single location)
	installPath := filepath.Join(
		runtimeEnvContext.PathMeta.Shared.Installs,
		strings.ToLower(sdkName),
	)

	if !util.FileExists(installPath) {
		// compatibility bug path introduced in v1.0.2
		newVar := filepath.Join(runtimeEnvContext.PathMeta.Shared.Installs, "cache", strings.ToLower(sdkName))
		if util.FileExists(newVar) {
			installPath = newVar
		}
	}

	return &impl{
		Name:        sdkName,
		InstallPath: installPath,
		envContext:  runtimeEnvContext,
		plugin:      plugin,
	}, nil
}

// ensureVfoxInGitignore adds .vfox/ to .gitignore if needed
// Returns (true, nil) if the entry was added, (false, nil) if it already existed or .gitignore doesn't exist
func ensureVfoxInGitignore(projectDir string) (bool, error) {
	gitignorePath := filepath.Join(projectDir, ".gitignore")

	// Check if .gitignore exists
	if _, err := os.Stat(gitignorePath); os.IsNotExist(err) {
		// .gitignore doesn't exist, don't create it
		return false, nil
	}

	// Read existing content
	content, err := os.ReadFile(gitignorePath)
	if err != nil {
		return false, fmt.Errorf("failed to read .gitignore: %w", err)
	}

	contentStr := string(content)

	// Check if .vfox/ or .vfox is already in the file
	lines := strings.Split(contentStr, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == ".vfox/" || trimmed == ".vfox" {
			// Already exists, no need to add
			return false, nil
		}
	}

	// Append .vfox/ to the file
	f, err := os.OpenFile(gitignorePath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return false, fmt.Errorf("failed to open .gitignore: %w", err)
	}
	defer f.Close()

	// Add newline if the file doesn't end with one
	if len(contentStr) > 0 && !strings.HasSuffix(contentStr, "\n") {
		if _, err := f.WriteString("\n"); err != nil {
			return false, fmt.Errorf("failed to write to .gitignore: %w", err)
		}
	}

	if _, err := f.WriteString(".vfox/\n"); err != nil {
		return false, fmt.Errorf("failed to write to .gitignore: %w", err)
	}

	return true, nil
}
