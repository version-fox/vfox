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
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"sort"
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

// Sdk interface defines the methods for managing software development kits (SDKs).
type Sdk interface {
	Install(version Version) error                                        // Install a specific runtime of the SDK
	Reinstall(version Version) error                                      // Replace an installed runtime with a fresh copy
	InstallResolved(version Version, reinstall bool) (Version, error)     // Install and report the version that ended up on disk
	InstallAs(version Version, name Version) (Version, error)             // Install version as a second copy kept under name
	Uninstall(version Version) error                                      // Uninstall a specific runtime of the SDK
	Available(args []string) ([]*AvailableRuntimePackage, error)          // List available runtime of the SDK
	EnvKeys(runtimePackage *RuntimePackage) (*env.Envs, error)            // Get environment variables for a specific runtime of the SDK
	Use(version Version, scope env.UseScope) error                        // Use a specific runtime in a given scope
	UseWithConfig(version Version, scope env.UseScope, unlink bool) error // Use with link configuration
	Unuse(scopes ...env.UseScope) error                                   // Unuse the current runtime in a given scope
	GetRuntimePackage(version Version) (*RuntimePackage, error)           // Get the runtime package for a specific version
	CheckRuntimeExist(version Version) bool                               // Check if a specific runtime version is installed
	ResolveInstalledVersion(version Version) (Version, error)             // The installed version a typed one stands for, as `vfox use` selects it
	InstalledList() []Version

	// HasVersion reports whether anything is on disk under the version's name, whole or not: an
	// installation, a link whose target is gone, an interrupted installation. CheckRuntimeExist is
	// the stricter question, whether the version can be used.
	HasVersion(version Version) bool

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
	return b.install(version, "", reinstall)
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
	return b.install(version, name, false)
}

