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
	"bytes"
	"strings"
	"testing"

	"github.com/pterm/pterm"
)

func TestLineConfirmAnswers(t *testing.T) {
	cases := []struct {
		name         string
		input        string
		defaultValue bool
		want         bool
	}{
		{"y", "y\n", false, true},
		{"yes word", "Yes\n", false, true},
		{"n", "n\n", true, false},
		{"no word", "NO\n", true, false},
		{"enter takes default yes", "\n", true, true},
		{"enter takes default no", "\n", false, false},
		{"invalid answer asks again", "maybe\ny\n", false, true},
		{"end of input is no", "", true, false},
		{"invalid answer then end of input is no", "maybe", true, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := pterm.InteractiveConfirmPrinter{
				DefaultValue: tc.defaultValue,
				DefaultText:  "Please confirm",
				ConfirmText:  "Yes",
				RejectText:   "No",
				Delimiter:    ": ",
			}
			var out bytes.Buffer
			got, err := lineConfirm(strings.NewReader(tc.input), &out, p, "Install nodejs?")
			if err != nil {
				t.Fatalf("lineConfirm() error: %v", err)
			}
			if got != tc.want {
				t.Errorf("lineConfirm() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestLineConfirmReportsReadError(t *testing.T) {
	p := pterm.InteractiveConfirmPrinter{ConfirmText: "Yes", RejectText: "No", Delimiter: ": "}
	var out bytes.Buffer
	if _, err := lineConfirm(brokenReader{}, &out, p, "Install?"); err == nil {
		t.Fatal("lineConfirm() error = nil for a failing read, want an error")
	}
}

func TestConfirmWithoutKeysReadsALine(t *testing.T) {
	p := pterm.InteractiveConfirmPrinter{ConfirmText: "Yes", RejectText: "No", Delimiter: ": "}
	var out bytes.Buffer
	got, err := confirm(false, &p, "Install?", strings.NewReader("y\n"), &out)
	if err != nil || !got {
		t.Fatalf("confirm(false) = %v, %v, want true", got, err)
	}
}

func TestShortAnswer(t *testing.T) {
	for _, tc := range []struct{ text, fallback, want string }{
		{"", "y", "y"},
		{"Yes", "n", "y"},
		{"No", "y", "n"},
	} {
		if got := shortAnswer(tc.text, tc.fallback); got != tc.want {
			t.Errorf("shortAnswer(%q, %q) = %q, want %q", tc.text, tc.fallback, got, tc.want)
		}
	}
}

func TestLineConfirmShowsDefaultInUpperCase(t *testing.T) {
	p := pterm.InteractiveConfirmPrinter{ConfirmText: "Yes", RejectText: "No", Delimiter: ": "}
	var out bytes.Buffer
	if _, err := lineConfirm(strings.NewReader("\n"), &out, p, "Install nodejs?"); err != nil {
		t.Fatal(err)
	}
	if want := "Install nodejs? [y/N]: "; out.String() != want {
		t.Errorf("prompt = %q, want %q", out.String(), want)
	}

	p.DefaultValue = true
	out.Reset()
	if _, err := lineConfirm(strings.NewReader("\n"), &out, p, "Install nodejs?"); err != nil {
		t.Fatal(err)
	}
	if want := "Install nodejs? [Y/n]: "; out.String() != want {
		t.Errorf("prompt = %q, want %q", out.String(), want)
	}
}

func TestLineConfirmUsesDefaultTextWhenNoQuestion(t *testing.T) {
	p := pterm.InteractiveConfirmPrinter{DefaultText: "Please confirm", ConfirmText: "Yes", RejectText: "No", Delimiter: ": "}
	var out bytes.Buffer
	if _, err := lineConfirm(strings.NewReader("y\n"), &out, p, ""); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(out.String(), "Please confirm ") {
		t.Errorf("prompt = %q, want the default text", out.String())
	}
}
