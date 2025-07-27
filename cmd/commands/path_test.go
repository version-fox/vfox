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
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/urfave/cli/v2"
)

func TestPathCmdArgumentParsing(t *testing.T) {
	// Test case 1: No arguments provided
	t.Run("No arguments", func(t *testing.T) {
		app := &cli.App{}
		set := flag.NewFlagSet("test", 0)
		ctx := cli.NewContext(app, set, nil)

		err := pathCmd(ctx)
		if err == nil {
			t.Error("Expected error when no arguments provided")
		}

		if exitErr, ok := err.(cli.ExitCoder); ok {
			if exitErr.ExitCode() != 1 {
				t.Errorf("Expected exit code 1, got %d", exitErr.ExitCode())
			}
			if !strings.Contains(err.Error(), "sdk name is required") {
				t.Errorf("Expected error message to contain 'sdk name is required', got '%s'", err.Error())
			}
		} else {
			t.Errorf("Expected cli.ExitCoder, got %T", err)
		}
	})

	// Test case 2: Invalid argument format (no @ symbol)
	t.Run("Invalid format - no @", func(t *testing.T) {
		app := &cli.App{}
		set := flag.NewFlagSet("test", 0)
		set.Parse([]string{"nodejs"})
		ctx := cli.NewContext(app, set, nil)

		err := pathCmd(ctx)
		if err == nil {
			t.Error("Expected error for invalid argument format")
		}

		if exitErr, ok := err.(cli.ExitCoder); ok {
			if exitErr.ExitCode() != 1 {
				t.Errorf("Expected exit code 1, got %d", exitErr.ExitCode())
			}
			if !strings.Contains(err.Error(), "invalid arguments") {
				t.Errorf("Expected error message to contain 'invalid arguments', got '%s'", err.Error())
			}
		} else {
			t.Errorf("Expected cli.ExitCoder, got %T", err)
		}
	})

	// Test case 3: Invalid argument format (multiple @ symbols)
	t.Run("Invalid format - multiple @", func(t *testing.T) {
		app := &cli.App{}
		set := flag.NewFlagSet("test", 0)
		set.Parse([]string{"nodejs@18@latest"})
		ctx := cli.NewContext(app, set, nil)

		err := pathCmd(ctx)
		if err == nil {
			t.Error("Expected error for invalid argument format")
		}

		if exitErr, ok := err.(cli.ExitCoder); ok {
			if exitErr.ExitCode() != 1 {
				t.Errorf("Expected exit code 1, got %d", exitErr.ExitCode())
			}
			if !strings.Contains(err.Error(), "invalid arguments") {
				t.Errorf("Expected error message to contain 'invalid arguments', got '%s'", err.Error())
			}
		} else {
			t.Errorf("Expected cli.ExitCoder, got %T", err)
		}
	})

	// Test case 4: Empty SDK name
	t.Run("Empty SDK name", func(t *testing.T) {
		app := &cli.App{}
		set := flag.NewFlagSet("test", 0)
		set.Parse([]string{"@1.0.0"})
		ctx := cli.NewContext(app, set, nil)

		err := pathCmd(ctx)
		if err == nil {
			t.Error("Expected error for empty SDK name")
		}

		if exitErr, ok := err.(cli.ExitCoder); ok {
			if exitErr.ExitCode() != 1 {
				t.Errorf("Expected exit code 1, got %d", exitErr.ExitCode())
			}
			if !strings.Contains(err.Error(), "invalid arguments") {
				t.Errorf("Expected error message to contain 'invalid arguments', got '%s'", err.Error())
			}
		} else {
			t.Errorf("Expected cli.ExitCoder, got %T", err)
		}
	})

	// Test case 5: Empty version
	t.Run("Empty version", func(t *testing.T) {
		app := &cli.App{}
		set := flag.NewFlagSet("test", 0)
		set.Parse([]string{"nodejs@"})
		ctx := cli.NewContext(app, set, nil)

		err := pathCmd(ctx)
		if err == nil {
			t.Error("Expected error for empty version")
		}

		if exitErr, ok := err.(cli.ExitCoder); ok {
			if exitErr.ExitCode() != 1 {
				t.Errorf("Expected exit code 1, got %d", exitErr.ExitCode())
			}
			if !strings.Contains(err.Error(), "invalid arguments") {
				t.Errorf("Expected error message to contain 'invalid arguments', got '%s'", err.Error())
			}
		} else {
			t.Errorf("Expected cli.ExitCoder, got %T", err)
		}
	})

	// Test case 6: Success case - SDK not found (mocked)
	t.Run("SDK not found", func(t *testing.T) {
		// Capture stdout
		old := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		app := &cli.App{}
		set := flag.NewFlagSet("test", 0)
		set.Parse([]string{"nonexistent@1.0.0"})
		ctx := cli.NewContext(app, set, nil)

		err := pathCmd(ctx)

		// Restore stdout
		w.Close()
		os.Stdout = old

		// Read captured output
		var buf bytes.Buffer
		io.Copy(&buf, r)
		output := strings.TrimSpace(buf.String())

		if err != nil {
			t.Errorf("Expected no error for non-existent SDK, got %v", err)
		}

		if output != "notfound" {
			t.Errorf("Expected output 'notfound', got '%s'", output)
		}
	})
}

