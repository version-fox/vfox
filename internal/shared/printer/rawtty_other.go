//go:build !windows

// Copyright 2026 vfox project contributors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package printer

import (
	"errors"
	"os"
)

var errNoRawMode = errors.New("the raw mode is switched with stty, on Windows only")

// rawModeAvailable reports false: off Windows the keyboard library reads the
// terminal, so no pipe needs raw mode.
func rawModeAvailable() bool {
	return false
}

// enterRawMode is not reached off Windows, where the keyboard library reads key
// presses itself.
func enterRawMode(*os.File) (func(), error) {
	return nil, errNoRawMode
}
