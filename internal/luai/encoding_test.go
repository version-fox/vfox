package luai

import (
	"testing"

	lua "github.com/yuin/gopher-lua"
)

type testStruct struct {
	Field1 string
	Field2 int
	Field3 bool
}

func TestMarshal(t *testing.T) {
	luaVm := lua.NewState()

	test := testStruct{
		Field1: "test",
		Field2: 1,
		Field3: true,
	}

	table, err := Marshal(luaVm, &test)
	if err != nil {
		t.Fatal(err)
	}

	field1 := table.RawGetString("Field1")
	if field1.Type() != lua.LTString {
		t.Errorf("expected string, got %s", field1.Type())
	}

	if field1.String() != "test" {
		t.Errorf("expected 'test', got '%s'", field1.String())
	}

	field2 := table.RawGetString("Field2")
	if field2.Type() != lua.LTNumber {
		t.Errorf("expected number, got %s", field2.Type())
	}

	if field2.String() != "1" {
		t.Errorf("expected '1', got '%s'", field2.String())
	}

	field3 := table.RawGetString("Field3")
	if field3.Type() != lua.LTBool {
		t.Errorf("expected bool, got %s", field3.Type())
	}

	if field3.String() != "true" {
		t.Errorf("expected 'true', got '%s'", field3.String())
	}

	struct2 := testStruct{}
	err = Unmarshal(table, &struct2)
	if err != nil {
		t.Fatal(err)
	}

	if struct2.Field1 != "test" {
		t.Errorf("expected 'test', got '%s'", struct2.Field1)
	}

	if struct2.Field2 != 1 {
		t.Errorf("expected 1, got %d", struct2.Field2)
	}

	if struct2.Field3 != true {
		t.Errorf("expected true, got %t", struct2.Field3)
	}
}
