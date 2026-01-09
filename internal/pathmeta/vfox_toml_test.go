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
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadVfoxToml_NotExists(t *testing.T) {
	// Test loading a non-existent file
	config, err := LoadVfoxToml("/nonexistent/path/vfox.toml")
	if err != nil {
		t.Fatalf("expected no error for non-existent file, got: %v", err)
	}

	if config == nil {
		t.Fatal("expected config to be returned")
	}

	if len(config.Tools) != 0 {
		t.Fatalf("expected empty tools map, got %d tools", len(config.Tools))
	}
}

func TestLoadVfoxToml_SimpleFormat(t *testing.T) {
	// Create a temporary TOML file
	tmpDir := t.TempDir()
	tomlPath := filepath.Join(tmpDir, "vfox.toml")

	content := `[tools]
nodejs = "21.5.1"
python = "3.11.0"
`

	if err := os.WriteFile(tomlPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Load the file
	config, err := LoadVfoxToml(tomlPath)
	if err != nil {
		t.Fatalf("failed to load vfox.toml: %v", err)
	}

	// Verify tools
	if len(config.Tools) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(config.Tools))
	}

	// Check nodejs
	nodeVersion, ok := config.GetToolVersion("nodejs")
	if !ok {
		t.Fatal("nodejs tool not found")
	}
	if nodeVersion != "21.5.1" {
		t.Errorf("expected nodejs version 21.5.1, got %s", nodeVersion)
	}

	// Check python
	pythonVersion, ok := config.GetToolVersion("python")
	if !ok {
		t.Fatal("python tool not found")
	}
	if pythonVersion != "3.11.0" {
		t.Errorf("expected python version 3.11.0, got %s", pythonVersion)
	}
}

