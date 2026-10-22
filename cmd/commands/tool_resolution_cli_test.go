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
	"os"
	"path/filepath"
	"testing"

	"github.com/version-fox/vfox/internal/env"
)

func TestVersionInEffectIsWhatEnvFallsBackTo(t *testing.T) {
	f := newCLIFixture(t, false)
	f.installOnDisk(t, "alpha", "1.0.0", "alphatool")
	must(t, os.WriteFile(filepath.Join(f.userHome, ".vfox.toml"), []byte("[tools]\nalpha = \"1.0.0\"\nplain = \"9.9.9\"\n"), 0644))
	must(t, os.WriteFile(filepath.Join(f.work, ".vfox.toml"), []byte("[tools]\nalpha = \"2.0.0\"\n"), 0644)) // not installed
	manager := f.manager(t)
	alpha, err := manager.LookupSdk("alpha")
	must(t, err)
	plain, err := manager.LookupSdk("plain")
	must(t, err)

	inEffect := versionInEffect(manager)
	if inEffect == nil {
		t.Fatal("the version files can be read, so there is a resolver")
	}
	if got := inEffect(alpha); got != "1.0.0" {
		t.Errorf("alpha: %q, want the installed version the global scope asks for", got)
	}
	// Sdk.Current stops at the highest-priority version, which is not installed.
	if got := alpha.Current(); got != "" {
		t.Errorf("Current() = %q: the situation of this test is gone", got)
	}
	if got := inEffect(plain); got != "" {
		t.Errorf("plain: %q, want none: the version it is configured for is not installed", got)
	}

	// Version files that cannot be read: the caller goes by Current instead.
	must(t, os.WriteFile(filepath.Join(f.work, ".vfox.toml"), []byte("[tools\n"), 0644))
	if versionInEffect(manager) != nil {
		t.Error("without readable version files there is nothing to resolve with")
	}
}

// Legacy version files fill in the tools that .vfox.toml does not name, the way
// env and activate read them.
func TestLoadToolChainLetsLegacyFilesFillGapsOnly(t *testing.T) {
	f := newCLIFixture(t, false)
	f.writePlugin(t, "legacy", legacyPlugin)
	must(t, os.WriteFile(filepath.Join(f.userHome, ".legacy_filenames"), []byte("legacy true\n"), 0644))
	must(t, os.WriteFile(filepath.Join(f.work, ".legacy-version"), []byte("1.0.0\n"), 0644))
	manager := f.manager(t)

	version := func(scopes ...env.UseScope) (string, bool) {
		t.Helper()
		chain, err := loadToolChain(manager, scopes...)
		must(t, err)
		v, _, ok := chain.GetToolVersion("legacy")
		return v, ok
	}

	if v, ok := version(env.Global, env.Session, env.Project); !ok || v != "1.0.0" {
		t.Errorf("legacy = %q (%v), want the version of the legacy file", v, ok)
	}
	if v, ok := version(env.Global, env.Project); !ok || v != "1.0.0" {
		t.Errorf("legacy = %q (%v) without the session scope", v, ok)
	}
	if _, ok := version(env.Global); ok {
		t.Error("with no project scope there is no project to read a legacy file for")
	}

	must(t, os.WriteFile(filepath.Join(f.work, ".vfox.toml"), []byte("[tools]\nlegacy = \"3.0.0\"\n"), 0644))
	if v, ok := version(env.Global, env.Session, env.Project); !ok || v != "3.0.0" {
		t.Errorf("legacy = %q (%v), want .vfox.toml to win over the legacy file", v, ok)
	}
}
