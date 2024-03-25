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

package luai

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/version-fox/vfox/internal/logger"
	lua "github.com/yuin/gopher-lua"
)

func setupSuite(tb testing.TB) func(tb testing.TB) {
	logger.SetLevel(logger.DebugLevel)

	return func(tb testing.TB) {
		logger.SetLevel(logger.InfoLevel)
	}
}

type testStruct struct {
	Field1 string
	Field2 int
	Field3 bool
}

type testStructTag struct {
	Field1 string `luai:"field1"`
	Field2 int    `luai:"field2"`
	Field3 bool   `luai:"field3"`
}

type complexStruct struct {
	Field1       string
	Field2       int
	Field3       bool
	SimpleStruct *testStruct
	Struct       testStructTag
	Map          map[string]interface{}
	Slice        []any
}

func TestExample(t *testing.T) {
	teardownSuite := setupSuite(t)
	defer teardownSuite(t)

	t.Run("TableWithEmptyFieldAndIncompatibleType", func(t *testing.T) {
		L := NewLuaVM()
		defer L.Close()

		output := struct {
			Field1  string  `luai:"field1"`
			Field2  *string `luai:"field2"`
			AString string  `luai:"a_string"`
		}{}

		if err := L.Instance.DoString(`
			return {
				field1 = "value1",	
				--- notice: here we return a number
				a_string = 8,
            }
		`); err != nil {
			t.Errorf("map test failed: %v", err)
		}

		table := L.ReturnedValue()
		err := Unmarshal(table, &output)
		if err != nil {
			t.Fatalf("unmarshal map failed: %v", err)
		}
		fmt.Printf("output: %+v\n", output)
		if output.Field1 != "value1" {
			t.Errorf("expected %+v, got %+v", "value1", output.Field1)
		}
		if output.Field2 != nil {
			t.Errorf("expected %+v, got %+v", nil, output.Field2)
		}
		if output.AString != "8" {
			t.Errorf("expected %+v, got %+v", "", output.AString)
		}
	})
}

func TestCases(t *testing.T) {
	teardownSuite := setupSuite(t)
	defer teardownSuite(t)

	var unmarshalTests = []struct {
		CaseName            string
		in                  any
		ptr                 any // new(type)
		out                 any
		luaValidationScript string
		err                 error
	}{
		{
			CaseName: "Struct",
			in: testStruct{
				Field1: "test",
				Field2: 1,
				Field3: true,
			},
			ptr: new(testStruct),
			out: testStruct{
				Field1: "test",
				Field2: 1,
				Field3: true,
			},
			luaValidationScript: `
				assert(m.Field1 == "test")
				assert(m.Field2 == 1)
				assert(m.Field3 == true)
				print("lua Struct done")
			`,
		},
		{
			CaseName: "Struct with Tag",
			in: testStructTag{
				Field1: "test",
				Field2: 1,
				Field3: true,
			},
			ptr: new(testStructTag),
			out: testStructTag{
				Field1: "test",
				Field2: 1,
				Field3: true,
			},
			luaValidationScript: `
				assert(m.field1 == "test")
				assert(m.field2 == 1)
				assert(m.field3 == true)
				print("lua Struct with Tag done")
			`,
		},
		{
			CaseName: "Map",
			in: map[string]interface{}{
				"key1": "value1",
				"key2": 2,
				"key3": true,
			},
			ptr: new(map[string]any),
			out: map[string]interface{}{
				"key1": "value1",
				"key2": float64(2),
				"key3": true,
			},
		},
		{
			CaseName: "Slice",
			in:       []any{"value1", 2, true},
			ptr:      new([]any),
			out:      []any{"value1", float64(2), true},
		},
		{
			CaseName: "Any",
			in: map[string]interface{}{
				"key1": "value1",
				"key2": 2,
				"key3": true,
			},
			ptr: new(any),
			out: map[string]interface{}{
				"key1": "value1",
				"key2": float64(2),
				"key3": true,
			},
			luaValidationScript: `
			assert(m.key1 == "value1")
			assert(m.key2 == 2)
			assert(m.key3 == true)
			print("Any Done")
		`,
		},
		{
			CaseName: "Map[Int]",
			in: map[int]int{
				1: 1,
				2: 2,
			},
			ptr: new(map[int]int),
			out: map[int]int{
				1: 1,
				2: 2,
			},
			luaValidationScript: `
				assert(m[1] == 1)
				assert(m[2] == 2)
				print("lua Map[Int] done")
			`,
		},
		{
			CaseName: "MapSliceStructUnified",
			in: complexStruct{
				Field1: "value1",
				Field2: 123,
				Field3: true,
				Struct: testStructTag{
					Field1: "value1",
					Field2: 2,
					Field3: true,
				},
				Map: map[string]interface{}{
					"key1": "value1",
					"key2": float64(2),
					"key3": true,
				},
				Slice: []any{"value1", 2, true},
			},
			ptr: new(complexStruct),
			out: complexStruct{
				Field1: "value1",
				Field2: 123,
				Field3: true,
				Struct: testStructTag{
					Field1: "value1",
					Field2: 2,
					Field3: true,
				},
				Map: map[string]interface{}{
					"key1": "value1",
					"key2": float64(2),
					"key3": true,
				},
				Slice: []any{"value1", float64(2), true},
			},
			luaValidationScript: `
				assert(m.Field1 == "value1")
				assert(m.Field2 == 123)
				assert(m.Field3 == true)
				assert(m.Struct.field1 == "value1")
				assert(m.Struct.field2 == 2)
				assert(m.Struct.field3 == true)
				assert(m.Map.key1 == "value1")
				assert(m.Map.key2 == 2)
				assert(m.Map.key3 == true)
				assert(m.Slice[1] == "value1")
				assert(m.Slice[2] == 2)
				assert(m.Slice[3] == true)
				print("lua MapSliceStructUnified done")
			`,
		},
	}

	for _, tt := range unmarshalTests {
		t.Run(tt.CaseName, func(t *testing.T) {
			L := lua.NewState()
			defer L.Close()

			table, err := Marshal(L, tt.in)
			if err != nil {
				t.Fatalf("marshal map failed: %v", err)
			}

			if tt.luaValidationScript != "" {
				L.SetGlobal("m", table)

				if err := L.DoString(tt.luaValidationScript); err != nil {
					t.Errorf("validate %s error: %v", tt.CaseName, err)
				}
			}

			if tt.ptr == nil {
				return
			}

			typ := reflect.TypeOf(tt.ptr)
			if typ.Kind() != reflect.Pointer {
				t.Fatalf("%s: unmarshalTest.ptr %T is not a pointer type", tt.CaseName, tt.ptr)
			}

			typ = typ.Elem()

			// equals to: v = new(right-type)
			v := reflect.New(typ)

			if !reflect.DeepEqual(tt.ptr, v.Interface()) {
				// There's no reason for ptr to point to non-zero data,
				// as we decode into new(right-type), so the data is
				// discarded.
				// This can easily mean tests that silently don't test
				// what they should. To test decoding into existing
				// data, see TestPrefilled.
				t.Fatalf("%s: unmarshalTest.ptr %#v is not a pointer to a zero value", tt.CaseName, tt.ptr)
			}

			err = Unmarshal(table, v.Interface())

			if err != tt.err {
				t.Errorf("expected %+v, got %+v", tt.err, err)
			}

			// get the value out of the pointer, equals to: v = *v
			got := v.Elem().Interface()

			if !reflect.DeepEqual(tt.out, got) {
				t.Errorf("expected %+v, got %+v", tt.out, got)
			}
		})
	}
}
