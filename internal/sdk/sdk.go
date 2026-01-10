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
	"github.com/version-fox/vfox/internal/env"
	"github.com/version-fox/vfox/internal/pathmeta"
	"github.com/version-fox/vfox/internal/plugin"
	"github.com/version-fox/vfox/internal/shared/logger"
	"github.com/version-fox/vfox/internal/shared/util"
)

const (
	packageInstalledPrefix = "v-"   // Prefix of path for installed SDK packages
	additionRuntimePrefix  = "add-" // Prefix of path for additional runtime packages
)

// Sdk interface defines the methods for managing software development kits (SDKs).
type Sdk interface {
	Install(version Version) error                                        // Install a specific runtime of the SDK
	Uninstall(version Version) error                                      // Uninstall a specific runtime of the SDK
	Available(args []string) ([]*AvailableRuntimePackage, error)          // List available runtime of the SDK
	EnvKeys(runtimePackage *RuntimePackage) (*env.Envs, error)            // Get environment variables for a specific runtime of the SDK
	Use(version Version, scope env.UseScope) error                        // Use a specific runtime in a given scope
	UseWithConfig(version Version, scope env.UseScope, unlink bool) error // Use with link configuration
	Unuse(scope env.UseScope) error                                       // Unuse the current runtime in a given scope
	GetRuntimePackage(version Version) (*RuntimePackage, error)           // Get the runtime package for a specific version
	CheckRuntimeExist(version Version) bool                               // Check if a specific runtime version is installed
	InstalledList() []Version
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

func (b *impl) Available(args []string) ([]*AvailableRuntimePackage, error) {
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

// Install installs a specific version of the SDK.
// For main runtime, it will be installed to {InstallPath}/v-{main_version}/{main_name}-{main_version}
// For additional runtimes, it will be installed to {InstallPath}/v-{main_version}/add-{addition_name}-{addition_version}
func (b *impl) Install(version Version) error {
	label := b.Label(version)
	if b.CheckRuntimeExist(version) {
		fmt.Printf("%s is already installed\n", label)
		return nil
	}

	ctx := &plugin.PreInstallHookCtx{
		Version: string(version),
	}
	installInfo, err := b.plugin.PreInstall(ctx)
	if b.plugin.IsNoResultProvided(err) {
		return fmt.Errorf("no installable runtime provided")
	}
	if err != nil {
		return fmt.Errorf("plugin [PreInstall] method error: %w", err)
	}
	if installInfo == nil {
		return fmt.Errorf("no information about the current version")
	}

	mainSdk := installInfo.PreInstallPackageItem
	mainSdk.Name = b.plugin.Name

	sdkVersion := Version(mainSdk.Version)
	// A second check is required because the plug-in may change the version number,
	// for example, latest is resolved to a specific version number.
	label = b.Label(sdkVersion)
	if b.CheckRuntimeExist(sdkVersion) {
		fmt.Printf("%s is already installed\n", label)
		return nil
	}
	success := false
	newDirPath := b.packagePath(sdkVersion)

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
	installedPackage := make(map[string]*plugin.InstalledPackageItem)

	path, err := b.preInstallSdk(mainSdk, filepath.Join(newDirPath, b.runtimePathDirName(true, mainSdk)))

	if err != nil {
		return err
	}
	installedPackage[mainSdk.Name] = &plugin.InstalledPackageItem{
		Name:    mainSdk.Name,
		Version: mainSdk.Version,
		Path:    path,
	}
	if len(installInfo.Addition) > 0 {
		pterm.Printf("There are %d additional files that need to be downloaded...\n", len(installInfo.Addition))
		for _, oSdk := range installInfo.Addition {
			path, err = b.preInstallSdk(oSdk, filepath.Join(newDirPath, b.runtimePathDirName(false, oSdk)))
			if err != nil {
				return err
			}
			installedPackage[oSdk.Name] = &plugin.InstalledPackageItem{
				Name:    oSdk.Name,
				Version: oSdk.Version,
				Path:    path,
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
			return fmt.Errorf("plugin [PostInstall] method error: %w", err)
		}
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

func (b *impl) moveLocalFile(info *plugin.PreInstallPackageItem, targetPath string) error {
	pterm.Printf("Moving %s to %s...\n", info.Path, targetPath)
	if err := util.MoveFiles(info.Path, targetPath); err != nil {
		return fmt.Errorf("failed to move file, err:%w", err)
	}
	return nil
}

func (b *impl) moveRemoteFile(info *plugin.PreInstallPackageItem, targetPath string) error {
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

func (b *impl) runtimePathDirName(isMain bool, info *plugin.PreInstallPackageItem) string {
	var name string
	if isMain {
		if info.Version == "" {
			// {name}
			name = info.Name
		} else {
			// {name}-{version}
			name = info.Name + "-" + info.Version
		}
	} else {
		if info.Version == "" {
			// add-{name}
			name = additionRuntimePrefix + info.Name
		} else {
			// add-{name}-{version}
			name = additionRuntimePrefix + info.Name + "-" + info.Version
		}
	}
	return name
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
	if !b.CheckRuntimeExist(version) {
		return fmt.Errorf("%s is not installed", pterm.Red(label))
	}
	path := b.packagePath(version)
	sdkPackage, err := b.GetRuntimePackage(version)
	if err != nil {
		return
	}
	if b.plugin.HasFunction("PreUninstall") {
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
			return
		}
	}

	// If the version to be uninstalled is currently in use, unuse it first.
	if b.Current() == version {
		if err = b.Unuse(env.Global); err != nil {
			return err
		}
	}

	err = os.RemoveAll(path)
	if err != nil {
		return
	}
	pterm.Printf("Uninstalled %s successfully!\n", label)
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

func (b *impl) preUse(version Version, scope env.UseScope) (Version, error) {
	if !b.plugin.HasFunction("PreUse") {
		logger.Debug("plugin does not have PreUse function")
		return "", nil
	}
	installedSdks, err := b.getAllRuntimes()
	sdks := make(map[string]*plugin.InstalledPackageItem)

	for _, sdk := range installedSdks {
		sdks[sdk.Runtime.Name] = &plugin.InstalledPackageItem{
			Name:    sdk.Runtime.Name,
			Version: string(sdk.Runtime.Version),
			Path:    sdk.PackagePath,
		}
	}
	preUseCtx := &plugin.PreUseHookCtx{
		Cwd:             b.envContext.PathMeta.Working.Directory,
		PreviousVersion: string(b.Current()),
		Scope:           scope.String(),
		Version:         string(version),
		InstalledSdks:   sdks,
	}
	preUseResult, err := b.plugin.PreUse(preUseCtx)
	if err != nil {
		return "", fmt.Errorf("plugin [preUse] error: err:%w", err)
	}

	newVersion := preUseResult.Version

	// If the plugin does not return a version, it means that the plugin does
	// not want to change the version or not implement the preUse function.
	// We can simply fuzzy match the version based on the input version.
	if newVersion == "" {
		// Before fuzzy matching, perform exact matching first.
		if b.CheckRuntimeExist(version) {
			return version, nil
		}
		installedVersions := make(util.VersionSort, 0, len(installedSdks))
		for _, sdk := range installedSdks {
			installedVersions = append(installedVersions, string(sdk.Runtime.Version))
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
	if !env.IsHookEnv() {
		return fmt.Errorf("vfox requires hook support. Please ensure vfox is properly initialized with 'vfox activate'")
	}

	// Resolve version with preUse hook
	resolvedVersion, err := b.preUse(version, scope)
	if err != nil {
		return err
	}

	// Verify version exists
	label := b.Label(resolvedVersion)
	if !b.CheckRuntimeExist(resolvedVersion) {
		return &VersionNotExistsError{Label: label}
	}

	return b.useInHook(resolvedVersion, scope, unlink)
}

func (b *impl) useInHook(version Version, scope env.UseScope, unlink bool) error {

	runtimePackage, err := b.GetRuntimePackage(version)
	if err != nil {
		logger.Debugf("Failed to get runtime package for %s: %v", b.Label(version), err)
		return ErrRuntimeNotFound
	}

	// Ensure .vfox directory exists for project scope
	if env.Project == scope {
		if !util.FileExists(b.envContext.PathMeta.Working.ProjectSdkDir) {
			err := os.MkdirAll(b.envContext.PathMeta.Working.ProjectSdkDir, pathmeta.ReadWriteAuth)
			if err != nil {
				return fmt.Errorf("failed to create .vfox directory: %w", err)
			}
		}
	}

	// Load config for the specified scope
	vfoxToml, err := b.envContext.LoadVfoxTomlByScope(scope)
	if err != nil {
		return err
	}

	logger.Debugf("Load %v tool versions: %v\n", scope, vfoxToml.GetAllTools())

	// Determine link flag based on scope and unlink parameter
	if scope == env.Project && unlink {
		attr := pathmeta.Attr{
			UnLinkAttrFlag: BoolYes,
		}
		// Update version record
		vfoxToml.SetToolWithAttr(b.Name, string(version), attr)
		if err := vfoxToml.Save(); err != nil {
			return fmt.Errorf("failed to save tool versions, err:%w", err)
		}
	} else {
		// Update version record
		vfoxToml.SetTool(b.Name, string(version))
		if err := vfoxToml.Save(); err != nil {
			return fmt.Errorf("failed to save tool versions, err:%w", err)
		}
	}

	// Create symlinks for the specified scope
	if err := b.createSymlinksForScope(runtimePackage, scope); err != nil {
		return fmt.Errorf("failed to create symlinks, err:%w", err)
	}

	pterm.Printf("Now using %s.\n", pterm.LightGreen(b.Label(version)))
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
			versions = append(versions, Version(strings.TrimPrefix(d.Name(), "v-")))
		}
	}
	sort.Slice(versions, func(i, j int) bool {
		return versions[i] > versions[j]
	})
	return versions
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
	if len(legacyFilenames) > 0 {
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
				versions := b.InstalledList()
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
	versionPath := b.packagePath(version)
	items := make(map[string]*Runtime)
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
	return p2, nil
}

func (b *impl) CheckRuntimeExist(version Version) bool {
	return util.FileExists(b.packagePath(version))
}

func (b *impl) packagePath(version Version) string {
	return filepath.Join(b.InstallPath, packageInstalledPrefix+string(version))
}

func (b *impl) Download(u *url.URL, headers map[string]string) (string, error) {
	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return "", err
	}
	for key, value := range headers {
		req.Header.Add(key, value)
	}
	resp, err := b.envContext.HttpClient().Do(req)
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

func (b *impl) Label(version Version) string {
	return fmt.Sprintf("%s@%s", strings.ToLower(b.Name), version)
}

// Unuse removes the version setting for the SDK from the specified scope
func (b *impl) Unuse(scope env.UseScope) error {
	// Create a config chain to manage all affected scopes
	chain := env.NewVfoxTomlChain()

	if scope == env.Global {
		globalConfig, err := b.envContext.LoadVfoxTomlByScope(env.Global)
		if err != nil {
			return fmt.Errorf("failed to read global config: %w", err)
		}

		// Check if the SDK is currently set globally
		if version, ok := globalConfig.GetToolVersion(b.Name); ok {
			// Remove shims for the current version
			if err := b.removeSymlinksForScope(Version(version), env.Global); err != nil {
				logger.Debugf("Failed to remove global symlinks: %v\n", err)
			}
		}

		// Remove from global config
		globalConfig.RemoveTool(b.Name)
		chain.Add(globalConfig, scope)

	} else if scope == env.Project {
		projectConfig, err := b.envContext.LoadVfoxTomlByScope(env.Project)
		if err != nil {
			return fmt.Errorf("failed to read project config: %w", err)
		}

		// Check if the SDK is currently set in project
		if version, ok := projectConfig.GetToolVersion(b.Name); ok {
			// Remove shims for the current version
			if err := b.removeSymlinksForScope(Version(version), env.Project); err != nil {
				logger.Debugf("Failed to remove project symlinks: %v\n", err)
			}
		}

		// Remove from project config
		projectConfig.RemoveTool(b.Name)
		chain.Add(projectConfig, scope)
	} else if scope == env.Session {
		projectConfig, err := b.envContext.LoadVfoxTomlByScope(env.Project)
		if err != nil {
			return fmt.Errorf("failed to read project config: %w", err)
		}

		// Check if the SDK is currently set in project
		if version, ok := projectConfig.GetToolVersion(b.Name); ok {
			// Remove shims for the current version
			if err := b.removeSymlinksForScope(Version(version), env.Project); err != nil {
				logger.Debugf("Failed to remove project symlinks: %v\n", err)
			}
		}

		// Remove from project config
		projectConfig.RemoveTool(b.Name)
		chain.Add(projectConfig, scope)
	}

	// For session scope, or in addition to global/project scope,
	// also remove from the session level
	sessionConfig, err := b.envContext.LoadVfoxTomlByScope(env.Session)
	if err != nil {
		return fmt.Errorf("failed to read session config: %w", err)
	}

	// Check if the SDK is currently set in session
	if version, ok := sessionConfig.GetToolVersion(b.Name); ok {
		// Remove shims for the current version
		if err := b.removeSymlinksForScope(Version(version), env.Session); err != nil {
			logger.Debugf("Failed to remove session symlinks: %v\n", err)
		}
	}

	sessionConfig.RemoveTool(b.Name)
	chain.Add(sessionConfig, scope)

	// Save all modified configs
	if err = chain.Save(); err != nil {
		return fmt.Errorf("failed to save configs: %w", err)
	}

	pterm.Printf("Unset %s successfully.\n", pterm.LightGreen(b.Name))

	return nil
}

func (b *impl) getAllRuntimes() ([]*RuntimePackage, error) {
	versions := b.InstalledList()
	var runtimes []*RuntimePackage
	for _, v := range versions {
		runtime, err := b.GetRuntimePackage(v)
		if err != nil {
			return nil, err
		}
		runtimes = append(runtimes, runtime)
	}
	return runtimes, nil
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

	return &impl{
		Name:        sdkName,
		InstallPath: installPath,
		envContext:  runtimeEnvContext,
		plugin:      plugin,
	}, nil
}
