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
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/pterm/pterm"
	"github.com/urfave/cli/v3"

	"github.com/version-fox/vfox/internal"
	"github.com/version-fox/vfox/internal/env"
)

// TestCLIProcess is the child of cliFixture.runProcess: it runs one vfox command line
// in a fresh process, as a user's shell would. Started by itself it does nothing.
func TestCLIProcess(t *testing.T) {
	if os.Getenv("VFOX_TEST_CLI") != "1" {
		return
	}
	separator := slices.Index(os.Args, "--")
	if separator < 0 {
		os.Exit(2)
	}
	if err := newTestApp().Run(context.Background(), append([]string{"vfox"}, os.Args[separator+1:]...)); err != nil {
		code := 1
		var exitCoder cli.ExitCoder
		if errors.As(err, &exitCoder) {
			code = exitCoder.ExitCode()
		}
		if err.Error() != "" {
			fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(code)
	}
	os.Exit(0)
}

// pristine are the commands the tests exercise, copied before any run could
// change the state cli keeps in them: which flags were set, and the streams.
var pristine = cloneCommands(
	Activate, Alias, Config, Current, Doctor, Env, Info, Install, Link, List, Uninstall, Use,
)

// newTestApp is vfox with a fresh copy of those commands.
func newTestApp() *cli.Command {
	return &cli.Command{Name: "vfox", Commands: cloneCommands(pristine...)}
}

func cloneCommands(commands ...*cli.Command) []*cli.Command {
	clones := make([]*cli.Command, len(commands))
	for i, c := range commands {
		clone := *c
		clone.Flags = make([]cli.Flag, len(c.Flags))
		for j, f := range c.Flags {
			v := reflect.ValueOf(f).Elem()
			copied := reflect.New(v.Type())
			copied.Elem().Set(v)
			clone.Flags[j] = copied.Interface().(cli.Flag)
		}
		clone.Commands = cloneCommands(c.Commands...)
		clones[i] = &clone
	}
	return clones
}

// alphaPlugin is a minimal SDK plugin. It puts <path>/bin on PATH, finds
// "installations" in the directory ALPHA_DETECT_DIR, and fingerprints an
// installation by the contents of its VERSION file.
const alphaPlugin = `PLUGIN = {name="alpha", version="0.0.1", description="test", author="test", minRuntimeVersion="0.0.0"}
PLUGIN.shadowHints = {{match="pnpm", message="pnpm put its own alphatool first.", url="https://example.com/pnpm"}}
function PLUGIN:Available(ctx) return {{version="1.0.0"}} end
function PLUGIN:PreInstall(ctx) return {version=ctx.version} end
function PLUGIN:EnvKeys(ctx) return {{key="PATH", value=ctx.path .. "/bin"}} end
function PLUGIN:Detect(ctx)
  local dir = os.getenv("ALPHA_DETECT_DIR")
  if dir == nil then return nil end
  return {{version="found-1", path=dir, note="from the test"}}
end
function PLUGIN:Fingerprint(ctx)
  local f = io.open(ctx.path .. "/VERSION", "r")
  if f == nil then return nil end
  local content = f:read("*a")
  f:close()
  return {facts={version=content}}
end
`

// plainPlugin has no optional hooks at all.
const plainPlugin = `PLUGIN = {name="plain", version="0.0.1", description="test", author="test", minRuntimeVersion="0.0.0"}
function PLUGIN:Available(ctx) return {} end
function PLUGIN:PreInstall(ctx) return {version=ctx.version} end
function PLUGIN:EnvKeys(ctx) return {{key="PATH", value=ctx.path .. "/bin"}} end
`

type cliFixture struct {
	root, userHome, shared, work, system string
	env                                  map[string]string
}

// newCLIFixture sets up a vfox home with the alpha and plain plugins. When
// splitShared is true the shared root is separate from the user's directory,
// as with a VFOX_HOME that several users share.
//
// On Windows the tests select SDKs globally, which puts the SDK's directories in
// the PATH that the registry holds for the user who runs them. They only run
// where that does not matter: a CI runner, or a VM that is thrown away.
func newCLIFixture(t *testing.T, splitShared bool) *cliFixture {
	t.Helper()
	if runtime.GOOS == "windows" && os.Getenv("VFOX_TEST_WINDOWS_REGISTRY") != "1" {
		t.Skip("these tests change the user PATH in the Windows registry: set VFOX_TEST_WINDOWS_REGISTRY=1 on a runner or VM that is thrown away (CI does)")
	}
	root := t.TempDir()
	if runtime.GOOS == "windows" {
		// TEMP can name the user's directory by its short name (RUNNER~1) while
		// calls that resolve a path report the long one, and the tests compare
		// paths as text: use the long spelling throughout.
		resolved, err := filepath.EvalSymlinks(root)
		must(t, err)
		root = resolved
	}
	f := &cliFixture{
		root:     root,
		userHome: filepath.Join(root, ".vfox"),
		work:     filepath.Join(root, "project"),
		system:   filepath.Join(root, "system-bin"),
		env:      map[string]string{},
	}
	f.shared = f.userHome
	if splitShared {
		f.shared = filepath.Join(root, "shared")
	}
	for _, dir := range []string{f.userHome, f.shared, f.work, f.system} {
		must(t, os.MkdirAll(dir, 0755))
	}
	f.writePlugin(t, "alpha", alphaPlugin)
	f.writePlugin(t, "plain", plainPlugin)
	return f
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func (f *cliFixture) writePlugin(t *testing.T, name, source string) {
	t.Helper()
	dir := filepath.Join(f.shared, "plugin", name)
	must(t, os.MkdirAll(dir, 0755))
	must(t, os.WriteFile(filepath.Join(dir, "main.lua"), []byte(source), 0644))
}

// installOnDisk lays out an installed version the way vfox did before it kept
// receipts, with the given commands on its bin directory.
func (f *cliFixture) installOnDisk(t *testing.T, sdk, version string, commands ...string) string {
	t.Helper()
	main := filepath.Join(f.shared, "cache", sdk, "v-"+version, sdk+"-"+version)
	must(t, os.MkdirAll(filepath.Join(main, "bin"), 0755))
	for _, name := range commands {
		writeCommand(t, filepath.Join(main, "bin"), name)
	}
	return main
}

// external creates an SDK that belongs to somebody else.
func (f *cliFixture) external(t *testing.T, commands ...string) string {
	t.Helper()
	parent := filepath.Join(f.root, "external")
	must(t, os.MkdirAll(parent, 0755))
	// Two calls within one tick of the clock would get the same name from it, and
	// the clock on Windows ticks slowly: let the file system pick a free one.
	dir, err := os.MkdirTemp(parent, "sdk-")
	must(t, err)
	must(t, os.MkdirAll(filepath.Join(dir, "bin"), 0755))
	for _, name := range commands {
		writeCommand(t, filepath.Join(dir, "bin"), name)
	}
	return dir
}

func (f *cliFixture) writeUserConfig(t *testing.T, content string) {
	t.Helper()
	must(t, os.WriteFile(filepath.Join(f.userHome, "config.yaml"), []byte(content), 0644))
}

type cliResult struct {
	stdout, stderr string
	code           int
}

func (r cliResult) output() string { return r.stdout + r.stderr }

// inProcess lists the commands that run inside the test process, which is where
// the coverage report can see them. The others set variables for a shell
// (activate, env) or may hand over to a shell (use), so each run of them gets a
// process of its own.
var inProcess = map[string]bool{
	"alias": true, "config": true, "doctor": true, "info": true, "install": true, "link": true, "list": true, "uninstall": true,
}

// run executes `vfox <args>` inside the fixture: in this process for the
// commands in inProcess, in a fresh one for the rest.
func (f *cliFixture) run(t *testing.T, args ...string) cliResult {
	t.Helper()
	if len(args) > 0 && inProcess[args[0]] {
		return f.runHere(t, args...)
	}
	return f.runProcess(t, args...)
}

// stripped reports whether the fixture removes a variable from the environment
// it gives to vfox: everything vfox reads from the environment of whoever runs
// the tests.
func stripped(key string) bool {
	if runtime.GOOS == "windows" {
		key = strings.ToUpper(key) // the names are case-insensitive there: Path is PATH
	}
	return strings.HasPrefix(key, "VFOX_") || strings.HasPrefix(key, "__VFOX_") || key == "HOME" || key == "PATH" ||
		key == "VIRTUAL_ENV" || key == "CONDA_PREFIX" || key == "TMUX" ||
		key == "VSCODE_RESOLVING_ENVIRONMENT" || key == "INTELLIJ_ENVIRONMENT_READER"
}

// environ is what the fixture sets for vfox. A value of "-" removes a variable
// the fixture would set.
func (f *cliFixture) environ() map[string]string {
	environ := map[string]string{
		"HOME":              f.root,
		"VFOX_HOME":         f.shared,
		"PATH":              f.system,
		"__VFOX_CURTMPPATH": filepath.Join(f.userHome, "tmp", "test-session"),
		"__VFOX_SHELL":      "bash", // an activated shell, unless a test says otherwise
	}
	if runtime.GOOS == "windows" {
		environ["USERPROFILE"] = f.root // where os.UserHomeDir looks, instead of HOME
	}
	for k, v := range f.env {
		environ[k] = v
	}
	return environ
}

// command is a process that runs in the fixture: in the project directory, with
// the environment of the fixture and nothing of vfox from the environment of
// whoever runs the tests.
func (f *cliFixture) command(ctx context.Context, name string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = f.work
	for _, kv := range os.Environ() {
		if key, _, _ := strings.Cut(kv, "="); !stripped(key) {
			cmd.Env = append(cmd.Env, kv)
		}
	}
	for k, v := range f.environ() {
		if v != "-" {
			cmd.Env = append(cmd.Env, k+"="+v)
		}
	}
	return cmd
}

// runCommand runs cmd to its end. A process that exits with a code is a result;
// one that cannot be started fails the test.
func runCommand(t *testing.T, cmd *exec.Cmd) cliResult {
	t.Helper()
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	result := cliResult{}
	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			t.Fatalf("%v: %v", cmd.Args, err)
		}
		result.code = exitErr.ExitCode()
	}
	result.stdout, result.stderr = stdout.String(), stderr.String()
	return result
}