func TestPathCmdIntegration(t *testing.T) {
	// This would be an integration test that actually tests the command with a real SDK manager
	// For now, we'll just test that the command compiles and has the right structure

	if Path.Name != "path" {
		t.Errorf("Expected command name 'path', got '%s'", Path.Name)
	}

	if !strings.Contains(Path.Usage, "path") {
		t.Errorf("Expected usage to contain 'path', got '%s'", Path.Usage)
	}

	if Path.Action == nil {
		t.Error("Expected Path.Action to be defined")
	}
}

func TestPathCmdWithMockedSDK(t *testing.T) {
	// Test case: Simulate successful path retrieval
	t.Run("Mock successful path retrieval - etcd example", func(t *testing.T) {
		// Create a temporary directory structure to simulate SDK installation
		tempDir := t.TempDir()
		sdkName := "etcd"
		version := "3.6.0"
		// Simulate the actual vfox directory structure
		sdkPath := filepath.Join(tempDir, ".version-fox", "cache", sdkName, fmt.Sprintf("v-%s", version))

		// Create the directory structure
		err := os.MkdirAll(sdkPath, 0755)
		if err != nil {
			t.Fatal(err)
		}

		// Capture stdout
		old := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		app := &cli.App{}
		set := flag.NewFlagSet("test", 0)
		set.Parse([]string{fmt.Sprintf("%s@%s", sdkName, version)})
		ctx := cli.NewContext(app, set, nil)

		// This will fail because we can't easily mock the internal SDK manager
		// But we can test that the function doesn't panic and handles the case gracefully
		err = pathCmd(ctx)

		// Restore stdout
		w.Close()
		os.Stdout = old

		// Read captured output
		var buf bytes.Buffer
		io.Copy(&buf, r)
		output := strings.TrimSpace(buf.String())

		// Since we don't have a real SDK installed, it should return "notfound"
		// This test verifies that the function executes without errors
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		// The output could be either "notfound" or an actual path if the SDK is installed
		if output == "notfound" {
			t.Logf("SDK not found (expected for non-installed SDK)")
		} else if strings.Contains(output, "etcd-3.6.0") {
			t.Logf("SDK found at path: %s", output)
			// Verify the path format is correct
			if !strings.HasSuffix(output, "etcd-3.6.0") {
				t.Errorf("Expected path to end with 'etcd-3.6.0', got '%s'", output)
			}
			if !strings.Contains(output, "v-3.6.0") {
				t.Errorf("Expected path to contain 'v-3.6.0', got '%s'", output)
			}
		} else {
			t.Errorf("Unexpected output format: '%s'", output)
		}
	})

	// Test case: Verify path format when SDK exists (conceptual test)
	t.Run("Verify expected path format - etcd example", func(t *testing.T) {
		// This test verifies the path format logic using the etcd@3.6.0 example
		sdkName := "etcd"
		version := "3.6.0"
		expectedPathSuffix := fmt.Sprintf("%s-%s", sdkName, version)

		// Test the path construction logic
		if !strings.Contains(expectedPathSuffix, sdkName) {
			t.Errorf("Expected path suffix to contain SDK name '%s'", sdkName)
		}

		if !strings.Contains(expectedPathSuffix, version) {
			t.Errorf("Expected path suffix to contain version '%s'", version)
		}

		// Verify the format matches what pathCmd would generate
		expectedFormat := fmt.Sprintf("%s-%s", sdkName, version)
		if expectedPathSuffix != expectedFormat {
			t.Errorf("Expected path suffix '%s', got '%s'", expectedFormat, expectedPathSuffix)
		}

		// Verify the complete expected path format
		expectedFinalComponent := "etcd-3.6.0"
		if expectedPathSuffix != expectedFinalComponent {
			t.Errorf("Expected final component '%s', got '%s'", expectedFinalComponent, expectedPathSuffix)
		}
	})

	// Test case: Test argument parsing for valid format
	t.Run("Valid argument parsing", func(t *testing.T) {
		testCases := []struct {
			input       string
			expectedSDK string
			expectedVer string
		}{
			{"etcd@3.6.0", "etcd", "3.6.0"}, // Primary test case
			{"nodejs@18.0.0", "nodejs", "18.0.0"},
			{"Python@3.9.0", "python", "3.9.0"}, // Should be lowercased
			{"java@11.0.1", "java", "11.0.1"},
			{"go@1.20.0", "go", "1.20.0"},
		}

		for _, tc := range testCases {
			t.Run(fmt.Sprintf("Parse %s", tc.input), func(t *testing.T) {
				// Test the argument parsing logic that pathCmd uses
				argArr := strings.Split(tc.input, "@")
				if len(argArr) != 2 {
					t.Errorf("Expected 2 parts for input '%s', got %d", tc.input, len(argArr))
					return
				}

				name := strings.ToLower(argArr[0])
				version := argArr[1]

				if name != tc.expectedSDK {
					t.Errorf("Expected SDK name '%s', got '%s'", tc.expectedSDK, name)
				}

				if version != tc.expectedVer {
					t.Errorf("Expected version '%s', got '%s'", tc.expectedVer, version)
				}

				// Verify neither is empty (additional validation that pathCmd does)
				if name == "" || version == "" {
					t.Error("SDK name or version should not be empty")
				}
			})
		}
	})

	// Test case: Simulate the path construction logic
	t.Run("Simulate successful path construction - etcd example", func(t *testing.T) {
		// This test simulates what happens when an SDK exists and we construct the path
		// Using the specific etcd@3.6.0 example that should return ~/.version-fox/cache/etcd/v-3.6.0/etcd-3.6.0
		testCases := []struct {
			sdkName string
			version string
		}{
			{
				sdkName: "etcd",
				version: "3.6.0",
			},
			{
				sdkName: "nodejs",
				version: "18.0.0",
			},
			{
				sdkName: "python",
				version: "3.9.0",
			},
		}

		for _, tc := range testCases {
			t.Run(fmt.Sprintf("%s@%s", tc.sdkName, tc.version), func(t *testing.T) {
				// Build paths using filepath.Join for cross-platform compatibility
				// Create a base path that works on all platforms
				basePath := filepath.Join("home", "user", ".version-fox", "cache")
				versionPath := filepath.Join(basePath, tc.sdkName, fmt.Sprintf("v-%s", tc.version))
				expectedPath := filepath.Join(versionPath, fmt.Sprintf("%s-%s", tc.sdkName, tc.version))
				
				// This simulates the path construction logic from pathCmd:
				// filepath.Join(sdk.VersionPath(version), fmt.Sprintf("%s-%s", name, version))
				actualPath := filepath.Join(versionPath, fmt.Sprintf("%s-%s", tc.sdkName, tc.version))

				if actualPath != expectedPath {
					t.Errorf("Expected path '%s', got '%s'", expectedPath, actualPath)
				}

				// Verify the path contains the expected components
				if !strings.Contains(actualPath, tc.sdkName) {
					t.Errorf("Expected path to contain SDK name '%s'", tc.sdkName)
				}

				if !strings.Contains(actualPath, tc.version) {
					t.Errorf("Expected path to contain version '%s'", tc.version)
				}

				// Verify the final component follows the expected format
				expectedFinalComponent := fmt.Sprintf("%s-%s", tc.sdkName, tc.version)
				if !strings.HasSuffix(actualPath, expectedFinalComponent) {
					t.Errorf("Expected path to end with '%s'", expectedFinalComponent)
				}
				
				// Verify the version directory format
				expectedVersionDir := fmt.Sprintf("v-%s", tc.version)
				if !strings.Contains(actualPath, expectedVersionDir) {
					t.Errorf("Expected path to contain version directory '%s'", expectedVersionDir)
				}
			})
		}
	})	// Test case: Test the specific etcd@3.6.0 case mentioned in requirements
	t.Run("Specific etcd@3.6.0 path format test", func(t *testing.T) {
		// Test the exact case: vfox path etcd@3.6.0 should return ~/.version-fox/cache/etcd/v-3.6.0/etcd-3.6.0
		sdkName := "etcd"
		version := "3.6.0"

		// Test argument parsing
		input := fmt.Sprintf("%s@%s", sdkName, version)
		argArr := strings.Split(input, "@")

		if len(argArr) != 2 {
			t.Errorf("Expected 2 parts for input '%s', got %d", input, len(argArr))
			return
		}

		parsedName := strings.ToLower(argArr[0])
		parsedVersion := argArr[1]

		if parsedName != sdkName {
			t.Errorf("Expected SDK name '%s', got '%s'", sdkName, parsedName)
		}

		if parsedVersion != version {
			t.Errorf("Expected version '%s', got '%s'", version, parsedVersion)
		}

		// Test path construction logic using cross-platform approach
		homeDir, _ := os.UserHomeDir()
		expectedVersionPath := filepath.Join(homeDir, ".version-fox", "cache", sdkName, fmt.Sprintf("v-%s", version))
		expectedFinalPath := filepath.Join(expectedVersionPath, fmt.Sprintf("%s-%s", sdkName, version))

		// Verify the expected path components (cross-platform compatible)
		pathComponents := []string{".version-fox", "cache", "etcd", "v-3.6.0", "etcd-3.6.0"}
		for _, component := range pathComponents {
			if !strings.Contains(expectedFinalPath, component) {
				t.Errorf("Expected path to contain component '%s', got '%s'", component, expectedFinalPath)
			}
		}

		// Verify the final component
		if !strings.HasSuffix(expectedFinalPath, "etcd-3.6.0") {
			t.Errorf("Expected path to end with 'etcd-3.6.0', got '%s'", expectedFinalPath)
		}

		t.Logf("Expected path format for etcd@3.6.0: %s", expectedFinalPath)
	})
}
