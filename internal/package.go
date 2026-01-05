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
	"fmt"
	"os"
	"path/filepath"

	"github.com/version-fox/vfox/internal/base"
	"github.com/version-fox/vfox/internal/logger"
	"github.com/version-fox/vfox/internal/util"
)

// LocationPackage represents a package that needs to be linked
type LocationPackage struct {
	from     *base.Package
	sdk      *Sdk
	toPath   string
	location base.Location
}

func newLocationPackage(version base.Version, sdk *Sdk, location base.Location) (*LocationPackage, error) {
	var mockPath string
	switch location {
	case base.OriginalLocation:
		mockPath = ""
	case base.GlobalLocation:
		mockPath = filepath.Join(sdk.InstallPath, "current")
	case base.ShellLocation:
		mockPath = filepath.Join(sdk.sdkManager.PathMeta.Working.SessionShim, sdk.Name)
	default:
		return nil, fmt.Errorf("unknown location: %s", location)
	}
	sdkPackage, err := sdk.GetLocalSdkPackage(version)
	if err != nil {
		return nil, fmt.Errorf("failed to get local sdk info, err:%w", err)
	}
	return &LocationPackage{
		from:     sdkPackage,
		sdk:      sdk,
		toPath:   mockPath,
		location: location,
	}, nil
}

func (l *LocationPackage) ConvertLocation() *base.Package {
	if l.location == base.OriginalLocation {
		return l.from
	}
	clone := l.from.Clone()
	mockPath := l.toPath
	sdkPackage := clone
	hasAddition := len(sdkPackage.Additions) != 0
	if !hasAddition {
		sdkPackage.Main.Path = mockPath
	} else {
		sdkPackage.Main.Path = filepath.Join(mockPath, sdkPackage.Main.Name)
		for _, a := range sdkPackage.Additions {
			a.Path = filepath.Join(mockPath, a.Name)
		}
	}
	return clone
}

func (l *LocationPackage) Link() (*base.Package, error) {
	if l.location == base.OriginalLocation {
		return l.from, nil
	}
	mockPath := l.toPath
	sourcePackage := l.from
	targetPackage := l.ConvertLocation()
	// If the mock path already exists, delete it first.
	logger.Debugf("Removing old package path: %s\n", mockPath)
	_ = os.RemoveAll(mockPath)
	hasAddition := len(targetPackage.Additions) != 0
	if !hasAddition {
		logger.Debugf("Create symlink %s -> %s\n", sourcePackage.Main.Path, targetPackage.Main.Path)
		if err := util.MkSymlink(sourcePackage.Main.Path, targetPackage.Main.Path); err != nil {
			return nil, fmt.Errorf("failed to create symlink, err:%w", err)
		}
	} else {
		_ = os.MkdirAll(mockPath, 0755)
		logger.Debugf("Create symlink %s -> %s\n", sourcePackage.Main.Path, targetPackage.Main.Path)
		if err := util.MkSymlink(sourcePackage.Main.Path, targetPackage.Main.Path); err != nil {
			return nil, fmt.Errorf("failed to create symlink, err:%w", err)
		}
		for i, a := range targetPackage.Additions {
			sa := sourcePackage.Additions[i]
			logger.Debugf("Create symlink %s -> %s\n", sa.Path, a.Path)
			if err := util.MkSymlink(sa.Path, a.Path); err != nil {
				return nil, fmt.Errorf("failed to create symlink, err:%w", err)
			}
		}
	}
	return targetPackage, nil
}

// checkPackageValid checks if the package is valid
func checkPackageValid(p *base.Package) bool {
	if !util.FileExists(p.Main.Path) {
		return false
	}
	for _, a := range p.Additions {
		if !util.FileExists(a.Path) {
			return false
		}
	}
	return true
}
