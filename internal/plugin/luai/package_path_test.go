// Copyright 2026 vfox project contributors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package luai

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	lua "github.com/yuin/gopher-lua"
)

// A plugin lives under HOME, and Lua's package.path cannot hold a directory
// whose name contains ';' (its separator) or '?' (its module placeholder), so
// require failed for such a HOME (#821).
func TestLimitPackagePathLoadsModulesFromAnyDirectoryName(t *testing.T) {
	names := []string{"plain", "semi;colon"}
	if runtime.GOOS != "windows" { // '?' is not allowed in a Windows file name
		names = append(names, "question?mark")
	}
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			hooks := filepath.Join(t.TempDir(), name, "hooks")
			lib := filepath.Join(filepath.Dir(hooks), "lib")
			for _, dir := range []string{hooks, filepath.Join(lib, "nested")} {
				if err := os.MkdirAll(dir, 0o755); err != nil {
					t.Fatal(err)
				}
			}
			write := func(path, body string) {
				if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			write(filepath.Join(hooks, "util.lua"), "return 'hooks'")
			write(filepath.Join(lib, "util.lua"), "return 'lib'")
			write(filepath.Join(lib, "only_lib.lua"), "return 'lib only'")
			write(filepath.Join(lib, "nested", "mod.lua"), "return 'nested'")

			vm := NewLuaVM()
			defer vm.Instance.Close()
			vm.LimitPackagePath(hooks, lib)

			for module, want := range map[string]string{
				"util":       "hooks", // the first directory wins
				"only_lib":   "lib only",
				"nested.mod": "nested",
			} {
				if err := vm.Instance.DoString("result = require('" + module + "')"); err != nil {
					t.Fatalf("require(%q): %v", module, err)
				}
				if got := vm.Instance.GetGlobal("result"); got != lua.LString(want) {
					t.Errorf("require(%q) = %v, want %q", module, got, want)
				}
			}
		})
	}
}

func TestLimitPackagePathNamesTheFilesItTriedForAMissingModule(t *testing.T) {
	dir := t.TempDir()
	vm := NewLuaVM()
	defer vm.Instance.Close()
	vm.LimitPackagePath(dir)

	err := vm.Instance.DoString("require('missing')")
	if err == nil {
		t.Fatal("require of a missing module succeeded")
	}
	if want := filepath.Join(dir, "missing.lua"); !strings.Contains(err.Error(), want) {
		t.Errorf("error %q does not name %q", err, want)
	}
}
