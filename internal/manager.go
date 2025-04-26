/*
 *    Copyright 2025 Han Li and contributors
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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/mitchellh/go-ps"
	"github.com/pterm/pterm"
	"github.com/urfave/cli/v3"
	"github.com/version-fox/vfox/internal/cache"
	"github.com/version-fox/vfox/internal/config"
	"github.com/version-fox/vfox/internal/env"
	"github.com/version-fox/vfox/internal/logger"
	"github.com/version-fox/vfox/internal/toolset"
	"github.com/version-fox/vfox/internal/util"
)

const (
	cleanupFlagFilename = ".cleanup"
)

var (
	ManifestNotFound = errors.New("manifest not found")
)

type NotFoundError struct {
	Msg string
}

func (n NotFoundError) Error() string {
	return n.Msg
}

type Arg struct {
	Name    string
	Version string
}

type Manager struct {
	PathMeta   *PathMeta
	openSdks   map[string]*Sdk
	EnvManager env.Manager
	Config     *config.Config
}

func (m *Manager) GlobalEnvKeys() (SdkEnvs, error) {
	workToolVersion, err := toolset.NewToolVersion(m.PathMeta.WorkingDirectory)
	if err != nil {
		return nil, err
	}

	if err = m.ParseLegacyFile(func(sdkname, version string) {
		if _, ok := workToolVersion.Record[sdkname]; !ok {
			workToolVersion.Record[sdkname] = version
		}
	}); err != nil {
		return nil, err
	}
	homeToolVersion, err := toolset.NewToolVersion(m.PathMeta.HomePath)
	if err != nil {
		return nil, err
	}
	return m.EnvKeys(toolset.MultiToolVersions{
		workToolVersion,
		homeToolVersion,
	}, ShellLocation)
}

type SessionEnvOptions struct {
	WithGlobalEnv bool
}

// SessionEnvKeys returns the environment variables that need to be set and/or unset by the shell. This is determined by
// the contents of any .tool-versions files in the following locations, in the following order of precedence:
//
//  1. Current working directory (in this directory, any legacy version files are also considered)
//  2. vfox home directory (only if the WithGlobalEnv option is set to true)
//  3. Current session's vfox temp directory
//
// The function maintains environment state through two mechanisms:
//   - Updates the .tool-versions file in the current session's temp directory to track active SDK versions
//   - Uses a "flush_env.cache" file in the current session's temp directory to prevent redundant environment variable updates
//
// Parameters:
//   - opt: SessionEnvOptions controlling whether global environment variables should be included
//
// Returns:
//   - SdkEnvs: A slice of SDK environment configurations that need to be applied
//   - error: Any error encountered during processing
//
// The returned environment configurations remain valid for the entire shell session until one of the following
// conditions is met:
//   - A new .tool-versions file is encountered
//   - The environment is explicitly modified via the `use` command
func (m *Manager) SessionEnvKeys(opt SessionEnvOptions) (SdkEnvs, error) {
	tvs := toolset.MultiToolVersions{}

	workToolVersion, err := toolset.NewToolVersion(m.PathMeta.WorkingDirectory)
	if err != nil {
		return nil, err
	}

	if err = m.ParseLegacyFile(func(sdkname, version string) {
		if _, ok := workToolVersion.Record[sdkname]; !ok {
			workToolVersion.Record[sdkname] = version
		}
	}); err != nil {
		return nil, err
	}

	tvs = append(tvs, workToolVersion)

	if opt.WithGlobalEnv {
		homeToolVersion, err := toolset.NewToolVersion(m.PathMeta.HomePath)
		if err != nil {
			return nil, err
		}

		tvs = append(tvs, homeToolVersion)
	}

	curToolVersion, err := toolset.NewToolVersion(m.PathMeta.CurTmpPath)
	if err != nil {
		return nil, err
	}
	defer curToolVersion.Save()

	tvs = append(tvs, curToolVersion)

	flushCache, err := cache.NewFileCache(filepath.Join(m.PathMeta.CurTmpPath, "flush_env.cache"))
	if err != nil {
		return nil, err
	}
	defer flushCache.Close()

	var sdkEnvs []*SdkEnv

	tvs.FilterTools(func(name, version string) bool {
		if lookupSdk, err := m.LookupSdk(name); err == nil {
			vv, ok := flushCache.Get(name)
			if ok && string(vv) == version {
				logger.Debugf("Hit cache, skip flush environment, %s@%s\n", name, version)
				return true
			} else {
				logger.Debugf("No hit cache, name: %s cache: %s, expected: %s \n", name, string(vv), version)
			}
			v := Version(version)
			if keys, err := lookupSdk.EnvKeys(v, ShellLocation); err == nil {
				flushCache.Set(name, cache.Value(version), cache.NeverExpired)

				sdkEnvs = append(sdkEnvs, &SdkEnv{
					Sdk: lookupSdk, Env: keys,
				})

				curToolVersion.Record[name] = version
				return true
			}
		}
		return false
	})

	return sdkEnvs, nil
}

func (m *Manager) EnvKeys(tvs toolset.MultiToolVersions, location Location) (SdkEnvs, error) {
	var sdkEnvs SdkEnvs
	tools := make(map[string]struct{})
	for _, t := range tvs {
		for name, version := range t.Record {
			if _, ok := tools[name]; ok {
				continue
			}
			if lookupSdk, err := m.LookupSdk(name); err == nil {
				v := Version(version)
				if ek, err := lookupSdk.EnvKeys(v, location); err == nil {
					tools[name] = struct{}{}

					sdkEnvs = append(sdkEnvs, &SdkEnv{
						Sdk: lookupSdk,
						Env: ek,
					})
				}
			}
		}
	}
	return sdkEnvs, nil
}

// LookupSdk lookup sdk by name
func (m *Manager) LookupSdk(name string) (*Sdk, error) {
	pluginPath := filepath.Join(m.PathMeta.PluginPath, strings.ToLower(name))
	if !util.FileExists(pluginPath) {
		oldPath := filepath.Join(m.PathMeta.PluginPath, strings.ToLower(name)+".lua")
		if !util.FileExists(oldPath) {
			return nil, NotFoundError{Msg: fmt.Sprintf("%s not installed", name)}
		}
		logger.Debugf("Found old plugin path %s \n", oldPath)
		// FIXME !!! This snippet will be removed in a later version
		// rename old plugin path to new plugin path
		err := os.Mkdir(filepath.Join(m.PathMeta.PluginPath, strings.ToLower(name)), 0777)
		if err != nil {
			return nil, fmt.Errorf("failed to migrate an old plug-in: %w", err)
		}
		if err = os.Rename(oldPath, filepath.Join(pluginPath, "main.lua")); err != nil {
			return nil, fmt.Errorf("failed to migrate an old plug-in: %w", err)
		}
	}
	sdk, err := NewSdk(m, pluginPath)
	if err != nil {
		return nil, err
	}
	m.openSdks[strings.ToLower(name)] = sdk
	return sdk, nil
}

func (m *Manager) LookupSdkWithInstall(name string) (*Sdk, error) {
	source, err := m.LookupSdk(name)
	if err != nil {
		if errors.As(err, &NotFoundError{}) {
			fmt.Printf("[%s] not added yet, confirm that you want to use [%s]? \n", pterm.LightBlue(name), pterm.LightRed(name))
			if result, _ := pterm.DefaultInteractiveConfirm.
				WithTextStyle(&pterm.ThemeDefault.DefaultText).
				WithConfirmStyle(&pterm.ThemeDefault.DefaultText).
				WithRejectStyle(&pterm.ThemeDefault.DefaultText).
				WithDefaultText("Please confirm").
				Show(); result {

				manifest, err := m.fetchPluginManifest(m.GetRegistryAddress(name + ".json"))
				if err != nil {
					if errors.Is(err, ManifestNotFound) {
						return nil, fmt.Errorf("[%s] not found in remote registry, please check the name", pterm.LightRed(name))
					}
					return nil, err
				}

				if err = m.Add(manifest.Name, manifest.DownloadUrl, ""); err != nil {
					return nil, err
				}
				return m.LookupSdk(manifest.Name)
			} else {
				return nil, cli.Exit("", 1)
			}
		}
		return nil, fmt.Errorf("%s not supported, error: %w", name, err)
	} else {
		return source, nil
	}
}

func (m *Manager) LoadAllSdk() ([]*Sdk, error) {
	dir, err := os.ReadDir(m.PathMeta.PluginPath)
	if err != nil {
		return nil, fmt.Errorf("load sdks error: %w", err)
	}
	sdkSlice := make([]*Sdk, 0)
	for _, d := range dir {
		sdkName := d.Name()
		path := filepath.Join(m.PathMeta.PluginPath, sdkName)

		// Resolve symbolic link, useful for plugin development
		dirInfo, err := d.Info()
		if err != nil {
			return nil, fmt.Errorf("failed to get directory info: %w", err)
		}
		if dirInfo.Mode()&os.ModeSymlink != 0 {
			resolvedPath, err := filepath.EvalSymlinks(path)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve symboling link : %w", err)
			}
			dirFileInfo, err := os.Lstat(resolvedPath)
			if err != nil {
				return nil, fmt.Errorf("failed to get stat for resolved symbolic link : %w", err)
			}
			d = fs.FileInfoToDirEntry(dirFileInfo)
		}

		if d.IsDir() {
		} else if strings.HasSuffix(sdkName, ".lua") {
			logger.Debugf("Found old plugin: %s \n", path)
			// FIXME !!! This snippet will be removed in a later version
			// rename old plugin path to new plugin path
			newPluginDir := filepath.Join(m.PathMeta.PluginPath, strings.TrimSuffix(sdkName, ".lua"))
			err = os.Mkdir(newPluginDir, 0777)
			if err != nil {
				return nil, fmt.Errorf("failed to migrate an old plug-in: %w", err)
			}
			if err = os.Rename(filepath.Join(m.PathMeta.PluginPath, sdkName), filepath.Join(newPluginDir, "main.lua")); err != nil {
				return nil, fmt.Errorf("failed to migrate an old plug-in: %w", err)
			}
			path = newPluginDir
			sdkName = strings.TrimSuffix(sdkName, ".lua")
		} else {
			continue
		}
		sdk, _ := NewSdk(m, path)
		sdkSlice = append(sdkSlice, sdk)

		m.openSdks[strings.ToLower(sdkName)] = sdk
	}

	sort.Slice(sdkSlice, func(i, j int) bool {
		return sdkSlice[j].Plugin.SdkName > sdkSlice[i].Plugin.SdkName
	})
	return sdkSlice, nil
}

func (m *Manager) Close() {
	for _, handler := range m.openSdks {
		handler.Close()
	}
	_ = m.EnvManager.Close()
}

func (m *Manager) Remove(pluginName string) error {
	source, err := m.LookupSdk(pluginName)
	if err != nil {
		return err
	}

	if err = source.ClearCurrentEnv(); err != nil {
		return err
	}
	pPath := filepath.Join(m.PathMeta.PluginPath, pluginName)
	pterm.Printf("Removing %s plugin...\n", pPath)
	err = os.RemoveAll(pPath)
	if err != nil {
		return fmt.Errorf("remove failed, err: %w", err)
	}
	pterm.Printf("Removing %s sdk...\n", source.InstallPath)
	if err = os.RemoveAll(source.InstallPath); err != nil {
		return err
	}
	// clear legacy filenames
	if len(source.Plugin.LegacyFilenames) > 0 {
		lfr, err := m.loadLegacyFileRecord()
		if err != nil {
			return err
		}
		for _, filename := range source.Plugin.LegacyFilenames {
			delete(lfr.Record, filename)
		}
		if err = lfr.Save(); err != nil {
			return fmt.Errorf("remove legacy filenames failed: %w", err)
		}
	}
	pterm.Printf("Remove %s plugin successfully! \n", pterm.LightGreen(pluginName))
	return nil
}

func (m *Manager) Update(pluginName string) error {
	sdk, err := m.LookupSdk(pluginName)
	if err != nil {
		return fmt.Errorf("%s plugin not installed", pluginName)
	}
	pterm.Printf("Checking plugin manifest...\n")
	// Update search priority: updateUrl > registry > manifestUrl
	downloadUrl := sdk.Plugin.UpdateUrl
	if sdk.Plugin.UpdateUrl == "" {
		address := m.GetRegistryAddress(sdk.Plugin.Name + ".json")
		logger.Debugf("Fetching plugin %s from %s...\n", pluginName, address)
		registryManifest, err := m.fetchPluginManifest(address)
		if err != nil {
			if errors.Is(err, ManifestNotFound) {
				if sdk.Plugin.ManifestUrl != "" {
					logger.Debugf("Fetching plugin %s from %s...\n", pluginName, sdk.Plugin.ManifestUrl)
					du, err := m.fetchPluginManifest(sdk.Plugin.ManifestUrl)
					if err != nil {
						return err
					}
					if util.CompareVersion(du.Version, sdk.Plugin.Version) <= 0 {
						pterm.Printf("%s is already the latest version\n", pterm.Blue(pluginName))
						return nil
					}
					downloadUrl = du.DownloadUrl
				} else {
					return fmt.Errorf("%s plugin not support update", pluginName)
				}
			}
			return err
		}
		if util.CompareVersion(registryManifest.Version, sdk.Plugin.Version) <= 0 {
			pterm.Printf("%s is already the latest version\n", pterm.LightBlue(pluginName))
			return nil
		}
		downloadUrl = registryManifest.DownloadUrl

	}
	tempPlugin, err := m.installPluginToTemp(downloadUrl)
	if err != nil {
		return err
	}
	defer func() {
		_ = os.RemoveAll(tempPlugin.Path)
		tempPlugin.Close()
	}()
	if util.CompareVersion(tempPlugin.Version, sdk.Plugin.Version) <= 0 {
		pterm.Printf("%s is already the latest version\n", pterm.Blue(pluginName))
		return nil
	}
	success := false
	backupPath := sdk.Plugin.Path + "-bak"
	logger.Debugf("Backup %s plugin to %s \n", sdk.Plugin.Path, backupPath)
	if err = os.Rename(sdk.Plugin.Path, backupPath); err != nil {
		return fmt.Errorf("backup %s plugin failed, err: %w", sdk.Plugin.Path, err)
	}
	defer func() {
		if success {
			_ = os.RemoveAll(backupPath)
		} else {
			_ = os.Rename(backupPath, sdk.Plugin.Path)
		}
	}()
	if err = os.Rename(tempPlugin.Path, sdk.Plugin.Path); err != nil {
		return fmt.Errorf("update %s plugin failed, err: %w", pluginName, err)
	}

	// update legacy filenames
	if len(tempPlugin.LegacyFilenames) != len(sdk.Plugin.LegacyFilenames) {
		logger.Debugf("Update legacy filenames for %s plugin, from: %+v to: %+v \n", pluginName, sdk.Plugin.LegacyFilenames, tempPlugin.LegacyFilenames)
		lfr, err := m.loadLegacyFileRecord()
		if err != nil {
			return err
		}
		for _, filename := range sdk.Plugin.LegacyFilenames {
			delete(lfr.Record, filename)
		}
		for _, filename := range tempPlugin.LegacyFilenames {
			lfr.Record[filename] = pluginName
		}
		if err = lfr.Save(); err != nil {
			return fmt.Errorf("update legacy filenames failed: %w", err)
		}
	}
	success = true

	tempPlugin.ShowNotes()

	pterm.Printf("Update %s plugin successfully! version: %s \n", pterm.Green(pluginName), pterm.Blue(tempPlugin.Version))

	// It's probably an old format plugin, just a reminder.
	if tempPlugin.UpdateUrl != "" && tempPlugin.ManifestUrl != "" {
		pterm.Printf("%s\n", pterm.LightYellow("This plugin maybe an old format plugin, please update this plugin again!"))
	}

	return nil
}

// fetchPluginManifest fetch plugin from registry by manifest url
func (m *Manager) fetchPluginManifest(url string) (*RegistryPluginManifest, error) {
	resp, err := m.HttpClient().Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetch manifest error: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, ManifestNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch manifest error, status code: %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("fetch manifest error: %w", err)
	}
	var plugin RegistryPluginManifest
	if err = json.Unmarshal(body, &plugin); err != nil {
		return nil, fmt.Errorf("parse plugin error: %w", err)
	}
	logger.Debugf("Manifest found, name: %s, version: %s,  downloadUrl: %s \n", plugin.Name, plugin.Version, plugin.DownloadUrl)

	// Check if the plugin is compatible with the current runtime
	if plugin.MinRuntimeVersion != "" && util.CompareVersion(plugin.MinRuntimeVersion, RuntimeVersion) > 0 {
		return nil, fmt.Errorf("check failed: this plugin is not compatible with current vfox (>= %s), please upgrade vfox version to latest", pterm.LightRed(plugin.MinRuntimeVersion))
	}
	return &plugin, nil
}

// downloadPlugin download plugin from downloadUrl to plugin home directory.
func (m *Manager) downloadPlugin(downloadUrl string) (string, error) {
	req, err := http.NewRequest("GET", downloadUrl, nil)
	if err != nil {
		return "", err
	}
	resp, err := m.HttpClient().Do(req)
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
		return "", fmt.Errorf("plugin not found at %s", downloadUrl)
	}

	path := filepath.Join(m.PathMeta.PluginPath, filepath.Base(downloadUrl))
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return "", err
	}
	defer f.Close()
	fmt.Printf("Downloading %s... \n", filepath.Base(downloadUrl))
	_, err = io.Copy(f, resp.Body)
	if err != nil {
		return "", err
	}
	return path, nil
}

// Add a plugin to plugin home directory
// 1. If the plugin is an official plugin, fetch the plugin manifest from the registry.
// 2. If the plugin is a custom plugin, install the plugin from the specified URL.
// 3. Validate the plugin and install it to the plugin home directory.
// examples:
//
//	vfox add nodejs
//	vfox add --alias node nodejs
//	vfox add --source /path/to/plugin.zip
//	vfox add --source /path/to/plugin.zip --alias node [nodejs]
func (m *Manager) Add(pluginName, url, alias string) error {
	// For compatibility with older versions of plugin names <category>/<plugin-name>
	if strings.Contains(pluginName, "/") {
		pluginName = strings.Split(pluginName, "/")[1]
	}
	pluginPath := url
	pname := pluginName
	if len(alias) > 0 {
		pname = alias
	}
	var installPath string
	// first quick check.
	if pname != "" {
		installPath = filepath.Join(m.PathMeta.PluginPath, pname)
		if util.FileExists(installPath) {
			return fmt.Errorf("plugin named %s already exists", pname)
		}
	}
	// official plugin
	if len(url) == 0 {
		fmt.Printf("Fetching %s manifest... \n", pterm.Green(pluginName))
		pluginManifest, err := m.fetchPluginManifest(m.GetRegistryAddress(pluginName + ".json"))
		if err != nil {
			return err
		}
		pluginPath = pluginManifest.DownloadUrl
	}
	tempPlugin, err := m.installPluginToTemp(pluginPath)
	if err != nil {
		return err
	}
	defer func() {
		_ = os.RemoveAll(tempPlugin.Path)
		tempPlugin.Close()
	}()
	// check plugin exist again as the plugin may be from custom source without plugin name and alias.
	if pname == "" {
		pname = tempPlugin.Name
		installPath = filepath.Join(m.PathMeta.PluginPath, pname)
		logger.Debugf("No plugin name provided, use %s as plugin name, installPath: %s\n", pname, installPath)
		if util.FileExists(installPath) {
			return fmt.Errorf("plugin named %s already exists", pname)
		}
	}
	if err = os.Rename(tempPlugin.Path, installPath); err != nil {
		return fmt.Errorf("install plugin error: %w", err)
	}

	// set legacy filenames
	if len(tempPlugin.LegacyFilenames) > 0 {
		logger.Debugf("Add legacy filenames for %s plugin, %+v \n", pname, tempPlugin.LegacyFilenames)
		lfr, err := m.loadLegacyFileRecord()
		if err != nil {
			return err
		}
		for _, filename := range tempPlugin.LegacyFilenames {
			lfr.Record[filename] = pname
		}
		if err = lfr.Save(); err != nil {
			return fmt.Errorf("add legacy filenames failed: %w", err)
		}
	}

	pterm.Println("Plugin info:")
	pterm.Println("Name    ", "->", pterm.LightBlue(tempPlugin.Name))
	pterm.Println("Version ", "->", pterm.LightBlue(tempPlugin.Version))
	pterm.Println("Homepage", "->", pterm.LightBlue(tempPlugin.Homepage))
	pterm.Println("Desc    ", "->", pterm.LightBlue(tempPlugin.Description))

	tempPlugin.ShowNotes()

	pterm.Printf("Add %s plugin successfully! \n", pterm.LightGreen(pname))
	pterm.Printf("Please use `%s` to install the version you need.\n", pterm.LightBlue(fmt.Sprintf("vfox install %s@<version>", pname)))
	return nil
}

// installPluginToTemp install plugin from path that can be a local or remote file to temp dir.
// NOTE:
//
//	1.only support .lua or .zip file type plugin.
//	2.install plugin to temp dir first, then validate the plugin, if success, return *LuaPlugin
func (m *Manager) installPluginToTemp(path string) (*LuaPlugin, error) {
	ext := filepath.Ext(path)
	if ext != ".lua" && ext != ".zip" {
		return nil, fmt.Errorf("unsupported %s type plugin to install, only support .lua or .zip", ext)
	}
	localPath := path
	// remote file, download it first to local file.
	if strings.HasPrefix(path, "https://") || strings.HasPrefix(path, "http://") {
		logger.Debugf("Plugin from: %s \n", path)
		pluginPath, err := m.downloadPlugin(path)
		if err != nil {
			return nil, fmt.Errorf("download plugin error: %w", err)
		}
		localPath = pluginPath
		defer func() {
			_ = os.Remove(localPath)
		}()
	}
	success := false
	tempInstallPath, err := os.MkdirTemp(m.PathMeta.TempPath, "vfox-")
	if err != nil {
		return nil, fmt.Errorf("install plugin error: %w", err)
	}
	defer func() {
		if !success {
			_ = os.RemoveAll(tempInstallPath)
		}
	}()
	// make a directory to store the plugin and rename the plugin file to main.lua
	if ext == ".lua" {
		logger.Debugf("Moving plugin %s to %s \n", localPath, tempInstallPath)
		if err = os.Rename(localPath, filepath.Join(tempInstallPath, "main.lua")); err != nil {
			return nil, fmt.Errorf("install plugin error: %w", err)
		}
	} else {
		logger.Debugf("Unpacking plugin %s to %s \n", localPath, tempInstallPath)
		if err = util.NewDecompressor(localPath).Decompress(tempInstallPath); err != nil {
			return nil, fmt.Errorf("install plugin error: %w", err)
		}
	}

	// validate the plugin
	fmt.Printf("Validating %s ...\n", localPath)

	plugin, err := NewLuaPlugin(tempInstallPath, m)
	if err != nil {
		return nil, fmt.Errorf("validate plugin failed: %w", err)
	}
	// Check if the plugin is compatible with the current runtime
	if plugin.MinRuntimeVersion != "" && util.CompareVersion(plugin.MinRuntimeVersion, RuntimeVersion) > 0 {
		return nil, fmt.Errorf("check failed: this plugin is not compatible with current vfox (>= %s), please upgrade vfox version to latest", pterm.LightRed(plugin.MinRuntimeVersion))
	}

	success = true

	return plugin, nil
}

func (m *Manager) HttpClient() *http.Client {
	var client *http.Client
	if m.Config.Proxy.Enable {
		if uri, err := url.Parse(m.Config.Proxy.Url); err == nil {
			transPort := &http.Transport{
				Proxy: http.ProxyURL(uri),
			}
			client = &http.Client{
				Transport: transPort,
			}
		}
	} else {
		client = http.DefaultClient
	}

	return client
}

func (m *Manager) Available() (RegistryIndex, error) {
	client := m.HttpClient()
	resp, err := client.Get(m.GetRegistryAddress("index.json"))
	if err != nil {
		return nil, fmt.Errorf("get plugin index error: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get plugin index error, status code: %d", resp.StatusCode)
	}
	if str, err := io.ReadAll(resp.Body); err != nil {
		return nil, fmt.Errorf("read plugin index error: %w", err)
	} else {
		var index RegistryIndex
		err = json.Unmarshal(str, &index)
		if err != nil {
			return nil, fmt.Errorf("parse plugin index error: %w", err)
		}
		return index, nil
	}
}

func (m *Manager) CleanTmp() {
	// once per day
	cleanFlagPath := filepath.Join(m.PathMeta.TempPath, cleanupFlagFilename)
	if str, err := os.ReadFile(cleanFlagPath); err == nil {
		if i, err := strconv.ParseInt(string(str), 10, 64); err == nil && !util.IsBeforeToday(i) {
			return
		}
	}
	_ = os.WriteFile(cleanFlagPath, []byte(strconv.FormatInt(util.GetBeginOfToday(), 10)), os.ModePerm)

	procExists := make(map[string]struct{})
	if procList, err := ps.Processes(); err == nil {
		for _, v := range procList {
			if v != nil {
				procExists[strconv.Itoa(v.Pid())] = struct{}{}
			}
		}
	}

	dir, err := os.ReadDir(m.PathMeta.TempPath)
	if err == nil {
		for _, file := range dir {
			if !file.IsDir() {
				continue
			}
			timestamp, pid, ok := strings.Cut(file.Name(), "-")
			if !ok {
				continue
			}
			if _, ok = procExists[pid]; ok {
				continue
			}
			i, err := strconv.ParseInt(timestamp, 10, 64)
			if err != nil {
				continue
			}
			if util.IsBeforeToday(i) {
				_ = os.RemoveAll(filepath.Join(m.PathMeta.TempPath, file.Name()))
			}
		}
	}
}

func (m *Manager) GetRegistryAddress(uri string) string {
	if m.Config.Registry.Address != "" {
		return m.Config.Registry.Address + "/" + uri
	}
	return pluginRegistryAddress + "/" + uri
}

// loadLegacyFileRecord load legacy file record which store the mapping of legacy filename and sdk-name
func (m *Manager) loadLegacyFileRecord() (*toolset.FileRecord, error) {
	file := filepath.Join(m.PathMeta.HomePath, ".legacy_filenames")
	logger.Debugf("Loading legacy file record %s \n", file)
	mapFile, err := toolset.NewFileRecord(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read .legacy_filenames file %s: %w", file, err)
	}
	return mapFile, nil
}

// ParseLegacyFile parse legacy file and output the sdkname and version
func (m *Manager) ParseLegacyFile(output func(sdkname, version string)) error {
	// If the legacy version file is enabled, the legacy file will be parsed.
	if !m.Config.LegacyVersionFile.Enable {
		return nil
	}
	legacyFileRecord, err := m.loadLegacyFileRecord()
	if err != nil {
		return err
	}

	// There are some legacy files to be parsed.
	if len(legacyFileRecord.Record) > 0 {
		for filename, sdkname := range legacyFileRecord.Record {
			path := filepath.Join(m.PathMeta.WorkingDirectory, filename)
			if util.FileExists(path) {
				logger.Debugf("Parsing legacy file %s \n", path)
				if sdk, err := m.LookupSdk(sdkname); err == nil {
					// The .tool-version in the current directory has the highest priority,
					// checking to see if the version information in the legacy file exists in the former,
					//  and updating to the former record (Don’t fall into the file!) if it doesn't.
					if version, err := sdk.ParseLegacyFile(path); err == nil && version != "" {
						logger.Debugf("Found %s@%s in %s \n", sdkname, version, path)
						output(sdkname, string(version))
					}
				}
			}
		}

	}
	return nil
}

func NewSdkManager() *Manager {
	meta, err := newPathMeta()
	if err != nil {
		panic("Init path meta error " + err.Error())
	}
	return newSdkManager(meta)
}

func newSdkManager(meta *PathMeta) *Manager {
	envManger, err := env.NewEnvManager(meta.HomePath)
	if err != nil {
		panic("Init env manager error")
	}
	c, err := config.NewConfig(meta.HomePath)
	if err != nil {
		panic(fmt.Errorf("init Config error: %w", err))
	}

	// custom sdk path first
	if len(c.Storage.SdkPath) > 0 {
		err = c.Storage.Validate()
		if err != nil {
			panic(fmt.Errorf("validate storage error: %w", err))
		}
		meta.SdkCachePath = c.Storage.SdkPath
	}
	manager := &Manager{
		PathMeta:   meta,
		EnvManager: envManger,
		openSdks:   make(map[string]*Sdk),
		Config:     c,
	}
	return manager
}
