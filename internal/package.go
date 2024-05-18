/*
 *    Copyright 2024 Han Li and contributors
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
	"github.com/version-fox/vfox/internal/logger"
	"github.com/version-fox/vfox/internal/util"
	"os"
	"path/filepath"
)

// LocationPackage represents a package that needs to be linked
type LocationPackage struct {
	from     *Package
	sdk      *Sdk
	toPath   string
	location Location
}

func newLocationPackage(version Version, sdk *Sdk, location Location) (*LocationPackage, error) {
	var mockPath string
	switch location {
	case OriginalLocation:
		mockPath = ""
	case GlobalLocation:
		mockPath = filepath.Join(sdk.InstallPath, "current")
	case ShellLocation:
		mockPath = filepath.Join(sdk.sdkManager.PathMeta.CurTmpPath, sdk.Plugin.SdkName)
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

func (l *LocationPackage) ConvertLocation() *Package {
	if l.location == OriginalLocation {
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

func (l *LocationPackage) Link() (*Package, error) {
	if l.location == OriginalLocation {
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
func checkPackageValid(p *Package) bool {
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

type Package struct {
	Main      *Info
	Additions []*Info
}

func (p *Package) Clone() *Package {
	main := p.Main.Clone()
	additions := make([]*Info, len(p.Additions))
	for i, a := range p.Additions {
		additions[i] = a.Clone()
	}
	return &Package{
		Main:      main,
		Additions: additions,
	}
}

type Info struct {
	Name     string            `luai:"name"`
	Version  Version           `luai:"version"`
	Path     string            `luai:"path"`
	Headers  map[string]string `luai:"headers"`
	Note     string            `luai:"note"`
	Checksum *Checksum
}

func (i *Info) Clone() *Info {
	headers := make(map[string]string, len(i.Headers))
	for k, v := range i.Headers {
		headers[k] = v
	}
	return &Info{
		Name:     i.Name,
		Version:  i.Version,
		Path:     i.Path,
		Headers:  headers,
		Note:     i.Note,
		Checksum: i.Checksum,
	}
}

func (i *Info) label() string {
	return i.Name + "@" + string(i.Version)
}

func (i *Info) storagePath(parentDir string) string {
	if i.Version == "" {
		return filepath.Join(parentDir, i.Name)
	}
	return filepath.Join(parentDir, i.Name+"-"+string(i.Version))
}
