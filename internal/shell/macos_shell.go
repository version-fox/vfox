//go:build darwin || linux

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

package shell

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"syscall"
)

const (
	BASH = Type("bash")
	ZSH  = Type("zsh")
	KSH  = Type("ksh")
	// extend shell type
)

func (i *Shell) ReOpen() error {
	err := syscall.Exec(i.ShellPath, []string{i.ShellPath}, syscall.Environ())
	if err != nil {
		fmt.Printf("Failed to exec shell, err:%s\n", err.Error())
		return err
	}
	return nil
}

func NewShell() (*Shell, error) {
	// 获取当前用户
	currentUser, err := user.Current()
	if err != nil {
		return nil, err
	}
	shellPath := os.Getenv("SHELL")
	shell := filepath.Base(shellPath)
	var info *Shell
	switch Type(shell) {
	case BASH:
		info = &Shell{
			Type:       BASH,
			ShellPath:  shellPath,
			ConfigPath: filepath.Join(currentUser.HomeDir, ".bashrc"),
		}
	case ZSH:
		info = &Shell{
			Type:       ZSH,
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
