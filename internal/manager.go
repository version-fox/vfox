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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/pterm/pterm"
	"github.com/shirou/gopsutil/v4/process"
	"github.com/urfave/cli/v3"
	"github.com/version-fox/vfox/internal/config"
	"github.com/version-fox/vfox/internal/env"
	"github.com/version-fox/vfox/internal/pathmeta"
	"github.com/version-fox/vfox/internal/plugin"
	"github.com/version-fox/vfox/internal/sdk"
	"github.com/version-fox/vfox/internal/shared/logger"
	"github.com/version-fox/vfox/internal/shared/util"
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
	RuntimeEnvContext *env.RuntimeEnvContext // runtime environment context
	openSdks          map[string]sdk.Sdk
}

// LoadToolVersionByScope loads tool versions based on the specified scope (global, project, or session).
func (m *Manager) LoadToolVersionByScope(scope env.UseScope) (*pathmeta.ToolVersion, error) {
	envContext := m.RuntimeEnvContext
	toolVersions, err := envContext.LoadToolVersionByScope(scope)
	if err != nil {
		return nil, err
	}

	// Need to parse legacy file for project scope
	if env.Project == scope {
		if err = m.parseLegacyFile(envContext.PathMeta.Working.Directory, func(sdkname, version string) {
			logger.Debugf("parse legacy file: %s@%s\n", sdkname, version)
			if _, ok := toolVersions.Record[sdkname]; !ok {
				toolVersions.Record[sdkname] = version
			}
		}); err != nil {
			return nil, err
		}
	}
	return toolVersions, nil
}

func (m *Manager) EnvKeys(tvs pathmeta.MultiToolVersions) (*env.Envs, error) {
	sdkEnvs := &env.Envs{
		Variables: make(env.Vars),
		Paths:     env.NewPaths(env.EmptyPaths),
	}
	tools := make(map[string]struct{})
	for _, t := range tvs {
		for name, version := range t.Record {
			if _, ok := tools[name]; ok {
				continue
			}
			if lookupSdk, err := m.LookupSdk(name); err == nil {
				v := sdk.Version(version)
				if ek, err := lookupSdk.EnvKeys(v); err == nil {
					tools[name] = struct{}{}
					sdkEnvs.Merge(ek)
				}
			}
		}
	}
	return sdkEnvs, nil
}

// LookupSdk lookup sdk by name
func (m *Manager) LookupSdk(name string) (sdk.Sdk, error) {
	if s, ok := m.openSdks[name]; ok {
		return s, nil
	}

	// Query plugin directly from shared root
	pluginPath := filepath.Join(m.RuntimeEnvContext.PathMeta.Shared.Plugins, strings.ToLower(name))
	if !util.FileExists(pluginPath) {
		return nil, NotFoundError{Msg: fmt.Sprintf("%s not installed", name)}
	}
	s, err := sdk.NewSdk(m.RuntimeEnvContext, pluginPath)
	if err != nil {
		return nil, err
	}
	m.openSdks[strings.ToLower(name)] = s
	return s, nil
}

func (m *Manager) ResolveVersion(sdkName string, version sdk.Version) sdk.Version {
	if version == "" {
		// when version is empty, try to get version from workspace tool
		workspaceTool, err := m.RuntimeEnvContext.LoadToolVersionByScope(env.Project)
		if err != nil {
			logger.Errorf("Failed to get workspace tool version: %v", err)
			return version
		}

		logger.Debugf("workspace tool version: %+v\n", workspaceTool)
		if v, ok := workspaceTool.Record[sdkName]; ok {
			return sdk.Version(v)
		}
	}
	return version
}

