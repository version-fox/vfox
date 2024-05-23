/*
 *    Copyright 2024 Han Li and contributors
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

package toolset

import (
	"fmt"
	"path/filepath"
)

const ToolVersionFilename = ".tool-versions"

type MultiToolVersions []ToolVersion

// FilterTools filters tools by the given filter function
// and return the first one you find.
func (m MultiToolVersions) FilterTools(filter func(name, version string) bool) map[string]string {
	tools := make(map[string]string)
	for _, t := range m {
		_ = t.ForEach(func(name string, version string) error {
			_, ok := tools[name]
			if !ok && filter(name, version) {
				tools[name] = version
			}
			return nil
		})
	}
	return tools
}

func (m MultiToolVersions) Add(name, version string) {
	for _, t := range m {
		t.Set(name, version)
	}
}

func (m MultiToolVersions) Save() error {
	for _, t := range m {
		if err := t.Save(); err != nil {
			return err
		}
	}
	return nil
}

// ToolVersion represents a .tool-versions file
type ToolVersion = *FileRecord

func NewToolVersion(dirPath string) (ToolVersion, error) {
	file := filepath.Join(dirPath, ToolVersionFilename)
	mapFile, err := NewFileRecord(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read tool versions file %s: %w", file, err)
	}
	return mapFile, nil
}

func NewMultiToolVersions(paths []string) (MultiToolVersions, error) {
	var tools MultiToolVersions
	for _, p := range paths {
		tool, err := NewToolVersion(p)
		if err != nil {
			return nil, err
		}
		tools = append(tools, tool)
	}
	return tools, nil
}
