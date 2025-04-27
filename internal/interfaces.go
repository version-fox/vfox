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

import (
	"errors"

	lua "github.com/yuin/gopher-lua"
)

type LuaCheckSum struct {
	Sha256 string `json:"sha256"`
	Sha512 string `json:"sha512"`
	Sha1   string `json:"sha1"`
	Md5    string `json:"md5"`
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
	Args []string `json:"args"`
}

type AvailableHookResultItem struct {
	Version string `json:"version"`
	Note    string `json:"note"`

	Addition []*Info `json:"addition"`
}

type AvailableHookResult = []*AvailableHookResultItem

type PreInstallHookCtx struct {
	Version string `json:"version"`
}

type PreInstallHookResultAdditionItem struct {
	Name    string            `json:"name"`
	Url     string            `json:"url"`
	Headers map[string]string `json:"headers"`
	Note    string            `json:"note"`
	Sha256  string            `json:"sha256"`
	Sha512  string            `json:"sha512"`
	Sha1    string            `json:"sha1"`
	Md5     string            `json:"md5"`
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
		Headers:  i.Headers,
		Note:     i.Note,
		Checksum: sum.Checksum(),
	}
}

type PreInstallHookResult struct {
	Version  string                              `json:"version"`
	Url      string                              `json:"url"`
	Headers  map[string]string                   `json:"headers"`
	Note     string                              `json:"note"`
	Sha256   string                              `json:"sha256"`
	Sha512   string                              `json:"sha512"`
	Sha1     string                              `json:"sha1"`
	Md5      string                              `json:"md5"`
	Addition []*PreInstallHookResultAdditionItem `json:"addition"`
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
		Headers:  i.Headers,
		Note:     i.Note,
		Checksum: sum.Checksum(),
	}, nil
}

type PreUseHookCtx struct {
	Cwd             string           `json:"cwd"`
	Scope           string           `json:"scope"`
	Version         string           `json:"version"`
	PreviousVersion string           `json:"previousVersion"`
	InstalledSdks   map[string]*Info `json:"installedSdks"`
}

type PreUseHookResult struct {
	Version string `json:"version"`
}

type PostInstallHookCtx struct {
	RootPath string           `json:"rootPath"`
	SdkInfo  map[string]*Info `json:"sdkInfo"`
}

type EnvKeysHookCtx struct {
	Main *Info `json:"main"`
	// TODO Will be deprecated in future versions
	Path    string           `json:"path"`
	SdkInfo map[string]*Info `json:"sdkInfo"`
}

type EnvKeysHookResultItem struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type ParseLegacyFileHookCtx struct {
	Filepath             string         `json:"filepath"`
	Filename             string         `json:"filename"`
	GetInstalledVersions lua.LGFunction `json:"getInstalledVersions"`
}

type ParseLegacyFileResult struct {
	Version string `json:"version"`
}

type PreUninstallHookCtx struct {
	Main    *Info            `json:"main"`
	SdkInfo map[string]*Info `json:"sdkInfo"`
}

type LuaPluginInfo struct {
	Name              string   `json:"name"`
	Version           string   `json:"version"`
	Description       string   `json:"description"`
	UpdateUrl         string   `json:"updateUrl"`
	ManifestUrl       string   `json:"manifestUrl"`
	Homepage          string   `json:"homepage"`
	License           string   `json:"license"`
	MinRuntimeVersion string   `json:"minRuntimeVersion"`
	Notes             []string `json:"notes"`
	LegacyFilenames   []string `json:"legacyFilenames"`
}

// LuaRuntime represents the runtime information of the Lua environment.
type LuaRuntime struct {
	OsType        string `json:"osType"`
	ArchType      string `json:"archType"`
	Version       string `json:"version"`
	PluginDirPath string `json:"pluginDirPath"`
}
