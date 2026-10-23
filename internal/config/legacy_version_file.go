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

import "gopkg.in/yaml.v3"

const (
	LatestInstalledStrategy = "latest_installed"
	LatestAvailableStrategy = "latest_available"
	SpecifiedStrategy       = "specified"
	DefaultStrategy         = SpecifiedStrategy
)

// LegacyVersionFile represents whether to enable the ability to parse legacy version files,
type LegacyVersionFile struct {
	Enable bool `yaml:"enable"`
	// Support three strategies:
	// 1. latest_installed: use the latest installed version
	// 2. latest_available: use the latest available version
	// 3. specified: use the specified version in the legacy file
	// default: specified
	Strategy string `yaml:"strategy"`
}

var EmptyLegacyVersionFile = &LegacyVersionFile{
	Enable:   true,
	Strategy: DefaultStrategy,
}

// UnmarshalYAML keeps the defaults for keys the file does not mention. `vfox
// config` writes only the key it sets, so a file that holds nothing but
// "strategy" must not switch the parsing off.
func (l *LegacyVersionFile) UnmarshalYAML(value *yaml.Node) error {
	type plain LegacyVersionFile
	decoded := plain(*EmptyLegacyVersionFile)
	if err := value.Decode(&decoded); err != nil {
		return err
	}
	*l = LegacyVersionFile(decoded)
	return nil
}
