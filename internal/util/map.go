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

//type Map[K comparable, V any] interface {
//	Get(k K) (V, bool)
//	Set(k K, v V) bool
//	Remove(k K) V
//	Contains(k K) bool
//	Len() int
//	ForEach(f func(k K, v V) error) error
//}

type SortedMap[K comparable, V any] struct {
	keys []K
	vals map[K]V
}

func (s *SortedMap[K, V]) Get(k K) (V, bool) {
	v, ok := s.vals[k]
	return v, ok
}

func (s *SortedMap[K, V]) Set(k K, v V) bool {
	_, exists := s.vals[k]
	if !exists {
		s.keys = append(s.keys, k)
	}
	s.vals[k] = v
	return !exists
}

func (s *SortedMap[K, V]) Remove(k K) V {
	v := s.vals[k]
	delete(s.vals, k)
	for i, key := range s.keys {
		if key == k {
			s.keys = append(s.keys[:i], s.keys[i+1:]...)
			break
		}
	}
	return v
}

func (s *SortedMap[K, V]) Contains(k K) bool {
	_, exists := s.vals[k]
	return exists
}

func (s *SortedMap[K, V]) Len() int {
	return len(s.keys)
}

func (s *SortedMap[K, V]) Keys() []K {
	return s.keys
}

func (s *SortedMap[K, V]) Merge(sortedMap *SortedMap[K, V]) {
	for _, k := range sortedMap.keys {
		s.Set(k, sortedMap.vals[k])
	}
}

func (s *SortedMap[K, V]) ForEach(f func(k K, v V) error) error {
	for _, k := range s.keys {
		if err := f(k, s.vals[k]); err != nil {
			return err
		}
	}
	return nil
}

func NewSortedMap[K comparable, V any]() *SortedMap[K, V] {
	return &SortedMap[K, V]{
		keys: []K{},
		vals: make(map[K]V),
	}
}
