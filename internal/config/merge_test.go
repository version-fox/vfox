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
	"testing"
	"time"

	"github.com/version-fox/vfox/internal/shared/cache"
)

func TestMerge(t *testing.T) {
	tests := []struct {
		name         string
		sharedConfig *Config
		userConfig   *Config
		want         *Config
	}{
		{
			name:         "Both configs nil - return default",
			sharedConfig: nil,
			userConfig:   nil,
			want:         DefaultConfig,
		},
		{
			name: "Only shared config - return shared with defaults",
			sharedConfig: &Config{
				Proxy: &Proxy{
					Enable: true,
					Url:    "http://proxy.company.com",
				},
				Cache: &Cache{
					AvailableHookDuration: cache.Duration(24 * time.Hour),
				},
			},
			userConfig: nil,
			want: &Config{
				Proxy: &Proxy{
					Enable: true,
					Url:    "http://proxy.company.com",
				},
				Storage:           EmptyStorage,
				Registry:          EmptyRegistry,
				LegacyVersionFile: EmptyLegacyVersionFile,
				Cache: &Cache{
					AvailableHookDuration: cache.Duration(24 * time.Hour),
				},
			},
		},
		{
			name:         "Only user config - return user with defaults",
			sharedConfig: nil,
			userConfig: &Config{
				Proxy: &Proxy{
					Enable: false,
				},
			},
			want: &Config{
				Proxy:             &Proxy{Enable: false, Url: ""},
				Storage:           EmptyStorage,
				Registry:          EmptyRegistry,
				LegacyVersionFile: EmptyLegacyVersionFile,
				Cache:             EmptyCache,
			},
		},
		{
			name: "User config overrides shared config",
			sharedConfig: &Config{
				Proxy: &Proxy{
					Enable: true,
					Url:    "http://proxy.company.com",
				},
				Cache: &Cache{
					AvailableHookDuration: cache.Duration(24 * time.Hour),
				},
			},
			userConfig: &Config{
				Proxy: &Proxy{
					Enable: false,
					Url:    "", // User explicitly disables proxy with no URL
				},
				// Cache inherits from shared
			},
			want: &Config{
				Proxy: &Proxy{
					Enable: false, // User's choice - disable proxy
					Url:    "",
				},
				Storage:           EmptyStorage,
				Registry:          EmptyRegistry,
				LegacyVersionFile: EmptyLegacyVersionFile,
				Cache: &Cache{
					AvailableHookDuration: cache.Duration(24 * time.Hour), // Shared's choice
				},
			},
		},
		{
			name: "Empty user config inherits all from shared",
			sharedConfig: &Config{
				Proxy: &Proxy{
					Enable: true,
					Url:    "http://proxy.company.com",
				},
				Cache: &Cache{
					AvailableHookDuration: cache.Duration(24 * time.Hour),
				},
			},
			userConfig: &Config{
				// Empty config
				Proxy:   nil,
				Cache:   nil,
				Storage: nil,
			},
			want: &Config{
				Proxy: &Proxy{
					Enable: true,
					Url:    "http://proxy.company.com",
				},
				Storage:           EmptyStorage,
				Registry:          EmptyRegistry,
				LegacyVersionFile: EmptyLegacyVersionFile,
				Cache: &Cache{
					AvailableHookDuration: cache.Duration(24 * time.Hour),
				},
			},
		},
		{
			name: "User config with non-empty values overrides shared",
			sharedConfig: &Config{
				Proxy: &Proxy{
					Enable: true,
					Url:    "http://proxy.company.com",
				},
				Storage: &Storage{
					SdkPath: "/shared/sdk",
				},
			},
			userConfig: &Config{
				Proxy: &Proxy{
					Enable: true,
					Url:    "http://user.proxy.com",
				},
				Storage: &Storage{
					SdkPath: "/user/sdk",
				},
			},
			want: &Config{
				Proxy: &Proxy{
					Enable: true,
					Url:    "http://user.proxy.com", // User's proxy
				},
				Storage: &Storage{
					SdkPath: "/user/sdk", // User's storage
				},
				Registry:          EmptyRegistry,
				LegacyVersionFile: EmptyLegacyVersionFile,
				Cache:             EmptyCache,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Merge(tt.sharedConfig, tt.userConfig)

			// Check Proxy
			if got.Proxy == nil && tt.want.Proxy != nil {
				t.Errorf("Proxy = nil, want %v", tt.want.Proxy)
			} else if got.Proxy != nil && tt.want.Proxy != nil {
				if got.Proxy.Enable != tt.want.Proxy.Enable {
					t.Errorf("Proxy.Enable = %v, want %v", got.Proxy.Enable, tt.want.Proxy.Enable)
				}
				if got.Proxy.Url != tt.want.Proxy.Url {
					t.Errorf("Proxy.Url = %v, want %v", got.Proxy.Url, tt.want.Proxy.Url)
				}
			}

			// Check Cache
			if got.Cache == nil && tt.want.Cache != nil {
				t.Errorf("Cache = nil, want %v", tt.want.Cache)
			} else if got.Cache != nil && tt.want.Cache != nil {
				if got.Cache.AvailableHookDuration != tt.want.Cache.AvailableHookDuration {
					t.Errorf("Cache.AvailableHookDuration = %v, want %v", got.Cache.AvailableHookDuration, tt.want.Cache.AvailableHookDuration)
				}
			}

			// Check Storage
			if got.Storage == nil && tt.want.Storage != nil {
				t.Errorf("Storage = nil, want %v", tt.want.Storage)
			} else if got.Storage != nil && tt.want.Storage != nil {
				if got.Storage.SdkPath != tt.want.Storage.SdkPath {
					t.Errorf("Storage.SdkPath = %v, want %v", got.Storage.SdkPath, tt.want.Storage.SdkPath)
				}
			}
		})
	}
}

