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
	"encoding/gob"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	NeverExpired ExpireTime = -1
)

// Duration is a duration that represents the cache duration and some special values
// -1: never expire
// 0: never cache
type Duration time.Duration

func (d Duration) MarshalYAML() (interface{}, error) {
	switch d {
	case -1:
		return -1, nil
	case 0:
		return 0, nil
	default:
		return d.String(), nil
	}
}

func (d *Duration) UnmarshalYAML(node *yaml.Node) error {
	var data any
	err := node.Decode(&data)
	if err != nil {
		return err
	}
	switch va := data.(type) {
	case int:
		*d = Duration(va)
	case string:
		pd, err := time.ParseDuration(va)
		if err != nil {
			return err
		}
		*d = Duration(pd)
	}
	return nil
}

func (d Duration) String() string {
	switch d {
	case -1:
		return "-1"
	case 0:
		return "0"
	}

	var str string
	duration := time.Duration(d)
	if h := int(duration.Hours()); h > 0 {
		str = fmt.Sprintf("%dh", h)
	}
	if m := int(duration.Minutes()) % 60; m > 0 {
		str = fmt.Sprintf("%s%dm", str, m)
	}
	if s := int(duration.Seconds()) % 60; s > 0 {
		str = fmt.Sprintf("%s%ds", str, s)
	}

	return str
}

// ExpireTime in UnixNano
type ExpireTime int64

// Value CacheValue is a byte slice that can be unmarshaled into any type
type Value []byte

// Unmarshal the CacheValue into the given value
func (c Value) Unmarshal(v any) error {
	return json.Unmarshal(c, v)
}

// NewValue marshals the given value into a CacheValue
func NewValue(v any) (Value, error) {
	return json.Marshal(v)
}

// Item is a cache item
type Item struct {
	Val    []byte
	Expire int64 // Expire time in UnixNano, -1 means never expire
}

// FileCache is a cache that saves to a file
type FileCache struct {
	mu    sync.RWMutex
	items map[string]Item
	path  string
}

// NewFileCache creates a new FileCache
func NewFileCache(path string) (*FileCache, error) {
	fc := &FileCache{
		items: make(map[string]Item),
		path:  path,
	}
	if err := fc.loadFromFile(path); err != nil {
		return nil, err
	}
	return fc, nil
}

// Set a key value pair with a duration
func (c *FileCache) Set(key string, value Value, expireTime ExpireTime) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if expireTime == NeverExpired {
		c.items[key] = Item{
			Val:    value,
			Expire: int64(NeverExpired),
		}
	} else {
		c.items[key] = Item{
			Val:    value,
			Expire: time.Now().Add(time.Duration(expireTime)).UnixNano(),
		}
	}
}

// Get a value by key
func (c *FileCache) Get(key string) (Value, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	item, exists := c.items[key]
	if !exists || time.Now().UnixNano() > item.Expire {
		if item.Expire != int64(NeverExpired) {
			// Remove expired item
			delete(c.items, key)
			return nil, false
		}
	}
	return item.Val, true
}

// Remove a key from the cache
func (c *FileCache) Remove(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.items, key)
}

// Close the cache and save to file
func (c *FileCache) Close() error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	file, err := os.Create(c.path)
	if err != nil {
		return err
	}
	defer file.Close()
	encoder := gob.NewEncoder(file)
	return encoder.Encode(c.items)
}

// loadFromFile loads the cache from a file
func (c *FileCache) loadFromFile(filename string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	file, err := os.Open(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer file.Close()
	decoder := gob.NewDecoder(file)
	return decoder.Decode(&c.items)
}
