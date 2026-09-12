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

package codec

import (
	"strings"
	"testing"

	lua "github.com/yuin/gopher-lua"
)

func TestUnmarshalRejectsNestedTypeErrors(t *testing.T) {
	for _, source := range []string{
		`return {items = {{enabled = "yes"}}}`,
		`return {items = {{enabled = {}}}}`,
		`return {items = "wrong"}`,
		`return {items = {"wrong"}}`,
	} {
		t.Run(source, func(t *testing.T) {
			L := lua.NewState()
			defer L.Close()
			if err := L.DoString(source); err != nil {
				t.Fatal(err)
			}
			var result struct {
				Items []struct {
					Enabled bool `json:"enabled"`
				} `json:"items"`
			}
			err := Unmarshal(L.Get(-1), &result)
			if err == nil || !strings.Contains(err.Error(), "items") {
				t.Fatalf("error = %v, want field context", err)
			}
		})
	}
}

func TestMarshalFlattensEmbeddedFields(t *testing.T) {
	L := lua.NewState()
	defer L.Close()
	value, err := Marshal(L, OuterPtr{EmbeddedPtr: &EmbeddedPtr{D: "value", E: 42}, F: "outer"})
	if err != nil {
		t.Fatal(err)
	}
	table := value.(*lua.LTable)
	if table.RawGetString("D") != lua.LString("value") || table.RawGetString("E") != lua.LNumber(42) {
		t.Fatal("embedded fields must retain the flat hook protocol")
	}
}

type Embedded struct {
	A string
	B int
}

type Outer struct {
	Embedded
	C string
}

func TestUnmarshalEmbeddedStruct(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	// Create a Lua table with embedded fields
	table := L.NewTable()
	table.RawSetString("A", lua.LString("testA"))
	table.RawSetString("B", lua.LNumber(42))
	table.RawSetString("C", lua.LString("testC"))

	var result Outer
	err := Unmarshal(table, &result)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if result.A != "testA" {
		t.Errorf("expected A='testA', got %s", result.A)
	}
	if result.B != 42 {
		t.Errorf("expected B=42, got %d", result.B)
	}
	if result.C != "testC" {
		t.Errorf("expected C='testC', got %s", result.C)
	}
}

type EmbeddedPtr struct {
	D string
	E int
}

type OuterPtr struct {
	*EmbeddedPtr
	F string
}

func TestUnmarshalEmbeddedPtrStruct(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	// Create a Lua table with embedded pointer fields
	table := L.NewTable()
	table.RawSetString("D", lua.LString("testD"))
	table.RawSetString("E", lua.LNumber(24))
	table.RawSetString("F", lua.LString("testF"))

	var result OuterPtr
	err := Unmarshal(table, &result)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if result.EmbeddedPtr == nil {
		t.Fatal("EmbeddedPtr should not be nil")
	}
	if result.D != "testD" {
		t.Errorf("expected D='testD', got %s", result.D)
	}
	if result.E != 24 {
		t.Errorf("expected E=24, got %d", result.E)
	}
	if result.F != "testF" {
		t.Errorf("expected F='testF', got %s", result.F)
	}
}
