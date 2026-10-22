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
	"bytes"
	_ "embed"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"text/template"

	lua "github.com/yuin/gopher-lua"

	"github.com/version-fox/vfox/internal/env"
)

// The harness for the tests of clink_vfox.lua in clink_test.go. The script is
// the one users copy into Clink's scripts directory; here it runs in a gopher-lua
// VM. Clink itself is replaced by testdata/clink/fake_clink.lua and the io and
// os libraries by in-memory fakes, so nothing is executed, read or written
// outside the VM and the tests run on every platform of the CI matrix without
// Clink or vfox installed.
//
// The commands the script runs (vfox available, vfox ls, ...) return canned
// output. The output of vfox available and vfox ls is kept in testdata/clink
// and checked against the commands themselves by cmd/commands/clink_format_test.go,
// so the script is tested against the format it really parses.

const (
	clinkScript = "clink_vfox.lua"

	clinkUserProfile = `C:\Users\tester`
	// clinkAvailableCache is where the script caches the plugin names printed by
	// `vfox available`.
	clinkAvailableCache = clinkUserProfile + `\vfox_available.txt`
	// curTmpPathFlag is pathmeta.HookCurTmpPath; package shell must not import pathmeta.
	curTmpPathFlag = "__VFOX_CURTMPPATH"

	clinkPid      = 4242
	clinkNow      = 1_700_000_000
	secondsPerDay = 24 * 60 * 60
)

//go:embed testdata/clink/fake_clink.lua
var fakeClinkLua string

// clinkConfig is part of the output of `vfox config -l`: a "key = value" line for
// every setting, with nothing after the equals sign when the value is empty.
const clinkConfig = "proxy.url = \nproxy.enable = false\nstorage.sdkPath = \nregistry.address = \n"

// The plugins in testdata/clink/vfox_available.txt, in the order they are listed.
var clinkPlugins = []string{"nodejs", "java", "golang", "python", "dotnet-core", "my-plugin"}

// The SDKs in testdata/clink/vfox_ls.txt. python has no version installed.
var (
	clinkSdks        = []string{"golang", "java", "nodejs", "python"}
	clinkSdkVersions = []string{"golang@1.22.0", "java@21.0.1+12", "nodejs@20.10.0", "nodejs@21.5.1"}
	// `vfox use` and `vfox exec` complete installed SDKs and their versions.
	clinkUsable = []string{
		"golang", "golang@1.22.0",
		"java", "java@21.0.1+12",
		"nodejs", "nodejs@20.10.0", "nodejs@21.5.1",
	}
	clinkShells = []string{"bash", "zsh", "pwsh", "fish", "clink", "nushell"}
)

// startedCommand is a command the script started with io.popen or os.execute,
// with the environment it was started in.
type startedCommand struct {
	line string
	env  map[string]string
}

// clinkRuntime is the VM clink_vfox.lua runs in, and the world around it.
type clinkRuntime struct {
	t    *testing.T
	L    *lua.LState
	fake *lua.LTable // the fakeclink table of fake_clink.lua

	// outputs maps a command line to the output io.popen reads from it. Starting
	// any other command fails the test.
	outputs map[string]string
	// started lists the commands the script has started, in order.
	started []startedCommand
	// files is the file system behind io.open.
	files map[string]string
	// env is the process environment behind os.getenv and os.setenv.
	env map[string]string
	// envWrites logs every os.setenv call as "NAME=value", or "NAME" to unset.
	envWrites []string
}