func TestLoadVfoxToml_ComplexFormat(t *testing.T) {
	// Create a temporary TOML file with complex format
	tmpDir := t.TempDir()
	tomlPath := filepath.Join(tmpDir, "vfox.toml")

	content := `[tools]
java = { version = "21", vendor = "openjdk", flag = true }
go = { version = "1.21.5", experimental = false }
`

	if err := os.WriteFile(tomlPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Load the file
	config, err := LoadVfoxToml(tomlPath)
	if err != nil {
		t.Fatalf("failed to load vfox.toml: %v", err)
	}

	// Verify java
	javaConfig, ok := config.GetToolConfig("java")
	if !ok {
		t.Fatal("java tool not found")
	}
	if javaConfig.Version != "21" {
		t.Errorf("expected java version 21, got %s", javaConfig.Version)
	}
	if javaConfig.Attr["vendor"] != "openjdk" {
		t.Errorf("expected java vendor openjdk, got %s", javaConfig.Attr["vendor"])
	}
	if javaConfig.Attr["flag"] != "true" {
		t.Errorf("expected java flag true, got %s", javaConfig.Attr["flag"])
	}

	// Verify go
	goConfig, ok := config.GetToolConfig("go")
	if !ok {
		t.Fatal("go tool not found")
	}
	if goConfig.Version != "1.21.5" {
		t.Errorf("expected go version 1.21.5, got %s", goConfig.Version)
	}
	if goConfig.Attr["experimental"] != "false" {
		t.Errorf("expected go experimental false, got %s", goConfig.Attr["experimental"])
	}
}

func TestLoadVfoxToml_MixedFormat(t *testing.T) {
	// Create a temporary TOML file with mixed formats
	tmpDir := t.TempDir()
	tomlPath := filepath.Join(tmpDir, "vfox.toml")

	content := `[tools]
nodejs = "21.5.1"
java = { version = "21", vendor = "openjdk" }
python = "3.11.0"
rust = { version = "1.75.0", nightly = false }
`

	if err := os.WriteFile(tomlPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Load the file
	config, err := LoadVfoxToml(tomlPath)
	if err != nil {
		t.Fatalf("failed to load vfox.toml: %v", err)
	}

	// Verify all tools
	if len(config.Tools) != 4 {
		t.Fatalf("expected 4 tools, got %d", len(config.Tools))
	}

	// Check simple format tools
	nodeVersion, _ := config.GetToolVersion("nodejs")
	if nodeVersion != "21.5.1" {
		t.Errorf("expected nodejs version 21.5.1, got %s", nodeVersion)
	}

	pythonVersion, _ := config.GetToolVersion("python")
	if pythonVersion != "3.11.0" {
		t.Errorf("expected python version 3.11.0, got %s", pythonVersion)
	}

	// Check complex format tools
	javaConfig, _ := config.GetToolConfig("java")
	javaVersion := javaConfig.Version
	javaAttrs := javaConfig.Attr
	if javaVersion != "21" {
		t.Errorf("expected java version 21, got %s", javaVersion)
	}
	if javaAttrs["vendor"] != "openjdk" {
		t.Errorf("expected java vendor openjdk, got %s", javaAttrs["vendor"])
	}

	rustConfgi, _ := config.GetToolConfig("rust")
	rustVersion := rustConfgi.Version
	rustAttrs := rustConfgi.Attr
	if rustVersion != "1.75.0" {
		t.Errorf("expected rust version 1.75.0, got %s", rustVersion)
	}
	if rustAttrs["nightly"] != "false" {
		t.Errorf("expected rust nightly false, got %s", rustAttrs["nightly"])
	}
}

func TestVfoxToml_Save(t *testing.T) {
	// Create a config
	config := NewVfoxToml()
	config.SetTool("nodejs", "21.5.1")
	config.SetToolWithAttr("java", "21", map[string]string{"vendor": "openjdk", "flag": "true"})
	config.SetTool("python", "3.11.0")

	// Save to temp file
	tmpDir := t.TempDir()
	tomlPath := filepath.Join(tmpDir, "vfox.toml")

	if err := config.SaveToPath(tomlPath); err != nil {
		t.Fatalf("failed to save vfox.toml: %v", err)
	}

	// Read the file
	data, err := os.ReadFile(tomlPath)
	if err != nil {
		t.Fatalf("failed to read saved file: %v", err)
	}

	content := string(data)

	// Verify the format
	if !strings.Contains(content, "[tools]") {
		t.Error("expected [tools] section")
	}

	if !strings.Contains(content, `nodejs = "21.5.1"`) {
		t.Error("expected nodejs = \"21.5.1\"")
	}

	// Check that java uses inline format
	if !strings.Contains(content, `java = {version = "21",`) {
		t.Error("expected java to use inline table format")
	}

	if !strings.Contains(content, `vendor = "openjdk"`) {
		t.Error("expected vendor = \"openjdk\"")
	}

	if !strings.Contains(content, `flag = true`) {
		t.Error("expected flag = true")
	}

	if !strings.Contains(content, `python = "3.11.0"`) {
		t.Error("expected python = \"3.11.0\"")
	}
}

func TestVfoxToml_RoundTrip(t *testing.T) {
	// Create a config
	original := NewVfoxToml()
	original.SetTool("nodejs", "21.5.1")
	original.SetToolWithAttr("java", "21", Attr{"vendor": "openjdk", "flag": "true"})
	original.SetToolWithAttr("go", "1.21.5", Attr{"experimental": "false"})
	original.SetTool("python", "3.11.0")

	// Save to temp file
	tmpDir := t.TempDir()
	tomlPath := filepath.Join(tmpDir, "vfox.toml")

	if err := original.SaveToPath(tomlPath); err != nil {
		t.Fatalf("failed to save: %v", err)
	}

	// Load the file
	loaded, err := LoadVfoxToml(tomlPath)
	if err != nil {
		t.Fatalf("failed to load: %v", err)
	}

	// Verify all tools are preserved
	originalTools := original.GetAllTools()
	loadedTools := loaded.GetAllTools()

	if len(originalTools) != len(loadedTools) {
		t.Fatalf("tool count mismatch: %d vs %d", len(originalTools), len(loadedTools))
	}

	for name, version := range originalTools {
		loadedVersion, ok := loaded.GetToolVersion(name)
		if !ok {
			t.Errorf("tool %s not found in loaded config", name)
			continue
		}
		if loadedVersion != version {
			t.Errorf("tool %s version mismatch: %s vs %s", name, version, loadedVersion)
		}
	}

	// Verify attributes for tools with attributes
	javaConfig, _ := loaded.GetToolConfig("java")
	javaVersion := javaConfig.Version
	javaAttrs := javaConfig.Attr
	if javaVersion != "21" {
		t.Errorf("java version mismatch: expected 21, got %s", javaVersion)
	}
	if javaAttrs["vendor"] != "openjdk" {
		t.Errorf("java vendor mismatch: expected openjdk, got %s", javaAttrs["vendor"])
	}
	if javaAttrs["flag"] != "true" {
		t.Errorf("java flag mismatch: expected true, got %s", javaAttrs["flag"])
	}

	goConfig, _ := loaded.GetToolConfig("go")
	goVersion := goConfig.Version
	goAttrs := goConfig.Attr
	if goVersion != "1.21.5" {
		t.Errorf("go version mismatch: expected 1.21.5, got %s", goVersion)
	}
	if goAttrs["experimental"] != "false" {
		t.Errorf("go experimental mismatch: expected false, got %s", goAttrs["experimental"])
	}

	// Verify simple tools have no attributes
	nodeConfig, _ := loaded.GetToolConfig("nodejs")
	nodeAttrs := nodeConfig.Attr
	if len(nodeAttrs) != 0 {
		t.Errorf("nodejs should have no attributes, got %d", len(nodeAttrs))
	}
}

func TestVfoxToml_SetTool(t *testing.T) {
	config := NewVfoxToml()

	// Set tool without attributes
	config.SetTool("nodejs", "21.5.1")

	tc, ok := config.GetToolConfig("nodejs")
	version := tc.Version
	attrs := tc.Attr
	if !ok {
		t.Fatal("tool not found after SetTool")
	}
	if version != "21.5.1" {
		t.Errorf("expected version 21.5.1, got %s", version)
	}
	if len(attrs) != 0 {
		t.Errorf("expected no attributes, got %d", len(attrs))
	}

	// Set tool with attributes
	config.SetToolWithAttr("java", "21", Attr{"vendor": "openjdk"})

	tc, ok = config.GetToolConfig("java")
	version = tc.Version
	attrs = tc.Attr
	if !ok {
		t.Fatal("tool not found after SetTool")
	}
	if version != "21" {
		t.Errorf("expected version 21, got %s", version)
	}
	if attrs["vendor"] != "openjdk" {
		t.Errorf("expected vendor openjdk, got %s", attrs["vendor"])
	}

	// Update existing tool
	config.SetTool("nodejs", "22.0.0")

	version, _ = config.GetToolVersion("nodejs")
	if version != "22.0.0" {
		t.Errorf("expected updated version 22.0.0, got %s", version)
	}
}

func TestVfoxToml_RemoveTool(t *testing.T) {
	config := NewVfoxToml()
	config.SetTool("nodejs", "21.5.1")
	config.SetTool("java", "21")

	if len(config.Tools) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(config.Tools))
	}

	// Remove a tool
	config.RemoveTool("nodejs")

	if len(config.Tools) != 1 {
		t.Fatalf("expected 1 tool after removal, got %d", len(config.Tools))
	}

	_, ok := config.GetToolVersion("nodejs")
	if ok {
		t.Error("nodejs should not exist after removal")
	}

	_, ok = config.GetToolVersion("java")
	if !ok {
		t.Error("java should still exist")
	}
}

