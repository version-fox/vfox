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

package printer

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

// A pty that reaches a native program as a pipe, as Git Bash in mintty gives it,
// cannot be read a key at a time by the keyboard library. stty puts it in raw
// mode instead: no line buffering, no echo, and no signals from Ctrl-C, which then
// arrive as a key. stty comes with Git for Windows, next to bash.

// runStty runs stty on terminal with args, and writes its output to out, which may
// be nil.
func runStty(terminal *os.File, out io.Writer, args ...string) error {
	path, err := exec.LookPath("stty")
	if err != nil {
		return err
	}
	var stderr bytes.Buffer
	cmd := exec.Command(path, args...)
	cmd.Stdin = terminal
	cmd.Stdout = out
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("stty %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return nil
}

// rawModeAvailable reports whether stdin can be switched with stty.
func rawModeAvailable() bool {
	var mode bytes.Buffer
	return runStty(os.Stdin, &mode, "-g") == nil
}

// enterRawMode puts terminal in raw mode, and returns the function that restores
// the mode it had.
func enterRawMode(terminal *os.File) (func(), error) {
	var saved bytes.Buffer
	if err := runStty(terminal, &saved, "-g"); err != nil {
		return nil, err
	}
	state := strings.TrimSpace(saved.String())
	if state == "" {
		return nil, errors.New("stty printed no terminal mode")
	}
	if err := runStty(terminal, nil, "-icanon", "-echo", "-isig", "min", "1", "time", "0"); err != nil {
		return nil, err
	}
	return func() {
		if err := runStty(terminal, nil, state); err != nil {
			fmt.Fprintf(os.Stderr, "could not restore the terminal mode: %v\n", err)
		}
	}, nil
}
