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

package env

import (
	"os"
	"strconv"
)

const (
	HomeFromEnv   = "VFOX_HOME"
	PluginFromEnv = "VFOX_PLUGIN"
	CacheFromEnv  = "VFOX_CACHE"
	TempFromEnv   = "VFOX_TEMP"

	HookFlag = "__VFOX_SHELL"
	PidFlag  = "__VFOX_PID"
)

func IsHookEnv() bool {
	return os.Getenv(HookFlag) != ""
}

func GetPid() int {
	if pid := os.Getenv(PidFlag); pid != "" {
		p, _ := strconv.Atoi(pid) // Convert pid from string to int
		return p
	}
	return os.Getppid()
}

func GetVfoxHome() string {
	return os.Getenv(HomeFromEnv)
}

func GetVfoxPlugin() string {
	return os.Getenv(PluginFromEnv)
}

func GetVfoxCache() string {
	return os.Getenv(CacheFromEnv)
}

func GetVfoxTemp() string {
	return os.Getenv(TempFromEnv)
}
