/*
 *    Copyright 2024 [lihan aooohan@gmail.com]
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
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/version-fox/vfox/internal/env"
	"github.com/version-fox/vfox/internal/shell"
	"github.com/version-fox/vfox/internal/toolversion"
	"github.com/version-fox/vfox/plugin"

	"github.com/pterm/pterm"
	"github.com/version-fox/vfox/util"
)

const (
	pluginIndexUrl = "https://version-fox.github.io/version-fox-plugins/"
)

type Arg struct {
	Name    string
	Version string
}

type Manager struct {
	configPath     string
	sdkCachePath   string
	envConfigPath  string
	pluginPath     string
	executablePath string
	sdkMap         map[string]*Sdk
	EnvManager     env.Manager
	Shell          *shell.Shell
	osType         util.OSType
	archType       util.ArchType
}

// Use examples:
// 1. vfox use (--project)
// 2. vfox use java
// 3. vfox use java@11
// 4. vfox use --project java@11
// 5. vfox use --session java@11
// 6. vfox use --global java@11
func (m *Manager) Use(arg Arg, useScope UseScope) error {
	pwd, err := os.Getwd()
	if err != nil {
		pterm.Printf("Get current dir error, err: %s\n", err)
		return err
	}
	tv, _ := toolversion.NewToolVersions(pwd)
	var sdks []*Arg
	if arg.Name == "" {
		for sdkName, version := range tv.Sdks {
			_, ok := m.sdkMap[sdkName]
			if !ok {
				pterm.Printf("%s not supported.\n", sdkName)
				continue
			}
			sdks = append(sdks, &Arg{
				Name:    sdkName,
				Version: version,
			})
		}
		if len(sdks) == 0 {
			pterm.Println("Invalid parameter. format: <sdk-name>[@<version>]")
			return fmt.Errorf("invalid parameter")
		}
	} else if arg.Version == "" {
		source, ok := m.sdkMap[arg.Name]
		if !ok {
			pterm.Printf("%s not supported.\n", arg.Name)
			return fmt.Errorf("%s not supported", arg.Name)
		}
		version, ok := tv.Sdks[arg.Name]
		if ok {
			sdks = append(sdks, &Arg{
				Name:    arg.Name,
				Version: version,
			})
		} else {
			list := source.List()
			var arr []string
			for _, version := range list {
				arr = append(arr, string(version))
			}
			selectPrinter := pterm.InteractiveSelectPrinter{
				TextStyle:     &pterm.ThemeDefault.DefaultText,
				OptionStyle:   &pterm.ThemeDefault.DefaultText,
				Options:       arr,
				DefaultOption: "",
				MaxHeight:     5,
				Selector:      "->",
				SelectorStyle: &pterm.ThemeDefault.SuccessMessageStyle,
				Filter:        true,
				OnInterruptFunc: func() {
					os.Exit(0)
				},
			}
			result, _ := selectPrinter.Show(fmt.Sprintf("Please select a version of %s", arg.Name))
			sdks = append(sdks, &Arg{
				Name:    arg.Name,
				Version: result,
			})
		}
	} else {
		_, ok := m.sdkMap[arg.Name]
		if !ok {
			pterm.Printf("%s not supported.\n", arg.Name)
			return fmt.Errorf("%s not supported", arg.Name)
		}
		sdks = append(sdks, &arg)
	}
	for _, sdk := range sdks {
		source := m.sdkMap[sdk.Name]
		if useScope == Project {
			err = source.Use(Version(sdk.Version), env.Local)
			if err != nil {
				return err
			}
			err = tv.Add(arg.Name, sdk.Version)
			if err != nil {
				pterm.Printf("Failed to record %s version to %s\n", sdk.Version, tv)
				return err
			}
		} else {
			scope := env.Global
			if useScope == Session {
				scope = env.Local
			} else {
				scope = env.Global
			}
			err := source.Use(Version(sdk.Version), scope)
			if err != nil {
				return err
			}
		}
	}
	// TODO
	//return m.Shell.ReOpen()
	return err
}

// LookupSdk lookup sdk by name
func (m *Manager) LookupSdk(name string) (*Sdk, error) {
	pluginPath := filepath.Join(m.pluginPath, strings.ToLower(name)+".lua")
	if !util.FileExists(pluginPath) {
		return nil, fmt.Errorf("%s not installed", name)
	}
	content, err := m.loadLuaFromFileOrUrl(pluginPath)
	if err != nil {
		return nil, err
	}
	luaPlugin, err := NewLuaPlugin(content, pluginPath, m.osType, m.archType)
	if err != nil {
		return nil, err
	}
	sdk, _ := NewSdk(m, luaPlugin)
	return sdk, nil
}

func (m *Manager) LoadAllSdk() (map[string]*Sdk, error) {
	dir, err := os.ReadDir(m.pluginPath)
	if err != nil {
		return nil, fmt.Errorf("load sdks error: %w", err)
	}
	sdkMap := make(map[string]*Sdk)
	for _, d := range dir {
		if d.IsDir() {
			continue
		}
		if strings.HasSuffix(d.Name(), ".lua") {
			// filename first as sdk name
			path := filepath.Join(m.pluginPath, d.Name())
			content, _ := m.loadLuaFromFileOrUrl(path)
			source, err := NewLuaPlugin(content, path, m.osType, m.archType)
			if err != nil {
				pterm.Printf("Failed to load %s plugin, err: %s\n", path, err)
				continue
			}
			sdk, _ := NewSdk(m, source)
			name := strings.TrimSuffix(filepath.Base(path), ".lua")
			sdkMap[strings.ToLower(name)] = sdk
		}
	}
	return sdkMap, nil
}

func (m *Manager) Close() {
	for _, handler := range m.sdkMap {
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
	pPath := filepath.Join(m.pluginPath, pluginName+".lua")
	pterm.Printf("Removing %s plugin...\n", pPath)
	err = os.RemoveAll(pPath)
	if err != nil {
		return fmt.Errorf("remove failed, err: %w", err)
	}
	pterm.Printf("Removing %s sdk...\n", source.InstallPath)
	err = os.RemoveAll(source.InstallPath)
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
		pterm.Printf("Failed to load %s plugin, err: %s\n", updateUrl, err)
		return fmt.Errorf("fetch plugin failed")
	}
	source, err := NewLuaPlugin(content, updateUrl, m.osType, m.archType)
	if err != nil {
		pterm.Printf("Check %s plugin failed, err: %s\n", updateUrl, err)
		return err
	}
	success := false
	backupPath := sdk.Plugin.SourcePath + ".bak"
	err = util.CopyFile(sdk.Plugin.SourcePath, backupPath)
	if err != nil {
		pterm.Printf("Backup %s plugin failed, err: %s\n", updateUrl, err)
		return fmt.Errorf("backup failed")
	}
	defer func() {
		if success {
			_ = os.Remove(backupPath)
		} else {
			_ = os.Rename(backupPath, sdk.Plugin.SourcePath)
		}
	}()
	pterm.Println("Checking plugin version...")
	if util.CompareVersion(source.Version, sdk.Plugin.Version) <= 0 {
		pterm.Println("The plugin is already the latest version.")
		return fmt.Errorf("already the latest version")
	}
	err = os.WriteFile(sdk.Plugin.SourcePath, []byte(content), 0644)
	if err != nil {
		pterm.Printf("Update %s plugin failed, err: %s\n", updateUrl, err)
		return fmt.Errorf("write file error")
	}
	success = true
	pterm.Printf("Update %s plugin successfully! version: %s \n", pterm.LightGreen(pluginName), pterm.LightBlue(source.Version))
	return nil
}

func (m *Manager) Add(pluginName, url, alias string) error {
	pname := pluginName
	// official plugin
	if len(url) == 0 {
		args := strings.Split(pluginName, "/")
		if len(args) < 2 {
			pterm.Println("Invalid plugin name. Format: <category>/<plugin-name>")
			return fmt.Errorf("invalid plugin name")
		}
		category := args[0]
		name := args[1]
		pname = name
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

	if len(alias) > 0 {
		pname = alias
	}

	destPath := filepath.Join(m.pluginPath, pname+".lua")
	if util.FileExists(destPath) {
		pterm.Printf("Plugin %s already exists, please use %s to remove it first.\n", pterm.LightGreen(pname), pterm.LightBlue("vfox remove "+pname))
		return fmt.Errorf("plugin already exists")
	}

	pterm.Printf("Adding plugin from %s...\n", url)
	content, err := m.loadLuaFromFileOrUrl(url)
	if err != nil {
		pterm.Printf("Failed to load %s plugin, err: %s\n", url, err)
		return fmt.Errorf("install failed")
	}
	pterm.Println("Checking plugin...")
	source, err := NewLuaPlugin(content, url, m.osType, m.archType)
	if err != nil {
		pterm.Printf("Check %s plugin failed, err: %s\n", url, err)
		return err
	}
	defer source.Close()
	err = os.WriteFile(destPath, []byte(content), 0644)
	if err != nil {
		pterm.Printf("Add %s plugin failed, err: %s\n", url, err)
		return fmt.Errorf("write file error")
	}
	pterm.Println("Plugin info:")
	pterm.Println("Name   ", "->", pterm.LightBlue(source.Name))
	pterm.Println("Author ", "->", pterm.LightBlue(source.Author))
	pterm.Println("Version", "->", pterm.LightBlue(source.Version))
	pterm.Println("Desc   ", "->", pterm.LightBlue(source.Description))
	pterm.Println("Path   ", "->", pterm.LightBlue(destPath))
	pterm.Printf("Add %s plugin successfully! \n", pterm.LightGreen(pname))
	pterm.Printf("Please use `%s` to install the version you need.\n", pterm.LightBlue(fmt.Sprintf("vfox install %s@<version>", pname)))
	return nil
}

func (m *Manager) loadLuaFromFileOrUrl(path string) (string, error) {
	if !strings.HasSuffix(path, ".lua") {
		return "", fmt.Errorf("%s not a lua file", path)
	}
	if strings.HasPrefix(path, "https://") || strings.HasPrefix(path, "http://") {
		resp, err := http.Get(path)
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

func (m *Manager) Available() ([]*plugin.Category, error) {
	// TODO proxy
	resp, err := http.Get(pluginIndexUrl)
	if err != nil {
		pterm.Printf("Get plugin index error, err: %s\n", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		pterm.Printf("Get plugin index error, status code: %d\n", resp.StatusCode)
		return nil, fmt.Errorf("get plugin index error")
	}
	if str, err := io.ReadAll(resp.Body); err != nil {
		pterm.Printf("Read plugin index error, err: %s\n", err)
		return nil, fmt.Errorf("read plugin index error")
	} else {
		var categories []*plugin.Category
		err = json.Unmarshal(str, &categories)
		if err != nil {
			pterm.Printf("Parse plugin index error, err: %s\n", err)
			return nil, fmt.Errorf("parse plugin index error")
		}
		return categories, nil
	}
}

func (m *Manager) Activate(writer io.Writer, name string) error {
	path := m.executablePath
	path = strings.Replace(path, "\\", "/", -1)
	ctx := struct {
		SelfPath string
	}{
		SelfPath: path,
	}
	s := shell.NewShell(name)
	if s == nil {
		return fmt.Errorf("unknow target shell %s", name)
	}
	shellEnv, _ := m.EnvManager.ToShellEnv()
	exportStr := s.Export(shellEnv)
	str, err := s.Activate()
	if err != nil {
		return err
	}
	script := exportStr + "\n" + str
	hookTemplate, err := template.New("hook").Parse(script)
	if err != nil {
		return nil
	}
	return hookTemplate.Execute(writer, ctx)
}

func (m *Manager) Env(writer io.Writer, name string) error {
	s := shell.NewShell(name)
	if s == nil {
		return fmt.Errorf("unknow target shell %s", name)
	}
	//s.Export()
	return nil
}

func NewSdkManager() *Manager {
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		panic("Get user home dir error")
	}
	pluginPath := filepath.Join(userHomeDir, ".version-fox", "plugin")
	configPath := filepath.Join(userHomeDir, ".version-fox")
	sdkCachePath := filepath.Join(userHomeDir, ".version-fox", "cache")
	_ = os.MkdirAll(sdkCachePath, 0755)
	_ = os.MkdirAll(pluginPath, 0755)
	exePath, err := os.Executable()
	if err != nil {
		panic("Get executable path error")
	}
	if err != nil {
		panic("Init shell error")
	}
	envManger, err := env.NewEnvManager(configPath, nil)
	if err != nil {
		panic("Init env manager error")
	}
	manager := &Manager{
		configPath:     configPath,
		sdkCachePath:   sdkCachePath,
		pluginPath:     pluginPath,
		executablePath: exePath,
		EnvManager:     envManger,
		sdkMap:         make(map[string]*Sdk),
		osType:         util.GetOSType(),
		archType:       util.GetArchType(),
	}
	return manager
}
