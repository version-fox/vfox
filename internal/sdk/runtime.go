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

import "github.com/version-fox/vfox/internal/plugin"

// Version represents the version string of SDK runtime.
type Version string

// Runtime represents a runtime environment with its version and installation path.
type Runtime struct {
	Name    string  `json:"name"`
	Version Version `json:"version"`
	Path    string  `json:"path"`
}

// RuntimePackage represents a package of runtimes, including a main runtime and additional runtimes.
type RuntimePackage struct {
	*Runtime
	PackagePath string
	Additions   []*Runtime `json:"additions"`
}

type AvailableRuntime struct {
	Version Version
	Name    string
	Note    string
}

type AvailableRuntimePackage struct {
	*AvailableRuntime
	Additions []*AvailableRuntime
}

func convertAvailableHookResultItem2AvailableRuntimePackage(sdkName string, i []*plugin.AvailableHookResultItem) []*AvailableRuntimePackage {
	result := make([]*AvailableRuntimePackage, 0, len(i))
	for _, item := range i {
		runtimePackage := &AvailableRuntimePackage{
			AvailableRuntime: &AvailableRuntime{
				Name:    sdkName,
				Version: Version(item.Version),
				Note:    item.Note,
			},
			Additions: make([]*AvailableRuntime, 0, len(item.Addition)),
		}
		for _, addition := range item.Addition {
			runtimePackage.Additions = append(runtimePackage.Additions, &AvailableRuntime{
				Name:    addition.Name,
				Version: Version(addition.Version),
				Note:    addition.Note,
			})
		}
		result = append(result, runtimePackage)
	}
	return result
}

func convertRuntime2InstalledPackageItem(runtime *Runtime) *plugin.InstalledPackageItem {
	return &plugin.InstalledPackageItem{
		Name:    runtime.Name,
		Version: string(runtime.Version),
		Path:    runtime.Path,
	}
}
