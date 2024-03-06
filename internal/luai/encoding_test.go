package luai

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/version-fox/vfox/internal/logger"
	lua "github.com/yuin/gopher-lua"
)

const preloadScript = `
function printTable(t, indent)
    indent = indent or 0
    local strIndent = string.rep("  ", indent)
    for key, value in pairs(t) do
        local keyStr = tostring(key)
        local valueStr = tostring(value)
        if type(value) == "table" then
            print(strIndent .. "[" .. keyStr .. "] =>")
            printTable(value, indent + 1)
        else
            print(strIndent .. "[" .. keyStr .. "] => " .. valueStr)
        end
    end
end
`

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

func TestRegular(t *testing.T) {
	luaVm := lua.NewState()
	defer luaVm.Close()

	test := testStruct{
		Field1: "test",
		Field2: 1,
		Field3: true,
	}

	_table, err := Marshal(luaVm, &test)
	if err != nil {
		t.Fatal(err)
	}

	table := _table.(*lua.LTable)

	luaVm.SetGlobal("table", table)

	if err := luaVm.DoString(`
		assert(table.Field1 == "test")
		assert(table.Field2 == 1)
		assert(table.Field3 == true)
	`); err != nil {
		t.Fatal(err)
	}

	struct2 := testStruct{}
	err = Unmarshal(table, &struct2)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(test, struct2) {
		t.Errorf("expected %+v, got %+v", test, struct2)
	}
}

type testStructTag struct {
	Field1 string `luai:"field1"`
	Field2 int    `luai:"field2"`
	Field3 bool   `luai:"field3"`
}

func TestTag(t *testing.T) {
	luaVm := lua.NewState()
	defer luaVm.Close()

	test := testStructTag{
		Field1: "test",
		Field2: 1,
		Field3: true,
	}

	_table, err := Marshal(luaVm, &test)
	if err != nil {
		t.Fatal(err)
	}

	table := _table.(*lua.LTable)

	luaVm.SetGlobal("table", table)
	if err := luaVm.DoString(`
		assert(table.field1 == "test")
		assert(table.field2 == 1)
		assert(table.field3 == true)
	`); err != nil {
		t.Fatal(err)
	}

	struct2 := testStructTag{}
	err = Unmarshal(table, &struct2)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(test, struct2) {
		t.Errorf("expected %+v, got %+v", test, struct2)
	}
}

func TestMapAndSlice(t *testing.T) {
	teardownSuite := setupSuite(t)
	defer teardownSuite(t)

	// Test case for map
	m := map[string]interface{}{
		"key1": "value1",
		"key2": 2,
		"key3": true,
	}
	L := lua.NewState()
	defer L.Close()
	table, err := Marshal(L, m)
	if err != nil {
		t.Fatalf("marshal map failed: %v", err)
	}
	L.SetGlobal("m", table)
	if err := L.DoString(`
		assert(m.key1 == "value1")
		assert(m.key2 == 2)
		assert(m.key3 == true)
	`); err != nil {
		t.Errorf("map test failed: %v", err)
	}

	// Test case for slice
	s := []any{"value1", 2, true}

	slice, err := Marshal(L, s)
	if err != nil {
		t.Fatalf("marshal slice failed: %v", err)
	}

	L.SetGlobal("s", slice)
	if err := L.DoString(`
		assert(s[1] == "value1")
		assert(s[2] == 2)
		assert(s[3] == true)
	`); err != nil {
		t.Errorf("slice test failed: %v", err)
	}

	// Unmarshal

	// Test case for map
	m2 := map[string]any{}

	fmt.Println("==== start unmarshal ====")

	err = Unmarshal(table, &m2)
	if err != nil {
		t.Fatalf("unmarshal map failed: %v", err)
	}

	fmt.Printf("m2: %+v\n", m2)

	if !reflect.DeepEqual(m, m2) {
		t.Errorf("expected %+v, got %+v", m, m2)
	}

	// Test case for slice
	s2 := []any{}

	err = Unmarshal(slice, &s2)
	if err != nil {
		t.Fatalf("unmarshal slice failed: %v", err)
	}

	fmt.Printf("s2: %+v\n", s2)

	if !reflect.DeepEqual(s, s2) {
		t.Errorf("expected %+v, got %+v", s, s2)
	}

	var s3 any
	err = Unmarshal(slice, &s3)
	if err != nil {
		t.Fatalf("unmarshal slice failed: %v", err)
	}

	if !reflect.DeepEqual(s, s3) {
		t.Errorf("expected %+v, got %+v", s, s3)
	}

}

type complexStruct struct {
	Field1 string
	Field2 int
	Field3 bool
	Map    map[string]interface{}
	Slice  []any
}

func TestMapSliceStructUnified(t *testing.T) {
	teardownSuite := setupSuite(t)
	defer teardownSuite(t)

	m := map[string]interface{}{
		"key1": "value1",
		"key2": 2,
		"key3": true,
	}

	s := []any{"value1", 2, true}

	t.Run("TestCase1", func(t *testing.T) {
		L := lua.NewState()
		defer L.Close()

		input := complexStruct{
			Field1: "value1",
			Field2: 123,
			Field3: true,
			Map:    m,
			Slice:  s,
		}

		table, err := Marshal(L, input)
		if err != nil {
			t.Fatalf("marshal map failed: %v", err)
		}

		L.SetGlobal("m", table)

		L.DoString(preloadScript)

		if err := L.DoString(`
			assert(m.Field1 == "value1")
			assert(m.Field2 == 123)
			assert(m.Field3 == true)
			assert(m.Map.key1 == "value1")
			assert(m.Map.key2 == 2)
			assert(m.Map.key3 == true)
			assert(m.Slice[1] == "value1")
			assert(m.Slice[2] == 2)
			assert(m.Slice[3] == true)
			printTable(m)
			`); err != nil {
			t.Errorf("map test failed: %v", err)
		}

		// Unmarshal
		output := complexStruct{}
		err = Unmarshal(table, &output)
		if err != nil {
			t.Fatalf("unmarshal map failed: %v", err)
		}

		isEqual := reflect.DeepEqual(input, output)
		if !isEqual {
			t.Fatalf("expected %+v, got %+v", input, output)
		}

		fmt.Printf("output: %+v\n", output)
		if output.Field1 != "value1" {
			t.Errorf("expected value1, got %v", output.Field1)
		}

		if output.Field2 != 123 {
			t.Errorf("expected 123, got %v", output.Field2)
		}

		if output.Field3 != true {
			t.Errorf("expected true, got %v", output.Field3)
		}

		if output.Map["key1"] != "value1" {
			t.Errorf("expected value1, got %v", output.Map["key1"])
		}

		if output.Map["key2"] != 2 {
			t.Errorf("expected 2, got %v", output.Map["key2"])
		}

		if output.Map["key3"] != true {
			t.Errorf("expected true, got %v", output.Map["key3"])
		}

		if output.Slice[0] != "value1" {
			t.Errorf("expected value1, got %v", output.Slice[0])
		}

		if output.Slice[1] != 2 {
			t.Errorf("expected 2, got %v", output.Slice[1])
		}
	})
}
