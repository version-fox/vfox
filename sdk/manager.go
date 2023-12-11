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
	PluginManager *plugin.Manager
	EnvManager    env.Manager
	osType        util.OSType
	archType      util.ArchType
}

func (s *Manager) Install(config Arg) error {
	source := s.sdkMap[config.Name]
	if source == nil {
		return fmt.Errorf("%s not supported", config.Name)
	}
	if err := source.Install(Version(config.Version)); err != nil {
		return err
	}
	return nil
}

func (s *Manager) Uninstall(config Arg) error {
	source := s.sdkMap[config.Name]
	if source == nil {
		return fmt.Errorf("%s not supported", config.Name)
	}
	return s.sdkMap[config.Name].Uninstall(Version(config.Version))
}

func (s *Manager) Search(config Arg) error {
	source := s.sdkMap[config.Name]
	if source == nil {
		return fmt.Errorf("%s not supported", config.Name)
	}
	return s.sdkMap[config.Name].Search(config.Version)
}

func (s *Manager) Use(config Arg) error {
	source := s.sdkMap[config.Name]
	if source == nil {
		return fmt.Errorf("%s not supported", config.Name)
	}
	return s.sdkMap[config.Name].Use(Version(config.Version))
}

func (s *Manager) List(arg Arg) error {
	if arg.Name == "" {
		for name, _ := range s.sdkMap {
			fmt.Println("All current plugins: ")
			fmt.Printf("-> %s\n", name)
		}
		return nil
	}
	source := s.sdkMap[arg.Name]
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

func (s *Manager) Current(sdkName string) error {
	source := s.sdkMap[sdkName]
	if source == nil {
		return fmt.Errorf("%s not supported", sdkName)
	}
	current := source.Current()
	println(fmt.Sprintf("-> \t  %s", current))
	return nil
}

func (s *Manager) Close() {
	s.PluginManager.Close()
	for _, handler := range s.sdkMap {
		handler.Close()
	}
}

func NewSdkManager() *Manager {
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		panic("Get user home dir error")
	}
	pluginPath := filepath.Join(userHomeDir, ".version-fox", "plugin")
	configPath := filepath.Join(userHomeDir, ".version-fox")
	pluginManager, err := plugin.NewPluginManager(pluginPath)
	if err != nil {
		panic("Init plugin manager error")
	}
	envManger, err := env.NewEnvManager(configPath)
	if err != nil {
		panic("Init env manager error")
	}
	manager := &Manager{
		configPath:    configPath,
		sdkCachePath:  filepath.Join(userHomeDir, ".version-fox", ".cache"),
		envConfigPath: filepath.Join(userHomeDir, ".version-fox", "env.sh"),
		PluginManager: pluginManager,
		EnvManager:    envManger,
		sdkMap:        make(map[string]*Sdk),
		osType:        util.GetOSType(),
		archType:      util.GetArchType(),
	}
	_ = os.MkdirAll(manager.sdkCachePath, 0755)
	_ = os.MkdirAll(pluginPath, 0755)
	if !util.FileExists(manager.envConfigPath) {
		_, _ = os.Create(manager.envConfigPath)
	}

	return manager
}
