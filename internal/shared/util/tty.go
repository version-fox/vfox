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

package util

import (
	"os"

	"golang.org/x/term"
)

// IsTTY checks if the process is running in a TTY (interactive terminal). On
// Windows, Git Bash in mintty counts as one: its pty reaches native programs as
// a pipe, and isCygwinTerminal reports that pipe as a terminal.
func IsTTY() bool {
	return term.IsTerminal(int(os.Stdout.Fd())) || isCygwinTerminal(os.Stdout.Fd())
}

// CanReadKeys reports whether single key presses can be read from stdin, as
// arrow-key selection and y/n prompts need. On Windows the key reader needs a
// console, so a stdin that is a pipe (Git Bash in mintty) gets line input instead.
func CanReadKeys() bool {
	return canReadKeysFrom(os.Stdin.Fd())
}
