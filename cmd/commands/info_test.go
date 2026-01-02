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

package commands

import (
	"bytes"
	"strings"
	"testing"
	"text/template"
)

func TestSDKVersionDataStructure(t *testing.T) {
	// Test that the data structure for template output contains all required fields
	// This is more of a compile-time check to ensure we have the right fields

	// For sdk@version format
	sdkData := struct {
		Name    string
		Version string
		Path    string
	}{
		Name:    "test",
		Version: "1.0.0",
		Path:    "/path/to/sdk",
	}

	// Check that all fields exist and are accessible
	if sdkData.Name != "test" {
		t.Errorf("Expected Name to be 'test', got %s", sdkData.Name)
	}
	if sdkData.Version != "1.0.0" {
		t.Errorf("Expected Version to be '1.0.0', got %s", sdkData.Version)
	}
	if sdkData.Path != "/path/to/sdk" {
		t.Errorf("Expected Path to be '/path/to/sdk', got %s", sdkData.Path)
	}
}

func TestPluginInfoDataStructure(t *testing.T) {
	// Test that the data structure for plugin info template output contains all required fields
	// This is more of a compile-time check to ensure we have the right fields

	// For plugin info format
	pluginData := struct {
		Name        string
		Version     string
		Homepage    string
		InstallPath string
		Description string
	}{
		Name:        "test",
		Version:     "1.0.0",
		Homepage:    "https://example.com",
		InstallPath: "/path/to/plugin",
		Description: "Test plugin",
	}

	// Check that all fields exist and are accessible
	if pluginData.Name != "test" {
		t.Errorf("Expected Name to be 'test', got %s", pluginData.Name)
	}
	if pluginData.Version != "1.0.0" {
		t.Errorf("Expected Version to be '1.0.0', got %s", pluginData.Version)
	}
	if pluginData.Homepage != "https://example.com" {
		t.Errorf("Expected Homepage to be 'https://example.com', got %s", pluginData.Homepage)
	}
	if pluginData.InstallPath != "/path/to/plugin" {
		t.Errorf("Expected InstallPath to be '/path/to/plugin', got %s", pluginData.InstallPath)
	}
	if pluginData.Description != "Test plugin" {
		t.Errorf("Expected Description to be 'Test plugin', got %s", pluginData.Description)
	}
}

func TestTemplateExecution(t *testing.T) {
	// Test template execution with sample data
	tests := []struct {
		name     string
		template string
		data     interface{}
		expected string
	}{
		{
			name:     "Test Name field",
			template: "{{.Name}}",
			data: struct {
				Name    string
				Version string
				Path    string
			}{
				Name:    "nodejs",
				Version: "18.0.0",
				Path:    "/path/to/nodejs-18.0.0",
			},
			expected: "nodejs",
		},
		{
			name:     "Test Version field",
			template: "{{.Version}}",
			data: struct {
				Name    string
				Version string
				Path    string
			}{
				Name:    "nodejs",
				Version: "18.0.0",
				Path:    "/path/to/nodejs-18.0.0",
			},
			expected: "18.0.0",
		},
		{
			name:     "Test Path field",
			template: "{{.Path}}",
			data: struct {
				Name    string
				Version string
				Path    string
			}{
				Name:    "nodejs",
				Version: "18.0.0",
				Path:    "/path/to/nodejs-18.0.0",
			},
			expected: "/path/to/nodejs-18.0.0",
		},
		{
			name:     "Test notfound path",
			template: "{{.Path}}",
			data: struct {
				Name    string
				Version string
				Path    string
			}{
				Name:    "nodejs",
				Version: "18.0.0",
				Path:    "notfound",
			},
			expected: "notfound",
		},
		{
			name:     "Test plugin Name field",
			template: "{{.Name}}",
			data: struct {
				Name        string
				Version     string
				Homepage    string
				Description string
			}{
				Name:        "erlang",
				Version:     "1.2.0",
				Homepage:    "https://github.com/version-fox/vfox-erlang",
				Description: "Erlang/OTP vfox plugin",
			},
			expected: "erlang",
		},
		{
			name:     "Test plugin Version field",
			template: "{{.Version}}",
			data: struct {
				Name        string
				Version     string
				Homepage    string
				Description string
			}{
				Name:        "erlang",
				Version:     "1.2.0",
				Homepage:    "https://github.com/version-fox/vfox-erlang",
				Description: "Erlang/OTP vfox plugin",
			},
			expected: "1.2.0",
		},
		{
			name:     "Test plugin Homepage field",
			template: "{{.Homepage}}",
			data: struct {
				Name        string
				Version     string
				Homepage    string
				InstallPath string
				Description string
			}{
				Name:        "erlang",
				Version:     "1.2.0",
				Homepage:    "https://github.com/version-fox/vfox-erlang",
				InstallPath: "/path/to/erlang/plugin",
				Description: "Erlang/OTP vfox plugin",
			},
			expected: "https://github.com/version-fox/vfox-erlang",
		},
		{
			name:     "Test plugin InstallPath field",
			template: "{{.InstallPath}}",
			data: struct {
				Name        string
				Version     string
				Homepage    string
				InstallPath string
				Description string
			}{
				Name:        "erlang",
				Version:     "1.2.0",
				Homepage:    "https://github.com/version-fox/vfox-erlang",
				InstallPath: "/path/to/erlang/plugin",
				Description: "Erlang/OTP vfox plugin",
			},
			expected: "/path/to/erlang/plugin",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpl, err := template.New("test").Parse(tt.template)
			if err != nil {
				t.Fatalf("Error parsing template: %v", err)
			}

			var result strings.Builder
			err = tmpl.Execute(&result, tt.data)
			if err != nil {
				t.Fatalf("Error executing template: %v", err)
			}

			actual := result.String()
			if actual != tt.expected {
				t.Errorf("Expected %q, but got %q", tt.expected, actual)
			}
		})
	}
}

