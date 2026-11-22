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
		if got.sdkName != tc.want.sdkName || got.help != tc.want.help || got.debug != tc.want.debug || strings.Join(got.pluginArgs, " ") != strings.Join(tc.want.pluginArgs, " ") {
			t.Errorf("searchArgs(%q) = %+v, want %+v", tc.args, got, tc.want)
		}
	}
}
