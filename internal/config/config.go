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
	"fmt"
	"os"
	"path/filepath"

	"github.com/version-fox/vfox/internal/shared/util"
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

// Merge merges two configs with userConfig taking precedence over sharedConfig
// Returns a new config where userConfig fields override sharedConfig fields
// If a field in userConfig is nil/empty, the sharedConfig value is used
func Merge(sharedConfig, userConfig *Config) *Config {
	if sharedConfig == nil && userConfig == nil {
		return DefaultConfig
	}
	if sharedConfig == nil {
		return ensureDefaults(userConfig)
	}
	if userConfig == nil {
		return ensureDefaults(sharedConfig)
	}

	result := &Config{}

	// Merge Proxy: user overrides shared
	// Note: userConfig.Proxy may be nil, which means "use shared"
	result.Proxy = mergeProxy(sharedConfig.Proxy, userConfig.Proxy)

	// Merge Storage: user overrides shared
	result.Storage = mergeStorage(sharedConfig.Storage, userConfig.Storage)

	// Merge Registry: user overrides shared
	result.Registry = mergeRegistry(sharedConfig.Registry, userConfig.Registry)

	// Merge LegacyVersionFile: user overrides shared
	result.LegacyVersionFile = mergeLegacyVersionFile(sharedConfig.LegacyVersionFile, userConfig.LegacyVersionFile)

	// Merge Cache: user overrides shared
	result.Cache = mergeCache(sharedConfig.Cache, userConfig.Cache)

	// Apply defaults to any remaining nil fields
	return ensureDefaults(result)
}

// ensureDefaults ensures that all nil fields are replaced with empty defaults
func ensureDefaults(c *Config) *Config {
	if c == nil {
		return DefaultConfig
	}
	if c.Proxy == nil {
		c.Proxy = EmptyProxy
	}
	if c.Storage == nil {
		c.Storage = EmptyStorage
	}
	if c.Registry == nil {
		c.Registry = EmptyRegistry
	}
	if c.LegacyVersionFile == nil {
		c.LegacyVersionFile = EmptyLegacyVersionFile
	}
	if c.Cache == nil {
		c.Cache = EmptyCache
	}
	return c
}

// mergeProxy merges proxy configs with user taking precedence
// If user config is not nil, it means user explicitly set proxy settings
// (even if just to disable it), so use user config
func mergeProxy(shared, user *Proxy) *Proxy {
	if user != nil {
		return user
	}
	if shared != nil {
		return shared
	}
	return EmptyProxy
}

// mergeStorage merges storage configs with user taking precedence
func mergeStorage(shared, user *Storage) *Storage {
	if user != nil && !isStorageEmpty(user) {
		return user
	}
	if shared != nil {
		return shared
	}
	return EmptyStorage
}

// mergeRegistry merges registry configs with user taking precedence
func mergeRegistry(shared, user *Registry) *Registry {
	if user != nil && !isRegistryEmpty(user) {
		return user
	}
	if shared != nil {
		return shared
	}
	return EmptyRegistry
}

// mergeLegacyVersionFile merges legacy version file configs with user taking precedence
func mergeLegacyVersionFile(shared, user *LegacyVersionFile) *LegacyVersionFile {
	if user != nil && !isLegacyVersionFileEmpty(user) {
		return user
	}
	if shared != nil {
		return shared
	}
	return EmptyLegacyVersionFile
}

// mergeCache merges cache configs with user taking precedence
// Special case: EmptyCache (12h default) is considered "unset" if shared has a different value
func mergeCache(shared, user *Cache) *Cache {
	if user == nil {
		if shared != nil {
			return shared
		}
		return EmptyCache
	}
	// User config exists, check if it's the default EmptyCache
	if shared != nil && user.AvailableHookDuration == EmptyCache.AvailableHookDuration {
		// User has default cache duration, but shared has something different, use shared
		if shared.AvailableHookDuration != EmptyCache.AvailableHookDuration {
			return shared
		}
	}
	// User has a non-default cache setting, or both are default
	return user
}

// Helper functions to check if config is empty
// A config is considered empty if it's nil or all fields are at default/zero values
func isProxyEmpty(p *Proxy) bool {
	if p == nil {
		return true
	}
	// If proxy is explicitly disabled with no other settings, it's not empty
	// (user wants to disable proxy)
	if !p.Enable && p.Url == "" {
		return true
	}
	return false
}

func isStorageEmpty(s *Storage) bool {
	return s == nil || s.SdkPath == ""
}

func isRegistryEmpty(r *Registry) bool {
	return r == nil || r.Address == ""
}

func isLegacyVersionFileEmpty(l *LegacyVersionFile) bool {
	return l == nil
}

func isCacheEmpty(c *Cache) bool {
	return c == nil || c.AvailableHookDuration == 0
}

// LoadConfigWithFallback loads config from multiple paths with priority
// Priority: sharedPath (higher) > userPath (lower)
// If sharedPath doesn't exist or is empty, userConfig is used
// If both exist, they are merged with userConfig taking precedence for non-empty fields
func LoadConfigWithFallback(sharedPath, userPath string) (*Config, error) {
	var sharedConfig, userConfig *Config
	var err error

	// Load shared config (optional)
	if sharedPath != "" && util.FileExists(sharedPath) {
		sharedConfig, err = NewConfigWithPath(sharedPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load shared config from %s: %w", sharedPath, err)
		}
	}

	// Load user config (required - created if missing)
	userConfig, err = NewConfigWithPath(userPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load user config from %s: %w", userPath, err)
	}

	// Merge configs (user overrides shared)
	return Merge(sharedConfig, userConfig), nil
}
