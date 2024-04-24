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
	"errors"

	lua "github.com/yuin/gopher-lua"
)

type LuaCheckSum struct {
	Sha256 string `luai:"sha256"`
	Sha512 string `luai:"sha512"`
	Sha1   string `luai:"sha1"`
	Md5    string `luai:"md5"`
}

func (c *LuaCheckSum) Checksum() *Checksum {
	checksum := &Checksum{}

	if c.Sha256 != "" {
		checksum.Value = c.Sha256
		checksum.Type = "sha256"
	} else if c.Md5 != "" {
		checksum.Value = c.Md5
		checksum.Type = "md5"
	} else if c.Sha1 != "" {
		checksum.Value = c.Sha1
		checksum.Type = "sha1"
	} else if c.Sha512 != "" {
		checksum.Value = c.Sha512
		checksum.Type = "sha512"
	} else {
		return NoneChecksum
	}

	return checksum
}

type AvailableHookCtx struct {
	Args []string `luai:"args"`
}

type AvailableHookResultItem struct {
	Version string `luai:"version"`
	Note    string `luai:"note"`

	Addition []*Info `luai:"addition"`
}

type AvailableHookResult = []*AvailableHookResultItem

type PreInstallHookCtx struct {
	Version string `luai:"version"`
}

type PreInstallHookResultAdditionItem struct {
	Name   string `luai:"name"`
	Url    string `luai:"url"`
	Note   string `luai:"note"`
	Sha256 string `luai:"sha256"`
	Sha512 string `luai:"sha512"`
	Sha1   string `luai:"sha1"`
	Md5    string `luai:"md5"`
}

func (i *PreInstallHookResultAdditionItem) Info() *Info {
	sum := LuaCheckSum{
		Sha256: i.Sha256,
		Sha512: i.Sha512,
		Sha1:   i.Sha1,
		Md5:    i.Md5,
	}

	return &Info{
		Name:     i.Name,
		Version:  Version(""),
		Path:     i.Url,
		Note:     i.Note,
		Checksum: sum.Checksum(),
	}
}

type PreInstallHookResult struct {
	Version  string                              `luai:"version"`
	Url      string                              `luai:"url"`
	Note     string                              `luai:"note"`
	Sha256   string                              `luai:"sha256"`
	Sha512   string                              `luai:"sha512"`
	Sha1     string                              `luai:"sha1"`
	Md5      string                              `luai:"md5"`
	Addition []*PreInstallHookResultAdditionItem `luai:"addition"`
}

var ErrNoVersionProvided = errors.New("no version number provided")

func (i *PreInstallHookResult) Info() (*Info, error) {
	if i.Version == "" {
		return nil, ErrNoVersionProvided
	}

	sum := LuaCheckSum{
		Sha256: i.Sha256,
		Sha512: i.Sha512,
		Sha1:   i.Sha1,
		Md5:    i.Md5,
	}

	return &Info{
		Name:     "",
		Version:  Version(i.Version),
		Path:     i.Url,
		Note:     i.Note,
		Checksum: sum.Checksum(),
	}, nil
}

type PreUseHookCtx struct {
	Cwd             string           `luai:"cwd"`
	Scope           string           `luai:"scope"`
	Version         string           `luai:"version"`
	PreviousVersion string           `luai:"previousVersion"`
	InstalledSdks   map[string]*Info `luai:"installedSdks"`
}

type PreUseHookResult struct {
	Version string `luai:"version"`
}

type PostInstallHookCtx struct {
	RootPath string           `luai:"rootPath"`
	SdkInfo  map[string]*Info `luai:"sdkInfo"`
}

type EnvKeysHookCtx struct {
	Main *Info `luai:"main"`
	// TODO Will be deprecated in future versions
	Path    string           `luai:"path"`
	SdkInfo map[string]*Info `luai:"sdkInfo"`
}

type EnvKeysHookResultItem struct {
	Key   string `luai:"key"`
	Value string `luai:"value"`
}

type ParseLegacyFileHookCtx struct {
	Filepath             string         `luai:"filepath"`
	Filename             string         `luai:"filename"`
	GetInstalledVersions lua.LGFunction `luai:"getInstalledVersions"`
}

type ParseLegacyFileResult struct {
	Version string `luai:"version"`
}

type PreUninstallHookCtx struct {
	Main    *Info            `luai:"main"`
	SdkInfo map[string]*Info `luai:"sdkInfo"`
}

type LuaPluginInfo struct {
	Name              string   `luai:"name"`
	Version           string   `luai:"version"`
	Description       string   `luai:"description"`
	UpdateUrl         string   `luai:"updateUrl"`
	ManifestUrl       string   `luai:"manifestUrl"`
	Homepage          string   `luai:"homepage"`
	License           string   `luai:"license"`
	MinRuntimeVersion string   `luai:"minRuntimeVersion"`
	Notes             []string `luai:"notes"`
	LegacyFilenames   []string `luai:"legacyFilenames"`
}

// LuaRuntime represents the runtime information of the Lua environment.
type LuaRuntime struct {
	OsType        string `luai:"osType"`
	ArchType      string `luai:"archType"`
	Version       string `luai:"version"`
	PluginDirPath string `luai:"pluginDirPath"`
}
