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
	"sort"
	"strings"
)

// Shadow is a command of an SDK that a different executable in front of vfox's
// directories keeps from running.
type Shadow struct {
	Name   string       // the command that is shadowed
	Winner string       // full path of the executable that runs instead
	Dir    string       // directory of Winner
	Source ShadowSource // why Dir is in front of vfox's directories
}

// FindShadows reports which of the provided commands are shadowed by an
// executable in the directories that stay in front of vfox's own. It evaluates
// the order `vfox env` is going to export, so it can run before the shell has
// picked the new version up. An executable that is the very file the SDK
// provides, such as a link to it, shadows nothing.
func (m *RuntimeEnvContext) FindShadows(provided []Executable) []Shadow {
	prefix, _ := m.SplitSystemPaths()
	return findShadows(provided, prefix.Slice(), m.pathPolicy())
}

func findShadows(provided []Executable, prefixDirs []string, policy pathPolicy) []Shadow {
	if len(provided) == 0 || len(prefixDirs) == 0 {
		return nil
	}
	wanted := make(map[string]Executable, len(provided)) // lookup key -> the SDK's command
	for _, exe := range provided {
		wanted[lookupKey(exe.Name)] = exe
	}
	exts := pathExts()
	var shadows []Shadow
	for _, dir := range prefixDirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			// The directory in front may be a whole system bin directory, with
			// thousands of files: look at the name before asking the file system
			// anything, and only stat what could be one of the commands.
			exe, ok := wanted[lookupKey(candidateName(entry.Name(), exts))]
			if !ok {
				continue
			}
			path := filepath.Join(dir, entry.Name())
			info, err := os.Stat(path)
			if err != nil || info.IsDir() {
				continue
			}
			if _, isCommand := commandName(entry.Name(), info, exts); !isCommand {
				continue
			}
			delete(wanted, lookupKey(exe.Name)) // the first directory in PATH order wins, whatever it is
			if sameFile(info, exe.Path) {
				continue // it runs the SDK's own executable
			}
			shadows = append(shadows, Shadow{Name: exe.Name, Winner: path, Dir: dir, Source: policy.sourceOf(dir)})
		}
	}
	sort.Slice(shadows, func(i, j int) bool { return shadows[i].Name < shadows[j].Name })
	return shadows
}

// sameFile reports whether the file at path, which info describes, is the file
// at want once links are followed: a symlink or hard link to an SDK's command
// runs that command. A want that cannot be read is not the same file.
func sameFile(info os.FileInfo, want string) bool {
	wantInfo, err := os.Stat(want)
	return err == nil && os.SameFile(info, wantInfo)
}

// candidateName is the command a file name could stand for, before the file
// system confirms it is one: on Windows the name without a PATHEXT extension.
func candidateName(file string, exts []string) string {
	if runtime.GOOS == "windows" {
		if name, ok := windowsCommandName(file, exts); ok {
			return name
		}
	}
	return file
}

func lookupKey(name string) string {
	if runtime.GOOS == "windows" {
		return strings.ToLower(name)
	}
	return name
}
