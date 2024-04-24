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

package printer

import (
	"fmt"

	"github.com/pterm/pterm"
)

// show {message} (y/n)
// return true if user press y or Enter, otherwise false
func Promptf(message string, a ...any) bool {
	result, _ := pterm.DefaultInteractiveConfirm.Show(fmt.Sprintf(message, a...))

	// Print the user's answer in a formatted way.
	pterm.Printfln("You answered: %s", boolToText(result))

	return result
}

// boolToText converts a boolean value to a colored text.
// If the value is true, it returns a green "Yes".
// If the value is false, it returns a red "No".
func boolToText(b bool) string {
	if b {
		return pterm.Green("Yes")
	}
	return pterm.Red("No")
}
