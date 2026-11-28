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

package internal

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/version-fox/vfox/internal/config"
	"github.com/version-fox/vfox/internal/env"
	"github.com/version-fox/vfox/internal/pathmeta"
)

func migrateManager(t *testing.T, installs string) *Manager {
	t.Helper()
	root := filepath.Dir(installs)
	user := filepath.Join(root, "home", ".vfox")
	if err := os.MkdirAll(user, 0755); err != nil {
		t.Fatal(err)
	}
	return &Manager{
		RuntimeEnvContext: &env.RuntimeEnvContext{
			PathMeta: &pathmeta.PathMeta{
				User: pathmeta.UserPaths{
					Home:   user,
					Temp:   filepath.Join(user, "tmp"),
					Config: filepath.Join(user, "config.yaml"),
				},
				Shared: pathmeta.SharedPaths{Installs: installs},
				Working: pathmeta.WorkingPaths{
					GlobalSdkDir:  filepath.Join(user, "sdks"),
					ProjectSdkDir: filepath.Join(root, "project", ".vfox", "sdks"),
				},
			},
		},
	}
}

func silenceInstallRootRewrite(t *testing.T) *int {
	t.Helper()
	calls := 0
	old := env.RewriteInstallRootInRegistry
	env.RewriteInstallRootInRegistry = func(string, string) error {
		calls++
		return nil
	}
	t.Cleanup(func() { env.RewriteInstallRootInRegistry = old })
	return &calls
}

func TestMigrateMissingSourceUpdatesConfig(t *testing.T) {
	root := t.TempDir()
	installs := filepath.Join(root, "missing-cache")
	dest := filepath.Join(root, "dest")
	var gotOld, gotNew string
	calls := 0
	old := env.RewriteInstallRootInRegistry
	env.RewriteInstallRootInRegistry = func(oldRoot, newRoot string) error {
		calls++
		gotOld, gotNew = oldRoot, newRoot
		return nil
	}
	t.Cleanup(func() { env.RewriteInstallRootInRegistry = old })

	m := migrateManager(t, installs)
	if err := m.MigrateSdkPath(dest); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(dest)
	if err != nil || !info.IsDir() {
		t.Fatalf("destination: %v", err)
	}
	content, err := os.ReadFile(m.RuntimeEnvContext.PathMeta.User.Config)
	if err != nil {
		t.Fatal(err)
	}
	var cfg config.Config
	if err := yaml.Unmarshal(content, &cfg); err != nil {
		t.Fatal(err)
	}
	wantDest, err := filepath.Abs(dest)
	if err != nil {
		t.Fatal(err)
	}
	wantSource, err := filepath.Abs(installs)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Storage == nil || cfg.Storage.SdkPath != wantDest {
		t.Fatalf("storage.sdkPath = %+v, want %s", cfg.Storage, wantDest)
	}
	if calls != 1 || gotOld != wantSource || gotNew != wantDest {
		t.Fatalf("registry rewrite %q -> %q (%d), want %q -> %q", gotOld, gotNew, calls, wantSource, wantDest)
	}
}

