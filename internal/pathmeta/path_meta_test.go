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
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/version-fox/vfox/internal/shared/logger"
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
	t.Run("creates immediate shared subdirs only when shared root already exists", func(t *testing.T) {
		root := t.TempDir()
		existingRoot := filepath.Join(root, "existing")
		if err := os.MkdirAll(existingRoot, 0755); err != nil {
			t.Fatal(err)
		}
		meta := NewPathMeta(filepath.Join(root, "home"), existingRoot, root, 4242)
		for _, dir := range []string{meta.Shared.Installs, meta.Shared.Plugins} {
			info, statErr := os.Stat(dir)
			if statErr != nil || !info.IsDir() {
				t.Fatalf("expected directory %q to exist under existing root (stat err=%v)", dir, statErr)
			}
		}
	})

	t.Run("does not log failures at debug level when shared subdirectories already exist", func(t *testing.T) {
		root := t.TempDir()
		home := filepath.Join(root, "home")
		existingRoot := filepath.Join(root, "existing")
		for _, dir := range []string{
			filepath.Join(existingRoot, installedDirPrefix),
			filepath.Join(existingRoot, pluginDirPrefix),
		} {
			if err := os.MkdirAll(dir, 0755); err != nil {
				t.Fatal(err)
			}
		}
		t.Setenv(HookCurTmpPath, t.TempDir())

		logger.SetLevel(logger.DebugLevel)
		t.Cleanup(func() { logger.SetLevel(logger.InfoLevel) })

		origStdout := os.Stdout
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatal(err)
		}
		os.Stdout = w
		NewPathMeta(home, existingRoot, root, 4242)
		_ = w.Close()
		os.Stdout = origStdout

		var out bytes.Buffer
		_, _ = io.Copy(&out, r)
		_ = r.Close()

		if strings.Contains(out.String(), "failed") {
			t.Fatalf("unexpected failure log at debug level when shared subdirs exist: %q", out.String())
		}
	})

	t.Run("does not log shared directory failure at debug level when shared root does not exist", func(t *testing.T) {
		root := t.TempDir()
		home := filepath.Join(root, "home")
		missingShared := filepath.Join(root, "no-such-dir", "vfox-home")
		t.Setenv(HookCurTmpPath, t.TempDir())

		logger.SetLevel(logger.DebugLevel)
		t.Cleanup(func() { logger.SetLevel(logger.InfoLevel) })

		origStdout, origStderr := os.Stdout, os.Stderr
		rOut, wOut, err := os.Pipe()
		if err != nil {
			t.Fatal(err)
		}
		rErr, wErr, err := os.Pipe()
		if err != nil {
			t.Fatal(err)
		}
		os.Stdout, os.Stderr = wOut, wErr
		NewPathMeta(home, missingShared, root, 4242)
		_ = wOut.Close()
		_ = wErr.Close()
		os.Stdout, os.Stderr = origStdout, origStderr

		var out, errBuf bytes.Buffer
		_, _ = io.Copy(&out, rOut)
		_, _ = io.Copy(&errBuf, rErr)
		_ = rOut.Close()
		_ = rErr.Close()

		if errBuf.Len() != 0 {
			t.Fatalf("NewPathMeta wrote unexpected stderr at debug level: %q", errBuf.String())
		}
		if strings.Contains(out.String(), "create shared directory") || strings.Contains(out.String(), "failed") {
			t.Fatalf("unexpected failure log at debug level for missing Shared.Root: %q", out.String())
		}
		if _, statErr := os.Stat(filepath.Join(root, "no-such-dir")); !os.IsNotExist(statErr) {
			t.Fatalf("expected missing Shared.Root parent not to be created, stat err = %v", statErr)
		}
	})

	t.Run("does not create missing shared root or parent trees on read paths", func(t *testing.T) {
		for _, tc := range []struct {
			name      string
			rel       []string
			checkPath string
		}{
			{name: "missing directory", rel: []string{"vfox-home"}, checkPath: "vfox-home"},
			{name: "missing parent directories", rel: []string{"no-such-dir", "vfox-home"}, checkPath: "no-such-dir"},
		} {
			t.Run(tc.name, func(t *testing.T) {
				root := t.TempDir()
				shared := filepath.Join(append([]string{root}, tc.rel...)...)

				meta := NewPathMeta(filepath.Join(root, "home"), shared, root, 4242)
				if meta == nil || meta.Shared.Root != shared {
					t.Fatalf("unexpected PathMeta: %+v", meta)
				}
				if _, statErr := os.Stat(filepath.Join(root, tc.checkPath)); !os.IsNotExist(statErr) {
					t.Fatalf("expected %q not to be created on read path, stat err = %v", tc.checkPath, statErr)
				}
			})
		}
	})

	t.Run("keeps stdout and stderr clean on normal runs and logs diagnostics only at debug level", func(t *testing.T) {
		for _, tc := range []struct {
			name        string
			blockTemp   bool
			activated   bool
			failExe     bool
			wantDebug   string
			wantInitErr bool
		}{
			{name: "blocked GlobalSdkDir and missing SharedRoot", wantDebug: "create global sdk dir failed:", wantInitErr: true},
			{name: "blocked session temp directory", blockTemp: true, wantDebug: "create temp dir failed:", wantInitErr: true},
			{name: "blocked session temp directory in activated shell", blockTemp: true, activated: true, wantDebug: "create temp dir failed:", wantInitErr: true},
			{name: "executable lookup failure", failExe: true, wantDebug: "get executable path failed:", wantInitErr: true},
		} {
			t.Run(tc.name, func(t *testing.T) {
				root := t.TempDir()
				home := filepath.Join(root, "home")
				userVfox := filepath.Join(home, ".vfox")
				if err := os.MkdirAll(userVfox, 0755); err != nil {
					t.Fatal(err)
				}
				if tc.failExe {
					origExe := Executable
					Executable = func() (string, error) {
						return "", errors.New("readlink /proc/self/exe: no such file or directory")
					}
					t.Cleanup(func() { Executable = origExe })
				}
				// Block GlobalSdkDir and SharedRoot parent with regular files so directory creation fails.
				if err := os.WriteFile(filepath.Join(userVfox, "sdks"), []byte("file"), 0644); err != nil {
					t.Fatal(err)
				}
				if tc.blockTemp {
					if err := os.WriteFile(filepath.Join(userVfox, "tmp"), []byte("file"), 0644); err != nil {
						t.Fatal(err)
					}
					if tc.activated {
						t.Setenv(HookCurTmpPath, filepath.Join(userVfox, "tmp", "20261124-4242"))
					} else {
						t.Setenv(HookCurTmpPath, "")
					}
				} else {
					t.Setenv(HookCurTmpPath, t.TempDir())
				}
				blocker := filepath.Join(root, "not-a-dir")
				if err := os.WriteFile(blocker, []byte("file"), 0644); err != nil {
					t.Fatal(err)
				}
				shared := filepath.Join(blocker, "vfox-home")

				captureNewPathMeta := func(level logger.LoggerLevel) (meta *PathMeta, stdout, stderr string) {
					logger.SetLevel(level)
					t.Cleanup(func() { logger.SetLevel(logger.InfoLevel) })

					origStdout, origStderr := os.Stdout, os.Stderr
					rOut, wOut, err := os.Pipe()
					if err != nil {
						t.Fatal(err)
					}
					rErr, wErr, err := os.Pipe()
					if err != nil {
						t.Fatal(err)
					}
					os.Stdout, os.Stderr = wOut, wErr
					meta = NewPathMeta(home, shared, root, 4242)
					_ = wOut.Close()
					_ = wErr.Close()
					os.Stdout, os.Stderr = origStdout, origStderr

					var out, errBuf bytes.Buffer
					_, _ = io.Copy(&out, rOut)
					_, _ = io.Copy(&errBuf, rErr)
					_ = rOut.Close()
					_ = rErr.Close()
					return meta, out.String(), errBuf.String()
				}

				meta, outNormal, errNormal := captureNewPathMeta(logger.InfoLevel)
				if meta == nil || meta.Shared.Root != shared {
					t.Fatalf("unexpected PathMeta: %+v", meta)
				}
				if tc.failExe && meta.Executable != "" {
					t.Fatalf("expected empty Executable on error, got %q", meta.Executable)
				}
				if tc.wantInitErr && len(meta.InitErrors) == 0 {
					t.Fatalf("expected InitErrors to record %s", tc.name)
				}
				if outNormal != "" {
					t.Fatalf("NewPathMeta wrote to stdout at InfoLevel: %q", outNormal)
				}
				if errNormal != "" {
					t.Fatalf("NewPathMeta wrote to stderr at InfoLevel: %q", errNormal)
				}

				_, outDebug, errDebug := captureNewPathMeta(logger.DebugLevel)
				if errDebug != "" {
					t.Fatalf("NewPathMeta wrote unexpected stderr at DebugLevel: %q", errDebug)
				}
				if !strings.Contains(outDebug, tc.wantDebug) {
					t.Fatalf("NewPathMeta debug output = %q, want substring %q", outDebug, tc.wantDebug)
				}
				if tc.blockTemp && strings.Count(outDebug, tc.wantDebug) != 1 {
					t.Fatalf("NewPathMeta logged duplicate temp dir failure at DebugLevel: %q", outDebug)
				}
				if strings.Contains(outDebug, "create shared directory") {
					t.Fatalf("NewPathMeta logged shared directory failure when Shared.Root does not exist: %q", outDebug)
				}
			})
		}
	})
}
