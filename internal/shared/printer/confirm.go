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
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/pterm/pterm"
)

// Confirm asks a yes-or-no question with p, on the terminal of the process: see
// StdinPrompt.
func Confirm(p *pterm.InteractiveConfirmPrinter, text string) (bool, error) {
	return StdinPrompt().Confirm(p, text)
}

// ConfirmFromPipe asks text and takes the answer from a piped stdin, as in
// `yes | vfox install --all`. answered is false when stdin is a terminal, empty
// or cannot be read: there is no answer to act on, so the caller can explain
// how to skip the prompt instead of silently assuming "no".
func ConfirmFromPipe(p *pterm.InteractiveConfirmPrinter, text string) (confirmed, answered bool, err error) {
	info, statErr := os.Stdin.Stat()
	if statErr != nil || info.Mode()&os.ModeCharDevice != 0 {
		return false, false, nil
	}
	return confirmFromReader(os.Stdin, os.Stdout, *p, text)
}

// confirmFromReader reads one answer line from in. It reports answered false
// when in ends before it yields a byte, distinguishing an empty pipe from a
// refusal. Peeking keeps the byte for lineConfirm to read.
func confirmFromReader(in io.Reader, out io.Writer, p pterm.InteractiveConfirmPrinter, text string) (confirmed, answered bool, err error) {
	buffered := bufio.NewReader(in)
	if _, err := buffered.Peek(1); err != nil {
		if errors.Is(err, io.EOF) {
			return false, false, nil
		}
		return false, false, err
	}
	confirmed, err = lineConfirm(buffered, out, p, text)
	return confirmed, true, err
}

// lineConfirm asks text and reads an answer line. The confirm or reject text, or
// its first letter, settles it; an empty line gives the default; an input that
// ends gives no.
func lineConfirm(in io.Reader, out io.Writer, p pterm.InteractiveConfirmPrinter, text string) (bool, error) {
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

	input := bufio.NewReader(in)
	for {
		fmt.Fprintf(out, "%s [%s/%s]%s", text, shownYes, shownNo, p.Delimiter)
		line, readErr := input.ReadString('\n')
		if readErr != nil && readErr != io.EOF {
			return false, fmt.Errorf("read answer: %w", readErr)
		}
		answer := strings.ToLower(strings.TrimSpace(line))
		switch answer {
		case "":
			if readErr == io.EOF {
				fmt.Fprintln(out)
				return false, nil
			}
			return p.DefaultValue, nil
		case yes, strings.ToLower(p.ConfirmText):
			return true, nil
		case no, strings.ToLower(p.RejectText):
			return false, nil
		}
		if readErr == io.EOF {
			return false, nil
		}
	}
}

// shortAnswer returns the lower-case first letter of text, or fallback when text is empty.
func shortAnswer(text, fallback string) string {
	runes := []rune(strings.ToLower(text))
	if len(runes) == 0 {
		return fallback
	}
	return string(runes[0])
}
