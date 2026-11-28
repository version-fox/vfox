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
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/version-fox/vfox/internal/env"
)

// installRootRewrite records the roots passed to RewriteInstallRootInRegistry.
type installRootRewrite struct {
	calls            int
	oldRoot, newRoot string
	err              error
}

func stubInstallRootRewrite(t *testing.T) *installRootRewrite {
	t.Helper()
	rec := &installRootRewrite{}
	previous := env.RewriteInstallRootInRegistry
	env.RewriteInstallRootInRegistry = func(oldRoot, newRoot string) error {
		rec.calls++
		rec.oldRoot, rec.newRoot = oldRoot, newRoot
		return rec.err
	}
	t.Cleanup(func() { env.RewriteInstallRootInRegistry = previous })
	return rec
}

func sameLinkTarget(got, want string) bool {
	clean := func(p string) string {
		p = strings.TrimPrefix(p, `\\?\`)
		p = strings.TrimPrefix(p, `\??\`)
		return filepath.Clean(p)
	}
	got, want = clean(got), clean(want)
	if got == want {
		return true
	}
	return runtime.GOOS == "windows" && strings.EqualFold(got, want)
}

func linkTarget(t *testing.T, link string) string {
	t.Helper()
	got, err := env.ReadDirSymlink(link)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

// TestMigrateMovesInstallRoot moves the default install root, records it, and
// points an existing global link at the moved runtime.
func TestMigrateMovesInstallRoot(t *testing.T) {
	f := newCLIFixtureWithoutRegistryGuard(t, false)
	rec := stubInstallRootRewrite(t)
	runtime := f.installOnDisk(t, "alpha", "1.0.0", "alphatool")
	link := filepath.Join(f.userHome, "sdks", "alpha")
	must(t, os.MkdirAll(filepath.Dir(link), 0755))
	linkDirectory(t, runtime, link)

	dest := filepath.Join(f.root, "sdks-store")
	f.run(t, "migrate", dest).mustSucceed(t, "migrate")

	moved := filepath.Join(dest, "alpha", "v-1.0.0", "alpha-1.0.0")
	if !exists(filepath.Join(moved, "bin")) {
		t.Fatalf("bin missing at %s", moved)
	}
	if exists(filepath.Join(f.shared, "cache", "alpha")) {
		t.Fatal("install still under the old root")
	}
	if !exists(filepath.Join(f.shared, "plugin", "alpha", "main.lua")) {
		t.Fatal("plugin was moved")
	}
	if got := strings.TrimSpace(f.run(t, "config", "storage.sdkPath").mustSucceed(t, "config storage.sdkPath").stdout); got != dest {
		t.Fatalf("storage.sdkPath = %q, want %q", got, dest)
	}
	if got := linkTarget(t, link); !sameLinkTarget(got, moved) {
		t.Fatalf("global link -> %q, want %q", got, moved)
	}
	oldRoot := filepath.Join(f.shared, "cache")
	if rec.calls != 1 || rec.oldRoot != oldRoot || rec.newRoot != dest {
		t.Fatalf("registry rewrite %q -> %q (%d calls), want %q -> %q once", rec.oldRoot, rec.newRoot, rec.calls, oldRoot, dest)
	}
}

// A custom install root is what moves, and the user's other settings stay.
func TestMigrateCustomRootKeepsOtherSettings(t *testing.T) {
	f := newCLIFixtureWithoutRegistryGuard(t, false)
	stubInstallRootRewrite(t)
	custom := filepath.Join(f.root, "custom-sdks")
	main := filepath.Join(custom, "alpha", "v-1.0.0", "alpha-1.0.0")
	must(t, os.MkdirAll(filepath.Join(main, "bin"), 0755))
	writeCommand(t, filepath.Join(main, "bin"), "alphatool")
	f.writeUserConfig(t, "# keep me\npath:\n  mode: legacy\nstorage:\n  sdkPath: "+strconv.Quote(custom)+"\n")

	dest := filepath.Join(f.root, "other-sdks")
	f.run(t, "migrate", dest).mustSucceed(t, "migrate")

	if !exists(filepath.Join(dest, "alpha", "v-1.0.0", "alpha-1.0.0", "bin")) {
		t.Fatal("install was not moved out of the custom root")
	}
	if exists(filepath.Join(custom, "alpha")) {
		t.Fatal("custom root still holds the install")
	}
	out := f.run(t, "list", "alpha").mustSucceed(t, "list alpha").output()
	mustContain(t, "list alpha", out, "1.0.0")
	user := readFile(t, filepath.Join(f.userHome, "config.yaml"))
	mustContain(t, "user config", user, "# keep me", "legacy", "other-sdks")
	mustNotContain(t, "user config", user, "custom-sdks")
	if got := strings.TrimSpace(f.run(t, "config", "storage.sdkPath").mustSucceed(t, "config").stdout); got != dest {
		t.Fatalf("storage.sdkPath = %q, want %q", got, dest)
	}
}

func TestMigrateEmptyInstallRoot(t *testing.T) {
	f := newCLIFixtureWithoutRegistryGuard(t, false)
	stubInstallRootRewrite(t)
	dest := filepath.Join(f.root, "empty-dest")

	f.run(t, "migrate", dest).mustSucceed(t, "migrate empty root")

	info, err := os.Stat(dest)
	if err != nil || !info.IsDir() {
		t.Fatalf("destination: %v", err)
	}
	if got := strings.TrimSpace(f.run(t, "config", "storage.sdkPath").mustSucceed(t, "config").stdout); got != dest {
		t.Fatalf("storage.sdkPath = %q, want %q", got, dest)
	}
}

// A vfox link version is a symlink inside the install tree and stays one.
func TestMigrateKeepsExternalLink(t *testing.T) {
	f := newCLIFixtureWithoutRegistryGuard(t, false)
	stubInstallRootRewrite(t)
	external := f.external(t, "alphatool")
	link := filepath.Join(f.shared, "cache", "alpha", "v-hand", "alpha-hand")
	must(t, os.MkdirAll(filepath.Dir(link), 0755))
	linkDirectory(t, external, link)

	dest := filepath.Join(f.root, "linked-store")
	f.run(t, "migrate", dest).mustSucceed(t, "migrate")

	moved := filepath.Join(dest, "alpha", "v-hand", "alpha-hand")
	info, err := os.Lstat(moved)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSymlink == 0 && !env.IsDirSymlink(moved) {
		t.Fatalf("moved runtime is not a link: mode %v", info.Mode())
	}
	if got := linkTarget(t, moved); !sameLinkTarget(got, external) {
		t.Fatalf("link -> %q, want %q", got, external)
	}
	if !exists(filepath.Join(external, "bin", commandFile("alphatool"))) {
		t.Fatal("external target was removed")
	}
}

func TestMigrateRetargetsScopeLinks(t *testing.T) {
	f := newCLIFixtureWithoutRegistryGuard(t, false)
	stubInstallRootRewrite(t)
	runtime := f.installOnDisk(t, "alpha", "1.0.0", "alphatool")
	moved := filepath.Join(f.root, "scope-store", "alpha", "v-1.0.0", "alpha-1.0.0")

	session := filepath.Join(f.userHome, "tmp", "test-session")
	must(t, os.MkdirAll(session, 0755))
	inside := filepath.Join(session, "alpha")
	linkDirectory(t, runtime, inside)
	outsideDir := filepath.Join(f.root, "outside")
	must(t, os.MkdirAll(outsideDir, 0755))
	outside := filepath.Join(session, "other")
	linkDirectory(t, outsideDir, outside)
	outsideBefore := linkTarget(t, outside)

	project := filepath.Join(f.work, ".vfox", "sdks", "alpha")
	must(t, os.MkdirAll(filepath.Dir(project), 0755))
	linkDirectory(t, runtime, project)

	f.run(t, "migrate", filepath.Join(f.root, "scope-store")).mustSucceed(t, "migrate")

	if got := linkTarget(t, inside); !sameLinkTarget(got, moved) {
		t.Fatalf("session link -> %q, want %q", got, moved)
	}
	if got := linkTarget(t, outside); got != outsideBefore {
		t.Fatalf("outside link -> %q, want it left as %q", got, outsideBefore)
	}
	if got := linkTarget(t, project); !sameLinkTarget(got, moved) {
		t.Fatalf("project link -> %q, want %q", got, moved)
	}
}

func TestMigrateRejectsUnsafeDestinations(t *testing.T) {
	f := newCLIFixtureWithoutRegistryGuard(t, false)
	rec := stubInstallRootRewrite(t)
	runtime := f.installOnDisk(t, "alpha", "1.0.0", "alphatool")
	marker := filepath.Join(runtime, "MARKER")
	must(t, os.WriteFile(marker, []byte("stay"), 0644))
	cfg := "# keep\npath:\n  mode: legacy\n"
	f.writeUserConfig(t, cfg)
	configPath := filepath.Join(f.userHome, "config.yaml")
	source := filepath.Join(f.shared, "cache")

	unchanged := func(t *testing.T) {
		t.Helper()
		if readFile(t, marker) != "stay" {
			t.Fatal("source tree changed")
		}
		if readFile(t, configPath) != cfg {
			t.Fatalf("config changed:\n%s", readFile(t, configPath))
		}
	}

	t.Run("same path", func(t *testing.T) {
		res := f.run(t, "migrate", source).mustFail(t, "same path")
		mustContain(t, "same path", res.output(), "already the install root")
		unchanged(t)
	})
	t.Run("destination inside source", func(t *testing.T) {
		nested := filepath.Join(source, "nested")
		res := f.run(t, "migrate", nested).mustFail(t, "destination inside source")
		mustContain(t, "destination inside source", res.output(), "inside the install root")
		unchanged(t)
		if exists(nested) {
			t.Fatal("created a directory inside the install root")
		}
	})
	t.Run("source inside destination", func(t *testing.T) {
		res := f.run(t, "migrate", filepath.Dir(source)).mustFail(t, "source inside destination")
		mustContain(t, "source inside destination", res.output(), "is inside")
		unchanged(t)
	})
	t.Run("destination is a file", func(t *testing.T) {
		file := filepath.Join(f.root, "not-a-dir")
		must(t, os.WriteFile(file, []byte("file"), 0644))
		res := f.run(t, "migrate", file).mustFail(t, "destination file")
		mustContain(t, "destination file", res.output(), "not a directory")
		unchanged(t)
		if readFile(t, file) != "file" {
			t.Fatal("destination file changed")
		}
	})
	t.Run("name already exists", func(t *testing.T) {
		dest := filepath.Join(f.root, "taken")
		must(t, os.MkdirAll(filepath.Join(dest, "alpha"), 0755))
		res := f.run(t, "migrate", dest).mustFail(t, "name collision")
		mustContain(t, "name collision", res.output(), "already exists")
		unchanged(t)
		if exists(filepath.Join(dest, "alpha", "v-1.0.0")) {
			t.Fatal("source entry was moved into the destination")
		}
	})
	if rec.calls != 0 {
		t.Fatalf("registry rewrite ran %d times", rec.calls)
	}
}

func TestMigrateRejectsUnwritableDestination(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("POSIX chmod 0555 does not block owner writes on Windows or root")
	}
	f := newCLIFixtureWithoutRegistryGuard(t, false)
	stubInstallRootRewrite(t)
	runtime := f.installOnDisk(t, "alpha", "1.0.0", "alphatool")
	cfg := "# keep\n"
	f.writeUserConfig(t, cfg)
	dest := filepath.Join(f.root, "ro-dest")
	must(t, os.MkdirAll(dest, 0755))
	must(t, os.Chmod(dest, 0555))
	t.Cleanup(func() { _ = os.Chmod(dest, 0755) })

	res := f.run(t, "migrate", dest).mustFail(t, "unwritable destination")
	mustContain(t, "unwritable destination", res.output(), "permission denied")
	if !exists(runtime) {
		t.Fatal("source install was moved")
	}
	if readFile(t, filepath.Join(f.userHome, "config.yaml")) != cfg {
		t.Fatal("config changed")
	}
}

func TestMigrateSucceedsWhenRegistryRewriteFails(t *testing.T) {
	f := newCLIFixtureWithoutRegistryGuard(t, false)
	rec := stubInstallRootRewrite(t)
	rec.err = os.ErrPermission
	f.installOnDisk(t, "alpha", "1.0.0", "alphatool")
	dest := filepath.Join(f.root, "despite-registry")

	res := f.run(t, "migrate", dest).mustSucceed(t, "migrate despite registry")
	mustContain(t, "registry warning", res.output(), "permission denied")
	if !exists(filepath.Join(dest, "alpha", "v-1.0.0", "alpha-1.0.0", "bin")) {
		t.Fatal("install was not moved")
	}
	if rec.calls != 1 {
		t.Fatalf("registry rewrite calls = %d, want 1", rec.calls)
	}
}
