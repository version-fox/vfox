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

package sdk

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/version-fox/vfox/internal/env"
)

// A runtime directory can be a symlink (a junction on Windows): an SDK that
// lives elsewhere, linked into the install path. DirEntry.IsDir() is false for
// those, so every way of asking whether a version is installed has to follow the
// link, or they disagree with each other.
func TestRuntimeDirectoryThatIsALink(t *testing.T) {
	s := newTestSDK(t, &testPlugin{})
	external := newExternalSDK(t, "tool")
	main := filepath.Join(s.packagePath("ext"), "test-sdk-ext")
	if err := os.MkdirAll(s.packagePath("ext"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := env.CreateDirSymlink(external, main); err != nil {
		t.Fatal(err)
	}

	if !s.CheckRuntimeExist("ext") {
		t.Error("CheckRuntimeExist: a link to a directory is an installed runtime")
	}
	if got := s.InstalledList(); len(got) != 1 || got[0] != "ext" {
		t.Errorf("InstalledList = %v", got)
	}
	pkg, err := s.GetRuntimePackage("ext")
	if err != nil {
		t.Fatalf("GetRuntimePackage: %v, although CheckRuntimeExist says the version is there", err)
	}
	if pkg.Runtime.Name != "test-sdk" || pkg.Runtime.Path != main || pkg.PackagePath != s.packagePath("ext") {
		t.Errorf("runtime package = %+v", pkg.Runtime)
	}
}

func TestRuntimeLinkWhoseTargetIsGone(t *testing.T) {
	s := newTestSDK(t, &testPlugin{})
	external := newExternalSDK(t, "tool")
	if err := os.MkdirAll(s.packagePath("ext"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := env.CreateDirSymlink(external, filepath.Join(s.packagePath("ext"), "test-sdk-ext")); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(external); err != nil {
		t.Fatal(err)
	}

	if s.CheckRuntimeExist("ext") {
		t.Error("a link to nothing is not an installed runtime")
	}
	if _, err := s.GetRuntimePackage("ext"); err == nil {
		t.Error("GetRuntimePackage must agree with CheckRuntimeExist")
	}
	if got := s.InstalledList(); len(got) != 0 {
		t.Errorf("InstalledList = %v, want nothing", got)
	}
}
