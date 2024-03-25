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
	"fmt"
	"io"
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
	pluginIndexUrl      = "https://version-fox.github.io/version-fox-plugins/"
	cleanupFlagFilename = ".cleanup"
)

type Arg struct {
	Name    string
	Version string
}

type Manager struct {
	PathMeta   *PathMeta
	openSdks   map[string]*Sdk
	EnvManager env.Manager
	Record     env.Record
	osType     util.OSType
	archType   util.ArchType
	Config     *config.Config
}

func (m *Manager) EnvKeys() (*env.Envs, error) {
	shellEnvs := &env.Envs{
		Variables: make(env.Vars),
		Paths:     make(env.Paths, 0),
	}
	for k, v := range m.Record.Export() {
		if lookupSdk, err := m.LookupSdk(k); err == nil {
			if ek, err := lookupSdk.EnvKeys(Version(v)); err == nil {
				for key, value := range ek.Variables {
					shellEnvs.Variables[key] = value
				}
				shellEnvs.Paths = append(shellEnvs.Paths, ek.Paths...)
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
			return nil, fmt.Errorf("%s not installed", name)
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

func (m *Manager) LoadAllSdk() (map[string]*Sdk, error) {
	dir, err := os.ReadDir(m.PathMeta.PluginPath)
	if err != nil {
		return nil, fmt.Errorf("load sdks error: %w", err)
	}
	sdkMap := make(map[string]*Sdk)
	for _, d := range dir {
		sdkName := d.Name()
		path := filepath.Join(m.PathMeta.PluginPath, sdkName, "main.lua")
		if util.FileExists(path) {
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
			path = filepath.Join(newPluginDir, "main.lua")
			sdkName = strings.TrimSuffix(sdkName, ".lua")
		} else {
			continue
		}
		source, err := NewLuaPlugin(filepath.Dir(path), m)
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
	updateUrl := sdk.Plugin.UpdateUrl
	if updateUrl == "" {
		return fmt.Errorf("%s plugin not support update", pluginName)
	}
	pterm.Printf("Checking %s plugin...\n", updateUrl)
	content, err := m.loadLuaFromFileOrUrl(updateUrl)
	if err != nil {
		return fmt.Errorf("fetch plugin failed, err: %w", err)
	}
	source, err := NewLegacyLuaPlugin(content, updateUrl, m)
	if err != nil {
		return fmt.Errorf("check %s plugin failed, err: %w", updateUrl, err)
	}
	success := false
	backupPath := sdk.Plugin.Path + ".bak"
	err = util.CopyFile(sdk.Plugin.Path, backupPath)
	if err != nil {
		return fmt.Errorf("backup %s plugin failed, err: %w", updateUrl, err)
	}
	defer func() {
		if success {
			_ = os.Remove(backupPath)
		} else {
			_ = os.Rename(backupPath, sdk.Plugin.Path)
		}
	}()
	pterm.Println("Checking plugin version...")
	if util.CompareVersion(source.Version, sdk.Plugin.Version) <= 0 {
		success = true
		pterm.Printf("the plugin is already the latest version")
		return nil
	}
	err = os.WriteFile(sdk.Plugin.Path, []byte(content), 0644)
	if err != nil {
		return fmt.Errorf("update %s plugin failed: %w", updateUrl, err)
	}
	success = true
	pterm.Printf("Update %s plugin successfully! version: %s \n", pterm.LightGreen(pluginName), pterm.LightBlue(source.Version))
	return nil
}

func (m *Manager) Add(pluginName, url, alias string) error {
	// official plugin
	if len(url) == 0 {
		args := strings.Split(pluginName, "/")
		if len(args) < 2 {
			return fmt.Errorf("invalid plugin name, format: <category>/<plugin-name>")
		}
		category := args[0]
		name := args[1]
		availablePlugins, err := m.Available()
		if err != nil {
			return err
		}
		for _, available := range availablePlugins {
			if category == available.Name {
				for _, p := range available.Plugins {
					if name == p.Filename {
						url = p.Url
						break
					}
				}
			}
		}
	}

	pterm.Printf("Loading plugin from %s...\n", url)
	content, err := m.loadLuaFromFileOrUrl(url)
	if err != nil {
		return fmt.Errorf("failed to load plugin: %w", err)
	}
	pterm.Println("Checking plugin...")
	source, err := NewLegacyLuaPlugin(content, url, m)
	if err != nil {
		return fmt.Errorf("check plugin error: %w", err)
	}
	defer source.Close()

	// Check if the plugin is compatible with the current runtime
	if source.MinRuntimeVersion != "" && util.CompareVersion(source.MinRuntimeVersion, RuntimeVersion) > 0 {
		return fmt.Errorf("check failed: this plugin is not compatible with current vfox (>= %s), please upgrade vfox version to latest", source.MinRuntimeVersion)
	}

	pname := source.Name
	if len(alias) > 0 {
		pname = alias
	}
	destPath := filepath.Join(m.PathMeta.PluginPath, pname, "main.lua")
	if util.FileExists(destPath) {
		return fmt.Errorf("plugin %s already exists", pname)
	}
	if err = os.Mkdir(filepath.Dir(destPath), 0777); err != nil {
		return fmt.Errorf("add plugin error: %w", err)
	}
	if err = os.WriteFile(destPath, []byte(content), 0777); err != nil {
		return fmt.Errorf("add plugin error: %w", err)
	}
	pterm.Println("Plugin info:")
	pterm.Println("Name   ", "->", pterm.LightBlue(source.Name))
	pterm.Println("Version", "->", pterm.LightBlue(source.Version))
	pterm.Println("Desc   ", "->", pterm.LightBlue(source.Description))
	pterm.Println("Path   ", "->", pterm.LightBlue(destPath))
	pterm.Printf("Add %s plugin successfully! \n", pterm.LightGreen(pname))
	pterm.Printf("Please use `%s` to install the version you need.\n", pterm.LightBlue(fmt.Sprintf("vfox install %s@<version>", pname)))
	return nil
}

func (m *Manager) httpClient() *http.Client {
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

func (m *Manager) loadLuaFromFileOrUrl(path string) (string, error) {
	if !strings.HasSuffix(path, ".lua") {
		return "", fmt.Errorf("%s not a lua file", path)
	}
	if strings.HasPrefix(path, "https://") || strings.HasPrefix(path, "http://") {
		client := m.httpClient()
		resp, err := client.Get(path)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()
		cd := resp.Header.Get("Content-Disposition")
		if strings.HasPrefix(cd, "attachment") {
			return "", fmt.Errorf("not a lua file")
		}
		if resp.StatusCode == http.StatusNotFound {
			return "", fmt.Errorf("file not found")
		}
		if str, err := io.ReadAll(resp.Body); err != nil {
			return "", err
		} else {
			return string(str), nil
		}
	}

	if !util.FileExists(path) {
		return "", fmt.Errorf("file not found")
	}
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	str, err := io.ReadAll(file)
	if err != nil {
		return "", err
	}
	return string(str), nil

}

func (m *Manager) Available() ([]*Category, error) {
	client := m.httpClient()
	resp, err := client.Get(pluginIndexUrl)
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
		var categories []*Category
		err = json.Unmarshal(str, &categories)
		if err != nil {
			return nil, fmt.Errorf("parse plugin index error: %w", err)
		}
		return categories, nil
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

func NewSdkManagerWithSource(sources ...RecordSource) *Manager {
	if env.IsHookEnv() {
		return newSdkManagerWithSource(sources...)
	} else {
		return newSdkManagerWithSource(SessionRecordSource, GlobalRecordSource, ProjectRecordSource)
	}
}

func newSdkManagerWithSource(sources ...RecordSource) *Manager {
	meta, err := newPathMeta()
	if err != nil {
		panic("Init path meta error")
	}

	var paths []string
	for _, source := range sources {
		switch source {
		case GlobalRecordSource:
			paths = append(paths, meta.ConfigPath)
		case ProjectRecordSource:
			paths = append(paths, meta.WorkingDirectory)
		case SessionRecordSource:
			paths = append(paths, meta.CurTmpPath)
		}
	}
	var record env.Record
	if len(paths) == 0 {
		record = env.EmptyRecord
	} else if len(paths) == 1 {
		r, err := env.NewRecord(paths[0])
		if err != nil {
			panic(err)
		}
		record = r
	} else {
		r, err := env.NewRecord(paths[0], paths[1:]...)
		if err != nil {
			panic(err)
		}
		record = r
	}
	return newSdkManager(record, meta)
}

func NewSdkManager(sources ...RecordSource) *Manager {
	if len(sources) == 0 {
		return NewSdkManagerWithSource(SessionRecordSource, ProjectRecordSource)
	}
	return newSdkManagerWithSource(sources...)
}

func newSdkManager(record env.Record, meta *PathMeta) *Manager {
	envManger, err := env.NewEnvManager(meta.ConfigPath)
	if err != nil {
		panic("Init env manager error")
	}
	c, err := config.NewConfig(meta.ConfigPath)
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
		Record:     record,
		openSdks:   make(map[string]*Sdk),
		osType:     util.GetOSType(),
		archType:   util.GetArchType(),
		Config:     c,
	}
	return manager
}
