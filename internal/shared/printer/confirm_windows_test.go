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

package printer

import (
	"os"
	"testing"

	"github.com/pterm/pterm"
)

// Git Bash in mintty gives vfox a pipe for stdin, so Confirm reads the answer as a line.
func TestConfirmReadsLineFromPipe(t *testing.T) {
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		reader.Close()
		writer.Close()
	})
	if _, err := writer.WriteString("y\r\n"); err != nil {
		t.Fatal(err)
	}
	previous := os.Stdin
	os.Stdin = reader
	t.Cleanup(func() { os.Stdin = previous })

	p := pterm.DefaultInteractiveConfirm
	confirmed, err := Confirm(&p, "Proceed?")
	if err != nil {
		t.Fatalf("Confirm() error: %v", err)
	}
	if !confirmed {
		t.Error("Confirm() = false for y, want true")
	}
}
