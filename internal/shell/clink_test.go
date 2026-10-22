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

package shell

import (
	"fmt"
	"maps"
	"os"
	"slices"
	"strings"
	"testing"

	lua "github.com/yuin/gopher-lua"

	"github.com/version-fox/vfox/internal/env"
)

// Tests of clink_vfox.lua. See clink_harness_test.go for how the script is run.

func TestClinkStartup(t *testing.T) {
	rt := newClinkRuntime(t)
	// The shell was started from another vfox session, which left its state in
	// the environment.
	rt.env[env.HookFlag] = "bash"
	rt.env[env.PidFlag] = "1111"
	rt.env[curTmpPathFlag] = `C:\Users\tester\.vfox\tmp\1111`
	rt.env["PATH"] = `C:\Windows`
	nodePath, sessionDir := `C:\Users\tester\.vfox\tmp\4242\nodejs\bin;C:\Windows`, `C:\Users\tester\.vfox\tmp\4242`
	rt.outputs["vfox activate clink"] = activateOutput(t, env.Vars{
		"PATH":         &nodePath,
		curTmpPathFlag: &sessionDir,
	})

	rt.load()

	// Starting the shell runs vfox activate and the cleanup, and nothing slow
	// such as vfox available or vfox ls: those wait for the first completion.
	if got, want := rt.commands(), []string{"vfox activate clink", "vfox env -c"}; !slices.Equal(got, want) {
		t.Fatalf("commands = %q, want %q", got, want)
	}

	// vfox activate reads the hook state from its environment, so it has to see
	// one that belongs to this shell and no session directory of another.
	activate := rt.started[0].env
	for name, want := range map[string]string{env.HookFlag: "clink", env.PidFlag: fmt.Sprint(clinkPid)} {
		if activate[name] != want {
			t.Errorf("%s = %q when vfox activate runs, want %q", name, activate[name], want)
		}
	}
	if value, ok := activate[curTmpPathFlag]; ok {
		t.Errorf("%s = %q when vfox activate runs, want it unset", curTmpPathFlag, value)
	}

	// The variables it prints are applied. The line that runs the cleanup is not.
	want := map[string]string{
		"USERPROFILE":  clinkUserProfile,
		env.HookFlag:   "clink",
		env.PidFlag:    fmt.Sprint(clinkPid),
		curTmpPathFlag: sessionDir,
		"PATH":         nodePath,
	}
	if !maps.Equal(rt.env, want) {
		t.Errorf("environment = %v, want %v", rt.env, want)
	}

	priorities, ok := rt.callFake("prompt_filter_priorities").(*lua.LTable)
	if !ok || priorities.Len() != 1 || priorities.RawGetInt(1) != lua.LNumber(30) {
		t.Errorf("prompt filter priorities = %v, want one filter with priority 30", priorities)
	}
}

func TestClinkPromptUpdatesEnvironment(t *testing.T) {
	rt := loadedClink(t)
	rt.env["JAVA_HOME"] = `C:\java`
	rt.env["PATH"] = `C:\Windows`
	rt.envWrites = nil

	// What the user did with `vfox use` since the last prompt. NODE_OPTIONS
	// contains equals signs, and `vfox env` unsets JAVA_HOME.
	nodePath, nodeOptions := `C:\nodejs;C:\Windows`, "--max-old-space-size=4096 --title=a=b"
	rt.outputs["vfox env -s clink"] = Clink.Export(env.Vars{
		"PATH":         &nodePath,
		"NODE_OPTIONS": &nodeOptions,
		"JAVA_HOME":    nil,
	})
	rt.filterPrompt()

	if got, want := rt.commands()[len(rt.commands())-1], "vfox env -s clink"; got != want {
		t.Errorf("last command = %q, want %q", got, want)
	}
	for name, want := range map[string]string{"PATH": nodePath, "NODE_OPTIONS": nodeOptions} {
		if got := rt.env[name]; got != want {
			t.Errorf("%s = %q, want %q", name, got, want)
		}
	}
	if value, ok := rt.env["JAVA_HOME"]; ok {
		t.Errorf("JAVA_HOME = %q, want it unset", value)
	}

	// The next prompt changes nothing, so the environment is left alone.
	rt.envWrites = nil
	rt.filterPrompt()
	if len(rt.envWrites) != 0 {
		t.Errorf("setenv calls for unchanged variables: %q", rt.envWrites)
	}

	// Only what changed is written.
	nodePath = `C:\python;C:\Windows`
	rt.outputs["vfox env -s clink"] = Clink.Export(env.Vars{
		"PATH":         &nodePath,
		"NODE_OPTIONS": &nodeOptions,
	})
	rt.filterPrompt()
	if want := []string{"PATH=" + nodePath}; !slices.Equal(rt.envWrites, want) {
		t.Errorf("setenv calls = %q, want %q", rt.envWrites, want)
	}
}

