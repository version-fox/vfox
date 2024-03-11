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
	"errors"
	"reflect"
	"strconv"

	lua "github.com/yuin/gopher-lua"
)

// modified from https://cs.opensource.google/go/go/+/master:src/encoding/json/decode.go
// indirect walks down v allocating pointers as needed,
// until it gets to a non-pointer.
func indirect(v reflect.Value) reflect.Value {
	// Issue https://github.com/golang/go/issues/24153 indicates that it is generally not a guaranteed property
	// that you may round-trip a reflect.Value by calling Value.Addr().Elem()
	// and expect the value to still be settable for values derived from
	// unexported embedded struct fields.
	//
	// The logic below effectively does this when it first addresses the value
	// (to satisfy possible pointer methods) and continues to dereference
	// subsequent pointers as necessary.
	//
	// After the first round-trip, we set v back to the original value to
	// preserve the original RW flags contained in reflect.Value.
	v0 := v
	haveAddr := false

	// If v is a named type and is addressable,
	// start with its address, so that if the type has pointer methods,
	// we find them.
	if v.Kind() != reflect.Pointer && v.Type().Name() != "" && v.CanAddr() {
		haveAddr = true
		v = v.Addr()
	}
	for {
		// Load value from interface, but only if the result will be
		// usefully addressable.
		if v.Kind() == reflect.Interface && !v.IsNil() {
			e := v.Elem()
			if e.Kind() == reflect.Pointer && !e.IsNil() && (e.Elem().Kind() == reflect.Pointer) {
				haveAddr = false
				v = e
				continue
			}
		}

		if v.Kind() != reflect.Pointer {
			break
		}

		// Prevent infinite loop if v is an interface pointing to its own address:
		//     var v interface{}
		//     v = &v
		if v.Elem().Kind() == reflect.Interface && v.Elem().Elem() == v {
			v = v.Elem()
			break
		}
		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}

		if haveAddr {
			v = v0 // restore original value after round-trip Value.Addr().Elem()
			haveAddr = false
		} else {
			v = v.Elem()
		}
	}
	return v
}

func storeLiteral(value reflect.Value, lvalue lua.LValue) {
	value = indirect(value)
	switch lvalue.Type() {
	case lua.LTString:
		value.SetString(lvalue.String())
	case lua.LTNumber:
		value.SetInt(int64(lvalue.(lua.LNumber)))
	case lua.LTBool:
		value.SetBool(bool(lvalue.(lua.LBool)))
	}
}

func objectInterface(lvalue *lua.LTable) any {
	var v = make(map[string]any)
	lvalue.ForEach(func(key, value lua.LValue) {
		v[key.String()] = valueInterface(value)
	})
	return v
}

func valueInterface(lvalue lua.LValue) any {
	switch lvalue.Type() {
	case lua.LTTable:
		isArray := lvalue.(*lua.LTable).RawGetInt(1) != lua.LNil
		if isArray {
			return arrayInterface(lvalue.(*lua.LTable))
		}
		return objectInterface(lvalue.(*lua.LTable))
	case lua.LTString:
		return lvalue.String()
	case lua.LTNumber:
		return int(lvalue.(lua.LNumber))
	case lua.LTBool:
		return bool(lvalue.(lua.LBool))
	}
	return nil
}

func arrayInterface(lvalue *lua.LTable) any {
	var v = make([]any, 0)
	lvalue.ForEach(func(key, value lua.LValue) {
		v = append(v, valueInterface(value))
	})

	return v
}

func unmarshalWorker(value lua.LValue, reflected reflect.Value) error {

	switch value.Type() {
	case lua.LTTable:
		reflected = indirect(reflected)
		tagMap := make(map[string]int)

		switch reflected.Kind() {
		case reflect.Interface:
			// Decoding into nil interface? Switch to non-reflect code.
			if reflected.NumMethod() == 0 {
				result := valueInterface(value)
				reflected.Set(reflect.ValueOf(result))
			}
		// map[T1]T2 where T1 is string, an integer type
		case reflect.Map:
			t := reflected.Type()
			keyType := t.Key()
			// Map key must either have string kind, have an integer kind
			switch keyType.Kind() {
			case reflect.String,
				reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
				reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
			default:
				return errors.New("unmarshal: unsupported map key type " + keyType.String())
			}

			if reflected.IsNil() {
				reflected.Set(reflect.MakeMap(t))
			}

			var mapElem reflect.Value

			value.(*lua.LTable).ForEach(func(key, value lua.LValue) {
				// Figure out field corresponding to key.
				var subv reflect.Value

				elemType := t.Elem()
				if !mapElem.IsValid() {
					mapElem = reflect.New(elemType).Elem()
				} else {
					mapElem.SetZero()
				}

				subv = mapElem

				unmarshalWorker(value, subv)

				var kv reflect.Value
				switch keyType.Kind() {
				case reflect.String:
					kv = reflect.New(keyType).Elem()
					kv.SetString(key.String())
				case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
					s := key.String()
					n, err := strconv.ParseInt(s, 10, 64)
					if err != nil {
						break
					}
					kv = reflect.New(keyType).Elem()
					kv.SetInt(n)
				case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
					s := key.String()
					n, err := strconv.ParseUint(s, 10, 64)
					if err != nil {
						break
					}
					kv = reflect.New(keyType).Elem()
					kv.SetUint(n)
				default:
					panic("unmarshal: Unexpected key type") // should never occur
				}
				if kv.IsValid() {
					reflected.SetMapIndex(kv, subv)
				}

			})
		case reflect.Slice:
			i := 0

			value.(*lua.LTable).ForEach(func(key, value lua.LValue) {
				// Expand slice length, growing the slice if necessary.
				if i >= reflected.Cap() {
					reflected.Grow(1)
				}
				if i >= reflected.Len() {
					reflected.SetLen(i + 1)
				}
				if i < reflected.Len() {
					// Decode into element.
					unmarshalWorker(value, reflected.Index(i))
				} else {
					unmarshalWorker(value, reflect.Value{})
				}
				i++
			})

			// Truncate slice if necessary.
			if i < reflected.Len() {
				reflected.SetLen(i)
			}

			if i == 0 {
				reflected.Set(reflect.MakeSlice(reflected.Type(), 0, 0))
			}
		case reflect.Struct:
			for i := 0; i < reflected.NumField(); i++ {
				fieldTypeField := reflected.Type().Field(i)
				tag := fieldTypeField.Tag.Get("luai")
				if tag != "" {
					tagMap[tag] = i
				}
			}

			(value.(*lua.LTable)).ForEach(func(key, value lua.LValue) {
				fieldName := key.String()

				field := reflected.FieldByName(fieldName)

				// if field is not found, try to find it by tag
				if !field.IsValid() {
					fieldIndex, ok := tagMap[fieldName]
					if !ok {
						return
					}
					field = reflected.Field(fieldIndex)
				}

				if !field.IsValid() {
					return
				}

				unmarshalWorker(value, field)
			})
		}
	default:
		switch reflected.Kind() {
		case reflect.Interface:
			// Decoding into nil interface? Switch to non-reflect code.
			if reflected.NumMethod() == 0 {
				result := valueInterface(value)
				reflected.Set(reflect.ValueOf(result))
			}
		default:
			storeLiteral(reflected, value)
		}
	}
	return nil
}

func Unmarshal(value lua.LValue, v any) error {
	reflected := reflect.ValueOf(v)

	if reflected.Kind() != reflect.Pointer || reflected.IsNil() {
		return errors.New("unmarshal: value must be a pointer")
	}

	return unmarshalWorker(value, reflected)
}
