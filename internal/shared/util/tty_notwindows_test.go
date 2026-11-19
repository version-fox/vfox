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

package util

import (
	"os"
	"testing"
)

// Off Windows the key reader reads /dev/tty, and no pipe is a pseudo terminal.
func TestKeysAndPtysOffWindows(t *testing.T) {
	if !CanReadKeys() {
		t.Error("CanReadKeys() = false off Windows, want true")
	}
	if !canReadKeysFrom(os.Stdin.Fd()) {
		t.Error("canReadKeysFrom() = false off Windows, want true")
	}
	if isCygwinTerminal(os.Stdout.Fd()) {
		t.Error("isCygwinTerminal() = true off Windows, want false")
	}
}
