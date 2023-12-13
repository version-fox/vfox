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
	"fmt"
	"testing"
)

func TestSelect_Show(t *testing.T) {
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
		SourceFunc: func(page, size int) ([]*KV, error) {
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
