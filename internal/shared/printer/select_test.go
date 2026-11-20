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
	"fmt"
	"os"
	"testing"

	"github.com/version-fox/vfox/internal/shared/util"
)

func TestSelect_Show(t *testing.T) {
	if os.Getenv("CI") != "" || !util.IsTTY() {
		t.Skip("Skipping TestSelect_Show in CI environment because it requires user input")
	}

	source := []*KV{
		{
			Key:   "1",
			Value: "1",
		},
		{
			Key:   "2",
			Value: "2",
		},
		{
			Key:   "3",
			Value: "3",
		},
		{
			Key:   "4",
			Value: "4",
		},
		{
			Key:   "5",
			Value: "5",
		},
	}
	s := &PageKVSelect{
		index: 0,
		SourceFunc: func(page, size int, options []*KV) ([]*KV, error) {
			// 计算开始和结束索引
			start := page * size
			end := start + size

			// 检查索引是否超出范围
			if start > len(source) {
				return nil, fmt.Errorf("page is out of range")
			}
			if end > len(source) {
				end = len(source)
			}

			// 返回分页后的元素
			return source[start:end], nil
		},
		Size: 3,
	}
	show, err := s.Show()
	print(show)
	if err != nil {
		t.Fatal(err)
	}
}

func TestSearchPreservesOrder(t *testing.T) {
	options := []*KV{
		{Key: "24.1.0", Value: "24.1.0 (Current)"},
		{Key: "22.15.0", Value: "22.15.0 (LTS Jod)"},
		{Key: "20.19.0", Value: "20.19.0 (LTS Iron)"},
		{Key: "18.20.8", Value: "18.20.8 (LTS Hydrogen)"},
	}
	for _, tc := range []struct {
		name, query string
		want        []*KV
	}{
		{"empty", "", options},
		{"lts", "lts", options[1:]},
		{"uppercase", "LTS", options[1:]},
		{"mixed case", "LtS", options[1:]},
		{"fuzzy subsequence", "ls", options[1:]},
		{"no match", "not-a-version", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := &PageKVSelect{Options: options, fuzzySearchString: tc.query}
			s.search()
			assertSearchOptions(t, s.searchOptions, tc.want)
			// Clearing a filter must restore the full original sequence.
			s.fuzzySearchString = ""
			s.search()
			assertSearchOptions(t, s.searchOptions, options)
		})
	}
}

func TestSearchPreservesDuplicateLabels(t *testing.T) {
	options := []*KV{
		{Key: "first", Value: "22.15.0 (LTS)"},
		{Key: "second", Value: "22.15.0 (LTS)"},
	}
	s := &PageKVSelect{Options: options, fuzzySearchString: "lts"}
	s.search()
	assertSearchOptions(t, s.searchOptions, options)
}

func assertSearchOptions(t *testing.T, got, want []*KV) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %d options, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("option %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}
