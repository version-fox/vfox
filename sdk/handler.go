/*
 *    Copyright 2023 [lihan aooohan@gmail.com]
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
	"github.com/aooohan/version-fox/env"
	"github.com/aooohan/version-fox/util"
	"os"
	"path/filepath"
	"strings"
)

type Version string

type Handler struct {
	Operation  *Operation
	EnvManager env.Manager
	Name       string
	Source     Source
}

func (b *Handler) Install(version Version) error {
	label := b.label(version)
	if b.checkExists(version) {
		fmt.Printf("%s has been installed, no need to install it.\n", label)
		return fmt.Errorf("%s has been installed, no need to install it.\n", label)
	}
	downloadUrl := b.Source.DownloadUrl(b, version)
	filePath, err := b.Operation.Download(downloadUrl)
	if err != nil {
		println(fmt.Sprintf("Failed to download %s file, err:%s", label, err.Error()))
		return err
	}
	fileName := strings.TrimSuffix(filepath.Base(filePath), b.Source.FileExt(b))
	destPath := filepath.Dir(filePath)
	err = util.DecompressGzipTar(filePath, destPath)
	if err != nil {
		return err
	}
	newDirPath := b.VersionPath(version)
	err = os.Rename(filepath.Join(destPath, fileName), newDirPath)
	if err != nil {
		return err
	}
	fmt.Printf("Install %s success!\n", label)
	// del cache file
	_ = os.Remove(filePath)
	return nil
}

func (b *Handler) Uninstall(version Version) error {
	if !b.checkExists(version) {
		fmt.Printf("%s is not installed, no need to uninstall it.\n", b.label(version))
		return fmt.Errorf("%s is not installed, no need to uninstall it.\n", b.label(version))
	}
	err := os.RemoveAll(b.VersionPath(version))
	if err != nil {
		return err
	}
	fmt.Printf("Uninstall %s success!\n", b.label(version))
	remainVersion := b.List()
	if len(remainVersion) == 0 {
		_ = os.RemoveAll(b.Operation.LocalPath)
	}
	firstVersion := remainVersion[0]
	return b.Use(firstVersion)
}

func (b *Handler) Search(args string) error {
	versions := b.Source.Search(b, Version(args))
	if len(versions) == 0 {
		fmt.Printf("No available %s version.\n", b.Name)
		return nil
	}
	for _, version := range versions {
		fmt.Printf("-> %s\n", version)
	}
	return nil
}

func (b *Handler) Use(version Version) error {
	label := b.label(version)
	if !b.checkExists(version) {
		fmt.Printf("%s is not installed, please install it first.\n", label)
		return fmt.Errorf("%s is not installed, please install it first.\n", label)
	}
	keys := b.Source.EnvKeys(b, version)
	keys = append(keys, &env.KV{
		Key:   b.envVersionKey(),
		Value: string(version),
	})
	err := b.EnvManager.Load(keys)
	if err != nil {
		return fmt.Errorf("Use %s error, err: %s\n", label, err)
	}
	fmt.Printf("Now using %s\n", label)
	return b.EnvManager.ReShell()
}

func (b *Handler) List() []Version {
	if !util.FileExists(b.Operation.LocalPath) {
		return make([]Version, 0)
	}
	var versions []Version
	err := filepath.Walk(b.Operation.LocalPath, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() && strings.HasPrefix(info.Name(), "v-") {
			versions = append(versions, Version(strings.TrimPrefix(info.Name(), "v-")))
		}
		return nil
	})
	if err != nil {
		return nil
	}
	return versions
}

func (b *Handler) Current() Version {
	value, _ := b.EnvManager.Get(b.envVersionKey())
	return Version(value)
}

func (b *Handler) Close() {
	b.Source.Close()
}

func (b *Handler) checkExists(version Version) bool {
	return util.FileExists(b.VersionPath(version))
}

func (b *Handler) VersionPath(version Version) string {
	return filepath.Join(b.Operation.LocalPath, fmt.Sprintf("v-%s", version))
}

func (b *Handler) label(version Version) string {
	return fmt.Sprintf("%s@%s", strings.ToLower(b.Name), version)
}

func (b *Handler) envVersionKey() string {
	return fmt.Sprintf("%s_VERSION", strings.ToUpper(b.Name))
}

func NewHandler(manager *Manager, source Source) (*Handler, error) {
	name := source.Name()
	operation := &Operation{
		LocalPath:    filepath.Join(manager.sdkCachePath, name),
		vfConfigPath: manager.configPath,
		OsType:       manager.osType,
		ArchType:     manager.archType,
	}
	envManger, err := env.NewEnvManager(manager.configPath, manager.sdkCachePath, name)
	if err != nil {
		return nil, err
	}
	return &Handler{
		Operation:  operation,
		EnvManager: envManger,
		Name:       name,
		Source:     source,
	}, nil
}
