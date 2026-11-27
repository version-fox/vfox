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
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"syscall"

	"github.com/pterm/pterm"

	"github.com/version-fox/vfox/internal/config"
	"github.com/version-fox/vfox/internal/env"
	"github.com/version-fox/vfox/internal/pathmeta"
	"github.com/version-fox/vfox/internal/shared/util"
)

// renameEntry moves one path. Tests replace it to simulate a cross-device rename.
var renameEntry = os.Rename

// removeAll deletes a tree. Tests replace it to fail cleanup after a copy.
var removeAll = os.RemoveAll

// createDirSymlink creates a directory symlink or junction. Tests replace it
// to fail while a scope link is being retargeted.
var createDirSymlink = env.CreateDirSymlink

// linkChange is a directory symlink retargeted during a migration, remembered
// so it can be pointed back at its old target when a later step fails.
type linkChange struct {
	path   string
	target string
}

// MigrateSdkPath moves every entry of the current install root to dest and
// records dest as storage.sdkPath in the user config. Plugins, config.yaml, and
// the sdks link directory stay where they are. A nested cache/<sdk> tree left
// by the v1.0.2–v1.0.7 layout moves with the other children, which is where
// sdk.NewSdk still looks for it. Directory links that point into the old root
// are pointed at dest. On failure before the config is saved, moved entries and
// retargeted links are put back and the config is left unchanged.
func (m *Manager) MigrateSdkPath(dest string) error {
	meta := m.RuntimeEnvContext.PathMeta
	source, err := absolutePath(meta.Shared.Installs)
	if err != nil {
		return err
	}
	dest, err = absolutePath(dest)
	if err != nil {
		return err
	}
	if shared, err := installRootIsShared(meta); err != nil {
		return err
	} else if shared {
		return fmt.Errorf("the install root %s belongs to the shared VFOX_HOME %s; "+
			"moving it would remove the SDKs of every other user of that home, "+
			"so move it by hand and update storage.sdkPath in %s",
			source, meta.Shared.Root, meta.Shared.Config)
	}

	// Refuse before creating dest, so a destination inside the install root does
	// not become a new child of the tree that is supposed to stay put.
	if info, err := os.Stat(dest); err == nil && !info.IsDir() {
		return fmt.Errorf("%s is not a directory", dest)
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}
	if sameFile(source, dest) {
		return fmt.Errorf("%s is already the install root", dest)
	}
	if pathInside(source, dest) {
		return fmt.Errorf("%s is inside the install root %s", dest, source)
	}
	if pathInside(dest, source) {
		return fmt.Errorf("install root %s is inside %s", source, dest)
	}

	if err := os.MkdirAll(dest, pathmeta.ReadWriteAuth); err != nil {
		return err
	}
	if err := (&config.Storage{SdkPath: dest}).Validate(); err != nil {
		return err
	}

	names, err := installRootEntries(source)
	if err != nil {
		return err
	}
	for _, name := range names {
		target := filepath.Join(dest, name)
		if _, err := os.Lstat(target); err == nil {
			return fmt.Errorf("%s already exists", target)
		} else if !os.IsNotExist(err) {
			return err
		}
	}

	var moved []string
	for _, name := range names {
		if err := moveEntry(filepath.Join(source, name), filepath.Join(dest, name)); err != nil {
			return undoMigration(nil, moved, source, dest, err)
		}
		moved = append(moved, name)
	}

	changes, err := retargetInstallLinks(meta, source, dest)
	if err != nil {
		return undoMigration(changes, moved, source, dest, err)
	}

	file, err := config.LoadFile(meta.User.Config)
	if err != nil {
		return undoMigration(changes, moved, source, dest, err)
	}
	if err := file.Set("storage.sdkPath", dest); err != nil {
		return undoMigration(changes, moved, source, dest, err)
	}
	if err := file.Save(); err != nil {
		return undoMigration(changes, moved, source, dest, err)
	}

	// The files, links, and config are the source of truth. A registry that
	// cannot be updated is reported and does not undo the move, matching apply.
	if err := env.RewriteInstallRootInRegistry(source, dest); err != nil {
		pterm.Warning.Printf("Failed to rewrite the install root in the registry: %v\n", err)
	}
	return nil
}

// installRootIsShared reports whether the install root is shared with other
// users: VFOX_HOME names a root other than this user's vfox home, and this
// user's own config does not choose the install root. The migration only
// writes this user's config, so other users would keep resolving the old root
// after its entries are gone.
func installRootIsShared(meta *pathmeta.PathMeta) (bool, error) {
	if meta.Shared.Root == "" || sameFile(meta.Shared.Root, meta.User.Home) {
		return false, nil
	}
	if !util.FileExists(meta.User.Config) {
		return true, nil
	}
	c, err := config.NewConfigWithPath(meta.User.Config)
	if err != nil {
		return false, err
	}
	return c.Storage == nil || c.Storage.SdkPath == "", nil
}

