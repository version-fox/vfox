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

package config

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/version-fox/vfox/internal/shared/cache"
)

var (
	durationType      = reflect.TypeOf(time.Duration(0))
	cacheDurationType = reflect.TypeOf(cache.Duration(0))
)

// Set gives the setting that key names the value typed on the command line, and
// changes nothing else in the file. A key is the names of the sections down to
// the setting, joined with dots: plugin.http.timeout. Nothing is written until
// Save, and a key or a value that does not fit the setting is an error.
func (f *File) Set(key, value string) error {
	path := strings.Split(key, ".")
	t, err := settingType(path)
	if err != nil {
		return err
	}
	switch t.Kind() {
	case reflect.Struct:
		return fmt.Errorf("%s is a section, not a setting", key)
	case reflect.Map, reflect.Slice:
		return fmt.Errorf("%s holds a list or a table, which cannot be set from the command line: edit %s", key, f.path)
	}
	typed, err := parseSetting(t, value)
	if err != nil {
		return fmt.Errorf("%s: %w", key, err)
	}
	return f.put(path, typed)
}

// Unset removes the setting that key names from the file, and the sections that
// this leaves empty, instead of writing its default: what applies then is what
// the shared config and the defaults say. A section can be removed as a whole.
// It reports whether the file had the key; a key that is not set is not an
// error.
func (f *File) Unset(key string) (bool, error) {
	path := strings.Split(key, ".")
	if _, err := settingType(path); err != nil {
		return false, err
	}
	return f.remove(path), nil
}

// settingType follows a key through the yaml names of Config and returns the
// type it ends at: a section, or the type of a setting.
func settingType(key []string) (reflect.Type, error) {
	t := reflect.TypeOf(Config{})
	for _, name := range key {
		field, ok := sectionField(t, name)
		if !ok {
			return nil, fmt.Errorf("unknown config key: %s", strings.Join(key, "."))
		}
		t = field.Type
	}
	return deref(t), nil
}

func sectionField(t reflect.Type, name string) (reflect.StructField, bool) {
	if t = deref(t); t.Kind() != reflect.Struct {
		return reflect.StructField{}, false
	}
	for i := 0; i < t.NumField(); i++ {
		if tag, _, _ := strings.Cut(t.Field(i).Tag.Get("yaml"), ","); tag == name {
			return t.Field(i), true
		}
	}
	return reflect.StructField{}, false
}

func deref(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t
}

// parseSetting turns the text typed for a setting of type t into the value that
// is written to the file.
func parseSetting(t reflect.Type, text string) (any, error) {
	switch t {
	case durationType:
		d, err := time.ParseDuration(text)
		if err != nil {
			return nil, err
		}
		if d < 0 {
			return nil, errors.New("must not be negative")
		}
		return d, nil
	case cacheDurationType:
		// -1 never expires and 0 does not cache; anything else is a duration.
		if text == "-1" {
			return cache.Duration(-1), nil
		}
		d, err := time.ParseDuration(strings.ToLower(text))
		if err != nil {
			return nil, err
		}
		if d < 0 {
			return nil, errors.New("must be -1 (never expires), 0 (does not cache) or a duration")
		}
		if d > 0 && d < time.Second {
			return nil, errors.New("must be at least 1s")
		}
		return cache.Duration(d), nil
	}

	switch t.Kind() {
	case reflect.String:
		return text, nil
	case reflect.Bool:
		b, err := strconv.ParseBool(text)
		if err != nil {
			return nil, fmt.Errorf("%q is neither true nor false", text)
		}
		return b, nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n, err := strconv.ParseInt(text, 10, t.Bits())
		if err != nil {
			return nil, fmt.Errorf("%q is not a whole number", text)
		}
		return n, nil
	}
	return nil, errors.New("unsupported configuration type")
}