// runProcess executes `vfox <args>` in a fresh process inside the fixture.
func (f *cliFixture) runProcess(t *testing.T, args ...string) cliResult {
	t.Helper()
	exe, err := os.Executable()
	must(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cmd := f.command(ctx, exe, append([]string{"-test.run=^TestCLIProcess$", "--"}, args...)...)
	cmd.Env = append(cmd.Env, "VFOX_TEST_CLI=1")
	return runCommand(t, cmd)
}

// manager is a vfox manager in the environment of the fixture, for the tests
// that call the code behind a command directly.
func (f *cliFixture) manager(t *testing.T) *internal.Manager {
	t.Helper()
	for _, kv := range os.Environ() {
		if key, _, _ := strings.Cut(kv, "="); stripped(key) {
			t.Setenv(key, "")
		}
	}
	for k, v := range f.environ() {
		if v == "-" {
			v = ""
		}
		t.Setenv(k, v)
	}
	t.Chdir(f.work)
	manager, err := internal.NewSdkManager()
	must(t, err)
	t.Cleanup(manager.Close)
	return manager
}

// useStreams sets the output streams of a command and its subcommands, which
// cli would otherwise take from os.Stdout and os.Stderr when it first needs them.
func useStreams(cmd *cli.Command, out, err io.Writer) {
	cmd.Writer, cmd.ErrWriter = out, err
	for _, sub := range cmd.Commands {
		useStreams(sub, out, err)
	}
}

// runHere executes `vfox <args>` inside the fixture without a process of its
// own, so that the code it runs shows in the coverage report. While it runs the
// environment and the output streams are the fixture's, and they are the test's
// again afterwards.
func (f *cliFixture) runHere(t *testing.T, args ...string) cliResult {
	t.Helper()
	var restore []func()
	defer func() {
		for i := len(restore) - 1; i >= 0; i-- {
			restore[i]()
		}
	}()

	setenv := func(key, value string, set bool) {
		old, had := os.LookupEnv(key)
		restore = append(restore, func() {
			if had {
				_ = os.Setenv(key, old)
			} else {
				_ = os.Unsetenv(key)
			}
		})
		if set {
			_ = os.Setenv(key, value)
		} else {
			_ = os.Unsetenv(key)
		}
	}
	for _, kv := range os.Environ() {
		if key, _, _ := strings.Cut(kv, "="); stripped(key) {
			setenv(key, "", false)
		}
	}
	for k, v := range f.environ() {
		setenv(k, v, v != "-")
	}
	t.Chdir(f.work)

	// Everything vfox prints goes to os.Stdout, to pterm or to cli's error writer.
	capture := func(target **os.File) func() string {
		file, err := os.CreateTemp(t.TempDir(), "stream")
		must(t, err)
		old := *target
		*target = file
		restore = append(restore, func() { *target = old; _ = file.Close() })
		return func() string {
			data, err := os.ReadFile(file.Name())
			must(t, err)
			return string(data)
		}
	}
	previousStdout := os.Stdout
	stdout, stderr := capture(&os.Stdout), capture(&os.Stderr)
	pterm.SetDefaultOutput(os.Stdout)
	restore = append(restore, func() { pterm.SetDefaultOutput(previousStdout) })

	// An error that carries an exit code must not end the test binary.
	exitCode, previousExiter, previousWriter := -1, cli.OsExiter, cli.ErrWriter
	cli.OsExiter, cli.ErrWriter = func(code int) { exitCode = code }, os.Stderr
	restore = append(restore, func() { cli.OsExiter, cli.ErrWriter = previousExiter, previousWriter })

	app := newTestApp()
	app.ExitErrHandler = func(context.Context, *cli.Command, error) {}
	useStreams(app, os.Stdout, os.Stderr)
	err := app.Run(context.Background(), append([]string{"vfox"}, args...))

	result := cliResult{stdout: stdout(), stderr: stderr()}
	switch {
	case err != nil:
		result.code = 1
		var exitCoder cli.ExitCoder
		if errors.As(err, &exitCoder) {
			result.code = exitCoder.ExitCode()
		}
		if message := err.Error(); message != "" {
			result.stderr += message + "\n"
		}
	case exitCode > 0:
		result.code = exitCode
	}
	return result
}

func (r cliResult) mustSucceed(t *testing.T, what string) cliResult {
	t.Helper()
	if r.code != 0 {
		t.Fatalf("%s failed with %d\nstdout: %s\nstderr: %s", what, r.code, r.stdout, r.stderr)
	}
	return r
}

func (r cliResult) mustFail(t *testing.T, what string) cliResult {
	t.Helper()
	if r.code == 0 {
		t.Fatalf("%s must fail\nstdout: %s\nstderr: %s", what, r.stdout, r.stderr)
	}
	return r
}

func mustContain(t *testing.T, what, text string, wants ...string) {
	t.Helper()
	for _, want := range wants {
		if !strings.Contains(text, want) {
			t.Errorf("%s lacks %q:\n%s", what, want, text)
		}
	}
}

func mustNotContain(t *testing.T, what, text string, unwanted ...string) {
	t.Helper()
	for _, s := range unwanted {
		if strings.Contains(text, s) {
			t.Errorf("%s must not contain %q:\n%s", what, s, text)
		}
	}
}

func exists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}