func absolutePath(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	return filepath.Clean(abs), nil
}

func sameFile(a, b string) bool {
	if filepath.Clean(a) == filepath.Clean(b) {
		return true
	}
	ai, err1 := os.Stat(a)
	bi, err2 := os.Stat(b)
	if err1 != nil || err2 != nil {
		return false
	}
	return os.SameFile(ai, bi)
}

// pathInside reports whether child is strictly inside parent. A case-insensitive
// volume, such as the default macOS disk, treats CACHE/new as inside cache;
// filepath.Rel alone does not, and MkdirAll would create the destination there.
func pathInside(parent, child string) bool {
	parent = filepath.Clean(parent)
	child = filepath.Clean(child)
	if relInside(parent, child) {
		return true
	}
	if !volumeFoldsCase(parent) {
		return false
	}
	return relInside(strings.ToLower(parent), strings.ToLower(child))
}

func relInside(parent, child string) bool {
	rel, err := filepath.Rel(parent, child)
	if err != nil || rel == "." {
		return false
	}
	return filepath.IsLocal(rel)
}

// volumeFoldsCase reports whether path's volume compares file names without
// regard to case. Windows always does. Darwin usually does; a case-sensitive
// volume does not, which a stat of a flipped name tells apart.
func volumeFoldsCase(path string) bool {
	switch runtime.GOOS {
	case "windows":
		return true
	case "darwin":
		return darwinFoldsCase(filepath.Clean(path))
	default:
		return false
	}
}

func darwinFoldsCase(path string) bool {
	for {
		base := filepath.Base(path)
		flipped := flipASCIICase(base)
		if flipped != base {
			info, err := os.Lstat(path)
			if err == nil {
				alt, errAlt := os.Lstat(filepath.Join(filepath.Dir(path), flipped))
				if errAlt == nil {
					return os.SameFile(info, alt)
				}
				if os.IsNotExist(errAlt) {
					return false
				}
			}
		}
		next := filepath.Dir(path)
		if next == path {
			return true
		}
		path = next
	}
}

func flipASCIICase(s string) string {
	b := []byte(s)
	for i, c := range b {
		switch {
		case 'a' <= c && c <= 'z':
			b[i] = c - 'a' + 'A'
			return string(b)
		case 'A' <= c && c <= 'Z':
			b[i] = c - 'A' + 'a'
			return string(b)
		}
	}
	return s
}

// installRootEntries lists the names in source. A missing root has nothing to
// move; the caller still records dest so later installs use it.
func installRootEntries(source string) ([]string, error) {
	entries, err := os.ReadDir(source)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	names := make([]string, len(entries))
	for i, entry := range entries {
		names[i] = entry.Name()
	}
	sort.Strings(names)
	return names, nil
}

func undoMigration(changes []linkChange, moved []string, source, dest string, cause error) error {
	var errs []error
	if err := rollbackMoves(moved, source, dest); err != nil {
		errs = append(errs, err)
	}
	if err := restoreLinks(changes); err != nil {
		errs = append(errs, err)
	}
	if len(errs) == 0 {
		return cause
	}
	return errors.Join(append([]error{cause}, errs...)...)
}

