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

package commands

import (
	"bytes"
	"os"
	"testing"
)

func TestSelectVersionByLineReportsReadError(t *testing.T) {
	var out bytes.Buffer
	version, err := selectVersionByLine(failingReader{}, &out, []string{"18.20.8"}, "nodejs")
	if err == nil {
		t.Fatalf("selectVersionByLine() = %q, want an error for a failing read", version)
	}
}

// Without key presses chooseVersion reads the choice from stdin, so the test
// gives stdin a pipe that holds the choice.
func TestChooseVersionWithoutKeysReadsALine(t *testing.T) {
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		reader.Close()
		writer.Close()
	})
	if _, err := writer.WriteString("20.19.0\n"); err != nil {
		t.Fatal(err)
	}
	previous := os.Stdin
	os.Stdin = reader
	t.Cleanup(func() { os.Stdin = previous })

	version, err := chooseVersion(false, []string{"18.20.8", "20.19.0"}, "nodejs")
	if err != nil {
		t.Fatalf("chooseVersion() error: %v", err)
	}
	if version != "20.19.0" {
		t.Errorf("chooseVersion() = %q, want %q", version, "20.19.0")
	}
}
