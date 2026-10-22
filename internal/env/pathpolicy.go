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
	"runtime"
	"strings"
)

// ShadowSource says why a PATH entry sits in front of vfox's directories.
type ShadowSource string

const (
	// ShadowSourceUnknown: nothing is known about where the entry comes from.
	ShadowSourceUnknown ShadowSource = "unknown"
	// ShadowSourceActivatedEnv: the directory of an active virtualenv or conda
	// environment. It is meant to win.
	ShadowSourceActivatedEnv ShadowSource = "activated-env"
)

// pathPolicy classifies the PATH entries in front of vfox's first directory.
type pathPolicy struct {
	// activated holds the directories of an active virtualenv/conda environment.
	activated map[string]struct{}
}

func (m *RuntimeEnvContext) pathPolicy() pathPolicy {
	p := pathPolicy{activated: make(map[string]struct{})}
	for _, dir := range activatedEnvDirs() {
		p.activated[normalizePathEntry(dir)] = struct{}{}
	}
	return p
}

// sourceOf explains why the directory is in front of vfox's directories.
func (p pathPolicy) sourceOf(dir string) ShadowSource {
	n := normalizePathEntry(dir)
	if _, ok := p.activated[n]; ok {
		return ShadowSourceActivatedEnv
	}
	return ShadowSourceUnknown
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
