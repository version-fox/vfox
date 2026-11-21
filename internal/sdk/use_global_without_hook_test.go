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
	"errors"
	"path/filepath"
	"testing"

	"github.com/version-fox/vfox/internal/env"
)

// UseGlobalWithoutHook records the version in the global config and in the
// shims without a shell hook, instead of failing like Use does on platforms
// that require the hook (#809).
func TestUseGlobalWithoutHook(t *testing.T) {
	s := installedSDK(t)
	t.Setenv(env.HookFlag, "") // no hook in this shell
	mustInstall(t, s, "2.0.0")
	capturePterm(t)

	if err := s.UseGlobalWithoutHook("2.0.0"); err != nil {
		t.Fatalf("UseGlobalWithoutHook(2.0.0): %v", err)
	}

	global, err := s.envContext.LoadVfoxTomlByScope(env.Global)
	if err != nil {
		t.Fatal(err)
	}
	if v, ok := global.GetToolVersion(s.Name); !ok || v != "2.0.0" {
		t.Errorf("global config asks for %s = %q, %v; want 2.0.0, true", s.Name, v, ok)
	}
	if cur := s.Current(); cur != "2.0.0" {
		t.Errorf("Current() = %q, want 2.0.0", cur)
	}

	// The shim of the global scope points at the version.
	link := filepath.Join(s.envContext.PathMeta.Working.GlobalSdkDir, s.plugin.Metadata.Name)
	target, err := env.ReadDirSymlink(link)
	if err != nil {
		t.Fatalf("the global shim %s: %v", link, err)
	}
	if want := s.mainRuntimePath("2.0.0"); target != want {
		t.Errorf("shim %s points at %s, want %s", link, target, want)
	}

	// A version that is not installed is refused, like Use refuses it.
	err = s.UseGlobalWithoutHook("9.9.9")
	var notExists *VersionNotExistsError
	if !errors.As(err, &notExists) {
		t.Errorf("UseGlobalWithoutHook(9.9.9) = %v, want a VersionNotExistsError", err)
	}
}
