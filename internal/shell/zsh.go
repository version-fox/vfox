/*
 *    Copyright 2025 Han Li and contributors
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

import "github.com/version-fox/vfox/internal/env"

// Based on https://github.com/direnv/direnv/blob/master/internal/cmd/shell_zsh.go

type zsh struct{}

var Zsh = zsh{}

const zshHook = `
if [[ -z "$__VFOX_PID" || -z "$__VFOX_SHELL" ]]; then
  {{.EnvContent}}

  export __VFOX_PID=$$;

  _vfox_hook() {
    trap -- '' SIGINT;
    eval "$("{{.SelfPath}}" env -s zsh)";
    trap - SIGINT;
  }
  typeset -ag precmd_functions;
  if [[ -z "${precmd_functions[(r)_vfox_hook]+1}" ]]; then
    precmd_functions=( _vfox_hook ${precmd_functions[@]} )
  fi
  typeset -ag chpwd_functions;
  if [[ -z "${chpwd_functions[(r)_vfox_hook]+1}" ]]; then
    chpwd_functions=( _vfox_hook ${chpwd_functions[@]} )
  fi

  trap 'vfox env --cleanup' EXIT
fi
`

func (z zsh) Activate(config ActivateConfig) (string, error) {
	return zshHook, nil
}

func (z zsh) Export(envs env.Vars) (out string) {
	for key, value := range envs {
		if value == nil {
			out += z.unset(key)
		} else {
			out += z.export(key, *value)
		}
	}
	return out
}

func (z zsh) export(key, value string) string {
	return "export " + z.escape(key) + "=" + z.escape(value) + ";"
}

func (z zsh) unset(key string) string {
	return "unset " + z.escape(key) + ";"
}

func (z zsh) escape(str string) string {
	return BashEscape(str)
}