func (m *Manager) LookupSdkWithInstall(name string, autoConfirm bool) (sdk.Sdk, error) {
	source, err := m.LookupSdk(name)
	if err != nil {
		if errors.As(err, &NotFoundError{}) {
			if autoConfirm {
				fmt.Printf("[%s] not added yet, automatically proceeding with installation.\n", pterm.LightBlue(name))
			} else if util.IsNonInteractiveTerminal() {
				return nil, cli.Exit(fmt.Sprintf("Plugin %s is not installed. Use the -y flag to automatically install plugins in non-interactive environments", name), 1)
			} else {
				fmt.Printf("[%s] not added yet, confirm that you want to use [%s]? \n", pterm.LightBlue(name), pterm.LightRed(name))
				result, _ := pterm.DefaultInteractiveConfirm.
					WithTextStyle(&pterm.ThemeDefault.DefaultText).
					WithConfirmStyle(&pterm.ThemeDefault.DefaultText).
					WithRejectStyle(&pterm.ThemeDefault.DefaultText).
					WithDefaultText("Please confirm").
					Show()
				if !result {
					return nil, cli.Exit(fmt.Sprintf("Plugin %s is not installed. Installation cancelled by user", name), 1)
				}
			}
			// TODO: need to optimize
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
		}
		return nil, fmt.Errorf("%s not supported, error: %w", name, err)
	} else {
		return source, nil
	}
}

func (m *Manager) LoadAllSdk() ([]sdk.Sdk, error) {
	dir := m.RuntimeEnvContext.PathMeta.Shared.Plugins

	if !util.FileExists(dir) {
		return []sdk.Sdk{}, nil // Return empty if shared root does not exist
	}

	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read plugins directory error: %w", err)
	}

	sdkSlice := make([]sdk.Sdk, 0)

	for _, f := range files {
		sdkName := f.Name()
		path := filepath.Join(dir, sdkName)

		if f.IsDir() {
			s, err := sdk.NewSdk(m.RuntimeEnvContext, path)
			if err == nil {
				sdkSlice = append(sdkSlice, s)
				m.openSdks[strings.ToLower(sdkName)] = s
			}
		} else if strings.HasSuffix(sdkName, ".lua") {
			// Compatible with old format
			logger.Warnf("Found old plugin format: %s", path)
		}
	}

	sort.Slice(sdkSlice, func(i, j int) bool {
		return sdkSlice[j].Metadata().Name > sdkSlice[i].Metadata().Name
	})

	return sdkSlice, nil
}

func (m *Manager) Close() {
	for _, handler := range m.openSdks {
		handler.Close()
	}
}

func (m *Manager) Remove(pluginName string) error {
	// TODO: check write permission
	source, err := m.LookupSdk(pluginName)
	if err != nil {
		return err
	}

	if err = source.Unuse(env.Global); err != nil {
		return err
	}
	sdkMetadata := source.Metadata()
	pterm.Printf("Removing %s plugin...\n", sdkMetadata.PluginInstalledPath)
	err = os.RemoveAll(sdkMetadata.PluginInstalledPath)
	if err != nil {
		return fmt.Errorf("remove failed, err: %w", err)
	}
	pterm.Printf("Removing %s sdk...\n", sdkMetadata.SdkInstalledPath)
	if err = os.RemoveAll(sdkMetadata.SdkInstalledPath); err != nil {
		return err
	}
	// clear legacy filenames
	if len(sdkMetadata.PluginMetadata.LegacyFilenames) > 0 {
		lfr, err := m.loadLegacyFileRecord()
		if err != nil {
			return err
		}
		delete(lfr.Record, sdkMetadata.Name)
		if err = lfr.Save(); err != nil {
			return fmt.Errorf("remove legacy filenames failed: %w", err)
		}
	}
	pterm.Printf("Remove %s plugin successfully! \n", pterm.LightGreen(pluginName))
	return nil
}

