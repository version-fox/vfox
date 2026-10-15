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

package pathmeta

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewPathMeta_Shims(t *testing.T) {
	tmpHome := t.TempDir()
	tmpWork := t.TempDir()

	meta, err := NewPathMeta(tmpHome, "", tmpWork, 12345)
	if err != nil {
		t.Fatalf("NewPathMeta failed: %v", err)
	}

	// Global shim dir is stable under the user home and created eagerly.
	wantGlobal := filepath.Join(meta.User.Home, "shims")
	if meta.Working.GlobalShimDir != wantGlobal {
		t.Errorf("GlobalShimDir = %q, want %q", meta.Working.GlobalShimDir, wantGlobal)
	}
	if info, err := os.Stat(wantGlobal); err != nil || !info.IsDir() {
		t.Errorf("expected global shim dir to exist at %q, err: %v", wantGlobal, err)
	}

	// Project shim dir is stable per project directory.
	wantProject := filepath.Join(tmpWork, ".vfox", "shims")
	if meta.Working.ProjectShimDir != wantProject {
		t.Errorf("ProjectShimDir = %q, want %q", meta.Working.ProjectShimDir, wantProject)
	}

	// Session shim dir lives under the session temp dir, away from session configs.
	if meta.Working.SessionShimDir != filepath.Join(meta.Working.SessionSdkDir, "shims") {
		t.Errorf("SessionShimDir = %q, want under SessionSdkDir %q",
			meta.Working.SessionShimDir, meta.Working.SessionSdkDir)
	}

	// All shim dirs must be recognized as vfox-managed so SplitSystemPaths
	// treats them like other vfox paths.
	for _, dir := range []string{
		filepath.Join(meta.Working.GlobalShimDir, "rimraf"),
		filepath.Join(meta.Working.ProjectShimDir, "rimraf"),
		filepath.Join(meta.Working.SessionShimDir, "rimraf"),
	} {
		if !IsVfoxRelatedPath(dir) {
			t.Errorf("expected %q to be vfox-related", dir)
		}
	}
}
