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
	"strings"
	"testing"
)

func TestSelectVersionByLine(t *testing.T) {
	var out bytes.Buffer
	version, err := selectVersionByLine(strings.NewReader("2\r\n"), &out, []string{"18.20.8", "20.19.0"}, "nodejs")
	if err != nil {
		t.Fatalf("selectVersionByLine() error: %v", err)
	}
	if version != "20.19.0" {
		t.Errorf("selectVersionByLine() = %q, want %q", version, "20.19.0")
	}
	if !strings.Contains(out.String(), "Please select a version of nodejs") {
		t.Errorf("prompt does not name the SDK:\n%s", out.String())
	}
}

func TestSelectVersionByLineCancelIsAnError(t *testing.T) {
	var out bytes.Buffer
	version, err := selectVersionByLine(strings.NewReader("\n"), &out, []string{"18.20.8"}, "nodejs")
	if err == nil {
		t.Fatalf("selectVersionByLine() = %q, want an error for a cancelled selection", version)
	}
}
