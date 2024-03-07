// Lua Interface
// Marshal and Unmarshal Lua Table to Go Struct

package luai

import (
	"errors"
	"reflect"

	"github.com/version-fox/vfox/internal/logger"
	lua "github.com/yuin/gopher-lua"
)

func Marshal(state *lua.LState, v any) (lua.LValue, error) {
	reflected := reflect.ValueOf(v)
	if reflected.Kind() == reflect.Ptr {
		reflected = reflected.Elem()
	}

	switch reflected.Kind() {
	case reflect.Struct:
		table := state.NewTable()
		for i := 0; i < reflected.NumField(); i++ {
			field := reflected.Field(i)
			fieldType := reflected.Type().Field(i)
			if field.Kind() == reflect.Ptr {
				field = field.Elem()
			}

			tag := fieldType.Tag.Get("luai")
			if tag == "" {
				tag = fieldType.Name
			}

			sub, err := Marshal(state, field.Interface())
			if err != nil {
				return nil, err
			}
			logger.Debugf("field: %v, tag: %v, sub: %v, kind: %s\n", field, tag, sub, field.Kind())
			table.RawSetString(tag, sub)
		}
		return table, nil
	case reflect.String:
		return lua.LString(reflected.String()), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return lua.LNumber(reflected.Int()), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return lua.LNumber(reflected.Uint()), nil
	case reflect.Float32, reflect.Float64:
		return lua.LNumber(reflected.Float()), nil
	case reflect.Bool:
		return lua.LBool(reflected.Bool()), nil
	case reflect.Array, reflect.Slice:
		table := state.NewTable()
		for i := 0; i < reflected.Len(); i++ {
			value, err := Marshal(state, reflected.Index(i).Interface())
			if err != nil {
				return nil, err
			}
			table.RawSetInt(i+1, value)
		}
		return table, nil
	case reflect.Map:
		table := state.NewTable()
		for _, key := range reflected.MapKeys() {
			value, err := Marshal(state, reflected.MapIndex(key).Interface())
			if err != nil {
				return nil, err
			}

			table.RawSetString(key.String(), value)
		}
		return table, nil
	default:
		return nil, errors.New("marshal: unsupported type " + reflected.Kind().String() + " for reflected ")
	}

}
