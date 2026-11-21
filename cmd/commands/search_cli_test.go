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

import "testing"

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

func TestSearchStillShowsItsHelp(t *testing.T) {
	f := newCLIFixture(t, false)

	for _, flag := range []string{"--help", "-h"} {
		out := f.runHere(t, "search", flag).mustSucceed(t, "search "+flag).output()
		mustContain(t, "search "+flag, out, "Search a version of the target SDK")
	}
}
