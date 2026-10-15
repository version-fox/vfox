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

package sdk

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/version-fox/vfox/internal/config"
	"github.com/version-fox/vfox/internal/env"
	"github.com/version-fox/vfox/internal/pathmeta"
	"github.com/version-fox/vfox/internal/plugin"
)

// TriggerPostUse must be a safe no-op for SDK instances without a plugin
// (e.g. unit-test fixtures), preserving backward compatibility for plugins
// that do not implement the optional PostUse hook.
func TestTriggerPostUse_NilPluginNoop(t *testing.T) {
	tempDir := t.TempDir()
	ctx := &env.RuntimeEnvContext{
		PathMeta: &pathmeta.PathMeta{
			Working: pathmeta.WorkingPaths{
				Directory:      tempDir,
				GlobalShimDir:  filepath.Join(tempDir, "shims-global"),
				ProjectShimDir: filepath.Join(tempDir, "project", ".vfox", "shims"),
				SessionShimDir: filepath.Join(tempDir, "session", "shims"),
			},
		},
	}
	s := &impl{Name: "test-sdk", envContext: ctx, InstallPath: filepath.Join(tempDir, "installs")}
	if err := s.TriggerPostUse("1.0.0", env.Global, "0.9.0"); err != nil {
		t.Fatalf("expected nil-plugin TriggerPostUse to be a no-op, got %v", err)
	}
}

// newPostUseRecorderSdk builds a real SDK backed by the postuse_recorder test
// plugin and a fake installation of the given versions.
func newPostUseRecorderSdk(t *testing.T) (*impl, *env.RuntimeEnvContext, string) {
	t.Helper()
	tempDir := t.TempDir()
	userHome := filepath.Join(tempDir, "user")
	if err := os.MkdirAll(userHome, 0755); err != nil {
		t.Fatal(err)
	}
	pathMeta := &pathmeta.PathMeta{
		User: pathmeta.UserPaths{
			Home: userHome,
		},
		Working: pathmeta.WorkingPaths{
			Directory:      filepath.Join(tempDir, "project"),
			ProjectSdkDir:  filepath.Join(tempDir, "project", ".vfox", "sdks"),
			SessionSdkDir:  filepath.Join(tempDir, "session"),
			GlobalSdkDir:   filepath.Join(userHome, "sdks"),
			ProjectShimDir: filepath.Join(tempDir, "project", ".vfox", "shims"),
			SessionShimDir: filepath.Join(tempDir, "session", "shims"),
			GlobalShimDir:  filepath.Join(userHome, "shims"),
		},
	}
	envCtx := &env.RuntimeEnvContext{
		UserConfig: config.DefaultConfig,
		PathMeta:   pathMeta,
	}
	wrapper, err := plugin.CreatePlugin("../plugin/testdata/plugins/postuse_recorder", envCtx)
	if err != nil {
		t.Fatalf("CreatePlugin failed: %v", err)
	}
	t.Cleanup(wrapper.Close)
	installPath := filepath.Join(tempDir, "installs", "postuse_recorder")
	for _, version := range []string{"1.0.0", "2.0.0"} {
		if err := os.MkdirAll(filepath.Join(installPath, "v-"+version, "postuse_recorder-"+version), 0755); err != nil {
			t.Fatal(err)
		}
	}
	s := &impl{
		Name:        "postuse_recorder",
		envContext:  envCtx,
		plugin:      wrapper,
		InstallPath: installPath,
	}
	return s, envCtx, tempDir
}

// TriggerPostUse with a real plugin must deliver the resolved version,
// previous version, scope, main runtime and shim dir to Lua.
func TestTriggerPostUse_RecordsContext(t *testing.T) {
	s, envCtx, _ := newPostUseRecorderSdk(t)
	trace := filepath.Join(t.TempDir(), "trace")
	t.Setenv("VFOX_TEST_POSTUSE_TRACE", trace)

	if err := s.TriggerPostUse("2.0.0", env.Project, "1.0.0"); err != nil {
		t.Fatalf("TriggerPostUse failed: %v", err)
	}
	data, err := os.ReadFile(trace)
	if err != nil {
		t.Fatalf("expected trace file: %v", err)
	}
	want := "2.0.0|project|1.0.0|postuse_recorder|2.0.0|" + envCtx.PathMeta.Working.ProjectShimDir + "\n"
	if string(data) != want {
		t.Errorf("trace = %q, want %q", data, want)
	}
	marker, err := os.ReadFile(filepath.Join(envCtx.PathMeta.Working.ProjectShimDir, "postuse-marker"))
	if err != nil {
		t.Fatalf("expected shim marker: %v", err)
	}
	if string(marker) != "2.0.0" {
		t.Errorf("marker = %q, want %q", marker, "2.0.0")
	}
}

// Use must capture the previously selected version in the same scope and pass
// it to PostUse, so plugins can prune shims that point at the old runtime.
func TestUseWithConfig_TriggersPostUse(t *testing.T) {
	s, envCtx, _ := newPostUseRecorderSdk(t)
	trace := filepath.Join(t.TempDir(), "trace")
	t.Setenv("VFOX_TEST_POSTUSE_TRACE", trace)
	t.Setenv(env.HookFlag, "bash")

	if err := s.UseWithConfig("1.0.0", env.Global, false); err != nil {
		t.Fatalf("first Use failed: %v", err)
	}
	if err := s.UseWithConfig("2.0.0", env.Global, false); err != nil {
		t.Fatalf("second Use failed: %v", err)
	}
	data, err := os.ReadFile(trace)
	if err != nil {
		t.Fatalf("expected trace file: %v", err)
	}
	shimDir := envCtx.PathMeta.Working.GlobalShimDir
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 PostUse calls, got %d: %q", len(lines), data)
	}
	if lines[0] != "1.0.0|global||postuse_recorder|1.0.0|"+shimDir {
		t.Errorf("first call = %q", lines[0])
	}
	if lines[1] != "2.0.0|global|1.0.0|postuse_recorder|2.0.0|"+shimDir {
		t.Errorf("second call = %q", lines[1])
	}
}
