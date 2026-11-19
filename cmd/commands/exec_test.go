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
	"runtime"
	"strings"
	"testing"
)

// A command typed by its bare name is found in the SDK's PATH even when its
// file carries an extension, as every tool on Windows does. The extension
// comes back as PATHEXT spells it, which the file system does not mind.
func TestLookPathInEnvFindsABareNameInTheGivenPath(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "bin")
	must(t, os.MkdirAll(bin, 0755))
	want := writeCommand(t, bin, "alphatool")

	got, err := lookPathInEnv("alphatool", bin)
	if err != nil || !strings.EqualFold(got, want) {
		t.Fatalf("lookPathInEnv(alphatool) = (%q, %v), want %q", got, err, want)
	}
}

// The name with its extension keeps working, and a directory that carries the
// command's name is not taken for it.
func TestLookPathInEnvTakesAnExtensionAsTypedAndSkipsDirectories(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "bin")
	must(t, os.MkdirAll(bin, 0755))
	want := writeCommand(t, bin, "alphatool")
	must(t, os.MkdirAll(filepath.Join(bin, "betatool"), 0755))

	got, err := lookPathInEnv(filepath.Base(want), bin)
	if err != nil || !strings.EqualFold(got, want) {
		t.Fatalf("lookPathInEnv(%s) = (%q, %v), want %q", filepath.Base(want), got, err, want)
	}
	if got, err := lookPathInEnv("betatool", bin); err == nil {
		t.Fatalf("lookPathInEnv(betatool) = %q, want an error for a directory", got)
	}
}

// On Windows the extensions are tried in the order of PATHEXT, as a shell does.
func TestLookPathInEnvFollowsThePathextOrder(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("PATHEXT is a Windows notion")
	}
	t.Setenv("PATHEXT", ".EXE;.CMD")
	bin := filepath.Join(t.TempDir(), "bin")
	must(t, os.MkdirAll(bin, 0755))
	must(t, os.WriteFile(filepath.Join(bin, "alphatool.cmd"), []byte("@echo off\r\n"), 0755))
	want := filepath.Join(bin, "alphatool.exe")
	must(t, os.WriteFile(want, []byte{}, 0755))

	got, err := lookPathInEnv("alphatool", bin)
	if err != nil || !strings.EqualFold(got, want) {
		t.Fatalf("lookPathInEnv(alphatool) = (%q, %v), want %q", got, err, want)
	}
}
