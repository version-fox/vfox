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
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestDebugFlagCLI runs vfox list with each form of the root --debug flag and
// looks for the debug line that list prints.
func TestDebugFlagCLI(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		flag  string
		debug bool
	}{
		{"--debug", true},
		{"--debug=true", true},
		{"--debug=false", false},
	} {
		root := t.TempDir()
		command := exec.Command(executable, "-test.run=^TestPluginCLIProcess$", "--", tc.flag, "list")
		command.Dir = root
		command.Env = append(os.Environ(), "VFOX_PLUGIN_CLI_PROCESS=1",
			"HOME="+filepath.Join(root, "user"), "USERPROFILE="+filepath.Join(root, "user"),
			"VFOX_HOME="+filepath.Join(root, "shared"))
		// list exits with 1 when no SDK is installed.
		out, _ := command.CombinedOutput()
		if !strings.Contains(string(out), "you don't have any sdk installed yet") {
			t.Fatalf("vfox %s list did not run: %q", tc.flag, out)
		}
		if got := strings.Contains(string(out), "Loading all SDKs from"); got != tc.debug {
			t.Errorf("vfox %s list: debug output = %v, want %v: %q", tc.flag, got, tc.debug, out)
		}
	}
}
