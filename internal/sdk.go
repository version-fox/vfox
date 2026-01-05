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

package internal

import (
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"syscall"

	"github.com/pterm/pterm"
	"github.com/schollz/progressbar/v3"
	"github.com/version-fox/vfox/internal/base"
	"github.com/version-fox/vfox/internal/env"
	"github.com/version-fox/vfox/internal/logger"
	"github.com/version-fox/vfox/internal/plugin"
	"github.com/version-fox/vfox/internal/shell"
	"github.com/version-fox/vfox/internal/shim"
	"github.com/version-fox/vfox/internal/toolset"
	"github.com/version-fox/vfox/internal/util"
)

type SdkEnv struct {
	Sdk *Sdk
	Env *env.Envs
}

type SdkEnvs []*SdkEnv

// ToVars export the env vars of SDK to shell
func (d *SdkEnvs) ToVars() env.Vars {
	vars := make(env.Vars)
	for _, sdkEnv := range *d {
		for key, value := range sdkEnv.Env.Variables {
			vars[key] = value
		}
	}
	return vars
}

func (d *SdkEnvs) ToEnvs() *env.Envs {
	envs := &env.Envs{
		Variables: make(env.Vars),
		Paths:     env.NewPaths(env.EmptyPaths),
	}
	for _, sdkEnv := range *d {
		for key, value := range sdkEnv.Env.Variables {
			envs.Variables[key] = value
		}
	}

	return envs
}

func (d *SdkEnvs) ToExportEnvs() env.Vars {
	envKeys := d.ToEnvs()

	exportEnvs := make(env.Vars)
	for k, v := range envKeys.Variables {
		exportEnvs[k] = v
	}

	osPaths := env.NewPaths(env.OsPaths)
	pathsStr := envKeys.Paths.Merge(osPaths).String()
	exportEnvs["PATH"] = &pathsStr

	return exportEnvs
}

type Sdk struct {
	Name       string
	sdkManager *Manager
	Plugin     *plugin.PluginWrapper
	// current sdk install path
	InstallPath          string
	localSdkPackageCache map[base.Version]*base.Package
}

func (b *Sdk) Install(version base.Version) error {
	label := b.Label(version)
	if b.CheckExists(version) {
		fmt.Printf("%s is already installed\n", label)
		return nil
	}
	installInfo, err := b.Plugin.PreInstall(version)
	if err != nil {
		return fmt.Errorf("plugin [PreInstall] method error: %w", err)
	}
	if installInfo == nil {
		return fmt.Errorf("no information about the current version")
	}
	mainSdk := installInfo.Main

	sdkVersion := base.Version(mainSdk.Version)
	// A second check is required because the plug-in may change the version number,
	// for example, latest is resolved to a specific version number.
	label = b.Label(sdkVersion)
	if b.CheckExists(sdkVersion) {
		fmt.Printf("%s is already installed\n", label)
		return nil
	}
	success := false
	newDirPath := b.VersionPath(sdkVersion)

	sigs := make(chan os.Signal, 1)

	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		_ = <-sigs
		if !success {
			_ = os.RemoveAll(newDirPath)
		}
		os.Exit(0)
	}()

	// Delete directory after failed installation
	defer func() {
		if !success {
			_ = os.RemoveAll(newDirPath)
		}
	}()
	var installedSdkInfos []*base.Info
	path, err := b.preInstallSdk(mainSdk, newDirPath)
	if err != nil {
		return err
	}
	installedSdkInfos = append(installedSdkInfos, &base.Info{
		Name:    mainSdk.Name,
		Version: mainSdk.Version,
		Note:    mainSdk.Note,
		Path:    path,
	})
	if len(installInfo.Additions) > 0 {
		pterm.Printf("There are %d additional files that need to be downloaded...\n", len(installInfo.Additions))
		for _, oSdk := range installInfo.Additions {
			path, err = b.preInstallSdk(oSdk, newDirPath)
			if err != nil {
				return err
			}
			installedSdkInfos = append(installedSdkInfos, &base.Info{
				Name:    oSdk.Name,
				Version: oSdk.Version,
				Path:    path,
			})
		}
	}
	err = b.Plugin.PostInstall(newDirPath, installedSdkInfos)
	if err != nil {
		return fmt.Errorf("plugin [PostInstall] method error: %w", err)
	}
	success = true
	pterm.Printf("Install %s success! \n", pterm.LightGreen(label))
	useCommand := fmt.Sprintf("vfox use %s", label)
	pterm.Printf("Please use `%s` to use it.\n", pterm.LightBlue(useCommand))

	// Copy command to clipboard in TTY mode
	if util.IsTTY() {
		if err := util.CopyToClipboard(useCommand); err == nil {
			pterm.Printf("%s\n", pterm.LightYellow("Copied to clipboard, you can paste it now."))
		}
		// Silently ignore clipboard errors (not supported, utility not found, etc.)
	}

	return nil
}

