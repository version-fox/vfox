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

package env

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/version-fox/vfox/internal/config"
	"github.com/version-fox/vfox/internal/pathmeta"
)

func TestGetUserAddedPaths(t *testing.T) {
	// Save original env vars
	origPath := os.Getenv("PATH")
	origOriginalPath := os.Getenv(OriginalPathFlag)
	defer func() {
		os.Setenv("PATH", origPath)
		os.Setenv(OriginalPathFlag, origOriginalPath)
	}()

	t.Run("no original path stored", func(t *testing.T) {
		// Clear the original path flag
		os.Setenv(OriginalPathFlag, "")
		os.Setenv("PATH", "/usr/bin:/bin")

		ctx := createTestContext(t)
		userPaths := ctx.GetUserAddedPaths()

		if len(userPaths.Slice()) != 0 {
			t.Errorf("Expected empty user paths when no original path stored, got: %v", userPaths.Slice())
		}
	})

	t.Run("no user additions", func(t *testing.T) {
		// Set original and current PATH to be the same
		originalPath := "/usr/bin:/bin"
		os.Setenv(OriginalPathFlag, originalPath)
		os.Setenv("PATH", originalPath)

		ctx := createTestContext(t)
		userPaths := ctx.GetUserAddedPaths()

		if len(userPaths.Slice()) != 0 {
			t.Errorf("Expected empty user paths when no additions made, got: %v", userPaths.Slice())
		}
	})

	t.Run("user added single path", func(t *testing.T) {
		// Set original PATH
		originalPath := "/usr/bin:/bin"
		os.Setenv(OriginalPathFlag, originalPath)

		// User adds a new path at the front
		currentPath := "/home/user/mybin:/usr/bin:/bin"
		os.Setenv("PATH", currentPath)

		ctx := createTestContext(t)
		userPaths := ctx.GetUserAddedPaths()

		expected := []string{"/home/user/mybin"}
		actual := userPaths.Slice()

		if len(actual) != len(expected) {
			t.Fatalf("Expected %d user paths, got %d: %v", len(expected), len(actual), actual)
		}

		for i, exp := range expected {
			if filepath.Clean(actual[i]) != filepath.Clean(exp) {
				t.Errorf("Expected user path[%d] = %s, got %s", i, exp, actual[i])
			}
		}
	})

	t.Run("user added multiple paths", func(t *testing.T) {
		// Set original PATH
		originalPath := "/usr/bin:/bin"
		os.Setenv(OriginalPathFlag, originalPath)

		// User adds multiple new paths
		currentPath := "/home/user/bin1:/home/user/bin2:/usr/bin:/bin:/opt/tools"
		os.Setenv("PATH", currentPath)

		ctx := createTestContext(t)
		userPaths := ctx.GetUserAddedPaths()

		expected := []string{"/home/user/bin1", "/home/user/bin2", "/opt/tools"}
		actual := userPaths.Slice()

		if len(actual) != len(expected) {
			t.Fatalf("Expected %d user paths, got %d: %v", len(expected), len(actual), actual)
		}

		for i, exp := range expected {
			if filepath.Clean(actual[i]) != filepath.Clean(exp) {
				t.Errorf("Expected user path[%d] = %s, got %s", i, exp, actual[i])
			}
		}
	})

	t.Run("vfox paths are not included as user additions", func(t *testing.T) {
		// Set original PATH
		originalPath := "/usr/bin:/bin"
		os.Setenv(OriginalPathFlag, originalPath)

		// Current PATH includes vfox paths and a user-added path
		currentPath := "/home/user/.vfox/sdk/nodejs/bin:/home/user/mybin:/usr/bin:/bin"
		os.Setenv("PATH", currentPath)

		ctx := createTestContext(t)
		userPaths := ctx.GetUserAddedPaths()

		expected := []string{"/home/user/mybin"}
		actual := userPaths.Slice()

		if len(actual) != len(expected) {
			t.Fatalf("Expected %d user paths, got %d: %v", len(expected), len(actual), actual)
		}

		if filepath.Clean(actual[0]) != filepath.Clean(expected[0]) {
			t.Errorf("Expected user path = %s, got %s", expected[0], actual[0])
		}
	})

	t.Run("handles path with spaces and special characters", func(t *testing.T) {
		// Set original PATH
		originalPath := "/usr/bin:/bin"
		os.Setenv(OriginalPathFlag, originalPath)

		// User adds a path with spaces
		userAddedPath := "/home/user/my tools/bin"
		currentPath := strings.Join([]string{userAddedPath, "/usr/bin", "/bin"}, string(os.PathListSeparator))
		os.Setenv("PATH", currentPath)

		ctx := createTestContext(t)
		userPaths := ctx.GetUserAddedPaths()

		if len(userPaths.Slice()) != 1 {
			t.Fatalf("Expected 1 user path, got %d: %v", len(userPaths.Slice()), userPaths.Slice())
		}

		actual := filepath.Clean(userPaths.Slice()[0])
		expected := filepath.Clean(userAddedPath)

		if actual != expected {
			t.Errorf("Expected user path = %s, got %s", expected, actual)
		}
	})
}

func TestCleanSystemPaths(t *testing.T) {
	// Save original PATH
	origPath := os.Getenv("PATH")
	defer func() {
		os.Setenv("PATH", origPath)
	}()

	t.Run("removes vfox paths", func(t *testing.T) {
		// Set PATH with vfox-managed paths
		vfoxPath := "/home/user/.vfox/sdk/nodejs/bin"
		systemPath := "/usr/bin:/bin"
		currentPath := strings.Join([]string{vfoxPath, systemPath}, string(os.PathListSeparator))
		os.Setenv("PATH", currentPath)

		ctx := createTestContext(t)
		cleanPaths := ctx.CleanSystemPaths()

		// Should only have system paths
		actual := cleanPaths.Slice()
		if len(actual) != 2 {
			t.Fatalf("Expected 2 clean paths, got %d: %v", len(actual), actual)
		}

		// Check that vfox path is not in the result
		for _, p := range actual {
			if strings.Contains(p, ".vfox") {
				t.Errorf("vfox path should be removed: %s", p)
			}
		}
	})

	t.Run("preserves non-vfox paths", func(t *testing.T) {
		// Set PATH with only system paths
		systemPath := "/usr/bin:/bin:/usr/local/bin"
		os.Setenv("PATH", systemPath)

		ctx := createTestContext(t)
		cleanPaths := ctx.CleanSystemPaths()

		// Should have all system paths
		actual := cleanPaths.Slice()
		if len(actual) != 3 {
			t.Fatalf("Expected 3 clean paths, got %d: %v", len(actual), actual)
		}
	})
}

// createTestContext creates a minimal RuntimeEnvContext for testing
func createTestContext(t *testing.T) *RuntimeEnvContext {
	t.Helper()

	tmpDir := t.TempDir()

	return &RuntimeEnvContext{
		UserConfig: &config.Config{},
		PathMeta: &pathmeta.PathMeta{
			User: pathmeta.UserPaths{
				Home: tmpDir,
				Temp: filepath.Join(tmpDir, "tmp"),
			},
			Shared: pathmeta.SharedPaths{
				Root: tmpDir,
			},
			Working: pathmeta.WorkingPaths{
				Directory:     tmpDir,
				SessionSdkDir: filepath.Join(tmpDir, "tmp", "session"),
				GlobalSdkDir:  filepath.Join(tmpDir, "sdk"),
			},
		},
		CurrentWorkingDir: tmpDir,
	}
}