func TestLoadConfigWithFallback(t *testing.T) {
	tempDir := t.TempDir()

	// Create shared config
	sharedConfigPath := filepath.Join(tempDir, "shared_config.yaml")
	sharedConfigContent := []byte(`
proxy:
  enable: true
  url: http://proxy.company.com
cache:
  availableHookDuration: 24h
`)
	err := os.WriteFile(sharedConfigPath, sharedConfigContent, 0644)
	if err != nil {
		t.Fatalf("Failed to create shared config: %v", err)
	}

	// Create user config
	userConfigPath := filepath.Join(tempDir, "user_config.yaml")
	userConfigContent := []byte(`
proxy:
  enable: false
`)
	err = os.WriteFile(userConfigPath, userConfigContent, 0644)
	if err != nil {
		t.Fatalf("Failed to create user config: %v", err)
	}

	// Test loading with both configs
	t.Run("Load both configs - user overrides shared", func(t *testing.T) {
		config, err := LoadConfigWithFallback(sharedConfigPath, userConfigPath)
		if err != nil {
			t.Fatalf("LoadConfigWithFallback failed: %v", err)
		}

		// User's proxy choice should win
		if config.Proxy.Enable {
			t.Errorf("Expected proxy to be disabled (user's choice), got enabled")
		}

		// Shared cache should be inherited
		if config.Cache.AvailableHookDuration != cache.Duration(24*time.Hour) {
			t.Errorf("Expected cache duration 24h (shared's choice), got %v", config.Cache.AvailableHookDuration)
		}
	})

	// Test loading with only user config
	t.Run("Load only user config", func(t *testing.T) {
		_, err := LoadConfigWithFallback("", userConfigPath)
		if err != nil {
			t.Fatalf("LoadConfigWithFallback failed: %v", err)
		}
	})

	// Test loading non-existent shared config
	t.Run("Load with non-existent shared config", func(t *testing.T) {
		config, err := LoadConfigWithFallback(filepath.Join(tempDir, "nonexistent.yaml"), userConfigPath)
		if err != nil {
			t.Fatalf("LoadConfigWithFallback failed: %v", err)
		}

		// Should only have user config
		if config.Proxy.Enable {
			t.Errorf("Expected proxy to be disabled")
		}
	})
}

func TestIsEmptyHelpers(t *testing.T) {
	t.Run("isProxyEmpty", func(t *testing.T) {
		tests := []struct {
			name string
			p    *Proxy
			want bool
		}{
			{"nil proxy", nil, true},
			{"disabled proxy with empty url", &Proxy{Enable: false, Url: ""}, true},
			{"enabled proxy with url", &Proxy{Enable: true, Url: "http://proxy.com"}, false},
			{"disabled proxy with url", &Proxy{Enable: false, Url: "http://proxy.com"}, false},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if got := isProxyEmpty(tt.p); got != tt.want {
					t.Errorf("isProxyEmpty() = %v, want %v", got, tt.want)
				}
			})
		}
	})

	t.Run("isStorageEmpty", func(t *testing.T) {
		tests := []struct {
			name string
			s    *Storage
			want bool
		}{
			{"nil storage", nil, true},
			{"storage with empty path", &Storage{SdkPath: ""}, true},
			{"storage with path", &Storage{SdkPath: "/path/to/sdk"}, false},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if got := isStorageEmpty(tt.s); got != tt.want {
					t.Errorf("isStorageEmpty() = %v, want %v", got, tt.want)
				}
			})
		}
	})

	t.Run("isRegistryEmpty", func(t *testing.T) {
		tests := []struct {
			name string
			r    *Registry
			want bool
		}{
			{"nil registry", nil, true},
			{"registry with empty address", &Registry{Address: ""}, true},
			{"registry with address", &Registry{Address: "https://registry.example.com"}, false},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if got := isRegistryEmpty(tt.r); got != tt.want {
					t.Errorf("isRegistryEmpty() = %v, want %v", got, tt.want)
				}
			})
		}
	})

	t.Run("isCacheEmpty", func(t *testing.T) {
		tests := []struct {
			name string
			c    *Cache
			want bool
		}{
			{"nil cache", nil, true},
			{"cache with zero duration", &Cache{AvailableHookDuration: 0}, true},
			{"cache with duration", &Cache{AvailableHookDuration: cache.Duration(24 * time.Hour)}, false},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if got := isCacheEmpty(tt.c); got != tt.want {
					t.Errorf("isCacheEmpty() = %v, want %v", got, tt.want)
				}
			})
		}
	})
}