// newClinkRuntime returns a VM that has not run the script yet. Its environment
// is that of a fresh Clink session. Every command the script runs prints the
// canned output set up below, which tests change through outputs.
func newClinkRuntime(t *testing.T) *clinkRuntime {
	t.Helper()

	// The real io and os libraries are not opened: they would run the commands
	// the script starts and touch the real file system.
	L := lua.NewState(lua.Options{SkipOpenLibs: true})
	t.Cleanup(L.Close)
	for _, lib := range []struct {
		name string
		open lua.LGFunction
	}{
		{lua.BaseLibName, lua.OpenBase},
		{lua.TabLibName, lua.OpenTable},
		{lua.StringLibName, lua.OpenString},
		{lua.MathLibName, lua.OpenMath},
		{lua.CoroutineLibName, lua.OpenCoroutine},
	} {
		L.Push(L.NewFunction(lib.open))
		L.Push(lua.LString(lib.name))
		L.Call(1, 0)
	}

	rt := &clinkRuntime{
		t: t,
		L: L,
		outputs: map[string]string{
			"vfox activate clink": "",
			"vfox available":      readClinkFixture(t, "vfox_available.txt"),
			"vfox ls":             readClinkFixture(t, "vfox_ls.txt"),
			"vfox config -l":      clinkConfig,
			"vfox env -s clink":   "",
		},
		files: map[string]string{},
		env:   map[string]string{"USERPROFILE": clinkUserProfile},
	}
	rt.installIO()
	rt.installOS()
	rt.run(fakeClinkLua, "fake_clink.lua")
	fake, ok := L.GetGlobal("fakeclink").(*lua.LTable)
	if !ok {
		t.Fatal("fake_clink.lua does not define fakeclink")
	}
	rt.fake = fake
	return rt
}

// loadedClink returns a runtime that has run the script with the default setup.
func loadedClink(t *testing.T) *clinkRuntime {
	t.Helper()
	rt := newClinkRuntime(t)
	rt.load()
	return rt
}

// readClinkFixture reads a file of canned output. Git may check text files out
// with CRLF line endings on Windows; the commands print LF.
func readClinkFixture(t *testing.T, name string) string {
	t.Helper()
	content, err := os.ReadFile(filepath.Join("testdata", "clink", name))
	if err != nil {
		t.Fatal(err)
	}
	return strings.ReplaceAll(string(content), "\r\n", "\n")
}

// load runs the script, as Clink does when it starts.
func (rt *clinkRuntime) load() {
	rt.t.Helper()
	script, err := os.ReadFile(clinkScript)
	if err != nil {
		rt.t.Fatal(err)
	}
	rt.run(string(script), clinkScript)
}

func (rt *clinkRuntime) run(source, name string) {
	rt.t.Helper()
	fn, err := rt.L.Load(strings.NewReader(source), name)
	if err != nil {
		rt.t.Fatalf("load %s: %v", name, err)
	}
	rt.L.Push(fn)
	if err := rt.L.PCall(0, lua.MultRet, nil); err != nil {
		rt.t.Fatalf("run %s: %v", name, err)
	}
}

func (rt *clinkRuntime) callFake(function string, args ...lua.LValue) lua.LValue {
	rt.t.Helper()
	err := rt.L.CallByParam(lua.P{Fn: rt.fake.RawGetString(function), NRet: 1, Protect: true}, args...)
	if err != nil {
		rt.t.Fatalf("fakeclink.%s: %v", function, err)
	}
	result := rt.L.Get(-1)
	rt.L.Pop(1)
	return result
}

// complete returns the sorted matches Clink offers when Tab is pressed after
// line. The last word of line is the one being completed.
func (rt *clinkRuntime) complete(line string) []string {
	rt.t.Helper()
	result, ok := rt.callFake("complete", lua.LString(line)).(*lua.LTable)
	if !ok {
		rt.t.Fatalf("fakeclink.complete(%q) did not return a table", line)
	}
	var matches []string
	result.ForEach(func(_, match lua.LValue) { matches = append(matches, match.String()) })
	slices.Sort(matches)
	return slices.Compact(matches)
}

// filterPrompt draws a prompt, which is when the script updates the environment.
func (rt *clinkRuntime) filterPrompt() {
	rt.t.Helper()
	rt.callFake("filter_prompt")
}

// commands returns the command lines the script started.
func (rt *clinkRuntime) commands() []string {
	lines := make([]string, len(rt.started))
	for i, command := range rt.started {
		lines[i] = command.line
	}
	return lines
}

