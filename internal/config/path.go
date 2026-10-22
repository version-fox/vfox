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

// How vfox treats the PATH entries that sit in front of its own directories.
const (
	// PathModeLegacy keeps every entry before vfox's first directory in front
	// of the vfox directories. This is the behavior of vfox 1.0.x.
	PathModeLegacy = "legacy"
	// PathModeBaseline keeps only the entries that were added after
	// `vfox activate` ran in this shell, such as a virtualenv activated later.
	// Entries that were already in PATH at activation time (the system
	// directories that macOS path_helper or the Windows machine PATH put in
	// front) go behind the vfox directories.
	PathModeBaseline = "baseline"
	// PathModeAggressive keeps only active virtualenv/conda directories and the
	// directories listed in Preserve in front of the vfox directories.
	PathModeAggressive = "aggressive"
)

// Path configures the order of PATH entries around vfox's own directories.
type Path struct {
	Mode string `yaml:"mode"`
	// Preserve lists directories that always stay in front of the vfox
	// directories. Environment variables and a leading ~ are expanded.
	Preserve []string `yaml:"preserve"`
}

var EmptyPath = &Path{
	Mode: PathModeLegacy,
}

// UnmarshalYAML keeps the default mode when the file only sets "preserve".
func (p *Path) UnmarshalYAML(value *yaml.Node) error {
	type plain Path
	decoded := plain(*EmptyPath)
	if err := value.Decode(&decoded); err != nil {
		return err
	}
	*p = Path(decoded)
	return nil
}

// EffectiveMode returns the configured mode, falling back to PathModeLegacy
// for an empty or unknown value.
func (p *Path) EffectiveMode() string {
	if p == nil {
		return PathModeLegacy
	}
	switch mode := strings.ToLower(strings.TrimSpace(p.Mode)); mode {
	case PathModeBaseline, PathModeAggressive:
		return mode
	default:
		return PathModeLegacy
	}
}
