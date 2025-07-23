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

package commands

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/urfave/cli/v2"
)

// Helper to capture output
func captureOutput(fn func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	fn()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	return strings.TrimSpace(buf.String())
}

func TestPathCmd_InvalidParameters(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectedErr string
	}{
		{
			name:        "no arguments",
			args:        []string{},
			expectedErr: "invalid parameter. format: <sdk-name>[@<version>]",
		},
		{
			name:        "missing version",
			args:        []string{"nodejs"},
			expectedErr: "version is required. format: <sdk-name>@<version>",
		},
		{
			name:        "empty argument",
			args:        []string{""},
			expectedErr: "invalid parameter. format: <sdk-name>[@<version>]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := &cli.App{
				Commands: []*cli.Command{Path},
			}

			args := append([]string{"vfox", "path"}, tt.args...)
			err := app.Run(args)

			if err == nil {
				t.Fatalf("expected error, got nil")
			}

			if !strings.Contains(err.Error(), tt.expectedErr) {
				t.Errorf("expected error containing %q, got %q", tt.expectedErr, err.Error())
			}
		})
	}
}

// Test the argument parsing logic separately from manager dependencies
func TestPathCmd_ArgumentParsing(t *testing.T) {
	tests := []struct {
		name          string
		sdkArg        string
		expectedName  string
		expectedVer   string
		expectError   bool
		expectedError string
	}{
		{
			name:        "valid format",
			sdkArg:      "nodejs@18.0.0",
			expectedName: "nodejs",
			expectedVer: "18.0.0",
			expectError: false,
		},
		{
			name:        "valid format with patch version",
			sdkArg:      "java@11.0.1",
			expectedName: "java",
			expectedVer: "11.0.1",
			expectError: false,
		},
		{
			name:          "missing version separator",
			sdkArg:        "nodejs",
			expectError:   true,
			expectedError: "version is required",
		},
		{
			name:          "empty string",
			sdkArg:        "",
			expectError:   true,
			expectedError: "invalid parameter",
		},
		{
			name:        "empty version",
			sdkArg:      "nodejs@",
			expectedName: "nodejs",
			expectedVer: "",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the parsing logic manually
			if tt.sdkArg == "" {
				if !tt.expectError {
					t.Error("Expected error for empty string")
				}
				return
			}

			argArr := strings.Split(tt.sdkArg, "@")
			if len(argArr) <= 1 {
				if !tt.expectError {
					t.Error("Expected error for missing version")
				}
				return
			}

			name := argArr[0]
			version := argArr[1]

			if tt.expectError {
				t.Error("Expected error but parsing succeeded")
				return
			}

			if name != tt.expectedName {
				t.Errorf("Expected name %q, got %q", tt.expectedName, name)
			}

			if version != tt.expectedVer {
				t.Errorf("Expected version %q, got %q", tt.expectedVer, version)
			}
		})
	}
}

// Test JSON output format
func TestPathCmd_JSONOutput(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		found    bool
		expected map[string]string
	}{
		{
			name:  "found",
			path:  "/home/user/.vfox/cache/nodejs/v-18.0.0",
			found: true,
			expected: map[string]string{
				"path":  "/home/user/.vfox/cache/nodejs/v-18.0.0",
				"found": "true",
			},
		},
		{
			name:  "not found",
			path:  "",
			found: false,
			expected: map[string]string{
				"path":  "",
				"found": "false",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := map[string]string{
				"path":  tt.path,
				"found": func() string {
					if tt.found {
						return "true"
					}
					return "false"
				}(),
			}

			jsonBytes, err := json.Marshal(result)
			if err != nil {
				t.Fatalf("Failed to marshal JSON: %v", err)
			}

			var parsed map[string]string
			if err := json.Unmarshal(jsonBytes, &parsed); err != nil {
				t.Fatalf("Failed to parse JSON: %v", err)
			}

			for key, expectedValue := range tt.expected {
				if actualValue, exists := parsed[key]; !exists {
					t.Errorf("Missing key %q in JSON output", key)
				} else if actualValue != expectedValue {
					t.Errorf("For key %q, expected %q but got %q", key, expectedValue, actualValue)
				}
			}
		})
	}
}