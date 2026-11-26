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

// Package cache exposes a small persistent key/value cache to plugins as `vfox.cache`.
package cache

import (
	"encoding/gob"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sync"
	"time"

	lua "github.com/yuin/gopher-lua"

	"github.com/version-fox/vfox/internal/plugin/luai/module/json"
	"github.com/version-fox/vfox/internal/shared/cache"
)

// maxSeconds is the largest expire time that still fits in a time.Duration.
const maxSeconds = int64(math.MaxInt64 / int64(time.Second))

// writeMu serializes read-modify-write cycles inside this process. Other processes
// are not locked out: the last writer of two overlapping writes wins.
var writeMu sync.Mutex

// Preload adds the module to the given Lua state's package.preload table. After it
// has been preloaded, it can be loaded using require:
//
//	local cache = require("vfox.cache")
//	local c = cache.new("/path/to/cache-file")
//	c.set("key", { 1, 2, 3 }, 3600) -- expires after 3600 seconds, -1 never expires
//	local value = c.get("key")      -- nil when missing or expired
//	c.remove("key")
func Preload(L *lua.LState) {
	L.PreloadModule("vfox.cache", Loader)
}

// Loader is the module loader function.
func Loader(L *lua.LState) int {
	t := L.NewTable()
	L.SetFuncs(t, map[string]lua.LGFunction{"new": apiNew})
	L.Push(t)
	return 1
}

// load reads the items stored in the file. A missing file is an empty cache.
func load(path string) (map[string]cache.Item, error) {
	items := make(map[string]cache.Item)
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return items, nil
		}
		return nil, err
	}
	defer file.Close()
	if err := gob.NewDecoder(file).Decode(&items); err != nil {
		return nil, err
	}
	return items, nil
}

// save writes the items to a temporary file next to path and renames it over path,
// so readers never see a half written file. The temporary file is synced first; the
// directory is not, so the latest change can still be lost on power failure. The
// file is private to the user (0600) on Unix; Windows ignores the mode bits.
func save(path string, items map[string]cache.Item) error {
	dir := filepath.Dir(path)
	// The mode only applies to directories MkdirAll creates; existing ones keep theirs.
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if err := gob.NewEncoder(tmp).Encode(items); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

func expired(item cache.Item, now int64) bool {
	return item.Expire != int64(cache.NeverExpired) && now > item.Expire
}

// update loads the file again, applies fn and saves the result, dropping expired
// items on the way. Reloading keeps changes that other cache objects, or other
// processes, made after this one was created.
func update(path string, fn func(items map[string]cache.Item)) error {
	writeMu.Lock()
	defer writeMu.Unlock()
	items, err := load(path)
	if err != nil {
		return err
	}
	now := time.Now().UnixNano()
	for k, item := range items {
		if expired(item, now) {
			delete(items, k)
		}
	}
	fn(items)
	return save(path, items)
}

// apiNew lua cache.new(path): opens (or creates) the cache stored in the file at path.
// It returns the cache table, or nil and an error message. A file that cannot be
// decoded is reported and left alone, never overwritten.
func apiNew(L *lua.LState) int {
	path := L.CheckString(1)
	if _, err := load(path); err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(fmt.Sprintf("open cache %q: %s", path, err)))
		return 2
	}

	t := L.NewTable()
	t.RawSetString("set", L.NewFunction(func(L *lua.LState) int {
		key := L.CheckString(1)
		value := L.CheckAny(2)
		seconds := L.OptNumber(3, -1)
		if seconds != -1 && (math.IsNaN(float64(seconds)) || seconds < 0 || seconds > lua.LNumber(maxSeconds)) {
			L.ArgError(3, "expire time must be -1 (never expires) or a number of seconds >= 0")
		}
		data, err := json.Encode(value)
		if err != nil {
			L.Push(lua.LNil)
			L.Push(lua.LString(err.Error()))
			return 2
		}
		expire := int64(cache.NeverExpired)
		if seconds != -1 {
			expire = time.Now().Add(time.Duration(float64(seconds) * float64(time.Second))).UnixNano()
		}
		if err := update(path, func(items map[string]cache.Item) {
			items[key] = cache.Item{Val: data, Expire: expire}
		}); err != nil {
			L.Push(lua.LNil)
			L.Push(lua.LString(err.Error()))
			return 2
		}
		L.Push(lua.LTrue)
		return 1
	}))
	t.RawSetString("get", L.NewFunction(func(L *lua.LState) int {
		key := L.CheckString(1)
		items, err := load(path)
		if err != nil {
			L.Push(lua.LNil)
			L.Push(lua.LString(err.Error()))
			return 2
		}
		item, ok := items[key]
		if !ok || expired(item, time.Now().UnixNano()) {
			L.Push(lua.LNil)
			return 1
		}
		decoded, err := json.Decode(L, item.Val)
		if err != nil {
			L.Push(lua.LNil)
			L.Push(lua.LString(err.Error()))
			return 2
		}
		L.Push(decoded)
		return 1
	}))
	t.RawSetString("remove", L.NewFunction(func(L *lua.LState) int {
		key := L.CheckString(1)
		if err := update(path, func(items map[string]cache.Item) { delete(items, key) }); err != nil {
			L.Push(lua.LFalse)
			L.Push(lua.LString(err.Error()))
			return 2
		}
		L.Push(lua.LTrue)
		return 1
	}))
	L.Push(t)
	return 1
}
