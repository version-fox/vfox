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

package plugin

import (
	"os"
	"path/filepath"
	"strings"
)

type Manager struct {
	plugins  map[string]*LuaPlugin
	Path     string
	aliasMap map[string]string
}

func (m *Manager) Add(url, name string) error {
	return nil
}

func (m *Manager) Remove(name string) error {
	return nil
}

func (m *Manager) List(name string) []*LuaPlugin {
	return nil
}

func (m *Manager) Update(name string) error {
	return nil
}

func (m *Manager) Get(name string) *LuaPlugin {
	return nil
}

func (m *Manager) Info(name string) string {
	return ""
}

func (m *Manager) Close() {

}

func NewPluginManager(pluginPath string) (*Manager, error) {
	// TODO alias name
	var plugins = make(map[string]*LuaPlugin)
	_ = filepath.Walk(pluginPath, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, ".lua") {
			source := NewLuaSource(path)
			if source == nil {
				return nil
			}
			plugins[strings.ToLower(source.Name())] = source
		}
		return nil
	})
	return &Manager{
		plugins: plugins,
		Path:    pluginPath,
	}, nil
}
