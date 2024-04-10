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
	"encoding/json"
	"errors"
	"fmt"
	"github.com/urfave/cli/v2"
	"github.com/version-fox/vfox/internal/logger"
	"github.com/version-fox/vfox/internal/toolset"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pterm/pterm"
	"github.com/version-fox/vfox/internal/config"
	"github.com/version-fox/vfox/internal/env"
	"github.com/version-fox/vfox/internal/util"
)

const (
	cleanupFlagFilename = ".cleanup"
)

var (
	manifestNotFound = errors.New("manifest not found")
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

func (m *Manager) EnvKeys(tvs toolset.MultiToolVersions) (*env.Envs, error) {
	shellEnvs := &env.Envs{
		Variables: make(env.Vars),
		Paths:     env.NewPaths(env.EmptyPaths),
	}

	tools := make(map[string]string)
	for _, t := range tvs {
		for name, version := range t.Record {
			if _, ok := tools[name]; ok {
				continue
			}
			if lookupSdk, err := m.LookupSdk(name); err == nil {
				if ek, err := lookupSdk.EnvKeys(Version(version)); err == nil {
					for key, value := range ek.Variables {
						shellEnvs.Variables[key] = value
					}
					shellEnvs.Paths.Merge(ek.Paths)
				}
			}
		}
	}
	return shellEnvs, nil
}

// LookupSdk lookup sdk by name
func (m *Manager) LookupSdk(name string) (*Sdk, error) {
	pluginPath := filepath.Join(m.PathMeta.PluginPath, strings.ToLower(name))
	if !util.FileExists(pluginPath) {
		oldPath := filepath.Join(m.PathMeta.PluginPath, strings.ToLower(name)+".lua")
		if !util.FileExists(oldPath) {
			return nil, NotFoundError{Msg: fmt.Sprintf("%s not installed", name)}
		}
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
	luaPlugin, err := NewLuaPlugin(pluginPath, m)
	if err != nil {
		return nil, err
	}
	sdk, _ := NewSdk(m, luaPlugin)
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
				if errors.Is(err, manifestNotFound) {
					return nil, fmt.Errorf("[%s] not found in remote registry, please check the name", pterm.LightRed(name))
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

func (m *Manager) LoadAllSdk() (map[string]*Sdk, error) {
	dir, err := os.ReadDir(m.PathMeta.PluginPath)
	if err != nil {
		return nil, fmt.Errorf("load sdks error: %w", err)
	}
	sdkMap := make(map[string]*Sdk)
	for _, d := range dir {
		sdkName := d.Name()
		path := filepath.Join(m.PathMeta.PluginPath, sdkName)
		if d.IsDir() {
		} else if strings.HasSuffix(sdkName, ".lua") {
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
		source, err := NewLuaPlugin(path, m)
		if err != nil {
			pterm.Printf("Failed to load %s plugin, err: %s\n", filepath.Dir(path), err)
			continue
		}
		sdk, _ := NewSdk(m, source)
		sdkMap[strings.ToLower(sdkName)] = sdk
		m.openSdks[strings.ToLower(sdkName)] = sdk
	}
	return sdkMap, nil
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

	source.clearCurrentEnvConfig()
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
			if errors.Is(err, manifestNotFound) {
				if sdk.Plugin.ManifestUrl != "" {
					logger.Debugf("Fetching plugin %s from %s...\n", pluginName, sdk.Plugin.ManifestUrl)
					du, err := m.fetchPluginManifest(sdk.Plugin.ManifestUrl)
					if err != nil {
						return err
					}
					if util.CompareVersion(du.Version, sdk.Plugin.Version) <= 0 {
						pterm.Printf("%s is already the latest version\n", pterm.LightBlue(pluginName))
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
	pterm.Println("Comparing plugin version...")
	if util.CompareVersion(tempPlugin.Version, sdk.Plugin.Version) <= 0 {
		pterm.Printf("%s is already the latest version\n", pterm.LightBlue(pluginName))
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
	success = true
	// print some notes if there are
	if len(tempPlugin.Notes) != 0 {
		fmt.Println(pterm.LightYellow("Notes:"))
		for _, note := range tempPlugin.Notes {
			fmt.Println("  -", note)
		}
	}
	pterm.Printf("Update %s plugin successfully! version: %s \n", pterm.LightGreen(pluginName), pterm.LightBlue(tempPlugin.Version))

	return nil
}

// fetchPluginManifest fetch plugin from registry by manifest url
func (m *Manager) fetchPluginManifest(url string) (*RegistryPluginManifest, error) {
	fmt.Println("Fetching plugin manifest...")
	resp, err := m.HttpClient().Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetch manifest error: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, manifestNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch manifest error, status code: %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("fetch manfiest error: %w", err)
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
	fmt.Printf("Downloading %s... \n", downloadUrl)
	_, err = io.Copy(f, resp.Body)
	if err != nil {
		return "", err
	}
	return path, nil
}

func (m *Manager) Add(pluginName, url, alias string) error {
	pluginPath := url
	// official plugin
	if len(url) == 0 {
		pname := pluginName
		// For compatibility with older versions of plugin names <category>/<plugin-name>
		if strings.Contains(pluginName, "/") {
			pname = strings.Split(pluginName, "/")[1]
		}

		installPath := filepath.Join(m.PathMeta.PluginPath, pname)
		if util.FileExists(installPath) {
			return fmt.Errorf("plugin %s already exists", pname)
		}

		pluginManifest, err := m.fetchPluginManifest(m.GetRegistryAddress(pname + ".json"))
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
	// set alias name
	pname := tempPlugin.Name
	if len(alias) > 0 {
		pname = alias
	}
	installPath := filepath.Join(m.PathMeta.PluginPath, pname)
	if util.FileExists(installPath) {
		return fmt.Errorf("plugin %s already exists", pname)
	}
	if err = os.Rename(tempPlugin.Path, installPath); err != nil {
		return fmt.Errorf("install plugin error: %w", err)
	}
	pterm.Println("Plugin info:")
	pterm.Println("Name    ", "->", pterm.LightBlue(tempPlugin.Name))
	pterm.Println("Version ", "->", pterm.LightBlue(tempPlugin.Version))
	pterm.Println("Homepage", "->", pterm.LightBlue(tempPlugin.Homepage))
	pterm.Println("Desc    ", "->", pterm.LightBlue(tempPlugin.Description))

	// print some notes if there are
	if len(tempPlugin.Notes) != 0 {
		fmt.Println(pterm.LightYellow("Notes:"))
		for _, note := range tempPlugin.Notes {
			fmt.Println("  ", note)
		}
	}
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
		return nil, fmt.Errorf("unsupported %s type plugin to install, only supoort .lua or .zip", ext)
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
	if !util.FileExists(cleanFlagPath) {
		_ = os.WriteFile(cleanFlagPath, []byte(strconv.FormatInt(util.GetBeginOfToday(), 10)), 0777)
	} else {
		if str, err := os.ReadFile(cleanFlagPath); err == nil {
			if i, err := strconv.ParseInt(string(str), 10, 64); err == nil && !util.IsBeforeToday(i) {
				return
			}
		}
	}
	dir, err := os.ReadDir(m.PathMeta.TempPath)
	if err == nil {
		_ = os.RemoveAll(m.PathMeta.CurTmpPath)
		for _, file := range dir {
			if !file.IsDir() {
				continue
			}
			names := strings.SplitN(file.Name(), "-", 2)
			if len(names) != 2 {
				continue
			}
			timestamp := names[0]
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

func NewSdkManager() *Manager {
	meta, err := newPathMeta()
	if err != nil {
		panic("Init path meta error")
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
