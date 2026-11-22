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
	"slices"
	"strings"
	"testing"
)

// echoPlugin lists the arguments its Available hook receives as versions.
const echoPlugin = `PLUGIN = {name="echo", version="0.0.1", description="test", author="test", minRuntimeVersion="0.0.0"}
function PLUGIN:Available(ctx)
  local list = {}
  for _, arg in ipairs(ctx.args) do table.insert(list, {version=arg}) end
  return list
end
function PLUGIN:PreInstall(ctx) return {version=ctx.version} end
function PLUGIN:EnvKeys(ctx) return {} end
`

func TestSearchPassesArgumentsThatStartWithADashToThePlugin(t *testing.T) {
	f := newCLIFixture(t, false)
	f.writePlugin(t, "echo", echoPlugin)

	out := f.runHere(t, "search", "echo", "--no-cache", "-x", "plain").mustSucceed(t, "search with dash arguments").output()

	mustContain(t, "search output", out, " - --no-cache", " - -x", " - plain")
}

func TestSearchStillAcceptsADoubleDashBeforeThePluginArguments(t *testing.T) {
	f := newCLIFixture(t, false)
	f.writePlugin(t, "echo", echoPlugin)

	out := f.runHere(t, "search", "echo", "--", "--no-cache").mustSucceed(t, "search with --").output()

	mustContain(t, "search output", out, " - --no-cache")
	mustNotContain(t, "search output", out, " - --\n")
}

func TestSearchStillAcceptsADoubleDashBeforeTheSDKName(t *testing.T) {
	f := newCLIFixture(t, false)
	f.writePlugin(t, "echo", echoPlugin)

	out := f.runHere(t, "search", "--", "echo", "--no-cache").mustSucceed(t, "search -- echo").output()

	mustContain(t, "search output", out, " - --no-cache")
}

func TestSearchStillShowsItsHelp(t *testing.T) {
	f := newCLIFixture(t, false)

	for _, args := range [][]string{{"--help"}, {"-h"}, {"echo", "--help"}, {"echo", "-x", "-h"}} {
		out := f.runHere(t, append([]string{"search"}, args...)...).mustSucceed(t, "search help").output()
		mustContain(t, "search help", out, "Search a version of the target SDK")
	}
}

func TestSearchPassesHelpAndDebugAfterADoubleDashToThePlugin(t *testing.T) {
	f := newCLIFixture(t, false)
	f.writePlugin(t, "echo", echoPlugin)

	out := f.runHere(t, "search", "echo", "--", "--help", "--debug").mustSucceed(t, "search with -- --help").output()

	mustContain(t, "search output", out, " - --help", " - --debug")
}

func TestSearchArgs(t *testing.T) {
	for _, tc := range []struct {
		args  []string
		want  searchCall
		error bool
	}{
		{args: []string{"clang", "--no-cache"}, want: searchCall{sdkName: "clang", pluginArgs: []string{"--no-cache"}}},
		{args: []string{"--", "clang", "--no-cache"}, want: searchCall{sdkName: "clang", pluginArgs: []string{"--no-cache"}}},
		{args: []string{"clang", "--debug"}, want: searchCall{sdkName: "clang", debug: true}},
		{args: []string{"--debug", "clang"}, want: searchCall{sdkName: "clang", debug: true}},
		{args: []string{"clang", "-debug=true"}, want: searchCall{sdkName: "clang", debug: true}},
		{args: []string{"clang", "--debug=false"}, want: searchCall{sdkName: "clang"}},
		{args: []string{"clang", "--debug=maybe"}, error: true},
		{args: []string{"clang", "--", "--debug", "--help"}, want: searchCall{sdkName: "clang", pluginArgs: []string{"--debug", "--help"}}},
		{args: []string{"clang", "--help"}, want: searchCall{sdkName: "clang", help: true}},
		{args: []string{"clang", "-v", "--version"}, want: searchCall{sdkName: "clang", pluginArgs: []string{"-v", "--version"}}},
		{args: []string{"clang", "debug"}, want: searchCall{sdkName: "clang", pluginArgs: []string{"debug"}}},
		{args: []string{"debug"}, want: searchCall{sdkName: "debug"}},
		{args: []string{"clang", "a b"}, want: searchCall{sdkName: "clang", pluginArgs: []string{"a b"}}},
		{args: []string{"--version"}, error: true},
		{args: []string{"--no-cache", "clang"}, error: true},
		{args: []string{"--", "--version"}, error: true},
	} {
		got, err := searchArgs(tc.args)
		if tc.error {
			if err == nil {
				t.Errorf("searchArgs(%q): want an error, got %+v", tc.args, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("searchArgs(%q): %v", tc.args, err)
			continue
		}
		if got.sdkName != tc.want.sdkName || got.help != tc.want.help || got.debug != tc.want.debug || !slices.Equal(got.pluginArgs, tc.want.pluginArgs) {
			t.Errorf("searchArgs(%q) = %+v, want %+v", tc.args, got, tc.want)
		}
	}
}

func TestSearchPassesAPlainDebugArgumentToThePlugin(t *testing.T) {
	f := newCLIFixture(t, false)
	f.writePlugin(t, "echo", echoPlugin)

	out := f.runHere(t, "search", "echo", "x", "debug", "y").mustSucceed(t, "search with debug").output()

	mustContain(t, "search output", out, " - x", " - debug", " - y")
}

func TestSearchRejectsAnSDKNameThatStartsWithADash(t *testing.T) {
	f := newCLIFixture(t, false)

	for _, args := range [][]string{{"--version"}, {"--no-cache", "echo"}, {"--", "--version"}} {
		out := f.runHere(t, append([]string{"search"}, args...)...).mustFail(t, "search with a dash sdk name").output()
		mustContain(t, "search error", out, "cannot start with a dash", "vfox --version")
	}
}

// counterPlugin returns how many times its Available hook ran, and with
// --no-cache removes the cache vfox keeps for it, as a refresh flag should.
const counterPlugin = `PLUGIN = {name="counter", version="0.0.1", description="test", author="test", minRuntimeVersion="0.0.0"}
function PLUGIN:Available(ctx)
  local f = io.open("count", "r")
  local n = 0
  if f then n = tonumber(f:read("*a")) f:close() end
  n = n + 1
  f = io.open("count", "w") f:write(tostring(n)) f:close()
  if ctx.args[1] == "--no-cache" then os.remove(RUNTIME.pluginDirPath .. "/.available.cache") end
  return {{version="call-" .. n}}
end
function PLUGIN:PreInstall(ctx) return {version=ctx.version} end
function PLUGIN:EnvKeys(ctx) return {} end
`

func TestSearchWithPluginArgumentsSkipsTheCache(t *testing.T) {
	f := newCLIFixture(t, false)
	f.writePlugin(t, "counter", counterPlugin)

	for i, step := range []struct {
		args []string
		want string
	}{
		{[]string{"counter"}, "call-1"},
		{[]string{"counter"}, "call-1"}, // from the cache
		{[]string{"counter", "--no-cache"}, "call-2"},
		{[]string{"counter", "--no-cache"}, "call-3"}, // arguments are never cached
		{[]string{"counter"}, "call-4"},               // the plugin removed the cache
	} {
		out := f.runHere(t, append([]string{"search"}, step.args...)...).mustSucceed(t, "search").output()
		if !strings.Contains(out, " - "+step.want+"\n") {
			t.Fatalf("step %d, search %v: want %s in %q", i+1, step.args, step.want, out)
		}
	}
}
