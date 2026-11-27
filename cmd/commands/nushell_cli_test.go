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

package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// nushellOrSkip finds Nushell for the tests that start a real session. CI
// installs it and sets VFOX_TEST_NUSHELL=1, so that an installation that went
// wrong fails the job instead of skipping the tests.
func nushellOrSkip(t *testing.T) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("the Nushell session is exercised on Unix only")
	}
	nu, err := exec.LookPath("nu")
	if err != nil {
		if os.Getenv("VFOX_TEST_NUSHELL") == "1" {
			t.Fatalf("VFOX_TEST_NUSHELL=1 asks for Nushell, but nu is not on PATH: %v", err)
		}
		t.Skip("nu is not installed; install Nushell to run the native Nushell tests")
	}
	return nu
}

// buildVfox builds the vfox command of this module. The vfox.nu that vfox
// generates runs the very file that generated it, with arguments the test binary
// does not take, so the tests cannot stand in for it the way TestCLIProcess does.
func buildVfox(t *testing.T) string {
	t.Helper()
	goTool, err := exec.LookPath("go")
	if err != nil {
		t.Fatalf("the go command is needed to build vfox for this test: %v", err)
	}
	name := "vfox"
	if runtime.GOOS == "windows" {
		// go build takes an explicit -o name as it is, and Windows runs only .exe files.
		name += ".exe"
	}
	bin := filepath.Join(t.TempDir(), name)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, goTool, "build", "-buildvcs=false", "-o", bin, ".")
	cmd.Dir = filepath.Join("..", "..")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
	return bin
}

// nushellState is what nushellProbe prints about a session.
type nushellState struct {
	Stage string
	Path  []string
	Orig  string
	Shell string
}

// nushellProbe runs in a Nushell that has loaded the configuration `vfox
// activate` asked for. It reports the session as it starts, puts a directory in
// front of PATH the way `pnpm setup` or a virtualenv does after the start, runs
// the prompt hooks the way the REPL does before it shows the next prompt, and
// reports again. `do --env` keeps what a hook changes in this session. The
// format verb stands for the directory that goes in front.
const nushellProbe = `
def state [stage: string] {
  {stage: $stage, path: $env.PATH, orig: ($env.__VFOX_ORIG_PATH? | default ""), shell: ($env.__VFOX_SHELL? | default "")} | to json --raw
}
print (state "started")
$env.PATH = ($env.PATH | prepend '%s')
for hook in $env.config.hooks.pre_prompt { do --env $hook }
print (state "prompt")
`

