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
	"testing"
)

func TestFindShadows(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses POSIX paths")
	}
	root := t.TempDir()
	sdkBin := filepath.Join(root, "sdk", "bin")
	pnpm := filepath.Join(root, "pnpm", "bin")
	system := filepath.Join(root, "system", "bin")
	venv := filepath.Join(root, "venv", "bin")
	writeCommand(t, sdkBin, "node")
	writeCommand(t, sdkBin, "npm")
	writeCommand(t, sdkBin, "npx")
	provided := ListExecutables([]string{sdkBin})

	pnpmNode := writeCommand(t, pnpm, "node")
	writeCommand(t, pnpm, "pnpm") // not provided by the SDK: not a shadow
	systemNode := writeCommand(t, system, "node")
	writeCommand(t, system, "npm")
	venvNpx := writeCommand(t, venv, "npx")

	policy := pathPolicy{activated: map[string]struct{}{normalizePathEntry(venv): {}}}

	got := findShadows(provided, []string{venv, pnpm, system}, policy)
	want := []Shadow{
		{Name: "node", Winner: pnpmNode, Dir: pnpm, Source: ShadowSourceUnknown},
		{Name: "npm", Winner: filepath.Join(system, "npm"), Dir: system, Source: ShadowSourceUnknown},
		{Name: "npx", Winner: venvNpx, Dir: venv, Source: ShadowSourceActivatedEnv},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("shadows = %+v\nwant     %+v", got, want)
	}
	for _, shadow := range got {
		if shadow.Winner == systemNode {
			t.Error("the first directory in PATH order must win: node is shadowed by pnpm's, not the system's")
		}
	}

	// Nothing to shadow, or nothing in front.
	if findShadows(nil, []string{pnpm}, policy) != nil || findShadows(provided, nil, policy) != nil {
		t.Error("no commands or no directories in front mean no shadows")
	}
}

// A directory in front of vfox's only shadows a command when it holds a
// runnable file of that name.
func TestFindShadowsIgnoresWhatCannotShadow(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses POSIX permissions")
	}
	root := t.TempDir()
	sdkBin := filepath.Join(root, "sdk", "bin")
	writeCommand(t, sdkBin, "node")
	provided := ListExecutables([]string{sdkBin})

	missing := filepath.Join(root, "missing", "bin")
	directory := filepath.Join(root, "directory", "bin")
	if err := os.MkdirAll(filepath.Join(directory, "node"), 0755); err != nil { // a directory called node
		t.Fatal(err)
	}
	plain := filepath.Join(root, "plain", "bin")
	if err := os.MkdirAll(plain, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(plain, "node"), []byte("not a program"), 0644); err != nil { // no executable bit
		t.Fatal(err)
	}

	if got := findShadows(provided, []string{missing, directory, plain}, pathPolicy{}); got != nil {
		t.Errorf("shadows = %+v, want none", got)
	}
}
