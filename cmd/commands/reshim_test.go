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
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/urfave/cli/v3"
	"github.com/version-fox/vfox/internal/env"
)

type reshimFixture struct {
	home, work, session, system string
}

// newReshimFixture builds an isolated HOME with:
//   - shimtool 1.0.0 installed and selected at project scope with unlink, so
//     reshim downgrades it to session scope (covers the unlink branch);
//   - ghost selected globally but with no plugin (covers the lookup skip);
//   - emptytool with a plugin but nothing installed (covers the resolve skip).
func newReshimFixture(t *testing.T) *reshimFixture {
	t.Helper()
	root := t.TempDir()
	f := &reshimFixture{
		home:    filepath.Join(root, ".vfox"),
		work:    filepath.Join(root, "project"),
		session: filepath.Join(root, ".vfox", "tmp", "test-session"),
		system:  filepath.Join(root, "system-bin"),
	}
	for _, dir := range []string{f.home, f.work, f.session, f.system} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
	}
	writePlugin := func(name, postUse string) {
		pluginDir := filepath.Join(f.home, "plugin", name)
		if err := os.MkdirAll(pluginDir, 0755); err != nil {
			t.Fatal(err)
		}
		script := fmt.Sprintf(`PLUGIN = {name=%q, version="0.0.1", description="test", author="test", minRuntimeVersion="0.0.0"}
function PLUGIN:Available(ctx) return {} end
function PLUGIN:PreInstall(ctx) return {} end
function PLUGIN:EnvKeys(ctx) return {{key="PATH", value=ctx.path .. "/bin"}} end
%s
`, name, postUse)
		if err := os.WriteFile(filepath.Join(pluginDir, "main.lua"), []byte(script), 0600); err != nil {
			t.Fatal(err)
		}
	}
	writePlugin("shimtool", `function PLUGIN:PostUse(ctx)
  local f = assert(io.open(ctx.shimDir .. "/reshim-marker", "w"))
  f:write(ctx.version .. "|" .. ctx.scope .. "|" .. ctx.previousVersion)
  f:close()
end
`)
	writePlugin("emptytool", "")
	if err := os.MkdirAll(filepath.Join(f.home, "cache", "shimtool", "v-1.0.0", "shimtool-1.0.0"), 0755); err != nil {
		t.Fatal(err)
	}
	// Project scope selects shimtool with unlink, so reshim must downgrade it
	// to session scope (mirrors env/activate behavior).
	projectConfig := "[tools]\nshimtool = { version = \"1.0.0\", unlink = \"true\" }\n"
	if err := os.WriteFile(filepath.Join(f.work, ".vfox.toml"), []byte(projectConfig), 0600); err != nil {
		t.Fatal(err)
	}
	globalConfig := "[tools]\nghost = \"9.9.9\"\nemptytool = \"2.0.0\"\n"
	if err := os.WriteFile(filepath.Join(f.home, "vfox.toml"), []byte(globalConfig), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", root)
	t.Setenv("VFOX_HOME", f.home)
	t.Setenv("__VFOX_CURTMPPATH", f.session)
	t.Setenv(env.PathVarName, f.system)
	t.Chdir(f.work)
	return f
}

func TestReshimCmdSyncsPostUseShims(t *testing.T) {
	f := newReshimFixture(t)
	if err := reshimCmd(context.Background(), &cli.Command{}); err != nil {
		t.Fatalf("reshimCmd failed: %v", err)
	}
	// Project+unlink downgrades to session scope, so the marker lands in the
	// session shim dir with previous == current (no version switch).
	marker := filepath.Join(f.session, "shims", "reshim-marker")
	data, err := os.ReadFile(marker)
	if err != nil {
		t.Fatalf("expected PostUse marker at %s: %v", marker, err)
	}
	if string(data) != "1.0.0|session|1.0.0" {
		t.Errorf("marker = %q, want %q", data, "1.0.0|session|1.0.0")
	}
	// Scope links are recreated as well.
	if _, err := os.Lstat(filepath.Join(f.session, "shimtool")); err != nil {
		t.Errorf("expected session symlink for shimtool: %v", err)
	}
	// Reshim is idempotent.
	if err := reshimCmd(context.Background(), &cli.Command{}); err != nil {
		t.Fatalf("second reshimCmd failed: %v", err)
	}
}

func TestReshimCmdWithoutTools(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HOME", root)
	t.Setenv("VFOX_HOME", filepath.Join(root, ".vfox"))
	t.Setenv("__VFOX_CURTMPPATH", filepath.Join(root, ".vfox", "tmp", "s"))
	t.Setenv(env.PathVarName, t.TempDir())
	t.Chdir(t.TempDir())
	if err := reshimCmd(context.Background(), &cli.Command{}); err != nil {
		t.Fatalf("reshimCmd with no tools failed: %v", err)
	}
}
