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
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/version-fox/vfox/internal/env"
)

// buildExecPath must join entries with the native path list separator no
// matter which shell hook is active. On Windows, env.Paths.String() emits a
// POSIX-style, ":"-joined PATH for bash/zsh hooks, meant for the hook shell;
// a process launched by exec cannot consume it. Splitting the exec PATH on
// os.PathListSeparator must always reproduce the original entries, so this
// check exercises the separator logic on every platform.
func TestBuildExecPathUsesNativeSeparator(t *testing.T) {
	for _, shell := range []string{"", "bash", "zsh"} {
		t.Run("shell="+shell, func(t *testing.T) {
			t.Setenv(env.HookFlag, shell)

			paths := env.NewPaths(env.EmptyPaths)
			first := filepath.Join(t.TempDir(), "one")
			second := filepath.Join(t.TempDir(), "two")
			paths.Add(first)
			paths.Add(second)

			got := strings.Split(buildExecPath(paths), string(os.PathListSeparator))
			if !slices.Equal(got, paths.Slice()) {
				t.Fatalf("buildExecPath = %q, want %q", got, paths.Slice())
			}
		})
	}
}
