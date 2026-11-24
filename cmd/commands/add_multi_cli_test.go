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
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newMockAddRegistry(t *testing.T) *httptest.Server {
	t.Helper()
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
	t.Cleanup(srv.Close)
	return srv
}

func TestAddMultiplePluginsExitStatus(t *testing.T) {
	srv := newMockAddRegistry(t)

	newRegistryFixture := func(t *testing.T) *cliFixture {
		t.Helper()
		f := newCLIFixture(t, false)
		f.writeUserConfig(t, "registry:\n  address: "+srv.URL+"\n")
		return f
	}

	t.Run("all fail exits non-zero", func(t *testing.T) {
		f := newRegistryFixture(t)
		res := f.runProcess(t, "add", "missing-a", "missing-b").mustFail(t, "add missing-a missing-b")
		mustContain(t, "add missing-a stderr failure", res.stderr, "Add plugin(missing-a) failed:", "manifest not found")
		mustContain(t, "add missing-b stderr failure", res.stderr, "Add plugin(missing-b) failed:", "manifest not found")
		if strings.Contains(res.stderr, "\x1b[") {
			t.Fatalf("expected plain text on stderr without ANSI escapes, got %q", res.stderr)
		}
		if strings.Contains(res.stdout, "Add plugin(") {
			t.Fatalf("expected failure details on stderr, not stdout: %q", res.stdout)
		}
		mustContain(t, "stderr summary", res.stderr, "failed to add plugins: missing-a, missing-b")
	})

	t.Run("partial failure installs valid plugin and exits non-zero", func(t *testing.T) {
		f := newRegistryFixture(t)
		res := f.runHere(t, "add", "one", "missing").mustFail(t, "add one missing")
		mustContain(t, "add missing stderr failure", res.stderr, "Add plugin(missing) failed:", "manifest not found")
		if strings.Contains(res.stderr, "\x1b[") {
			t.Fatalf("expected plain text on stderr without ANSI escapes, got %q", res.stderr)
		}
		if strings.Contains(res.stdout, "Add plugin(") {
			t.Fatalf("expected failure details on stderr, not stdout: %q", res.stdout)
		}
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

	t.Run("duplicate plugin names succeed once", func(t *testing.T) {
		f := newRegistryFixture(t)
		res := f.runProcess(t, "add", "two", "two").mustSucceed(t, "add two two")
		if strings.Contains(res.output(), "failed:") {
			t.Fatalf("expected deduplicated add to succeed without failure output: %s", res.output())
		}
		if _, err := os.Stat(filepath.Join(f.shared, "plugin", "two", "main.lua")); err != nil {
			t.Fatalf("expected plugin two to be installed: %v", err)
		}
	})

	t.Run("category-qualified duplicate plugin names succeed once", func(t *testing.T) {
		for _, tc := range []struct {
			name string
			args []string
		}{
			{name: "category_first", args: []string{"add", "cat/two", "two"}},
			{name: "bare_first", args: []string{"add", "two", "cat/two"}},
		} {
			t.Run(tc.name, func(t *testing.T) {
				f := newRegistryFixture(t)
				res := f.runProcess(t, tc.args...).mustSucceed(t, strings.Join(tc.args, " "))
				if strings.Contains(res.output(), "failed:") {
					t.Fatalf("expected deduplicated add to succeed without failure output: %s", res.output())
				}
				if _, err := os.Stat(filepath.Join(f.shared, "plugin", "two", "main.lua")); err != nil {
					t.Fatalf("expected plugin two to be installed: %v", err)
				}
			})
		}
	})

	t.Run("rejects --source and --alias with multiple plugins", func(t *testing.T) {
		for _, tc := range []struct {
			name string
			args []string
		}{
			{name: "source", args: []string{"add", "--source", "./no-such.lua", "one", "two"}},
			{name: "short_source", args: []string{"add", "-s", "./no-such.lua", "one", "two"}},
			{name: "alias", args: []string{"add", "--alias", "uno", "one", "two"}},
		} {
			t.Run(tc.name, func(t *testing.T) {
				f := newRegistryFixture(t)
				res := f.runHere(t, tc.args...).mustFail(t, tc.name)
				mustContain(t, tc.name+" stderr", res.stderr, "apply to one plugin")
				for _, name := range []string{"one", "two", "uno"} {
					if _, err := os.Stat(filepath.Join(f.shared, "plugin", name)); !os.IsNotExist(err) {
						t.Fatalf("expected plugin %s not to be installed, stat err = %v", name, err)
					}
				}
			})
		}
	})
}

func TestAddMultiplePluginsStderrAndDedup(t *testing.T) {
	srv := newMockAddRegistry(t)

	setupHome := func(t *testing.T) string {
		t.Helper()
		home := t.TempDir()
		vfoxHome := filepath.Join(home, "vfox")
		if err := os.MkdirAll(vfoxHome, 0o755); err != nil {
			t.Fatalf("mkdir vfox home: %v", err)
		}
		if err := os.WriteFile(filepath.Join(vfoxHome, "config.yaml"), []byte("registry:\n  address: "+srv.URL+"\n"), 0o644); err != nil {
			t.Fatalf("write config.yaml: %v", err)
		}
		t.Setenv("HOME", home)
		t.Setenv("USERPROFILE", home)
		t.Setenv("VFOX_HOME", vfoxHome)
		return vfoxHome
	}

	t.Run("failure reason written to stderr", func(t *testing.T) {
		setupHome(t)
		app := newTestApp()
		var outBuf, errBuf bytes.Buffer
		useStreams(app, &outBuf, &errBuf)

		err := app.Run(t.Context(), []string{"vfox", "add", "missing-a", "missing-b"})
		if err == nil {
			t.Fatal("expected error for missing plugins, got nil")
		}
		stderr := errBuf.String()
		if !strings.Contains(stderr, "missing-a") || !strings.Contains(stderr, "missing-b") || !strings.Contains(stderr, "manifest not found") {
			t.Fatalf("stderr missing plugin failure details: %q", stderr)
		}
		if strings.Contains(outBuf.String(), "failed:") {
			t.Fatalf("unexpected failure details on stdout: %q", outBuf.String())
		}
	})

	t.Run("duplicate args deduplicated", func(t *testing.T) {
		vfoxHome := setupHome(t)
		app := newTestApp()
		var outBuf, errBuf bytes.Buffer
		useStreams(app, &outBuf, &errBuf)

		if err := app.Run(t.Context(), []string{"vfox", "add", "two", "two"}); err != nil {
			t.Fatalf("expected deduplicated add to succeed, got error: %v (stderr=%q)", err, errBuf.String())
		}
		if errBuf.Len() != 0 {
			t.Fatalf("unexpected stderr on deduplicated add: %q", errBuf.String())
		}
		if _, err := os.Stat(filepath.Join(vfoxHome, "plugin", "two", "main.lua")); err != nil {
			t.Fatalf("expected plugin two to be installed: %v", err)
		}
	})
}

func TestAddRejectsSourceAndAliasWithMultiplePlugins(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
	}{
		{name: "source", args: []string{"vfox", "add", "--source", "./no-such.lua", "one", "two"}},
		{name: "short_source", args: []string{"vfox", "add", "-s", "./no-such.lua", "one", "two"}},
		{name: "alias", args: []string{"vfox", "add", "--alias", "uno", "one", "two"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			vfoxHome := filepath.Join(home, "vfox")
			t.Setenv("HOME", home)
			t.Setenv("VFOX_HOME", vfoxHome)

			app := newTestApp()
			var outBuf, errBuf bytes.Buffer
			useStreams(app, &outBuf, &errBuf)

			err := app.Run(t.Context(), tc.args)
			if err == nil {
				t.Fatalf("expected error for %v, got nil", tc.args)
			}
			if !strings.Contains(err.Error(), "apply to one plugin") {
				t.Fatalf("error = %q, want substring %q", err.Error(), "apply to one plugin")
			}
			if _, statErr := os.Stat(filepath.Join(vfoxHome, "plugin")); !os.IsNotExist(statErr) {
				t.Fatalf("expected plugin dir not to be created, stat err = %v", statErr)
			}
		})
	}
}
