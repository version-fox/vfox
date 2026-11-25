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
	"testing"
)

// The extensions a bare command name may stand for follow PATHEXT, in its
// order, with a dot added where PATHEXT leaves it out; a PowerShell script is
// not a process, so .PS1 is left out. The list is built from the given value,
// so this holds on every runner, not only on Windows.
func TestWindowsPathExtsFollowPathextAndSkipPowerShell(t *testing.T) {
	for _, tc := range []struct {
		pathExt string
		want    []string
	}{
		{"", []string{".COM", ".EXE", ".BAT", ".CMD"}},
		{".CMD;.EXE", []string{".CMD", ".EXE"}},
		{".COM;.EXE;.BAT;.CMD;.PS1", []string{".COM", ".EXE", ".BAT", ".CMD"}},
		{"EXE;;.cmd", []string{".EXE", ".cmd"}},
		{".ps1", []string{".COM", ".EXE", ".BAT", ".CMD"}},
	} {
		if got := windowsPathExts(tc.pathExt); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("windowsPathExts(%q) = %q, want %q", tc.pathExt, got, tc.want)
		}
	}
}

// A bare name takes the first extension in PATHEXT order that exists in the
// directory, a typed extension is taken as it is, and a directory that carries
// the name is not an executable. Nothing here runs a file, so the rules are
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
	must(t, os.MkdirAll(filepath.Join(bin, "betatool"), 0755))
	must(t, os.MkdirAll(filepath.Join(bin, "gammatool.exe"), 0755))
	exts := []string{".CMD", ".EXE"}

	if got, ok := findWindowsExecutable(bin, "alphatool", exts); !ok || got != filepath.Join(bin, "alphatool.CMD") {
		t.Errorf("bare name = (%q, %v), want %q in PATHEXT order", got, ok, cmdFile)
	}
	if got, ok := findWindowsExecutable(bin, "alphatool", []string{".EXE", ".CMD"}); !ok || got != filepath.Join(bin, "alphatool.EXE") {
		t.Errorf("bare name with .EXE first = (%q, %v), want %q", got, ok, exeFile)
	}
	if got, ok := findWindowsExecutable(bin, "alphatool.exe", exts); !ok || got != exeFile {
		t.Errorf("typed extension = (%q, %v), want %q as typed", got, ok, exeFile)
	}
	if got, ok := findWindowsExecutable(bin, "alphatool.ps1", exts); ok {
		t.Errorf("a typed .ps1 that does not exist = (%q, %v), want not found", got, ok)
	}
	if got, ok := findWindowsExecutable(bin, "betatool", exts); ok {
		t.Errorf("a directory named like the command = (%q, %v), want not found", got, ok)
	}
	if got, ok := findWindowsExecutable(bin, "gammatool", exts); ok {
		t.Errorf("a directory named like the command with an extension = (%q, %v), want not found", got, ok)
	}
}
