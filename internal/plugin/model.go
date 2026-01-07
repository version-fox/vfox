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

package plugin

import (
	"github.com/version-fox/vfox/internal/shared"
)

// PreInstallPackageItem represents the package information returned by PreInstall hook
type PreInstallPackageItem struct {
	Name          string            `json:"name"`
	Version       string            `json:"version"`
	Path          string            `json:"url"`     // optional, remote URL or local file path
	Headers       map[string]string `json:"headers"` // optional, request headers for downloading
	Note          string            `json:"note"`    // optional, additional note
	*CheckSumItem                   // optional, checksum information
}

func (p *PreInstallPackageItem) Label() string {
	return p.Name + "@" + p.Version
}

type CheckSumItem struct {
	Sha256 string `json:"sha256"`
	Sha512 string `json:"sha512"`
	Sha1   string `json:"sha1"`
	Md5    string `json:"md5"`
}

func (c *CheckSumItem) Checksum() *shared.Checksum {
	checksum := &shared.Checksum{}

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
		return shared.NoneChecksum
	}

	return checksum
}

type AvailableHookCtx struct {
	Args []string `json:"args"`
}

type AvailableHookResultItem struct {
	Version string `json:"version"`
	Note    string `json:"note"`

	Addition []*PreInstallPackageItem `json:"addition"`
}

type PreInstallHookCtx struct {
	Version string `json:"version"`
}

type PreInstallHookResult struct {
	*PreInstallPackageItem
	Addition []*PreInstallPackageItem `json:"addition"`
}

type PreUseHookCtx struct {
	Cwd             string                           `json:"cwd"`
	Scope           string                           `json:"scope"`
	Version         string                           `json:"version"`
	PreviousVersion string                           `json:"previousVersion"`
	InstalledSdks   map[string]*InstalledPackageItem `json:"installedSdks"`
}

type PreUseHookResult struct {
	Version string `json:"version"`
}

type PostInstallHookCtx struct {
	RootPath string                           `json:"rootPath"`
	SdkInfo  map[string]*InstalledPackageItem `json:"sdkInfo"`
}

// InstalledPackageItem represents the installed SDK base information to export to the plugins.
type InstalledPackageItem struct {
	Path    string `json:"path"`
	Version string `json:"version"`
	Name    string `json:"name"`
}

type EnvKeysHookCtx struct {
	Main    *InstalledPackageItem            `json:"main"`
	Path    string                           `json:"path"` // TODO Will be deprecated in future versions
	SdkInfo map[string]*InstalledPackageItem `json:"sdkInfo"`
}

type EnvKeysHookResultItem struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type ParseLegacyFileHookCtx struct {
	Filepath             string          `json:"filepath"`
	Filename             string          `json:"filename"`
	GetInstalledVersions func() []string `json:"getInstalledVersions"`
	// Support three strategies:
	// 1. latest_installed: use the latest installed version
	// 2. latest_available: use the latest available version
	// 3. specified: use the specified version in the legacy file
	// default: specified
	Strategy string `json:"strategy"`
}

type ParseLegacyFileResult struct {
	Version string `json:"version"`
}

type PreUninstallHookCtx struct {
	Main    *InstalledPackageItem            `json:"main"`
	SdkInfo map[string]*InstalledPackageItem `json:"sdkInfo"`
}

// RuntimeInfo represents the runtime information of the current exec environment.
type RuntimeInfo struct {
	OsType        string `json:"osType"`
	ArchType      string `json:"archType"`
	Version       string `json:"version"`
	PluginDirPath string `json:"pluginDirPath"`
}
