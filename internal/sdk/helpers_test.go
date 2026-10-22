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
	"runtime"
	"testing"

	"github.com/version-fox/vfox/internal/config"
	"github.com/version-fox/vfox/internal/env"
	"github.com/version-fox/vfox/internal/pathmeta"
	"github.com/version-fox/vfox/internal/plugin"
)

// testPlugin implements the hooks the receipt, link and diagnose code calls.
// A hook that is nil is absent, which HasFunction reports; anything that is not
// overridden panics through the nil embedded interface, so a test notices when
// the code under test starts calling something new.
type testPlugin struct {
	plugin.Plugin
	preInstall   func(*plugin.PreInstallHookCtx) (*plugin.PreInstallHookResult, error)
	postInstall  func(*plugin.PostInstallHookCtx) error
	preUninstall func(*plugin.PreUninstallHookCtx) error
	envKeys      func(*plugin.EnvKeysHookCtx) ([]*plugin.EnvKeysHookResultItem, error)
	fingerprint  func(*plugin.FingerprintHookCtx) (*plugin.FingerprintHookResult, error)

	preInstallCalls  int
	preUninstallCall int
}

func (p *testPlugin) HasFunction(name string) bool {
	switch name {
	case "PostInstall":
		return p.postInstall != nil
	case "PreUninstall":
		return p.preUninstall != nil
	case "Fingerprint":
		return p.fingerprint != nil
	}
	return false
}

func (p *testPlugin) PreInstall(ctx *plugin.PreInstallHookCtx) (*plugin.PreInstallHookResult, error) {
	p.preInstallCalls++
	if p.preInstall != nil {
		return p.preInstall(ctx)
	}
	return &plugin.PreInstallHookResult{PreInstallPackageItem: &plugin.PreInstallPackageItem{Version: ctx.Version}}, nil
}

func (p *testPlugin) PostInstall(ctx *plugin.PostInstallHookCtx) error { return p.postInstall(ctx) }

func (p *testPlugin) PreUninstall(ctx *plugin.PreUninstallHookCtx) error {
	p.preUninstallCall++
	return p.preUninstall(ctx)
}

func (p *testPlugin) EnvKeys(ctx *plugin.EnvKeysHookCtx) ([]*plugin.EnvKeysHookResultItem, error) {
	if p.envKeys != nil {
		return p.envKeys(ctx)
	}
	return []*plugin.EnvKeysHookResultItem{{Key: "PATH", Value: filepath.Join(ctx.Path, "bin")}}, nil
}

func (p *testPlugin) Fingerprint(ctx *plugin.FingerprintHookCtx) (*plugin.FingerprintHookResult, error) {
	return p.fingerprint(ctx)
}

// withCommand makes PostInstall put a command into the installed SDK's bin
// directory, so that the installation looks like a real one.
func (p *testPlugin) withCommand(t *testing.T, name string) *testPlugin {
	p.postInstall = func(ctx *plugin.PostInstallHookCtx) error {
		main := ctx.SdkInfo["test-sdk"].Path
		return writeTestCommandAt(filepath.Join(main, "bin"), name)
	}
	return p
}

// newTestSDK returns an SDK with a complete environment context, rooted in a
// temporary directory.
func newTestSDK(t *testing.T, p *testPlugin) *impl {
	t.Helper()
	root := t.TempDir()
	home := filepath.Join(root, "home")
	meta := &pathmeta.PathMeta{
		User: pathmeta.UserPaths{Home: home},
		Working: pathmeta.WorkingPaths{
			Directory:     filepath.Join(root, "project"),
			ProjectSdkDir: filepath.Join(root, "project", ".vfox", "sdks"),
			SessionSdkDir: filepath.Join(root, "session"),
			GlobalSdkDir:  filepath.Join(home, "sdks"),
		},
	}
	for _, dir := range []string{home, meta.Working.Directory, meta.Working.SessionSdkDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
	}
	// Whatever shell the tests run in must not leak into the PATH checks.
	t.Setenv(env.OrigPathFlag, "")
	t.Setenv("VIRTUAL_ENV", "")
	t.Setenv("CONDA_PREFIX", "")
	return &impl{
		Name:        "test-sdk",
		InstallPath: filepath.Join(root, "installs", "test-sdk"),
		envContext: &env.RuntimeEnvContext{
			UserConfig:        &config.Config{},
			PathMeta:          meta,
			RuntimeVersion:    "9.9.9",
			CurrentWorkingDir: meta.Working.Directory,
		},
		plugin: &plugin.Wrapper{
			Metadata: &plugin.Metadata{Name: "test-sdk", Version: "0.0.1"},
			Plugin:   p,
		},
	}
}

// writeTestCommandAt creates something a shell would run. Windows decides by
// extension, everything else by the executable bit.
func writeTestCommandAt(dir, name string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(commandFile(dir, name), []byte("echo "+name), 0755)
}

// commandFile is where writeTestCommandAt puts the command called name in dir.
func commandFile(dir, name string) string {
	if runtime.GOOS == "windows" {
		name += ".cmd"
	}
	return filepath.Join(dir, name)
}

// binCommand is where the command called name is in the bin directory of the
// SDK that lives at root.
func binCommand(root, name string) string {
	return commandFile(filepath.Join(root, "bin"), name)
}

func writeTestCommand(t *testing.T, dir, name string) {
	t.Helper()
	if err := writeTestCommandAt(dir, name); err != nil {
		t.Fatal(err)
	}
}

// newExternalSDK creates an SDK somebody else installed: <dir>/bin/<commands>.
func newExternalSDK(t *testing.T, commands ...string) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "external-sdk")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	for _, name := range commands {
		writeTestCommand(t, filepath.Join(dir, "bin"), name)
	}
	return dir
}

func mustInstall(t *testing.T, s *impl, version Version) {
	t.Helper()
	if err := s.Install(version); err != nil {
		t.Fatalf("Install(%s): %v", version, err)
	}
}

func mustLink(t *testing.T, s *impl, version Version, target string) {
	t.Helper()
	if err := s.Link(version, target, LinkOptions{}); err != nil {
		t.Fatalf("Link(%s): %v", version, err)
	}
}

func exists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}

func findingIDs(findings []Finding) []string {
	ids := make([]string, 0, len(findings))
	for _, f := range findings {
		ids = append(ids, f.ID)
	}
	return ids
}

func findingByID(findings []Finding, id string) *Finding {
	for i := range findings {
		if findings[i].ID == id {
			return &findings[i]
		}
	}
	return nil
}