func (rt *clinkRuntime) ran(line string) bool {
	return slices.Contains(rt.commands(), line)
}

func (rt *clinkRuntime) start(line string) {
	rt.started = append(rt.started, startedCommand{line: line, env: maps.Clone(rt.env)})
}

// installIO replaces the io library. io.popen reads the canned output of a
// command and io.open reads and writes the in-memory files.
func (rt *clinkRuntime) installIO() {
	L := rt.L
	ioLib := L.NewTable()
	L.SetFuncs(ioLib, map[string]lua.LGFunction{
		"popen": func(L *lua.LState) int {
			line := L.CheckString(1)
			rt.start(line)
			output, ok := rt.outputs[line]
			if !ok {
				rt.t.Errorf("unexpected command: io.popen(%q)", line)
			}
			L.Push(rt.readHandle(output))
			return 1
		},
		"open": func(L *lua.LState) int {
			name, mode := L.CheckString(1), L.OptString(2, "r")
			switch mode {
			case "r", "rb":
				content, ok := rt.files[name]
				if !ok {
					L.Push(lua.LNil)
					L.Push(lua.LString(name + ": No such file or directory"))
					L.Push(lua.LNumber(2))
					return 3
				}
				L.Push(rt.readHandle(content))
			case "w", "wb":
				rt.files[name] = "" // opening for writing truncates the file
				L.Push(rt.writeHandle(name))
			default:
				L.ArgError(2, "unsupported mode "+mode)
			}
			return 1
		},
	})
	L.SetGlobal("io", ioLib)
}

// installOS replaces the os library. os.execute only records the command.
func (rt *clinkRuntime) installOS() {
	L := rt.L
	osLib := L.NewTable()
	L.SetFuncs(osLib, map[string]lua.LGFunction{
		"getenv": func(L *lua.LState) int {
			if value, ok := rt.env[L.CheckString(1)]; ok {
				L.Push(lua.LString(value))
			} else {
				L.Push(lua.LNil)
			}
			return 1
		},
		"setenv": func(L *lua.LState) int {
			name := L.CheckString(1)
			if L.Get(2) == lua.LNil {
				delete(rt.env, name)
				rt.envWrites = append(rt.envWrites, name)
			} else {
				value := L.CheckString(2) // numbers are converted, as in Lua
				rt.env[name] = value
				rt.envWrites = append(rt.envWrites, name+"="+value)
			}
			L.Push(lua.LTrue)
			return 1
		},
		"getpid": func(L *lua.LState) int {
			L.Push(lua.LNumber(clinkPid))
			return 1
		},
		"time": func(L *lua.LState) int {
			L.Push(lua.LNumber(clinkNow))
			return 1
		},
		"execute": func(L *lua.LState) int {
			rt.start(L.CheckString(1))
			L.Push(lua.LTrue)
			L.Push(lua.LString("exit"))
			L.Push(lua.LNumber(0))
			return 3
		},
	})
	L.SetGlobal("os", osLib)
}

// readHandle is the file object io.popen and io.open return for reading.
func (rt *clinkRuntime) readHandle(content string) *lua.LTable {
	L := rt.L
	lines := splitLines(content)
	next := func() lua.LValue {
		if len(lines) == 0 {
			return lua.LNil
		}
		line := lines[0]
		lines = lines[1:]
		return lua.LString(line)
	}
	handle := L.NewTable()
	L.SetFuncs(handle, map[string]lua.LGFunction{
		"lines": func(L *lua.LState) int {
			L.Push(L.NewFunction(func(L *lua.LState) int {
				L.Push(next())
				return 1
			}))
			return 1
		},
		"read": func(L *lua.LState) int {
			if format := L.OptString(2, "*l"); format != "*l" && format != "l" {
				L.ArgError(2, "unsupported read format "+format)
			}
			L.Push(next())
			return 1
		},
		"close": func(L *lua.LState) int {
			L.Push(lua.LTrue)
			return 1
		},
	})
	return handle
}

