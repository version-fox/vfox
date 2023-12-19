//go:build darwin || linux

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

package env

import (
	"bufio"
	"fmt"
	"github.com/version-fox/vfox/util"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
)

const (
	BASH = ShellType("bash")
	ZSH  = ShellType("zsh")
	KSH  = ShellType("ksh")
	// extend shell type
)

type macosEnvManager struct {
	shellInfo *ShellInfo
	// ~/.version_fox/env.sh
	vfEnvPath string
	envMap    map[string]string
	// $PATH
	pathMap map[string]string
}

func (m *macosEnvManager) ReShell() error {
	// flush env to file
	m.Flush()
	command := exec.Command(m.shellInfo.ShellPath)
	command.Stdin = os.Stdin
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	if err := command.Start(); err != nil {
		return err
	}
	if err := command.Wait(); err != nil {
		return err
	}
	return nil
}

func (m *macosEnvManager) Load(kvs []*KV) error {
	for _, kv := range kvs {
		if kv.Key == "PATH" {
			m.pathMap[kv.Value] = kv.Value
		} else {
			m.envMap[kv.Key] = kv.Value
		}
	}
	return nil
}
func (m *macosEnvManager) Remove(key string) error {
	if key == "PATH" {
		return fmt.Errorf("can not remove PATH variable")
	}
	delete(m.envMap, key)
	delete(m.pathMap, key)
	return nil
}

func (m *macosEnvManager) Flush() {
	file, err := os.OpenFile(m.vfEnvPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		fmt.Printf("Failed to open the file %s, err:%s\n", m.vfEnvPath, err.Error())
		return
	}
	defer file.Close()
	for k, v := range m.envMap {
		str := fmt.Sprintf("export %s=%s\n", k, v)
		if _, err := file.WriteString(str); err != nil {
			fmt.Printf("Failed to flush env variable to file,value: err:%s\n", err.Error())
			return
		}
	}

	pathValue := fmt.Sprintf("export PATH=%s\n", m.pathEnvValue())
	if _, err := file.WriteString(pathValue); err != nil {
		fmt.Printf("Failed to flush PATH variable to file, err:%s\n", err.Error())
		return
	}
}

func (m *macosEnvManager) Get(key string) (string, error) {
	if key == "PATH" {
		return m.pathEnvValue(), nil
	} else {
		return m.envMap[key], nil
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
						m.pathMap[path] = path
					}
				} else {
					m.envMap[key] = value
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
	shellInfo, err := NewShellInfo()
	if err != nil {
		return nil, err
	}
	envPath := filepath.Join(vfConfigPath, "env.sh")
	if !util.FileExists(envPath) {
		_, _ = os.Create(envPath)
	}
	manager := &macosEnvManager{
		shellInfo: shellInfo,
		vfEnvPath: envPath,
		envMap:    make(map[string]string),
		pathMap:   make(map[string]string),
	}
	err = manager.loadEnvFile()
	if err != nil {
		fmt.Printf("Failed to load env file: %s, err:%s\n", manager.vfEnvPath, err.Error())
	}
	if err := appendEnvSourceIfNotExist(manager.shellInfo.ConfigPath, manager.vfEnvPath); err != nil {
		return nil, err
	}
	return manager, nil
}

func NewShellInfo() (*ShellInfo, error) {
	// 获取当前用户
	currentUser, err := user.Current()
	if err != nil {
		return nil, err
	}
	shellPath := os.Getenv("SHELL")
	shell := filepath.Base(shellPath)
	var info *ShellInfo
	switch ShellType(shell) {
	case BASH:
		info = &ShellInfo{
			ShellType:  BASH,
			ShellPath:  shellPath,
			ConfigPath: filepath.Join(currentUser.HomeDir, ".bashrc"),
		}
	case ZSH:
		info = &ShellInfo{
			ShellType:  ZSH,
			ShellPath:  shellPath,
			ConfigPath: filepath.Join(currentUser.HomeDir, ".zshrc"),
		}
	//case KSH:
	//	info = &ShellInfo{
	//		ShellType:  shellType,
	//		ConfigPath: filepath.Join(currentUser.HomeDir, ".kshrc"),
	//	}
	default:
		return nil, fmt.Errorf("unsupported shell type")
	}
	return info, nil

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
