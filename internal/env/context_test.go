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
	"runtime"
	"strings"
	"testing"

	"github.com/version-fox/vfox/internal/config"
	"github.com/version-fox/vfox/internal/pathmeta"
)

func TestSplitSystemPaths(t *testing.T) {
	// Skip on Windows as path separator is different
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows due to path separator differences")
	}

	tests := []struct {
		name           string
		pathEnv        string
		wantPrefix     []string
		wantClean      []string
	}{
		{
			name: "virtualenv before vfox paths",
			pathEnv: strings.Join([]string{
				"/home/user/venv/bin",                           // virtualenv - should be in prefix
				"/home/user/.version-fox/sdks/python/bin",       // vfox path - should be removed
				"/home/user/.version-fox/sdks/nodejs/bin",       // vfox path - should be removed
				"/usr/local/bin",                                // system path - should be in clean
				"/usr/bin",                                      // system path - should be in clean
			}, ":"),
			wantPrefix: []string{"/home/user/venv/bin"},
			wantClean:  []string{"/usr/local/bin", "/usr/bin"},
		},
		{
			name: "multiple paths before vfox",
			pathEnv: strings.Join([]string{
				"/home/user/venv/bin",                           // user path - prefix
				"/home/user/.local/bin",                         // user path - prefix
				"/home/user/.version-fox/sdks/python/bin",       // vfox path - removed
				"/usr/bin",                                      // system path - clean
			}, ":"),
			wantPrefix: []string{"/home/user/venv/bin", "/home/user/.local/bin"},
			wantClean:  []string{"/usr/bin"},
		},
		{
			name: "no paths before vfox",
			pathEnv: strings.Join([]string{
				"/home/user/.version-fox/sdks/python/bin",       // vfox path - removed
				"/usr/local/bin",                                // system path - clean
				"/usr/bin",                                      // system path - clean
			}, ":"),
			wantPrefix: []string{},
			wantClean:  []string{"/usr/local/bin", "/usr/bin"},
		},
		{
			name: "no vfox paths at all",
			pathEnv: strings.Join([]string{
				"/home/user/venv/bin",
				"/usr/local/bin",
				"/usr/bin",
			}, ":"),
			wantPrefix: []string{"/home/user/venv/bin", "/usr/local/bin", "/usr/bin"},
			wantClean:  []string{},
		},
		{
			name: "project vfox path (.vfox directory)",
			pathEnv: strings.Join([]string{
				"/home/user/venv/bin",                           // virtualenv - prefix
				"/home/user/project/.vfox/sdks/python/bin",      // project vfox path - removed
				"/usr/bin",                                      // system path - clean
			}, ":"),
			wantPrefix: []string{"/home/user/venv/bin"},
			wantClean:  []string{"/usr/bin"},
		},
		{
			name: "empty PATH",
			pathEnv: "",
			wantPrefix: []string{},
			wantClean:  []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original PATH and restore after test
			origPath := os.Getenv("PATH")
			defer os.Setenv("PATH", origPath)

			// Set test PATH
			os.Setenv("PATH", tt.pathEnv)

			// Create minimal RuntimeEnvContext
			ctx := &RuntimeEnvContext{
				UserConfig: &config.Config{},
				PathMeta:   &pathmeta.PathMeta{},
			}

			prefixPaths, cleanPaths := ctx.SplitSystemPaths()

			// Verify prefix paths
			gotPrefix := prefixPaths.Slice()
			if len(gotPrefix) != len(tt.wantPrefix) {
				t.Errorf("prefixPaths length = %d, want %d\ngot: %v\nwant: %v",
					len(gotPrefix), len(tt.wantPrefix), gotPrefix, tt.wantPrefix)
			} else {
				for i, want := range tt.wantPrefix {
					if gotPrefix[i] != want {
						t.Errorf("prefixPaths[%d] = %q, want %q", i, gotPrefix[i], want)
					}
				}
			}

			// Verify clean paths
			gotClean := cleanPaths.Slice()
			if len(gotClean) != len(tt.wantClean) {
				t.Errorf("cleanPaths length = %d, want %d\ngot: %v\nwant: %v",
					len(gotClean), len(tt.wantClean), gotClean, tt.wantClean)
			} else {
				for i, want := range tt.wantClean {
					if gotClean[i] != want {
						t.Errorf("cleanPaths[%d] = %q, want %q", i, gotClean[i], want)
					}
				}
			}
		})
	}
}

func TestSplitSystemPaths_VirtualenvPriority(t *testing.T) {
	// This test specifically verifies the fix for GitHub issue #622
	// When a user activates a Python virtualenv after vfox has set up the environment,
	// the virtualenv path should maintain highest priority after vfox re-evaluates PATH
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows due to path separator differences")
	}

	// Save original PATH
	origPath := os.Getenv("PATH")
	defer os.Setenv("PATH", origPath)

	// Simulate: user activated virtualenv, then vfox hook runs
	// The virtualenv path appears before vfox paths
	testPath := strings.Join([]string{
		"/root/venv/bin",                              // virtualenv activated by user
		"/root/.version-fox/sdks/python/bin",          // vfox global python
		"/root/.version-fox/sdks/protobuf/bin",        // vfox global protobuf
		"/root/.local/bin",                            // user local bin
		"/usr/local/bin",                              // system paths
		"/usr/bin",
	}, ":")
	os.Setenv("PATH", testPath)

	ctx := &RuntimeEnvContext{
		UserConfig: &config.Config{},
		PathMeta:   &pathmeta.PathMeta{},
	}

	prefixPaths, cleanPaths := ctx.SplitSystemPaths()

	// The virtualenv path should be captured as prefix (highest priority)
	gotPrefix := prefixPaths.Slice()
	if len(gotPrefix) != 1 || gotPrefix[0] != "/root/venv/bin" {
		t.Errorf("Expected virtualenv path in prefix, got: %v", gotPrefix)
	}

	// System paths should be in clean paths
	gotClean := cleanPaths.Slice()
	expectedClean := []string{"/root/.local/bin", "/usr/local/bin", "/usr/bin"}
	if len(gotClean) != len(expectedClean) {
		t.Errorf("cleanPaths = %v, want %v", gotClean, expectedClean)
	}

	// Verify vfox paths are excluded from both
	allPaths := append(gotPrefix, gotClean...)
	for _, p := range allPaths {
		if strings.Contains(p, ".version-fox") {
			t.Errorf("vfox path should be excluded: %s", p)
		}
	}
}

func TestCleanSystemPaths_BackwardCompatibility(t *testing.T) {
	// Ensure CleanSystemPaths still works as before for backward compatibility
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows due to path separator differences")
	}

	origPath := os.Getenv("PATH")
	defer os.Setenv("PATH", origPath)

	testPath := strings.Join([]string{
		"/home/user/.version-fox/sdks/python/bin",
		"/usr/local/bin",
		"/usr/bin",
	}, ":")
	os.Setenv("PATH", testPath)

	ctx := &RuntimeEnvContext{
		UserConfig: &config.Config{},
		PathMeta:   &pathmeta.PathMeta{},
	}

	cleanPaths := ctx.CleanSystemPaths()
	got := cleanPaths.Slice()

	// CleanSystemPaths should return all non-vfox paths
	// When no paths before vfox, prefix is empty, so clean gets everything
	expected := []string{"/usr/local/bin", "/usr/bin"}
	if len(got) != len(expected) {
		t.Errorf("CleanSystemPaths() = %v, want %v", got, expected)
	}
}
