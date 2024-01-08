//go:build darwin || linux

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

package env

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

type macosEnvManager struct {
	// ~/.version_fox/env.sh
	vfEnvPath string
	store     *Store
}

func (m *macosEnvManager) Paths(paths []string) string {
	oldPath := os.Getenv("PATH")
	paths = append(paths, oldPath)
	return strings.Join(paths, ":")
}

func (m *macosEnvManager) Close() error {
	return nil
}

func (m *macosEnvManager) Load(key, value string) {
	m.store.Add(&KV{
		Key:   key,
		Value: value,
	})
}
func (m *macosEnvManager) Remove(key string) error {
	if key == "PATH" {
		return fmt.Errorf("can not remove PATH variable")
	}
	m.store.Remove(key)
	return nil
}

func (m *macosEnvManager) Flush() error {
	for k, v := range m.store.envMap {
		if err := os.Setenv(k, v); err != nil {
			return err
		}
	}
	var newPaths []string
	for path := range m.store.pathMap {
		newPaths = append(newPaths, path)
	}
	oldPaths := strings.Split(os.Getenv("PATH"), ":")
	for _, path := range oldPaths {
		if strings.Contains(path, ".version-fox") {
			continue
		}
		newPaths = append(newPaths, path)
	}
	return os.Setenv("PATH", strings.Join(newPaths, ":"))
	//file, err := os.OpenFile(m.vfEnvPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	//if err != nil {
	//	fmt.Printf("Failed to open the file %s, err:%s\n", m.vfEnvPath, err.Error())
	//	return err
	//}
	//defer file.Close()
	//for k, v := range m.store.envMap {
	//	str := fmt.Sprintf("export %s=%s\n", k, v)
	//	if _, err := file.WriteString(str); err != nil {
	//		fmt.Printf("Failed to flush env variable to file,value: err:%s\n", err.Error())
	//		return err
	//	}
	//}
	//
	//pathValue := fmt.Sprintf("export PATH=%s\n", m.pathEnvValue())
	//if _, err := file.WriteString(pathValue); err != nil {
	//	fmt.Printf("Failed to flush PATH variable to file, err:%s\n", err.Error())
	//	return err
	//}
	//return nil
}

func (m *macosEnvManager) Get(key string) (string, bool) {
	if key == "PATH" {
		return m.pathEnvValue(), true
	} else {
		s, ok := m.store.envMap[key]
		return s, ok
	}
}

func (m *macosEnvManager) pathEnvValue() string {
	var pathValues []string
	for k, _ := range m.store.pathMap {
		pathValues = append(pathValues, k)
	}
	pathValues = append(pathValues, "$PATH")
	return strings.Join(pathValues, ":")
}

func (m *macosEnvManager) loadEnvFile() error {
	file, err := os.Open(m.vfEnvPath)
	if err != nil {
		return err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "export") {
			line = strings.TrimPrefix(line, "export")
			line = strings.TrimSpace(line)
			kv := strings.Split(line, "=")
			if len(kv) == 2 {
				key := kv[0]
				value := kv[1]
				if key == "PATH" {
					pathArray := strings.Split(value, ":")
					for _, path := range pathArray {
						if path == "$PATH" {
							continue
						}
						m.store.Add(&KV{
							Key:   "PATH",
							Value: path,
						})
					}
				} else {
					m.store.Add(&KV{
						Key:   key,
						Value: value,
					})
				}
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return nil
}

func NewEnvManager(vfConfigPath string) (Manager, error) {
	//envPath := filepath.Join(vfConfigPath, "env.sh")
	//if !util.FileExists(envPath) {
	//	_, _ = os.Create(envPath)
	//}
	manager := &macosEnvManager{
		//vfEnvPath: envPath,
		store: NewStore(),
	}
	//err := manager.loadEnvFile()
	//if err != nil {
	//	fmt.Printf("Failed to load env file: %s, err:%s\n", manager.vfEnvPath, err.Error())
	//}
	//if err := appendEnvSourceIfNotExist(manager.shellInfo.ConfigPath, manager.vfEnvPath); err != nil {
	//	return nil, err
	//}
	return manager, nil
}

func appendEnvSourceIfNotExist(parentEnvPath, childEnvPath string) error {
	shellConfigFile, err := os.Open(parentEnvPath)
	if err != nil {
		return err
	}
	defer shellConfigFile.Close()
	command := fmt.Sprintf("source %s", childEnvPath)
	stat, _ := os.Stat(parentEnvPath)
	if stat.Size() > 0 {
		scanner := bufio.NewScanner(shellConfigFile)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.Contains(line, command) {
				return nil
			}
		}
		if err := scanner.Err(); err != nil {
			return err
		}
	}

	file, err := os.OpenFile(parentEnvPath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.WriteString("\n" + command + "\n")
	return err
}
