/*
 *    Copyright 2026 Han Li and contributors
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
	"encoding/base64"
	"fmt"
	"regexp"
	"strings"

	"github.com/version-fox/vfox/internal/env"
)

// Based on https://github.com/direnv/direnv/blob/master/internal/cmd/shell_pwsh.go
type pwsh struct{}

// Pwsh shell instance
var Pwsh Shell = pwsh{}

// hook is a template. {{.EnvContent}} is the export of the environment, and
// {{.SelfPath}} stands for the vfox executable: Activate puts that in as a
// PowerShell expression, see PowerShellEscape.
const hook = `
{{.EnvContent}}

<#
Due to a bug in PowerShell, we have to cleanup first when the shell open.
#>
& {{.SelfPath}} env --cleanup 2>$null | Out-Null;

$env:__VFOX_PID = $pid;
$env:__VFOX_SHELL = 'pwsh';

$OutputEncoding = [console]::InputEncoding = [console]::OutputEncoding = [Text.UTF8Encoding]::UTF8;

if (-not $env:__VFOX_INITIALIZED) {
	$env:__VFOX_INITIALIZED = '1';
	$originalPrompt = $function:prompt;
	function prompt {
		$export = & {{.SelfPath}} env -s pwsh;
		if ($export) {
			Invoke-Expression -Command $export;
		}
		&$originalPrompt;	
	}
		
	Register-EngineEvent -SourceIdentifier VfoxCleanup -Action {
        & {{.SelfPath}} env --cleanup
    } | Out-Null;
}
`

func (sh pwsh) Activate(config ActivateConfig) (string, error) {
	// text/template reads "{{" in a path as the start of an action. A path that
	// has it takes the encoded form, which has no braces.
	self := PowerShellEscape(config.SelfPath)
	if strings.Contains(config.SelfPath, "{{") {
		self = powerShellUTF8(config.SelfPath)
	}
	return strings.ReplaceAll(hook, "{{.SelfPath}}", self), nil
}

func (sh pwsh) Export(e env.Vars) (out string) {
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
	if !isEnvName(key) {
		return fmt.Sprintf("[System.Environment]::SetEnvironmentVariable(%s,%s);", PowerShellEscape(key), PowerShellEscape(value))
	}
	return fmt.Sprintf("$env:%s=%s;", key, PowerShellEscape(value))
}

func (sh pwsh) unset(key string) string {
	if !isEnvName(key) {
		return fmt.Sprintf("[System.Environment]::SetEnvironmentVariable(%s,$null);", PowerShellEscape(key))
	}
	return fmt.Sprintf("Remove-Item -Path 'env:/%s';", key)
}

// envName matches the names that $env: can spell without quotes.
var envName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func isEnvName(key string) bool {
	return envName.MatchString(key)
}

// PowerShellEscape returns a PowerShell expression that evaluates to str.
//
// PowerShell decodes the output of a native command with the console's code
// page, and the hook can switch the console to UTF-8 only after the output of
// `vfox activate` has been decoded. On many Windows systems that page is not
// UTF-8, so the output must be ASCII. Printable ASCII becomes a single-quoted
// literal, in which the only escape is a doubled quote. Anything else becomes
// the base64 of its UTF-8 bytes, which PowerShell decodes when the expression runs.
func PowerShellEscape(str string) string {
	if isPrintableASCII(str) {
		return "'" + strings.ReplaceAll(str, "'", "''") + "'"
	}
	return powerShellUTF8(str)
}

// powerShellUTF8 is the expression that decodes the base64 of the UTF-8 bytes of str.
func powerShellUTF8(str string) string {
	return "([System.Text.Encoding]::UTF8.GetString([System.Convert]::FromBase64String('" +
		base64.StdEncoding.EncodeToString([]byte(str)) + "')))"
}

// isPrintableASCII reports whether every byte of str is a printable ASCII character.
func isPrintableASCII(str string) bool {
	for i := 0; i < len(str); i++ {
		if str[i] < 0x20 || str[i] > 0x7e {
			return false
		}
	}
	return true
}
