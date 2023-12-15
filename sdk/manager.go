/*
 *    Copyright 2023 [lihan aooohan@gmail.com]
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
	"fmt"
	"github.com/aooohan/version-fox/env"
	"github.com/aooohan/version-fox/printer"
	"github.com/aooohan/version-fox/util"
	"github.com/pterm/pterm"
	"github.com/pterm/pterm/putils"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type Arg struct {
	Name    string
	Version string
}

type Manager struct {
	configPath    string
	sdkCachePath  string
	envConfigPath string
	pluginPath    string
	sdkMap        map[string]*Sdk
	EnvManager    env.Manager
	osType        util.OSType
	archType      util.ArchType
}

func (m *Manager) Install(config Arg) error {
	source := m.sdkMap[config.Name]
	if source == nil {
		pterm.PrintOnErrorf("%s not supported\n", config.Name)
		return fmt.Errorf("%s not supported", config.Name)
	}
	if err := source.Install(Version(config.Version)); err != nil {
		return err
	}
	return nil
}

func (m *Manager) Uninstall(config Arg) error {
	source := m.sdkMap[config.Name]
	if source == nil {
		return fmt.Errorf("%s not supported", config.Name)
	}
	version := Version(config.Version)
	cv := source.Current()
	if err := source.Uninstall(version); err != nil {
		return err
	}
	remainVersion := source.List()
	if len(remainVersion) == 0 {
		_ = os.RemoveAll(source.sdkRootPath)
		return nil
	}
	if cv == version {
		pterm.Println("Auto switch to the other version.")
		firstVersion := remainVersion[0]
		return source.Use(firstVersion)
	}
	return nil
}

func (m *Manager) Available(sdkName string) error {
	source := m.sdkMap[sdkName]
	if source == nil {
		pterm.Printf("%s not supported\n", sdkName)
		return fmt.Errorf("%s not supported", sdkName)
	}
	result, err := source.Available()
	if err != nil {
		pterm.Printf("Plugin error: %s\n", err)
		return err
	}
	if len(result) == 0 {
		pterm.Println("No Available version.")
		return nil
	}
	kvSelect := printer.PageKVSelect{
		TopText: "Please select a version of " + sdkName,
		Filter:  true,
		Size:    20,
		SourceFunc: func(page, size int) ([]*printer.KV, error) {
			start := page * size
			end := start + size

			if start > len(result) {
				return nil, fmt.Errorf("page is out of range")
			}
			if end > len(result) {
				end = len(result)
			}
			versions := result[start:end]
			var arr []*printer.KV
			for _, p := range versions {
				var value string
				if p.Main.Note != "" {
					value = fmt.Sprintf("v%s (%s)", p.Main.Version, p.Main.Note)
				} else {
					value = fmt.Sprintf("v%s", p.Main.Version)
				}
				if len(p.Additional) != 0 {
					var additional []string
					for _, a := range p.Additional {
						additional = append(additional, fmt.Sprintf("%s v%s", a.Name, a.Version))
					}
					value = fmt.Sprintf("%s [%s]", value, strings.Join(additional, ","))
				}
				arr = append(arr, &printer.KV{
					Key:   string(p.Main.Version),
					Value: value,
				})
			}
			return arr, nil
		},
	}
	version, err := kvSelect.Show()
	if err != nil {
		pterm.Printf("Select version error, err: %s\n", err)
		return err
	}
	return source.Install(Version(version.Key))
}

func (m *Manager) Use(config Arg) error {
	source := m.sdkMap[config.Name]
	if source == nil {
		fmt.Printf("%s not supported\n", config.Name)
		return fmt.Errorf("%s not supported", config.Name)
	}
	if config.Version == "" {
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
		result, _ := selectPrinter.Show(fmt.Sprintf("Please select a version of %s", config.Name))

		return source.Use(Version(result))

	}
	return m.sdkMap[config.Name].Use(Version(config.Version))
}

func (m *Manager) List(arg Arg) error {
	if arg.Name == "" {
		tree := pterm.LeveledList{}
		for name, sdk := range m.sdkMap {
			tree = append(tree, pterm.LeveledListItem{Level: 0, Text: name})
			for _, version := range sdk.List() {
				tree = append(tree, pterm.LeveledListItem{Level: 1, Text: "v" + string(version)})
			}
		}
		// Generate tree from LeveledList.
		root := putils.TreeFromLeveledList(tree)
		root.Text = "All installed sdk versions"
		// Render TreePrinter
		_ = pterm.DefaultTree.WithRoot(root).Render()
		return nil
	}
	source := m.sdkMap[arg.Name]
	if source == nil {
		return fmt.Errorf("%s not supported", arg.Name)
	}
	curVersion := source.Current()
	list := source.List()
	if len(list) == 0 {
		pterm.Println("No Available version.")
		return nil
	}
	for _, version := range list {
		if version == curVersion {
			pterm.Println("->", fmt.Sprintf("v%s", version), pterm.LightGreen("<— current"))
		} else {
			pterm.Println("->", fmt.Sprintf("v%s", version))
		}
	}

	return nil
}

func (m *Manager) Current(sdkName string) error {
	if sdkName == "" {
		for name, sdk := range m.sdkMap {
			current := sdk.Current()
			if current == "" {
				pterm.Printf("%s -> N/A \n", name)
			} else {
				pterm.Printf("%s -> %s\n", name, pterm.LightGreen("v"+string(current)))
			}
		}
		return nil
	}
	source := m.sdkMap[sdkName]
	if source == nil {
		pterm.Printf("%s not supported\n", sdkName)
		return fmt.Errorf("%s not supported", sdkName)
	}
	current := source.Current()
	if current == "" {
		pterm.Printf("No current version of %s\n", sdkName)
		return nil
	}
	pterm.Println("->", pterm.LightGreen("v"+string(current)))
	return nil
}

func (m *Manager) loadSdk() {
	dir, err := os.ReadDir(m.pluginPath)
	if err != nil {
		pterm.Printf("Read plugin dir error, err: %s\n", err)
		return
	}
	for _, d := range dir {
		if d.IsDir() {
			continue
		}
		if strings.HasSuffix(d.Name(), ".lua") {
			// filename first as sdk name
			path := filepath.Join(m.pluginPath, d.Name())
			content, _ := m.loadLuaFromFileOrUrl(path)
			source, err := NewLuaPlugin(content, m.osType, m.archType)
			if err != nil {
				pterm.Printf("Failed to load %s plugin, err: %s\n", path, err)
				continue
			}
			sdk, _ := NewSdk(m, source)
			name := strings.TrimSuffix(filepath.Base(path), ".lua")
			m.sdkMap[strings.ToLower(name)] = sdk
		}
	}
}

func (m *Manager) Close() {
	for _, handler := range m.sdkMap {
		handler.Close()
	}
}

func (m *Manager) Remove(pluginName string) error {
	source := m.sdkMap[pluginName]
	if source == nil {
		pterm.Println("This plugin has not been added.")
		return fmt.Errorf("%s not installed", pluginName)
	}
	pterm.Println("Removing this plugin will remove the installed sdk along with the plugin.")
	result, _ := pterm.DefaultInteractiveConfirm.
		WithTextStyle(&pterm.ThemeDefault.DefaultText).
		WithConfirmStyle(&pterm.ThemeDefault.DefaultText).
		WithRejectStyle(&pterm.ThemeDefault.DefaultText).
		WithDefaultText("Please confirm").
		Show()
	if result {
		source.clearCurrentEnvConfig()
		pPath := filepath.Join(m.pluginPath, pluginName+".lua")
		pterm.Printf("Removing %s plugin...\n", pPath)
		err := os.RemoveAll(pPath)
		if err != nil {
			pterm.Printf("Remove %s plugin failed, err: %s\n", pluginName, err)
			return fmt.Errorf("remove failed")
		}
		pterm.Printf("Removing %s sdk...\n", source.sdkRootPath)
		err = os.RemoveAll(source.sdkRootPath)
		pterm.Printf("Remove %s plugin successfully! \n", pterm.LightGreen(pluginName))
	} else {
		pterm.Println("Remove canceled.")
	}
	return nil
}

func (m *Manager) Update(pluginName string) error {
	return nil
}

func (m *Manager) Add(pluginName, url string) error {
	pterm.Printf("Adding plugin from %s...\n", url)
	content, err := m.loadLuaFromFileOrUrl(url)
	if err != nil {
		pterm.Printf("Failed to load %s plugin, err: %s\n", url, err)
		return fmt.Errorf("install failed")
	}
	pterm.Println("Checking plugin...")
	source, err := NewLuaPlugin(content, m.osType, m.archType)
	if err != nil {
		pterm.Printf("Check %s plugin failed, err: %s", url, err)
	}
	destPath := filepath.Join(m.pluginPath, pluginName+".lua")
	err = os.WriteFile(destPath, []byte(content), 0644)
	if err != nil {
		pterm.Printf("Add %s plugin failed, err: %s\n", url, err)
		return fmt.Errorf("write file error")
	}
	pterm.Println("Plugin info:")
	pterm.Println("Name   ", "->", pterm.LightBlue(source.Name))
	pterm.Println("Author ", "->", pterm.LightBlue(source.Author))
	pterm.Println("Version", "->", pterm.LightBlue(source.Version))
	pterm.Println("Path   ", "->", pterm.LightBlue(destPath))
	pterm.Printf("Add %s plugin successfully! \n", pterm.LightGreen(pluginName))
	pterm.Printf("Please use `%s` to install the version you need.\n", pterm.LightBlue(fmt.Sprintf("vf install %s@<version>", pluginName)))
	return nil
}

func (m *Manager) loadLuaFromFileOrUrl(path string) (string, error) {
	if !strings.HasSuffix(path, ".lua") {
		return "", fmt.Errorf("not a lua file")
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
	str, err := io.ReadAll(file)
	if err != nil {
		return "", err
	}
	return string(str), nil

}

func NewSdkManager() *Manager {
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		panic("Get user home dir error")
	}
	pluginPath := filepath.Join(userHomeDir, ".version-fox", "plugin")
	configPath := filepath.Join(userHomeDir, ".version-fox")
	sdkCachePath := filepath.Join(userHomeDir, ".version-fox", ".cache")
	envConfigPath := filepath.Join(userHomeDir, ".version-fox", "env.sh")
	_ = os.MkdirAll(sdkCachePath, 0755)
	_ = os.MkdirAll(pluginPath, 0755)
	if !util.FileExists(envConfigPath) {
		_, _ = os.Create(envConfigPath)
	}
	envManger, err := env.NewEnvManager(configPath)
	if err != nil {
		panic("Init env manager error")
	}
	manager := &Manager{
		configPath:    configPath,
		sdkCachePath:  sdkCachePath,
		envConfigPath: envConfigPath,
		pluginPath:    pluginPath,
		EnvManager:    envManger,
		sdkMap:        make(map[string]*Sdk),
		osType:        util.GetOSType(),
		archType:      util.GetArchType(),
	}

	manager.loadSdk()

	return manager
}
