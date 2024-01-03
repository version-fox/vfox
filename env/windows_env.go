//go:build windows

/*
 *    Copyright 2024 [lihan aooohan@gmail.com]
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
	"os/exec"
	"strings"
	"syscall"
	"unsafe"

	"github.com/pterm/pterm"
	"golang.org/x/sys/windows/registry"
)

type windowsEnvManager struct {
	shellInfo *ShellInfo
	key       registry.Key
	// $PATH
	pathMap        map[string]struct{}
	deletedPathMap map[string]struct{}
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

func (w *windowsEnvManager) Flush() {
	defer w.key.Close()
	customPaths := make([]string, 0, len(w.pathMap))
	customPathSet := make(map[string]struct{})
	if len(w.pathMap) > 0 {
		for path := range w.pathMap {
			customPaths = append(customPaths, path)
			customPathSet[path] = struct{}{}
		}
		pathValue := strings.Join(customPaths, ";")
		w.Load([]*KV{
			{
				Key:   "VERSION_FOX_PATH",
				Value: pathValue,
			}},
		)

	} else {
		_ = w.Remove("VERSION_FOX_PATH")
	}
	// user env
	oldPath, err := w.Get("PATH")
	if err != nil {
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
}

func (w *windowsEnvManager) Load(kvs []*KV) error {
	for _, kv := range kvs {
		if kv.Key == "PATH" {
			w.pathMap[kv.Value] = struct{}{}
		} else {
			err := os.Setenv(kv.Key, kv.Value)
			if err != nil {
				return err
			}
			err = w.key.SetStringValue(kv.Key, kv.Value)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (w *windowsEnvManager) Get(key string) (string, error) {
	val, _, err := w.key.GetStringValue(key)
	if err != nil {
		return "", err
	}
	return val, nil
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

func (w *windowsEnvManager) ReShell(callback func()) error {
	// flush env to file
	w.Flush()
	command := exec.Command(w.shellInfo.ShellPath)
	command.Stdin = os.Stdin
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	if err := command.Run(); err != nil {
		pterm.Printf("Failed to start shell, err:%s\n", err.Error())
		return err
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

func NewEnvManager(vfConfigPath string) (Manager, error) {
	k, err := registry.OpenKey(registry.CURRENT_USER, `Environment`, registry.SET_VALUE|registry.QUERY_VALUE)
	if err != nil {
		return nil, err
	}
	manager := &windowsEnvManager{
		shellInfo:      NewShellInfo(),
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

func NewShellInfo() *ShellInfo {
	ppid := os.Getppid()

	// On Windows, os.FindProcess does not actually find the process.
	// So, we use this workaround to get the parent process name.
	cmd := exec.Command("tasklist", "/FI", fmt.Sprintf("PID eq %d", ppid), "/NH", "/FO", "CSV")
	output, _ := cmd.Output()
	fields := strings.Split(string(output), ",")
	parentProcessName := strings.Trim(fields[0], "\" ")
	cmd = exec.Command("wmic", "process", "where", fmt.Sprintf("ProcessId=%d", ppid), "get", "ExecutablePath", "/format:list")
	output, _ = cmd.Output()
	path := strings.TrimPrefix(strings.TrimSpace(string(output)), "ExecutablePath=")
	return &ShellInfo{
		ShellType:  ShellType(parentProcessName),
		ShellPath:  path,
		ConfigPath: "",
	}
}
