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
	"errors"
	"fmt"
	"github.com/aooohan/version-fox/env"
	"github.com/aooohan/version-fox/util"
	"github.com/schollz/progressbar/v3"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

type Version string

type Handler struct {
	sdkManager *Manager
	envManager env.Manager
	// current sdk install path
	localPath string
	// sdk name
	Name string
	// sdk source
	Source Plugin
}

func (b *Handler) Install(version Version) error {
	label := b.label(version)
	if b.checkExists(version) {
		fmt.Printf("%s has been installed, no need to install it.\n", label)
		return fmt.Errorf("%s has been installed, no need to install it.\n", label)
	}
	downloadUrl := b.Source.DownloadUrl(
		&PluginContext{
			Handler: b,
			Version: version,
		},
	)
	filePath, err := b.Download(downloadUrl)
	if err != nil {
		println(fmt.Sprintf("Failed to download %s file, err:%s", label, err.Error()))
		return err
	}
	decompressor := util.NewDecompressor(filePath)
	if decompressor == nil {
		fmt.Printf("Unable to process current file type, file: %s\n", filePath)
		return fmt.Errorf("unknown file type")
	}
	fileName := decompressor.Filename()
	destPath := filepath.Dir(filePath)
	err = decompressor.Decompress(destPath)
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
		_ = os.RemoveAll(b.localPath)
	}
	firstVersion := remainVersion[0]
	return b.Use(firstVersion)
}

func (b *Handler) Search(args string) error {
	versions := b.Source.Search(
		&PluginContext{
			Handler: b,
			Version: Version(args),
		},
	)
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
	keys := b.Source.EnvKeys(
		&PluginContext{
			Handler: b,
			Version: version,
		},
	)
	keys = append(keys, &env.KV{
		Key:   b.envVersionKey(),
		Value: string(version),
	})
	err := b.envManager.Load(keys)
	if err != nil {
		return fmt.Errorf("Use %s error, err: %s\n", label, err)
	}
	fmt.Printf("Now using %s\n", label)
	return b.envManager.ReShell()
}

func (b *Handler) List() []Version {
	if !util.FileExists(b.localPath) {
		return make([]Version, 0)
	}
	var versions []Version
	err := filepath.Walk(b.localPath, func(path string, info os.FileInfo, err error) error {
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
	value, _ := b.envManager.Get(b.envVersionKey())
	return Version(value)
}

func (b *Handler) Close() {
	b.envManager.Flush()
	b.Source.Close()
}

func (b *Handler) Update() {
	// TODO 插件升级系统
}

func (b *Handler) checkExists(version Version) bool {
	return util.FileExists(b.VersionPath(version))
}

func (b *Handler) VersionPath(version Version) string {
	return filepath.Join(b.localPath, fmt.Sprintf("v-%s", version))
}

func (b *Handler) Download(url *url.URL) (string, error) {
	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return "", errors.New("source file not found")
	}

	err = os.MkdirAll(b.localPath, 0755)
	if err != nil {
		return "", err
	}

	path := filepath.Join(b.localPath, filepath.Base(url.Path))

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return "", err
	}

	defer f.Close()

	bar := progressbar.DefaultBytes(
		resp.ContentLength,
		"Downloading",
	)
	_, err = io.Copy(io.MultiWriter(f, bar), resp.Body)
	if err != nil {
		return "", err
	}
	return path, nil
}

func (b *Handler) label(version Version) string {
	return fmt.Sprintf("%s@%s", strings.ToLower(b.Name), version)
}

func (b *Handler) envVersionKey() string {
	return fmt.Sprintf("%s_VERSION", strings.ToUpper(b.Name))
}

func NewHandler(manager *Manager, source Plugin) (*Handler, error) {
	name := source.Name()
	envManger, err := env.NewEnvManager(manager.configPath, name)
	if err != nil {
		return nil, err
	}
	return &Handler{
		sdkManager: manager,
		envManager: envManger,
		localPath:  filepath.Join(manager.sdkCachePath, strings.ToLower(name)),
		Name:       name,
		Source:     source,
	}, nil
}
