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
	"testing"
	"time"
)

type kv struct {
	key   string
	value Value
}

type test1 struct {
	V1 string
	V2 int
	V3 bool
}

func newKv(t *testing.T, v any) Value {
	value, err := NewValue(v)
	if err != nil {
		t.Fatalf("Failed to create value: %v", err)
	}
	return value
}

func TestFileCache(t *testing.T) {
	cache, err := NewFileCache("testfile.cache")
	if err != nil {
		t.Errorf("Failed to create cache: %v", err)
	}
	testdata := []kv{
		{"test1", Value("test1")},
		{"test2", newKv(t, true)},
		{"test3", newKv(t, 123)},
		{"test4", newKv(t, 'c')},
		{"test5", nil},
	}
	t.Run("TestSimpleType", func(t *testing.T) {

		for _, d := range testdata {
			cache.Set(d.key, d.value, NeverExpired)
		}

		for _, d := range testdata {
			v, ok := cache.Get(d.key)
			if !ok {
				t.Errorf("Failed to get key %s", d.key)
			}
			if string(v) != string(d.value) {
				t.Errorf("Expected %s, got %s", string(d.value), string(v))
			}
		}
	})

	t.Run("TestStructType", func(t *testing.T) {
		td1 := test1{
			V1: "test",
			V2: 123,
			V3: true,
		}
		cache.Set("test5", newKv(t, td1), NeverExpired)

		v, ok := cache.Get("test5")
		if !ok {
			t.Errorf("Failed to get key test5")
		}
		result := test1{}
		if err = v.Unmarshal(&result); err != nil {
			t.Errorf("Failed to unmarshal: %v", err)
		}
		if result.V1 != td1.V1 || result.V2 != td1.V2 || result.V3 != td1.V3 {
			t.Errorf("Expected %v, got %v", td1, result)
		}
	})

	t.Run("TestExpire", func(t *testing.T) {
		el := len(cache.items)
		cache.Set("test6", newKv(t, "123"), ExpireTime(time.Second))
		time.Sleep(time.Second * 2)

		_, ok := cache.Get("test6")
		if ok {
			t.Errorf("Expected key test6 to be expired")
		}
		if len(cache.items) != el {
			t.Errorf("Expected %d items, got %d", el, len(cache.items))
		}
	})

	t.Run("TestRemove", func(t *testing.T) {
		cache.Set("test7", newKv(t, "123"), NeverExpired)
		cache.Remove("test7")
		_, ok := cache.Get("test7")
		if ok {
			t.Errorf("Expected key test7 to be removed")
		}
	})

	t.Run("TestToFile", func(t *testing.T) {
		for _, d := range testdata {
			cache.Set(d.key, d.value, NeverExpired)
		}

		if err := cache.Close(); err != nil {
			t.Errorf("Failed to close cache: %v", err)
		}

		newCache, err := NewFileCache("testfile.cache")
		if err != nil {
			t.Errorf("Failed to create cache: %v", err)
		}
		for _, d := range testdata {
			v, ok := newCache.Get(d.key)
			if !ok {
				t.Errorf("Failed to get key %s", d.key)
			}
			if string(v) != string(d.value) {
				t.Errorf("Expected %s, got %s", string(d.value), string(v))
			}
		}
	})

	defer os.Remove("testfile.cache")
}
