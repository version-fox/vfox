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

package pathmeta

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/version-fox/vfox/internal/shared/logger"
	"github.com/version-fox/vfox/internal/shared/util"
)

type UserPaths struct {
	Home   string // ~/.vfox
	Temp   string // ~/.vfox/tmp (session temporary)
	Config string // ~/.vfox/config.yaml (user override config)
}

type SharedPaths struct {
	Root     string // /opt/vfox or C:\Program Files\vfox (from VFOX_ROOT or default)
	Installs string // ${SharedRoot}/installs (SDK unique install location)
	Plugins  string // ${SharedRoot}/plugins (plugins)
	Config   string // ${SharedRoot}/config.yaml (global config)
}

type WorkingPaths struct {
	Directory     string // Current working directory
	ProjectSdkDir string // .vfox/sdk (project-level)
	SessionSdkDir string // Session temporary shim path
	GlobalSdkDir  string // Global shim path in user home
}

type PathMeta struct {
	User       UserPaths
	Shared     SharedPaths
	Working    WorkingPaths
	Executable string // Executable path
}

const (
	HookCurTmpPath = "__VFOX_CURTMPPATH"

	oldVfoxDirPrefix    = ".version-fox" // Old vfox dir prefix
	vfoxDirPrefix       = ".vfox"
	pluginDirPrefix     = "plugin" // Shared plugins directory
	tmpDirPrefix        = "tmp"    // User-level temp
	installedDirPrefix  = "cache"  // Shared installed directory
	configFilePrefix    = "config.yaml"
	symlinkSdkDirPrefix = "sdks"

	ReadWriteAuth = 0755 // can write and read
)

func newTempPath(pid int) string {
	timestamp := util.GetBeginOfToday()
	name := fmt.Sprintf("%d-%d", timestamp, pid)
	return name
}

// NewPathMeta creates a new PathMeta instance based on the provided parameters and environment variables.
// If not provide sharedRoot, it will use userHome as sharedRoot.
func NewPathMeta(userHome, sharedRoot, currentDir string, pid int) (*PathMeta, error) {
	vfoxUserHome := filepath.Join(userHome, oldVfoxDirPrefix)
	// Compatibility check for old directory
	if !util.FileExists(vfoxUserHome) {
		vfoxUserHome = filepath.Join(userHome, vfoxDirPrefix)
	}
	// Use userHome as sharedRoot if not provided
	if len(sharedRoot) == 0 {
		sharedRoot = vfoxUserHome
	}
	meta := &PathMeta{
		User: UserPaths{
			Home:   vfoxUserHome,
			Temp:   filepath.Join(vfoxUserHome, tmpDirPrefix),
			Config: filepath.Join(vfoxUserHome, configFilePrefix),
		},
		Shared: SharedPaths{
			Root:     sharedRoot,
			Installs: filepath.Join(sharedRoot, installedDirPrefix),
			Plugins:  filepath.Join(sharedRoot, pluginDirPrefix),
			Config:   filepath.Join(sharedRoot, configFilePrefix),
		},
		Working: WorkingPaths{
			Directory:     currentDir,
			ProjectSdkDir: filepath.Join(vfoxDirPrefix, symlinkSdkDirPrefix),
			SessionSdkDir: generateSessionShimPath(filepath.Join(vfoxUserHome, tmpDirPrefix), pid),
			GlobalSdkDir:  filepath.Join(vfoxUserHome, symlinkSdkDirPrefix),
		},
		Executable: getExecutablePath(),
	}

	// Initialize necessary directories
	_ = os.MkdirAll(meta.User.Temp, ReadWriteAuth)
	_ = os.MkdirAll(meta.Working.GlobalSdkDir, ReadWriteAuth)
	_ = os.Mkdir(meta.Shared.Installs, ReadWriteAuth)
	_ = os.Mkdir(meta.Shared.Plugins, ReadWriteAuth)

	return meta, nil
}

func generateSessionShimPath(userTemp string, pid int) string {
	curTmpPath := os.Getenv(HookCurTmpPath)
	if curTmpPath == "" {
		name := newTempPath(pid)
		curTmpPath = filepath.Join(userTemp, name)
	}
	if !util.FileExists(curTmpPath) {
		err := os.MkdirAll(curTmpPath, ReadWriteAuth)
		if err != nil {
			logger.Errorf("create temp dir failed: %v", err)
		}
	}
	return curTmpPath
}

func getExecutablePath() string {
	exePath, err := os.Executable()
	if err != nil {
		logger.Errorf("get executable path failed: %v", err)
		return ""
	}
	return exePath
}
