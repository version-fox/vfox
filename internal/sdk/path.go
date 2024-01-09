/*
 *    Copyright 2024 [lihan aooohan@gmail.com]
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

package sdk

import (
	"fmt"
	"os"
	"path/filepath"
)

type RecordSource string

const (
	GlobalRecordSource  RecordSource = "global"
	ProjectRecordSource RecordSource = "project"
	SessionRecordSource RecordSource = "session"
)

type PathMeta struct {
	TempPath       string
	ConfigPath     string
	SdkCachePath   string
	PluginPath     string
	ExecutablePath string
}

func newPathMeta() (*PathMeta, error) {
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("get user home dir error: %w", err)
	}
	pluginPath := filepath.Join(userHomeDir, ".version-fox", "plugin")
	configPath := filepath.Join(userHomeDir, ".version-fox")
	sdkCachePath := filepath.Join(userHomeDir, ".version-fox", "cache")
	tmpPath := filepath.Join(userHomeDir, ".version-fox", "temp")
	_ = os.MkdirAll(sdkCachePath, 0755)
	_ = os.MkdirAll(pluginPath, 0755)
	_ = os.MkdirAll(tmpPath, 0755)
	exePath, err := os.Executable()
	if err != nil {
		panic("Get executable path error")
	}
	return &PathMeta{
		TempPath:       tmpPath,
		ConfigPath:     configPath,
		SdkCachePath:   sdkCachePath,
		PluginPath:     pluginPath,
		ExecutablePath: exePath,
	}, nil
}
