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
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// Every setting of Config is meant to be set with `vfox config`. A setting of a
// type that parseSetting does not know fails here, so that whoever adds one
// also teaches `vfox config` to write it. Lists and tables are the exception:
// they are edited in the file.
func TestEverySettingOfConfigCanBeSet(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	sample := func(typ reflect.Type) (string, bool) {
		switch {
		case typ == durationType, typ == cacheDurationType:
			return "1m", true
		case typ.Kind() == reflect.Bool:
			return "true", true
		case typ.Kind() == reflect.String:
			return "sample", true
		case typ.Kind() >= reflect.Int && typ.Kind() <= reflect.Int64:
			return "1", true
		}
		return "", false
	}

	settings := 0
	var walk func(prefix string, typ reflect.Type)
	walk = func(prefix string, typ reflect.Type) {
		typ = deref(typ)
		for i := 0; i < typ.NumField(); i++ {
			field := typ.Field(i)
			key := prefix + strings.Split(field.Tag.Get("yaml"), ",")[0]
			fieldType := deref(field.Type)
			switch fieldType.Kind() {
			case reflect.Struct:
				walk(key+".", fieldType)
			case reflect.Map, reflect.Slice:
				file, err := LoadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				if err := file.Set(key, "x"); err == nil || !strings.Contains(err.Error(), "holds a list or a table") {
					t.Errorf("Set(%s) = %v, want it refused as a list or a table", key, err)
				}
			default:
				value, ok := sample(fieldType)
				if !ok {
					t.Errorf("%s has the type %s, which `vfox config` cannot set: teach parseSetting about it", key, fieldType)
					continue
				}
				settings++
				setKey(t, path, key, value)
			}
		}
	}
	walk("", reflect.TypeOf(Config{}))

	if settings == 0 {
		t.Fatal("found no setting at all")
	}
	if _, err := NewConfigWithPath(path); err != nil {
		t.Errorf("the file with every setting set does not load: %v:\n%s", err, readConfigFile(t, path))
	}
}

func TestParseSettingOfWholeNumbers(t *testing.T) {
	for _, c := range []struct {
		typ   reflect.Type
		text  string
		want  int64
		fails bool
	}{
		{reflect.TypeOf(int(0)), "42", 42, false},
		{reflect.TypeOf(int(0)), "-7", -7, false},
		{reflect.TypeOf(int8(0)), "127", 127, false},
		{reflect.TypeOf(int8(0)), "128", 0, true}, // does not fit
		{reflect.TypeOf(int64(0)), "9223372036854775807", 9223372036854775807, false},
		{reflect.TypeOf(int(0)), "1.5", 0, true},
		{reflect.TypeOf(int(0)), "many", 0, true},
		{reflect.TypeOf(int(0)), "", 0, true},
	} {
		got, err := parseSetting(c.typ, c.text)
		if c.fails {
			if err == nil {
				t.Errorf("parseSetting(%s, %q) = %v, want an error", c.typ, c.text, got)
			}
			continue
		}
		if err != nil || got != c.want {
			t.Errorf("parseSetting(%s, %q) = %v, %v, want %d", c.typ, c.text, got, err, c.want)
		}
	}
}

func TestParseSettingRefusesTypesItDoesNotKnow(t *testing.T) {
	for _, typ := range []reflect.Type{reflect.TypeOf(float64(0)), reflect.TypeOf(uint(0)), reflect.TypeOf(struct{}{})} {
		if got, err := parseSetting(typ, "1"); err == nil {
			t.Errorf("parseSetting(%s) = %v, want an error", typ, got)
		}
	}
}