func TestClinkPromptEnvironmentLines(t *testing.T) {
	// FOO is "old" before each prompt.
	tests := []struct {
		name      string
		output    string
		want      string
		wantUnset bool
	}{
		{name: "value", output: `set "FOO=bar"`, want: "bar"},
		{name: "value with spaces", output: `set "FOO=a b"`, want: "a b"},
		{name: "equals signs in the value", output: `set "FOO=a=b=c"`, want: "a=b=c"},
		{name: "path list", output: `set "FOO=C:\a;C:\Program Files (x86)\b"`, want: `C:\a;C:\Program Files (x86)\b`},
		{name: "empty value unsets the variable", output: `set "FOO="`, wantUnset: true},
		{name: "assignment without quotes", output: `set FOO=bar`, want: "old"},
		{name: "other output", output: "echo FOO=bar\nREM set \"FOO=bar\"\n\n", want: "old"},
		{name: "cleanup command", output: `"C:/vfox.exe" env --cleanup > nul 2> nul`, want: "old"},
		{name: "no variable name", output: `set "=bar"`, want: "old"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rt := loadedClink(t)
			rt.env["FOO"] = "old"
			rt.outputs["vfox env -s clink"] = tt.output

			rt.filterPrompt()

			got, set := rt.env["FOO"]
			if tt.wantUnset {
				if set {
					t.Errorf("FOO = %q, want it unset", got)
				}
			} else if !set || got != tt.want {
				t.Errorf("FOO = %q (set: %v), want %q", got, set, tt.want)
			}
		})
	}
}

func TestClinkAvailableCache(t *testing.T) {
	// cache is the file as written age seconds ago: a timestamp, then one plugin per line.
	cache := func(age int64, names ...string) string {
		content := fmt.Sprintf("%d\n", clinkNow-age)
		for _, name := range names {
			content += name + "\n"
		}
		return content
	}
	fetched := cache(0, clinkPlugins...) // what is written after running vfox available

	tests := []struct {
		name      string
		cache     string // the file before the first completion; empty means there is none
		wantFetch bool
		wantAdd   []string // the plugins `vfox add` offers
		wantCache string   // the file afterwards
	}{
		{name: "no cache", wantFetch: true, wantAdd: clinkPlugins, wantCache: fetched},
		{
			name:      "recent cache",
			cache:     cache(60*60, "alpha", "beta"),
			wantAdd:   []string{"alpha", "beta"},
			wantCache: cache(60*60, "alpha", "beta"),
		},
		{
			name:      "cache that is a day old",
			cache:     cache(secondsPerDay, "alpha", "beta"),
			wantAdd:   []string{"alpha", "beta"},
			wantCache: cache(secondsPerDay, "alpha", "beta"),
		},
		{
			name:      "expired cache",
			cache:     cache(secondsPerDay+1, "alpha", "beta"),
			wantFetch: true,
			wantAdd:   clinkPlugins,
			wantCache: fetched,
		},
		{
			name:      "cache without plugins",
			cache:     cache(60 * 60),
			wantFetch: true,
			wantAdd:   clinkPlugins,
			wantCache: fetched,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rt := newClinkRuntime(t)
			if tt.cache != "" {
				rt.files[clinkAvailableCache] = tt.cache
			}
			rt.load()
			if rt.ran("vfox available") {
				t.Fatal("vfox available ran before the first completion")
			}

			assertMatches(t, rt.complete("vfox add "), tt.wantAdd)

			if got := rt.ran("vfox available"); got != tt.wantFetch {
				t.Errorf("vfox available ran: %v, want %v", got, tt.wantFetch)
			}
			if got := rt.files[clinkAvailableCache]; got != tt.wantCache {
				t.Errorf("cache file = %q, want %q", got, tt.wantCache)
			}
		})
	}
}

