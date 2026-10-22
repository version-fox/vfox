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

package env

import (
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/version-fox/vfox/internal/config"
	"github.com/version-fox/vfox/internal/pathmeta"
)

// policyFor builds a policy the way pathPolicy does, from plain strings.
func policyFor(mode string, orig []string, activated, preserved []string) pathPolicy {
	p := pathPolicy{mode: mode, activated: map[string]struct{}{}, preserved: map[string]struct{}{}}
	if orig != nil {
		p.orig = map[string]struct{}{}
		for _, entry := range orig {
			p.orig[normalizePathEntry(entry)] = struct{}{}
		}
	}
	for _, entry := range activated {
		p.activated[normalizePathEntry(entry)] = struct{}{}
	}
	for _, entry := range preserved {
		p.preserved[normalizePathEntry(entry)] = struct{}{}
	}
	return p
}

func joinPath(entries ...string) string {
	return strings.Join(entries, string(os.PathListSeparator))
}

// The PATH shapes behind #337 (macOS path_helper), #732 (Windows machine PATH),
// #622 (virtualenv) and pnpm's `pnpm setup` all look the same to vfox: some
// directory sits in front of its own. Only a baseline can tell them apart.
func TestSplitSystemPathsByMode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses POSIX paths")
	}
	const vfox = "/home/me/.vfox/sdks/node/bin"
	systemDirs := []string{"/usr/local/bin", "/usr/bin", "/bin"}

	tests := []struct {
		name       string
		policy     pathPolicy
		path       []string
		wantPrefix []string
		wantClean  []string
	}{
		{
			name:       "legacy keeps everything in front, whatever it is",
			policy:     policyFor(config.PathModeLegacy, systemDirs, nil, nil),
			path:       []string{"/usr/local/bin", "/usr/bin", vfox, "/bin"},
			wantPrefix: []string{"/usr/local/bin", "/usr/bin"},
			wantClean:  []string{"/bin"},
		},
		{
			name:       "baseline: directories that were in PATH at activation go behind vfox (path_helper, #337)",
			policy:     policyFor(config.PathModeBaseline, systemDirs, nil, nil),
			path:       []string{"/usr/local/bin", "/usr/bin", vfox, "/bin"},
			wantPrefix: nil,
			wantClean:  []string{"/usr/local/bin", "/usr/bin", "/bin"},
		},
		{
			name:       "baseline: a directory added after activation stays in front (pnpm)",
			policy:     policyFor(config.PathModeBaseline, systemDirs, nil, nil),
			path:       []string{"/home/me/.local/share/pnpm/bin", vfox, "/usr/bin"},
			wantPrefix: []string{"/home/me/.local/share/pnpm/bin"},
			wantClean:  []string{"/usr/bin"},
		},
		{
			name:       "baseline: both at once, each where it belongs",
			policy:     policyFor(config.PathModeBaseline, systemDirs, nil, nil),
			path:       []string{"/home/me/venv/bin", "/usr/local/bin", vfox, "/usr/bin", "/bin"},
			wantPrefix: []string{"/home/me/venv/bin"},
			wantClean:  []string{"/usr/local/bin", "/usr/bin", "/bin"},
		},
		{
			name:       "baseline: an active virtualenv stays in front even though it was inherited",
			policy:     policyFor(config.PathModeBaseline, []string{"/home/me/venv/bin", "/usr/bin"}, []string{"/home/me/venv/bin"}, nil),
			path:       []string{"/home/me/venv/bin", vfox, "/usr/bin"},
			wantPrefix: []string{"/home/me/venv/bin"},
			wantClean:  []string{"/usr/bin"},
		},
		{
			name:       "baseline without a recorded baseline behaves like legacy",
			policy:     policyFor(config.PathModeBaseline, nil, nil, nil),
			path:       []string{"/usr/local/bin", vfox, "/usr/bin"},
			wantPrefix: []string{"/usr/local/bin"},
			wantClean:  []string{"/usr/bin"},
		},
		{
			name:       "aggressive keeps only the virtualenv and what is listed in preserve",
			policy:     policyFor(config.PathModeAggressive, nil, []string{"/home/me/venv/bin"}, []string{"/home/me/.cargo/bin"}),
			path:       []string{"/home/me/venv/bin", "/home/me/.cargo/bin", "/home/me/.local/share/pnpm/bin", vfox, "/usr/bin"},
			wantPrefix: []string{"/home/me/venv/bin", "/home/me/.cargo/bin"},
			wantClean:  []string{"/home/me/.local/share/pnpm/bin", "/usr/bin"},
		},
		{
			name:       "preserve also holds in baseline mode",
			policy:     policyFor(config.PathModeBaseline, []string{"/home/me/.cargo/bin"}, nil, []string{"/home/me/.cargo/bin"}),
			path:       []string{"/home/me/.cargo/bin", vfox, "/usr/bin"},
			wantPrefix: []string{"/home/me/.cargo/bin"},
			wantClean:  []string{"/usr/bin"},
		},
		{
			name:       "nothing changes when vfox is not in PATH yet",
			policy:     policyFor(config.PathModeAggressive, nil, nil, nil),
			path:       []string{"/usr/local/bin", "/usr/bin"},
			wantPrefix: nil,
			wantClean:  []string{"/usr/local/bin", "/usr/bin"},
		},
		{
			name:       "demoted entries keep their order and come before what was behind vfox already",
			policy:     policyFor(config.PathModeAggressive, nil, nil, nil),
			path:       []string{"/a", "/b", vfox, "/c", "/d"},
			wantPrefix: nil,
			wantClean:  []string{"/a", "/b", "/c", "/d"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prefix, clean := splitSystemPaths(joinPath(tt.path...), tt.policy)
			if got := prefix.Slice(); !equalPaths(got, tt.wantPrefix) {
				t.Errorf("prefix = %v, want %v", got, tt.wantPrefix)
			}
			if got := clean.Slice(); !equalPaths(got, tt.wantClean) {
				t.Errorf("clean = %v, want %v", got, tt.wantClean)
			}
		})
	}
}

