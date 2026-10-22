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
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/version-fox/vfox/internal/config"
	"github.com/version-fox/vfox/internal/pathmeta"
	"github.com/version-fox/vfox/internal/shared/logger"
)

// ShadowSource says why a PATH entry sits in front of vfox's directories.
type ShadowSource string

const (
	// ShadowSourceUnknown: no baseline was recorded, so inherited and added
	// entries cannot be told apart.
	ShadowSourceUnknown ShadowSource = "unknown"
	// ShadowSourceActivatedEnv: the directory of an active virtualenv or conda
	// environment. It is meant to win.
	ShadowSourceActivatedEnv ShadowSource = "activated-env"
	// ShadowSourcePreserved: listed in `path.preserve`.
	ShadowSourcePreserved ShadowSource = "preserved"
	// ShadowSourceInherited: already in PATH when vfox was activated, such as
	// the directories macOS path_helper or the Windows machine PATH put first.
	ShadowSourceInherited ShadowSource = "inherited"
	// ShadowSourceAdded: added to PATH after vfox was activated, such as the
	// $PNPM_HOME that `pnpm setup` prepends at the end of a shell rc file.
	ShadowSourceAdded ShadowSource = "added-after-activation"
)

// pathPolicy decides which PATH entries in front of vfox's first directory
// stay in front of it.
type pathPolicy struct {
	mode string
	// orig is the PATH recorded at activation. nil means none was recorded.
	orig map[string]struct{}
	// activated holds the directories of an active virtualenv/conda environment.
	activated map[string]struct{}
	// preserved holds the directories listed in `path.preserve`.
	preserved map[string]struct{}
}

func (m *RuntimeEnvContext) pathPolicy() pathPolicy {
	p := pathPolicy{
		mode:      config.PathModeLegacy,
		orig:      parseOrigPath(os.Getenv(OrigPathFlag)),
		activated: make(map[string]struct{}),
		preserved: make(map[string]struct{}),
	}
	if m != nil && m.UserConfig != nil && m.UserConfig.Path != nil {
		p.mode = m.UserConfig.Path.EffectiveMode()
		for _, dir := range m.UserConfig.Path.Preserve {
			if dir = expandPreservedDir(dir); dir != "" {
				p.preserved[normalizePathEntry(dir)] = struct{}{}
			}
		}
	}
	for _, dir := range activatedEnvDirs() {
		p.activated[normalizePathEntry(dir)] = struct{}{}
	}
	return p
}

// key identifies the decisions the policy takes apart from the PATH itself and
// the baseline, which stay the same within a shell session: the mode, the
// directories `path.preserve` expands to now (an environment variable it names
// can change without the PATH changing) and those of an active virtualenv or
// conda environment. Output rendered under another key is stale.
func (p pathPolicy) key() string {
	return fmt.Sprintf("%s preserve=%q activated=%q", p.mode, sortedEntries(p.preserved), sortedEntries(p.activated))
}

func sortedEntries(set map[string]struct{}) []string {
	entries := make([]string, 0, len(set))
	for entry := range set {
		entries = append(entries, entry)
	}
	sort.Strings(entries)
	return entries
}

// keepInFront reports whether an entry that sits before vfox's first directory
// stays in front of the vfox directories.
func (p pathPolicy) keepInFront(entry string) bool {
	n := normalizePathEntry(entry)
	if _, ok := p.activated[n]; ok {
		return true
	}
	if _, ok := p.preserved[n]; ok {
		return true
	}
	switch p.mode {
	case config.PathModeAggressive:
		return false
	case config.PathModeBaseline:
		if p.orig == nil {
			return true // no baseline recorded: behave like legacy
		}
		_, inherited := p.orig[n]
		return !inherited
	default:
		return true
	}
}

// sourceOf explains why the directory is in front of vfox's directories.
func (p pathPolicy) sourceOf(dir string) ShadowSource {
	n := normalizePathEntry(dir)
	if _, ok := p.activated[n]; ok {
		return ShadowSourceActivatedEnv
	}
	if _, ok := p.preserved[n]; ok {
		return ShadowSourcePreserved
	}
	if p.orig == nil {
		return ShadowSourceUnknown
	}
	if _, ok := p.orig[n]; ok {
		return ShadowSourceInherited
	}
	return ShadowSourceAdded
}

