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
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows/registry"
)

type windowsEnvManager struct {
	key registry.Key
	// $PATH
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
		w.pathMap[path] = struct{}{}
	}
	return nil
}

func (w *windowsEnvManager) Flush() (err error) {
	// TODO move this close method to other place
	defer w.key.Close()
	customPaths := make([]string, 0, len(w.pathMap))
	customPathSet := make(map[string]struct{})
	if len(w.pathMap) > 0 {
		for path := range w.pathMap {
			customPaths = append(customPaths, path)
			customPathSet[path] = struct{}{}
		}
		pathValue := strings.Join(customPaths, ";")
		w.Load("VERSION_FOX_PATH", pathValue)
	} else {
		_ = w.Remove("VERSION_FOX_PATH")
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
	w.key.SetStringValue("PATH", strings.Join(userNewPaths, ";"))
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
	os.Setenv("PATH", strings.Join(sysNewPaths, ";"))
	_ = w.broadcastEnvironment()
	return
}

func (w *windowsEnvManager) Load(key, value string) {
	if key == "PATH" {
		w.pathMap[value] = struct{}{}
	} else {
		// TODO handle error
		err := os.Setenv(key, value)
		if err != nil {
		}
		err = w.key.SetStringValue(key, value)
		if err != nil {
		}
	}
}

func (w *windowsEnvManager) Get(key string) (string, bool) {
	val, _, err := w.key.GetStringValue(key)
	if err != nil {
		return "", false
	}
	return val, true
}

func (w *windowsEnvManager) Remove(key string) error {
	if key == "PATH" {
		return fmt.Errorf("can not remove PATH variable")
	}
	if _, ok := w.pathMap[key]; ok {
		delete(w.pathMap, key)
		w.deletedPathMap[key] = struct{}{}
	} else {
		_ = w.key.DeleteValue(key)
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

func (w *windowsEnvManager) pathEnvValue() string {
	var paths []string
	for path := range w.pathMap {
		paths = append(paths, path)
	}
	return w.Paths(paths)
}

func (w *windowsEnvManager) Paths(paths []string) string {
	s := os.Getenv("PATH")
	paths = append(paths, s)
	return strings.Join(paths, ";")
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
	}
	err = manager.loadPathValue()
	if err != nil {
		return nil, err
	}
	return manager, nil
}