// pathList is the value of a PATH that holds the given directories.
func pathList(dirs ...string) string {
	return strings.Join(dirs, string(os.PathListSeparator))
}

// shellPath is how vfox writes a directory for the bash hook that the fixture
// pretends to run in: Git Bash spells C:\a\b as /c/a/b, which is what the
// PATH a command tells the shell to export holds on Windows.
func shellPath(dir string) string {
	if runtime.GOOS != "windows" {
		return dir
	}
	dir = filepath.ToSlash(dir)
	if len(dir) > 1 && dir[1] == ':' {
		dir = "/" + strings.ToLower(dir[:1]) + dir[2:]
	}
	return dir
}

// shellPaths are the entries of the PATH that bash gets for the directories.
func shellPaths(dirs ...string) []string {
	paths := make([]string, len(dirs))
	for i, dir := range dirs {
		paths[i] = shellPath(dir)
	}
	return paths
}

// commandFile is the file that provides a command in a bin directory. Windows
// runs a file only if its name has an extension from PATHEXT.
func commandFile(name string) string {
	if runtime.GOOS == "windows" {
		return name + ".cmd"
	}
	return name
}

// writeCommand puts a command in dir and returns its file. No test runs it.
func writeCommand(t *testing.T, dir, name string) string {
	t.Helper()
	path := filepath.Join(dir, commandFile(name))
	content := "#!/bin/sh\n"
	if runtime.GOOS == "windows" {
		content = "@echo off\r\n"
	}
	must(t, os.WriteFile(path, []byte(content), 0755))
	return path
}

// linkDirectory makes link lead to the directory target, the way a user does by
// hand: with a symlink, or with a junction on Windows, which needs no privilege.
func linkDirectory(t *testing.T, target, link string) {
	t.Helper()
	must(t, env.CreateDirSymlink(target, link))
}

// linkFile gives the file a second name, the way Homebrew does with a symlink.
// A symlink takes a privilege on Windows, so a hard link stands in there, which
// os.SameFile sees through as well.
func linkFile(t *testing.T, target, link string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		must(t, os.Link(target, link))
		return
	}
	must(t, os.Symlink(target, link))
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(content)
}
