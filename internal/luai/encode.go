/*
 *    Copyright 2025 Han Li and contributors
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
	"errors"
	"reflect"

	lua "github.com/yuin/gopher-lua"
)

func Marshal(state *lua.LState, v any) (lua.LValue, error) {
	reflected := reflect.ValueOf(v)
	if reflected.Kind() == reflect.Ptr {
		reflected = reflected.Elem()
	}

	if !reflected.IsValid() {
		return lua.LNil, nil
	}

	switch reflected.Kind() {
	case reflect.Struct:
		table := state.NewTable()
		for i := 0; i < reflected.NumField(); i++ {
			field := reflected.Field(i)
			if field.Kind() == reflect.Ptr {
				field = field.Elem()
			}

			fieldType := reflected.Type().Field(i)
			tag := fieldType.Tag.Get("json")
			if tag == "" {
				tag = fieldType.Name
			}

			if !field.IsValid() {
				continue
			}

			sub, err := Marshal(state, field.Interface())
			if err != nil {
				return nil, err
			}
			if lf, ok := sub.(*lua.LFunction); ok {
				state.SetField(table, tag, lf)
			} else {
				table.RawSetString(tag, sub)
			}

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
			field := reflected.Index(i)
			if !field.IsValid() {
				continue
			}

			value, err := Marshal(state, field.Interface())
			if err != nil {
				return nil, err
			}
			table.RawSetInt(i+1, value)
		}
		return table, nil
	case reflect.Map:
		table := state.NewTable()
		for _, key := range reflected.MapKeys() {
			field := reflected.MapIndex(key)
			if !field.IsValid() {
				continue
			}

			value, err := Marshal(state, field.Interface())
			if err != nil {
				return nil, err
			}

			switch key.Kind() {
			case reflect.String:
				table.RawSetString(key.String(), value)
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				table.RawSetInt(int(key.Int()), value)
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				table.RawSetInt(int(key.Uint()), value)
			default:
				return nil, errors.New("marshal: unsupported type " + key.Kind().String() + " for key")
			}

		}
		return table, nil
	case reflect.Func:
		if reflected.Type().ConvertibleTo(reflect.TypeOf(lua.LGFunction(nil))) {
			lf := reflected.Convert(reflect.TypeOf(lua.LGFunction(nil))).Interface().(lua.LGFunction)
			return state.NewFunction(lf), nil
		} else {
			return nil, errors.New("marshal: unsupported function type " + reflected.Type().String())
		}
	default:
		return nil, errors.New("marshal: unsupported type " + reflected.Kind().String() + " for reflected ")
	}

}
