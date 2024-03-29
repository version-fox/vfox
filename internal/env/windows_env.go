//go:build windows

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

package env

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"

	"github.com/version-fox/vfox/internal/util"
	"golang.org/x/sys/windows/registry"
)

type windowsEnvManager struct {
	key registry.Key
	// $PATH
	paths          []string
	pathMap        map[string]struct{}
	deletedPathMap map[string]struct{}
}

func (w *windowsEnvManager) Close() error {
	return w.key.Close()
}

func (w *windowsEnvManager) loadPathValue() error {
	val, _, err := w.key.GetStringValue("VERSION_FOX_PATH")
	if err != nil {
		if errors.Is(err, registry.ErrNotExist) {
			return nil
		}
		return err
	}
	if len(val) == 0 {
		return nil
	}
	s := strings.Split(val, ";")
	for _, path := range s {
		if _, ok := w.pathMap[path]; ok {
			continue
		}
		w.paths = append(w.paths, path)
		w.pathMap[path] = struct{}{}
	}
	return nil
}

func (w *windowsEnvManager) Flush() (err error) {
	customPaths := make([]string, 0, len(w.pathMap))
	customPathSet := make(map[string]struct{})
	if len(w.pathMap) > 0 {
		for i := len(w.paths) - 1; i >= 0; i-- {
			path := w.paths[i]
			customPaths = append(customPaths, path)
			customPathSet[path] = struct{}{}
		}
		pathValue := strings.Join(customPaths, ";")
		if err = w.Load((&Envs{
			Variables: Vars{
				"VERSION_FOX_PATH": &pathValue,
			},
			Paths: NewPaths(EmptyPaths),
		})); err != nil {
			return err
		}
	} else {
		_ = w.Remove(&Envs{
			Variables: Vars{
				"VERSION_FOX_PATH": nil,
			}})
	}
	// user env
	oldPath, success := w.Get("PATH")
	if !success {
		return
	}
	s := strings.Split(oldPath, ";")
	userNewPaths := append([]string{}, customPaths...)
	for _, v := range s {
		if _, ok := w.deletedPathMap[v]; ok {
			continue
		}
		if _, ok := customPathSet[v]; ok {
			continue
		}
		userNewPaths = append(userNewPaths, v)
	}
	if err = w.key.SetStringValue("PATH", strings.Join(userNewPaths, ";")); err != nil {
		return err
	}
	// sys env
	sysPath := os.Getenv("PATH")
	s2 := strings.Split(sysPath, ";")
	sysNewPaths := append([]string{}, customPaths...)
	for _, v := range s2 {
		if _, ok := w.deletedPathMap[v]; ok {
			continue
		}
		if _, ok := customPathSet[v]; ok {
			continue
		}
		sysNewPaths = append(sysNewPaths, v)
	}
	if err = os.Setenv("PATH", strings.Join(sysNewPaths, ";")); err != nil {
		return err
	}
	_ = w.broadcastEnvironment()
	return
}

func (w *windowsEnvManager) Load(envs *Envs) error {
	for k, v := range envs.Variables {
		err := os.Setenv(k, *v)
		if err != nil {
			return err
		}
		err = w.key.SetStringValue(k, *v)
		if err != nil {
			return err
		}
	}
	for _, path := range envs.Paths.Slice() {
		path = filepath.FromSlash(path)
		_, ok := w.pathMap[path]
		if !ok {
			w.pathMap[path] = struct{}{}
			w.paths = append(w.paths, path)
		}
	}
	return nil
}

func (w *windowsEnvManager) Get(key string) (string, bool) {
	val, _, err := w.key.GetStringValue(key)
	if err != nil {
		return "", false
	}
	return val, true
}

func (w *windowsEnvManager) Remove(envs *Envs) error {
	for k, _ := range envs.Variables {
		if k == "PATH" {
			return fmt.Errorf("can not remove PATH variable")
		}
		_ = w.key.DeleteValue(k)
	}

	for _, k := range envs.Paths.Slice() {
		if _, ok := w.pathMap[k]; ok {
			delete(w.pathMap, k)
			var newPaths []string
			for _, v := range w.paths {
				if v != k {
					newPaths = append(newPaths, v)
				}
			}
			w.paths = newPaths
			w.deletedPathMap[k] = struct{}{}
		}
	}
	return nil
}

func (w *windowsEnvManager) broadcastEnvironment() error {
	r, _, err := syscall.NewLazyDLL("user32.dll").NewProc("SendMessageTimeoutW").Call(
		0xffff, // HWND_BROADCAST
		0x1a,   // WM_SETTINGCHANGE
		0,
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr("Environment"))),
		0x02, // SMTO_ABORTIFHUNG
		5000, // 5 seconds
		0,
	)
	if r == 0 {
		return err
	}
	return nil
}

func (w *windowsEnvManager) Paths(paths []string) string {
	if os.Getenv(HookFlag) == "bash" {
		set := util.NewSortedSet[string]()
		for _, p := range paths {
			for _, pp := range strings.Split(p, ";") {
				set.Add(pp)
			}
		}
		return strings.Join(set.Slice(), ":")
	} else {
		set := util.NewSortedSetWithSlice(paths)
		return strings.Join(set.Slice(), ";")
	}
}

func NewEnvManager(vfConfigPath string) (Manager, error) {
	k, err := registry.OpenKey(registry.CURRENT_USER, `Environment`, registry.SET_VALUE|registry.QUERY_VALUE)
	if err != nil {
		return nil, err
	}
	manager := &windowsEnvManager{
		key:            k,
		pathMap:        make(map[string]struct{}),
		deletedPathMap: make(map[string]struct{}),
		paths:          make([]string, 0),
	}
	err = manager.loadPathValue()
	if err != nil {
		return nil, err
	}
	return manager, nil
}

func (p *Paths) String() string {
	if os.Getenv(HookFlag) == "bash" {
		return strings.Join(p.Slice(), ":")
	} else {
		return strings.Join(p.Slice(), ";")
	}
}
