//go:build windows

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

package env

import (
	"errors"
	"fmt"
	"github.com/version-fox/vfox/internal/shell"
	"os"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows/registry"
)

type windowsEnvManager struct {
	shellInfo *shell.Shell
	key       registry.Key
	store     *Store
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
		w.store.pathMap[path] = struct{}{}
	}
	return nil
}

func (w *windowsEnvManager) Flush(scope Scope) (err error) {
	// TODO maybe need to move this to other place
	// TODO  reimplement to flush env map to registry
	defer w.key.Close()
	customPaths := make([]string, 0, len(w.store.pathMap))
	customPathSet := make(map[string]struct{})
	if len(w.store.pathMap) > 0 {
		for path := range w.store.pathMap {
			customPaths = append(customPaths, path)
			customPathSet[path] = struct{}{}
		}
		pathValue := strings.Join(customPaths, ";")
		_ = w.Load([]*KV{
			{
				Key:   "VERSION_FOX_PATH",
				Value: pathValue,
			}},
		)

	} else {
		_ = w.Remove("VERSION_FOX_PATH")
	}
	// user env
	oldPath, _ := w.Get("PATH")
	s := strings.Split(oldPath, ";")
	userNewPaths := append([]string{}, customPaths...)
	for _, v := range s {
		if _, ok := w.store.deletedPathMap[v]; ok {
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
		if _, ok := w.store.deletedPathMap[v]; ok {
			continue
		}
		if _, ok := customPathSet[v]; ok {
			continue
		}
		sysNewPaths = append(sysNewPaths, v)
	}
	_ = os.Setenv("PATH", strings.Join(sysNewPaths, ";"))
	_ = w.broadcastEnvironment()
	return nil
}

func (w *windowsEnvManager) Load(kvs []*KV) error {
	for _, kv := range kvs {
		w.store.Add(kv)
	}
	return nil
}

func (w *windowsEnvManager) Get(key string) (string, error) {
	if key == "PATH" {
		return w.pathEnvValue(), nil
	} else {
		return w.store.envMap[key], nil
	}
}

func (w *windowsEnvManager) Remove(key string) error {
	if key == "PATH" {
		return fmt.Errorf("can not remove PATH variable")
	}
	w.store.Remove(key)
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
	for path := range w.store.pathMap {
		paths = append(paths, path)
	}
	return strings.Join(paths, ";")
}

func NewEnvManager(vfConfigPath string, shellInfo *shell.Shell) (Manager, error) {
	k, err := registry.OpenKey(registry.CURRENT_USER, `Environment`, registry.SET_VALUE|registry.QUERY_VALUE)
	if err != nil {
		return nil, err
	}
	manager := &windowsEnvManager{
		shellInfo: shellInfo,
		key:       k,
		store:     NewStore(),
	}
	err = manager.loadPathValue()
	if err != nil {
		return nil, err
	}
	return manager, nil
}
