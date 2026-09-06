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

package fs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	lua "github.com/yuin/gopher-lua"
)

func runLua(t *testing.T, root, script string) {
	t.Helper()
	L := lua.NewState()
	defer L.Close()
	Preload(L, root)
	if err := L.DoString(script); err != nil {
		t.Fatal(err)
	}
}

func writeFile(t *testing.T, path, content string, mode os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), mode); err != nil {
		t.Fatal(err)
	}
}

func assertFileContent(t *testing.T, path, want string) {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != want {
		t.Fatalf("content of %s = %q, want %q", path, content, want)
	}
}

func assertNotExist(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Lstat(path); !os.IsNotExist(err) {
		t.Fatalf("%s still exists or stat failed unexpectedly: %v", path, err)
	}
}

func TestRequire(t *testing.T) {
	runLua(t, t.TempDir(), `
		local fs = require("fs")
		assert(type(fs) == "table")
		assert(type(fs.copy) == "function")
		assert(type(fs.remove) == "function")
		assert(type(fs.move) == "function")
		assert(type(fs.symlink) == "function")
		assert(fs.read == nil)
		assert(fs.write == nil)
		assert(pcall(require, "file") == false)
	`)
}

func TestCopyFilePreservesContentAndMode(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source.txt")
	destination := filepath.Join(root, "destination.txt")
	writeFile(t, source, "vfox", 0o700)

	runLua(t, root, `assert(require("fs").copy("source.txt", "destination.txt") == true)`)

	assertFileContent(t, destination, "vfox")
	sourceInfo, err := os.Stat(source)
	if err != nil {
		t.Fatal(err)
	}
	destinationInfo, err := os.Stat(destination)
	if err != nil {
		t.Fatal(err)
	}
	if destinationInfo.Mode().Perm() != sourceInfo.Mode().Perm() {
		t.Fatalf("copied file mode = %v, want %v", destinationInfo.Mode().Perm(), sourceInfo.Mode().Perm())
	}
}

func TestCopyDirectoryRecursively(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "source", "nested", "content.txt"), "nested", 0o600)
	if err := os.MkdirAll(filepath.Join(root, "source", "empty"), 0o755); err != nil {
		t.Fatal(err)
	}

	runLua(t, root, `assert(require("fs").copy("source", "destination") == true)`)

	assertFileContent(t, filepath.Join(root, "destination", "nested", "content.txt"), "nested")
	if info, err := os.Stat(filepath.Join(root, "destination", "empty")); err != nil || !info.IsDir() {
		t.Fatalf("empty directory was not copied: %v", err)
	}
}

func TestRemoveFile(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "remove.txt")
	writeFile(t, path, "remove", 0o600)

	runLua(t, root, `assert(require("fs").remove("remove.txt") == true)`)
	assertNotExist(t, path)
}

func TestRemoveDirectoryRecursively(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "remove-dir")
	writeFile(t, filepath.Join(path, "nested", "content.txt"), "remove", 0o600)

	runLua(t, root, `assert(require("fs").remove("remove-dir") == true)`)
	assertNotExist(t, path)
}

func TestRemoveDirectorySymlinkKeepsTarget(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "target")
	link := filepath.Join(root, "target-link")
	writeFile(t, filepath.Join(target, "content.txt"), "keep", 0o600)
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}

	runLua(t, root, `assert(require("fs").remove("target-link") == true)`)

	assertNotExist(t, link)
	assertFileContent(t, filepath.Join(target, "content.txt"), "keep")
}

func TestMoveRenamesFile(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "before.txt"), "rename", 0o600)

	runLua(t, root, `assert(require("fs").move("before.txt", "after.txt") == true)`)

	assertNotExist(t, filepath.Join(root, "before.txt"))
	assertFileContent(t, filepath.Join(root, "after.txt"), "rename")
}

func TestMoveRenamesDirectory(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "before", "nested", "content.txt"), "rename", 0o600)

	runLua(t, root, `assert(require("fs").move("before", "after") == true)`)

	assertNotExist(t, filepath.Join(root, "before"))
	assertFileContent(t, filepath.Join(root, "after", "nested", "content.txt"), "rename")
}

func TestMoveIntoExistingDirectory(t *testing.T) {
	tests := []struct {
		name        string
		sourcePath  string
		moveSource  string
		content     string
		destination string
	}{
		{name: "file", sourcePath: "item.txt", moveSource: "item.txt", content: "file", destination: filepath.Join("destination", "item.txt")},
		{name: "directory", sourcePath: filepath.Join("item", "nested.txt"), moveSource: "item", content: "directory", destination: filepath.Join("destination", "item", "nested.txt")},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			writeFile(t, filepath.Join(root, test.sourcePath), test.content, 0o600)
			if err := os.Mkdir(filepath.Join(root, "destination"), 0o755); err != nil {
				t.Fatal(err)
			}

			runLua(t, root, `assert(require("fs").move("`+test.moveSource+`", "destination") == true)`)

			assertFileContent(t, filepath.Join(root, test.destination), test.content)
		})
	}
}

