/*
 *
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
 *
 */

package commands

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestAddMultiplePluginsExitStatus(t *testing.T) {
	const pluginTpl = `PLUGIN = {name=%q, version="0.0.1", description="test", author="test", minRuntimeVersion="0.0.0"}
function PLUGIN:Available(ctx) return {} end
function PLUGIN:PreInstall(ctx) return {version=ctx.version} end
function PLUGIN:EnvKeys(ctx) return {} end
`
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/one.json":
			fmt.Fprintf(w, `{"name":"one","downloadUrl":%q}`, srv.URL+"/plugins/one.lua")
		case "/two.json":
			fmt.Fprintf(w, `{"name":"two","downloadUrl":%q}`, srv.URL+"/plugins/two.lua")
		case "/plugins/one.lua":
			fmt.Fprintf(w, pluginTpl, "one")
		case "/plugins/two.lua":
			fmt.Fprintf(w, pluginTpl, "two")
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	newRegistryFixture := func(t *testing.T) *cliFixture {
		t.Helper()
		f := newCLIFixture(t, false)
		f.writeUserConfig(t, "registry:\n  address: "+srv.URL+"\n")
		return f
	}

	t.Run("all fail exits non-zero", func(t *testing.T) {
		f := newRegistryFixture(t)
		res := f.runProcess(t, "add", "missing-a", "missing-b").mustFail(t, "add missing-a missing-b")
		mustContain(t, "add missing-a failure", res.output(), "Add plugin(", "missing-a", "failed:")
		mustContain(t, "add missing-b failure", res.output(), "missing-b", "failed:")
		mustContain(t, "stderr summary", res.stderr, "failed to add plugins: missing-a, missing-b")
	})

	t.Run("partial failure installs valid plugin and exits non-zero", func(t *testing.T) {
		f := newRegistryFixture(t)
		res := f.runHere(t, "add", "one", "missing").mustFail(t, "add one missing")
		mustContain(t, "add missing failure", res.output(), "Add plugin(", "missing", "failed:")
		mustContain(t, "stderr summary", res.stderr, "failed to add plugin: missing")
		if _, err := os.Stat(filepath.Join(f.shared, "plugin", "one", "main.lua")); err != nil {
			t.Fatalf("expected valid plugin one to be installed despite missing sibling: %v", err)
		}
	})

	t.Run("all succeed exits zero", func(t *testing.T) {
		f := newRegistryFixture(t)
		f.runProcess(t, "add", "one", "two").mustSucceed(t, "add one two")
		for _, name := range []string{"one", "two"} {
			if _, err := os.Stat(filepath.Join(f.shared, "plugin", name, "main.lua")); err != nil {
				t.Fatalf("expected plugin %s to be installed: %v", name, err)
			}
		}
	})
}
