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
	"errors"
	"fmt"
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

func storeLiteral(value reflect.Value, lvalue lua.LValue) error {
	value = indirect(value)
	mismatch := func() error {
		return fmt.Errorf("cannot unmarshal %s into %s", lvalue.Type(), value.Type())
	}
	switch value.Kind() {
	case reflect.String:
		// Retain the existing Lua scalar-to-string conversion, including numbers.
		if lvalue.Type() != lua.LTString && lvalue.Type() != lua.LTNumber && lvalue.Type() != lua.LTBool {
			return mismatch()
		}
		value.SetString(lvalue.String())
	case reflect.Bool:
		v, ok := lvalue.(lua.LBool)
		if !ok {
			return mismatch()
		}
		value.SetBool(bool(v))
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		v, ok := lvalue.(lua.LNumber)
		if !ok {
			return mismatch()
		}
		value.SetInt(int64(v))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		v, ok := lvalue.(lua.LNumber)
		if !ok {
			return mismatch()
		}
		value.SetUint(uint64(v))
	case reflect.Float32, reflect.Float64:
		v, ok := lvalue.(lua.LNumber)
		if !ok {
			return mismatch()
		}
		value.SetFloat(float64(v))
	default:
		return mismatch()
	}
	return nil
}

func objectInterface(lvalue *lua.LTable) any {
	var v = make(map[string]any)
	lvalue.ForEach(func(key, value lua.LValue) {
		v[key.String()] = valueInterface(value)
	})
	return v
}

// To unmarshal lua obj into an interface value,
// Unmarshal stores one of these in the interface value:
//
//   - bool, for LTBool
//   - float64, for LTNumber
//   - string, for LTString
//   - []interface{}, for LTTable arrays
//   - map[string]interface{}, for LTTable objects
//   - nil for LTNil
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
		return float64(lvalue.(lua.LNumber))
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
	if !reflected.IsValid() || !reflected.CanSet() {
		return errors.New("unmarshal: destination cannot be set")
	}
	if value == nil || value == lua.LNil {
		switch reflected.Kind() {
		case reflect.Pointer, reflect.Interface, reflect.Map, reflect.Slice:
			reflected.SetZero()
			return nil
		default:
			return fmt.Errorf("cannot unmarshal nil into %s", reflected.Type())
		}
	}
	reflected = indirect(reflected)
	if reflected.Kind() == reflect.Interface && reflected.NumMethod() == 0 {
		result := valueInterface(value)
		if result == nil {
			return fmt.Errorf("cannot unmarshal %s into interface", value.Type())
		}
		reflected.Set(reflect.ValueOf(result))
		return nil
	}
	table, ok := value.(*lua.LTable)
	if !ok {
		return storeLiteral(reflected, value)
	}

	var decodeErr error
	switch reflected.Kind() {
	case reflect.Map:
		t := reflected.Type()
		keyType := t.Key()
		if reflected.IsNil() {
			reflected.Set(reflect.MakeMap(t))
		}
		table.ForEach(func(key, value lua.LValue) {
			if decodeErr != nil {
				return
			}
			kv := reflect.New(keyType).Elem()
			switch keyType.Kind() {
			case reflect.String:
				kv.SetString(key.String())
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				n, err := strconv.ParseInt(key.String(), 10, keyType.Bits())
				if err != nil {
					decodeErr = err
					return
				}
				kv.SetInt(n)
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
				n, err := strconv.ParseUint(key.String(), 10, keyType.Bits())
				if err != nil {
					decodeErr = err
					return
				}
				kv.SetUint(n)
			default:
				decodeErr = fmt.Errorf("unmarshal: unsupported map key type %s", keyType)
				return
			}
			elem := reflect.New(t.Elem()).Elem()
			if err := unmarshalWorker(value, elem); err != nil {
				decodeErr = fmt.Errorf("[%s]: %w", key, err)
				return
			}
			reflected.SetMapIndex(kv, elem)
		})
	case reflect.Slice:
		length := table.Len()
		reflected.Set(reflect.MakeSlice(reflected.Type(), length, length))
		table.ForEach(func(key, value lua.LValue) {
			if decodeErr != nil {
				return
			}
			n, ok := key.(lua.LNumber)
			if !ok || n < 1 || n > lua.LNumber(length) || n != lua.LNumber(int(n)) {
				decodeErr = fmt.Errorf("unmarshal: expected array index, got %s", key)
				return
			}
			if err := unmarshalWorker(value, reflected.Index(int(n)-1)); err != nil {
				decodeErr = fmt.Errorf("[%d]: %w", int(n), err)
			}
		})
		if decodeErr == nil {
			for i := 1; i <= length; i++ {
				if table.RawGetInt(i) == lua.LNil {
					return fmt.Errorf("unmarshal: missing array item [%d]", i)
				}
			}
		}
	case reflect.Struct:
		// Keep the flat protocol for embedded result and checksum structs.
		for i := 0; i < reflected.NumField(); i++ {
			f := reflected.Type().Field(i)
			field := reflected.Field(i)
			if f.Anonymous && field.CanSet() && field.Kind() == reflect.Ptr && field.IsNil() {
				field.Set(reflect.New(field.Type().Elem()))
			}
		}
		table.ForEach(func(key, value lua.LValue) {
			if decodeErr != nil {
				return
			}
			field := findField(reflected, key.String())
			if !field.IsValid() || !field.CanSet() {
				return
			}
			if err := unmarshalWorker(value, field); err != nil {
				decodeErr = fmt.Errorf("%s: %w", key, err)
			}
		})
	default:
		return fmt.Errorf("cannot unmarshal table into %s", reflected.Type())
	}
	return decodeErr
}

// findField finds a field in the struct, including embedded fields recursively
func findField(reflected reflect.Value, fieldName string) reflect.Value {
	// First, try direct field name
	field := reflected.FieldByName(fieldName)
	if field.IsValid() {
		return field
	}

	// Try tags
	t := reflected.Type()
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		tag := f.Tag.Get("json")
		if tag == fieldName {
			return reflected.Field(i)
		}
	}

	// Now, check embedded fields
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if f.Anonymous {
			embeddedValue := reflected.Field(i)
			if embeddedValue.Kind() == reflect.Ptr {
				if embeddedValue.IsNil() {
					embeddedValue.Set(reflect.New(embeddedValue.Type().Elem()))
				}
				embeddedValue = embeddedValue.Elem()
			} else if embeddedValue.Kind() == reflect.Struct {
				// ok
			} else {
				continue
			}
			subField := findField(embeddedValue, fieldName)
			if subField.IsValid() {
				return subField
			}
		}
	}
	return reflect.Value{}
}

func Unmarshal(value lua.LValue, v any) error {
	reflected := reflect.ValueOf(v)

	if reflected.Kind() != reflect.Pointer || reflected.IsNil() {
		return errors.New("unmarshal: value must be a pointer")
	}

	return unmarshalWorker(value, reflected.Elem())
}
