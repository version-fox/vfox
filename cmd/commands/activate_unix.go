//go:build darwin || linux

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
	"github.com/version-fox/vfox/internal/env"
	"github.com/version-fox/vfox/internal/pathmeta"
)

// generatePATH generates the PATH string with three layers
func generatePATH(pathMeta *pathmeta.PathMeta) *env.Paths {
	// Unix: relative for project, absolute for others
	paths := env.NewPaths(env.EmptyPaths)
	paths.Add(pathMeta.Working.ProjectShim)
	paths.Add(pathMeta.Working.SessionShim)
	paths.Add(pathMeta.Working.GlobalShim)
	paths.Add("$PATH")
	return paths
}
