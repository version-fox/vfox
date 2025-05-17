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

package base

import (
	"errors"
	"path/filepath"

	"github.com/version-fox/vfox/internal/util"
)

type Version string

type CheckSum struct {
	Sha256 string `json:"sha256"`
	Sha512 string `json:"sha512"`
	Sha1   string `json:"sha1"`
	Md5    string `json:"md5"`
}

func (c *CheckSum) Checksum() *util.Checksum {
	checksum := &util.Checksum{}

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
		return util.NoneChecksum
	}

	return checksum
}

type Info struct {
	Name     string            `json:"name"`
	Version  string            `json:"version"`
	Path     string            `json:"path"`
	Headers  map[string]string `json:"headers"`
	Note     string            `json:"note"`
	Checksum *util.Checksum
}

func (i *Info) Clone() *Info {
	headers := make(map[string]string, len(i.Headers))
	for k, v := range i.Headers {
		headers[k] = v
	}
	return &Info{
		Name:     i.Name,
		Version:  i.Version,
		Path:     i.Path,
		Headers:  headers,
		Note:     i.Note,
		Checksum: i.Checksum,
	}
}

func (i *Info) Label() string {
	return i.Name + "@" + string(i.Version)
}

func (i *Info) StoragePath(parentDir string) string {
	if i.Version == "" {
		return filepath.Join(parentDir, i.Name)
	}
	return filepath.Join(parentDir, i.Name+"-"+string(i.Version))
}

type AvailableHookCtx struct {
	Args []string `json:"args"`
}

type AvailableHookResultItem struct {
	Version string `json:"version"`
	Note    string `json:"note"`

	Addition []*Info `json:"addition"`
}

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
	sum := CheckSum{
		Sha256: i.Sha256,
		Sha512: i.Sha512,
		Sha1:   i.Sha1,
		Md5:    i.Md5,
	}

	return &Info{
		Name:     i.Name,
		Version:  "",
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

	sum := CheckSum{
		Sha256: i.Sha256,
		Sha512: i.Sha512,
		Sha1:   i.Sha1,
		Md5:    i.Md5,
	}

	return &Info{
		Name:     "",
		Version:  i.Version,
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
	Version Version `json:"version"`
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
	Filepath             string           `json:"filepath"`
	Filename             string           `json:"filename"`
	GetInstalledVersions func() []Version `json:"getInstalledVersions"`
}

type ParseLegacyFileResult struct {
	Version Version `json:"version"`
}

type PreUninstallHookCtx struct {
	Main    *Info            `json:"main"`
	SdkInfo map[string]*Info `json:"sdkInfo"`
}

type PluginInfo struct {
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

// RuntimeInfo represents the runtime information of the current exec environment.
type RuntimeInfo struct {
	OsType        string `json:"osType"`
	ArchType      string `json:"archType"`
	Version       string `json:"version"`
	PluginDirPath string `json:"pluginDirPath"`
}

type Package struct {
	Main      *Info
	Additions []*Info
}

func (p *Package) Clone() *Package {
	main := p.Main.Clone()
	additions := make([]*Info, len(p.Additions))
	for i, a := range p.Additions {
		additions[i] = a.Clone()
	}
	return &Package{
		Main:      main,
		Additions: additions,
	}
}

var ErrNoResultProvide = errors.New("no result provided")

type Plugin interface {
	Available(ctx *AvailableHookCtx) ([]*AvailableHookResultItem, error)

	PreInstall(ctx *PreInstallHookCtx) (*PreInstallHookResult, error)
	PostInstall(ctx *PostInstallHookCtx) error
	PreUninstall(ctx *PreUninstallHookCtx) error
	PreUse(ctx *PreUseHookCtx) (*PreUseHookResult, error)

	ParseLegacyFile(ctx *ParseLegacyFileHookCtx) (*ParseLegacyFileResult, error)
	EnvKeys(ctx *EnvKeysHookCtx) ([]*EnvKeysHookResultItem, error)

	HasFunction(name string) bool
	Close()
}
