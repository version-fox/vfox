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

package cmd

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCompletionScriptsUseCurrentCliFlag(t *testing.T) {
	t.Parallel()

	scriptPaths := []string{
		filepath.Join("..", "completions", "bash_autocomplete"),
		filepath.Join("..", "completions", "zsh_autocomplete"),
		filepath.Join("..", "completions", "powershell_autocomplete.ps1"),
	}

	for _, scriptPath := range scriptPaths {
		scriptPath := scriptPath

		t.Run(filepath.Base(scriptPath), func(t *testing.T) {
			t.Parallel()

			content, err := os.ReadFile(scriptPath)
			if err != nil {
				t.Fatalf("read completion script %q: %v", scriptPath, err)
			}

			script := string(content)
			if strings.Contains(script, "--generate-bash-completion") {
				t.Fatalf("completion script %q still references deprecated completion flag", scriptPath)
			}
			if !strings.Contains(script, "--generate-shell-completion") {
				t.Fatalf("completion script %q does not reference current completion flag", scriptPath)
			}
		})
	}
}

func TestGenerateShellCompletionFlagProducesSuggestions(t *testing.T) {
	t.Parallel()

	command := newCmd()
	output := &bytes.Buffer{}
	command.app.Writer = output
	command.app.ErrWriter = output

	err := command.app.Run(context.Background(), []string{"vfox", "--generate-shell-completion"})
	if err != nil {
		t.Fatalf("run shell completion: %v", err)
	}

	suggestions := output.String()
	if !strings.Contains(suggestions, "install") {
		t.Fatalf("shell completion output missing install command: %q", suggestions)
	}
	if !strings.Contains(suggestions, "use") {
		t.Fatalf("shell completion output missing use command: %q", suggestions)
	}
}
