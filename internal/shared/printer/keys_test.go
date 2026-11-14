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
	"bufio"
	"io"
	"strings"
	"testing"
)

func TestReadKey(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  keyPress
	}{
		{"up", "\x1b[A", keyPress{kind: keyUp}},
		{"down", "\x1b[B", keyPress{kind: keyDown}},
		{"application cursor up", "\x1bOA", keyPress{kind: keyUp}},
		{"application cursor down", "\x1bOB", keyPress{kind: keyDown}},
		{"modified down", "\x1b[1;5B", keyPress{kind: keyDown}},
		{"carriage return", "\r", keyPress{kind: keyEnter}},
		{"line feed", "\n", keyPress{kind: keyEnter}},
		{"delete", "\x7f", keyPress{kind: keyBackspace}},
		{"backspace", "\x08", keyPress{kind: keyBackspace}},
		{"ctrl-c", "\x03", keyPress{kind: keyCancel}},
		{"ctrl-d", "\x04", keyPress{kind: keyCancel}},
		{"letter", "a", keyPress{kind: keyRune, r: 'a'}},
		{"multi-byte character", "é", keyPress{kind: keyRune, r: 'é'}},
		{"other control", "\x01", keyPress{kind: keyOther}},
		{"escape before a letter", "\x1ba", keyPress{kind: keyOther}},
		{"unknown sequence", "\x1b[2~", keyPress{kind: keyOther}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := readKey(bufio.NewReader(strings.NewReader(tc.input)))
			if err != nil {
				t.Fatalf("readKey() error: %v", err)
			}
			if got != tc.want {
				t.Errorf("readKey(%q) = %+v, want %+v", tc.input, got, tc.want)
			}
		})
	}
}

func TestReadKeyLeavesTheNextKey(t *testing.T) {
	// The escape that is not an arrow key leaves its following letter for the next read.
	in := bufio.NewReader(strings.NewReader("\x1ba"))
	if got, err := readKey(in); err != nil || got.kind != keyOther {
		t.Fatalf("first readKey() = %+v, %v, want an unused key", got, err)
	}
	if got, err := readKey(in); err != nil || got != (keyPress{kind: keyRune, r: 'a'}) {
		t.Errorf("second readKey() = %+v, %v, want the letter a", got, err)
	}
}

func TestReadKeyEndOfInput(t *testing.T) {
	if _, err := readKey(bufio.NewReader(strings.NewReader(""))); err != io.EOF {
		t.Errorf("readKey() error = %v, want io.EOF", err)
	}
}
