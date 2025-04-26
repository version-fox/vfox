//go:build windows

package shell

import (
	"fmt"

	"golang.org/x/sys/windows"
)

type windowsProcessUtils struct{}

var processUtils = windowsProcessUtils{}

func GetProcessUtils() ProcessUtils {
	return processUtils
}

func (w windowsProcessUtils) GetPath(pid int) (string, error) {
	hProcess, err := windows.OpenProcess(windows.PROCESS_QUERY_INFORMATION, false, uint32(pid))
	if err != nil {
		return "", fmt.Errorf("failed to open process: %w", err)
	}
	defer windows.CloseHandle(hProcess)

	var exePath [windows.MAX_PATH]uint16
	size := uint32(len(exePath))
	if err := windows.QueryFullProcessImageName(hProcess, 0, &exePath[0], &size); err != nil {
		return "", fmt.Errorf("failed to query process path: %w", err)
	}

	return windows.UTF16ToString(exePath[:size]), nil
}
