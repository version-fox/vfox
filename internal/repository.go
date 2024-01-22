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

package internal

type RemotePluginInfo struct {
	Filename string `json:"name"`
	Author   string `json:"plugin_author"`
	Desc     string `json:"plugin_desc"`
	Name     string `json:"plugin_name"`
	Version  string `json:"plugin_version"`
	Sha256   string `json:"sha256"`
	Url      string `json:"url"`
}

type Category struct {
	Name    string              `json:"category"`
	Count   string              `json:"count"`
	Plugins []*RemotePluginInfo `json:"files"`
}
