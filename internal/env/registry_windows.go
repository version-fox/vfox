//go:build windows

package env

import (
	"errors"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows/registry"
)

const (
	registryEnvironmentKey = `Environment`
	registryPathName       = "Path"
)

// applyEnvsToRegistry writes the provided environment variables and PATH segments
// to HKEY_CURRENT_USER\Environment, mimicking the Windows behavior from v0.x.
func applyEnvsToRegistry(envs *Envs) error {
	if envs == nil {
		return nil
	}
	key, err := registry.OpenKey(registry.CURRENT_USER, registryEnvironmentKey, registry.SET_VALUE|registry.QUERY_VALUE)
	if err != nil {
		return err
	}
	// Closing only releases the handle: the values are already set.
	defer func() { _ = key.Close() }()

	if err := setPathValue(key, envs.Paths.Slice()); err != nil {
		return err
	}
	if err := setEnvVariables(key, envs.Variables); err != nil {
		return err
	}
	return broadcastEnvironmentChange()
}

// removeEnvsFromRegistry removes the provided env vars and PATH segments from the registry.
func removeEnvsFromRegistry(envs *Envs) error {
	if envs == nil {
		return nil
	}
	key, err := registry.OpenKey(registry.CURRENT_USER, registryEnvironmentKey, registry.SET_VALUE|registry.QUERY_VALUE)
	if err != nil {
		return err
	}
	// Closing only releases the handle: the values are already set.
	defer func() { _ = key.Close() }()

	if err := removePathValue(key, envs.Paths.Slice()); err != nil {
		return err
	}
	if err := deleteEnvVariables(key, envs.Variables); err != nil {
		return err
	}
	return broadcastEnvironmentChange()
}

func setPathValue(key registry.Key, paths []string) error {
	if len(paths) == 0 {
		return nil
	}
	current, _, err := key.GetStringValue(registryPathName)
	if err != nil && !errors.Is(err, registry.ErrNotExist) {
		return err
	}
	merged := dedupOrderedPaths(append(paths, splitSemicolonSeparated(current)...))
	if len(merged) == 0 {
		return nil
	}
	return key.SetExpandStringValue(registryPathName, strings.Join(merged, ";"))
}

func removePathValue(key registry.Key, paths []string) error {
	if len(paths) == 0 {
		return nil
	}
	current, _, err := key.GetStringValue(registryPathName)
	if err != nil && !errors.Is(err, registry.ErrNotExist) {
		return err
	}
	if current == "" {
		return nil
	}
	remaining := removePaths(splitSemicolonSeparated(current), paths)
	return key.SetExpandStringValue(registryPathName, strings.Join(remaining, ";"))
}

func setEnvVariables(key registry.Key, vars Vars) error {
	for name, value := range vars {
		if value == nil {
			continue
		}
		if strings.Contains(*value, "%") {
			if err := key.SetExpandStringValue(name, *value); err != nil {
				return err
			}
		} else {
			if err := key.SetStringValue(name, *value); err != nil {
				return err
			}
		}
	}
	return nil
}

func deleteEnvVariables(key registry.Key, vars Vars) error {
	for name := range vars {
		_ = key.DeleteValue(name)
	}
	return nil
}

func broadcastEnvironmentChange() error {
	area, err := syscall.UTF16PtrFromString("Environment")
	if err != nil {
		return err
	}
	r, _, err := syscall.NewLazyDLL("user32.dll").NewProc("SendMessageTimeoutW").Call(
		0xffff, // HWND_BROADCAST
		0x1a,   // WM_SETTINGCHANGE
		0,
		uintptr(unsafe.Pointer(area)),
		0x02, // SMTO_ABORTIFHUNG
		5000, // 5 seconds
		0,
	)
	if r == 0 {
		return err
	}
	return nil
}
