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

// IsIDEEnvironmentResolution detects if the current shell session was launched by an IDE
// for the purpose of environment variable resolution. This is useful to avoid certain
// shell initialization behaviors that might interfere with IDE environment detection.
//
// Supported IDEs:
//   - Visual Studio Code: Detects via VSCODE_RESOLVING_ENVIRONMENT environment variable
//     Reference: https://code.visualstudio.com/docs/configure/command-line#_how-do-i-detect-when-a-shell-was-launched-by-vs-code
//   - JetBrains IDEs (IntelliJ IDEA, PyCharm, etc.): Detects via INTELLIJ_ENVIRONMENT_READER environment variable
//     Reference: https://youtrack.jetbrains.com/articles/SUPPORT-A-1727/Shell-Environment-Loading
//
// Returns true if any of the supported IDE environment resolution indicators are present.
func IsIDEEnvironmentResolution() bool {
	return os.Getenv("VSCODE_RESOLVING_ENVIRONMENT") != "" ||
		os.Getenv("INTELLIJ_ENVIRONMENT_READER") != ""
}

// IsMultiplexerEnvironment detects if the current shell session is running inside a multiplexer
// or other environment where new panes/windows should have isolated environments.
// This includes tmux, screen, and potentially other terminal multiplexers.
//
// Supported multiplexers:
//   - tmux: Detects via TMUX environment variable
//   - screen: Detects via STY environment variable (GNU screen)
//   - Windows Terminal: Detects via WT_SESSION environment variable
//
// Returns true if running in a multiplexer environment.
func IsMultiplexerEnvironment() bool {
	return os.Getenv("TMUX") != ""
}

func GetPid() int {
	if IsHookEnv() {
		if pid := os.Getenv(PidFlag); pid != "" {
			p, _ := strconv.Atoi(pid) // Convert pid from string to int
			return p
		}
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