func (b *Sdk) moveLocalFile(info *base.Info, targetPath string) error {
	pterm.Printf("Moving %s to %s...\n", info.Path, targetPath)
	if err := util.MoveFiles(info.Path, targetPath); err != nil {
		return fmt.Errorf("failed to move file, err:%w", err)
	}
	return nil
}

func (b *Sdk) moveRemoteFile(info *base.Info, targetPath string) error {
	u, err := url.Parse(info.Path)
	label := info.Label()
	if err != nil {
		return err
	}
	filePath, err := b.Download(u, info.Headers)
	if err != nil {
		return fmt.Errorf("failed to download %s file, err:%w", label, err)
	}
	defer func() {
		// del cache file
		_ = os.Remove(filePath)
	}()
	pterm.Printf("Verifying checksum %s...\n", info.Checksum.Value)
	checksum := info.Checksum.Verify(filePath)
	if !checksum {
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
func (b *Sdk) preInstallSdk(info *base.Info, sdkDestPath string) (string, error) {
	pterm.Printf("Preinstalling %s...\n", info.Label())
	path := info.StoragePath(sdkDestPath)
	if !util.FileExists(path) {
		if err := os.MkdirAll(path, 0755); err != nil {
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

func (b *Sdk) Uninstall(version base.Version) (err error) {
	label := b.Label(version)
	if !b.CheckExists(version) {
		return fmt.Errorf("%s is not installed", pterm.Red(label))
	}
	path := b.VersionPath(version)
	sdkPackage, err := b.GetLocalSdkPackage(version)
	if err != nil {
		return
	}
	// Give the plugin a chance before actually uninstalling targeted version.
	err = b.Plugin.PreUninstall(sdkPackage)
	if err != nil {
		return
	}

	if b.Current() == version {
		if err = b.ClearCurrentEnv(); err != nil {
			return err
		}

		tv, err := toolset.NewToolVersion(b.sdkManager.PathMeta.User.Home)
		if err != nil {
			return err
		}
		delete(tv.Record, b.Name)
		_ = tv.Save()
	}

	err = os.RemoveAll(path)
	if err != nil {
		return
	}
	pterm.Printf("Uninstalled %s successfully!\n", label)
	return
}

func (b *Sdk) Available(args []string) ([]*base.Package, error) {
	return b.Plugin.Available(args)
}

func (b *Sdk) ToLinkPackage(version base.Version, location base.Location) error {
	linkPackage, err := b.GetLinkPackage(version, location)
	if err != nil {
		return err
	}
	_, err = linkPackage.Link()
	return err
}

// GetLinkPackage will make symlink according base.Location and return the sdk package.
func (b *Sdk) GetLinkPackage(version base.Version, location base.Location) (*LocationPackage, error) {
	return newLocationPackage(version, b, location)
}

// MockEnvKeys It just simulates to get the environment configuration information,
// if the corresponding location of the package does not exist, then it will return
// the empty environment information, without calling the EnvKeys hook.
func (b *Sdk) MockEnvKeys(version base.Version, location base.Location) (*env.Envs, error) {
	label := b.Label(version)
	if !b.CheckExists(version) {
		return nil, fmt.Errorf("%s is not installed", label)
	}
	linkPackage, err := b.GetLinkPackage(version, location)
	if err != nil {
		return nil, err
	}
	sdkPackage := linkPackage.ConvertLocation()
	if !checkPackageValid(sdkPackage) {
		logger.Debugf("Package is invalid: %v\n", sdkPackage)
		return &env.Envs{
			Variables: make(env.Vars),
			Paths:     env.NewPaths(env.EmptyPaths),
			BinPaths:  env.NewPaths(env.EmptyPaths),
		}, nil
	}
	keys, err := b.Plugin.EnvKeys(sdkPackage)
	if err != nil {
		return nil, fmt.Errorf("plugin [EnvKeys] error: err:%w", err)
	}
	return keys, nil
}

// EnvKeys Only return the really installed path of this SDK
func (b *Sdk) EnvKeys(version base.Version) (*env.Envs, error) {
	label := b.Label(version)
	if !b.CheckExists(version) {
		return nil, fmt.Errorf("%s is not installed", label)
	}

	sdkInstalledPackage, err := b.GetLocalSdkPackage(version)
	if err != nil {
		return nil, err
	}
	keys, err := b.Plugin.EnvKeys(sdkInstalledPackage)
	if err != nil {
		return nil, fmt.Errorf("plugin [EnvKeys] error: err:%w", err)
	}
	return keys, nil
}

func (b *Sdk) PreUse(version base.Version, scope base.UseScope) (base.Version, error) {
	installedSdks := b.getLocalSdkPackages()
	newVersion, err := b.Plugin.PreUse(version, b.Current(), scope, b.sdkManager.PathMeta.Working.Directory, installedSdks)
	if err != nil {
		return "", fmt.Errorf("plugin [PreUse] error: err:%w", err)
	}

	// If the plugin does not return a version, it means that the plugin does
	// not want to change the version or not implement the PreUse function.
	// We can simply fuzzy match the version based on the input version.
	if newVersion == "" {
		// Before fuzzy matching, perform exact matching first.
		if b.CheckExists(version) {
			return version, nil
		}
		installedVersions := make(util.VersionSort, 0, len(installedSdks))
		for _, sdk := range installedSdks {
			installedVersions = append(installedVersions, string(sdk.Main.Version))
		}
		sort.Sort(installedVersions)
		version := string(version)
		prefix := version + "."
		for _, v := range installedVersions {
			if version == v {
				newVersion = base.Version(v)
				break
			}
			if strings.HasPrefix(v, prefix) {
				newVersion = base.Version(v)
				break
			}
		}
	}
	if newVersion == "" {
		return version, nil
	}

	return newVersion, nil
}

type VersionNotExistsError struct {
	Label string
}

func (e *VersionNotExistsError) Error() string {
	return fmt.Sprintf("%s is not installed", e.Label)
}

func (b *Sdk) Use(version base.Version, scope base.UseScope) error {
	logger.Debugf("Use SDK version: %s, scope: %v\n", string(version), scope)

	// Verify hook environment is available
	if !env.IsHookEnv() {
		return fmt.Errorf("vfox requires hook support. Please ensure vfox is properly initialized with 'vfox activate'")
	}

	// Resolve version with PreUse hook
	resolvedVersion, err := b.PreUse(version, scope)
	if err != nil {
		return err
	}

	// Verify version exists
	label := b.Label(resolvedVersion)
	if !b.CheckExists(resolvedVersion) {
		return &VersionNotExistsError{Label: label}
	}

	return b.useInHook(resolvedVersion, scope)
}

func (b *Sdk) useInHook(version base.Version, scope base.UseScope) error {

	// 特殊，需要判断是否存在目录，不存在在创建
	if base.Project == scope {
		if !util.FileExists(b.sdkManager.PathMeta.Working.Directory) {
			err := os.MkdirAll(b.sdkManager.PathMeta.Working.Directory, 0755)
			if err != nil {
				return fmt.Errorf("failed to create .vfox directory: %w", err)
			}
		}
	}

	toolVersions, err := b.sdkManager.LoadToolVersionByScope(scope)
	if err != nil {
		return err
	}

	logger.Debugf("Load %v tool versions: %v\n", scope, toolVersions.Record)

	// Update version record in all affected scopes
	toolVersions.Record[b.Name] = string(version)
	if err := toolVersions.Save(); err != nil {
		return fmt.Errorf("failed to save tool versions, err:%w", err)
	}

	// Create symlinks for the specified scope
	if err := b.createSymlinksForScope(version, scope); err != nil {
		return fmt.Errorf("failed to create symlinks, err:%w", err)
	}

	pterm.Printf("Now using %s.\n", pterm.LightGreen(b.Label(version)))
	return nil
}

func (b *Sdk) List() []base.Version {
	if !util.FileExists(b.InstallPath) {
		return make([]base.Version, 0)
	}
	var versions []base.Version
	dir, err := os.ReadDir(b.InstallPath)
	if err != nil {
		return nil
	}
	for _, d := range dir {
		if d.IsDir() && strings.HasPrefix(d.Name(), "v-") {
			versions = append(versions, base.Version(strings.TrimPrefix(d.Name(), "v-")))
		}
	}
	sort.Slice(versions, func(i, j int) bool {
		return versions[i] > versions[j]
	})
	return versions
}

func (b *Sdk) getLocalSdkPackages() []*base.Package {
	var infos []*base.Package
	for _, version := range b.List() {
		info, err := b.GetLocalSdkPackage(version)
		if err != nil {
			continue
		}
		infos = append(infos, info)
	}
	return infos
}

// Current returns the current version of the SDK.
// Lookup priority is: project > session > global
func (b *Sdk) Current() base.Version {
	toolVersion, err := toolset.NewMultiToolVersions([]string{
		b.sdkManager.PathMeta.Working.Directory,
		b.sdkManager.PathMeta.Working.SessionShim,
		b.sdkManager.PathMeta.User.Home,
	})
	if err != nil {
		return ""
	}
	current := toolVersion.FilterTools(func(name, version string) bool {
		return name == b.Name && b.CheckExists(base.Version(version))
	})
	if len(current) == 0 {
		return ""
	}
	return base.Version(current[b.Name])
}

func (b *Sdk) ParseLegacyFile(path string) (base.Version, error) {
	return b.Plugin.ParseLegacyFile(path, func() []base.Version {
		return b.List()
	})
}

func (b *Sdk) Close() {
	b.Plugin.Close()
}

// clearGlobalEnv Mainly used to clear record from Windows registry
func (b *Sdk) clearGlobalEnv(version base.Version) {
	if version == "" {
		return
	}
	sdkPackage, err := b.GetLinkPackage(version, base.GlobalLocation)
	if err != nil {
		return
	}
	envKV, err := b.Plugin.EnvKeys(sdkPackage.ConvertLocation())
	if err != nil {
		return
	}
	// Compatible symbolic link paths for v0.5.0-0.5.2
	envKV.Paths.Add(filepath.Join(b.InstallPath, "current"))
}

// createSymlinksForScope creates symlinks in the appropriate SDK directory for the scope
func (b *Sdk) createSymlinksForScope(version base.Version, scope base.UseScope) error {
	// Get SDK binaries
	envKeys, err := b.EnvKeys(version)
	if err != nil {
		return fmt.Errorf("failed to get env keys for %s, err:%w", b.Label(version), err)
	}

	binPaths, err := envKeys.Paths.ToBinPaths()
	if err != nil {
		return fmt.Errorf("failed to get bin paths, err:%w", err)
	}

	// Determine target SDK directory based on scope
	var sdkDir string
	switch scope {
	case base.Global:
		sdkDir = b.sdkManager.PathMeta.Working.GlobalShim
	case base.Project:
		sdkDir = b.sdkManager.PathMeta.Working.ProjectShim
	case base.Session:
		sdkDir = b.sdkManager.PathMeta.Working.SessionShim
	}

	// Whatever scope used, the session scope must be created symlinks.
	if scope != base.Session {
		_ = b.createBinSymlinks(binPaths, b.sdkManager.PathMeta.Working.SessionShim)
	}
	// Create symlinks for each binary
	return b.createBinSymlinks(binPaths, sdkDir)

}

// createBinSymlinks creates symlinks for all binaries in the given paths
func (b *Sdk) createBinSymlinks(binPaths *env.Paths, targetDir string) error {
	// Ensure target directory exists
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create SDK directory %s, err:%w", targetDir, err)
	}

	for _, binPath := range binPaths.Slice() {
		binName := filepath.Base(binPath)
		binShim := shim.NewShim(binPath, targetDir)
		if err := binShim.Generate(); err != nil {
			logger.Debugf("Failed to create symlink for %s, err:%v\n", binName, err)
			continue
		}
	}
	return nil
}

func (b *Sdk) GetLocalSdkPackage(version base.Version) (*base.Package, error) {
	p, ok := b.localSdkPackageCache[version]
	if ok {
		return p, nil
	}
	versionPath := b.VersionPath(version)
	items := make(map[string]*base.Info)
	dir, err := os.ReadDir(versionPath)
	if err != nil {
		return nil, err
	}
	for _, d := range dir {
		if d.IsDir() {
			if strings.HasSuffix(d.Name(), string(version)) {
				name := strings.TrimSuffix(d.Name(), "-"+string(version))
				if name == "" {
					continue
				}
				logger.Debugf("Load SDK package item: name:%s, version: %s \n", name, version)
				items[name] = &base.Info{
					Name:    name,
					Version: string(version),
					Path:    filepath.Join(versionPath, d.Name()),
				}
			}
		}
	}
	main, ok := items[b.Plugin.Name]
	if !ok {
		return nil, errors.New("main sdk not found")
	}
	delete(items, b.Plugin.Name)
	if main.Path == "" {
		return nil, errors.New("main sdk not found")
	}
	var additions []*base.Info
	for _, v := range items {
		additions = append(additions, v)
	}
	p2 := &base.Package{
		Main:      main,
		Additions: additions,
	}
	b.localSdkPackageCache[version] = p2
	return p2, nil
}

func (b *Sdk) CheckExists(version base.Version) bool {
	return util.FileExists(b.VersionPath(version))
}

func (b *Sdk) VersionPath(version base.Version) string {
	return filepath.Join(b.InstallPath, fmt.Sprintf("v-%s", version))
}

func (b *Sdk) Download(u *url.URL, headers map[string]string) (string, error) {
	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return "", err
	}
	for key, value := range headers {
		req.Header.Add(key, value)
	}
	resp, err := b.sdkManager.HttpClient().Do(req)
	if err != nil {
		var urlErr *url.Error
		if errors.As(err, &urlErr) {
			var netErr net.Error
			if errors.As(urlErr.Err, &netErr) && netErr.Timeout() {
				return "", errors.New("request timeout")
			}
		}
		return "", err
	}

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return "", errors.New("source file not found")
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

func (b *Sdk) Label(version base.Version) string {
	return fmt.Sprintf("%s@%s", strings.ToLower(b.Name), version)
}

func (b *Sdk) ClearCurrentEnv() error {
	current := b.Current()

	if current != "" {
		envKeys, err := b.MockEnvKeys(current, base.GlobalLocation)
		if err != nil {
			return err
		}
		fmt.Println("Cleaning up the shims...")
		if paths, err := envKeys.Paths.ToBinPaths(); err == nil {
			for _, p := range paths.Slice() {
				if err = shim.NewShim(p, b.sdkManager.PathMeta.Working.GlobalShim).Clear(); err != nil {
					return err
				}
			}
		}

		fmt.Println("Cleaning up env config...")
		envKeys.Paths.Add(filepath.Join(b.InstallPath, "current"))

		envKeys, err = b.MockEnvKeys(current, base.OriginalLocation)
		if err != nil {
			return err
		}
	}

	fmt.Println("Cleaning up current link...")
	curPath := filepath.Join(b.InstallPath, "current")
	if err := os.RemoveAll(curPath); err != nil {
		return fmt.Errorf("failed to remove current link, err:%w", err)
	}

	// clear tool versions
	toolVersion, err := toolset.NewMultiToolVersions([]string{
		b.sdkManager.PathMeta.Working.SessionShim,
		b.sdkManager.PathMeta.User.Home,
	})
	if err != nil {
		return err
	}
	for _, tv := range toolVersion {
		delete(tv.Record, b.Name)
	}
	return nil
}

// Unuse removes the version setting for the SDK from the specified scope
func (b *Sdk) Unuse(scope base.UseScope) error {
	var multiToolVersion toolset.MultiToolVersions

	if scope == base.Global {
		toolVersion, err := toolset.NewToolVersion(b.sdkManager.PathMeta.User.Home)
		if err != nil {
			return fmt.Errorf("failed to read tool versions, err:%w", err)
		}

		// Check if the SDK is currently set globally
		if oldVersion, ok := toolVersion.Record[b.Name]; ok {
			// Clear global environment for the current version
			b.clearGlobalEnv(base.Version(oldVersion))

			// Remove shims for the current version
			envKeys, err := b.MockEnvKeys(base.Version(oldVersion), base.GlobalLocation)
			if err == nil {
				if paths, err := envKeys.Paths.ToBinPaths(); err == nil {
					for _, p := range paths.Slice() {
						if err = shim.NewShim(p, b.sdkManager.PathMeta.Working.GlobalShim).Clear(); err != nil {
							// Log but don't fail on shim cleanup errors
							logger.Debugf("Failed to clear shim %s: %v\n", p, err)
						}
					}
				}
			}

			// Flush environment changes
		}

		// Remove from global tool versions
		delete(toolVersion.Record, b.Name)
		multiToolVersion = append(multiToolVersion, toolVersion)

	} else if scope == base.Project {
		toolVersion, err := toolset.NewToolVersion(b.sdkManager.PathMeta.Working.Directory)
		if err != nil {
			return fmt.Errorf("failed to read tool versions, err:%w", err)
		}

		// Remove from project tool versions
		delete(toolVersion.Record, b.Name)
		multiToolVersion = append(multiToolVersion, toolVersion)
	}

	// For session scope, or in addition to global/project scope,
	// also remove from the session level
	sessionToolVersion, err := toolset.NewToolVersion(b.sdkManager.PathMeta.Working.SessionShim)
	if err != nil {
		return fmt.Errorf("failed to read tool versions, err:%w", err)
	}
	delete(sessionToolVersion.Record, b.Name)
	multiToolVersion = append(multiToolVersion, sessionToolVersion)

	// Save all modified tool version files
	if err = multiToolVersion.Save(); err != nil {
		return fmt.Errorf("failed to save tool versions, err:%w", err)
	}

	pterm.Printf("Unset %s successfully.\n", pterm.LightGreen(b.Name))

	// Reopen shell to apply changes if not in hook environment
	if !env.IsHookEnv() {
		return shell.Open(os.Getppid())
	}
	return nil
}

// NewSdk creates a new SDK instance.
func NewSdk(manager *Manager, pluginPath string) (*Sdk, error) {
	sdkName := filepath.Base(pluginPath)
	plugin, err := plugin.CreatePluginFromPath(pluginPath, manager.Config, RuntimeVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to create plugin: %w", err)
	}

	// SDK install path fixed to shared root (single location)
	installPath := filepath.Join(
		manager.PathMeta.Shared.Installs,
		strings.ToLower(sdkName),
	)

	return &Sdk{
		Name:                 sdkName,
		sdkManager:           manager,
		InstallPath:          installPath,
		Plugin:               plugin,
		localSdkPackageCache: make(map[base.Version]*base.Package),
	}, nil
}
