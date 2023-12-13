/*
 *    Copyright 2023 [lihan aooohan@gmail.com]
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
	"atomicgo.dev/cursor"
	"atomicgo.dev/keyboard"
	"atomicgo.dev/keyboard/keys"
	"fmt"
	"github.com/lithammer/fuzzysearch/fuzzy"
	"github.com/pterm/pterm"
	"sort"
)

type PageKVSelect struct {
	index             int
	options           []*KV
	Size              int
	result            *KV
	fuzzySearchString string
	Filter            bool
	SourceFunc        func(page, size int) ([]*KV, error)
	text              string
}

type KV struct {
	Key   string
	Value string
}

func (s *PageKVSelect) changeIndex(value int) {
	s.index += value
	if s.index < 0 {
		s.index = len(s.options) - 1
	}
	if s.index > len(s.options)-1 {
		s.index = 0
	}
}

func (s *PageKVSelect) renderSelect() string {
	var content string
	if s.Filter {
		content += pterm.Sprintf("%s %s: %s\n", s.text, pterm.LightGreen("[type to search]"), s.fuzzySearchString)
	} else {
		content += pterm.Sprintf("%s:\n", s.text)
	}
	if s.options == nil || len(s.options) == 0 {
		return pterm.Sprintln("No data")
	}

	// find options that match fuzzy search string
	var optionMap = make(map[string]*KV)
	var keysArr []string
	for _, kv := range s.options {
		optionMap[kv.Key] = kv
		keysArr = append(keysArr, kv.Key)
	}
	rankedResults := fuzzy.RankFindFold(s.fuzzySearchString, keysArr)
	if len(rankedResults) != len(s.options) {
		sort.Sort(rankedResults)
	}

	for i, option := range s.options {
		if i == s.index {
			content += pterm.Sprintln(pterm.LightGreen("->"), option.Key, " ", option.Value)
		} else {
			content += pterm.Sprintln(" ", option.Key, " ", option.Value)
		}
	}
	content += pterm.Sprintln("Press ↑/↓ to select and press ←/→ to page, and press Enter to confirm")
	return content
}

func (s *PageKVSelect) loadPageData(page int) (err error) {
	s.options, err = s.SourceFunc(page, s.Size)
	s.index = 0
	return err
}

func (s *PageKVSelect) Show() (*KV, error) {

	area, err := pterm.DefaultArea.Start()
	if err != nil {
		return nil, fmt.Errorf("could not start area: %w", err)
	}
	defer area.Stop()

	page := 0
	if err := s.loadPageData(page); err != nil {
		return nil, err
	}

	area.Update(s.renderSelect())
	cursor.Hide()
	defer cursor.Show()

	err = keyboard.Listen(func(key keys.Key) (stop bool, err error) {
		switch key.Code {
		case keys.RuneKey:
			if s.Filter {
				// Fuzzy search for options
				// append to fuzzy search string
				s.fuzzySearchString += key.String()
				s.index = 0
				//p.displayedOptions = append([]string{}, p.fuzzySearchMatches[:maxHeight]...)
				area.Update(s.renderSelect())
			}
		case keys.CtrlC:
			return true, nil
		case keys.Down:
			s.changeIndex(1)
			area.Update(s.renderSelect())
		case keys.Up:
			s.changeIndex(-1)
			area.Update(s.renderSelect())
		case keys.Left:
			if page > 0 {
				page--
				if err := s.loadPageData(page); err != nil {
					return true, err
				}
				area.Update(s.renderSelect())
			}
		case keys.Right:
			page++
			if err := s.loadPageData(page); err != nil {
				return true, err
			}
			area.Update(s.renderSelect())
		default:
			return true, nil
		}
		return false, nil // Return false to continue listening
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start keyboard listener: %w", err)
	}
	return nil, nil
}
