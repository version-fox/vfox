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

func NewEnvs() *Envs {
	return &Envs{
		Variables: make(Vars),
		Paths:     NewPaths(EmptyPaths),
	}
}

func (e *Envs) Merge(envs *Envs) {
	if envs == nil {
		return
	}
	e.Paths.Merge(envs.Paths)
	e.Variables.Merge(envs.Variables)
}

// MergeByScopePriority merges envs from different scopes with proper priority
// scopePriority: list of scopes in order from HIGHEST to LOWEST priority
// For example: [Project, Session, Global] means Project has highest priority
//
// NOTE: PATH and Vars are merged differently due to their different semantics:
// - Paths.Merge appends (first added has higher priority)
// - Vars.Merge overwrites (last added wins)
//
// So this method handles them separately:
// - Paths: merged in given order (HIGHEST priority first)
// - Vars: merged in reverse order (LOWEST priority first, HIGHEST overrides)
func (e *Envs) MergeByScopePriority(envsByScope map[UseScope]*Envs, scopePriority []UseScope) {
	if len(envsByScope) == 0 {
		return
	}

	// Merge Paths in given order (HIGHEST to LOWEST priority)
	// HIGHEST priority Paths will come FIRST in PATH
	for _, scope := range scopePriority {
		if envs := envsByScope[scope]; envs != nil {
			e.Paths.Merge(envs.Paths)
		}
	}

	// Merge Vars in REVERSE order (LOWEST to HIGHEST priority)
	// HIGHEST priority Vars will override LOWER priority ones
	for i := len(scopePriority) - 1; i >= 0; i-- {
		scope := scopePriority[i]
		if envs := envsByScope[scope]; envs != nil {
			e.Variables.Merge(envs.Variables)
		}
	}
}
