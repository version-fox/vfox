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
	"runtime"
	"strings"
	"testing"

	lua "github.com/yuin/gopher-lua"
)

// TestPopenHandleLinesIteratesLines checks that the Windows io.popen handle
// exposes :lines(), the iterator plugin code uses to enumerate command
// output. Every call must return one line without its trailing CR/LF and
// return nil at end of file so generic-for terminates.
func TestPopenHandleLinesIteratesLines(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("io.popen override under test only exists on Windows")
	}

	vm := NewLuaVM()
	if err := vm.Prepare(nil); err != nil {
		t.Fatalf("Prepare failed: %v", err)
	}
	defer vm.Close()

	script := `
local handle = io.popen("echo alpha&echo beta&echo gamma")
local lines = {}
for line in handle:lines() do
	lines[#lines + 1] = line
end
handle:close()
return table.concat(lines, "|")
`
	if err := vm.Instance.DoString(script); err != nil {
		t.Fatalf("Lua execution failed: %v", err)
	}
	got, ok := vm.Instance.Get(-1).(lua.LString)
	if !ok {
		t.Fatalf("expected string result, got %s", vm.Instance.Get(-1).Type())
	}
	if want := "alpha|beta|gamma"; string(got) != want {
		t.Errorf("lines = %q, want %q", string(got), want)
	}
}

// TestPopenHandleLinesRejectsWriteMode ensures a write-only handle reports
// the same error as read() instead of exposing a usable iterator.
func TestPopenHandleLinesRejectsWriteMode(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("io.popen override under test only exists on Windows")
	}

	vm := NewLuaVM()
	if err := vm.Prepare(nil); err != nil {
		t.Fatalf("Prepare failed: %v", err)
	}
	defer vm.Close()

	script := `
local handle = io.popen("rem", "w")
local ok, err = pcall(function() return handle:lines() end)
handle:close()
return tostring(ok) .. "|" .. tostring(err)
`
	if err := vm.Instance.DoString(script); err != nil {
		t.Fatalf("Lua execution failed: %v", err)
	}
	got := vm.Instance.Get(-1).String()
	if !strings.HasPrefix(got, "false|") {
		t.Fatalf("lines() on write handle did not raise: %q", got)
	}
	if !strings.Contains(got, "write-only") {
		t.Errorf("error %q does not mention write-only", got)
	}
}
