/*
 *    Copyright 2024 Han Li and contributors
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
	Load(envs *Envs) error
	Get(key string) (string, bool)
	Remove(envs *Envs) error
	io.Closer
}

// Vars is a map of environment variables
type Vars map[string]*string

// Envs is a struct that contains environment variables and PATH.
type Envs struct {
	Variables Vars
	BinPaths  *Paths
	Paths     *Paths
}
