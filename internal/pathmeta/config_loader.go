/*
 *
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
 *
 */

package pathmeta

import (
	"fmt"
	"path/filepath"

	"github.com/version-fox/vfox/internal/shared/logger"
	"github.com/version-fox/vfox/internal/shared/util"
)

// ConfigFileNames defines the config file names and their priority
var ConfigFileNames = []string{
	".vfox.toml", // priority 1
	"vfox.toml",  // priority 2
}

// LoadConfig loads the config file from the specified directory
// Priority: .vfox.toml > vfox.toml > .tool-versions
// If .tool-versions is read, it will be automatically saved as .vfox.toml (without deleting the original file)
// pathmeta is not aware of scope, the caller should set it
func LoadConfig(dir string) (*VfoxToml, error) {
	config, err := loadConfigInternal(dir)
	if err != nil {
		return nil, err
	}

	// If .tool-versions was read, automatically migrate to .vfox.toml (silent)
	if isToolVersionsFile(config.Path) {
		if err := migrateToVfoxToml(config, dir); err != nil {
			logger.Debugf("Failed to migrate .tool-versions to .vfox.toml: %v", err)
			// Migration failure doesn't block the flow, return in-memory config
		}
	}

	return config, nil
}

// loadConfigInternal internal loading logic
func loadConfigInternal(dir string) (*VfoxToml, error) {
	// 1. Try to load TOML config
	for _, filename := range ConfigFileNames {
		tomlPath := filepath.Join(dir, filename)
		if util.FileExists(tomlPath) {
			config, err := LoadVfoxToml(tomlPath)
			if err != nil {
				return nil, fmt.Errorf("failed to load %s: %w", filename, err)
			}

			// Path is already set by LoadVfoxToml
			// Scope is not tracked by pathmeta

			// If .tool-versions also exists, log a warning
			toolVersionsPath := filepath.Join(dir, ".tool-versions")
			if util.FileExists(toolVersionsPath) {
				logger.Debugf("Both %s and .tool-versions exist, using %s (you can delete .tool-versions)",
					filename, filename)
			}

			return config, nil
		}
	}

	// 2. TOML doesn't exist, try to read .tool-versions
	toolVersionsPath := filepath.Join(dir, ".tool-versions")
	if util.FileExists(toolVersionsPath) {
		config, err := loadFromToolVersions(toolVersionsPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load .tool-versions: %w", err)
		}

		config.Path = toolVersionsPath
		// Scope is not tracked by pathmeta

		return config, nil
	}

	// 3. Neither exists, return empty config with default path
	config := NewVfoxToml()
	config.Path = DetermineConfigPath(dir) // Set path so Save() knows where to save

	return config, nil
}

// loadFromToolVersions loads and converts .tool-versions to VfoxToml
func loadFromToolVersions(path string) (*VfoxToml, error) {
	fileRecord, err := NewFileRecord(path)
	if err != nil {
		return nil, err
	}

	config := NewVfoxToml()
	for name, version := range fileRecord.Record {
		config.Tools.Set(name, version)
	}

	return config, nil
}

// isToolVersionsFile checks if the file is .tool-versions
func isToolVersionsFile(path string) bool {
	return filepath.Base(path) == ".tool-versions"
}

// migrateToVfoxToml migrates the config to .vfox.toml
// After reading .tool-versions, it's automatically saved as .vfox.toml (without deleting the original file)
func migrateToVfoxToml(config *VfoxToml, dir string) error {
	if !isToolVersionsFile(config.Path) {
		return nil
	}

	// Prefer .vfox.toml
	tomlPath := filepath.Join(dir, ".vfox.toml")

	// Save as .vfox.toml
	if err := config.SaveToPath(tomlPath); err != nil {
		return err
	}

	// Don't delete .tool-versions, let users decide

	return nil
}

// DetermineConfigPath determines the default path for the config file
// Used for path inference when saving new configs
func DetermineConfigPath(dir string) string {
	// Prefer .vfox.toml
	dotVfoxToml := filepath.Join(dir, ".vfox.toml")
	if util.FileExists(dotVfoxToml) {
		return dotVfoxToml
	}

	// If vfox.toml exists, use it
	vfoxToml := filepath.Join(dir, "vfox.toml")
	if util.FileExists(vfoxToml) {
		return vfoxToml
	}

	// Default to .vfox.toml
	return dotVfoxToml
}
