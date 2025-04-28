/*
 *    Copyright 2025 Han Li and contributors
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

package internal

import (
	_ "embed"
	"errors"
	"fmt"
	"regexp"

	"github.com/pterm/pterm"
	"github.com/version-fox/vfox/internal/env"
)

const (
	luaPluginObjKey = "PLUGIN"
	osType          = "OS_TYPE"
	archType        = "ARCH_TYPE"
	runtime         = "RUNTIME"
)

type HookFunc struct {
	Name     string
	Required bool
	Filename string
}

var (
	// HookFuncMap is a map of built-in hook functions.
	HookFuncMap = map[string]HookFunc{
		"Available":       {Name: "Available", Required: true, Filename: "available"},
		"PreInstall":      {Name: "PreInstall", Required: true, Filename: "pre_install"},
		"EnvKeys":         {Name: "EnvKeys", Required: true, Filename: "env_keys"},
		"PostInstall":     {Name: "PostInstall", Required: false, Filename: "post_install"},
		"PreUse":          {Name: "PreUse", Required: false, Filename: "pre_use"},
		"ParseLegacyFile": {Name: "ParseLegacyFile", Required: false, Filename: "parse_legacy_file"},
		"PreUninstall":    {Name: "PreUninstall", Required: false, Filename: "pre_uninstall"},
	}
)

type Plugin struct {
	Name string
	Path string

	*PluginInfo

	Available       func(args []string) ([]*Package, error)
	PreInstall      func(version Version) (*Package, error)
	PostInstall     func(rootPath string, sdks []*Info) error
	PreUninstall    func(p *Package) error
	PreUse          func(version Version, previousVersion Version, scope UseScope, cwd string, installedSdks []*Package) (Version, error)
	EnvKeys         func(sdkPackage *Package) (*env.Envs, error)
	Label           func(version string) string
	ParseLegacyFile func(path string, installedVersions func() []Version) (Version, error)
	Close           func()
}

// ShowNotes prints the notes of the plugin.
func (l *Plugin) ShowNotes() {
	// print some notes if there are
	if len(l.Notes) != 0 {
		fmt.Println(pterm.Yellow("Notes:"))
		fmt.Println("======")
		for _, note := range l.Notes {
			fmt.Println("  ", note)
		}
	}
}

var ErrPluginNotFound = errors.New("plugin not found")

func CreatePluginFromPath(tempInstallPath string, manager *Manager) (*Plugin, error) {
	if IsLuaPluginDir(tempInstallPath) {
		luaPlugin, err := NewLuaPlugin(tempInstallPath, manager)
		if err != nil {
			return nil, err
		}
		return FromLuaPlugin(luaPlugin), nil
	}

	return nil, ErrPluginNotFound
}

func FromLuaPlugin(source *LuaPlugin) *Plugin {
	result := &Plugin{
		Name:       source.Name,
		Path:       source.Path,
		PluginInfo: source.PluginInfo,

		Available:       source.Available,
		PreUse:          source.PreUse,
		PreUninstall:    source.PreUninstall,
		ParseLegacyFile: source.ParseLegacyFile,
		PreInstall:      source.PreInstall,
		PostInstall:     source.PostInstall,
		EnvKeys:         source.EnvKeys,
		Label:           source.Label,
		Close:           source.Close,
	}
	return result
}

func isValidName(name string) bool {
	// The regular expression means: start with a letter,
	// followed by any number of letters, digits, underscores, or hyphens.
	re := regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_\-]*$`)
	return re.MatchString(name)
}
