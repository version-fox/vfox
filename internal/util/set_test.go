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

package util

import (
	"reflect"
	"testing"
)

func TestMapSet(t *testing.T) {
	s := NewSet[int]()

	s.Add(1)
	if !s.Contains(1) {
		t.Errorf("Expected set to contain 1")
	}

	s.Remove(1)
	if s.Contains(1) {
		t.Errorf("Expected set to not contain 1")
	}

	if s.Len() != 0 {
		t.Errorf("Expected set length to be 0, got %d", s.Len())
	}
}

func TestOrderedSet(t *testing.T) {
	s := NewSortedSet[int]()

	s.Add(1)
	if !s.Contains(1) {
		t.Errorf("Expected set to contain 1")
	}

	s.Remove(1)
	if s.Contains(1) {
		t.Errorf("Expected set to not contain 1")
	}

	if s.Len() != 0 {
		t.Errorf("Expected set length to be 0, got %d", s.Len())
	}

	for v := range s.Slice() {
		t.Errorf("Expected no iteration over set, but got %v", v)
	}
}

func TestSortedSetSort(t *testing.T) {
	s := NewSortedSet[string]()

	elements := []string{"ss23434444444jl2342342424s", "89999809898998bbb", "99234234234234aaa", "1232ssssssssssssssss414141fff"}

	for _, r := range elements {
		s.Add(r)
	}

	if !reflect.DeepEqual(s.Slice(), elements) {
		t.Errorf("Expected set to contain %v, got %v", elements, s.Slice())
	}

	e := elements[0]
	elements = elements[1:]
	s.Remove(e)
	if !reflect.DeepEqual(s.Slice(), elements) {
		t.Errorf("Expected set to contain %v, got %v", elements, s.Slice())
	}

}
