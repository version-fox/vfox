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

import (
	"fmt"
	"github.com/version-fox/vfox/internal/env"
	"regexp"
)

// Based on https://github.com/direnv/direnv/blob/master/internal/cmd/shell_pwsh.go
type pwsh struct{}

// Pwsh shell instance
var Pwsh Shell = pwsh{}

const hook = `using namespace System;
using namespace System.Management.Automation;

$hook = [EventHandler[LocationChangedEventArgs]] {
  param([object] $source, [LocationChangedEventArgs] $eventArgs)
  end {
    $export = {{.SelfPath}} env -s pwsh;
    if ($export) {
      Invoke-Expression -Command $export;
    }
  }
};
$currentAction = $ExecutionContext.SessionState.InvokeCommand.LocationChangedAction;
if ($currentAction) {
  $ExecutionContext.SessionState.InvokeCommand.LocationChangedAction = [Delegate]::Combine($currentAction, $hook);
}
else {
  $ExecutionContext.SessionState.InvokeCommand.LocationChangedAction = $hook;
};
`

func (sh pwsh) Activate() (string, error) {
	return hook, nil
}

func (sh pwsh) Export(e env.Envs) (out string) {
	for key, value := range e {
		if value == nil {
			out += sh.unset(key)
		} else {
			out += sh.export(key, *value)
		}
	}
	return out
}

func (sh pwsh) export(key, value string) string {
	value = sh.escape(value)
	if !regexp.MustCompile(`'.*'`).MatchString(value) {
		value = fmt.Sprintf("'%s'", value)
	}
	return fmt.Sprintf("$env:%s=%s;", sh.escape(key), value)
}

func (sh pwsh) unset(key string) string {
	return fmt.Sprintf("Remove-Item -Path 'env:/%s';", sh.escape(key))
}

func (pwsh) escape(str string) string {
	return PowerShellEscape(str)
}

func PowerShellEscape(str string) string {
	if str == "" {
		return "''"
	}
	in := []byte(str)
	out := ""
	i := 0
	l := len(in)
	escape := false

	hex := func(char byte) {
		escape = true
		out += fmt.Sprintf("\\x%02x", char)
	}

	backslash := func(char byte) {
		escape = true
		out += string([]byte{BACKTICK, char})
	}

	escaped := func(str string) {
		escape = true
		out += str
	}

	quoted := func(char byte) {
		escape = true
		out += string([]byte{char})
	}

	literal := func(char byte) {
		out += string([]byte{char})
	}

	for i < l {
		char := in[i]
		switch {
		case char == ACK:
			hex(char)
		case char == TAB:
			escaped("`t")
		case char == LF:
			escaped("`n")
		case char == CR:
			escaped("`r")
		case char <= US:
			hex(char)
		// case char <= AMPERSTAND:
		// 	quoted(char)
		case char == SINGLE_QUOTE:
			backslash(char)
		case char <= PLUS:
			quoted(char)
		case char <= NINE:
			literal(char)
		// case char <= QUESTION:
		// 	quoted(char)
		case char <= UPPERCASE_Z:
			literal(char)
		// case char == OPEN_BRACKET:
		// 	quoted(char)
		// case char == BACKSLASH:
		// 	quoted(char)
		case char == UNDERSCORE:
			literal(char)
		// case char <= CLOSE_BRACKET:
		// 	quoted(char)
		// case char <= BACKTICK:
		// 	quoted(char)
		// case char <= TILDA:
		// 	quoted(char)
		case char == DEL:
			hex(char)
		default:
			quoted(char)
		}
		i++
	}

	if escape {
		out = "'" + out + "'"
	}

	return out
}