func TestSymlinkFileAndDirectory(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "target.txt"), "file", 0o600)
	writeFile(t, filepath.Join(root, "target-dir", "content.txt"), "directory", 0o600)

	runLua(t, root, `
		local fs = require("fs")
		assert(fs.symlink("target.txt", "file-link") == true)
		assert(fs.symlink("target-dir", "dir-link") == true)
	`)

	assertFileContent(t, filepath.Join(root, "file-link"), "file")
	assertFileContent(t, filepath.Join(root, "dir-link", "content.txt"), "directory")
}

func TestOperationsReportErrors(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "already-exists"), nil, 0o600); err != nil {
		t.Fatal(err)
	}

	runLua(t, root, `
		local fs = require("fs")
		local cases = {
			{fs.copy, "missing", "copy"},
			{fs.copy, "already-exists", "missing/copy"},
			{fs.remove, "missing"},
			{fs.move, "missing", "move"},
			{fs.symlink, "target", "already-exists"},
		}
		for _, case in ipairs(cases) do
			local ok, err = pcall(case[1], case[2], case[3])
			assert(ok == false)
			assert(type(err) == "string")
		end
	`)
}

func TestOperationsValidateArguments(t *testing.T) {
	L := lua.NewState()
	defer L.Close()
	Preload(L, t.TempDir())

	for _, operation := range []string{"copy", "remove", "move", "symlink"} {
		err := L.DoString(`require("fs").` + operation + `(nil, "link")`)
		if err == nil {
			t.Fatalf("%s() with a non-string path returned no error", operation)
		}
		if !strings.Contains(err.Error(), "string expected") {
			t.Fatalf("%s() error = %q, want an argument type error", operation, err)
		}
	}
	for _, operation := range []string{"copy", "move", "symlink"} {
		err := L.DoString(`require("fs").` + operation + `("source", nil)`)
		if err == nil || !strings.Contains(err.Error(), "string expected") {
			t.Fatalf("%s() accepted a non-string destination: %v", operation, err)
		}
	}
}

func TestSymlinkRelativeSourceInSubdirectory(t *testing.T) {
	root := t.TempDir()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(cwd); err != nil {
			t.Error(err)
		}
	})
	writeFile(t, "source.txt", "content", 0600)
	if err := os.Mkdir("links", 0755); err != nil {
		t.Fatal(err)
	}
	runLua(t, "", `assert(require("fs").symlink("source.txt", "links/link.txt"))`)
	assertFileContent(t, "links/link.txt", "content")
}

func TestCopyDirectoryPreservesSymlinks(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "source", "actual"), "content", 0700)
	if err := os.Symlink("actual", filepath.Join(root, "source", "link")); err != nil {
		t.Fatal(err)
	}
	runLua(t, root, `assert(require("fs").copy("source", "destination"))`)
	assertFileContent(t, filepath.Join(root, "destination", "link"), "content")
	target, err := os.Readlink(filepath.Join(root, "destination", "link"))
	if err != nil {
		t.Fatal(err)
	}
	if target != "actual" {
		t.Fatalf("link target = %q, want actual", target)
	}

}

func TestCopyDirectorySymlinkSourceAndNestedLinks(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "source", "nested", "data"), "keep", 0700)
	for name, target := range map[string]string{"dir-link": "nested", "dangling": "missing"} {
		if err := os.Symlink(target, filepath.Join(root, "source", name)); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink("source", filepath.Join(root, "source-link")); err != nil {
		t.Fatal(err)
	}
	runLua(t, root, `assert(require("fs").copy("source-link", "destination"))`)
	for name, want := range map[string]string{"dir-link": "nested", "dangling": "missing"} {
		got, err := os.Readlink(filepath.Join(root, "destination", name))
		if err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Fatalf("%s target = %q, want %q", name, got, want)
		}
	}
	assertFileContent(t, filepath.Join(root, "destination", "nested", "data"), "keep")
}

func TestCopyDirectoryDoesNotOverwriteFiles(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "source", "data"), "new", 0600)
	writeFile(t, filepath.Join(root, "destination", "data"), "keep", 0600)
	runLua(t, root, `assert(not pcall(require("fs").copy, "source", "destination"))`)
	assertFileContent(t, filepath.Join(root, "destination", "data"), "keep")
}
