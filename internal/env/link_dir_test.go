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
	"testing"

	"github.com/version-fox/vfox/internal/pathmeta"
)

// In the home directory the project link directory is the global one, also
// when the working directory reaches it through a symlink; project links go to
// the session directory there. Anywhere else they stay in the project.
func TestGetLinkDirPathByScopeProjectInHome(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	elsewhere := filepath.Join(root, "project")
	session := filepath.Join(root, "session")
	for _, dir := range []string{filepath.Join(home, ".vfox", "sdks"), elsewhere} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
	}
	link := filepath.Join(root, "home-link")
	if err := os.Symlink(home, link); err != nil {
		t.Skipf("symlinks are not available: %v", err)
	}
	for _, c := range []struct {
		name, work, want string
	}{
		{"the home directory", home, session},
		{"the home directory through a symlink", link, session},
		{"another directory", elsewhere, filepath.Join(elsewhere, ".vfox", "sdks")},
	} {
		ctx := &RuntimeEnvContext{PathMeta: &pathmeta.PathMeta{Working: pathmeta.WorkingPaths{
			ProjectSdkDir: filepath.Join(c.work, ".vfox", "sdks"),
			SessionSdkDir: session,
			GlobalSdkDir:  filepath.Join(home, ".vfox", "sdks"),
		}}}
		if got := ctx.GetLinkDirPathByScope(Project); got != c.want {
			t.Errorf("%s: project link directory = %s, want %s", c.name, got, c.want)
		}
	}
}
