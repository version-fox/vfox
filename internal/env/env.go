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

package env

// Vars is a map of environment variables
type Vars map[string]*string

func (v Vars) Merge(vars Vars) {
	if vars == nil {
		return
	}
	for key, value := range vars {
		v[key] = value
	}
}

// Envs is a struct that contains environment variables and PATH.
type Envs struct {
	Variables Vars
	Paths     *Paths
}

func (e *Envs) Merge(envs *Envs) {
	if envs == nil {
		return
	}
	e.Paths.Merge(envs.Paths)
	e.Variables.Merge(envs.Variables)
}
