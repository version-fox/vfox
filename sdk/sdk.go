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
	"github.com/aooohan/version-fox/plugin"
	"github.com/aooohan/version-fox/util"
	"github.com/pterm/pterm"
	"github.com/schollz/progressbar/v3"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Version string

type Sdk struct {
	sdkManager *Manager
	Plugin     *plugin.LuaPlugin
	// current sdk install path
	sdkPath string
}

func (b *Sdk) Install(version Version) error {
	label := b.label(version)
	if b.checkExists(version) {
		pterm.Printf("%s is already installed.\n", pterm.LightGreen(label))
		return fmt.Errorf("%s has been installed\n", label)
	}
	downloadUrl := b.Plugin.DownloadUrl(string(version))
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
	pterm.Printf("Unpacking %s...\n", filePath)
	err = decompressor.Decompress(destPath)
	if err != nil {
		return err
	}
	newDirPath := b.VersionPath(version)
	err = os.Rename(filepath.Join(destPath, fileName), newDirPath)
	if err != nil {
		return err
	}
	pterm.Printf("Install %s success! \n", pterm.LightGreen(label))
	pterm.Printf("Please use %s to use it.\n", pterm.LightBlue(fmt.Sprintf("vf use %s", label)))
	// del cache file
	_ = os.Remove(filePath)
	return nil
}

func (b *Sdk) Uninstall(version Version) error {
	label := b.label(version)
	if !b.checkExists(version) {
		pterm.Printf("%s is not installed...\n", pterm.Red(label))
		return fmt.Errorf("%s is not installed", label)
	}
	path := b.VersionPath(version)
	err := os.RemoveAll(path)
	if err != nil {
		return err
	}
	if b.Current() == version {
		b.clearEnvConfig(version, path)
	}
	pterm.Printf("Uninstalled %s successfully!\n", label)
	return nil
}

func (b *Sdk) Available(version string) []Version {
	versions := b.Plugin.Available(version)
	var result []Version
	for _, version := range versions {
		v := Version(version)
		result = append(result, v)
	}
	return result
}

func (b *Sdk) Use(version Version) error {
	label := b.label(version)
	if !b.checkExists(version) {
		pterm.Printf("No %s installed, please install it first.", pterm.Yellow(label))
		return fmt.Errorf("%s is not installed", label)
	}
	keys := b.Plugin.EnvKeys(string(version), b.VersionPath(version))
	keys = append(keys, &env.KV{
		Key:   b.envVersionKey(),
		Value: string(version),
	})
	err := b.sdkManager.EnvManager.Load(keys)
	if err != nil {
		return fmt.Errorf("Use %s error, err: %s\n", label, err)
	}
	pterm.Printf("Now using %s\n", pterm.LightGreen(label))
	return b.sdkManager.EnvManager.ReShell()
}

func (b *Sdk) List() []Version {
	if !util.FileExists(b.sdkPath) {
		return make([]Version, 0)
	}
	var versions []Version
	err := filepath.Walk(b.sdkPath, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() && strings.HasPrefix(info.Name(), "v-") {
			versions = append(versions, Version(strings.TrimPrefix(info.Name(), "v-")))
		}
		return nil
	})
	if err != nil {
		return nil
	}
	sort.Slice(versions, func(i, j int) bool {
		return versions[i] > versions[j]
	})
	return versions
}

func (b *Sdk) Current() Version {
	value, _ := b.sdkManager.EnvManager.Get(b.envVersionKey())
	return Version(value)
}

func (b *Sdk) Close() {
	b.sdkManager.EnvManager.Flush()
	b.Plugin.Close()
}
func (b *Sdk) clearCurrentEnvConfig() {
	b.clearEnvConfig(b.Current(), b.VersionPath(b.Current()))
}

func (b *Sdk) clearEnvConfig(version Version, path string) {
	envKV := b.Plugin.EnvKeys(string(version), path)
	envManager := b.sdkManager.EnvManager
	for _, kv := range envKV {
		if kv.Key == "PATH" {
			_ = envManager.Remove(kv.Value)
		} else {
			_ = envManager.Remove(kv.Key)
		}
	}
	_ = envManager.Remove(b.envVersionKey())
}

func (b *Sdk) checkExists(version Version) bool {
	return util.FileExists(b.VersionPath(version))
}

func (b *Sdk) VersionPath(version Version) string {
	return filepath.Join(b.sdkPath, fmt.Sprintf("v-%s", version))
}

func (b *Sdk) Download(url *url.URL) (string, error) {
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

	err = os.MkdirAll(b.sdkPath, 0755)
	if err != nil {
		return "", err
	}

	path := filepath.Join(b.sdkPath, filepath.Base(url.Path))

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

func (b *Sdk) label(version Version) string {
	return fmt.Sprintf("%s@%s", strings.ToLower(b.Plugin.Name), version)
}

func (b *Sdk) envVersionKey() string {
	return fmt.Sprintf("%s_VERSION", strings.ToUpper(b.Plugin.Name))
}

func NewSdk(manager *Manager, source *plugin.LuaPlugin) (*Sdk, error) {
	name := source.Name
	return &Sdk{
		sdkManager: manager,
		sdkPath:    filepath.Join(manager.sdkCachePath, strings.ToLower(name)),
		Plugin:     source,
	}, nil
}
