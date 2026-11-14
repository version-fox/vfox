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
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/pterm/pterm"
	"github.com/version-fox/vfox/internal/shared/util"
)

// keyHint says what the keys do, under the list.
const keyHint = "Up and Down move, typing filters the list, Enter chooses, Ctrl-C cancels"

// KeySelect chooses one of Options by key presses. The arrow keys move the
// highlight, typed text filters the list, Enter chooses, and Ctrl-C cancels. It
// serves a terminal that delivers key presses in raw mode (see Prompt.Choose).
type KeySelect struct {
	TopText string
	Options []*KV
	// DisabledOptions are listed as installed and cannot be chosen. It may be nil.
	DisabledOptions util.Set[string]
	// Terminal, when set, is put in raw mode for the selection and restored after it.
	Terminal *os.File
	In       io.Reader
	Out      io.Writer
}

// Show lists the options and reads key presses until one is chosen. It returns
// nil when the selection is cancelled or the input ends.
func (s *KeySelect) Show() (*KV, error) {
	if s.Terminal != nil {
		restore, err := enterRawMode(s.Terminal)
		if err != nil {
			return nil, fmt.Errorf("raw mode: %w", err)
		}
		defer restore()
	}
	input := bufio.NewReader(s.In)
	filter, notice := "", ""
	highlight, drawn := 0, 0
	for {
		matches := fuzzyMatches(s.Options, filter)
		if highlight >= len(matches) {
			highlight = 0
		}
		drawn = s.draw(drawn, filter, matches, highlight, notice)
		notice = ""

		press, err := readKey(input)
		if err == io.EOF {
			return nil, nil
		}
		if err != nil {
			return nil, fmt.Errorf("read selection: %w", err)
		}
		switch press.kind {
		case keyUp:
			if n := len(matches); n > 0 {
				highlight = (highlight + n - 1) % n
			}
		case keyDown:
			if n := len(matches); n > 0 {
				highlight = (highlight + 1) % n
			}
		case keyRune:
			filter += string(press.r)
			highlight = 0
		case keyBackspace:
			if runes := []rune(filter); len(runes) > 0 {
				filter = string(runes[:len(runes)-1])
			}
			highlight = 0
		case keyEnter:
			if len(matches) == 0 {
				continue
			}
			chosen := matches[highlight]
			if isDisabledIn(s.DisabledOptions, chosen.Key) {
				notice = chosen.Key + " is already installed, choose another version."
				continue
			}
			return chosen, nil
		case keyCancel:
			return nil, nil
		}
	}
}

// draw writes the title, the matches around the highlight, the notice and the
// hint. It starts over the block drawn last time, which is drawn lines long, and
// it returns the length of the block it wrote.
func (s *KeySelect) draw(drawn int, filter string, matches []*KV, highlight int, notice string) int {
	var b strings.Builder
	if drawn > 0 {
		fmt.Fprintf(&b, "\x1b[%dA", drawn)
	}
	b.WriteString("\r\x1b[J")

	var lines []string
	if filter == "" {
		lines = append(lines, s.TopText+":")
	} else {
		lines = append(lines, fmt.Sprintf("%s [filter: %s]:", s.TopText, filter))
	}
	if len(matches) == 0 {
		lines = append(lines, "  No version matches the filter.")
	}
	start := 0
	if highlight >= lineSelectLimit {
		start = highlight - lineSelectLimit + 1
	}
	end := min(start+lineSelectLimit, len(matches))
	for i := start; i < end; i++ {
		marker := "   "
		if i == highlight {
			marker = "-> "
		}
		label := matches[i].Value
		if isDisabledIn(s.DisabledOptions, matches[i].Key) {
			label += " (installed)"
		}
		lines = append(lines, marker+label)
	}
	if more := len(matches) - end; more > 0 {
		lines = append(lines, fmt.Sprintf("   ... %d more, type part of a version to narrow the list", more))
	}
	if notice != "" {
		lines = append(lines, notice)
	}
	lines = append(lines, keyHint)
	for _, line := range lines {
		b.WriteString(line)
		b.WriteString("\n")
	}
	fmt.Fprint(s.Out, b.String())
	return len(lines)
}

// confirmKeys asks a yes-or-no question with key presses, on a terminal in raw
// mode when terminal is set: y or n settles it, Enter gives the default, and
// Ctrl-C refuses.
func confirmKeys(terminal *os.File, in io.Reader, out io.Writer, p pterm.InteractiveConfirmPrinter, text string) (bool, error) {
	if terminal != nil {
		restore, err := enterRawMode(terminal)
		if err != nil {
			return false, fmt.Errorf("raw mode: %w", err)
		}
		defer restore()
	}
	if text == "" {
		text = p.DefaultText
	}
	yes, no := shortAnswer(p.ConfirmText, "y"), shortAnswer(p.RejectText, "n")
	shownYes, shownNo := yes, no
	if p.DefaultValue {
		shownYes = strings.ToUpper(yes)
	} else {
		shownNo = strings.ToUpper(no)
	}
	fmt.Fprintf(out, "%s [%s/%s]%s", text, shownYes, shownNo, p.Delimiter)

	input := bufio.NewReader(in)
	for {
		press, err := readKey(input)
		if err == io.EOF {
			fmt.Fprintln(out)
			return false, nil
		}
		if err != nil {
			return false, fmt.Errorf("read answer: %w", err)
		}
		switch press.kind {
		case keyEnter:
			return p.DefaultValue, answered(out, p, p.DefaultValue)
		case keyCancel:
			return false, answered(out, p, false)
		case keyRune:
			switch strings.ToLower(string(press.r)) {
			case yes:
				return true, answered(out, p, true)
			case no:
				return false, answered(out, p, false)
			}
		}
	}
}

// answered shows the answer that was given, and returns the error of writing it.
func answered(out io.Writer, p pterm.InteractiveConfirmPrinter, value bool) error {
	text := p.RejectText
	if value {
		text = p.ConfirmText
	}
	_, err := fmt.Fprintln(out, text)
	return err
}
