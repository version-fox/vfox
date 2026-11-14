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

package util

import (
	"os"
	"testing"

	"golang.org/x/sys/windows"
)

// createNamedPipe returns the server end of a named pipe called name. The name
// is what tells a pty apart from an ordinary pipe.
func createNamedPipe(t *testing.T, name string) *os.File {
	t.Helper()
	path, err := windows.UTF16PtrFromString(`\\.\pipe\` + name)
	if err != nil {
		t.Fatal(err)
	}
	handle, err := windows.CreateNamedPipe(path, windows.PIPE_ACCESS_DUPLEX, windows.PIPE_TYPE_BYTE, 1, 4096, 4096, 0, nil)
	if err != nil {
		t.Fatalf("CreateNamedPipe(%s): %v", name, err)
	}
	file := os.NewFile(uintptr(handle), name)
	t.Cleanup(func() { file.Close() })
	return file
}

// replaceStdout points os.Stdout at file until the test ends.
func replaceStdout(t *testing.T, file *os.File) {
	t.Helper()
	previous := os.Stdout
	os.Stdout = file
	t.Cleanup(func() { os.Stdout = previous })
}

// Git Bash in mintty gives native programs a pipe named \msys-<id>-pty<N>-to-master
// instead of a console. The person at mintty is at a terminal, so IsTTY must say so.
func TestIsTTYRecognizesMinttyPty(t *testing.T) {
	replaceStdout(t, createNamedPipe(t, "msys-1888ae32e00d56aa-pty0-to-master"))
	if !IsTTY() {
		t.Error("IsTTY() = false for a mintty pty, want true")
	}
}

func TestIsTTYRejectsOrdinaryPipe(t *testing.T) {
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		reader.Close()
		writer.Close()
	})
	replaceStdout(t, writer)
	if IsTTY() {
		t.Error("IsTTY() = true for an ordinary pipe, want false")
	}
}

// Git Bash's stdin is a pipe too, which the key reader cannot read.
func TestCanReadKeysRejectsPipe(t *testing.T) {
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		reader.Close()
		writer.Close()
	})
	previous := os.Stdin
	os.Stdin = reader
	t.Cleanup(func() { os.Stdin = previous })
	if CanReadKeys() {
		t.Error("CanReadKeys() = true for a pipe, want false")
	}
}

func TestIsCygwinPipeName(t *testing.T) {
	cases := map[string]bool{
		`\msys-1888ae32e00d56aa-pty0-to-master`:                  true,
		`\msys-1888ae32e00d56aa-pty1-from-master`:                true,
		`\cygwin-ab12cd34-pty2-to-master`:                        true,
		`\Device\NamedPipe\msys-1888ae32e00d56aa-pty0-to-master`: true,
		`\msys-1888ae32e00d56aa-pty0-to-slave`:                   false,
		`\msys--pty0-to-master`:                                  false,
		`\msys-1888ae32e00d56aa-tty0-to-master`:                  false,
		`\other-1888ae32e00d56aa-pty0-to-master`:                 false,
		`\Win32Pipes.1234.5678`:                                  false,
		``:                                                       false,
	}
	for name, want := range cases {
		if got := isCygwinPipeName(name); got != want {
			t.Errorf("isCygwinPipeName(%q) = %v, want %v", name, got, want)
		}
	}
}