func rollbackMoves(moved []string, source, dest string) error {
	var errs []error
	for i := len(moved) - 1; i >= 0; i-- {
		name := moved[i]
		if err := moveEntry(filepath.Join(dest, name), filepath.Join(source, name)); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func restoreLinks(changes []linkChange) error {
	var errs []error
	for i := len(changes) - 1; i >= 0; i-- {
		change := changes[i]
		if err := createDirSymlink(change.target, change.path); err != nil {
			errs = append(errs, fmt.Errorf("restore %s -> %s: %w", change.path, change.target, err))
		}
	}
	return errors.Join(errs...)
}

// moveEntry renames src to dst. A cross-device rename is copied instead, and a
// symlink or junction is recreated with the same target: a version added with
// vfox link must stay a link to that external path, not become a copy of it.
func moveEntry(src, dst string) error {
	err := renameEntry(src, dst)
	if err == nil {
		return nil
	}
	if !isCrossDeviceRenameError(err) {
		return err
	}
	if err := copyEntry(src, dst); err != nil {
		if rmErr := removeAll(dst); rmErr != nil {
			return fmt.Errorf("copy %s to %s: %w; partial copy remains at %s: %w", src, dst, err, dst, rmErr)
		}
		return err
	}
	if err := removeAll(src); err != nil {
		// dst is a complete copy. removeAll may already have deleted source
		// files, so removing dst would drop the only remaining copy.
		return fmt.Errorf("copied %s to %s but could not remove the source: %w", src, dst, err)
	}
	return nil
}

// isCrossDeviceRenameError matches the failures util treats as a cross-device rename.
func isCrossDeviceRenameError(err error) bool {
	var linkErr *os.LinkError
	if errors.As(err, &linkErr) {
		err = linkErr.Err
	}
	if errors.Is(err, syscall.EXDEV) {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "cross-device link") ||
		strings.Contains(msg, "different disk drive") ||
		strings.Contains(msg, "not same device")
}

// copyEntry copies src to dst without following links. Lstat, not Stat: Stat
// would copy the directory a link points at and drop the link.
func copyEntry(src, dst string) error {
	info, err := os.Lstat(src)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(src)
		if err != nil {
			return err
		}
		return os.Symlink(target, dst)
	}
	// A Windows junction has no ModeSymlink. Readlink recognizes it; a real
	// file or directory fails here and is copied below.
	if target, err := env.ReadDirSymlink(src); err == nil {
		return env.CreateDirSymlink(target, dst)
	}
	if info.IsDir() {
		if err := os.MkdirAll(dst, info.Mode().Perm()); err != nil {
			return err
		}
		entries, err := os.ReadDir(src)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			if err := copyEntry(filepath.Join(src, entry.Name()), filepath.Join(dst, entry.Name())); err != nil {
				return err
			}
		}
		return nil
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("cannot copy %s", src)
	}
	if err := util.CopyFile(src, dst); err != nil {
		return err
	}
	return util.ChangeModeIfNot(dst, info.Mode())
}

// retargetInstallLinks points directory symlinks at dest when they point into
// oldRoot: one level of the global and project link directories, and two levels
// of the user temp directory (a session directory, then its links).
func retargetInstallLinks(meta *pathmeta.PathMeta, oldRoot, newRoot string) ([]linkChange, error) {
	var changes []linkChange
	for _, dir := range []string{meta.Working.GlobalSdkDir, meta.Working.ProjectSdkDir} {
		got, err := retargetLinks(dir, oldRoot, newRoot, 1)
		changes = append(changes, got...)
		if err != nil {
			return changes, err
		}
	}
	got, err := retargetLinks(meta.User.Temp, oldRoot, newRoot, 2)
	changes = append(changes, got...)
	return changes, err
}

func retargetLinks(dir, oldRoot, newRoot string, levels int) ([]linkChange, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var changes []linkChange
	for _, entry := range entries {
		path := filepath.Join(dir, entry.Name())
		if levels > 1 {
			info, lerr := os.Lstat(path)
			if lerr != nil {
				return changes, lerr
			}
			if info.IsDir() && info.Mode()&os.ModeSymlink == 0 && !env.IsDirSymlink(path) {
				nested, err := retargetLinks(path, oldRoot, newRoot, levels-1)
				changes = append(changes, nested...)
				if err != nil {
					return changes, err
				}
			}
			continue
		}
		change, ok, err := retargetOne(path, oldRoot, newRoot)
		if ok {
			changes = append(changes, change)
		}
		if err != nil {
			return changes, err
		}
	}
	return changes, nil
}

func retargetOne(path, oldRoot, newRoot string) (linkChange, bool, error) {
	if !env.IsDirSymlink(path) {
		return linkChange{}, false, nil
	}
	target, err := env.ReadDirSymlink(path)
	if err != nil {
		return linkChange{}, false, err
	}
	abs := cleanLinkTarget(path, target)
	rel, err := filepath.Rel(oldRoot, abs)
	if err != nil || !filepath.IsLocal(rel) {
		return linkChange{}, false, nil
	}
	newTarget := newRoot
	if rel != "." {
		newTarget = filepath.Join(newRoot, rel)
	}
	if err := createDirSymlink(newTarget, path); err != nil {
		// CreateDirSymlink removes the old link before creating the new one.
		// Remember a link that is gone so undoMigration puts the previous
		// target back and reports it when that restore fails.
		if _, statErr := os.Lstat(path); errors.Is(statErr, fs.ErrNotExist) {
			return linkChange{path: path, target: target}, true, err
		}
		return linkChange{}, false, err
	}
	return linkChange{path: path, target: target}, true, nil
}

func cleanLinkTarget(link, target string) string {
	target = strings.TrimPrefix(target, `\\?\`)
	target = strings.TrimPrefix(target, `\??\`)
	if !filepath.IsAbs(target) {
		target = filepath.Join(filepath.Dir(link), target)
	}
	return filepath.Clean(target)
}
