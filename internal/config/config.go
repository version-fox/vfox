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

package config

import (
	"github.com/version-fox/vfox/internal/logger"
	"github.com/version-fox/vfox/internal/util"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
)

type Config struct {
	Proxy    *Proxy    `yaml:"proxy"`
	Storage  *Storage  `yaml:"storage"`
	Registry *Registry `yaml:"registry"`
}

const filename = "config.yaml"
const VFoxPath = ".version-fox"

var (
	defaultConfig = &Config{
		Proxy:    EmptyProxy,
		Storage:  EmptyStorage,
		Registry: EmptyRegistry,
	}
)

func NewConfigWithPath(p string) (*Config, error) {
	if !util.FileExists(p) {
		content, err := yaml.Marshal(defaultConfig)
		if err == nil {
			_ = os.WriteFile(p, content, 0644)
			return defaultConfig, nil
		}
	}
	content, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	config := &Config{}
	err = yaml.Unmarshal(content, config)
	if err != nil {
		return nil, err
	}
	if config.Proxy == nil {
		config.Proxy = EmptyProxy
	}
	if config.Storage == nil {
		config.Storage = EmptyStorage
	}
	if config.Registry == nil {
		config.Registry = EmptyRegistry
	}
	return config, nil

}

func NewConfig(path string) (*Config, error) {
	p := filepath.Join(path, filename)
	return NewConfigWithPath(p)
}

func GetHomePath() string {
	userHomeDir, _ := os.UserHomeDir()
	homePath := filepath.Join(userHomeDir, VFoxPath)
	return homePath
}

// SaveConfig save config to config.yaml
func SaveConfig(config *Config) bool {
	content, err := yaml.Marshal(config)
	if err != nil {
		return false
	}

	err = util.FileSave(filepath.Join(GetHomePath(), filename), content)
	logger.Info(string(content))
	if err != nil {
		return false
	}
	return true
}

func GetConfig(config *Config) string {
	content, _ := yaml.Marshal(config)
	return string(content)
}