func TestClinkAvailableOutput(t *testing.T) {
	plain := readClinkFixture(t, "vfox_available.txt")
	// The colored output is derived from the fixture. Make sure it is colored, or
	// the "colored" case below would repeat the plain one.
	if colored := colorAvailable(plain); !strings.Contains(colored, "\x1b[36mnodejs") || !strings.Contains(colored, "\x1b[91m✗") {
		t.Fatalf("colorAvailable did not color the output: %q", colored)
	}

	tests := []struct {
		name   string
		output string
		want   []string
	}{
		{name: "plain", output: plain, want: clinkPlugins},
		{name: "colored", output: colorAvailable(plain), want: clinkPlugins},
		{name: "only the header and the footer", output: "AVAILABLE PLUGINS\n\n  Use 'vfox add <plugin>' to install\n"},
		{name: "error", output: "Error: fetch registry: timeout\n"},
		{name: "nothing", output: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rt := newClinkRuntime(t)
			rt.outputs["vfox available"] = tt.output
			rt.load()

			assertMatches(t, rt.complete("vfox add "), tt.want)
		})
	}
}

func TestClinkInstalledOutput(t *testing.T) {
	plain := readClinkFixture(t, "vfox_ls.txt")
	// The fixture has to exercise what the script copes with: every glyph of the
	// tree, and escape sequences once colored.
	for _, glyph := range []string{"├", "─", "┬", "│", "└"} {
		if !strings.Contains(plain, glyph) {
			t.Fatalf("testdata/clink/vfox_ls.txt has no %q", glyph)
		}
	}
	if colored := colorTree(plain); !strings.Contains(colored, "\x1b[90m\x1b[90m├\x1b[0m\x1b[0m\x1b[90m\x1b[90m─\x1b[0m\x1b[0m\x1b[90m\x1b[90m┬\x1b[0m\x1b[0m\x1b[39m\x1b[39mnodejs") {
		t.Fatalf("colorTree did not color the output: %q", colored)
	}

	tests := []struct {
		name     string
		output   string
		sdks     []string // offered by `vfox unuse`, `vfox info`, ...
		versions []string // offered by `vfox uninstall`
		usable   []string // offered by `vfox use`
	}{
		{name: "plain", output: plain, sdks: clinkSdks, versions: clinkSdkVersions, usable: clinkUsable},
		{name: "colored", output: colorTree(plain), sdks: clinkSdks, versions: clinkSdkVersions, usable: clinkUsable},
		{name: "nothing installed", output: "you don't have any sdk installed yet\n"},
		{name: "no output", output: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rt := newClinkRuntime(t)
			rt.outputs["vfox ls"] = tt.output
			rt.load()

			assertMatches(t, rt.complete("vfox unuse "), tt.sdks)
			assertMatches(t, rt.complete("vfox uninstall "), tt.versions)
			assertMatches(t, rt.complete("vfox use "), tt.usable)
		})
	}
}

