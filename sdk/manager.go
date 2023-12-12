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
	"github.com/aooohan/version-fox/plugin"
	"github.com/aooohan/version-fox/util"
	"github.com/pterm/pterm"
	"github.com/pterm/pterm/putils"
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
	if err := source.Uninstall(Version(config.Version)); err != nil {
		return err
	}
	remainVersion := source.List()
	if len(remainVersion) == 0 {
		_ = os.RemoveAll(source.sdkPath)
	}
	pterm.Println("Auto switch to the other version.")
	firstVersion := remainVersion[0]
	return source.Use(firstVersion)
}

func (m *Manager) Search(config Arg) error {
	source := m.sdkMap[config.Name]
	if source == nil {
		pterm.Printf("%s not supported\n", config.Name)
		return fmt.Errorf("%s not supported", config.Name)
	}
	result := source.Search(config.Version)
	if len(result) == 0 {
		pterm.Println("No available version.")
		return nil
	}
	for _, version := range result {
		pterm.Println("->", fmt.Sprintf("v%s", version))
	}
	return nil
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
			DefaultText:   "Please select an option",
			Options:       []string{},
			OptionStyle:   &pterm.ThemeDefault.DefaultText,
			DefaultOption: "",
			MaxHeight:     5,
			Selector:      "->",
			SelectorStyle: &pterm.ThemeDefault.SuccessMessageStyle,
			Filter:        true,
		}
		result, _ := selectPrinter.
			WithOptions(arr).
			Show(fmt.Sprintf("Please select a version of %s", config.Name))

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
		pterm.Println("No available version.")
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
	_ = filepath.Walk(m.pluginPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, ".lua") {
			source := plugin.NewLuaSource(path, m.osType, m.archType)
			if source == nil {
				return nil
			}
			sdk, _ := NewSdk(m, source)
			m.sdkMap[strings.ToLower(source.Name)] = sdk
		}
		return nil
	})
}

func (m *Manager) Close() {
	for _, handler := range m.sdkMap {
		handler.Close()
	}
}

func (m *Manager) Remove(pluginName string) error {
	return nil
}

func (m *Manager) Update(pluginName string) error {
	return nil
}

func (m *Manager) Add(pluginName, url string) error {
	return nil
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
