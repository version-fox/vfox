//go:build darwin

package env

import (
	"bufio"
	"fmt"
	"github.com/aooohan/version-fox/util"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
)

const (
	BASH = ShellType("bash")
	ZSH  = ShellType("zsh")
	KSH  = ShellType("ksh")
	// extend shell type
)

type macosEnvManager struct {
	shellInfo *ShellInfo
	// ~/.version_fox/.cache/node/env.sh
	sdkEnvPath string
	// ~/.version_fox/env.sh
	vfEnvPath string
}

func (m *macosEnvManager) ReShell() error {
	return exec.Command(m.shellInfo.ShellPath).Run()
}

func (m *macosEnvManager) Load(kvs []*KV) error {
	if err := appendEnvSourceIfNotExist(m.vfEnvPath, m.sdkEnvPath); err != nil {
		return err
	}
	if !util.FileExists(m.sdkEnvPath) {
		_, _ = os.Create(m.sdkEnvPath)
	}
	// create if it not exists, else trunc
	file, err := os.OpenFile(m.sdkEnvPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer file.Close()
	for _, kv := range kvs {
		var str string
		if kv.Key == "PATH" {
			str = fmt.Sprintf("export %s=%s:$%s\n", kv.Key, kv.Value, kv.Key)
		} else {
			str = fmt.Sprintf("export %s=%s\n", kv.Key, kv.Value)
		}
		if _, err := file.WriteString(str); err != nil {
			return err
		}
	}
	return nil
}

func (m *macosEnvManager) Get(key string) (string, error) {
	line, err := m.checkEnvKey(key)
	if err != nil {
		return "", err
	}
	if line == "" {
		return "", fmt.Errorf("key %s not set", key)
	}
	return strings.Split(line, "=")[1], nil

}

func (m *macosEnvManager) checkEnvKey(key string) (string, error) {
	file, err := os.Open(m.sdkEnvPath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	key = fmt.Sprintf("export %s=", key)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, key) {
			return line, nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", nil
}

func NewEnvManager(vfConfigPath, sdkCachePath, sdkName string) (Manager, error) {
	shellInfo, err := NewShellInfo()
	if err != nil {
		return nil, err
	}
	manager := &macosEnvManager{
		shellInfo:  shellInfo,
		sdkEnvPath: filepath.Join(sdkCachePath, sdkName, "env.sh"),
		vfEnvPath:  filepath.Join(vfConfigPath, "env.sh"),
	}
	if err := appendEnvSourceIfNotExist(manager.shellInfo.ConfigPath, manager.vfEnvPath); err != nil {
		return nil, err
	}
	return manager, nil
}

func NewShellInfo() (*ShellInfo, error) {
	// 获取当前用户
	currentUser, err := user.Current()
	if err != nil {
		return nil, err
	}
	shellPath := os.Getenv("SHELL")
	shell := filepath.Base(shellPath)
	var info *ShellInfo
	switch ShellType(shell) {
	case BASH:
		info = &ShellInfo{
			ShellType:  BASH,
			ShellPath:  shellPath,
			ConfigPath: filepath.Join(currentUser.HomeDir, ".bashrc"),
		}
	case ZSH:
		info = &ShellInfo{
			ShellType:  ZSH,
			ShellPath:  shellPath,
			ConfigPath: filepath.Join(currentUser.HomeDir, ".zshrc"),
		}
	//case KSH:
	//	info = &ShellInfo{
	//		ShellType:  shellType,
	//		ConfigPath: filepath.Join(currentUser.HomeDir, ".kshrc"),
	//	}
	default:
		return nil, fmt.Errorf("unsupported shell type")
	}
	return info, nil

}

func appendEnvSourceIfNotExist(parentEnvPath, childEnvPath string) error {
	shellConfigFile, err := os.Open(parentEnvPath)
	if err != nil {
		return err
	}
	defer shellConfigFile.Close()
	command := fmt.Sprintf("source %s", childEnvPath)
	stat, _ := os.Stat(parentEnvPath)
	if stat.Size() > 0 {
		scanner := bufio.NewScanner(shellConfigFile)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.Contains(line, command) {
				return nil
			}
		}
		if err := scanner.Err(); err != nil {
			return err
		}
	}

	file, err := os.OpenFile(parentEnvPath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.WriteString("\n" + command + "\n")
	return err
}