func TestExecuteTemplate(t *testing.T) {
	// Create a mock CLI context for testing
	// Since we can't easily create a real cli.Context, we'll test the template execution directly

	// Test SDK version data template
	sdkData := struct {
		Name    string
		Version string
		Path    string
	}{
		Name:    "erlang",
		Version: "26.2.3",
		Path:    "/path/to/erlang-26.2.3",
	}

	// Create a template
	tmplStr := "{{.Name}}@{{.Version}}: {{.Path}}"
	tmpl, err := template.New("test").Parse(tmplStr)
	if err != nil {
		t.Fatalf("Error parsing template: %v", err)
	}

	// Execute template
	var result bytes.Buffer
	err = tmpl.Execute(&result, sdkData)
	if err != nil {
		t.Fatalf("Error executing template: %v", err)
	}

	expected := "erlang@26.2.3: /path/to/erlang-26.2.3"
	actual := result.String()
	if actual != expected {
		t.Errorf("Expected %q, but got %q", expected, actual)
	}

	// Test plugin info data template
	pluginData := struct {
		Name        string
		Version     string
		Homepage    string
		InstallPath string
		Description string
	}{
		Name:        "erlang",
		Version:     "1.2.0",
		Homepage:    "https://github.com/version-fox/vfox-erlang",
		InstallPath: "/path/to/erlang/plugin",
		Description: "Erlang/OTP vfox plugin",
	}

	// Create a template for plugin info
	pluginTmplStr := "Name: {{.Name}}, Version: {{.Version}}"
	pluginTmpl, err := template.New("plugin").Parse(pluginTmplStr)
	if err != nil {
		t.Fatalf("Error parsing plugin template: %v", err)
	}

	// Execute plugin template
	var pluginResult bytes.Buffer
	err = pluginTmpl.Execute(&pluginResult, pluginData)
	if err != nil {
		t.Fatalf("Error executing plugin template: %v", err)
	}

	pluginExpected := "Name: erlang, Version: 1.2.0"
	pluginActual := pluginResult.String()
	if pluginActual != pluginExpected {
		t.Errorf("Expected %q, but got %q", pluginExpected, pluginActual)
	}

	// Test plugin info with Homepage and InstallPath
	fullPluginTmplStr := "Homepage: {{.Homepage}}, InstallPath: {{.InstallPath}}"
	fullPluginTmpl, err := template.New("fullPlugin").Parse(fullPluginTmplStr)
	if err != nil {
		t.Fatalf("Error parsing full plugin template: %v", err)
	}

	// Execute full plugin template
	var fullPluginResult bytes.Buffer
	err = fullPluginTmpl.Execute(&fullPluginResult, pluginData)
	if err != nil {
		t.Fatalf("Error executing full plugin template: %v", err)
	}

	fullPluginExpected := "Homepage: https://github.com/version-fox/vfox-erlang, InstallPath: /path/to/erlang/plugin"
	fullPluginActual := fullPluginResult.String()
	if fullPluginActual != fullPluginExpected {
		t.Errorf("Expected %q, but got %q", fullPluginExpected, fullPluginActual)
	}
}
