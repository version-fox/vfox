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

package cache

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	lua "github.com/yuin/gopher-lua"
)

func eval(t *testing.T, script string, args ...string) {
	t.Helper()
	s := lua.NewState()
	defer s.Close()
	Preload(s)
	for i, a := range args {
		s.SetGlobal("arg"+string(rune('1'+i)), lua.LString(a))
	}
	if err := s.DoString(script); err != nil {
		t.Fatal(err)
	}
}

func TestSetGetRemove(t *testing.T) {
	file := filepath.Join(t.TempDir(), "nested", "cache.bin")
	eval(t, `
	local c = require("vfox.cache").new(arg1)
	assert(c.get("missing") == nil)
	assert(c.set("s", "text") == true)
	assert(c.set("n", 42, -1))
	assert(c.set("t", { a = 1, list = { "x", "y" } }, 3600))
	assert(c.get("s") == "text")
	assert(c.get("n") == 42)
	local t = c.get("t")
	assert(t.a == 1 and t.list[2] == "y")
	assert(c.remove("s") == true)
	assert(c.get("s") == nil)
	`, file)
}

func TestPersistsAcrossStates(t *testing.T) {
	file := filepath.Join(t.TempDir(), "cache.bin")
	eval(t, `
	local c = require("vfox.cache").new(arg1)
	c.set("keep", "me", -1)
	c.set("gone", "x", -1)
	c.remove("gone")
	`, file)
	eval(t, `
	local c = require("vfox.cache").new(arg1)
	assert(c.get("keep") == "me")
	assert(c.get("gone") == nil)
	`, file)
}

func TestExpiredEntryIsNil(t *testing.T) {
	file := filepath.Join(t.TempDir(), "cache.bin")
	eval(t, `
	local c = require("vfox.cache").new(arg1)
	c.set("k", "v", 0)
	assert(c.get("k") == nil)
	`, file)
}

func TestNegativeExpireOtherThanNeverIsRejected(t *testing.T) {
	s := lua.NewState()
	defer s.Close()
	Preload(s)
	s.SetGlobal("arg1", lua.LString(filepath.Join(t.TempDir(), "cache.bin")))
	err := s.DoString(`require("vfox.cache").new(arg1).set("k", "v", -5)`)
	if err == nil || !strings.Contains(err.Error(), "expire time") {
		t.Fatalf("expected an expire time error, got %v", err)
	}
}

func TestUnserializableValueReturnsError(t *testing.T) {
	file := filepath.Join(t.TempDir(), "cache.bin")
	eval(t, `
	local c = require("vfox.cache").new(arg1)
	local ok, err = c.set("f", function() end)
	assert(ok == nil and type(err) == "string")
	assert(c.get("f") == nil)
	`, file)
}

func TestNewReportsCorruptFile(t *testing.T) {
	dir := t.TempDir()
	eval(t, `
	local c, err = require("vfox.cache").new(arg1)
	assert(c == nil and type(err) == "string")
	`, dir) // a directory cannot be read as a cache file
}

func TestTwoCacheObjectsKeepEachOthersChanges(t *testing.T) {
	file := filepath.Join(t.TempDir(), "cache.bin")
	eval(t, `
	local mod = require("vfox.cache")
	local a, b = mod.new(arg1), mod.new(arg1)
	a.set("a", 1)
	b.set("b", 2)
	assert(a.get("b") == 2 and b.get("a") == 1)
	`, file)
}

func TestConcurrentWritersDoNotLoseUpdates(t *testing.T) {
	file := filepath.Join(t.TempDir(), "cache.bin")
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			eval(t, `require("vfox.cache").new(arg1).set("k"..arg2, true)`, file, string(rune('a'+i)))
		}(i)
	}
	wg.Wait()
	eval(t, `
	local c = require("vfox.cache").new(arg1)
	for _, k in ipairs({"ka","kb","kc","kd","ke","kf","kg","kh"}) do assert(c.get(k) == true, k) end
	`, file)
}

func TestInvalidExpireValues(t *testing.T) {
	for _, v := range []string{"0/0", "1e30", "math.huge", `"soon"`, "{}"} {
		s := lua.NewState()
		Preload(s)
		s.SetGlobal("arg1", lua.LString(filepath.Join(t.TempDir(), "cache.bin")))
		if err := s.DoString(`require("vfox.cache").new(arg1).set("k", "v", ` + v + `)`); err == nil {
			t.Errorf("expire %s was accepted", v)
		}
		s.Close()
	}
}

func TestCorruptFileIsReportedAndNotOverwritten(t *testing.T) {
	file := filepath.Join(t.TempDir(), "cache.bin")
	if err := os.WriteFile(file, []byte("not a gob"), 0o600); err != nil {
		t.Fatal(err)
	}
	eval(t, `
	local c, err = require("vfox.cache").new(arg1)
	assert(c == nil and type(err) == "string")
	`, file)
	if data, _ := os.ReadFile(file); string(data) != "not a gob" {
		t.Fatalf("corrupt file was modified: %q", data)
	}
}

func TestExpiryIsPersistedAndNoTempFilesRemain(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "cache.bin")
	eval(t, `
	local c = require("vfox.cache").new(arg1)
	c.set("short", "v", 0.05)
	c.set("long", "v", 3600)
	assert(c.get("short") == "v")
	`, file)
	time.Sleep(100 * time.Millisecond)
	eval(t, `
	local c = require("vfox.cache").new(arg1)
	assert(c.get("short") == nil and c.get("long") == "v")
	`, file)
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Fatalf("expected only the cache file, got %v", entries)
	}
	if runtime.GOOS != "windows" {
		if info, _ := os.Stat(file); info.Mode().Perm() != 0o600 {
			t.Fatalf("mode %v, want 0600", info.Mode().Perm())
		}
	}
}

func TestLuaValueRoundTrips(t *testing.T) {
	file := filepath.Join(t.TempDir(), "cache.bin")
	eval(t, `
	local c = require("vfox.cache").new(arg1)
	assert(c.set("b", false) == true)
	assert(c.get("b") == false)
	assert(c.set("t", true) == true)
	assert(c.get("t") == true)
	assert(c.set("z", 0) == true)
	assert(c.get("z") == 0)
	assert(c.set("e", "") == true)
	assert(c.get("e") == "")
	assert(c.set("arr", {}) == true)
	local a = c.get("arr")
	assert(type(a) == "table" and next(a) == nil)
	assert(c.set("nested", { list = { 1, 2 }, obj = { k = "v" } }) == true)
	local n = c.get("nested")
	assert(n.list[2] == 2 and n.obj.k == "v")
	assert(c.set("nilval", nil) == true)
	assert(c.get("nilval") == nil) -- stored as JSON null, same as a missing key
	`, file)
}