func TestClinkCompletions(t *testing.T) {
	tests := []struct {
		line string
		want []string
	}{
		{"vfox ", []string{
			"add", "use", "u", "unuse", "info", "remove", "search", "update", "available", "upgrade",
			"current", "c", "list", "ls", "uninstall", "un", "install", "i", "env", "activate", "config",
			"cd", "exec", "x", "help", "h",
		}},
		{"vfox -", []string{"--debug", "--help", "-h", "--version", "-v", "-V"}},
		{"vfox --d", []string{"--debug"}},
		{"vfox help ", nil},

		// use completes the SDKs that have a version installed, and their versions.
		{"vfox use ", clinkUsable},
		{"vfox use node", []string{"nodejs", "nodejs@20.10.0", "nodejs@21.5.1"}},
		{"vfox use nodejs@2", []string{"nodejs@20.10.0", "nodejs@21.5.1"}},
		{"vfox use -", []string{"--global", "-g", "--session", "-s", "--project", "-p", "--unlink", "--help", "-h"}},
		{"vfox use --global ", clinkUsable},
		{"vfox use nodejs ", nil},
		{"vfox use nodejs -", []string{"--global", "-g", "--session", "-s", "--project", "-p", "--unlink", "--help", "-h"}},

		// unuse completes every SDK.
		{"vfox unuse ", clinkSdks},
		{"vfox unuse p", []string{"python"}},
		{"vfox unuse -", []string{"--global", "-g", "--session", "-s", "--project", "-p", "--help", "-h"}},
		{"vfox unuse --project ", clinkSdks},

		// exec completes the SDKs and versions that can be used, but not the command that follows.
		{"vfox exec ", clinkUsable},
		{"vfox exec java@", []string{"java@21.0.1+12"}},
		{"vfox exec -", []string{"--help", "-h"}},
		{"vfox exec nodejs@21.5.1 ", nil},

		{"vfox info ", clinkSdks},
		{"vfox info -", []string{"--format", "-f", "--help", "-h"}},
		{"vfox info nodejs -", []string{"--format", "-f", "--help", "-h"}},

		{"vfox remove ", clinkSdks},
		{"vfox remove -", []string{"--yes", "-y", "--help", "-h"}},

		{"vfox env -", []string{"--shell", "-s", "--cleanup", "-c", "--json", "-j", "--full", "--help", "-h"}},
		{"vfox env --s", []string{"--shell"}},
		{"vfox env --shell ", clinkShells},
		{"vfox env -s ", clinkShells},
		{"vfox env -s p", []string{"pwsh"}},
		{"vfox env ", nil},
		{"vfox activate ", clinkShells},
		{"vfox activate -", []string{"--help", "-h"}},

		{"vfox install ", clinkSdks},
		{"vfox install -", []string{"--all", "-a", "--yes", "-y", "--help", "-h"}},
		{"vfox uninstall ", clinkSdkVersions},
		{"vfox uninstall nodejs@2", []string{"nodejs@20.10.0", "nodejs@21.5.1"}},
		{"vfox uninstall -", []string{"--help", "-h"}},
		{"vfox update ", clinkSdks},
		{"vfox update -", []string{"--all", "-a", "--help", "-h"}},
		{"vfox search ", clinkSdks},
		{"vfox search -", []string{"--help", "-h"}},
		{"vfox current ", clinkSdks},
		{"vfox list ", clinkSdks},
		{"vfox cd ", clinkSdks},
		{"vfox cd -", []string{"--plugin", "-p", "--help", "-h"}},
		{"vfox available -", []string{"--help", "-h"}},
		{"vfox upgrade -", []string{"--help", "-h"}},
		{"vfox available ", nil},

		{"vfox add ", clinkPlugins},
		{"vfox add d", []string{"dotnet-core"}},
		{"vfox add -", []string{"--source", "-s", "--alias", "--help", "-h"}},

		{"vfox config ", []string{"proxy.url", "proxy.enable", "storage.sdkPath", "registry.address"}},
		{"vfox config proxy", []string{"proxy.url", "proxy.enable"}},
		{"vfox config -", []string{"--list", "-l", "--unset", "-un", "--help", "-h"}},
	}
	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			assertMatches(t, loadedClink(t).complete(tt.line), tt.want)
		})
	}
}

func TestClinkCommandAliases(t *testing.T) {
	for alias, command := range map[string]string{
		"u": "use", "x": "exec", "c": "current", "ls": "list", "un": "uninstall", "i": "install",
	} {
		t.Run(alias, func(t *testing.T) {
			rt := loadedClink(t)
			for _, typed := range []string{" ", " -"} {
				want := rt.complete("vfox " + command + typed)
				if len(want) == 0 {
					t.Fatalf("vfox %s%s completes nothing", command, typed)
				}
				assertMatches(t, rt.complete("vfox "+alias+typed), want)
			}
		})
	}
}

