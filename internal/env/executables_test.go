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
	"reflect"
	"runtime"
	"sort"
	"testing"
)

// writeCommand creates something a shell would run from dir.
func writeCommand(t *testing.T, dir, name string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS == "windows" {
		name += ".cmd"
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("echo "+name), 0755); err != nil {
		t.Fatal(err)
	}
	return path
}

func commandNames(executables []Executable) []string {
	names := make([]string, 0, len(executables))
	for _, exe := range executables {
		names = append(names, exe.Name)
	}
	return names
}

func TestListExecutables(t *testing.T) {
	root := t.TempDir()
	first := filepath.Join(root, "first")
	second := filepath.Join(root, "second")
	writeCommand(t, first, "node")
	writeCommand(t, first, "npm")
	winner := writeCommand(t, second, "node") // same name, later directory
	writeCommand(t, second, "corepack")
	if err := os.WriteFile(filepath.Join(first, "README"), []byte("not a command"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(first, "subdir"), 0755); err != nil {
		t.Fatal(err)
	}

	got := ListExecutables([]string{first, filepath.Join(root, "missing"), second})
	names := commandNames(got)
	sort.Strings(names)
	// Not the README, not the subdirectory, and node only once.
	if want := []string{"corepack", "node", "npm"}; !reflect.DeepEqual(names, want) {
		t.Errorf("commands = %v, want %v", names, want)
	}
	for _, exe := range got {
		if exe.Name == "node" && exe.Dir != first {
			t.Errorf("node comes from %s, want the first directory in order", exe.Dir)
		}
		if exe.Path == winner {
			t.Error("a later directory must not override an earlier one")
		}
	}
}

func TestListExecutablesFollowsLinks(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks need privileges on Windows")
	}
	root := t.TempDir()
	target := writeCommand(t, filepath.Join(root, "real"), "tool")
	dir := filepath.Join(root, "bin")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(dir, "tool")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, "gone"), filepath.Join(dir, "dangling")); err != nil {
		t.Fatal(err)
	}
	got := ListExecutables([]string{dir})
	if len(got) != 1 || got[0].Name != "tool" {
		t.Errorf("commands = %+v, want only the working link", got)
	}
}

func TestWindowsCommandName(t *testing.T) {
	exts := []string{".ps1", ".COM", ".EXE", ".BAT", ".CMD"}
	tests := []struct {
		file, name string
		ok         bool
	}{
		{"node.exe", "node", true},
		{"NODE.EXE", "NODE", true},
		{"npm.cmd", "npm", true},
		{"npm.ps1", "npm", true},
		{"tool.sh", "", false},
		{"README.txt", "", false},
		{"noextension", "", false},
		{"archive.tar.exe", "archive.tar", true},
	}
	for _, tt := range tests {
		name, ok := windowsCommandName(tt.file, exts)
		if name != tt.name || ok != tt.ok {
			t.Errorf("windowsCommandName(%q) = (%q, %v), want (%q, %v)", tt.file, name, ok, tt.name, tt.ok)
		}
	}
}
