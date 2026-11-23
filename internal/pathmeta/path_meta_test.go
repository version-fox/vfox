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
	"bytes"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestIsVfoxRelatedPath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{
			name:     "absolute path with .vfox directory",
			path:     "/home/user/project/.vfox/sdks/nodejs/bin",
			expected: true,
		},
		{
			name:     "absolute path with .version-fox directory",
			path:     "/home/user/project/.version-fox/sdks/nodejs/bin",
			expected: true,
		},
		{
			name:     "relative path with .vfox directory",
			path:     "project/.vfox/sdks/nodejs/bin",
			expected: true,
		},
		{
			name:     "absolute path without .vfox",
			path:     "/usr/local/bin",
			expected: false,
		},
		{
			name:     "path with similar but not .vfox name",
			path:     "/home/user/project/myvfox/bin",
			expected: false,
		},
		{
			name:     "path ending with .vfox",
			path:     "/home/user/project/.vfox",
			expected: true,
		},
		{
			name:     "path starting with .vfox",
			path:     ".vfox/sdks/nodejs/bin",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsVfoxRelatedPath(tt.path)
			if result != tt.expected {
				t.Errorf("IsVfoxRelatedPath(%q) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestIsVfoxRelatedPath_Windows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Skipping Windows-specific test on non-Windows platform")
	}

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{
			name:     "Windows path with .vfox directory",
			path:     "C:\\Users\\user\\project\\.vfox\\sdks\\nodejs\\bin",
			expected: true,
		},
		{
			name:     "Windows path without .vfox",
			path:     "C:\\Program Files\\Nodejs",
			expected: false,
		},
		{
			name:     "Windows path with .version-fox",
			path:     "C:\\Users\\user\\project\\.version-fox\\sdks",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsVfoxRelatedPath(tt.path)
			if result != tt.expected {
				t.Errorf("IsVfoxRelatedPath(%q) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestContainsPathSegment(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		part     string
		expected bool
	}{
		{
			name:     "path contains segment",
			path:     "/home/user/.vfox/sdks",
			part:     ".vfox",
			expected: true,
		},
		{
			name:     "path does not contain segment",
			path:     "/home/user/project/sdks",
			part:     ".vfox",
			expected: false,
		},
		{
			name:     "exact match",
			path:     ".vfox",
			part:     ".vfox",
			expected: true,
		},
		{
			name:     "relative path contains segment",
			path:     "project/.vfox/sdks",
			part:     ".vfox",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := containsPathSegment(tt.path, tt.part)
			if result != tt.expected {
				t.Errorf("containsPathSegment(%q, %q) = %v, want %v", tt.path, tt.part, result, tt.expected)
			}
		})
	}
}

func TestApplyStoragePath(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()

	// Create a PathMeta instance with default paths
	meta := &PathMeta{
		Shared: SharedPaths{
			Root:     tmpDir,
			Installs: filepath.Join(tmpDir, "default", "cache"),
		},
	}

	t.Run("Empty storage path does not change Installs", func(t *testing.T) {
		originalInstalls := meta.Shared.Installs
		err := meta.ApplyStoragePath("")
		if err != nil {
			t.Errorf("ApplyStoragePath with empty path should not error: %v", err)
		}
		if meta.Shared.Installs != originalInstalls {
			t.Errorf("Installs path changed when it shouldn't: got %q, want %q", meta.Shared.Installs, originalInstalls)
		}
	})

	t.Run("Valid storage path updates Installs", func(t *testing.T) {
		customPath := filepath.Join(tmpDir, "custom")

		// Now test ApplyStoragePath
		err := meta.ApplyStoragePath(customPath)
		if err != nil {
			t.Errorf("ApplyStoragePath with valid path should not error: %v", err)
		}
		expectedPath := customPath
		if meta.Shared.Installs != expectedPath {
			t.Errorf("Installs path not updated correctly: got %q, want %q", meta.Shared.Installs, expectedPath)
		}
	})
}

func TestNewPathMetaSharedDirectories(t *testing.T) {
	t.Run("creates missing shared root and nested parents", func(t *testing.T) {
		for _, tc := range []struct {
			name string
			rel  []string
		}{
			{name: "single missing directory", rel: []string{"vfox-home"}},
			{name: "missing parent directories", rel: []string{"no-such-dir", "vfox-home"}},
		} {
			t.Run(tc.name, func(t *testing.T) {
				root := t.TempDir()
				shared := filepath.Join(append([]string{root}, tc.rel...)...)

				meta, err := NewPathMeta(filepath.Join(root, "home"), shared, root, 4242)
				if err != nil {
					t.Fatalf("NewPathMeta returned error: %v", err)
				}
				for _, dir := range []string{meta.Shared.Installs, meta.Shared.Plugins} {
					info, statErr := os.Stat(dir)
					if statErr != nil || !info.IsDir() {
						t.Fatalf("expected directory %q to exist (stat err=%v)", dir, statErr)
					}
				}
			})
		}
	})

	t.Run("stays best-effort and silent on stdout when directories cannot be created", func(t *testing.T) {
		for _, tc := range []struct {
			name      string
			blockTemp bool
		}{
			{name: "blocked GlobalSdkDir and SharedRoot"},
			{name: "blocked session temp directory", blockTemp: true},
		} {
			t.Run(tc.name, func(t *testing.T) {
				root := t.TempDir()
				home := filepath.Join(root, "home")
				userVfox := filepath.Join(home, ".vfox")
				if err := os.MkdirAll(userVfox, 0755); err != nil {
					t.Fatal(err)
				}
				// Block GlobalSdkDir and SharedRoot with regular files so directory creation fails.
				if err := os.WriteFile(filepath.Join(userVfox, "sdks"), []byte("file"), 0644); err != nil {
					t.Fatal(err)
				}
				if tc.blockTemp {
					if err := os.WriteFile(filepath.Join(userVfox, "tmp"), []byte("file"), 0644); err != nil {
						t.Fatal(err)
					}
					t.Setenv(HookCurTmpPath, "")
				} else {
					t.Setenv(HookCurTmpPath, t.TempDir())
				}
				blocker := filepath.Join(root, "not-a-dir")
				if err := os.WriteFile(blocker, []byte("file"), 0644); err != nil {
					t.Fatal(err)
				}
				shared := filepath.Join(blocker, "vfox-home")

				origStdout := os.Stdout
				r, w, err := os.Pipe()
				if err != nil {
					t.Fatal(err)
				}
				os.Stdout = w
				meta, metaErr := NewPathMeta(home, shared, root, 4242)
				_ = w.Close()
				os.Stdout = origStdout

				var out bytes.Buffer
				_, _ = io.Copy(&out, r)
				_ = r.Close()

				if metaErr != nil {
					t.Fatalf("NewPathMeta returned unexpected error: %v", metaErr)
				}
				if meta == nil || meta.Shared.Root != shared {
					t.Fatalf("unexpected PathMeta: %+v", meta)
				}
				if out.Len() != 0 {
					t.Fatalf("NewPathMeta wrote to stdout when directory creation failed: %q", out.String())
				}
			})
		}
	})
}
