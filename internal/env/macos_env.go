//go:build darwin || linux

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

package env

import (
	"fmt"
	"github.com/version-fox/vfox/internal/util"
	"os"
	"strings"
)

type macosEnvManager struct {
	envMap        map[string]string
	deletedEnvMap map[string]struct{}
	// $PATH
	paths          []string
	pathMap        map[string]struct{}
	deletedPathMap map[string]struct{}
}

func (m *macosEnvManager) Paths(paths []string) string {
	set := util.NewSortedSetWithSlice[string](paths)
	return strings.Join(set.Slice(), ":")
}

func (m *macosEnvManager) Close() error {
	return nil
}

func (m *macosEnvManager) Load(envs *Envs) error {
	for k, v := range envs.Variables {
		m.envMap[k] = *v
	}
	for _, path := range envs.Paths.Slice() {
		if _, ok := m.pathMap[path]; ok {
			continue
		}
		m.paths = append(m.paths, path)
		m.pathMap[path] = struct{}{}
	}
	return nil
}
func (m *macosEnvManager) Remove(envs *Envs) error {
	for k, _ := range envs.Variables {
		if k == "PATH" {
			return fmt.Errorf("can not remove PATH variable")
		}
		delete(m.envMap, k)
		m.deletedEnvMap[k] = struct{}{}
	}
	for _, k := range envs.Paths.Slice() {
		if _, ok := m.pathMap[k]; ok {
			delete(m.pathMap, k)
			var newPaths []string
			for _, v := range m.paths {
				if v != k {
					newPaths = append(newPaths, v)
				}
			}
			m.paths = newPaths
			m.deletedPathMap[k] = struct{}{}
		}
	}
	return nil
}

func (m *macosEnvManager) Flush() error {
	for k, _ := range m.deletedEnvMap {
		if err := os.Unsetenv(k); err != nil {
			return err
		}
	}
	for k, v := range m.envMap {
		if err := os.Setenv(k, v); err != nil {
			return err
		}
	}
	var newPaths []string
	for path := range m.pathMap {
		newPaths = append(newPaths, path)
	}
	oldPaths := strings.Split(os.Getenv("PATH"), ":")
	for _, path := range oldPaths {
		if _, ok := m.deletedPathMap[path]; ok {
			continue
		}
		if _, ok := m.pathMap[path]; ok {
			continue
		}
		newPaths = append(newPaths, path)
	}
	return os.Setenv("PATH", strings.Join(newPaths, ":"))
}

func (m *macosEnvManager) Get(key string) (string, bool) {
	if key == "PATH" {
		return m.pathEnvValue(), true
	} else {
		s, ok := m.envMap[key]
		return s, ok
	}
}

func (m *macosEnvManager) pathEnvValue() string {
	var pathValues []string
	for k, _ := range m.pathMap {
		pathValues = append(pathValues, k)
	}
	pathValues = append(pathValues, "$PATH")
	return strings.Join(pathValues, ":")
}

func NewEnvManager(vfConfigPath string) (Manager, error) {
	manager := &macosEnvManager{
		envMap:         make(map[string]string),
		pathMap:        make(map[string]struct{}),
		deletedPathMap: make(map[string]struct{}),
		deletedEnvMap:  make(map[string]struct{}),
	}
	return manager, nil
}

func (p *Paths) String() string {
	return strings.Join(p.Slice(), ":")
}
