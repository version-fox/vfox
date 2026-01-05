/*
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
 */

package internal

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/version-fox/vfox/internal/env"
	"github.com/version-fox/vfox/internal/logger"
	"github.com/version-fox/vfox/internal/util"
)

type UserPaths struct {
	Home   string // ~/.vfox
	Temp   string // ~/.vfox/tmp (session temporary)
	Cache  string // ~/.vfox/cache (download cache)
	Config string // ~/.vfox/config.yaml (user override config)
}

type SharedPaths struct {
	Root     string // /opt/vfox or C:\Program Files\vfox (from VFOX_ROOT or default)
	Installs string // ${SharedRoot}/installs (SDK unique install location)
	Plugins  string // ${SharedRoot}/plugins (plugins)
	Config   string // ${SharedRoot}/config.yaml (global config)
}

type WorkingPaths struct {
	Directory   string // Current working directory
	ProjectShim string // .vfox/sdk (project-level)
	SessionShim string // Session temporary shim path
	GlobalShim  string // Global shim path in user home
}

type PathMeta struct {
	User       UserPaths
	Shared     SharedPaths
	Working    WorkingPaths
	Executable string // Executable path
}

const (
	HookCurTmpPath = "__VFOX_CURTMPPATH"

	vfoxDirPrefix      = ".vfox"
	pluginDirPrefix    = "plugins"   // Shared plugins directory
	cacheDirPrefix     = "cache"     // User-level cache
	tmpDirPrefix       = "tmp"       // User-level temp
	installedDirPrefix = "installed" // Shared installed directory
	configFilePrefix   = "config.yaml"
	shimDirPrefix      = "shims"
)

func newTempPath() string {
	pid := env.GetPid()
	timestamp := util.GetBeginOfToday()
	name := fmt.Sprintf("%d-%d", timestamp, pid)
	return name
}

func newPathMeta() (*PathMeta, error) {
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("get user home dir error: %w", err)
	}

	userHome := getVfoxUserHomeDir(userHomeDir)

	// Get shared root (by priority)
	sharedRoot := env.GetVfoxRoot()
	if sharedRoot == "" {
		// use UserHome as sharedRoot if not set VFOX_ROOT
		sharedRoot = userHome
	}

	meta := &PathMeta{
		User: UserPaths{
			Home:   userHome,
			Temp:   filepath.Join(userHome, tmpDirPrefix),
			Cache:  filepath.Join(userHome, cacheDirPrefix),
			Config: filepath.Join(userHome, configFilePrefix),
		},
		Shared: SharedPaths{
			Root:     sharedRoot,
			Installs: filepath.Join(sharedRoot, installedDirPrefix),
			Plugins:  filepath.Join(sharedRoot, pluginDirPrefix),
			Config:   filepath.Join(sharedRoot, configFilePrefix),
		},
		Working: WorkingPaths{
			Directory:   getWorkingDirectory(),
			ProjectShim: filepath.Join(vfoxDirPrefix, shimDirPrefix),
			SessionShim: generateSessionShimPath(filepath.Join(userHome, tmpDirPrefix)),
			GlobalShim:  filepath.Join(userHome, shimDirPrefix),
		},
		Executable: getExecutablePath(),
	}

	// Initialize necessary directories
	_ = os.MkdirAll(meta.User.Temp, 0755)
	_ = os.MkdirAll(meta.User.Cache, 0755)
	_ = os.MkdirAll(meta.Working.GlobalShim, 0755)
	_ = os.Mkdir(meta.Shared.Installs, 0755)
	_ = os.Mkdir(meta.Shared.Plugins, 0755)

	return meta, nil
}

func getVfoxUserHomeDir(UserHome string) string {
	vfoxHomeDir := env.GetVfoxHome()
	if len(vfoxHomeDir) != 0 {
		return vfoxHomeDir
	}
	return filepath.Join(UserHome, vfoxDirPrefix)
}

func getWorkingDirectory() string {
	wd, err := os.Getwd()
	if err != nil {
		logger.Errorf("get current working directory failed: %v", err)
		return ""
	}
	return wd
}

func generateSessionShimPath(userTemp string) string {
	curTmpPath := os.Getenv(HookCurTmpPath)
	if curTmpPath == "" {
		name := newTempPath()
		curTmpPath = filepath.Join(userTemp, name)
	}
	if !util.FileExists(curTmpPath) {
		err := os.MkdirAll(curTmpPath, 0755)
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
