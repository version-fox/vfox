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

package shell

import "strings"

type Type string

//type Shell struct {
//	Type
//	ShellPath  string
//	ConfigPath string
//}

type Process interface {
	Open() error
}

type Envs map[string]*string

type Shell interface {
	Activate() (string, error)
	Export(envs Envs) string
}

func NewShell(name string) Shell {
	switch strings.ToLower(name) {
	case "bash":
		return Bash
	}
	return nil

}
