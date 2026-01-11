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
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
)

// Attr is a type alias for map[string]string, used for tool attributes
type Attr map[string]string

// ToolVersions is a type alias for map of tool name to version
type ToolVersions map[string]string

// ToolConfig represents a tool configuration with version and optional attributes
// Supports two formats:
// 1. Simple: nodejs = "21.5.1"
// 2. Complex: java = { version = "21", vendor = "openjdk" }
type ToolConfig struct {
	Version string
	Attr    Attr // Additional attributes (e.g., vendor, dist)
}

// UnmarshalTOML custom unmarshaling to support both simple and complex formats
func (t *ToolConfig) UnmarshalTOML(data interface{}) error {
	t.Attr = make(Attr)

	switch v := data.(type) {
	case string:
		// Simple format: nodejs = "21.5.1"
		t.Version = v
	case map[string]interface{}:
		// Complex format: java = { version = "21", vendor = "openjdk" }
		for key, val := range v {
			switch key {
			case "version":
				if version, ok := val.(string); ok {
					t.Version = version
				}
			default:
				// Convert other attributes to strings
				t.Attr[key] = fmt.Sprintf("%v", val)
			}
		}

		// If no explicit version field, check Attr
		if t.Version == "" {
			if version, ok := t.Attr["version"]; ok {
				t.Version = version
				delete(t.Attr, "version") // Avoid duplication
			} else {
				t.Version = "unknown"
			}
		}
	default:
		return fmt.Errorf("invalid tool config format: %T", data)
	}

	return nil
}

// Tools is a map of tool name to tool configuration
type Tools map[string]*ToolConfig

// UnmarshalTOML implements the toml.Unmarshaler interface
func (t *Tools) UnmarshalTOML(data interface{}) error {
	*t = make(Tools)

	switch v := data.(type) {
	case map[string]interface{}:
		for key, val := range v {
			config := &ToolConfig{}
			if err := config.UnmarshalTOML(val); err != nil {
				return fmt.Errorf("failed to unmarshal tool %s: %w", key, err)
			}
			(*t)[key] = config
		}
	default:
		return fmt.Errorf("invalid tools format: %T", data)
	}

	return nil
}

// MarshalTOML implements the toml.Marshaler interface
func (t *Tools) MarshalTOML() ([]byte, error) {
	if *t == nil {
		*t = make(Tools)
	}

	lines := []string{"[tools]"}

	for _, name := range t.SortedKeys() {
		tool := (*t)[name]
		lines = append(lines, fmt.Sprintf("%s = %s", name, tool.MarshalInline()))
	}

	return []byte(strings.Join(lines, "\n") + "\n"), nil
}

// Set adds or updates a tool configuration (simple version only)
func (t *Tools) Set(name, version string) {
	if *t == nil {
		*t = make(Tools)
	}
	(*t)[name] = &ToolConfig{
		Version: version,
		Attr:    make(Attr),
	}
}

// SetWithAttr adds or updates a tool configuration with attributes
func (t *Tools) SetWithAttr(name, version string, attr Attr) {
	if *t == nil {
		*t = make(Tools)
	}
	config := &ToolConfig{
		Version: version,
		Attr:    make(Attr),
	}
	for k, v := range attr {
		config.Attr[k] = v
	}
	(*t)[name] = config
}

// Get retrieves a tool configuration
func (t *Tools) Get(name string) (*ToolConfig, bool) {
	if *t == nil {
		return nil, false
	}
	config, ok := (*t)[name]
	if !ok || config == nil {
		return nil, false
	}
	return config, true
}

// GetVersion retrieves only the version of a tool
func (t *Tools) GetVersion(name string) (string, bool) {
	config, ok := t.Get(name)
	if !ok {
		return "", false
	}
	return config.Version, true
}

// Remove removes a tool from the collection
func (t *Tools) Remove(name string) {
	if *t != nil {
		delete(*t, name)
	}
}

// List returns all tools as a map of name -> version
func (t *Tools) List() ToolVersions {
	if *t == nil {
		return make(ToolVersions)
	}

	result := make(ToolVersions, len(*t))
	for name, config := range *t {
		result[name] = config.Version
	}
	return result
}

