/*
 *    Copyright 2025 Han Li and contributors
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

package internal

const (
	pluginRegistryAddress = "https://vfox-plugins.lhan.me"
)

// RegistryIndex is the index of the registry
type RegistryIndex []*RegistryIndexItem

// RegistryIndexItem is the item in the registry index
type RegistryIndexItem struct {
	Name     string `json:"name"`
	Desc     string `json:"desc"`
	Homepage string `json:"homepage"`
}

// RegistryPluginManifest is the manifest of a remote plugin
type RegistryPluginManifest struct {
	Name              string `json:"name"`
	Version           string `json:"version"`
	License           string `json:"license"`
	Author            string `json:"author"`
	DownloadUrl       string `json:"downloadUrl"`
	MinRuntimeVersion string `json:"minRuntimeVersion"`
}
