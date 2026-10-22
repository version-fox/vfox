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

	"github.com/version-fox/vfox/internal/config"
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

	orig := []string{system}
	policy := policyFor(config.PathModeLegacy, orig, []string{venv}, nil)

	got := findShadows(provided, []string{venv, pnpm, system}, policy)
	want := []Shadow{
		{Name: "node", Winner: pnpmNode, Dir: pnpm, Source: ShadowSourceAdded},
		{Name: "npm", Winner: filepath.Join(system, "npm"), Dir: system, Source: ShadowSourceInherited},
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

	// Without a recorded baseline the source of a plain directory is unknown.
	unknown := findShadows(provided, []string{pnpm}, policyFor(config.PathModeLegacy, nil, nil, nil))
	if len(unknown) != 1 || unknown[0].Source != ShadowSourceUnknown {
		t.Errorf("unknown = %+v", unknown)
	}
	// A directory listed in path.preserve is on purpose.
	preserved := findShadows(provided, []string{pnpm}, policyFor(config.PathModeLegacy, nil, nil, []string{pnpm}))
	if len(preserved) != 1 || preserved[0].Source != ShadowSourcePreserved {
		t.Errorf("preserved = %+v", preserved)
	}
	// Nothing to shadow, or nothing in front.
	if findShadows(nil, []string{pnpm}, policy) != nil || findShadows(provided, nil, policy) != nil {
		t.Error("no commands or no directories in front mean no shadows")
	}
}

// FindShadows looks at the PATH `vfox env` is going to export: what stays in
// front of vfox's directories.
func TestFindShadowsFollowsTheExportedOrder(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses POSIX paths")
	}
	root := t.TempDir()
	sdkBin := filepath.Join(root, ".vfox", "sdks", "node", "bin")
	system := filepath.Join(root, "system", "bin")
	pnpm := filepath.Join(root, "pnpm", "bin")
	writeCommand(t, sdkBin, "node")
	writeCommand(t, system, "node")
	writeCommand(t, pnpm, "node")
	provided := ListExecutables([]string{sdkBin})

	t.Setenv("PATH", joinPath(pnpm, system, sdkBin))
	t.Setenv(OrigPathFlag, system)
	t.Setenv("VIRTUAL_ENV", "")
	t.Setenv("CONDA_PREFIX", "")

	legacy := pathContext(config.PathModeLegacy).FindShadows(provided)
	if len(legacy) != 1 || legacy[0].Dir != pnpm || legacy[0].Source != ShadowSourceAdded {
		t.Errorf("legacy: %+v, want node shadowed by pnpm's directory", legacy)
	}

	// In baseline mode the inherited directory no longer sits in front, and the
	// one pnpm added still does; either way the winner is the first of them.
	baseline := pathContext(config.PathModeBaseline).FindShadows(provided)
	if len(baseline) != 1 || baseline[0].Dir != pnpm {
		t.Errorf("baseline: %+v", baseline)
	}

	// Once the only thing in front is inherited, baseline mode has nothing left to report.
	t.Setenv("PATH", joinPath(system, sdkBin))
	if got := pathContext(config.PathModeBaseline).FindShadows(provided); got != nil {
		t.Errorf("baseline with only inherited directories in front: %+v", got)
	}
	if got := pathContext(config.PathModeLegacy).FindShadows(provided); len(got) != 1 || got[0].Source != ShadowSourceInherited {
		t.Errorf("legacy with only inherited directories in front: %+v", got)
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
