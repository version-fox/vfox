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
	"strconv"
	"strings"

	"github.com/lithammer/fuzzysearch/fuzzy"
	"github.com/version-fox/vfox/internal/shared/util"
)

// lineSelectLimit is how many matches LineSelect lists at a time.
const lineSelectLimit = 20

// LineSelect chooses one of Options by reading lines. It serves terminals that
// cannot deliver single key presses (see util.CanReadKeys), such as Git Bash in
// mintty.
type LineSelect struct {
	TopText string
	Options []*KV
	// DisabledOptions are listed as installed and cannot be chosen. It may be nil.
	DisabledOptions util.Set[string]
	In              io.Reader
	Out             io.Writer
}

// Show lists the options and reads lines until one is chosen. A number chooses
// the entry at that position in the list, the key of an option chooses that
// option, and other text filters the list. An empty line clears the filter, or
// cancels when there is none. Show returns nil when the selection is cancelled
// or the input ends.
func (s *LineSelect) Show() (*KV, error) {
	input := bufio.NewReader(s.In)
	filter := ""
	for {
		matches := s.matches(filter)
		listed := matches
		if len(listed) > lineSelectLimit {
			listed = listed[:lineSelectLimit]
		}
		s.render(filter, matches, listed)

		line, readErr := input.ReadString('\n')
		if readErr != nil && readErr != io.EOF {
			return nil, fmt.Errorf("read selection: %w", readErr)
		}
		line = strings.TrimSpace(line)
		if line == "" {
			if readErr == io.EOF || filter == "" {
				fmt.Fprintln(s.Out)
				return nil, nil
			}
			filter = ""
			continue
		}

		chosen := s.choose(line, listed)
		if chosen == nil {
			filter = line
			continue
		}
		if s.isDisabled(chosen.Key) {
			fmt.Fprintf(s.Out, "%s is already installed, choose another version.\n", chosen.Key)
			continue
		}
		return chosen, nil
	}
}

// matches returns the options whose value fuzzily contains filter, in the order
// the plugin gave them. An empty filter matches every option.
func (s *LineSelect) matches(filter string) []*KV {
	var found []*KV
	for _, option := range s.Options {
		if fuzzy.MatchFold(filter, option.Value) {
			found = append(found, option)
		}
	}
	return found
}

// choose returns the option that line picks, or nil when line is a filter. A
// number picks from the listed entries; a key picks from all options.
func (s *LineSelect) choose(line string, listed []*KV) *KV {
	if n, err := strconv.Atoi(line); err == nil && n >= 1 && n <= len(listed) {
		return listed[n-1]
	}
	for _, option := range s.Options {
		if option.Key == line {
			return option
		}
	}
	return nil
}

func (s *LineSelect) isDisabled(key string) bool {
	return s.DisabledOptions != nil && s.DisabledOptions.Contains(key)
}

// render writes the title, the listed matches numbered, and the prompt.
func (s *LineSelect) render(filter string, matches, listed []*KV) {
	if filter == "" {
		fmt.Fprintf(s.Out, "%s:\n", s.TopText)
	} else {
		fmt.Fprintf(s.Out, "%s [filter: %s]:\n", s.TopText, filter)
	}
	if len(matches) == 0 {
		fmt.Fprintln(s.Out, "  No version matches the filter.")
	}
	for i, option := range listed {
		label := option.Value
		if s.isDisabled(option.Key) {
			label += " (installed)"
		}
		fmt.Fprintf(s.Out, "  %d) %s\n", i+1, label)
	}
	if more := len(matches) - len(listed); more > 0 {
		fmt.Fprintf(s.Out, "  ... %d more, type part of a version to narrow the list\n", more)
	}
	action := "cancel"
	if filter != "" {
		action = "clear the filter"
	}
	fmt.Fprintf(s.Out, "Type a number or a version to choose it, or text to filter the list (Enter to %s): ", action)
}
