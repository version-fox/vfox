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

import "fmt"

// ErrorStore is a struct that stores errors
// And can throw them all at once
type ErrorStore struct {
	errors map[string]error
}

// NewErrorStore creates a new ErrorStore
func NewErrorStore() *ErrorStore {
	return &ErrorStore{
		errors: make(map[string]error),
	}
}

// Add adds an error to the store
func (e *ErrorStore) Add(note string, err error) {
	e.errors[note] = err
}

// Add and show in the console
func (e *ErrorStore) AddAndShow(note string, err error) {
	e.Add(note, err)
	fmt.Println(err)
}

// get all error notes
func (e *ErrorStore) GetNotes() []string {
	notes := make([]string, 0, len(e.errors))
	for note := range e.errors {
		notes = append(notes, note)
	}
	return notes
}

func (e *ErrorStore) HasError() bool {
	return len(e.errors) > 0
}
