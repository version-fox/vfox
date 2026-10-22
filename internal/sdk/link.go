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
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/pterm/pterm"

	"github.com/version-fox/vfox/internal/env"
	"github.com/version-fox/vfox/internal/pathmeta"
	"github.com/version-fox/vfox/internal/plugin"
	"github.com/version-fox/vfox/internal/shared/logger"
)

// LinkOptions tune Link.
type LinkOptions struct {
	// Force points an already linked version somewhere else. It never replaces
	// a version vfox installed.
	Force bool
}

var versionNamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._+\-]*$`)

// ValidVersionName reports whether name can serve as a version name the user
// picks: it becomes a directory name and is typed in commands.
func ValidVersionName(name string) bool {
	return versionNamePattern.MatchString(name) && name != "latest"
}

// StripVersionPrefix returns the version a user means by what they typed after
// "@". A leading "v" is the usual way to spell a version (v20.1.0 for 20.1.0),
// but a name the user picked can start with one too, such as a linked version
// called vendor. So a version that is on disk under the name as typed is left
// alone, even when it is broken, and the "v" only goes when there is none.
// s may be nil.
func StripVersionPrefix(s Sdk, typed Version) Version {
	if typed == "" || (s != nil && s.HasVersion(typed)) {
		return typed
	}
	return Version(strings.TrimPrefix(string(typed), "v"))
}

// Link registers an existing SDK directory as a version. Nothing is downloaded
// or copied: the version directory holds a link to the SDK, which stays where
// it is and stays its owner's. Uninstalling the version removes the link only.
func (b *impl) Link(version Version, target string, opts LinkOptions) (linkErr error) {
	label := b.Label(version)
	if !ValidVersionName(string(version)) {
		return fmt.Errorf("%q is not a usable version name: use letters, digits, '.', '_', '+' and '-', and not \"latest\"", version)
	}
	abs, err := filepath.Abs(target)
	if err != nil {
		return fmt.Errorf("resolve %s: %w", target, err)
	}
	info, err := os.Stat(abs)
	if err != nil {
		return fmt.Errorf("cannot link %s: %w", label, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("cannot link %s: %s is not a directory", label, abs)
	}
	if err := b.validateLinkTarget(version, abs); err != nil {
		return err
	}

	lock, err := acquireInstallLock(filepath.Join(b.InstallPath, ".v-"+string(version)+".lock"))
	if err != nil {
		return fmt.Errorf("cannot link %s: %w", label, err)
	}
	defer func() {
		if err := lock.Close(); err != nil {
			linkErr = errors.Join(linkErr, fmt.Errorf("close installation lock: %w", err))
		}
	}()
	if b.hasPendingInstall(version) {
		return fmt.Errorf("cannot link %s: it is being installed", label)
	}

	packagePath := b.packagePath(version)
	if _, err := os.Lstat(packagePath); err == nil {
		existing, linked := b.linkedTarget(version)
		switch {
		case linked && opts.Force:
			if err := b.removeLink(version); err != nil {
				return err
			}
		case linked:
			return fmt.Errorf("%s is already linked to %s; use --force to point it elsewhere", label, existing)
		default:
			return fmt.Errorf("%s is already installed; uninstall it first to link a different copy under this name", label)
		}
	}

	if err := os.MkdirAll(packagePath, pathmeta.ReadWriteAuth); err != nil {
		return fmt.Errorf("create %s: %w", packagePath, err)
	}
	linked := false
	defer func() {
		if !linked {
			if err := b.removeLink(version); err != nil {
				logger.Debugf("Failed to clean up after linking %s: %v\n", label, err)
			}
		}
	}()
	if err := env.CreateDirSymlink(abs, b.mainRuntimePath(version)); err != nil {
		return fmt.Errorf("link %s: %w", label, err)
	}

	receipt := b.newReceipt(version, OriginLinked)
	receipt.Requested = string(version)
	receipt.Target = abs
	if pkg, err := b.GetRuntimePackage(version); err == nil {
		facts, err := b.fingerprint(version, sdkInfoOf(pkg), pkg.Runtime.Path)
		if err != nil {
			logger.Debugf("No facts recorded for %s: %v\n", label, err)
		}
		receipt.Facts = facts
	}
	if err := writeReceipt(packagePath, receipt); err != nil {
		// Without a receipt the link still counts as linked: it is a link.
		logger.Debugf("Failed to write the receipt of %s: %v\n", label, err)
	}
	linked = true

	pterm.Printf("Linked %s -> %s\n", pterm.LightGreen(label), abs)
	pterm.Printf("Please use `%s` to use it.\n", pterm.LightBlue("vfox use "+label))
	return nil
}

// validateLinkTarget asks the plugin whether the directory looks like one of its
// SDKs: the directories its EnvKeys puts on PATH must hold commands.
func (b *impl) validateLinkTarget(version Version, target string) error {
	pkg := &RuntimePackage{
		Runtime:     &Runtime{Name: b.sdkName(), Version: version, Path: target},
		PackagePath: target,
	}
	envs, err := b.EnvKeys(pkg)
	if err != nil {
		return fmt.Errorf("%s does not look like a %s installation: %w", target, b.sdkName(), err)
	}
	dirs := envs.Paths.Slice()
	if len(dirs) == 0 || len(env.ListExecutables(dirs)) > 0 {
		return nil // nothing to check, or it checks out
	}
	return fmt.Errorf("%s does not look like a %s installation: none of the directories the plugin puts on PATH %v holds a command", target, b.sdkName(), dirs)
}

// DetectedRuntime is an SDK found on this machine that vfox did not install.
type DetectedRuntime struct {
	Version Version // the name to link it under
	Path    string
	Note    string
}

// ErrDetectNotSupported is returned by Detect for a plugin without a Detect hook.
var ErrDetectNotSupported = errors.New("the plugin cannot detect SDKs installed elsewhere")

// Detect asks the plugin for SDKs on this machine that could be linked.
func (b *impl) Detect() ([]DetectedRuntime, error) {
	if b.plugin == nil || !b.plugin.HasFunction("Detect") {
		return nil, ErrDetectNotSupported
	}
	items, err := b.plugin.Detect(&plugin.DetectHookCtx{})
	if b.plugin.IsNoResultProvided(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("plugin [Detect] method error: %w", err)
	}
	var found []DetectedRuntime
	seen := make(map[string]struct{})
	for _, item := range items {
		if item == nil || item.Path == "" || !ValidVersionName(item.Version) {
			logger.Debugf("Ignoring unusable detected SDK: %+v\n", item)
			continue
		}
		if _, dup := seen[item.Version]; dup {
			continue
		}
		seen[item.Version] = struct{}{}
		found = append(found, DetectedRuntime{Version: Version(item.Version), Path: item.Path, Note: item.Note})
	}
	return found, nil
}
