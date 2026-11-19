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
	"io"
	"os"

	"github.com/pterm/pterm"
	"github.com/version-fox/vfox/internal/shared/util"
)

// Input says how a prompt takes its answer.
type Input int

const (
	// Keyboard reads key presses with the keyboard library: a console, or a Unix
	// terminal. The prompts of pterm do that.
	Keyboard Input = iota
	// RawKeys reads key presses from a pty that reaches the program as a pipe, such
	// as Git Bash in mintty. stty puts that terminal in raw mode for the prompt.
	RawKeys
	// Line reads a line: a pipe, a file, or a terminal that cannot be put in raw
	// mode.
	Line
)

// InputFor says how the prompts of the process take their answers.
func InputFor() Input {
	if util.CanReadKeys() {
		return Keyboard
	}
	if util.IsPtyInput() && rawModeAvailable() {
		return RawKeys
	}
	return Line
}

// Prompt says how a prompt takes its answer: from key presses or from a line, and
// from where.
type Prompt struct {
	Input Input
	// Terminal is the terminal that RawKeys puts in raw mode. It may be nil.
	Terminal *os.File
	In       io.Reader
	Out      io.Writer
}

// StdinPrompt is the prompt of the process: its answer comes from stdin, and its
// text goes to stdout.
func StdinPrompt() Prompt {
	return Prompt{Input: InputFor(), Terminal: os.Stdin, In: os.Stdin, Out: os.Stdout}
}

// Choose lets the person pick one of options. It handles RawKeys and Line; the
// Keyboard selection is the select printer of pterm, which the caller uses.
func (p Prompt) Choose(topText string, options []*KV, disabled util.Set[string]) (*KV, error) {
	if p.Input == RawKeys {
		return (&KeySelect{
			TopText:         topText,
			Options:         options,
			DisabledOptions: disabled,
			Terminal:        p.Terminal,
			In:              p.In,
			Out:             p.Out,
		}).Show()
	}
	return (&LineSelect{
		TopText:         topText,
		Options:         options,
		DisabledOptions: disabled,
		In:              p.In,
		Out:             p.Out,
	}).Show()
}

// Confirm asks a yes-or-no question. The Keyboard answer is q's own Show, which
// reads stdin itself.
func (p Prompt) Confirm(q *pterm.InteractiveConfirmPrinter, text string) (bool, error) {
	switch p.Input {
	case RawKeys:
		return confirmKeys(p.Terminal, p.In, p.Out, *q, text)
	case Line:
		return lineConfirm(p.In, p.Out, *q, text)
	default:
		return q.Show(text)
	}
}
