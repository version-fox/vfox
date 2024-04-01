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
	"io"
	"os"
	"strings"
)

type Manager interface {
	Flush() error
	Load(envs *Envs) error
	Get(key string) (string, bool)
	Remove(envs *Envs) error
	io.Closer
}

// Vars is a map of environment variables
type Vars map[string]*string

// Envs is a struct that contains environment variables and PATH.
type Envs struct {
	Variables Vars
	Paths     *Paths
}

type PathFrom int

const (
	EmptyPaths PathFrom = iota
	OsPaths
	PreviousPaths
)

// Paths is a slice of PATH.
type Paths struct {
	util.Set[string]
}

func (p *Paths) Merge(other *Paths) *Paths {
	for _, path := range other.Slice() {
		p.Add(path)
	}
	return p
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
	case PreviousPaths:
		if preStr := os.Getenv(PreviousPathsFlag); preStr != "" {
			paths = strings.Split(preStr, string(os.PathListSeparator))
		}
	default:

	}
	return &Paths{
		util.NewSortedSetWithSlice(paths),
	}
}
