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

import "fmt"

type bash struct{}

var Bash = bash{}

func (b bash) Name() string {
	return "bash"
}

func (b bash) Activate() (string, error) {
	var script = `
			export MISE_SHELL=bash
            export __MISE_ORIG_PATH="$PATH"

            vfox() {{
            }}

            _vfox_hook() {{
              local previous_exit_status=$?;
              eval "$(mise hook-env{flags} -s bash)";
              return $previous_exit_status;
            }};
            if [[ ";${{PROMPT_COMMAND:-}};" != *";_vfox_hook;"* ]]; then
              PROMPT_COMMAND="_vfox_hook${{PROMPT_COMMAND:+;$PROMPT_COMMAND}}"
            fi
`

	return script, nil
}

//https://github.com/direnv/direnv/blob/master/internal/cmd/shell_bash.go

func (b bash) Export(envs Envs) (out string) {
	for key, value := range envs {
		if value == nil {
			out += b.unset(key)
		} else {
			out += b.export(key, *value)
		}
	}
	return
}

func (b bash) export(key, value string) string {
	return "export " + b.escape(key) + "=" + b.escape(value) + ";"
}

func (b bash) unset(key string) string {
	return "unset " + b.escape(key) + ";"
}

func (b bash) escape(str string) string {
	return BashEscape(str)
}

// nolint
const (
	ACK           = 6
	TAB           = 9
	LF            = 10
	CR            = 13
	US            = 31
	SPACE         = 32
	AMPERSTAND    = 38
	SINGLE_QUOTE  = 39
	PLUS          = 43
	NINE          = 57
	QUESTION      = 63
	UPPERCASE_Z   = 90
	OPEN_BRACKET  = 91
	BACKSLASH     = 92
	UNDERSCORE    = 95
	CLOSE_BRACKET = 93
	BACKTICK      = 96
	LOWERCASE_Z   = 122
	TILDA         = 126
	DEL           = 127
)

// https://github.com/solidsnack/shell-escape/blob/master/Text/ShellEscape/Bash.hs
/*
A Bash escaped string. The strings are wrapped in @$\'...\'@ if any
bytes within them must be escaped; otherwise, they are left as is.
Newlines and other control characters are represented as ANSI escape
sequences. High bytes are represented as hex codes. Thus Bash escaped
strings will always fit on one line and never contain non-ASCII bytes.
*/
func BashEscape(str string) string {
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
		out += string([]byte{BACKSLASH, char})
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
			escaped(`\t`)
		case char == LF:
			escaped(`\n`)
		case char == CR:
			escaped(`\r`)
		case char <= US:
			hex(char)
		case char <= AMPERSTAND:
			quoted(char)
		case char == SINGLE_QUOTE:
			backslash(char)
		case char <= PLUS:
			quoted(char)
		case char <= NINE:
			literal(char)
		case char <= QUESTION:
			quoted(char)
		case char <= UPPERCASE_Z:
			literal(char)
		case char == OPEN_BRACKET:
			quoted(char)
		case char == BACKSLASH:
			backslash(char)
		case char == UNDERSCORE:
			literal(char)
		case char <= CLOSE_BRACKET:
			quoted(char)
		case char <= BACKTICK:
			quoted(char)
		case char <= TILDA:
			quoted(char)
		case char == DEL:
			hex(char)
		default:
			hex(char)
		}
		i++
	}

	if escape {
		out = "$'" + out + "'"
	}

	return out
}
