/*
 *    Copyright 2024 [lihan aooohan@gmail.com]
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

package env

import (
	"io"
)

type Manager interface {
	Flush() error
	Load(key, value string)
	Get(key string) (string, bool)
	Remove(key string) error
	Paths(paths []string) string
	io.Closer
}

type Envs map[string]*string

type KV struct {
	Key   string
	Value string
}

type Store struct {
	envMap        map[string]string
	deletedEnvMap map[string]struct{}
	// $PATH
	pathMap        map[string]struct{}
	deletedPathMap map[string]struct{}
}

func (s *Store) Add(kv *KV) {
	if kv.Key == "PATH" {
		s.pathMap[kv.Value] = struct{}{}
	} else {
		s.envMap[kv.Key] = kv.Value
	}
}

func (s *Store) Remove(key string) {
	if _, ok := s.pathMap[key]; ok {
		delete(s.pathMap, key)
		s.deletedPathMap[key] = struct{}{}
	} else {
		delete(s.envMap, key)
		s.deletedEnvMap[key] = struct{}{}
	}
}

func NewStore() *Store {
	return &Store{
		envMap:         make(map[string]string),
		pathMap:        make(map[string]struct{}),
		deletedPathMap: make(map[string]struct{}),
		deletedEnvMap:  make(map[string]struct{}),
	}
}
