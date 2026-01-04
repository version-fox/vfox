//go:build windows

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
	"strings"
)

const Newline = "\r\n"
const PathVarName = "Path"

func (p *Paths) String() string {

	if os.Getenv(HookFlag) == "bash" {
		pps := p.Slice()
		paths := make([]string, 0)
		for _, path := range pps {
			path = filepath.ToSlash(path)
			// Convert drive letter (e.g., "C:") to "/c"
			if len(path) > 1 && path[1] == ':' {
				path = "/" + strings.ToLower(string(path[0])) + path[2:]
			}
			paths = append(paths, path)
		}
		return strings.Join(paths, ":")
	} else {
		return strings.Join(p.Slice(), ";")
	}
}

func (p *Paths) Add(path string) bool {
	path = filepath.FromSlash(path)
	return p.SortedSet.Add(path)
}
