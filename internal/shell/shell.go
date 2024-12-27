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

package shell

import (
	"strings"

	"github.com/version-fox/vfox/internal/env"
)

type Shell interface {
	// Activate generates a shell script to be placed in the shell's configuration file, which will set up initial
	// environment variables and set a hook to update the environment variables when needed.
	Activate() (string, error)

	// Export generates a string that can be used by the shell to set or unset the given environment variables. (The
	// input specifies environment variables to be unset by giving them a nil value.)
	Export(envs env.Vars) string
}

func NewShell(name string) Shell {
	switch strings.ToLower(name) {
	case "bash":
		return Bash
	case "zsh":
		return Zsh
	case "pwsh":
		return Pwsh
	case "fish":
		return Fish
	case "clink":
		return Clink
	case "nushell":
		return Nushell
	}
	return nil
}
