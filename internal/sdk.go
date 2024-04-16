/*
 *    Copyright 2024 Han Li and contributors
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
	"github.com/version-fox/vfox/internal/toolset"
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

	"github.com/schollz/progressbar/v3"
	"github.com/version-fox/vfox/internal/env"
	"github.com/version-fox/vfox/internal/logger"
	"github.com/version-fox/vfox/internal/shell"

	"github.com/pterm/pterm"
	"github.com/version-fox/vfox/internal/util"
)

type Version string

type Sdk struct {
	sdkManager *Manager
	Plugin     *LuaPlugin
	// current sdk install path
	InstallPath string
}

func (b *Sdk) Install(version Version) error {
	label := b.label(version)
	if b.checkExists(version) {
		return fmt.Errorf("%s is already installed", label)
	}
	installInfo, err := b.Plugin.PreInstall(version)
	if err != nil {
		return fmt.Errorf("plugin [PreInstall] method error: %w", err)
	}
	if installInfo == nil {
		return fmt.Errorf("no information about the current version")
	}
	mainSdk := installInfo.Main
	success := false
	newDirPath := b.VersionPath(mainSdk.Version)

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
	// A second check is required because the plug-in may change the version number,
	// for example, latest is resolved to a specific version number.
	label = b.label(mainSdk.Version)
	if b.checkExists(mainSdk.Version) {
		return fmt.Errorf("%s is already installed", label)
	}
	var installedSdkInfos []*Info
	path, err := b.preInstallSdk(mainSdk, newDirPath)
	if err != nil {
		return err
	}
	installedSdkInfos = append(installedSdkInfos, &Info{
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
			installedSdkInfos = append(installedSdkInfos, &Info{
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
	pterm.Printf("Please use %s to use it.\n", pterm.LightBlue(fmt.Sprintf("vfox use %s", label)))
	return nil
}

func (b *Sdk) moveLocalFile(info *Info, targetPath string) error {
	pterm.Printf("Moving %s to %s...\n", info.Path, targetPath)
	if err := util.MoveFiles(info.Path, targetPath); err != nil {
		return fmt.Errorf("failed to move file, err:%w", err)
	}
	return nil
}

func (b *Sdk) moveRemoteFile(info *Info, targetPath string) error {
	u, err := url.Parse(info.Path)
	label := info.label()
	if err != nil {
		return err
	}
	filePath, err := b.Download(u)
	if err != nil {
		return fmt.Errorf("failed to download %s file, err:%w", label, err)
	}
	defer func() {
		// del cache file
		_ = os.Remove(filePath)
	}()
	pterm.Printf("Verifying checksum %s...\n", info.Checksum.Value)
	checksum := info.Checksum.verify(filePath)
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
func (b *Sdk) preInstallSdk(info *Info, sdkDestPath string) (string, error) {
	pterm.Printf("Preinstalling %s...\n", info.label())
	path := info.storagePath(sdkDestPath)
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

func (b *Sdk) Uninstall(version Version) error {
	label := b.label(version)
	if !b.checkExists(version) {
		pterm.Printf("%s is not installed...\n", pterm.Red(label))
		return fmt.Errorf("%s is not installed", label)
	}
	if b.Current() == version {
		b.clearEnvConfig(version)
	}
	path := b.VersionPath(version)
	err := os.RemoveAll(path)
	if err != nil {
		return err
	}
	pterm.Printf("Uninstalled %s successfully!\n", label)
	return nil
}

func (b *Sdk) Available(args []string) ([]*Package, error) {
	return b.Plugin.Available(args)
}

func (b *Sdk) EnvKeys(version Version) (*env.Envs, error) {
	label := b.label(version)
	if !b.checkExists(version) {
		return nil, fmt.Errorf("%s is not installed", label)
	}
	sdkPackage, err := b.GetLocalSdkPackage(version)
	if err != nil {
		return nil, fmt.Errorf("failed to get local sdk info, err:%w", err)
	}
	keys, err := b.Plugin.EnvKeys(sdkPackage)
	if err != nil {
		return nil, fmt.Errorf("plugin [EnvKeys] error: err:%w", err)
	}
	return keys, nil
}

func (b *Sdk) PreUse(version Version, scope UseScope) (Version, error) {
	if !b.Plugin.HasFunction("PreUse") {
		logger.Debug("plugin does not have PreUse function")
		return version, nil
	}

	newVersion, err := b.Plugin.PreUse(version, b.Current(), scope, b.sdkManager.PathMeta.WorkingDirectory, b.getLocalSdkPackages())
	if err != nil {
		return "", fmt.Errorf("plugin [PreUse] error: err:%w", err)
	}

	// If the plugin does not return a version, it means that the plugin does not want to change the version.
	if newVersion == "" {
		return version, nil
	}

	return newVersion, nil
}

func (b *Sdk) Use(version Version, scope UseScope) error {
	logger.Debugf("use sdk version: %s\n", string(version))

	version, err := b.PreUse(version, scope)
	if err != nil {
		return err
	}

	label := b.label(version)
	if !b.checkExists(version) {
		return fmt.Errorf("%s is not installed", label)
	}

	if !env.IsHookEnv() {
		pterm.Printf("Warning: The current shell lacks hook support or configuration. It has switched to global scope automatically.\n")
		sdkPackage, err := b.GetLocalSdkPackage(version)
		if err != nil {
			pterm.Printf("Failed to get local sdk info, err:%s\n", err.Error())
			return err
		}

		toolVersion, err := toolset.NewToolVersion(b.sdkManager.PathMeta.HomePath)
		if err != nil {
			return fmt.Errorf("failed to read tool versions, err:%w", err)
		}
		keys, err := b.Plugin.EnvKeys(sdkPackage)
		if err != nil {
			return fmt.Errorf("plugin [EnvKeys] method error: %w", err)
		}

		// clear global env
		if oldVersion, ok := toolVersion.Record[b.Plugin.SdkName]; ok {
			b.clearGlobalEnv(Version(oldVersion))
		}

		if err = b.sdkManager.EnvManager.Load(keys); err != nil {
			return err
		}
		err = b.sdkManager.EnvManager.Flush()
		if err != nil {
			return err
		}
		toolVersion.Record[b.Plugin.SdkName] = string(version)
		if err = toolVersion.Save(); err != nil {
			return fmt.Errorf("failed to save tool versions, err:%w", err)
		}
		return shell.GetProcess().Open(os.Getppid())
	} else {
		return b.useInHook(version, scope)
	}
}

func (b *Sdk) useInHook(version Version, scope UseScope) error {
	var multiToolVersion toolset.MultiToolVersions
	if scope == Global {
		sdkPackage, err := b.GetLocalSdkPackage(version)
		if err != nil {
			pterm.Printf("Failed to get local sdk info, err:%s\n", err.Error())
			return err
		}
		toolVersion, err := toolset.NewToolVersion(b.sdkManager.PathMeta.HomePath)
		if err != nil {
			return fmt.Errorf("failed to read tool versions, err:%w", err)
		}
		keys, err := b.Plugin.EnvKeys(sdkPackage)
		if err != nil {
			return fmt.Errorf("plugin [EnvKeys] method error: %w", err)
		}

		// clear global env
		if oldVersion, ok := toolVersion.Record[b.Plugin.SdkName]; ok {
			b.clearGlobalEnv(Version(oldVersion))
		}

		if err = b.sdkManager.EnvManager.Load(keys); err != nil {
			return err
		}
		err = b.sdkManager.EnvManager.Flush()
		if err != nil {
			return err
		}
		multiToolVersion = append(multiToolVersion, toolVersion)
	} else if scope == Project {
		toolVersion, err := toolset.NewToolVersion(b.sdkManager.PathMeta.WorkingDirectory)
		if err != nil {
			return fmt.Errorf("failed to read tool versions, err:%w", err)
		}
		multiToolVersion = append(multiToolVersion, toolVersion)
	}

	// It must also be saved once at the session level.
	toolVersion, err := toolset.NewToolVersion(b.sdkManager.PathMeta.CurTmpPath)
	if err != nil {
		return fmt.Errorf("failed to read tool versions, err:%w", err)
	}
	multiToolVersion = append(multiToolVersion, toolVersion)

	multiToolVersion.Add(b.Plugin.SdkName, string(version))

	if err = multiToolVersion.Save(); err != nil {
		return fmt.Errorf("failed to save tool versions, err:%w", err)
	}

	pterm.Printf("Now using %s.\n", pterm.LightGreen(b.label(version)))
	if !env.IsHookEnv() {
		return shell.GetProcess().Open(os.Getppid())
	}
	return nil
}

func (b *Sdk) List() []Version {
	if !util.FileExists(b.InstallPath) {
		return make([]Version, 0)
	}
	var versions []Version
	dir, err := os.ReadDir(b.InstallPath)
	if err != nil {
		return nil
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

func (b *Sdk) getLocalSdkPackages() []*Package {
	var infos []*Package
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
func (b *Sdk) Current() Version {
	toolVersion, err := toolset.NewMultiToolVersions([]string{
		b.sdkManager.PathMeta.WorkingDirectory,
		b.sdkManager.PathMeta.CurTmpPath,
		b.sdkManager.PathMeta.HomePath,
	})
	if err != nil {
		return ""
	}
	current := toolVersion.FilterTools(func(name, version string) bool {
		return name == b.Plugin.SdkName && b.checkExists(Version(version))
	})
	if len(current) == 0 {
		return ""
	}
	return Version(current[b.Plugin.SdkName])
}

func (b *Sdk) ParseLegacyFile(path string) (Version, error) {
	return b.Plugin.ParseLegacyFile(path, func() []Version {
		return b.List()
	})
}

func (b *Sdk) Close() {
	b.Plugin.Close()
}

// clearGlobalEnv Mainly used to clear record from Windows registry
func (b *Sdk) clearGlobalEnv(version Version) {
	if version == "" {
		return
	}
	sdkPackage, err := b.GetLocalSdkPackage(version)
	if err != nil {
		return
	}
	envKV, err := b.Plugin.EnvKeys(sdkPackage)
	if err != nil {
		return
	}
	envManager := b.sdkManager.EnvManager
	_ = envManager.Remove(envKV)
}

// clearCurrentEnvConfig will clear all env config about the current sdk for all places.
// such as: global, session, project and Windows registry
func (b *Sdk) clearCurrentEnvConfig() {
	b.clearEnvConfig(b.Current())
}

// clearEnvConfig will clear all env config about the current sdk for all places.
// such as: global, session and Windows registry
func (b *Sdk) clearEnvConfig(version Version) {
	if version == "" {
		return
	}
	sdkPackage, err := b.GetLocalSdkPackage(version)
	if err != nil {
		return
	}
	envKV, err := b.Plugin.EnvKeys(sdkPackage)
	if err != nil {
		return
	}
	envManager := b.sdkManager.EnvManager
	_ = envManager.Remove(envKV)
	_ = envManager.Flush()

	// clear tool versions
	toolVersion, err := toolset.NewMultiToolVersions([]string{
		b.sdkManager.PathMeta.CurTmpPath,
		b.sdkManager.PathMeta.HomePath,
	})
	for _, tv := range toolVersion {
		delete(tv.Record, b.Plugin.SdkName)
	}
}

func (b *Sdk) GetLocalSdkPackage(version Version) (*Package, error) {
	versionPath := b.VersionPath(version)
	mainSdk := &Info{
		Name:    b.Plugin.Name,
		Version: version,
	}
	var additions []*Info
	dir, err := os.ReadDir(versionPath)
	if err != nil {
		return nil, err
	}
	for _, d := range dir {
		if d.IsDir() {
			split := strings.SplitN(d.Name(), "-", 2)
			name := split[0]
			if name == b.Plugin.Name {
				mainSdk.Path = filepath.Join(versionPath, d.Name())
				continue
			}
			if len(split) != 2 {
				continue
			}
			v := split[1]
			additions = append(additions, &Info{
				Name:    name,
				Version: Version(v),
				Path:    filepath.Join(versionPath, d.Name()),
			})
		}
	}
	if err != nil {
		return nil, err
	}

	if mainSdk.Path == "" {
		return nil, errors.New("main sdk not found")

	}
	return &Package{
		Main:      mainSdk,
		Additions: additions,
	}, nil
}

func (b *Sdk) checkExists(version Version) bool {
	return util.FileExists(b.VersionPath(version))
}

func (b *Sdk) VersionPath(version Version) string {
	return filepath.Join(b.InstallPath, fmt.Sprintf("v-%s", version))
}

func (b *Sdk) Download(u *url.URL) (string, error) {
	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return "", err
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

	path := filepath.Join(b.InstallPath, filepath.Base(u.Path))

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

func (b *Sdk) label(version Version) string {
	return fmt.Sprintf("%s@%s", strings.ToLower(b.Plugin.Name), version)
}

// NewSdk creates a new SDK instance.
func NewSdk(manager *Manager, pluginPath string) (*Sdk, error) {
	luaPlugin, err := NewLuaPlugin(pluginPath, manager)
	if err != nil {
		return nil, fmt.Errorf("failed to create lua plugin: %w", err)
	}
	return &Sdk{
		sdkManager:  manager,
		InstallPath: filepath.Join(manager.PathMeta.SdkCachePath, strings.ToLower(luaPlugin.SdkName)),
		Plugin:      luaPlugin,
	}, nil
}
