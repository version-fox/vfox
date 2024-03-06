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

		tag := fieldTypeField.Tag.Get("luai")
		if tag == "" {
			tag = fieldTypeField.Name
		}

		switch field.Kind() {
		case reflect.Struct:
			subTable, err := Marshal(state, field.Interface())
			if err != nil {
				return nil, err
			}
			table.RawSetString(tag, subTable)
		case reflect.String:
			table.RawSetString(tag, lua.LString(field.String()))
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			table.RawSetString(tag, lua.LNumber(field.Int()))
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			table.RawSetString(tag, lua.LNumber(field.Uint()))
		case reflect.Float32, reflect.Float64:
			table.RawSetString(tag, lua.LNumber(field.Float()))
		case reflect.Bool:
			table.RawSetString(tag, lua.LBool(field.Bool()))
		default:
			return nil, errors.New("marshal: unsupported type " + field.Kind().String() + " for field " + fieldTypeField.Name + " in " + fieldType.Name() + " struct")
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

			// if field is not found, try to find it by tag
			if !field.IsValid() {
				for i := 0; i < reflected.NumField(); i++ {
					fieldTypeField := reflected.Type().Field(i)
					tag := fieldTypeField.Tag.Get("luai")
					if tag == fieldName {
						field = reflected.Field(i)
						break
					}
				}
			}

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
