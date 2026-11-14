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

import "bufio"

// keyKind says what a key press asks for.
type keyKind int

const (
	keyOther     keyKind = iota // a key the prompts do not use
	keyRune                     // a printable character, which filters the list
	keyUp                       // the up arrow
	keyDown                     // the down arrow
	keyEnter                    // Enter
	keyBackspace                // Backspace or Delete
	keyCancel                   // Ctrl-C or Ctrl-D
)

// keyPress is one key press, read from a terminal in raw mode.
type keyPress struct {
	kind keyKind
	r    rune // the character, for keyRune
}

// readKey reads one key press from in. An arrow key arrives as an escape
// sequence: ESC [ A to ESC [ D, or ESC O A to ESC O D.
func readKey(in *bufio.Reader) (keyPress, error) {
	b, err := in.ReadByte()
	if err != nil {
		return keyPress{}, err
	}
	switch b {
	case 0x1b:
		return readEscape(in), nil
	case '\r', '\n':
		return keyPress{kind: keyEnter}, nil
	case 0x7f, 0x08:
		return keyPress{kind: keyBackspace}, nil
	case 0x03, 0x04:
		return keyPress{kind: keyCancel}, nil
	}
	if b < 0x20 {
		return keyPress{kind: keyOther}, nil
	}
	if err := in.UnreadByte(); err != nil {
		return keyPress{}, err
	}
	r, _, err := in.ReadRune()
	if err != nil {
		return keyPress{}, err
	}
	return keyPress{kind: keyRune, r: r}, nil
}

// readEscape reads what follows an escape. A sequence that is not an arrow key,
// or an escape with nothing after it, is a key the prompts do not use.
func readEscape(in *bufio.Reader) keyPress {
	next, err := in.ReadByte()
	if err != nil {
		return keyPress{kind: keyOther}
	}
	if next != '[' && next != 'O' {
		_ = in.UnreadByte()
		return keyPress{kind: keyOther}
	}
	// Parameters, such as the 1;5 of ESC [ 1 ; 5 A, come before the final byte.
	for {
		c, err := in.ReadByte()
		if err != nil {
			return keyPress{kind: keyOther}
		}
		if c >= 0x40 && c <= 0x7e {
			switch c {
			case 'A':
				return keyPress{kind: keyUp}
			case 'B':
				return keyPress{kind: keyDown}
			}
			return keyPress{kind: keyOther}
		}
	}
}
