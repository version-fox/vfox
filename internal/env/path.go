/*
 *    Copyright 2024 Han Li and contributors
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
	"github.com/version-fox/vfox/internal/util"
	"os"
	"path/filepath"
	"strings"
)

type PathFrom int

const (
	EmptyPaths PathFrom = iota
	OsPaths
)

// Paths is a slice of PATH.
type Paths struct {
	*util.SortedSet[string]
}

func (p *Paths) Merge(other *Paths) *Paths {
	for _, path := range other.Slice() {
		p.Add(path)
	}
	return p
}

// ToBinPaths returns a BinPaths from Paths which contains only executable files.
func (p *Paths) ToBinPaths() *Paths {
	bins := NewPaths(EmptyPaths)
	for _, path := range p.Slice() {
		dir, err := os.ReadDir(path)
		if err != nil {
			return nil
		}
		for _, d := range dir {
			if d.IsDir() {
				continue
			}
			file := filepath.Join(path, d.Name())
			if util.IsExecutable(file) {
				bins.Add(file)
			}
		}
	}
	return bins
}

// NewPaths returns a new Paths.
// from is the source of the paths.
// If from is OsPaths, it returns the paths from the environment variable PATH.
// If from is PreviousPaths, it returns the paths from the environment variable __VFOX_PREVIOUS_PATHS
// If from is neither OsPaths nor PreviousPaths, it returns an empty Paths.
func NewPaths(from PathFrom) *Paths {
	var paths []string
	switch from {
	case OsPaths:
		paths = strings.Split(os.Getenv("PATH"), string(os.PathListSeparator))
	default:

	}
	p := &Paths{
		util.NewSortedSet[string](),
	}
	for _, v := range paths {
		p.Add(v)
	}
	return p
}
