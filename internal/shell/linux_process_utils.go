//go:build linux

package shell

import (
	"fmt"
	"os"
	"strconv"
)

type linuxProcessUtils struct{}

var processUtils = linuxProcessUtils{}

func GetProcessUtils() ProcessUtils {
	return processUtils
}

func (l linuxProcessUtils) GetPath(pid int) (string, error) {
	path, err := os.Readlink("/proc/" + strconv.Itoa(pid) + "/exe")
	if err != nil {
		return "", fmt.Errorf("failed to get process path: %w", err)
	}
	return path, nil
}
