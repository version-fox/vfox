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
	"fmt"
	"strings"
	"testing"

	"github.com/version-fox/vfox/internal/shared/util"
)

func lineSelectOptions() []*KV {
	return []*KV{
		{Key: "24.1.0", Value: "24.1.0 (Current)"},
		{Key: "22.15.0", Value: "22.15.0 (LTS Jod)"},
		{Key: "20.19.0", Value: "20.19.0 (LTS Iron)"},
		{Key: "18.20.8", Value: "18.20.8 (LTS Hydrogen)"},
	}
}

// runLineSelect shows a LineSelect fed with input and returns its choice and output.
func runLineSelect(t *testing.T, input string, options []*KV, disabled util.Set[string]) (*KV, string, error) {
	t.Helper()
	var out bytes.Buffer
	s := &LineSelect{
		TopText:         "Please select a version of nodejs",
		Options:         options,
		DisabledOptions: disabled,
		In:              strings.NewReader(input),
		Out:             &out,
	}
	chosen, err := s.Show()
	return chosen, out.String(), err
}

func assertChosen(t *testing.T, chosen *KV, err error, wantKey string) {
	t.Helper()
	if err != nil {
		t.Fatalf("Show() error: %v", err)
	}
	if chosen == nil {
		t.Fatalf("Show() = nil, want %q", wantKey)
	}
	if chosen.Key != wantKey {
		t.Errorf("Show() chose %q, want %q", chosen.Key, wantKey)
	}
}

func TestLineSelectChoosesByNumber(t *testing.T) {
	chosen, _, err := runLineSelect(t, "2\n", lineSelectOptions(), nil)
	assertChosen(t, chosen, err, "22.15.0")
}

func TestLineSelectAcceptsCarriageReturnLineEnding(t *testing.T) {
	chosen, _, err := runLineSelect(t, "3\r\n", lineSelectOptions(), nil)
	assertChosen(t, chosen, err, "20.19.0")
}

func TestLineSelectChoosesByKey(t *testing.T) {
	chosen, _, err := runLineSelect(t, "18.20.8\n", lineSelectOptions(), nil)
	assertChosen(t, chosen, err, "18.20.8")
}

func TestLineSelectVersionWinsOverPosition(t *testing.T) {
	// "2" is the version of the first entry, so it chooses that entry and not
	// the second one.
	options := []*KV{
		{Key: "2", Value: "2"},
		{Key: "1.0.0", Value: "1.0.0"},
	}
	chosen, _, err := runLineSelect(t, "2\n", options, nil)
	assertChosen(t, chosen, err, "2")
}

func TestLineSelectNumberWithoutNewlineAtEndOfInput(t *testing.T) {
	chosen, _, err := runLineSelect(t, "2", lineSelectOptions(), nil)
	assertChosen(t, chosen, err, "22.15.0")
}

func TestLineSelectFilterThenNumberChoosesFromFilteredList(t *testing.T) {
	// "lts" leaves three matches, so 2 is 20.19.0 in the filtered list.
	chosen, out, err := runLineSelect(t, "lts\n2\n", lineSelectOptions(), nil)
	assertChosen(t, chosen, err, "20.19.0")
	filtered := "  1) 22.15.0 (LTS Jod)\n  2) 20.19.0 (LTS Iron)\n  3) 18.20.8 (LTS Hydrogen)\n"
	if !strings.Contains(out, "[filter: lts]") || !strings.Contains(out, filtered) {
		t.Errorf("filter lts did not list the three LTS versions:\n%s", out)
	}
}

func TestLineSelectEmptyLineClearsFilterThenCancels(t *testing.T) {
	chosen, out, err := runLineSelect(t, "lts\n\n\n", lineSelectOptions(), nil)
	if err != nil {
		t.Fatalf("Show() error: %v", err)
	}
	if chosen != nil {
		t.Errorf("Show() chose %q, want cancel", chosen.Key)
	}
	// The full list is drawn at the start and again after the filter is cleared.
	if got := strings.Count(out, "24.1.0 (Current)"); got != 2 {
		t.Errorf("full list drawn %d times, want 2:\n%s", got, out)
	}
}

func TestLineSelectCancelsOnEndOfInput(t *testing.T) {
	chosen, _, err := runLineSelect(t, "", lineSelectOptions(), nil)
	if err != nil {
		t.Fatalf("Show() error: %v", err)
	}
	if chosen != nil {
		t.Errorf("Show() chose %q at end of input, want nil", chosen.Key)
	}
}

func TestLineSelectRefusesDisabledOption(t *testing.T) {
	disabled := util.NewSet[string]()
	disabled.Add("22.15.0")
	chosen, out, err := runLineSelect(t, "2\n3\n", lineSelectOptions(), disabled)
	assertChosen(t, chosen, err, "20.19.0")
	if !strings.Contains(out, "22.15.0 is already installed") {
		t.Errorf("output does not refuse the installed version:\n%s", out)
	}
	if !strings.Contains(out, "22.15.0 (LTS Jod) (installed)") {
		t.Errorf("output does not mark the installed version:\n%s", out)
	}
}

func TestLineSelectReportsEmptyFilter(t *testing.T) {
	chosen, out, err := runLineSelect(t, "zz\n", lineSelectOptions(), nil)
	if err != nil {
		t.Fatalf("Show() error: %v", err)
	}
	if chosen != nil {
		t.Errorf("Show() chose %q, want cancel", chosen.Key)
	}
	if !strings.Contains(out, "No version matches the filter.") {
		t.Errorf("output does not report the empty filter:\n%s", out)
	}
}

func TestLineSelectListsAtMostLimitAndSaysHowMany(t *testing.T) {
	var options []*KV
	for i := 0; i < lineSelectLimit+5; i++ {
		version := fmt.Sprintf("1.%d.0", i)
		options = append(options, &KV{Key: version, Value: version})
	}
	chosen, out, err := runLineSelect(t, "\n", options, nil)
	if err != nil || chosen != nil {
		t.Fatalf("Show() = %v, %v, want cancel", chosen, err)
	}
	if !strings.Contains(out, fmt.Sprintf("  %d) 1.%d.0\n", lineSelectLimit, lineSelectLimit-1)) {
		t.Errorf("last listed entry missing:\n%s", out)
	}
	if strings.Contains(out, fmt.Sprintf("  %d) ", lineSelectLimit+1)) {
		t.Errorf("more than %d entries listed:\n%s", lineSelectLimit, out)
	}
	if !strings.Contains(out, "... 5 more") {
		t.Errorf("output does not say how many are left:\n%s", out)
	}
}
