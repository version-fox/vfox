/*
 *    Copyright 2024 Han Li and contributors
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
	"strings"

	"github.com/version-fox/vfox/internal/env"
)

// Based on https://github.com/direnv/direnv/blob/master/internal/cmd/shell_fish.go
type fish struct{}

// Fish adds support for the fish shell as a host
var Fish Shell = fish{}

const fishHook = `
{{.EnvContent}}
    function __vfox_export_eval --on-event fish_prompt;
        "{{.SelfPath}}" env -s fish | source;

        if test "$vfox_fish_mode" != "disable_arrow";
            function __vfox_cd_hook --on-variable PWD;
                if test "$vfox_fish_mode" = "eval_after_arrow";
                    set -g __vfox_export_again 0;
                else;
                    "{{.SelfPath}}" env -s fish | source;
                end;
            end;
        end;
    end;

    function __vfox_export_eval_2 --on-event fish_preexec;
        if set -q __vfox_export_again;
            set -e __vfox_export_again;
            "{{.SelfPath}}" env -s fish | source;
            echo;
        end;

        functions --erase __vfox_cd_hook;
    end;
	function cleanup_on_exit --on-process-exit %self

		"{{.SelfPath}}" env --cleanup
	end;
`

func (sh fish) Activate() (string, error) {
	return fishHook, nil
}

func (sh fish) Export(e env.Envs) (out string) {
	for key, value := range e {
		if value == nil {
			out += sh.unset(key)
		} else {
			out += sh.export(key, *value)
		}
	}
	return out
}

func (sh fish) export(key, value string) string {
	if key == "PATH" {
		command := "set -x -g PATH"
		for _, path := range strings.Split(value, ":") {
			command += " " + sh.escape(path)
		}
		return command + ";"
	}
	return "set -x -g " + sh.escape(key) + " " + sh.escape(value) + ";"
}

func (sh fish) unset(key string) string {
	return "set -e -g " + sh.escape(key) + ";"
}

func (sh fish) escape(str string) string {
	in := []byte(str)
	out := "'"
	i := 0
	l := len(in)

	hex := func(char byte) {
		out += fmt.Sprintf("'\\X%02x'", char)
	}

	backslash := func(char byte) {
		out += string([]byte{BACKSLASH, char})
	}

	escaped := func(str string) {
		out += "'" + str + "'"
	}

	literal := func(char byte) {
		out += string([]byte{char})
	}

	for i < l {
		char := in[i]
		switch {
		case char == TAB:
			escaped(`\t`)
		case char == LF:
			escaped(`\n`)
		case char == CR:
			escaped(`\r`)
		case char <= US:
			hex(char)
		case char == SINGLE_QUOTE:
			backslash(char)
		case char == BACKSLASH:
			backslash(char)
		case char <= TILDA:
			literal(char)
		case char == DEL:
			hex(char)
		default:
			hex(char)
		}
		i++
	}

	out += "'"

	return out
}
