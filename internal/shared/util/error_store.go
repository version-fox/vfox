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

import "fmt"

type errorItem struct {
	Note string
	Err  error
}

// ErrorStore is a struct that stores errors
type ErrorStore struct {
	errors []errorItem
}

// NewErrorStore creates a new ErrorStore
func NewErrorStore() *ErrorStore {
	return &ErrorStore{
		errors: make([]errorItem, 0),
	}
}

// Add adds an error to the store
func (e *ErrorStore) Add(note string, err error) {
	e.errors = append(e.errors, errorItem{Note: note, Err: err})
}

// Add and show in the console
func (e *ErrorStore) AddAndShow(note string, err error) {
	e.Add(note, err)
	fmt.Println(err)
}

// get all error notes
func (e *ErrorStore) GetNotes() []string {
	notes := make([]string, 0, len(e.errors))

	for _, item := range e.errors {
		notes = append(notes, item.Note)
	}

	return notes
}

// get notes set
func (e *ErrorStore) GetNotesSet() Set[string] {
	set := NewSet[string]()
	for _, item := range e.errors {
		set.Add(item.Note)
	}
	return set
}

func (e *ErrorStore) HasError() bool {
	return len(e.errors) > 0
}