func TestClinkRepeatedArguments(t *testing.T) {
	tests := []struct {
		line string
		want []string
	}{
		// install takes any number of SDKs.
		{"vfox install nodejs ", clinkSdks},
		{"vfox install nodejs java ", clinkSdks},
		{"vfox install --yes nodejs ", clinkSdks},
		// add takes any number of plugins, and does not offer one that is already on the line.
		{"vfox add nodejs ", []string{"java", "golang", "python", "dotnet-core", "my-plugin"}},
		{"vfox add nodejs java ", []string{"golang", "python", "dotnet-core", "my-plugin"}},
		{"vfox add nodejs g", []string{"golang"}},
		// A hyphen in the name of a plugin is just a character.
		{"vfox add dotnet-core ", []string{"nodejs", "java", "golang", "python", "my-plugin"}},
		{"vfox add dotnet-core my-plugin ", []string{"nodejs", "java", "golang", "python"}},
		// The others take one.
		{"vfox unuse nodejs ", nil},
		{"vfox info nodejs ", nil},
		{"vfox cd nodejs ", nil},
		{"vfox uninstall nodejs@21.5.1 ", nil},
	}
	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			assertMatches(t, loadedClink(t).complete(tt.line), tt.want)
		})
	}
}

// The names of the plugins are matched as text, not as Lua patterns, to leave
// out the ones that are already on the line. In a pattern a hyphen is a
// quantifier and a dot is a wildcard, so a plugin like gcc-arm-none-eabi was
// offered again after it had been entered.
func TestClinkAddMatchesPluginNamesLiterally(t *testing.T) {
	var available strings.Builder
	available.WriteString("AVAILABLE PLUGINS\n\n")
	for _, name := range []string{"gcc-arm-none-eabi", "a.b", "axb", "nodejs"} {
		fmt.Fprintf(&available, "  %-19s ✓  https://github.com/version-fox/vfox-%s\n", name, name)
	}
	available.WriteString("\n  Use 'vfox add <plugin>' to install\n")

	tests := []struct {
		line string
		want []string
	}{
		{"vfox add ", []string{"gcc-arm-none-eabi", "a.b", "axb", "nodejs"}},
		{"vfox add gcc-arm-none-eabi ", []string{"a.b", "axb", "nodejs"}},
		// The pattern a.b matches axb as well.
		{"vfox add axb ", []string{"gcc-arm-none-eabi", "a.b", "nodejs"}},
		{"vfox add a.b ", []string{"gcc-arm-none-eabi", "axb", "nodejs"}},
	}
	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			rt := newClinkRuntime(t)
			rt.outputs["vfox available"] = available.String()
			rt.load()

			assertMatches(t, rt.complete(tt.line), tt.want)
		})
	}
}

// Clink runs the delayed initialization of an argmatcher every time the command
// is used on a new input line, not just the first time, so running it again must
// neither register the commands twice nor list the plugins again.
func TestClinkDelayedInitializationIsIdempotent(t *testing.T) {
	rt := loadedClink(t)

	for range 3 {
		assertMatches(t, rt.complete("vfox use "), clinkUsable)
	}
	// help takes no arguments. A second registration of the commands would
	// offer them as its argument.
	assertMatches(t, rt.complete("vfox help "), nil)
	assertMatches(t, rt.complete("vfox -"), []string{"--debug", "--help", "-h", "--version", "-v", "-V"})

	var available int
	for _, command := range rt.commands() {
		if command == "vfox available" {
			available++
		}
	}
	if available != 1 {
		t.Errorf("vfox available ran %d times, want 1", available)
	}
}

// Unlike the plugins, which are cached for a day, the installed SDKs change as
// soon as the user installs one, so they are listed again for every completion.
func TestClinkListsInstalledSdksForEveryCompletion(t *testing.T) {
	rt := loadedClink(t)
	assertMatches(t, rt.complete("vfox unuse "), clinkSdks)

	// ruby is installed while the shell is running.
	rt.outputs["vfox ls"] = "All installed sdk versions\n└─┬ruby\n  └──3.3.0\n"

	assertMatches(t, rt.complete("vfox unuse "), []string{"ruby"})
	assertMatches(t, rt.complete("vfox uninstall "), []string{"ruby@3.3.0"})
}

func TestClinkWindowsLineEndings(t *testing.T) {
	script, err := os.ReadFile(clinkScript)
	if err != nil {
		t.Fatal(err)
	}
	// Git checks the script out with CRLF on Windows, and users save it that way.
	crlf := strings.ReplaceAll(strings.ReplaceAll(string(script), "\r\n", "\n"), "\n", "\r\n")

	rt := newClinkRuntime(t)
	rt.run(crlf, clinkScript)

	assertMatches(t, rt.complete("vfox use "), clinkUsable)
	assertMatches(t, rt.complete("vfox env -s "), clinkShells)
}
