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

package env

import "github.com/version-fox/vfox/internal/pathmeta"

// chainItem represents a single config in the chain with its scope
type chainItem struct {
	config *pathmeta.VfoxToml
	scope  UseScope
}

// VfoxTomlChain is a chain of VfoxToml configs, supporting multi-config merging
type VfoxTomlChain []*chainItem

// NewVfoxTomlChain creates a new config chain
func NewVfoxTomlChain() VfoxTomlChain {
	return make(VfoxTomlChain, 0, 3)
}

// Add appends a config to the end of the chain
func (c *VfoxTomlChain) Add(config *pathmeta.VfoxToml, useScope UseScope) {
	*c = append(*c, &chainItem{
		config: config,
		scope:  useScope,
	})
}

// Merge merges configs by priority
// Later configs in the chain override earlier ones (if the same tool exists)
// Example: [Global, Session, Project] → Project has highest priority
func (c *VfoxTomlChain) Merge() *pathmeta.VfoxToml {
	if len(*c) == 0 {
		return pathmeta.NewVfoxToml()
	}

	if len(*c) == 1 {
		return (*c)[0].config
	}

	result := pathmeta.NewVfoxToml()

	// Merge in order, later overrides earlier
	for _, config := range *c {
		if config == nil || config.config == nil {
			continue
		}

		for name, toolConfig := range config.config.Tools {
			if toolConfig == nil {
				continue
			}
			// Copy the tool config to result
			result.Tools.SetWithAttr(name, toolConfig.Version, toolConfig.Attr)
		}
	}

	return result
}

// GetAllTools returns all tools after merging
func (c *VfoxTomlChain) GetAllTools() pathmeta.ToolVersions {
	return c.Merge().GetAllTools()
}

// GetToolConfig retrieves a tool config (searches by priority)
// Searches from tail to head (high priority to low priority)
func (c *VfoxTomlChain) GetToolConfig(name string) (*pathmeta.ToolConfig, UseScope, bool) {
	// Search from high priority to low priority
	for i := len(*c) - 1; i >= 0; i-- {
		item := (*c)[i]
		if item == nil || item.config == nil {
			continue
		}
		if config, ok := item.config.Tools.Get(name); ok {
			return config, item.scope, true
		}
	}
	return nil, Global, false
}

// GetToolVersion retrieves only the version of a tool (searches by priority)
func (c *VfoxTomlChain) GetToolVersion(name string) (string, UseScope, bool) {
	if config, scope, ok := c.GetToolConfig(name); ok {
		return config.Version, scope, true
	}
	return "", Global, false
}

// GetByIndex returns the config at the specified index
func (c *VfoxTomlChain) GetByIndex(index int) *pathmeta.VfoxToml {
	if index < 0 || index >= len(*c) {
		return nil
	}
	return (*c)[index].config
}

// GetTomlByScope returns the config and scope for the given scope
func (c *VfoxTomlChain) GetTomlByScope(scope UseScope) (*pathmeta.VfoxToml, bool) {
	for _, item := range *c {
		if item == nil || item.config == nil {
			continue
		}
		if item.scope == scope {
			return item.config, true
		}
	}
	return nil, false
}

// Length returns the length of the config chain
func (c *VfoxTomlChain) Length() int {
	return len(*c)
}

// IsEmpty checks if the chain is empty
func (c *VfoxTomlChain) IsEmpty() bool {
	return len(*c) == 0
}

// Save saves all configs in the chain
// Calls Save() on each VfoxToml (which won't create files if empty)
func (c *VfoxTomlChain) Save() error {
	for _, config := range *c {
		if config == nil || config.config == nil {
			continue
		}
		if err := config.config.Save(); err != nil {
			return err
		}
	}
	return nil
}

// AddTool adds a tool to all configs in the chain
func (c *VfoxTomlChain) AddTool(name, version string) {
	for _, config := range *c {
		if config == nil || config.config == nil {
			continue
		}
		config.config.SetTool(name, version)
	}
}

// RemoveTool removes a tool from all configs in the chain
func (c *VfoxTomlChain) RemoveTool(name string) {
	for _, config := range *c {
		if config == nil || config.config == nil {
			continue
		}
		config.config.RemoveTool(name)
	}
}
