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

import (
	"fmt"

	"github.com/version-fox/vfox/internal/env"
)

const clinkHook = `
{{.EnvContent}}
"{{.SelfPath}}" env --cleanup > nul 2> nul
`

type clink struct{}

var Clink = clink{}

func (b clink) Activate(config ActivateConfig) (string, error) {
	return clinkHook, nil
}

func (b clink) Export(envs env.Vars) (out string) {
	for key, value := range envs {
		if value == nil {
			out += b.set(key, "")
		} else {
			out += b.set(key, *value)
		}
	}
	return
}

func (b clink) set(key, value string) string {
	return fmt.Sprintf("set \"%s=%s\"\n", key, value)
}