// splitSystemPaths splits pathEnv around vfox's own directories. The entries
// in front of the first vfox directory are divided by the policy: those it
// keeps in front go to prefix, the others join clean, ahead of the entries
// that were behind vfox already, so their relative order is preserved.
func splitSystemPaths(pathEnv string, p pathPolicy) (prefix *Paths, clean *Paths) {
	systemPaths := strings.Split(pathEnv, string(os.PathListSeparator))

	prefix = NewPaths(EmptyPaths)
	clean = NewPaths(EmptyPaths)

	// First pass: check if there are any vfox paths
	hasVfoxPath := false
	for _, path := range systemPaths {
		if path == "" {
			continue
		}
		if pathmeta.IsVfoxRelatedPath(filepath.Clean(path)) {
			hasVfoxPath = true
			break
		}
	}

	// If no vfox paths, all paths are clean system paths (no user-injected paths to preserve)
	if !hasVfoxPath {
		for _, path := range systemPaths {
			if path == "" {
				continue
			}
			clean.Add(path)
		}
		return prefix, clean
	}

	// Second pass: split by first vfox path
	foundVfoxPath := false
	for _, path := range systemPaths {
		if path == "" {
			continue
		}
		cleanPath := filepath.Clean(path)

		if pathmeta.IsVfoxRelatedPath(cleanPath) {
			logger.Debugf("Removing vfox path from system PATH: %s", path)
			foundVfoxPath = true
			continue
		}

		switch {
		case foundVfoxPath:
			// Path appears after first vfox path - normal system path
			clean.Add(path)
		case p.keepInFront(path):
			// Path appears before first vfox path and is meant to win (e.g., virtualenv)
			prefix.Add(path)
		default:
			logger.Debugf("Moving %s behind the vfox paths (%s mode)", path, p.mode)
			clean.Add(path)
		}
	}

	return prefix, clean
}

// BaselinePath returns the PATH to record as the activation baseline: the
// current PATH without vfox's own directories, in order.
func (m *RuntimeEnvContext) BaselinePath() *Paths {
	baseline := NewPaths(EmptyPaths)
	for _, path := range strings.Split(os.Getenv("PATH"), string(os.PathListSeparator)) {
		if path == "" || pathmeta.IsVfoxRelatedPath(filepath.Clean(path)) {
			continue
		}
		baseline.Add(path)
	}
	return baseline
}

// parseOrigPath reads the value of OrigPathFlag. nil means no baseline.
func parseOrigPath(value string) map[string]struct{} {
	if value == "" {
		return nil
	}
	sep := string(os.PathListSeparator)
	// Git Bash and MSYS2 export a list of POSIX-style paths separated by ':'.
	if runtime.GOOS == "windows" && !strings.Contains(value, ";") && strings.HasPrefix(value, "/") {
		sep = ":"
	}
	entries := make(map[string]struct{})
	for _, entry := range strings.Split(value, sep) {
		if entry = normalizePathEntry(entry); entry != "" {
			entries[entry] = struct{}{}
		}
	}
	return entries
}

// normalizePathEntry makes two spellings of the same directory compare equal.
func normalizePathEntry(entry string) string {
	entry = strings.TrimSpace(entry)
	if entry == "" {
		return ""
	}
	if runtime.GOOS == "windows" {
		// "/c/Users/me" (Git Bash, MSYS2) -> "C:/Users/me"
		if len(entry) >= 2 && entry[0] == '/' && isASCIILetter(entry[1]) && (len(entry) == 2 || entry[2] == '/') {
			entry = strings.ToUpper(entry[1:2]) + ":" + entry[2:]
			if len(entry) == 2 {
				entry += "/"
			}
		}
		return strings.ToLower(filepath.Clean(filepath.FromSlash(entry)))
	}
	return filepath.Clean(entry)
}

func isASCIILetter(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}

// activatedEnvDirs lists where an active virtualenv or conda environment keeps
// its executables. Directories that do not apply to the platform never show up
// in PATH, so listing the union is harmless.
func activatedEnvDirs() []string {
	var dirs []string
	if venv := os.Getenv("VIRTUAL_ENV"); venv != "" {
		dirs = append(dirs, filepath.Join(venv, "bin"), filepath.Join(venv, "Scripts"))
	}
	if conda := os.Getenv("CONDA_PREFIX"); conda != "" {
		dirs = append(dirs,
			filepath.Join(conda, "bin"),
			conda,
			filepath.Join(conda, "Scripts"),
			filepath.Join(conda, "Library", "bin"),
			filepath.Join(conda, "Library", "usr", "bin"),
			filepath.Join(conda, "Library", "mingw-w64", "bin"),
		)
	}
	return dirs
}

// expandPreservedDir expands environment variables and a leading ~ in a
// `path.preserve` entry.
func expandPreservedDir(dir string) string {
	dir = strings.TrimSpace(os.ExpandEnv(dir))
	if dir == "~" || strings.HasPrefix(dir, "~/") || strings.HasPrefix(dir, `~\`) {
		if home, err := os.UserHomeDir(); err == nil {
			dir = filepath.Join(home, dir[1:])
		}
	}
	return dir
}
