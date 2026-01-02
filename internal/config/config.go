/*
 *    Copyright 2026 Han Li and contributors
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
	"os"
	"path/filepath"

	"github.com/version-fox/vfox/internal/util"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Proxy             *Proxy             `yaml:"proxy"`
	Storage           *Storage           `yaml:"storage"`
	Registry          *Registry          `yaml:"registry"`
	LegacyVersionFile *LegacyVersionFile `yaml:"legacyVersionFile"`
	Cache             *Cache             `yaml:"cache"`
}

const filename = "config.yaml"

var (
	DefaultConfig = &Config{
		Proxy:             EmptyProxy,
		Storage:           EmptyStorage,
		Registry:          EmptyRegistry,
		LegacyVersionFile: EmptyLegacyVersionFile,
		Cache:             EmptyCache,
	}
)

func NewConfigWithPath(p string) (*Config, error) {
	if !util.FileExists(p) {
		content, err := yaml.Marshal(DefaultConfig)
		if err == nil {
			_ = os.WriteFile(p, content, 0666)
			return DefaultConfig, nil
		}
	}
	_ = util.ChangeModeIfNot(p, 0666)
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
	if config.LegacyVersionFile == nil {
		config.LegacyVersionFile = EmptyLegacyVersionFile
	}
	if config.Cache == nil {
		config.Cache = EmptyCache
	}
	return config, nil

}

func NewConfig(path string) (*Config, error) {
	p := filepath.Join(path, filename)
	return NewConfigWithPath(p)
}

func (c *Config) SaveConfig(path string) error {
	p := filepath.Join(path, filename)
	content, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(p, content, os.ModePerm)
}
