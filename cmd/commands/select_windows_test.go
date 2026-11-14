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
	"os"
	"testing"
	"time"
)

// Git Bash in mintty gives vfox a pipe for stdin, not a console. The key reader
// cannot read from such a pipe, so the choice typed into it must still be read.
func TestSelectFromOptionsReadsChoiceFromPipe(t *testing.T) {
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		reader.Close()
		writer.Close()
	})
	if _, err := writer.WriteString("2\r\n"); err != nil {
		t.Fatal(err)
	}
	previous := os.Stdin
	os.Stdin = reader
	t.Cleanup(func() { os.Stdin = previous })

	type outcome struct {
		version string
		err     error
	}
	done := make(chan outcome, 1)
	go func() {
		version, err := selectFromOptions([]string{"18.20.8", "20.19.0"}, "nodejs")
		done <- outcome{version, err}
	}()
	select {
	case got := <-done:
		if got.err != nil {
			t.Fatalf("selectFromOptions() error: %v", got.err)
		}
		if got.version != "20.19.0" {
			t.Errorf("selectFromOptions() = %q, want %q", got.version, "20.19.0")
		}
	case <-time.After(15 * time.Second):
		t.Fatal("selectFromOptions() did not return: the key reader never reads from a pipe")
	}
}
