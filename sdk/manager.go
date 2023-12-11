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
	return m.sdkMap[config.Name].Uninstall(Version(config.Version))
}

func (m *Manager) Search(config Arg) error {
	source := m.sdkMap[config.Name]
	if source == nil {
		return fmt.Errorf("%s not supported", config.Name)
	}
	return m.sdkMap[config.Name].Search(config.Version)
}

func (m *Manager) Use(config Arg) error {
	source := m.sdkMap[config.Name]
	if source == nil {
		return fmt.Errorf("%s not supported", config.Name)
	}
	return m.sdkMap[config.Name].Use(Version(config.Version))
}

func (m *Manager) List(arg Arg) error {
	if arg.Name == "" {
		for name, _ := range m.sdkMap {
			fmt.Println("All current plugins: ")
			fmt.Printf("-> %s\n", name)
		}
		return nil
	}
	source := m.sdkMap[arg.Name]
	if source == nil {
		return fmt.Errorf("%s not supported", arg.Name)
	}
	curVersion := source.Current()
	list := source.List()
	if len(list) == 0 {
		fmt.Printf("-> %s\n", "no version installed")
		return nil
	}
	for _, version := range list {
		if version == curVersion {
			fmt.Printf("-> %s  (current)\n", version)
		} else {
			fmt.Printf("-> %s\n", version)
		}
	}
	return nil
}

func (m *Manager) Current(sdkName string) error {
	source := m.sdkMap[sdkName]
	if source == nil {
		return fmt.Errorf("%s not supported", sdkName)
	}
	current := source.Current()
	println(fmt.Sprintf("-> \t  %s", current))
	return nil
}

func (m *Manager) loadSdk() {
	_ = filepath.Walk(m.pluginPath, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, ".lua") {
			source := plugin.NewLuaSource(path)
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
		EnvManager:    envManger,
		sdkMap:        make(map[string]*Sdk),
		osType:        util.GetOSType(),
		archType:      util.GetArchType(),
	}

	manager.loadSdk()

	return manager
}
