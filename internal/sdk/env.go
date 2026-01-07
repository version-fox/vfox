/*
 *
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
 *
 */

package sdk

import "github.com/version-fox/vfox/internal/env"

type SdkEnv struct {
	Sdk *impl
	Env *env.Envs
}

type SdkEnvs []*SdkEnv

// ToVars export the env vars of SDK to shell
func (d *SdkEnvs) ToVars() env.Vars {
	vars := make(env.Vars)
	for _, sdkEnv := range *d {
		for key, value := range sdkEnv.Env.Variables {
			vars[key] = value
		}
	}
	return vars
}

func (d *SdkEnvs) ToEnvs() *env.Envs {
	envs := &env.Envs{
		Variables: make(env.Vars),
		Paths:     env.NewPaths(env.EmptyPaths),
	}
	for _, sdkEnv := range *d {
		for key, value := range sdkEnv.Env.Variables {
			envs.Variables[key] = value
		}
	}

	return envs
}

func (d *SdkEnvs) ToExportEnvs() env.Vars {
	envKeys := d.ToEnvs()

	exportEnvs := make(env.Vars)
	for k, v := range envKeys.Variables {
		exportEnvs[k] = v
	}

	osPaths := env.NewPaths(env.OsPaths)
	pathsStr := envKeys.Paths.Merge(osPaths).String()
	exportEnvs["PATH"] = &pathsStr

	return exportEnvs
}