func equalPaths(a, b []string) bool {
	return len(a) == len(b) && (len(a) == 0 || reflect.DeepEqual(a, b))
}

// pathContext builds the context SplitSystemPaths reads its settings from.
func pathContext(mode string, preserve ...string) *RuntimeEnvContext {
	return &RuntimeEnvContext{
		UserConfig: &config.Config{Path: &config.Path{Mode: mode, Preserve: preserve}},
		PathMeta:   &pathmeta.PathMeta{},
	}
}

func TestSplitSystemPathsReadsTheEnvironment(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses POSIX paths")
	}
	root := t.TempDir()
	venv := filepath.Join(root, "venv")
	vfoxDir := filepath.Join(root, ".vfox", "sdks", "node", "bin")
	inherited := filepath.Join(root, "inherited", "bin")
	added := filepath.Join(root, "added", "bin")
	cargo := filepath.Join(root, "cargo", "bin")

	t.Setenv("PATH", joinPath(inherited, added, filepath.Join(venv, "bin"), cargo, vfoxDir, "/usr/bin"))
	t.Setenv(OrigPathFlag, joinPath(inherited, filepath.Join(venv, "bin"), "/usr/bin"))
	t.Setenv("VIRTUAL_ENV", venv)
	t.Setenv("CONDA_PREFIX", "")

	prefix, clean := pathContext(config.PathModeBaseline, cargo).SplitSystemPaths()
	if got, want := prefix.Slice(), []string{added, filepath.Join(venv, "bin"), cargo}; !reflect.DeepEqual(got, want) {
		t.Errorf("prefix = %v, want %v", got, want)
	}
	if got, want := clean.Slice(), []string{inherited, "/usr/bin"}; !reflect.DeepEqual(got, want) {
		t.Errorf("clean = %v, want %v", got, want)
	}

	// The same PATH under the default settings is what vfox always did.
	prefix, _ = (&RuntimeEnvContext{UserConfig: &config.Config{}}).SplitSystemPaths()
	if got := len(prefix.Slice()); got != 4 {
		t.Errorf("legacy prefix = %v, want all four entries", prefix.Slice())
	}
}

// The key tells when cached output no longer follows the settings.
func TestPathPolicyKey(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses POSIX paths")
	}
	t.Setenv("VIRTUAL_ENV", "")
	t.Setenv("CONDA_PREFIX", "")
	t.Setenv("PNPM_HOME", "")

	base := pathContext("baseline", "/opt/a", "/opt/b").PathPolicyKey()
	for name, ctx := range map[string]*RuntimeEnvContext{
		"the same settings":        pathContext("baseline", "/opt/a", "/opt/b"),
		"another order":            pathContext("baseline", "/opt/b", "/opt/a"),
		"another spelling":         pathContext("Baseline", "/opt/a/", "/opt/./b"),
		"the same twice":           pathContext("baseline", "/opt/a", "/opt/b", "/opt/a"),
		"an empty entry on top":    pathContext("baseline", "", "/opt/a", "/opt/b"),
		"an unset variable on top": pathContext("baseline", "$PNPM_HOME", "/opt/a", "/opt/b"),
	} {
		if got := ctx.PathPolicyKey(); got != base {
			t.Errorf("%s: key = %q, want %q", name, got, base)
		}
	}

	for name, ctx := range map[string]*RuntimeEnvContext{
		"another mode":      pathContext("aggressive", "/opt/a", "/opt/b"),
		"a directory fewer": pathContext("baseline", "/opt/a"),
		"a directory more":  pathContext("baseline", "/opt/a", "/opt/b", "/opt/c"),
		"another directory": pathContext("baseline", "/opt/a", "/opt/c"),
		"nothing preserved": pathContext("baseline"),
	} {
		if got := ctx.PathPolicyKey(); got == base {
			t.Errorf("%s: key = %q, want it to differ from %q", name, got, base)
		}
	}

	t.Run("an environment variable of path.preserve", func(t *testing.T) {
		ctx := pathContext("baseline", "$PNPM_HOME")
		unset := ctx.PathPolicyKey()
		t.Setenv("PNPM_HOME", "/data/pnpm")
		set := ctx.PathPolicyKey()
		t.Setenv("PNPM_HOME", "/data/other")
		other := ctx.PathPolicyKey()
		if unset == set || set == other || unset == other {
			t.Errorf("keys = %q, %q, %q, want three different ones", unset, set, other)
		}
	})

	t.Run("an active virtualenv or conda environment", func(t *testing.T) {
		ctx := pathContext("baseline")
		without := ctx.PathPolicyKey()
		t.Setenv("VIRTUAL_ENV", "/home/me/venv")
		venv := ctx.PathPolicyKey()
		t.Setenv("CONDA_PREFIX", "/home/me/conda")
		both := ctx.PathPolicyKey()
		if without == venv || venv == both {
			t.Errorf("keys = %q, %q, %q, want three different ones", without, venv, both)
		}
	})

	t.Run("no settings", func(t *testing.T) {
		want := pathContext(config.PathModeLegacy).PathPolicyKey()
		for name, ctx := range map[string]*RuntimeEnvContext{
			"a nil context": nil,
			"no config":     {},
			"no path":       {UserConfig: &config.Config{}},
			"empty mode":    pathContext(""),
			"unknown mode":  pathContext("whatever"),
		} {
			if got := ctx.PathPolicyKey(); got != want {
				t.Errorf("%s: key = %q, want the legacy default %q", name, got, want)
			}
		}
	})
}

