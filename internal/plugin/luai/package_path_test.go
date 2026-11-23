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
	// gopher-lua puts "\n\t" in front of each loader's message, so a message
	// that starts with one more gave an empty indented line.
	if strings.Contains(err.Error(), "\n\t\n\t") {
		t.Errorf("error %q has an empty line", err)
	}
}

// A plugin may extend package.path at load time to require a vendored module.
// That worked when the search went through package.path, and must keep working.
func TestLimitPackagePathSearchesTheEntriesAPluginAddsToPackagePath(t *testing.T) {
	root := t.TempDir()
	own := filepath.Join(root, "hooks")
	vendor := filepath.Join(root, "vendor")
	for _, dir := range []string{own, vendor} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for path, body := range map[string]string{
		filepath.Join(vendor, "vendored_helper.lua"): "return 'vendored'",
		filepath.Join(vendor, "util.lua"):            "return 'vendor util'",
		filepath.Join(own, "util.lua"):               "return 'own util'",
	} {
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	vm := NewLuaVM()
	defer vm.Instance.Close()
	vm.LimitPackagePath(own)
	vm.Instance.SetGlobal("vendor_template", lua.LString(filepath.Join(vendor, "?.lua")))

	err := vm.Instance.DoString(`
		package.path = package.path .. ";" .. vendor_template
		helper = require("vendored_helper")
		util = require("util")
	`)
	if err != nil {
		t.Fatalf("require after extending package.path: %v", err)
	}
	if got := vm.Instance.GetGlobal("helper"); got != lua.LString("vendored") {
		t.Errorf("vendored_helper = %v, want %q", got, "vendored")
	}
	// The plugin's own directories still come first.
	if got := vm.Instance.GetGlobal("util"); got != lua.LString("own util") {
		t.Errorf("util = %v, want %q", got, "own util")
	}
}

// A file that cannot be read because of permissions is not a missing file. The
// error used to say "no file", which sent the reader looking for it.
func TestLimitPackagePathReportsAStatErrorThatIsNotNotFound(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("directory permissions do not block stat on Windows")
	}
	if os.Geteuid() == 0 {
		t.Skip("root ignores directory permissions")
	}
	hooks := filepath.Join(t.TempDir(), "hooks")
	if err := os.MkdirAll(hooks, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(hooks, "util.lua"), []byte("return 1"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(hooks, 0o600); err != nil { // no execute: stat inside fails
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(hooks, 0o755) })

	vm := NewLuaVM()
	defer vm.Instance.Close()
	vm.LimitPackagePath(hooks)

	err := vm.Instance.DoString("require('util')")
	if err == nil {
		t.Fatal("require through a directory without execute permission succeeded")
	}
	if !strings.Contains(err.Error(), "permission denied") {
		t.Errorf("error %q does not give the stat error", err)
	}
	if strings.Contains(err.Error(), "no file '"+filepath.Join(hooks, "util.lua")+"'") {
		t.Errorf("error %q calls a file that is there missing", err)
	}
}
