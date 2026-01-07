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

package plugin

import (
	"fmt"
	"path/filepath"
	"regexp"

	"github.com/pterm/pterm"
	"github.com/version-fox/vfox/internal/env"
	"github.com/version-fox/vfox/internal/shared/util"
)

type (
	HookFunc struct {
		Name     string
		Required bool
		Filename string
	}
	// Metadata represents the metadata of a plugin.
	Metadata struct {
		Name              string   `json:"name"`
		Version           string   `json:"version"`
		Description       string   `json:"description"`
		UpdateUrl         string   `json:"updateUrl"`
		ManifestUrl       string   `json:"manifestUrl"`
		Homepage          string   `json:"homepage"`
		License           string   `json:"license"`
		MinRuntimeVersion string   `json:"minRuntimeVersion"`
		Notes             []string `json:"notes"`
		LegacyFilenames   []string `json:"legacyFilenames"`
	}

	// Plugin is the interface that all plugins must implement.
	Plugin interface {
		Available(ctx *AvailableHookCtx) ([]*AvailableHookResultItem, error)
		PreInstall(ctx *PreInstallHookCtx) (*PreInstallHookResult, error)
		PostInstall(ctx *PostInstallHookCtx) error
		PreUninstall(ctx *PreUninstallHookCtx) error
		PreUse(ctx *PreUseHookCtx) (*PreUseHookResult, error)
		ParseLegacyFile(ctx *ParseLegacyFileHookCtx) (*ParseLegacyFileResult, error)
		EnvKeys(ctx *EnvKeysHookCtx) ([]*EnvKeysHookResultItem, error)
		HasFunction(name string) bool
		Close()
	}
)

var (
	// HookFuncMap is a map of built-in hook functions.
	HookFuncMap = map[string]HookFunc{
		"Available":       {Name: "Available", Required: true, Filename: "available"},
		"PreInstall":      {Name: "PreInstall", Required: true, Filename: "pre_install"},
		"EnvKeys":         {Name: "EnvKeys", Required: true, Filename: "env_keys"},
		"PostInstall":     {Name: "PostInstall", Required: false, Filename: "post_install"},
		"preUse":          {Name: "preUse", Required: false, Filename: "pre_use"},
		"ParseLegacyFile": {Name: "ParseLegacyFile", Required: false, Filename: "parse_legacy_file"},
		"PreUninstall":    {Name: "PreUninstall", Required: false, Filename: "pre_uninstall"},
	}
)

// ShowNotes prints the notes of the plugin.
func (l *Metadata) ShowNotes() {
	// print some notes if there are
	if len(l.Notes) != 0 {
		fmt.Println(pterm.Yellow("Notes:"))
		fmt.Println("======")
		for _, note := range l.Notes {
			fmt.Println("  ", note)
		}
	}
}

func CreatePlugin(tempInstallPath string, runtimeEnvCtx *env.RuntimeEnvContext) (*Wrapper, error) {
	if isLuaPluginDir(tempInstallPath) {
		luaPlugin, err := NewLuaPlugin(tempInstallPath, runtimeEnvCtx)
		if err != nil {
			return nil, err
		}

		if err = luaPlugin.validate(); err != nil {
			return nil, err
		}

		return luaPlugin, nil
	}

	return nil, ErrPluginNotFound
}

// NewLuaPlugin creates a new LuaPlugin instance from the specified directory path.
// The plugin directory must meet one of the following conditions:
// - The directory must contain a metadata.lua file and a hooks directory that includes all must be implemented hook functions.
// - The directory contain a main.lua file that defines the plugin object and all hook functions.
func NewLuaPlugin(pluginDirPath string, ctx *env.RuntimeEnvContext) (*Wrapper, error) {
	plugin, metadata, err := CreateLuaPlugin(pluginDirPath, ctx)
	if err != nil {
		return nil, err
	}
	source := &Wrapper{
		metadata,
		plugin,
		pluginDirPath,
	}
	return source, nil
}

func isValidName(name string) bool {
	// The regular expression means: start with a letter,
	// followed by any number of letters, digits, underscores, or hyphens.
	re := regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_\-]*$`)
	return re.MatchString(name)
}

func isLuaPluginDir(pluginDirPath string) bool {
	metadataPath := filepath.Join(pluginDirPath, "metadata.lua")
	if util.FileExists(metadataPath) {
		return true
	}

	// legacy lua plugin
	hookPath := filepath.Join(pluginDirPath, "main.lua")
	if util.FileExists(hookPath) {
		return true
	}

	return false
}
