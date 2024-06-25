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
	"fmt"
	"os"
	"path/filepath"

	"github.com/version-fox/vfox/internal/env"
	"github.com/version-fox/vfox/internal/util"
)

type PathMeta struct {
	TempPath string
	// Temporary directory for the current process
	CurTmpPath       string
	HomePath         string
	SdkCachePath     string
	PluginPath       string
	ExecutablePath   string
	WorkingDirectory string
	GlobalShimsPath  string
}

const (
	HookCurTmpPath = "__VFOX_CURTMPPATH"
)

func newPathMeta() (*PathMeta, error) {
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("get user home dir error: %w", err)
	}
	pluginPath := filepath.Join(userHomeDir, ".version-fox", "plugin")
	homePath := filepath.Join(userHomeDir, ".version-fox")
	sdkCachePath := filepath.Join(userHomeDir, ".version-fox", "cache")
	tmpPath := filepath.Join(userHomeDir, ".version-fox", "temp")
	_ = os.MkdirAll(sdkCachePath, 0755)
	_ = os.MkdirAll(pluginPath, 0755)
	_ = os.MkdirAll(tmpPath, 0755)
	exePath, err := os.Executable()
	if err != nil {
		return nil, err
	}
	curTmpPath := os.Getenv(HookCurTmpPath)
	if curTmpPath == "" {
		pid := env.GetPid()
		timestamp := util.GetBeginOfToday()
		name := fmt.Sprintf("%d-%d", timestamp, pid)
		curTmpPath = filepath.Join(tmpPath, name)
	}
	if !util.FileExists(curTmpPath) {
		err = os.Mkdir(curTmpPath, 0755)
		if err != nil {
			return nil, fmt.Errorf("create temp dir failed: %w", err)
		}
	}

	globalShimsPath := filepath.Join(homePath, "shims")
	_ = os.MkdirAll(globalShimsPath, 0777)

	workingDirectory, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("get current working directory failed: %w", err)
	}

	return &PathMeta{
		TempPath:         tmpPath,
		CurTmpPath:       curTmpPath,
		HomePath:         homePath,
		SdkCachePath:     sdkCachePath,
		PluginPath:       pluginPath,
		ExecutablePath:   exePath,
		WorkingDirectory: workingDirectory,
		GlobalShimsPath:  globalShimsPath,
	}, nil
}
