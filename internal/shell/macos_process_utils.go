//go:build darwin

package shell

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

type macosProcessPath struct{}

var processPath = macosProcessPath{}

func GetProcessUtils() ProcessUtils {
	return processPath
}

func (m macosProcessPath) GetPath(pid int) (string, error) {
	out, err := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "command=").Output()
	if err != nil {
		return "", fmt.Errorf("failed to get process path: %w", err)
	}
	outCommand := strings.Fields(string(out))
	if len(outCommand) == 0 {
		return "", fmt.Errorf("process not found")
	}
	return outCommand[0], nil
}
