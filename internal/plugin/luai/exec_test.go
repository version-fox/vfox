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
	"bufio"
	"os"
	"testing"

	lua "github.com/yuin/gopher-lua"
)

func newTestPopenHandle(t *testing.T, content string) *popenHandle {
	t.Helper()
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe failed: %v", err)
	}
	if _, err := writer.Write([]byte(content)); err != nil {
		t.Fatalf("write failed: %v", err)
	}
	writer.Close()
	return &popenHandle{f: reader, reader: bufio.NewReader(reader), mode: "r"}
}

func readPopenString(t *testing.T, h *popenHandle, format string) lua.LValue {
	t.Helper()
	defer h.f.Close()
	value, err := readPopen(h, format)
	if err != nil {
		t.Fatalf("readPopen(%q) failed: %v", format, err)
	}
	return value
}

func TestReadPopenAll(t *testing.T) {
	h := newTestPopenHandle(t, "hello\nworld\n")
	value := readPopenString(t, h, "*a")
	if got := string(value.(lua.LString)); got != "hello\nworld\n" {
		t.Errorf("read *a = %q, want %q", got, "hello\nworld\n")
	}
}

func TestReadPopenLineStripsNewline(t *testing.T) {
	h := newTestPopenHandle(t, "hello\nworld\n")
	value := readPopenString(t, h, "*l")
	if got := string(value.(lua.LString)); got != "hello" {
		t.Errorf("read *l = %q, want %q", got, "hello")
	}
	// Second line is buffered and readable.
	value, err := readPopen(h, "*l")
	if err != nil {
		t.Fatalf("second read failed: %v", err)
	}
	defer h.f.Close()
	if got := string(value.(lua.LString)); got != "world" {
		t.Errorf("second read *l = %q, want %q", got, "world")
	}
}

func TestReadPopenLineEOFFinalLineWithoutNewline(t *testing.T) {
	h := newTestPopenHandle(t, "noeol")
	value := readPopenString(t, h, "*l")
	if got := string(value.(lua.LString)); got != "noeol" {
		t.Errorf("read *l = %q, want %q", got, "noeol")
	}
}

func TestReadPopenLineEOFReturnsNil(t *testing.T) {
	h := newTestPopenHandle(t, "")
	value := readPopenString(t, h, "*l")
	if value != lua.LNil {
		t.Errorf("read *l at EOF = %v, want nil", value)
	}
}

func TestReadPopenBytes(t *testing.T) {
	h := newTestPopenHandle(t, "abcdef")
	value := readPopenString(t, h, "3")
	if got := string(value.(lua.LString)); got != "abc" {
		t.Errorf("read 3 = %q, want %q", got, "abc")
	}
}

func TestReadPopenBytesEOFReturnsNil(t *testing.T) {
	h := newTestPopenHandle(t, "")
	value := readPopenString(t, h, "3")
	if value != lua.LNil {
		t.Errorf("read 3 at EOF = %v, want nil", value)
	}
}