func TestVfoxToml_GetAllTools(t *testing.T) {
	config := NewVfoxToml()
	config.SetTool("nodejs", "21.5.1")
	config.SetToolWithAttr("java", "21", map[string]string{"vendor": "openjdk"})
	config.SetTool("python", "3.11.0")

	allTools := config.GetAllTools()

	if len(allTools) != 3 {
		t.Fatalf("expected 3 tools, got %d", len(allTools))
	}

	if allTools["nodejs"] != "21.5.1" {
		t.Errorf("expected nodejs version 21.5.1, got %s", allTools["nodejs"])
	}

	if allTools["java"] != "21" {
		t.Errorf("expected java version 21, got %s", allTools["java"])
	}

	if allTools["python"] != "3.11.0" {
		t.Errorf("expected python version 3.11.0, got %s", allTools["python"])
	}
}

func TestVfoxToml_MarshalTOML(t *testing.T) {
	config := NewVfoxToml()
	config.SetToolWithAttr("nodejs", "21.5.1", nil)
	config.SetToolWithAttr("java", "21", map[string]string{"vendor": "openjdk", "flag": "true"})
	config.SetToolWithAttr("python", "3.11.0", nil)
	config.SetToolWithAttr("rust", "1.75.0", map[string]string{"nightly": "false"})

	data, err := config.MarshalTOML()
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	content := string(data)

	// Verify structure
	lines := strings.Split(content, "\n")

	// First line should be [tools]
	if !strings.HasPrefix(lines[0], "[tools]") {
		t.Errorf("first line should be [tools], got: %s", lines[0])
	}

	// Verify simple format tools
	if !strings.Contains(content, `nodejs = "21.5.1"`) {
		t.Error("expected nodejs simple format")
	}

	if !strings.Contains(content, `python = "3.11.0"`) {
		t.Error("expected python simple format")
	}

	// Verify inline table format
	if !strings.Contains(content, `java = {version = "21",`) {
		t.Error("expected java inline table format")
	}

	if !strings.Contains(content, `vendor = "openjdk"`) {
		t.Error("expected vendor attribute")
	}

	if !strings.Contains(content, `flag = true`) {
		t.Error("expected flag attribute")
	}

	if !strings.Contains(content, `rust = {version = "1.75.0",`) {
		t.Error("expected rust inline table format")
	}

	if !strings.Contains(content, `nightly = false`) {
		t.Error("expected nightly attribute")
	}
}

func TestToolConfig_MarshalInline(t *testing.T) {
	tests := []struct {
		name     string
		config   *ToolConfig
		expected string
	}{
		{
			name:     "simple format",
			config:   &ToolConfig{Version: "21.5.1", Attr: make(Attr)},
			expected: `"21.5.1"`,
		},
		{
			name: "complex format with string attribute",
			config: &ToolConfig{
				Version: "21",
				Attr:    Attr{"vendor": "openjdk"},
			},
			expected: `{version = "21", vendor = "openjdk"}`,
		},
		{
			name: "complex format with boolean attribute",
			config: &ToolConfig{
				Version: "21",
				Attr:    Attr{"flag": "true"},
			},
			expected: `{version = "21", flag = true}`,
		},
		{
			name: "complex format with integer attribute",
			config: &ToolConfig{
				Version: "1.21.5",
				Attr:    Attr{"port": "8080"},
			},
			expected: `{version = "1.21.5", port = 8080}`,
		},
		{
			name: "complex format with multiple attributes",
			config: &ToolConfig{
				Version: "21",
				Attr:    Attr{"vendor": "openjdk", "flag": "true"},
			},
			expected: `{version = "21", flag = true, vendor = "openjdk"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.MarshalInline()
			if result != tt.expected {
				t.Errorf("MarshalInline() = %s, want %s", result, tt.expected)
			}
		})
	}
}
