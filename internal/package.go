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

import (
	"path/filepath"

	"github.com/version-fox/vfox/internal/plugin"
)

type Package struct {
	Main      *Info
	Additions []*Info
}

type Info struct {
	Name     string
	Version  Version
	Path     string
	Note     string
	Checksum *Checksum
}

func (i *Info) label() string {
	return i.Name + "@" + string(i.Version)
}

func (i *Info) storagePath(parentDir string) string {
	if i.Version == "" {
		return filepath.Join(parentDir, i.Name)
	}
	return filepath.Join(parentDir, i.Name+"-"+string(i.Version))
}

func NewLuaSDKInfo(info *Info) *plugin.LuaSDKInfo {
	return &plugin.LuaSDKInfo{
		Name:    info.Name,
		Version: string(info.Version),
		Path:    info.Path,
		Note:    info.Note,
	}
}