// install is InstallResolved and InstallAs. Without a name the installation is kept
// under the version PreInstall resolves the request to; with one, under name, and the
// version it resolves to is recorded in the receipt.
func (b *impl) install(version, name Version, reinstall bool) (installed Version, installErr error) {
	// PreInstall does not know the name of a copy, so a copy is reinstalled from the
	// version it was asked for, and the reinstall has to resolve to the version it has.
	exact := version
	if reinstall && name == "" {
		if receipt := b.copyReceipt(version); receipt != nil {
			version, name, exact = Version(receipt.Requested), version, Version(receipt.Resolved)
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
		lock, err := acquireInstallLock(filepath.Join(b.InstallPath, ".v-"+string(typed)+".lock"))
		if err != nil {
			return "", fmt.Errorf("cannot install %s: %w", label, err)
		}
		if err := lock.Close(); err != nil {
			return "", fmt.Errorf("close installation lock: %w", err)
		}
	}

	// The SDK lock is held from the list PreInstall is given until the name the
	// request resolves to is claimed, that is, until its installation marker is
	// written. The plugin can map the request to an installed version under another
	// name, and the lock of each version is per name, so without it a name another
	// installation is about to claim would not be seen. It is released before the
	// download and PostInstall, so other versions of the SDK, including one that a
	// PostInstall installs, install side by side as before.
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
	requestLabel := label
	for {
		label = requestLabel
		var err error
		sdkLock, err = waitForInstallLock(filepath.Join(b.InstallPath, sdkInstallLockName), func() {
			pterm.Printf("Waiting for another installation of %s to finish...\n", b.Name)
		})
		if errors.Is(err, errInstallLockStillHeld) {
			return "", fmt.Errorf("cannot install %s: another installation of %s is still running after %s; try again when it is done", label, b.Name, installLockWait)
		}
		if err != nil {
			return "", fmt.Errorf("cannot install %s: %w", label, err)
		}

		// Only a plain install is given the installed versions. A reinstall replaces the
		// version as it was typed, and --as makes a second copy on purpose, so for both
		// the plugin resolves the version as it would without the list.
		installedSdks := map[string]*plugin.InstalledPackageItem{}
		var installing map[Version]bool
		if !reinstall && !named {
			installedSdks = b.matchableInstalledItems()
			// A version directory that cannot be entered may hold an installation, so
			// the plugin is given it too, and a request resolved to it is refused below.
			for _, version := range b.uninspectableVersions() {
				installedSdks[string(version)] = b.installedItem(version)
			}
			// So is a version another installation is still installing. If the request
			// is mapped to it, the install waits for it below.
			installing = b.installingVersions()
			for version := range installing {
				installedSdks[string(version)] = b.installedItem(version)
			}
		}
		ctx := &plugin.PreInstallHookCtx{
			Version:       string(version),
			InstalledSdks: installedSdks,
		}
		logger.Debugf("Calling PreInstall hook for %s\n", label)
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
		// The list given to PreInstall leaves copies out, but a plugin can return any
		// name. A copy is selected only by its own name, so it cannot stand for the
		// version asked for.
		if !named && b.isCopy(sdkVersion) {
			return "", fmt.Errorf("plugin [PreInstall] method error: %s was resolved to %s, but %s is a copy made with `vfox install --as` and is selected only by its own name", b.Label(version), label, label)
		}
		if named && sdkVersion == name {
			return "", fmt.Errorf("%s resolves to %s, so a copy of it cannot be named %s: name the copy differently", b.Label(version), sdkVersion, name)
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
			if b.CheckRuntimeExist(key) {
				fmt.Printf("%s is already installed\n", label)
				logger.Debugf("SDK %s was installed while waiting for it\n", label)
				return key, nil
			}
			continue
		}
		lock, err = acquireInstallLock(filepath.Join(b.InstallPath, ".v-"+string(key)+".lock"))
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
		err = b.plugin.PostInstall(postCtx)
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

	// Copy command to clipboard in TTY mode
	if util.IsTTY() {
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
	if receipt == nil || receipt.Resolved == "" || receipt.Requested == "" {
		return nil
	}
	return receipt
}

// isCopy reports whether an installed version is a copy made with InstallAs.
// Its mark beside the directory is read first, so that a copy whose directory
// cannot be entered is still known as one.
func (b *impl) isCopy(version Version) bool {
	if _, err := os.Lstat(b.copyMarkerPath(version)); err == nil {
		if _, err := os.Lstat(b.packagePath(version)); err == nil {
			return true
		}
	}
	return b.copyReceipt(version) != nil
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
	file, err := openInstallLock(filepath.Join(b.InstallPath, ".v-"+string(version)+".lock"))
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
	lock, err := waitForInstallLock(filepath.Join(b.InstallPath, ".v-"+string(version)+".lock"), func() {
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

func (b *impl) Uninstall(version Version) (err error) {
	label := b.Label(version)
	logger.Debugf("Uninstalling SDK: %s\n", label)

	installed := b.CheckRuntimeExist(version)
	// A link whose target has gone away no longer counts as installed, but it
	// is still there to be removed.
	if !installed && !b.brokenLink(version) {
		logger.Debugf("SDK %s not installed\n", label)
		return fmt.Errorf("%s is not installed", pterm.Red(label))
	}
	path := b.packagePath(version)
	logger.Debugf("Uninstalling from path: %s\n", path)

	linkedTo, linked := b.linkedTarget(version)
	var sdkPackage *RuntimePackage
	if installed {
		sdkPackage, err = b.GetRuntimePackage(version)
		if err != nil {
			logger.Debugf("Failed to get runtime package for %s: %v\n", label, err)
			return
		}
	}
	// vfox does not own the files of a linked version, so the plugin has
	// nothing of its own to clean up in them.
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
			return
		}
	}

	// If the global scope asks for the version to be uninstalled, unuse it
	// there first. A version only a project or session asks for leaves the
	// global setting alone.
	if b.globalVersion() == version {
		logger.Debugf("SDK %s is currently in use, unusing it first\n", label)
		if err = b.Unuse(env.Global); err != nil {
			return err
		}
	}

	if linked {
		logger.Debugf("Removing link: %s\n", path)
		if err = b.removeLink(version); err != nil {
			logger.Debugf("Failed to remove link: %v\n", err)
			return
		}
		pterm.Printf("Unlinked %s successfully! %s was left untouched.\n", label, linkedTo)
		return
	}
	logger.Debugf("Removing directory: %s\n", path)
	err = os.RemoveAll(path)
	if err != nil {
		logger.Debugf("Failed to remove directory: %v\n", err)
		return
	}
	if err = os.Remove(b.copyMarkerPath(version)); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove copy marker: %w", err)
	}
	err = nil
	pterm.Printf("Uninstalled %s successfully!\n", label)
	logger.Debugf("SDK %s uninstalled successfully\n", label)
	return
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
		installedVersions := make(util.VersionSort, 0, len(installedSdks))
		for v := range installedSdks {
			installedVersions = append(installedVersions, v)
		}
		sort.Sort(installedVersions)
		version := string(version)
		prefix := version + "."
		for _, v := range installedVersions {
			if version == v {
				newVersion = v
				break
			}
			if strings.HasPrefix(v, prefix) {
				newVersion = v
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

	// Resolve version with preUse hook
	logger.Debugf("Resolving version with preUse hook for %s\n", version)
	resolvedVersion, err := b.preUse(version, scope)
	if err != nil {
		logger.Debugf("Failed to resolve version: %v\n", err)
		return err
	}
	logger.Debugf("Resolved version: %s\n", resolvedVersion)

	// Verify version exists
	label := b.Label(resolvedVersion)
	if !b.CheckRuntimeExist(resolvedVersion) {
		logger.Debugf("SDK %s not installed\n", label)
		return &VersionNotExistsError{Label: label}
	}

	return b.useInHook(resolvedVersion, scope, unlink, hookEnv)
}

func (b *impl) useInHook(version Version, scope env.UseScope, unlink bool, hookEnv bool) error {
	logger.Debugf("Using SDK in hook: version=%s, scope=%v, unlink=%v\n", version, scope, unlink)

	runtimePackage, err := b.GetRuntimePackage(version)
	if err != nil {
		logger.Debugf("Failed to get runtime package for %s: %v\n", b.Label(version), err)
		return ErrRuntimeNotFound
	}

	// Gather env keys early so we can update the registry if needed
	envs, err := b.EnvKeysForScope(version, scope)
	if err != nil {
		logger.Debugf("Failed to generate env keys for registry update: %v\n", err)
		return err
	}

	// Ensure .vfox directory exists for project scope
	if env.Project == scope {
		if !util.FileExists(b.envContext.PathMeta.Working.ProjectSdkDir) {
			logger.Debugf("Creating project SDK directory: %s\n", b.envContext.PathMeta.Working.ProjectSdkDir)
			err := os.MkdirAll(b.envContext.PathMeta.Working.ProjectSdkDir, pathmeta.ReadWriteAuth)
			if err != nil {
				logger.Debugf("Failed to create project SDK directory: %v\n", err)
				return fmt.Errorf("failed to create .vfox directory: %w", err)
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
		return err
	}

	logger.Debugf("Load %v tool versions: %v\n", scope, vfoxToml.GetAllTools())

	// On Windows global scope, remove old version's registry entries before adding new ones
	if runtime.GOOS == "windows" && scope == env.Global {
		if oldVersion, ok := vfoxToml.GetToolVersion(b.Name); ok && oldVersion != string(version) {
			oldEnvs, err := b.EnvKeysForScope(Version(oldVersion), scope)
			if err == nil && oldEnvs != nil {
				logger.Debugf("Removing old version %s registry entries\n", oldVersion)
				_ = env.RemoveEnvsFromRegistry(oldEnvs)
			}
		}
	}

	// Determine link flag based on scope and unlink parameter
	if scope == env.Project && unlink {
		logger.Debugf("Setting unlink flag for project scope\n")
		attr := pathmeta.Attr{
			UnLinkAttrFlag: BoolYes,
		}
		// Update version record
		vfoxToml.SetToolWithAttr(b.Name, string(version), attr)
		if err := vfoxToml.Save(); err != nil {
			logger.Debugf("Failed to save tool versions: %v\n", err)
			return fmt.Errorf("failed to save tool versions, err:%w", err)
		}
	} else {
		// Update version record
		vfoxToml.SetTool(b.Name, string(version))
		if err := vfoxToml.Save(); err != nil {
			logger.Debugf("Failed to save tool versions: %v\n", err)
			return fmt.Errorf("failed to save tool versions, err:%w", err)
		}
	}

	// Create symlinks for the specified scope
	logger.Debugf("Creating symlinks for scope: %v\n", scope)
	if err := b.createSymlinksForScope(runtimePackage, scope); err != nil {
		logger.Debugf("Failed to create symlinks: %v\n", err)
		return fmt.Errorf("failed to create symlinks, err:%w", err)
	}

	// On Windows, always update registry for global scope so GUI apps can pick up the PATH
	if runtime.GOOS == "windows" && scope == env.Global {
		if err := env.ApplyEnvsToRegistry(envs); err != nil {
			logger.Debugf("Failed to apply registry envs: %v\n", err)
			pterm.Warning.Printf("Failed to apply global registry environment: %v\n", err)
		} else {
			pterm.Println(pterm.LightCyan("Applied global environment variables to Windows registry."))
		}
	}

	pterm.Printf("Now using %s.\n", pterm.LightGreen(b.Label(version)))
	logger.Debugf("Successfully using SDK: %s\n", b.Label(version))

	// Say so when the shell is not going to run what was just selected. This
	// comes before the hand-off to a new shell below, which blocks.
	b.reportFindings(version, hookEnv)

	// In non-hook environment, spawn a new shell so the user can use the new version immediately
	if !hookEnv {
		return shell.Open(os.Getppid())
	}

	return nil
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
	sort.Slice(versions, func(i, j int) bool {
		return versions[i] > versions[j]
	})
	return versions
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
	pterm.Printf("Unset %s successfully.\n", pterm.LightGreen(b.Name))
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