// The generated vfox.nu records the PATH its session starts with in
// __VFOX_ORIG_PATH, because Nushell cannot evaluate what `vfox activate` prints
// and so never receives the line the other shells get. Whether `vfox env` sees
// that baseline cannot be told from the text of the script: it shows in the
// order of PATH that Nushell ends up with.
func TestNushellSessionUsesTheBaselineItRecorded(t *testing.T) {
	nu := nushellOrSkip(t)
	vfox := buildVfox(t)

	f := newCLIFixture(t, false)
	f.installOnDisk(t, "alpha", "1.0.0", "alphatool")
	must(t, os.WriteFile(filepath.Join(f.userHome, ".vfox.toml"), []byte("[tools]\nalpha = \"1.0.0\"\n"), 0644))
	inherited := filepath.Join(f.root, "usr-local", "bin") // what the machine put in front of vfox's directory
	added := filepath.Join(f.root, "pnpm", "bin")          // what a tool puts there once the shell runs
	vfoxBin := filepath.Join(f.userHome, "sdks", "alpha", "bin")
	for _, dir := range []string{inherited, added} {
		must(t, os.MkdirAll(dir, 0755))
	}
	if strings.Contains(added, "'") {
		t.Fatalf("%s cannot go into a single-quoted Nushell string", added)
	}

	// Nothing of vfox from a shell that is already running, and a Nushell that
	// keeps its files below the fixture.
	f.env["__VFOX_SHELL"] = "-"
	f.env["__VFOX_CURTMPPATH"] = "-"
	f.env["XDG_CONFIG_HOME"] = filepath.Join(f.root, "config")
	run := func(t *testing.T, name string, args ...string) cliResult {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		return runCommand(t, f.command(ctx, name, args...))
	}

	// Where this Nushell keeps its configuration, whatever the platform calls it.
	configDir := strings.TrimSpace(run(t, nu, "--no-config-file", "--commands", "print $nu.default-config-dir").mustSucceed(t, "nu").stdout)
	if !strings.HasPrefix(configDir, f.root) {
		t.Fatalf("Nushell's configuration directory %q is outside the fixture %q: the test would write into a real home", configDir, f.root)
	}
	must(t, os.MkdirAll(configDir, 0755))

	// The documented setup: vfox activate nushell $nu.default-config-dir | save --append $nu.config-path
	activated := run(t, vfox, "activate", "nushell", configDir).mustSucceed(t, "vfox activate nushell")
	configNu := filepath.Join(configDir, "config.nu")
	must(t, os.WriteFile(configNu, []byte(activated.stdout), 0644))
	if !exists(filepath.Join(configDir, "vfox.nu")) {
		t.Fatalf("vfox activate nushell wrote no vfox.nu to %s", configDir)
	}

	// A new terminal. The machine's directories come first and vfox's own comes
	// later, where Windows puts the user PATH and macOS path_helper the rest.
	f.env["PATH"] = pathList(inherited, vfoxBin, f.system)
	start := func(t *testing.T) (started, prompt nushellState) {
		t.Helper()
		// --config: Nushell loads no configuration for --commands otherwise.
		r := run(t, nu, "--config", configNu, "--commands", fmt.Sprintf(nushellProbe, added)).mustSucceed(t, "nu")
		var states []nushellState
		for _, line := range strings.Split(r.stdout, "\n") {
			if !strings.HasPrefix(line, "{") {
				continue
			}
			var state nushellState
			if err := json.Unmarshal([]byte(line), &state); err != nil {
				t.Fatalf("nu printed %q: %v\nstdout: %s\nstderr: %s", line, err, r.stdout, r.stderr)
			}
			states = append(states, state)
		}
		if len(states) != 2 || states[0].Stage != "started" || states[1].Stage != "prompt" {
			t.Fatalf("nu reported %+v, want the states \"started\" and \"prompt\"\nstdout: %s\nstderr: %s", states, r.stdout, r.stderr)
		}
		return states[0], states[1]
	}
	check := func(t *testing.T, what string, got nushellState, want ...string) {
		t.Helper()
		if strings.Join(got.Path, "\n") != strings.Join(want, "\n") {
			t.Errorf("%s: PATH = %v, want %v", what, got.Path, want)
		}
	}

	t.Run("the default mode moves what the session started with behind vfox", func(t *testing.T) {
		started, prompt := start(t)
		if started.Shell != "nushell" {
			t.Errorf("__VFOX_SHELL = %q", started.Shell)
		}
		// Recorded as the shell found it, vfox's own directory included.
		if want := pathList(inherited, vfoxBin, f.system); started.Orig != want {
			t.Errorf("__VFOX_ORIG_PATH = %q, want %q", started.Orig, want)
		}
		check(t, "when the session started", started, vfoxBin, inherited, f.system)
		// What came after the start is not part of the baseline, so it stays in front.
		check(t, "after a directory was put in front and the prompt hook ran", prompt, added, vfoxBin, inherited, f.system)
	})

	t.Run("the legacy mode keeps everything in front of vfox in front", func(t *testing.T) {
		f.writeUserConfig(t, "path:\n  mode: legacy\n")
		started, prompt := start(t)
		check(t, "when the session started", started, inherited, vfoxBin, f.system)
		check(t, "after a directory was put in front and the prompt hook ran", prompt, added, inherited, vfoxBin, f.system)
	})
}
