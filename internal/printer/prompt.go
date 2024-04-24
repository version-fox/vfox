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

	"atomicgo.dev/keyboard"

	"atomicgo.dev/keyboard/keys"
)

// show {message} (y/n)
// return true if user press y or Enter, otherwise false
func Prompt(message string) (bool, error) {
	fmt.Println(message + " (y/n)")

	result := false

	err := keyboard.Listen(func(key keys.Key) (stop bool, err error) {
		switch key.Code {
		case keys.CtrlC:
			{
				return true, nil // Stop listener by returning true on Ctrl+C
			}
		case keys.RuneKey:
			if key.String() == "y" {
				fmt.Printf("\rYou pressed: %s\n", key)
				result = true
				return true, nil
			}
			if key.String() == "n" {
				fmt.Printf("\rYou pressed: %s\n", key)
				result = false
				return true, nil
			}
		case keys.Escape:
			result = false
			return true, nil
		case keys.Enter:
			result = true
			return true, nil
		}

		return false, nil
	})

	if err != nil {
		return false, err
	}

	return result, nil
}
