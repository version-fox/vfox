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
