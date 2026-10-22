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

// Executable is a command found in a directory.
type Executable struct {
	Name string // as typed in a shell: "node", not "node.exe"
	Path string // full path of the file
	Dir  string // directory that holds it
}

// ListExecutables lists the commands in dirs. When a name occurs in several
// directories the first one wins, as it would in a shell. Directories that do
// not exist or cannot be read are skipped.
func ListExecutables(dirs []string) []Executable {
	var found []Executable
	seen := make(map[string]struct{})
	exts := pathExts()
	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			path := filepath.Join(dir, entry.Name())
			info, err := os.Stat(path) // follow links: SDK bin directories are full of them
			if err != nil || info.IsDir() {
				continue
			}
			name, ok := commandName(entry.Name(), info, exts)
			if !ok {
				continue
			}
			key := name
			if runtime.GOOS == "windows" {
				key = strings.ToLower(name)
			}
			if _, dup := seen[key]; dup {
				continue
			}
			seen[key] = struct{}{}
			found = append(found, Executable{Name: name, Path: path, Dir: dir})
		}
	}
	return found
}

// commandName returns the name a shell uses to run the file.
func commandName(file string, info os.FileInfo, exts []string) (string, bool) {
	if runtime.GOOS == "windows" {
		return windowsCommandName(file, exts)
	}
	if info.Mode().IsRegular() && info.Mode()&0111 != 0 {
		return file, true
	}
	return "", false
}

// windowsCommandName strips an extension listed in exts (PATHEXT).
func windowsCommandName(file string, exts []string) (string, bool) {
	ext := filepath.Ext(file)
	if ext == "" {
		return "", false
	}
	for _, candidate := range exts {
		if strings.EqualFold(ext, candidate) {
			return strings.TrimSuffix(file, ext), true
		}
	}
	return "", false
}

// pathExts returns the extensions Windows treats as runnable. PowerShell
// scripts are included because SDKs ship launchers such as npm.ps1.
func pathExts() []string {
	if runtime.GOOS != "windows" {
		return nil
	}
	exts := []string{".ps1"}
	value := os.Getenv("PATHEXT")
	if value == "" {
		value = ".COM;.EXE;.BAT;.CMD"
	}
	for _, ext := range strings.Split(value, ";") {
		if ext = strings.TrimSpace(ext); ext != "" {
			exts = append(exts, ext)
		}
	}
	return exts
}
