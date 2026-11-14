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

	"github.com/version-fox/vfox/internal/shared/util"
)

// runKeySelect shows a KeySelect fed with the key presses in input.
func runKeySelect(t *testing.T, input string, options []*KV, disabled util.Set[string]) (*KV, string, error) {
	t.Helper()
	var out bytes.Buffer
	s := &KeySelect{
		TopText:         "Please select a version of nodejs",
		Options:         options,
		DisabledOptions: disabled,
		In:              strings.NewReader(input),
		Out:             &out,
	}
	chosen, err := s.Show()
	return chosen, out.String(), err
}

func TestKeySelectDownArrowChoosesTheNextEntry(t *testing.T) {
	chosen, _, err := runKeySelect(t, "\x1b[B\r", lineSelectOptions(), nil)
	assertChosen(t, chosen, err, "22.15.0")
}

func TestKeySelectUpArrowWrapsToTheLastEntry(t *testing.T) {
	chosen, _, err := runKeySelect(t, "\x1b[A\r", lineSelectOptions(), nil)
	assertChosen(t, chosen, err, "18.20.8")
}

func TestKeySelectApplicationCursorKeys(t *testing.T) {
	chosen, _, err := runKeySelect(t, "\x1bOB\x1bOB\r", lineSelectOptions(), nil)
	assertChosen(t, chosen, err, "20.19.0")
}

func TestKeySelectTypedTextFiltersThenTheArrowChooses(t *testing.T) {
	chosen, out, err := runKeySelect(t, "lts\x1b[B\r", lineSelectOptions(), nil)
	assertChosen(t, chosen, err, "20.19.0")
	if !strings.Contains(out, "[filter: lts]") {
		t.Errorf("output does not show the filter:\n%s", out)
	}
}

func TestKeySelectBackspaceRemovesTheLastCharacter(t *testing.T) {
	chosen, _, err := runKeySelect(t, "ltz\x7f\r", lineSelectOptions(), nil)
	assertChosen(t, chosen, err, "22.15.0")
}

func TestKeySelectCtrlCCancels(t *testing.T) {
	chosen, _, err := runKeySelect(t, "\x1b[B\x03", lineSelectOptions(), nil)
	if err != nil || chosen != nil {
		t.Fatalf("Show() = %v, %v, want a cancelled selection", chosen, err)
	}
}

func TestKeySelectEndOfInputCancels(t *testing.T) {
	chosen, _, err := runKeySelect(t, "", lineSelectOptions(), nil)
	if err != nil || chosen != nil {
		t.Fatalf("Show() = %v, %v, want a cancelled selection", chosen, err)
	}
}

func TestKeySelectRefusesAnInstalledVersion(t *testing.T) {
	disabled := util.NewSet[string]()
	disabled.Add("22.15.0")
	chosen, out, err := runKeySelect(t, "\x1b[B\r\x1b[B\r", lineSelectOptions(), disabled)
	assertChosen(t, chosen, err, "20.19.0")
	if !strings.Contains(out, "22.15.0 is already installed") {
		t.Errorf("output does not refuse the installed version:\n%s", out)
	}
	if !strings.Contains(out, "22.15.0 (LTS Jod) (installed)") {
		t.Errorf("output does not mark the installed version:\n%s", out)
	}
}

func TestKeySelectDrawsTheHighlight(t *testing.T) {
	_, out, err := runKeySelect(t, "\x1b[B\x03", lineSelectOptions(), nil)
	if err != nil {
		t.Fatalf("Show() error: %v", err)
	}
	if !strings.Contains(out, "-> 22.15.0 (LTS Jod)") {
		t.Errorf("the highlight does not move with the down arrow:\n%s", out)
	}
}

func TestKeySelectRedrawsOverTheBlock(t *testing.T) {
	// Each key press redraws the block: the cursor goes back over the lines of
	// the block drawn last time.
	_, out, err := runKeySelect(t, "\x1b[B\x03", lineSelectOptions(), nil)
	if err != nil {
		t.Fatalf("Show() error: %v", err)
	}
	if !strings.Contains(out, "\x1b[") || !strings.Contains(out, "A\r\x1b[J") {
		t.Errorf("the second draw does not go back over the first:\n%q", out)
	}
}

func TestKeySelectReportsReadError(t *testing.T) {
	var out bytes.Buffer
	s := &KeySelect{TopText: "Please select a version of nodejs", Options: lineSelectOptions(), In: brokenReader{}, Out: &out}
	chosen, err := s.Show()
	if err == nil || chosen != nil {
		t.Fatalf("Show() = %v, %v, want an error for a failing read", chosen, err)
	}
}
