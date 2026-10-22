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

// failingDetectPlugin has a Detect hook that cannot do its job.
const failingDetectPlugin = `PLUGIN = {name="broken", version="0.0.1", description="test", author="test", minRuntimeVersion="0.0.0"}
function PLUGIN:Available(ctx) return {} end
function PLUGIN:PreInstall(ctx) return {version=ctx.version} end
function PLUGIN:EnvKeys(ctx) return {{key="PATH", value=ctx.path .. "/bin"}} end
function PLUGIN:Detect(ctx) error("cannot look") end
`

func TestLinkDetectReportsWhatItCannotDo(t *testing.T) {
	f := newCLIFixture(t, false)
	f.writePlugin(t, "broken", failingDetectPlugin)

	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{"an SDK without a plugin", []string{"link", "--detect", "-y", "nosuch"}, "vfox add nosuch"},
		{"a hook that fails", []string{"link", "--detect", "-y", "broken"}, "cannot look"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := f.run(t, tc.args...).mustFail(t, strings.Join(tc.args, " "))
			mustContain(t, "error", r.output(), tc.want)
		})
	}
}