func TestPathExpansion(t *testing.T) {
	if runtime.GOOS == "windows" {
		return
	}
	t.Setenv("PNPM_HOME", "/data/pnpm")
	t.Setenv("HOME", "/home/me")
	if got := expandPreservedDir("$PNPM_HOME/bin"); got != "/data/pnpm/bin" {
		t.Errorf("env expansion: %q", got)
	}
	if got := expandPreservedDir("~/.cargo/bin"); got != filepath.Join("/home/me", ".cargo/bin") {
		t.Errorf("tilde expansion: %q", got)
	}
}

func TestBaselinePathDropsVfoxDirectories(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses POSIX paths")
	}
	t.Setenv("PATH", joinPath("/usr/local/bin", "/home/me/.vfox/sdks/node/bin", "/usr/bin", "/proj/.vfox/sdks/go/bin", "/usr/local/bin", "", "/bin"))
	got := (&RuntimeEnvContext{}).BaselinePath().Slice()
	want := []string{"/usr/local/bin", "/usr/bin", "/bin"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("BaselinePath() = %v, want %v", got, want)
	}
}

func TestParseOrigPathAndNormalize(t *testing.T) {
	if parseOrigPath("") != nil {
		t.Error("an empty value means no baseline, not an empty one")
	}
	got := parseOrigPath(joinPath("/usr/bin", "", "/usr/bin/", "/opt/../opt/tools"))
	if runtime.GOOS != "windows" {
		for _, want := range []string{"/usr/bin", "/opt/tools"} {
			if _, ok := got[want]; !ok {
				t.Errorf("baseline lacks %q: %v", want, got)
			}
		}
		if len(got) != 2 {
			t.Errorf("baseline = %v", got)
		}
	}
	if normalizePathEntry("  ") != "" {
		t.Error("blank entries are dropped")
	}
}

func TestActivatedEnvDirs(t *testing.T) {
	t.Setenv("VIRTUAL_ENV", "")
	t.Setenv("CONDA_PREFIX", "")
	if dirs := activatedEnvDirs(); len(dirs) != 0 {
		t.Errorf("dirs = %v, want none", dirs)
	}
	t.Setenv("VIRTUAL_ENV", filepath.Join("v", "env"))
	t.Setenv("CONDA_PREFIX", filepath.Join("c", "env"))
	dirs := activatedEnvDirs()
	for _, want := range []string{
		filepath.Join("v", "env", "bin"), filepath.Join("v", "env", "Scripts"),
		filepath.Join("c", "env", "bin"), filepath.Join("c", "env"), filepath.Join("c", "env", "Library", "bin"),
	} {
		found := false
		for _, dir := range dirs {
			found = found || dir == want
		}
		if !found {
			t.Errorf("activatedEnvDirs() lacks %q: %v", want, dirs)
		}
	}
}

func TestSplitSystemPathsSkipsEmptyEntries(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses POSIX paths")
	}
	const vfox = "/home/me/.vfox/sdks/node/bin"
	prefix, clean := splitSystemPaths(joinPath("/a", "", vfox, "", "/b", ""), policyFor(config.PathModeLegacy, nil, nil, nil))
	if got := prefix.Slice(); !equalPaths(got, []string{"/a"}) {
		t.Errorf("prefix = %v, want [/a]", got)
	}
	if got := clean.Slice(); !equalPaths(got, []string{"/b"}) {
		t.Errorf("clean = %v, want [/b]", got)
	}
}
