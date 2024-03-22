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

	t.Run("TableWithEmptyFieldAndIncompitibleType", func(t *testing.T) {
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

	m := map[string]interface{}{
		"key1": "value1",
		"key2": 2,
		"key3": true,
	}
	mFloat64 := map[string]interface{}{
		"key1": "value1",
		"key2": float64(2),
		"key3": true,
	}

	s := []any{"value1", 2, true}
	sFloat64 := []any{"value1", float64(2), true}

	normalStruct := testStruct{
		Field1: "test",
		Field2: 1,
		Field3: true,
	}
	normalStructWithTag := testStructTag{
		Field1: "test",
		Field2: 1,
		Field3: true,
	}

	var unmarshalTests = []struct {
		CaseName            string
		in                  any
		ptr                 any
		out                 any
		luaValidationScript string
		err                 error
	}{
		{
			CaseName: "Struct",
			in:       normalStruct,
			ptr:      new(testStruct),
			out:      &normalStruct,
			luaValidationScript: `
				assert(table.Field1 == "test")
				assert(table.Field2 == 1)
				assert(table.Field3 == true)
				print("lua Struct done")
			`,
		},
		{
			CaseName: "Struct with Tag",
			in:       normalStructWithTag,
			ptr:      &testStructTag{},
			out:      &normalStructWithTag,
			luaValidationScript: `
				assert(table.field1 == "test")
				assert(table.field2 == 1)
				assert(table.field3 == true)
				print("lua Struct with Tag done")
			`,
		},
		{
			CaseName: "Map",
			in:       m,
			ptr:      &map[string]any{},
			out:      &mFloat64,
		},
		{
			CaseName: "Slice",
			in:       s,
			ptr:      &[]any{},
			out:      &sFloat64,
		},
		{
			CaseName: "Any",
			in:       m,
			ptr:      new(any),
			out:      &mFloat64,
		},
		{
			CaseName: "Map[Int]",
			in: map[int]int{
				1: 1,
				2: 2,
			},
			ptr: &map[int]int{},
			out: &map[int]int{
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
				Map:   m,
				Slice: s,
			},
			ptr: &complexStruct{},
			out: &complexStruct{
				Field1: "value1",
				Field2: 123,
				Field3: true,
				Struct: testStructTag{
					Field1: "value1",
					Field2: 2,
					Field3: true,
				},
				Map:   mFloat64,
				Slice: sFloat64,
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
		if tt.CaseName != "Map[Int]" {
			continue
		}
		t.Run(tt.CaseName, func(t *testing.T) {
			L := lua.NewState()
			defer L.Close()

			table, err := Marshal(L, tt.in)
			if err != nil {
				t.Fatalf("marshal map failed: %v", err)
			}

			L.SetGlobal("m", table)

			if tt.luaValidationScript != "" {
				if err := L.DoString(tt.luaValidationScript); err != nil {
					t.Errorf("validate %s error: %v", tt.CaseName, err)
				}
			}

			err = Unmarshal(table, tt.ptr)
			if err != tt.err {
				t.Errorf("expected %+v, got %+v", tt.err, err)
			}
			if !reflect.DeepEqual(tt.out, tt.ptr) {
				t.Errorf("expected %+v, got %+v", tt.out, tt.ptr)
			}
		})
	}
}
