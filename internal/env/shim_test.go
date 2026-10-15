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
	"testing"

	"github.com/version-fox/vfox/internal/pathmeta"
)

func TestGetShimDirPathByScope(t *testing.T) {
	meta := &pathmeta.PathMeta{}
	meta.Working.GlobalShimDir = "/home/user/.vfox/shims"
	meta.Working.ProjectShimDir = "/home/user/project/.vfox/shims"
	meta.Working.SessionShimDir = "/home/user/.vfox/tmp/20260102-1/shims"

	ctx := &RuntimeEnvContext{PathMeta: meta}

	cases := []struct {
		scope UseScope
		want  string
	}{
		{Global, "/home/user/.vfox/shims"},
		{Project, "/home/user/project/.vfox/shims"},
		{Session, "/home/user/.vfox/tmp/20260102-1/shims"},
		{UseScope(99), ""},
	}
	for _, tc := range cases {
		if got := ctx.GetShimDirPathByScope(tc.scope); got != tc.want {
			t.Errorf("GetShimDirPathByScope(%v) = %q, want %q", tc.scope, got, tc.want)
		}
	}
}
