// Lua Interface
// Marshal and Unmarshal Lua Table to Go Struct

package luai

import (
	"errors"
	"reflect"

	lua "github.com/yuin/gopher-lua"
)

func Marshal(state *lua.LState, v any) (*lua.LTable, error) {
	table := state.NewTable()

	reflected := reflect.ValueOf(v)
	if reflected.Kind() == reflect.Ptr {
		reflected = reflected.Elem()
	}

	if reflected.Kind() != reflect.Struct {
		return nil, errors.New("marshal: value must be a struct")
	}

	for i := 0; i < reflected.NumField(); i++ {
		field := reflected.Field(i)
		fieldType := reflected.Type()
		fieldTypeField := reflected.Type().Field(i)

		if field.Kind() == reflect.Ptr {
			field = field.Elem()
		}

		switch field.Kind() {
		case reflect.Struct:
			subTable, err := Marshal(state, field.Interface())
			if err != nil {
				return nil, err
			}
			table.RawSetString(fieldTypeField.Name, subTable)
		case reflect.String:
			table.RawSetString(fieldTypeField.Name, lua.LString(field.String()))
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			table.RawSetString(fieldTypeField.Name, lua.LNumber(field.Int()))
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			table.RawSetString(fieldTypeField.Name, lua.LNumber(field.Uint()))
		case reflect.Float32, reflect.Float64:
			table.RawSetString(fieldTypeField.Name, lua.LNumber(field.Float()))
		case reflect.Bool:
			table.RawSetString(fieldTypeField.Name, lua.LBool(field.Bool()))
		case reflect.Map:
			switch fieldType.Key().Kind() {
			case reflect.String,
				reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
				reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
			default:
				return nil, errors.New("marshal: unsupported map key type")
			}
		default:
			return nil, errors.New("marshal: unsupported type")
		}
	}
	return table, nil
}

func Unmarshal(table *lua.LTable, v any) error {
	// if v is array
	reflected := reflect.ValueOf(v)

	if reflected.Kind() == reflect.Ptr {
		reflected = reflected.Elem()
	}

	if reflected.Kind() != reflect.Array && reflected.Kind() != reflect.Struct {
		return errors.New("unmarshal: value must be a array or struct, got " + reflected.Kind().String())
	}

	table.ForEach(func(key, value lua.LValue) {
		switch key.Type() {
		case lua.LTString:
			fieldName := key.String()
			field := reflected.FieldByName(fieldName)
			luaType := value.Type()

			switch luaType {
			case lua.LTString:
				field.SetString(value.String())
			case lua.LTNumber:
				field.SetInt(int64(value.(lua.LNumber)))
			case lua.LTBool:
				field.SetBool(bool(value.(lua.LBool)))
			case lua.LTTable:
				Unmarshal(value.(*lua.LTable), field.Interface())
			default:
				return
			}
		case lua.LTNumber:
			fieldIndex := int(key.(lua.LNumber))
			field := reflected.Index(fieldIndex)
			luaType := value.Type()
			switch luaType {
			case lua.LTString:
				field.SetString(value.String())
			case lua.LTNumber:
				field.SetInt(int64(value.(lua.LNumber)))
			case lua.LTBool:
				field.SetBool(bool(value.(lua.LBool)))
			case lua.LTTable:
				Unmarshal(value.(*lua.LTable), field.Interface())
			default:
				return
			}
			return
		}
	})

	return nil
}
