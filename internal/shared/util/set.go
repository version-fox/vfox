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

package util

type Set[T comparable] interface {
	Add(v T) bool
	Remove(v T)
	Contains(v T) bool
	Len() int
	Slice() []T
}

type MapSet[T comparable] struct {
	values map[T]struct{}
}

func (s *MapSet[T]) Add(v T) bool {
	_, exists := s.values[v]
	if !exists {
		s.values[v] = struct{}{}
	}
	return !exists
}

func (s *MapSet[T]) Remove(v T) {
	delete(s.values, v)
}

func (s *MapSet[T]) Contains(v T) bool {
	_, exists := s.values[v]
	return exists
}

func (s *MapSet[T]) Len() int {
	return len(s.values)
}

func (s *MapSet[T]) Slice() []T {
	slice := make([]T, 0, len(s.values))
	for v := range s.values {
		slice = append(slice, v)
	}
	return slice
}

func NewSet[T comparable]() Set[T] {
	return &MapSet[T]{
		values: make(map[T]struct{}),
	}
}

func NewSetWithSlice[T comparable](slice []T) Set[T] {
	s := NewSet[T]()
	for _, v := range slice {
		s.Add(v)
	}
	return s
}

type SortedSet[T comparable] struct {
	elements []T
	set      MapSet[T]
}

func (s *SortedSet[T]) Add(v T) bool {
	exist := s.set.Contains(v)
	if !exist {
		s.elements = append(s.elements, v)
		s.set.Add(v)
	}
	return !exist
}

func (s *SortedSet[T]) Remove(v T) {
	exists := s.set.Contains(v)
	if exists {
		delete(s.set.values, v)
		for i, e := range s.elements {
			if e == v {
				s.elements = append(s.elements[:i], s.elements[i+1:]...)
				break
			}
		}
	}
}

func (s *SortedSet[T]) Contains(v T) bool {
	return s.set.Contains(v)
}

func (s *SortedSet[T]) Len() int {
	return len(s.elements)
}

func (s *SortedSet[T]) Slice() []T {
	return append([]T{}, s.elements...)
}

func (s *SortedSet[T]) AddWithIndex(index int, v T) bool {
	if index < 0 || index > len(s.elements) {
		return false
	}
	if s.set.Contains(v) {
		return false
	}
	s.elements = append(s.elements, v)
	copy(s.elements[index+1:], s.elements[index:])
	s.elements[index] = v
	return s.set.Add(v)
}

func NewSortedSet[T comparable]() *SortedSet[T] {
	return &SortedSet[T]{
		set:      MapSet[T]{values: make(map[T]struct{})},
		elements: make([]T, 0),
	}
}

func NewSortedSetWithSlice[T comparable](slice []T) Set[T] {
	s := NewSortedSet[T]()
	for _, v := range slice {
		s.Add(v)
	}
	return s
}
