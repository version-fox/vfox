/*
 *    Copyright 2024 Han Li and contributors
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
	"os"
)

type PageKVSelect struct {
	index             int
	Options           []*KV
	searchOptions     []*KV
	pageOptions       []*KV
	Size              int
	result            *KV
	isEmpty           bool
	fuzzySearchString string
	Filter            bool
	SourceFunc        func(page, size int, options []*KV) ([]*KV, error)
	TopText           string
}

type KV struct {
	Key   string
	Value string
}

func (s *PageKVSelect) changeIndex(value int) {
	s.index += value
	if s.index < 0 {
		s.index = len(s.pageOptions) - 1
	}
	if s.index > len(s.pageOptions)-1 {
		s.index = 0
	}
}

func (s *PageKVSelect) renderSelect() string {
	var content string
	if s.Filter {
		content += pterm.Sprintf("%s %s: %s\n", s.TopText, pterm.LightGreen("[type to search]"), s.fuzzySearchString)
	} else {
		content += pterm.Sprintf("%s:\n", s.TopText)
	}
	if (s.pageOptions == nil || len(s.pageOptions) == 0) && len(s.searchOptions) > 0 {
		return pterm.Sprintln("No data")
	}

	if len(s.pageOptions) != 0 {
		s.result = s.pageOptions[s.index]
	}

	indexMapper := make([]*KV, len(s.pageOptions))
	for i := 0; i < len(s.pageOptions); i++ {
		indexMapper[i] = s.pageOptions[i]
	}

	for i, option := range indexMapper {
		if i == s.index {
			content += pterm.Sprintln(pterm.LightGreen("-> "), option.Value)
		} else {
			content += pterm.Sprintln("  ", option.Value)
		}
	}
	content += pterm.Sprintln("Press ↑/↓ to select and press ←/→ to page, and press Enter to confirm")
	return content
}

func (s *PageKVSelect) search() {
	// find pageOptions that match fuzzy search string
	var optionMap = make(map[string]*KV)
	var valueArr []string
	for _, kv := range s.Options {
		optionMap[kv.Value] = kv
		valueArr = append(valueArr, kv.Value)
	}
	rankedResults := fuzzy.RankFindFold(s.fuzzySearchString, valueArr)

	s.searchOptions = nil
	for _, result := range rankedResults {
		s.searchOptions = append(s.searchOptions, optionMap[result.Target])
	}
}

func (s *PageKVSelect) loadPageData(page int) (err error) {
	options, err := s.SourceFunc(page, s.Size, s.searchOptions)
	s.index = 0
	if len(options) > 0 {
		s.pageOptions = options
	}
	if len(s.searchOptions) == 0 {
		s.pageOptions = nil
	}
	s.isEmpty = (page+1)*s.Size >= len(s.searchOptions)
	return err
}

func (s *PageKVSelect) Show() (*KV, error) {

	area, err := pterm.DefaultArea.Start()
	if err != nil {
		return nil, fmt.Errorf("could not start area: %w", err)
	}
	defer area.Stop()

	s.search()
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
				// Fuzzy search for pageOptions
				// append to fuzzy search string
				s.fuzzySearchString += key.String()
				s.index = 0
				page = 0
				s.search()
				if err := s.loadPageData(page); err != nil {
					return true, err
				}
				area.Update(s.renderSelect())
			}
		case keys.Backspace:
			// Remove last character from fuzzy search string
			if len(s.fuzzySearchString) > 0 {
				// Handle UTF-8 characters
				s.fuzzySearchString = string([]rune(s.fuzzySearchString)[:len([]rune(s.fuzzySearchString))-1])
			}
			s.index = 0
			page = 0
			s.search()
			if err := s.loadPageData(page); err != nil {
				return true, err
			}
			area.Update(s.renderSelect())
		case keys.CtrlC:
			s.result = nil
			os.Exit(0)
			return true, fmt.Errorf("keyboard interrupt")
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
			if !s.isEmpty {
				page++
				if err := s.loadPageData(page); err != nil {
					return true, err
				}
				area.Update(s.renderSelect())
			}
		case keys.Enter:
			s.result = s.pageOptions[s.index]
			return true, nil
		default:
			return false, nil
		}
		return false, nil // Return false to continue listening
	})
	if err != nil {
		return nil, err
	}

	return s.result, nil
}
