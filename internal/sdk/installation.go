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
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/version-fox/vfox/internal/env"
	"github.com/version-fox/vfox/internal/shared/logger"
)

// Installation is an installed version together with what vfox knows about it.
type Installation struct {
	*RuntimePackage
	Origin  Origin
	Target  string   // linked: the directory the main runtime points to
	Receipt *Receipt // nil for a version installed before receipts existed
}

// Installation describes an installed version.
func (b *impl) Installation(version Version) (*Installation, error) {
	pkg, err := b.GetRuntimePackage(version)
	if err != nil {
		return nil, err
	}
	return b.describe(pkg), nil
}

func (b *impl) describe(pkg *RuntimePackage) *Installation {
	inst := &Installation{RuntimePackage: pkg, Origin: OriginInstalled}
	receipt, err := readReceipt(pkg.PackagePath)
	if err != nil {
		logger.Debugf("Ignoring unreadable receipt of %s: %v\n", pkg.PackagePath, err)
	}
	inst.Receipt = receipt

	target, isLink := linkTarget(pkg.Runtime.Path)
	switch {
	case receipt != nil && receipt.Origin != "":
		// The receipt is the record of how the version got here.
		inst.Origin = receipt.Origin
	case isLink:
		// A link nobody recorded, such as one made by hand with `ln -s`.
		inst.Origin = OriginLinked
	}
	switch {
	case isLink:
		inst.Target = target
	case receipt != nil:
		inst.Target = receipt.Target
	}
	return inst
}

// linkTarget returns where a symlink or junction points.
func linkTarget(path string) (string, bool) {
	if !env.IsDirSymlink(path) {
		return "", false
	}
	target, err := env.ReadDirSymlink(path)
	if err != nil {
		return "", false
	}
	return target, true
}

// sdkName is the name vfox lays the SDK's directories out under.
func (b *impl) sdkName() string {
	if b.plugin != nil && b.plugin.Metadata != nil {
		return b.plugin.Name
	}
	return b.Name
}

// mainRuntimePath is where the main runtime of a version lives (or is linked).
func (b *impl) mainRuntimePath(version Version) string {
	return filepath.Join(b.packagePath(version), b.sdkName()+"-"+string(version))
}

// brokenLink reports whether the main runtime is a link whose target is gone.
func (b *impl) brokenLink(version Version) bool {
	main := b.mainRuntimePath(version)
	if !env.IsDirSymlink(main) {
		return false
	}
	_, err := os.Stat(main)
	return err != nil
}

// linkedTarget reports whether a version is a link to somewhere else, working
// even when the target has gone away.
func (b *impl) linkedTarget(version Version) (string, bool) {
	receipt, err := readReceipt(b.packagePath(version))
	if err != nil {
		logger.Debugf("Ignoring unreadable receipt of %s: %v\n", b.Label(version), err)
	}
	target, isLink := linkTarget(b.mainRuntimePath(version))
	switch {
	case receipt != nil && receipt.Origin == OriginLinked:
		if isLink {
			return target, true
		}
		return receipt.Target, true
	case receipt != nil && receipt.Origin != "":
		return "", false // vfox installed it; a link inside is the plugin's own doing
	case isLink:
		return target, true // a link nobody recorded
	}
	return "", false
}

// removeLink deletes a linked version: the link and the receipt, never the target.
func (b *impl) removeLink(version Version) error {
	main := b.mainRuntimePath(version)
	if _, err := os.Lstat(main); err == nil {
		if !env.IsDirSymlink(main) {
			return fmt.Errorf("refusing to delete %s: it is a real directory, not a link", main)
		}
		if err := env.RemoveDirSymlink(main); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove link %s: %w", main, err)
		}
	}
	return os.RemoveAll(b.packagePath(version))
}

func (b *impl) vfoxVersion() string {
	if b.envContext == nil {
		return ""
	}
	return b.envContext.RuntimeVersion
}

// newReceipt starts a receipt with what is true of every kind of installation.
func (b *impl) newReceipt(version Version, origin Origin) *Receipt {
	receipt := &Receipt{
		Schema:      receiptSchema,
		SDK:         b.sdkName(),
		Version:     string(version),
		Origin:      origin,
		InstalledAt: time.Now().UTC().Truncate(time.Second),
		Vfox:        b.vfoxVersion(),
	}
	if b.plugin != nil && b.plugin.Metadata != nil {
		receipt.Plugin = ReceiptPlugin{Name: b.plugin.Name, Version: b.plugin.Version}
	}
	return receipt
}
