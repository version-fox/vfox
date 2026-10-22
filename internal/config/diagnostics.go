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
	"strings"

	"gopkg.in/yaml.v3"
)

// Diagnostics controls the checks vfox runs on its own, as opposed to the ones
// run by `vfox doctor`.
type Diagnostics struct {
	// OnUse runs the fast checks for the selected SDK after `vfox use` and
	// prints the findings that deserve attention.
	OnUse bool `yaml:"onUse"`
	// Ignore silences findings. Each entry is <finding-id>[:<sdk>[@<version>]],
	// for example "sdk.drift:flutter@3.47.0" or "path.shadowed:nodejs".
	Ignore []string `yaml:"ignore"`
}

var EmptyDiagnostics = &Diagnostics{
	OnUse: true,
}

// UnmarshalYAML keeps the defaults for keys the file does not mention, so that
// a file which only sets "ignore" does not switch the checks off.
func (d *Diagnostics) UnmarshalYAML(value *yaml.Node) error {
	type plain Diagnostics
	decoded := plain(*EmptyDiagnostics)
	if err := value.Decode(&decoded); err != nil {
		return err
	}
	*d = Diagnostics(decoded)
	return nil
}

// Ignores reports whether a finding is silenced by the ignore list.
func (d *Diagnostics) Ignores(id, sdk, version string) bool {
	if d == nil {
		return false
	}
	for _, entry := range d.Ignore {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		entryID, scope, hasScope := strings.Cut(entry, ":")
		if entryID != id {
			continue
		}
		if !hasScope || scope == "" {
			return true
		}
		entrySDK, entryVersion, hasVersion := strings.Cut(scope, "@")
		if !strings.EqualFold(entrySDK, sdk) {
			continue
		}
		if !hasVersion || entryVersion == "" || entryVersion == version {
			return true
		}
	}
	return false
}
