/*
 *    Copyright 2026 Han Li and contributors
 *
 *    Licensed under the Apache License, Version 2.0 (the "License");
 *    you may not use this file except in compliance with the License.
 *    You may obtain a copy of the License at
 *
 *        http://www.apache.org/licenses/LICENSE-2.0
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
	"reflect"
	"runtime"
	"strings"
	"testing"
)

// The extensions a bare command name may stand for follow PATHEXT, in its
// order, with a dot added where PATHEXT leaves it out, and only the ones
// CreateProcess can start stay: .COM, .EXE, .BAT and .CMD, compared without
// regard to case. A script extension (.PS1, .VBS, .JS, ...) is dropped. The
// list is built from the given value, so this holds on every runner.
func TestWindowsPathExtsKeepOnlyProcessExtensions(t *testing.T) {
	for _, tc := range []struct {
		pathExt string
		want    []string
	}{
		{"", []string{".COM", ".EXE", ".BAT", ".CMD"}},
		{".CMD;.EXE", []string{".CMD", ".EXE"}},
		{".COM;.EXE;.BAT;.CMD;.PS1", []string{".COM", ".EXE", ".BAT", ".CMD"}},
		{"EXE;;.cmd", []string{".EXE", ".cmd"}},
		{".ps1", []string{".COM", ".EXE", ".BAT", ".CMD"}},
		{".COM;.EXE;.BAT;.CMD;.VBS;.VBE;.JS;.JSE;.WSF;.WSH;.MSC;.PS1", []string{".COM", ".EXE", ".BAT", ".CMD"}},
		{".JS;.exe;.VBS", []string{".exe"}},
	} {
		if got := windowsPathExts(tc.pathExt); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("windowsPathExts(%q) = %q, want %q", tc.pathExt, got, tc.want)
		}
	}
}

// A bare name takes the first extension in PATHEXT order that exists in the
// directory, a typed extension is taken as it is, and a directory that carries
// the name is not an executable. The extensions are spelled as the files are,
// so the case-sensitive file systems of the Linux and macOS runners agree with
// Windows. Nothing here runs a file, so the rules are
// checked on every runner.
func TestFindWindowsExecutableOrderAndTypedExtension(t *testing.T) {
	bin := t.TempDir()
	touch := func(name string) string {
		path := filepath.Join(bin, name)
		must(t, os.WriteFile(path, []byte{}, 0755))
		return path
	}
	cmdFile := touch("alphatool.cmd")
	exeFile := touch("alphatool.exe")
	must(t, os.MkdirAll(filepath.Join(bin, "betatool.cmd"), 0755))
	must(t, os.MkdirAll(filepath.Join(bin, "gammatool.exe"), 0755))
	exts := []string{".cmd", ".exe"}

	if got, ok := findWindowsExecutable(bin, "alphatool", exts); !ok || got != cmdFile {
		t.Errorf("bare name = (%q, %v), want %q in PATHEXT order", got, ok, cmdFile)
	}
	if got, ok := findWindowsExecutable(bin, "alphatool", []string{".exe", ".cmd"}); !ok || got != exeFile {
		t.Errorf("bare name with .EXE first = (%q, %v), want %q", got, ok, exeFile)
	}
	if got, ok := findWindowsExecutable(bin, "alphatool.exe", exts); !ok || got != exeFile {
		t.Errorf("typed extension = (%q, %v), want %q as typed", got, ok, exeFile)
	}
	if got, ok := findWindowsExecutable(bin, "alphatool.ps1", exts); ok {
		t.Errorf("a typed .ps1 that does not exist = (%q, %v), want not found", got, ok)
	}
	if got, ok := findWindowsExecutable(bin, "betatool", exts); ok {
		t.Errorf("a directory named like the command plus extension = (%q, %v), want not found", got, ok)
	}
	if got, ok := findWindowsExecutable(bin, "gammatool", exts); ok {
		t.Errorf("a directory named like the command with an extension = (%q, %v), want not found", got, ok)
	}
}

// A PATH walk takes the directories in order and, inside each, the usable
// PATHEXT entries in order; a script that comes first on PATH does not win
// over a .cmd in a later directory, because it is not in the list at all.
// Nothing here runs a file, so the rule is checked on every runner.
func TestFindInWindowsPathSkipsScriptsInEarlierDirectories(t *testing.T) {
	first := t.TempDir()
	second := t.TempDir()
	touch := func(dir, name string) string {
		path := filepath.Join(dir, name)
		must(t, os.WriteFile(path, []byte{}, 0755))
		return path
	}
	touch(first, "alphatool.ps1")
	touch(first, "alphatool.js")
	cmdFile := touch(second, "alphatool.cmd")
	pathEnv := first + string(os.PathListSeparator) + second
	pathExt := ".COM;.EXE;.BAT;.cmd;.VBS;.VBE;.JS;.JSE;.WSF;.WSH;.MSC;.ps1"

	if got, ok := findInWindowsPath("alphatool", pathEnv, pathExt); !ok || got != cmdFile {
		t.Errorf("findInWindowsPath = (%q, %v), want %q", got, ok, cmdFile)
	}
	if got, ok := findInWindowsPath("alphatool", first, pathExt); ok {
		t.Errorf("a directory with only scripts = (%q, %v), want not found", got, ok)
	}
}

// When the SDK PATH has no match, the fallback walks the process's own PATH
// with the same rules instead of exec.LookPath, which would return the first
// PATHEXT hit in PATH order and so pick a .ps1 ahead of a later .cmd.
func TestLookPathFallbackFollowsTheSameRulesOnTheProcessPath(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("lookPathInEnvWithExt uses PATHEXT only on Windows")
	}
	sdkBin := t.TempDir()
	first := t.TempDir()
	second := t.TempDir()
	touch := func(dir, name string) string {
		path := filepath.Join(dir, name)
		must(t, os.WriteFile(path, []byte("@echo off\r\n"), 0755))
		return path
	}
	touch(first, "alphatool.ps1")
	cmdFile := touch(second, "alphatool.cmd")
	t.Setenv("PATH", first+string(os.PathListSeparator)+second)
	t.Setenv("PATHEXT", ".COM;.EXE;.BAT;.CMD;.PS1")

	got, err := lookPathInEnvWithExt("alphatool", sdkBin, ".COM;.EXE;.BAT;.CMD;.PS1")
	if err != nil || !strings.EqualFold(got, cmdFile) {
		t.Errorf("fallback = (%q, %v), want %q", got, err, cmdFile)
	}
	if got, err := lookPathInEnvWithExt("missingtool", sdkBin, ".CMD"); err == nil {
		t.Errorf("a name found nowhere = %q, want an error", got)
	}
}
