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

package file

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	lua "github.com/yuin/gopher-lua"
)

func TestPreload(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "target.txt")
	if err := os.WriteFile(target, []byte("vfox"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, dir := range []string{"source-dir/nested", "delete-dir/nested"} {
		if err := os.MkdirAll(filepath.Join(root, filepath.FromSlash(dir)), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "source-dir", "nested", "content.txt"), []byte("directory"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "delete-dir", "nested", "content.txt"), []byte("delete"), 0o600); err != nil {
		t.Fatal(err)
	}

	L := lua.NewState()
	defer L.Close()
	Preload(L, root)

	script := `
		local file = require("file")
		assert(type(file) == "table")
		assert(type(file.copy) == "function")
		assert(type(file.remove) == "function")
		assert(type(file.move) == "function")
		assert(type(file.symlink) == "function")
		assert(file.read == nil)
		assert(file.write == nil)

		assert(file.copy("target.txt", "copied.txt") == true)
		assert(file.copy("target.txt", "preserved-mode.txt") == true)
		assert(file.move("copied.txt", "moved.txt") == true)
		assert(file.symlink("target.txt", "target-link.txt") == true)
		assert(file.remove("moved.txt") == true)

		assert(file.copy("source-dir", "copied-dir") == true)
		assert(file.move("copied-dir", "moved-dir") == true)
		assert(file.symlink("source-dir", "source-dir-link") == true)
		assert(file.remove("delete-dir") == true)
	`
	if err := L.DoString(script); err != nil {
		t.Fatal(err)
	}

	content, err := os.ReadFile(filepath.Join(root, "target-link.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "vfox" {
		t.Fatalf("linked file content = %q, want %q", content, "vfox")
	}
	sourceInfo, err := os.Stat(target)
	if err != nil {
		t.Fatal(err)
	}
	copyInfo, err := os.Stat(filepath.Join(root, "preserved-mode.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if copyInfo.Mode().Perm() != sourceInfo.Mode().Perm() {
		t.Fatalf("copied file mode = %v, want %v", copyInfo.Mode().Perm(), sourceInfo.Mode().Perm())
	}
	if _, err := os.Stat(filepath.Join(root, "moved.txt")); !os.IsNotExist(err) {
		t.Fatalf("removed file still exists or stat failed unexpectedly: %v", err)
	}
	for _, path := range []string{
		filepath.Join(root, "moved-dir", "nested", "content.txt"),
		filepath.Join(root, "source-dir-link", "nested", "content.txt"),
	} {
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if string(content) != "directory" {
			t.Fatalf("directory file content = %q, want %q", content, "directory")
		}
	}
	if _, err := os.Stat(filepath.Join(root, "delete-dir")); !os.IsNotExist(err) {
		t.Fatalf("recursively removed directory still exists or stat failed unexpectedly: %v", err)
	}
}

func TestOperationsReportErrors(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "already-exists"), nil, 0o600); err != nil {
		t.Fatal(err)
	}

	L := lua.NewState()
	defer L.Close()
	Preload(L, root)

	script := `
		local file = require("file")
		local cases = {
			{file.copy, "missing", "copy"},
			{file.copy, "already-exists", "missing/copy"},
			{file.remove, "missing"},
			{file.move, "missing", "move"},
			{file.symlink, "target", "already-exists"},
		}
		for _, case in ipairs(cases) do
			local ok, err = pcall(case[1], case[2], case[3])
			assert(ok == false)
			assert(type(err) == "string")
		end
	`
	if err := L.DoString(script); err != nil {
		t.Fatal(err)
	}
}

func TestOperationsValidateArguments(t *testing.T) {
	L := lua.NewState()
	defer L.Close()
	Preload(L, t.TempDir())

	for _, operation := range []string{"copy", "remove", "move", "symlink"} {
		err := L.DoString(`require("file").` + operation + `(nil, "link")`)
		if err == nil {
			t.Fatalf("%s() with a non-string path returned no error", operation)
		}
		if !strings.Contains(err.Error(), "string expected") {
			t.Fatalf("%s() error = %q, want an argument type error", operation, err)
		}
	}
}