// writeHandle is the file object io.open returns for writing.
func (rt *clinkRuntime) writeHandle(name string) *lua.LTable {
	L := rt.L
	handle := L.NewTable()
	L.SetFuncs(handle, map[string]lua.LGFunction{
		"write": func(L *lua.LState) int {
			for i := 2; i <= L.GetTop(); i++ {
				rt.files[name] += L.CheckString(i)
			}
			L.Push(handle)
			return 1
		},
		"close": func(L *lua.LState) int {
			L.Push(lua.LTrue)
			return 1
		},
	})
	return handle
}

func splitLines(content string) []string {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	if content == "" {
		return nil
	}
	return strings.Split(strings.TrimSuffix(content, "\n"), "\n")
}

func assertMatches(t *testing.T, got, want []string) {
	t.Helper()
	want = slices.Compact(slices.Sorted(slices.Values(want)))
	if !slices.Equal(got, want) {
		t.Errorf("matches = %q, want %q", got, want)
	}
}

// activateOutput is what `vfox activate clink` prints for the given variables:
// the hook template of Clink.Activate, rendered the way the activate command does.
func activateOutput(t *testing.T, vars env.Vars) string {
	t.Helper()
	hook, err := Clink.Activate(ActivateConfig{})
	if err != nil {
		t.Fatal(err)
	}
	tmpl, err := template.New("hook").Parse(hook)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	err = tmpl.Execute(&out, struct{ SelfPath, EnvContent string }{
		SelfPath:   "C:/Program Files/vfox/vfox.exe",
		EnvContent: Clink.Export(vars),
	})
	if err != nil {
		t.Fatal(err)
	}
	return out.String()
}

// ansi wraps text in a color the way pterm does.
func ansi(code, text string) string {
	return "\x1b[" + code + "m" + text + "\x1b[0m"
}

// colorAvailable adds the colors pterm uses when the terminal supports them to
// the plain output of `vfox available`.
func colorAvailable(plain string) string {
	var out strings.Builder
	for _, line := range strings.SplitAfter(plain, "\n") {
		text, newline := strings.CutSuffix(line, "\n")
		switch {
		case text == "AVAILABLE PLUGINS":
			text = ansi("1", text)
		case strings.Contains(text, "✓") || strings.Contains(text, "✗"):
			// "  <name padded to a common width> <✓|✗>  <homepage>"
			mark, markColor := "✓", "92"
			before, homepage, found := strings.Cut(text, mark)
			if !found {
				mark, markColor = "✗", "91"
				before, homepage, _ = strings.Cut(text, mark)
			}
			name := strings.TrimSuffix(strings.TrimPrefix(before, "  "), " ")
			text = "  " + ansi("36", name) + " " + ansi(markColor, mark) + "  " + ansi("97", strings.TrimPrefix(homepage, "  "))
		case strings.HasPrefix(text, "  Use "):
			text = "  " + ansi("90", strings.TrimPrefix(text, "  "))
		}
		out.WriteString(text)
		if newline {
			out.WriteString("\n")
		}
	}
	return out.String()
}

// colorTree adds the colors pterm uses when the terminal supports them to the
// plain output of `vfox ls`: it wraps every tree glyph and every label in
// its style twice.
func colorTree(plain string) string {
	var out strings.Builder
	for _, line := range strings.SplitAfter(plain, "\n") {
		text, newline := strings.CutSuffix(line, "\n")
		if text != "" {
			label := strings.TrimLeft(text, " │├└─┬")
			for _, r := range text[:len(text)-len(label)] {
				if r == ' ' {
					out.WriteRune(r)
				} else {
					out.WriteString(ansi("90", ansi("90", string(r))))
				}
			}
			out.WriteString(ansi("39", ansi("39", label)))
		}
		if newline {
			out.WriteString("\n")
		}
	}
	return out.String()
}
