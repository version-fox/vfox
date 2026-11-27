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
	"math"
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
	`, file)
	time.Sleep(200 * time.Millisecond)
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

func TestTTLBeyondTheStorableRangeIsRejected(t *testing.T) {
	s := lua.NewState()
	defer s.Close()
	Preload(s)
	s.SetGlobal("arg1", lua.LString(filepath.Join(t.TempDir(), "cache.bin")))
	err := s.DoString(`require("vfox.cache").new(arg1).set("far", "v", 8000000000)`)
	if err == nil || !strings.Contains(err.Error(), "expire time") {
		t.Fatalf("expected an expire time error, got %v", err)
	}
	// a long ttl that fits stays readable
	eval(t, `
	local c = require("vfox.cache").new(arg1)
	assert(c.set("far", "v", 3000000000) == true)
	assert(c.get("far") == "v")
	`, filepath.Join(t.TempDir(), "cache.bin"))
}

func TestSharedSubtableIsStoredAndCycleIsRejected(t *testing.T) {
	eval(t, `
	local c = require("vfox.cache").new(arg1)
	local shared = { 1, 2 }
	assert(c.set("k", { a = shared, b = shared }, -1) == true)
	local v = c.get("k")
	assert(v.a[2] == 2 and v.b[2] == 2)
	local loop = {}
	loop.self = loop
	local ok, err = c.set("loop", loop)
	assert(ok == nil and type(err) == "string")
	`, filepath.Join(t.TempDir(), "cache.bin"))
}

func TestNonUTF8RoundTrip(t *testing.T) {
	file := filepath.Join(t.TempDir(), "cache.bin")
	eval(t, `
	local c = require("vfox.cache").new(arg1)
	assert(c.set("k", "\255\254binary", -1) == nil)
	assert(c.get("k") == nil)
	local ok, err = c.set("\255", "v", -1)
	assert(ok == nil and err:find("UTF%-8"))
	ok, err = c.set("k", { nested = { "\255" } }, -1)
	assert(ok == nil and err:find("UTF%-8"))
	ok, err = c.set("k", { ["\255"] = 1 }, -1)
	assert(ok == nil and err:find("UTF%-8"))
	assert(c.set("k", "plain \226\130\172", -1) == true)
	assert(c.get("k") == "plain \226\130\172")
	`, file)
}

func TestExpireAtUsesOneClockReading(t *testing.T) {
	// Leave 500ns of slack below the bound, like the case a reviewer reported.
	const ttl = 7428000000
	base := int64(math.MaxInt64) - ttl*int64(time.Second) - 500
	now := time.Unix(0, base)
	for _, seconds := range []float64{0, 1, ttl - 2, ttl} {
		expire, ok := expireAt(now, seconds)
		if ok && expire < now.UnixNano() {
			t.Fatalf("ttl %v wrapped to %d", seconds, expire)
		}
	}
	if _, ok := expireAt(now, ttl); ok {
		t.Fatal("a ttl at the bound must keep a margin and be rejected")
	}
	if _, ok := expireAt(now, ttl-2); !ok {
		t.Fatal("a ttl two seconds inside the bound must be accepted")
	}
	for _, bad := range []float64{-1, math.NaN(), math.Inf(1), 1e19} {
		if _, ok := expireAt(time.Now(), bad); ok {
			t.Fatalf("ttl %v must be rejected", bad)
		}
	}
}

func TestCyclicFanOutIsRejected(t *testing.T) {
	file := filepath.Join(t.TempDir(), "cache.bin")
	eval(t, `
	local c = require("vfox.cache").new(arg1)
	local t = {}
	t.a = t
	t.b = t
	local ok, err = c.set("k", t, -1)
	assert(ok == nil and type(err) == "string")
	`, file)
}

func TestDeepInvalidStringIsRejected(t *testing.T) {
	file := filepath.Join(t.TempDir(), "cache.bin")
	eval(t, `
	local c = require("vfox.cache").new(arg1)
	local t = {"\255"}
	for i = 1, 250 do t = { t } end
	local ok, err = c.set("k", t, -1)
	assert(ok == nil and type(err) == "string")
	`, file)
}

func TestSharedSubtableFanOutIsCapped(t *testing.T) {
	file := filepath.Join(t.TempDir(), "cache.bin")
	start := time.Now()
	eval(t, `
	local c = require("vfox.cache").new(arg1)
	local t = {1}
	for i = 1, 40 do t = { a = t, b = t } end
	local ok, err = c.set("k", t, -1)
	assert(ok == nil and type(err) == "string")
	`, file)
	if time.Since(start) > 10*time.Second {
		t.Fatal("encoding a shared subtable chain took too long")
	}
}

// Go writes <, > and & as six-byte escapes, so the size limit has to count the
// escaped length. Two 10 MiB strings of "<" would otherwise write a 120 MB file.
func TestEscapedOutputCountsAgainstSizeLimit(t *testing.T) {
	file := filepath.Join(t.TempDir(), "cache.bin")
	eval(t, `
	local c = require("vfox.cache").new(arg1)
	local s = string.rep("<", 10 * 1024 * 1024)
	local ok, err = c.set("k", { s, s }, -1)
	assert(ok == nil and string.find(err, "64 MiB"), tostring(err))
	assert(c.get("k") == nil)
	-- the same data without escapes (20 MiB written) is accepted
	local plain = string.rep("a", 10 * 1024 * 1024)
	assert(c.set("k", { plain, plain }, -1) == true)
	-- keys are charged by their escaped size too
	local key = string.rep("&", 11 * 1024 * 1024)
	local ok2, err2 = c.set("k2", { [key] = 1 }, -1)
	assert(ok2 == nil and string.find(err2, "64 MiB"), tostring(err2))
	`, file)
	if info, err := os.Stat(file); err == nil && info.Size() > 64<<20 {
		t.Fatalf("cache file is %d bytes", info.Size())
	}
}

// TestDocumentedRefusals pins every refusal the cache docs list for set, with the
// message a plugin sees. The messages carry no "json: error calling MarshalJSON" prefix.
func TestDocumentedRefusals(t *testing.T) {
	file := filepath.Join(t.TempDir(), "cache.bin")
	eval(t, `
	local c = require("vfox.cache").new(arg1)
	local function refused(v, want)
		local ok, err = c.set("k", v, -1)
		assert(ok == nil, "stored")
		assert(err:find(want, 1, true), err)
		assert(not err:find("MarshalJSON", 1, true), err)
	end
	refused(0/0, "not finite")
	refused(1/0, "not finite")
	refused(-1/0, "not finite")
	refused({ 1/0 }, "not finite")
	refused({ x = 0/0 }, "not finite")
	refused(print, "cannot encode function")
	refused(coroutine.create(function() end), "cannot encode thread")
	refused(io.stdout, "cannot encode userdata")
	refused({ 1, nil, 3 }, "sparse")
	refused({ 1, a = 2 }, "mixed or invalid key")
	local self = {}
	self[1] = self
	refused(self, "recursively nested")
	local deep = {}
	for i = 1, 257 do deep = { deep } end
	refused(deep, "256 levels")
	refused("\255", "UTF-8")
	local ok, err = c.set("\255", 1, -1)
	assert(ok == nil and err:find("UTF-8", 1, true), err)
	-- the repeated-reference cap applies to set as well: 1,048,576 values
	local shared = { 1, 2 }
	local list = {}
	for i = 1, 524290 do list[i] = shared end
	refused(list, "referenced too many times")
	list[524290] = nil
	assert(c.set("k", list, -1) == true)
	assert(c.get("k")[524289][2] == 2)
	`, file)
}

func TestTTLErrorNamesBothBounds(t *testing.T) {
	file := filepath.Join(t.TempDir(), "cache.bin")
	limit := maxTTL(time.Now())
	if limit < 7_000_000_000 || limit > 7_500_000_000 {
		t.Fatalf("unexpected ttl limit %d", limit)
	}
	eval(t, `
	local c = require("vfox.cache").new(arg1)
	assert(c.set("a", 1, 7.4e9))
	for _, bad in ipairs({ 7.5e9, 1/0, 0/0, -0.5, -2 }) do
		local ok, err = pcall(c.set, "b", 1, bad)
		assert(not ok)
		assert(err:find("-1 (never expires)", 1, true) and err:find("from 0 up to 7", 1, true), err)
	end
	`, file)
}

func TestReadErrorsAreDescribed(t *testing.T) {
	dir := t.TempDir()
	corrupt := filepath.Join(dir, "corrupt.bin")
	empty := filepath.Join(dir, "empty.bin")
	if err := os.WriteFile(corrupt, []byte("garbage"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(empty, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	eval(t, `
	local c, err = require("vfox.cache").new(arg2)
	assert(c == nil and err:find("read cache", 1, true) and err:find("EOF", 1, true), err)
	local c, err = require("vfox.cache").new(arg1)
	assert(c == nil and err:find("read cache", 1, true), err)
	`, corrupt, empty)
	// a file that goes bad after new: get, set and remove report it and leave it alone
	good := filepath.Join(dir, "good.bin")
	s := lua.NewState()
	defer s.Close()
	Preload(s)
	s.SetGlobal("arg1", lua.LString(good))
	if err := s.DoString(`
	c = require("vfox.cache").new(arg1)
	assert(c.set("x", 1))
	`); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(good, []byte("garbage"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := s.DoString(`
	local v, err = c.get("x")
	assert(v == nil and err:find("read cache", 1, true) and err:find("EOF", 1, true), err)
	local ok, err = c.set("x", 2)
	assert(ok == nil and err:find("read cache", 1, true), err)
	local ok, err = c.remove("x")
	assert(ok == false and err:find("read cache", 1, true), err)
	`); err != nil {
		t.Fatal(err)
	}
	if data, _ := os.ReadFile(good); string(data) != "garbage" {
		t.Fatalf("file was overwritten: %q", data)
	}
}