func TestMigrateCopiesExternalLinkAcrossDevices(t *testing.T) {
	root := t.TempDir()
	installs := filepath.Join(root, "cache")
	external := filepath.Join(root, "external")
	if err := os.MkdirAll(filepath.Join(external, "bin"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(external, "bin", "tool"), []byte("tool"), 0755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(installs, "alpha", "v-hand", "alpha-hand")
	if err := os.MkdirAll(filepath.Dir(link), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(filepath.Dir(link), "note.txt"), []byte("keep"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := env.CreateDirSymlink(external, link); err != nil {
		t.Fatal(err)
	}

	previous := renameEntry
	renameEntry = func(oldPath, newPath string) error {
		return &os.LinkError{Op: "rename", Old: oldPath, New: newPath, Err: syscall.EXDEV}
	}
	t.Cleanup(func() { renameEntry = previous })
	silenceInstallRootRewrite(t)

	dest := filepath.Join(root, "moved")
	if err := migrateManager(t, installs).MigrateSdkPath(dest); err != nil {
		t.Fatal(err)
	}
	moved := filepath.Join(dest, "alpha", "v-hand", "alpha-hand")
	info, err := os.Lstat(moved)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSymlink == 0 && !env.IsDirSymlink(moved) {
		t.Fatalf("linked runtime copied as mode %v", info.Mode())
	}
	got, err := env.ReadDirSymlink(moved)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Clean(got) != filepath.Clean(external) && (runtime.GOOS != "windows" || !strings.EqualFold(filepath.Clean(got), filepath.Clean(external))) {
		t.Fatalf("link -> %q, want %q", got, external)
	}
	note, err := os.ReadFile(filepath.Join(dest, "alpha", "v-hand", "note.txt"))
	if err != nil || string(note) != "keep" {
		t.Fatalf("copied file %q, %v", note, err)
	}
	if _, err := os.Lstat(filepath.Join(installs, "alpha")); !os.IsNotExist(err) {
		t.Fatalf("source entry still present: %v", err)
	}
	if _, err := os.Stat(filepath.Join(external, "bin", "tool")); err != nil {
		t.Fatalf("external target: %v", err)
	}
}

func TestMigrateRollsBackAFailedMove(t *testing.T) {
	root := t.TempDir()
	installs := filepath.Join(root, "cache")
	for _, name := range []string{"a", "b"} {
		dir := filepath.Join(installs, name)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "keep.txt"), []byte(name), 0644); err != nil {
			t.Fatal(err)
		}
	}
	m := migrateManager(t, installs)
	cfg := "# keep\npath:\n  mode: legacy\n"
	if err := os.WriteFile(m.RuntimeEnvContext.PathMeta.User.Config, []byte(cfg), 0644); err != nil {
		t.Fatal(err)
	}
	previous := renameEntry
	renameEntry = func(src, dst string) error {
		if filepath.Base(src) == "b" {
			return os.ErrPermission
		}
		return os.Rename(src, dst)
	}
	t.Cleanup(func() { renameEntry = previous })
	calls := silenceInstallRootRewrite(t)

	dest := filepath.Join(root, "moved")
	if err := m.MigrateSdkPath(dest); err == nil {
		t.Fatal("expected the move to fail")
	}
	for _, name := range []string{"a", "b"} {
		content, err := os.ReadFile(filepath.Join(installs, name, "keep.txt"))
		if err != nil || string(content) != name {
			t.Fatalf("source %s: %q, %v", name, content, err)
		}
		if _, err := os.Lstat(filepath.Join(dest, name)); !os.IsNotExist(err) {
			t.Fatalf("destination still has %s: %v", name, err)
		}
	}
	got, err := os.ReadFile(m.RuntimeEnvContext.PathMeta.User.Config)
	if err != nil || string(got) != cfg {
		t.Fatalf("config = %q, %v", got, err)
	}
	if *calls != 0 {
		t.Fatalf("registry rewrite ran %d times", *calls)
	}
}

func TestMigrateRefusesCaseVariantInsideInstallRoot(t *testing.T) {
	root := t.TempDir()
	installs := filepath.Join(root, "cache")
	if err := os.MkdirAll(installs, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(installs, "keep.txt"), []byte("stay"), 0644); err != nil {
		t.Fatal(err)
	}
	silenceInstallRootRewrite(t)
	dest := filepath.Join(root, "CACHE", "new")
	err := migrateManager(t, installs).MigrateSdkPath(dest)
	if !volumeFoldsCase(installs) {
		if err != nil {
			t.Fatalf("case-sensitive volume: %v", err)
		}
		return
	}
	if err == nil {
		t.Fatal("expected a destination inside the install root to be refused")
	}
	content, readErr := os.ReadFile(filepath.Join(installs, "keep.txt"))
	if readErr != nil || string(content) != "stay" {
		t.Fatalf("source = %q, %v", content, readErr)
	}
	if _, statErr := os.Lstat(filepath.Join(installs, "new")); !os.IsNotExist(statErr) {
		t.Fatalf("created the destination inside the install root: %v", statErr)
	}
	outside := filepath.Join(root, "OTHER")
	if pathInside(installs, outside) {
		t.Fatalf("pathInside(%q, %q) = true", installs, outside)
	}
}

func TestMigrateKeepsCopyWhenSourceRemovalFails(t *testing.T) {
	root := t.TempDir()
	installs := filepath.Join(root, "cache")
	entry := filepath.Join(installs, "alpha")
	if err := os.MkdirAll(entry, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(entry, "keep.txt"), []byte("stay"), 0644); err != nil {
		t.Fatal(err)
	}
	m := migrateManager(t, installs)
	previousRename := renameEntry
	renameEntry = func(oldPath, newPath string) error {
		return &os.LinkError{Op: "rename", Old: oldPath, New: newPath, Err: syscall.EXDEV}
	}
	previousRemove := removeAll
	removeAll = func(path string) error {
		if path == entry {
			return fmt.Errorf("remove %s: %w", path, os.ErrPermission)
		}
		return os.RemoveAll(path)
	}
	t.Cleanup(func() {
		renameEntry = previousRename
		removeAll = previousRemove
	})
	silenceInstallRootRewrite(t)

	dest := filepath.Join(root, "moved")
	err := m.MigrateSdkPath(dest)
	if err == nil {
		t.Fatal("expected source removal to fail")
	}
	if !strings.Contains(err.Error(), entry) || !strings.Contains(err.Error(), filepath.Join(dest, "alpha")) {
		t.Fatalf("error does not name the source and the surviving copy: %v", err)
	}
	for _, path := range []string{filepath.Join(entry, "keep.txt"), filepath.Join(dest, "alpha", "keep.txt")} {
		content, readErr := os.ReadFile(path)
		if readErr != nil || string(content) != "stay" {
			t.Fatalf("%s = %q, %v", path, content, readErr)
		}
	}
	if _, statErr := os.Lstat(m.RuntimeEnvContext.PathMeta.User.Config); !os.IsNotExist(statErr) {
		t.Fatalf("config was written: %v", statErr)
	}
}

func TestMigrateRestoresLinkRemovedDuringRetarget(t *testing.T) {
	root := t.TempDir()
	installs := filepath.Join(root, "cache")
	version := filepath.Join(installs, "alpha")
	if err := os.MkdirAll(version, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(version, "keep.txt"), []byte("stay"), 0644); err != nil {
		t.Fatal(err)
	}
	m := migrateManager(t, installs)
	linkDir := m.RuntimeEnvContext.PathMeta.Working.GlobalSdkDir
	if err := os.MkdirAll(linkDir, 0755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(linkDir, "alpha")
	if err := env.CreateDirSymlink(version, link); err != nil {
		t.Fatal(err)
	}

	calls := 0
	previous := createDirSymlink
	createDirSymlink = func(target, linkPath string) error {
		calls++
		if calls == 1 {
			if err := env.RemoveDirSymlink(linkPath); err != nil {
				return err
			}
			return errors.New("create failed")
		}
		return previous(target, linkPath)
	}
	t.Cleanup(func() { createDirSymlink = previous })
	silenceInstallRootRewrite(t)

	dest := filepath.Join(root, "moved")
	err := m.MigrateSdkPath(dest)
	if err == nil {
		t.Fatal("expected retarget to fail")
	}
	if calls < 2 {
		t.Fatalf("undo did not restore the removed link (calls=%d): %v", calls, err)
	}
	got, readErr := env.ReadDirSymlink(link)
	if readErr != nil {
		t.Fatalf("link missing after rollback: %v (%v)", readErr, err)
	}
	if filepath.Clean(got) != filepath.Clean(version) && (runtime.GOOS != "windows" || !strings.EqualFold(filepath.Clean(got), filepath.Clean(version))) {
		t.Fatalf("link -> %q, want %q", got, version)
	}
	content, readErr := os.ReadFile(filepath.Join(version, "keep.txt"))
	if readErr != nil || string(content) != "stay" {
		t.Fatalf("source = %q, %v", content, readErr)
	}
	if _, statErr := os.Lstat(filepath.Join(dest, "alpha")); !os.IsNotExist(statErr) {
		t.Fatalf("destination still has the moved entry: %v", statErr)
	}
}

func TestMigrateValidateFailureDoesNotRename(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("POSIX chmod 0555 does not block owner writes on Windows or root")
	}
	root := t.TempDir()
	installs := filepath.Join(root, "cache", "alpha")
	if err := os.MkdirAll(installs, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(installs, "keep.txt"), []byte("stay"), 0644); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(root, "ro-dest")
	if err := os.MkdirAll(dest, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(dest, 0555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dest, 0755) })

	renames := 0
	previous := renameEntry
	renameEntry = func(src, dst string) error {
		renames++
		return os.Rename(src, dst)
	}
	t.Cleanup(func() { renameEntry = previous })
	silenceInstallRootRewrite(t)

	// installs above is the sdk directory; the install root is its parent.
	m := migrateManager(t, filepath.Dir(installs))
	if err := m.MigrateSdkPath(dest); err == nil {
		t.Fatal("expected an unwritable destination to fail")
	}
	if renames != 0 {
		t.Fatalf("renamed %d entries before validation failed", renames)
	}
	content, err := os.ReadFile(filepath.Join(installs, "keep.txt"))
	if err != nil || string(content) != "stay" {
		t.Fatalf("source = %q, %v", content, err)
	}
	if _, err := os.Lstat(m.RuntimeEnvContext.PathMeta.User.Config); !os.IsNotExist(err) {
		t.Fatalf("config was written: %v", err)
	}
}

func TestMigrateRefusesInstallRootSharedThroughVfoxHome(t *testing.T) {
	root := t.TempDir()
	sharedRoot := filepath.Join(root, "shared")
	installs := filepath.Join(sharedRoot, "cache")
	if err := os.MkdirAll(filepath.Join(installs, "alpha", "v-1"), 0755); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(root, "dest")
	manager := migrateManager(t, installs)
	meta := manager.RuntimeEnvContext.PathMeta
	meta.Shared.Root = sharedRoot
	meta.Shared.Config = filepath.Join(sharedRoot, "config.yaml")
	calls := silenceInstallRootRewrite(t)

	err := manager.MigrateSdkPath(dest)
	if err == nil || !strings.Contains(err.Error(), "shared VFOX_HOME") {
		t.Fatalf("MigrateSdkPath error = %v, want shared VFOX_HOME refusal", err)
	}
	if _, err := os.Stat(filepath.Join(installs, "alpha", "v-1")); err != nil {
		t.Fatalf("shared install moved: %v", err)
	}
	if _, err := os.Stat(dest); !os.IsNotExist(err) {
		t.Fatalf("dest created: %v", err)
	}
	if *calls != 0 {
		t.Fatalf("registry rewritten %d times", *calls)
	}
}

func TestMigrateMovesUserChosenRootUnderSharedVfoxHome(t *testing.T) {
	root := t.TempDir()
	sharedRoot := filepath.Join(root, "shared")
	installs := filepath.Join(root, "mine")
	if err := os.MkdirAll(filepath.Join(installs, "alpha", "v-1"), 0755); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(root, "dest")
	manager := migrateManager(t, installs)
	meta := manager.RuntimeEnvContext.PathMeta
	meta.Shared.Root = sharedRoot
	meta.Shared.Config = filepath.Join(sharedRoot, "config.yaml")
	if err := os.WriteFile(meta.User.Config, []byte("storage:\n  sdkPath: "+installs+"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	silenceInstallRootRewrite(t)

	if err := manager.MigrateSdkPath(dest); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dest, "alpha", "v-1")); err != nil {
		t.Fatalf("user install not moved: %v", err)
	}
}