// SortedKeys returns a sorted list of tool names
func (t *Tools) SortedKeys() []string {
	if *t == nil {
		return []string{}
	}

	names := make([]string, 0, len(*t))
	for name := range *t {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Len returns the number of tools
func (t *Tools) Len() int {
	if *t == nil {
		return 0
	}
	return len(*t)
}

// VfoxToml represents the vfox.toml configuration file
// Example:
//
//	[tools]
//	nodejs = "21.5.1"
//	java = { version = "21", vendor = "openjdk" }
type VfoxToml struct {
	Tools Tools  `toml:"tools"`
	Path  string // Config file path (empty for new configs)
}

// NewVfoxToml creates a new empty VfoxToml instance
func NewVfoxToml() *VfoxToml {
	return &VfoxToml{
		Tools: make(Tools),
		Path:  "",
	}
}

// IsNew checks if this is a new config (never saved before)
func (v *VfoxToml) IsNew() bool {
	return v.Path == ""
}

// IsEmpty checks if the config has no tools
func (v *VfoxToml) IsEmpty() bool {
	return v.Tools.Len() == 0
}

// Save saves the config to the recorded Path
// Returns error if Path is empty or config is empty
func (v *VfoxToml) Save() error {
	if v.Path == "" {
		return fmt.Errorf("cannot save: path is empty, use SaveTo(dir) or SaveToPath(path) instead")
	}
	return v.SaveToPath(v.Path)
}

// SaveTo saves the config to the specified directory (auto-selects filename)
// Prefers .vfox.toml, uses vfox.toml if it already exists
// Does not create a file if the config is empty
func (v *VfoxToml) SaveTo(dir string) error {
	path := DetermineConfigPath(dir)
	return v.SaveToPath(path)
}

// SaveToPath saves the config to the specified full path
// If the config is empty:
//   - If file doesn't exist: do nothing (don't create file)
//   - If file exists: update it with empty [tools] section
func (v *VfoxToml) SaveToPath(path string) error {
	// If config is empty
	if v.IsEmpty() {
		// Check if file exists
		if _, err := os.Stat(path); os.IsNotExist(err) {
			// File doesn't exist, nothing to do
			return nil
		}
		// File exists, need to update it with empty config
	}

	data, err := v.MarshalTOML()
	if err != nil {
		return err
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write vfox.toml: %w", err)
	}

	v.Path = path
	return nil
}

// LoadVfoxToml loads vfox.toml from the specified path
// Returns an empty VfoxToml if the file doesn't exist
func LoadVfoxToml(path string) (*VfoxToml, error) {
	config := NewVfoxToml()
	config.Path = path // Always set path so Save() knows where to save

	if _, err := os.Stat(path); os.IsNotExist(err) {
		// File doesn't exist, return empty config with path set
		return config, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read vfox.toml: %w", err)
	}

	if _, err := toml.Decode(string(data), config); err != nil {
		return nil, fmt.Errorf("failed to parse vfox.toml: %w", err)
	}

	return config, nil
}

// MarshalTOML serializes the configuration to TOML format
func (v *VfoxToml) MarshalTOML() ([]byte, error) {
	return v.Tools.MarshalTOML()
}

// SetTool sets or updates a tool configuration (simple version only)
func (v *VfoxToml) SetTool(name, version string) {
	v.Tools.Set(name, version)
}

// SetToolWithAttr sets or updates a tool configuration with attributes
func (v *VfoxToml) SetToolWithAttr(name, version string, attr Attr) {
	v.Tools.SetWithAttr(name, version, attr)
}

// GetToolVersion retrieves only the version of a tool
func (v *VfoxToml) GetToolVersion(name string) (string, bool) {
	return v.Tools.GetVersion(name)
}

// GetToolConfig retrieves the complete tool configuration (including attributes)
func (v *VfoxToml) GetToolConfig(name string) (*ToolConfig, bool) {
	return v.Tools.Get(name)
}

// RemoveTool removes a tool from the configuration
func (v *VfoxToml) RemoveTool(name string) {
	v.Tools.Remove(name)
}

// GetAllTools returns all tools as a ToolVersions map (name -> version)
func (v *VfoxToml) GetAllTools() ToolVersions {
	return v.Tools.List()
}

// MarshalInline serializes the tool configuration to an inline TOML value
// Returns just the value part (without the key), e.g.:
//   - "21.5.1" (for simple format)
//   - {version = "21", vendor = "openjdk"} (for complex format)
func (t *ToolConfig) MarshalInline() string {
	if len(t.Attr) == 0 {
		// Simple format: "21.5.1"
		return strconv.Quote(t.Version)
	}

	// Complex format: {version = "21", vendor = "openjdk"}
	attrKeys := make([]string, 0, len(t.Attr))
	for key := range t.Attr {
		attrKeys = append(attrKeys, key)
	}
	sort.Strings(attrKeys)

	parts := make([]string, 0, len(attrKeys)+1)
	parts = append(parts, fmt.Sprintf("version = %s", strconv.Quote(t.Version)))
	for _, key := range attrKeys {
		parts = append(parts, fmt.Sprintf("%s = %s", key, formatAttributeValue(t.Attr[key])))
	}

	return fmt.Sprintf("{%s}", strings.Join(parts, ", "))
}

// formatAttributeValue formats an attribute value based on its type
func formatAttributeValue(value string) string {
	// Check if it's a boolean
	if value == "true" || value == "false" {
		return value
	}

	// Check if it's an integer
	if _, err := strconv.ParseInt(value, 10, 64); err == nil {
		return value
	}

	// Check if it's a float
	if _, err := strconv.ParseFloat(value, 64); err == nil {
		return value
	}

	// Default: quote as string
	return strconv.Quote(value)
}
