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

package internal

import (
	"fmt"
	"sort"
	"strings"

	"github.com/version-fox/vfox/internal/config"
	"github.com/version-fox/vfox/internal/sdk"
	"github.com/version-fox/vfox/internal/shared/logger"
)

// Aliases are names the user gives to installed versions. They are resolved for
// what is typed on the command line and nowhere else: version files such as
// .vfox.toml always hold the real version, so they stay valid on other machines.

// AliasesFor returns the aliases of an SDK as alias -> version.
func (m *Manager) AliasesFor(sdkName string) map[string]string {
	result := make(map[string]string)
	if m.RuntimeEnvContext == nil || m.RuntimeEnvContext.UserConfig == nil {
		return result
	}
	for alias, version := range m.RuntimeEnvContext.UserConfig.Aliases[strings.ToLower(sdkName)] {
		result[alias] = version
	}
	return result
}

// ResolveAlias turns an alias into the version it names. A version that is
// installed under that very name always wins, and anything that is not an alias
// is returned unchanged, so it is safe to call on any input.
func (m *Manager) ResolveAlias(sdkName string, s sdk.Sdk, version sdk.Version) sdk.Version {
	if version == "" {
		return version
	}
	if s != nil && s.CheckRuntimeExist(version) {
		return version
	}
	if target, ok := m.AliasesFor(sdkName)[string(version)]; ok {
		logger.Debugf("Alias %s of %s resolves to %s\n", version, sdkName, target)
		return sdk.Version(target)
	}
	return version
}

// ResolveVersionName turns the version a user typed into the version to act on.
// The name as typed comes first: an installed version of that name, or an alias
// for one. Only when there is neither is a leading "v" read as the usual way to
// spell a version (v20.1.0), and the name without it gets the same two checks.
// What matches neither way is returned without the "v" and left to the caller,
// which prefix-matches or reports that it is not installed.
func (m *Manager) ResolveVersionName(sdkName string, s sdk.Sdk, typed sdk.Version) sdk.Version {
	if resolved := m.ResolveAlias(sdkName, s, typed); resolved != typed {
		return resolved
	}
	stripped := sdk.StripVersionPrefix(s, typed)
	if stripped == typed {
		return typed
	}
	return m.ResolveAlias(sdkName, s, stripped)
}

// SetAlias names an installed version. The alias is stored in the user config.
func (m *Manager) SetAlias(sdkName, alias string, version sdk.Version) error {
	sdkName = strings.ToLower(sdkName)
	s, err := m.LookupSdk(sdkName)
	if err != nil {
		return fmt.Errorf("%s not supported, error: %w", sdkName, err)
	}
	if !sdk.ValidVersionName(alias) {
		return fmt.Errorf("%q is not a usable alias: use letters, digits, '.', '_', '+' and '-', and not \"latest\"", alias)
	}
	if s.CheckRuntimeExist(sdk.Version(alias)) {
		return fmt.Errorf("%s@%s is an installed version, so it cannot also be an alias", sdkName, alias)
	}
	if !s.CheckRuntimeExist(version) {
		return fmt.Errorf("%s@%s is not installed", sdkName, version)
	}
	return m.updateAliases(func(aliases map[string]map[string]string) error {
		if aliases[sdkName] == nil {
			aliases[sdkName] = make(map[string]string)
		}
		aliases[sdkName][alias] = string(version)
		return nil
	})
}

// UnsetAlias removes an alias from the user config.
func (m *Manager) UnsetAlias(sdkName, alias string) error {
	sdkName = strings.ToLower(sdkName)
	return m.updateAliases(func(aliases map[string]map[string]string) error {
		if _, ok := aliases[sdkName][alias]; !ok {
			if _, shared := m.AliasesFor(sdkName)[alias]; shared {
				return fmt.Errorf("the alias %s of %s comes from the shared config and cannot be removed here", alias, sdkName)
			}
			return fmt.Errorf("%s has no alias %s", sdkName, alias)
		}
		delete(aliases[sdkName], alias)
		return nil
	})
}

// RemoveAliasesFor drops the user's aliases that name a version, which is
// about to go away. It returns the aliases it removed.
func (m *Manager) RemoveAliasesFor(sdkName string, version sdk.Version) ([]string, error) {
	sdkName = strings.ToLower(sdkName)
	var removed []string
	calls := 0
	err := m.updateAliases(func(aliases map[string]map[string]string) error {
		calls++
		var names []string
		for alias, target := range aliases[sdkName] {
			if target == string(version) {
				names = append(names, alias)
				delete(aliases[sdkName], alias)
			}
		}
		sort.Strings(names)
		if calls == 1 { // the first call works on the user's file; the second only mirrors it in memory
			removed = names
		}
		return nil
	})
	return removed, err
}

// updateAliases applies change to the aliases in the user's config file and
// then to the ones this process has loaded. The file is written only when
// change succeeds, and only its aliases are touched: values that come from the
// shared config stay out of it, and so does every default, which would turn a
// section the user never set into one the user declared.
func (m *Manager) updateAliases(change func(aliases map[string]map[string]string) error) error {
	file, err := config.LoadAliasFile(m.RuntimeEnvContext.PathMeta.User.Config)
	if err != nil {
		return fmt.Errorf("load user config: %w", err)
	}
	if err := change(file.Aliases); err != nil {
		return err
	}
	if err := file.Save(); err != nil {
		return fmt.Errorf("save user config: %w", err)
	}

	loaded := m.RuntimeEnvContext.UserConfig.Clone()
	if loaded.Aliases == nil {
		loaded.Aliases = make(map[string]map[string]string)
	}
	if err := change(loaded.Aliases); err != nil {
		// The loaded config also holds the shared aliases; it can legitimately
		// lack what the user file just changed.
		logger.Debugf("Could not mirror the alias change in memory: %v\n", err)
	}
	m.RuntimeEnvContext.UserConfig = loaded
	return nil
}
