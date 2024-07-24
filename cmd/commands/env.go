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

package commands

import (
	"encoding/json"
	"fmt"
	"github.com/urfave/cli/v2"
	"github.com/version-fox/vfox/internal"
	"github.com/version-fox/vfox/internal/cache"
	"github.com/version-fox/vfox/internal/env"
	"github.com/version-fox/vfox/internal/logger"
	"github.com/version-fox/vfox/internal/shell"
	"github.com/version-fox/vfox/internal/toolset"
	"path/filepath"
)

var Env = &cli.Command{
	Name: "env",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "shell",
			Aliases: []string{"s"},
			Usage:   "shell name",
		},
		&cli.BoolFlag{
			Name:    "cleanup",
			Aliases: []string{"c"},
			Usage:   "cleanup old temp files",
		},
		&cli.BoolFlag{
			Name:    "json",
			Aliases: []string{"j"},
			Usage:   "output json format",
		},
	},
	Action:   envCmd,
	Category: CategorySDK,
}

func envCmd(ctx *cli.Context) error {
	if ctx.IsSet("json") {
		return outputJSON()
	} else if ctx.IsSet("cleanup") {
		return cleanTmp()
	} else {
		return envFlag(ctx)
	}
}

func outputJSON() error {
	type SDKs map[string]map[string]*string
	data := struct {
		IsHookEnv bool     `json:"is_hook_env"`
		Paths     []string `json:"paths"`
		SDKs      SDKs     `json:"sdks"`
	}{
		IsHookEnv: env.IsHookEnv(),
		Paths:     []string{},
		SDKs:      make(SDKs),
	}
	manager := internal.NewSdkManager()
	defer manager.Close()
	tvs, err := toolset.NewMultiToolVersions([]string{
		manager.PathMeta.WorkingDirectory,
		manager.PathMeta.CurTmpPath,
		manager.PathMeta.HomePath,
	})
	if err != nil {
		return err
	}
	tvs.FilterTools(func(name, version string) bool {
		if lookupSdk, err := manager.LookupSdk(name); err == nil {
			if keys, err := lookupSdk.EnvKeys(internal.Version(version), internal.OriginalLocation); err == nil {
				data.SDKs[lookupSdk.Plugin.Name] = keys.Variables
				data.Paths = append(data.Paths, keys.Paths.Slice()...)
				return true
			}
		}
		return false
	})
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}
	fmt.Println(string(jsonData))
	return nil
}

func cleanTmp() error {
	manager := internal.NewSdkManager()
	defer manager.Close()
	// Clean up the old temp files, before today.
	manager.CleanTmp()
	return nil
}

func envFlag(ctx *cli.Context) error {
	shellName := ctx.String("shell")
	if shellName == "" {
		return cli.Exit("shell name is required", 1)
	}
	s := shell.NewShell(shellName)
	if s == nil {
		return fmt.Errorf("unknown target shell %s", shellName)
	}
	manager := internal.NewSdkManager()
	defer manager.Close()

	sdkEnvs, err := aggregateEnvKeys(manager)
	if err != nil {
		return err
	}

	if len(sdkEnvs) == 0 {
		return nil
	}

	envKeys := sdkEnvs.ToEnvs()

	exportEnvs := make(env.Vars)
	for k, v := range envKeys.Variables {
		exportEnvs[k] = v
	}

	osPaths := env.NewPaths(env.OsPaths)
	pathsStr := envKeys.Paths.Merge(osPaths).String()
	exportEnvs["PATH"] = &pathsStr

	exportStr := s.Export(exportEnvs)
	fmt.Println(exportStr)
	return nil
}

func aggregateEnvKeys(manager *internal.Manager) (internal.SdkEnvs, error) {
	workToolVersion, err := toolset.NewToolVersion(manager.PathMeta.WorkingDirectory)
	if err != nil {
		return nil, err
	}

	if err = manager.ParseLegacyFile(func(sdkname, version string) {
		if _, ok := workToolVersion.Record[sdkname]; !ok {
			workToolVersion.Record[sdkname] = version
		}
	}); err != nil {
		return nil, err
	}

	curToolVersion, err := toolset.NewToolVersion(manager.PathMeta.CurTmpPath)
	if err != nil {
		return nil, err
	}
	defer curToolVersion.Save()

	// Add the working directory to the first
	tvs := toolset.MultiToolVersions{workToolVersion, curToolVersion}

	flushCache, err := cache.NewFileCache(filepath.Join(manager.PathMeta.CurTmpPath, "flush_env.cache"))
	if err != nil {
		return nil, err
	}
	defer flushCache.Close()

	var sdkEnvs []*internal.SdkEnv

	tvs.FilterTools(func(name, version string) bool {
		if lookupSdk, err := manager.LookupSdk(name); err == nil {
			vv, ok := flushCache.Get(name)
			if ok && string(vv) == version {
				logger.Debugf("Hit cache, skip flush environment, %s@%s\n", name, version)
				return true
			} else {
				logger.Debugf("No hit cache, name: %s cache: %s, expected: %s \n", name, string(vv), version)
			}
			v := internal.Version(version)
			if keys, err := lookupSdk.EnvKeys(v, internal.ShellLocation); err == nil {
				flushCache.Set(name, cache.Value(version), cache.NeverExpired)

				sdkEnvs = append(sdkEnvs, &internal.SdkEnv{
					Sdk: lookupSdk, Env: keys,
				})

				// If we encounter a .tool-versions file, it is valid for the entire shell session,
				// unless we encounter the next .tool-versions file or manually switch to the use command.
				curToolVersion.Record[name] = version
				return true
			}
		}
		return false
	})

	return sdkEnvs, nil
}
