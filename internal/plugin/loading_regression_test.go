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

package plugin

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/version-fox/vfox/internal/config"
	"github.com/version-fox/vfox/internal/env"
)

func writePluginFile(t *testing.T, dir, name, body string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

const minimalPlugin = `PLUGIN = {name = "example", version = "0.0.1"}
function PLUGIN:Available(ctx) return {{version = "1.0.0"}} end
function PLUGIN:PreInstall(ctx) return {version = ctx.version, url = "example.zip"} end
function PLUGIN:EnvKeys(ctx) return {{key = "PATH", value = ctx.path}} end
`

func TestLoadingRejectsMalformedPlugin(t *testing.T) {
	for _, tc := range []struct{ name, source, want string }{
		{"object", `PLUGIN = 42`, "PLUGIN must be a table"},
		{"metadata", minimalPlugin + `PLUGIN.name = {}`, "name"},
		{"required hook", minimalPlugin + `PLUGIN.Available = true`, "Available"},
		{"optional hook", minimalPlugin + `PLUGIN.PreUse = true`, "PreUse"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			writePluginFile(t, dir, "main.lua", tc.source)
			p, err := CreatePlugin(dir, &env.RuntimeEnvContext{UserConfig: config.DefaultConfig})
			if p != nil {
				defer p.Close()
			}
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestPreUseHookRegistration(t *testing.T) {
	hook, ok := HookFuncMap["PreUse"]
	if !ok || hook.Name != "PreUse" || hook.Filename != "pre_use" {
		t.Fatalf("PreUse registration = %+v, found = %t", hook, ok)
	}
}

func TestHookRejectsNonTableResult(t *testing.T) {
	dir := t.TempDir()
	writePluginFile(t, dir, "main.lua", minimalPlugin+`function PLUGIN:Available(ctx) return "wrong" end`)
	p, err := CreatePlugin(dir, &env.RuntimeEnvContext{UserConfig: config.DefaultConfig})
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	_, err = p.Available(&AvailableHookCtx{})
	if err == nil || err == ErrNoResultProvide {
		t.Fatalf("wrong result type must be a conversion error, got %v", err)
	}
}