func (m *Manager) Update(pluginName string) error {
	source, err := m.LookupSdk(pluginName)
	if err != nil {
		return fmt.Errorf("%s plugin not installed", pluginName)
	}
	sdkMetadata := source.Metadata()
	pterm.Printf("Checking plugin manifest...\n")
	// Update search priority: updateUrl > registry > manifestUrl
	pluginMetadata := sdkMetadata.PluginMetadata
	downloadUrl := pluginMetadata.UpdateUrl
	if pluginMetadata.UpdateUrl == "" {
		address := m.GetRegistryAddress(pluginMetadata.Name + ".json")
		logger.Debugf("Fetching plugin %s from %s...\n", pluginName, address)
		registryManifest, err := m.fetchPluginManifest(address)
		if err != nil {
			if errors.Is(err, ManifestNotFound) {
				if pluginMetadata.ManifestUrl != "" {
					logger.Debugf("Fetching plugin %s from %s...\n", pluginName, pluginMetadata.ManifestUrl)
					du, err := m.fetchPluginManifest(pluginMetadata.ManifestUrl)
					if err != nil {
						return err
					}
					if util.CompareVersion(du.Version, pluginMetadata.Version) <= 0 {
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
		if util.CompareVersion(registryManifest.Version, pluginMetadata.Version) <= 0 {
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
		_ = os.RemoveAll(tempPlugin.InstalledPath)
		tempPlugin.Close()
	}()
	if util.CompareVersion(tempPlugin.Version, pluginMetadata.Version) <= 0 {
		pterm.Printf("%s is already the latest version\n", pterm.Blue(pluginName))
		return nil
	}
	success := false
	backupPath := sdkMetadata.PluginInstalledPath + "-bak"
	logger.Debugf("Backup %s plugin to %s \n", sdkMetadata.PluginInstalledPath, backupPath)
	if err = os.Rename(sdkMetadata.PluginInstalledPath, backupPath); err != nil {
		return fmt.Errorf("backup %s plugin failed, err: %w", sdkMetadata.PluginInstalledPath, err)
	}
	defer func() {
		if success {
			_ = os.RemoveAll(backupPath)
		} else {
			_ = os.Rename(backupPath, sdkMetadata.PluginInstalledPath)
		}
	}()
	if err = os.Rename(tempPlugin.InstalledPath, sdkMetadata.PluginInstalledPath); err != nil {
		return fmt.Errorf("update %s plugin failed, err: %w", pluginName, err)
	}

	// update legacy filenames
	if len(tempPlugin.LegacyFilenames) != len(pluginMetadata.LegacyFilenames) {
		logger.Debugf("Update legacy filenames for %s plugin, from: %+v to: %+v \n", pluginName, pluginMetadata.LegacyFilenames, tempPlugin.LegacyFilenames)
		lfr, err := m.loadLegacyFileRecord()
		if err != nil {
			return err
		}
		delete(lfr.Record, sdkMetadata.Name)
		lfr.Record[sdkMetadata.Name] = "true"
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
	resp, err := m.RuntimeEnvContext.HttpClient().Get(url)
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
	resp, err := m.RuntimeEnvContext.HttpClient().Do(req)
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

	path := filepath.Join(m.RuntimeEnvContext.PathMeta.Shared.Plugins, filepath.Base(downloadUrl))
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
		installPath = filepath.Join(m.RuntimeEnvContext.PathMeta.Shared.Plugins, pname)
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
		_ = os.RemoveAll(tempPlugin.InstalledPath)
		tempPlugin.Close()
	}()
	// check plugin exist again as the plugin may be from custom source without plugin name and alias.
	if pname == "" {
		pname = tempPlugin.Name
		installPath = filepath.Join(m.RuntimeEnvContext.PathMeta.Shared.Plugins, pname)
		logger.Debugf("No plugin name provided, use %s as plugin name, installPath: %s\n", pname, installPath)
		if util.FileExists(installPath) {
			return fmt.Errorf("plugin named %s already exists", pname)
		}
	}
	if err = os.Rename(tempPlugin.InstalledPath, installPath); err != nil {
		return fmt.Errorf("install plugin error: %w", err)
	}

	// set legacy filenames
	if len(tempPlugin.LegacyFilenames) > 0 {
		logger.Debugf("Add legacy filenames for %s plugin, %+v \n", pname, tempPlugin.LegacyFilenames)
		lfr, err := m.loadLegacyFileRecord()
		if err != nil {
			return err
		}
		lfr.Record[pname] = "true"
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
	return nil
}

// installPluginToTemp install plugin from path that can be a local or remote file to temp dir.
// NOTE:
//
//	1.only support .lua or .zip file type plugin.
//	2.install plugin to temp dir first, then validate the plugin, if success, return *LuaPlugin
func (m *Manager) installPluginToTemp(path string) (*plugin.Wrapper, error) {
	ext := filepath.Ext(path)
	if ext != ".lua" && ext != ".zip" {
		return nil, fmt.Errorf("unsupported %s type wrapper to install, only support .lua or .zip", ext)
	}
	localPath := path
	// remote file, download it first to local file.
	if strings.HasPrefix(path, "https://") || strings.HasPrefix(path, "http://") {
		logger.Debugf("Plugin from: %s \n", path)
		pluginPath, err := m.downloadPlugin(path)
		if err != nil {
			return nil, fmt.Errorf("download wrapper error: %w", err)
		}
		localPath = pluginPath
		defer func() {
			_ = os.Remove(localPath)
		}()
	}
	success := false
	tempInstallPath, err := os.MkdirTemp(m.RuntimeEnvContext.PathMeta.User.Temp, "vfox-")
	if err != nil {
		return nil, fmt.Errorf("install wrapper error: %w", err)
	}
	defer func() {
		if !success {
			_ = os.RemoveAll(tempInstallPath)
		}
	}()
	// make a directory to store the wrapper and rename the wrapper file to main.lua
	if ext == ".lua" {
		logger.Debugf("Moving wrapper %s to %s \n", localPath, tempInstallPath)
		if err = os.Rename(localPath, filepath.Join(tempInstallPath, "main.lua")); err != nil {
			return nil, fmt.Errorf("install wrapper error: %w", err)
		}
	} else {
		logger.Debugf("Unpacking wrapper %s to %s \n", localPath, tempInstallPath)
		if err = util.NewDecompressor(localPath).Decompress(tempInstallPath); err != nil {
			return nil, fmt.Errorf("install wrapper error: %w", err)
		}
	}

	// validate the wrapper
	fmt.Printf("Validating %s ...\n", localPath)

	wrapper, err := plugin.CreatePlugin(tempInstallPath, m.RuntimeEnvContext)
	if err != nil {
		return nil, fmt.Errorf("validate wrapper failed: %w", err)
	}
	// Check if the wrapper is compatible with the current runtime
	if wrapper.MinRuntimeVersion != "" && util.CompareVersion(wrapper.MinRuntimeVersion, RuntimeVersion) > 0 {
		return nil, fmt.Errorf("check failed: this wrapper is not compatible with current vfox (>= %s), please upgrade vfox version to latest", pterm.LightRed(wrapper.MinRuntimeVersion))
	}

	success = true

	return wrapper, nil
}

func (m *Manager) Available() (RegistryIndex, error) {
	client := m.RuntimeEnvContext.HttpClient()
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
	cleanFlagPath := filepath.Join(m.RuntimeEnvContext.PathMeta.User.Temp, cleanupFlagFilename)
	if str, err := os.ReadFile(cleanFlagPath); err == nil {
		if i, err := strconv.ParseInt(string(str), 10, 64); err == nil && !util.IsBeforeToday(i) {
			return
		}
	}
	_ = os.WriteFile(cleanFlagPath, []byte(strconv.FormatInt(util.GetBeginOfToday(), 10)), os.ModePerm)

	procExists := make(map[string]struct{})

	if procList, err := process.Pids(); err == nil {
		for _, v := range procList {
			procExists[fmt.Sprintf("%d", v)] = struct{}{}
		}
	} else {
		return
	}

	dir, err := os.ReadDir(m.RuntimeEnvContext.PathMeta.User.Temp)
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
				_ = os.RemoveAll(filepath.Join(m.RuntimeEnvContext.PathMeta.User.Temp, file.Name()))
			}
		}
	}
}

func (m *Manager) GetRegistryAddress(uri string) string {
	userConfig := m.RuntimeEnvContext.UserConfig
	if userConfig.Registry.Address != "" {
		return userConfig.Registry.Address + "/" + uri
	}
	return pluginRegistryAddress + "/" + uri
}

// loadLegacyFileRecord load legacy file record which store the sdk-name
func (m *Manager) loadLegacyFileRecord() (*pathmeta.FileRecord, error) {
	file := filepath.Join(m.RuntimeEnvContext.PathMeta.User.Home, ".legacy_filenames")
	logger.Debugf("Loading legacy file record %s \n", file)
	mapFile, err := pathmeta.NewFileRecord(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read .legacy_filenames file %s: %w", file, err)
	}
	return mapFile, nil
}

// parseLegacyFile parse legacy file and output the sdkname and version
func (m *Manager) parseLegacyFile(dirPath string, output func(sdkname, version string)) error {
	// If the legacy version file is enabled, the legacy file will be parsed.
	if !m.RuntimeEnvContext.UserConfig.LegacyVersionFile.Enable {
		logger.Debugf("Legacy version file is disabled \n")
		return nil
	}
	legacyFileRecord, err := m.loadLegacyFileRecord()
	if err != nil {
		return err
	}
	logger.Debugf("Legacy file record: %+v \n", legacyFileRecord)

	// There are some legacy files to be parsed.
	if len(legacyFileRecord.Record) > 0 {
		for sdkName, _ := range legacyFileRecord.Record {
			if s, err := m.LookupSdk(sdkName); err == nil {
				//The .tool-version in the current directory has the highest priority,
				//checking to see if the version information in the legacy file exists in the former,
				// and updating to the former record (Don’t fall into the file!) if it doesn't.
				if version, err := s.ParseLegacyFile(dirPath); err == nil && version != "" {
					logger.Debugf("Found %s@%s \n", sdkName, version)
					output(sdkName, string(version))
				}
			}
		}

	}
	return nil
}

// NewSdkManager create a new SdkManager
func NewSdkManager() (*Manager, error) {
	vfoxHomeDir := env.GetVfoxHome()
	if len(vfoxHomeDir) == 0 {
		userHomeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("get user home dir error: %w", err)
		}
		vfoxHomeDir = pathmeta.GetVfoxUserHomeDir(userHomeDir)
	}

	// Get shared root (by priority)
	sharedRoot := env.GetVfoxRoot()
	if sharedRoot == "" {
		// use UserHome as sharedRoot if not set VFOX_ROOT
		sharedRoot = vfoxHomeDir
	}
	currentDir := getWorkingDirectory()
	meta, err := pathmeta.NewPathMeta(vfoxHomeDir, sharedRoot, currentDir, env.GetPid())
	if err != nil {
		return nil, fmt.Errorf("init path meta failed: %w", err)
	}

	c, err := config.NewConfig(meta.User.Home) // Config from user directory
	if err != nil {
		return nil, fmt.Errorf("load config failed: %w", err)
	}

	return &Manager{
		RuntimeEnvContext: &env.RuntimeEnvContext{
			UserConfig:        c,
			CurrentWorkingDir: currentDir,
			PathMeta:          meta,
			RuntimeVersion:    RuntimeVersion,
		},
		openSdks: make(map[string]sdk.Sdk),
	}, nil
}

func getWorkingDirectory() string {
	wd, err := os.Getwd()
	if err != nil {
		logger.Errorf("get current working directory failed: %v", err)
		return ""
	}
	return wd
}
