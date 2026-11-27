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

package json

import (
	"encoding/json"
	"errors"
	"unicode/utf8"

	lua "github.com/yuin/gopher-lua"
)

// Preload adds json to the given Lua state's package.preload table. After it
// has been preloaded, it can be loaded using require:
//
//	local json = require("json")
//
// internal fork from https://github.com/layeh/gopher-json/
func Preload(L *lua.LState) {
	L.PreloadModule("json", loader)
}

// loader is the module loader function.
func loader(L *lua.LState) int {
	t := L.NewTable()
	L.SetFuncs(t, api)
	L.Push(t)
	return 1
}

var api = map[string]lua.LGFunction{
	"decode": apiDecode,
	"encode": apiEncode,
}

func apiDecode(L *lua.LState) int {
	str := L.CheckString(1)

	value, err := Decode(L, []byte(str))
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(err.Error()))
		return 2
	}
	L.Push(value)
	return 1
}

func apiEncode(L *lua.LState) int {
	value := L.CheckAny(1)

	data, err := Encode(value)
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(err.Error()))
		return 2
	}
	L.Push(lua.LString(string(data)))
	return 1
}

var (
	errNested      = errors.New("cannot encode recursively nested tables to JSON")
	errSparseArray = errors.New("cannot encode sparse array")
	errInvalidKeys = errors.New("cannot encode mixed or invalid key types")
	errTooLarge    = errors.New("cannot encode more than 1048576 values to JSON")
	errTooDeep     = errors.New("cannot encode tables nested more than 256 levels deep to JSON")
	errNotUTF8     = errors.New("cannot encode a string that is not valid UTF-8 to JSON")
)

// maxValues caps the values one Encode call may produce. A table that is referenced
// several times is encoded once per reference, so a few levels of sharing can expand
// into a tree with exponentially many values. It is a variable only so tests can
// lower it.
var maxValues = 1 << 20

const (
	// maxDepth caps table nesting. Each level re-validates the output of the level
	// below it, so the cost grows with depth, and the Go stack grows with it too.
	maxDepth = 256
)

type invalidTypeError lua.LValueType

func (i invalidTypeError) Error() string {
	return `cannot encode ` + lua.LValueType(i).String() + ` to JSON`
}

// Encode returns the JSON encoding of value. A string that is not valid UTF-8 is
// written with each bad byte replaced by U+FFFD, as encoding/json does.
func Encode(value lua.LValue) ([]byte, error) {
	return encode(value, false)
}

// EncodeStrict is Encode, but it returns an error for any string or table key that
// is not valid UTF-8, so the encoding always decodes back to the same bytes.
func EncodeStrict(value lua.LValue) ([]byte, error) {
	return encode(value, true)
}

func encode(value lua.LValue, strict bool) ([]byte, error) {
	return json.Marshal(jsonValue{
		LValue:  value,
		visited: make(map[*lua.LTable]bool),
		count:   new(int),
		strict:  strict,
	})
}

type jsonValue struct {
	lua.LValue
	visited map[*lua.LTable]bool
	count   *int
	depth   int
	strict  bool
}

func (j jsonValue) child(v lua.LValue) jsonValue {
	j.LValue = v
	j.depth++
	return j
}

func (j jsonValue) MarshalJSON() (data []byte, err error) {
	if *j.count++; *j.count > maxValues {
		return nil, errTooLarge
	}
	switch converted := j.LValue.(type) {
	case lua.LBool:
		data, err = json.Marshal(bool(converted))
	case lua.LNumber:
		data, err = json.Marshal(float64(converted))
	case *lua.LNilType:
		data = []byte(`null`)
	case lua.LString:
		if j.strict && !utf8.ValidString(string(converted)) {
			return nil, errNotUTF8
		}
		data, err = json.Marshal(string(converted))
	case *lua.LTable:
		if j.visited[converted] {
			return nil, errNested
		}
		if j.depth >= maxDepth {
			return nil, errTooDeep
		}
		j.visited[converted] = true
		// only a table that contains itself is recursive; the same table twice is not
		defer delete(j.visited, converted)

		key, value := converted.Next(lua.LNil)

		switch key.Type() {
		case lua.LTNil: // empty table
			data = []byte(`[]`)
		case lua.LTNumber:
			arr := make([]jsonValue, 0, converted.Len())
			expectedKey := lua.LNumber(1)
			for key != lua.LNil {
				if key.Type() != lua.LTNumber {
					err = errInvalidKeys
					return
				}
				if expectedKey != key {
					err = errSparseArray
					return
				}
				arr = append(arr, j.child(value))
				expectedKey++
				key, value = converted.Next(key)
			}
			data, err = json.Marshal(arr)
		case lua.LTString:
			obj := make(map[string]jsonValue)
			for key != lua.LNil {
				if key.Type() != lua.LTString {
					err = errInvalidKeys
					return
				}
				if j.strict && !utf8.ValidString(key.String()) {
					err = errNotUTF8
					return
				}
				obj[key.String()] = j.child(value)
				key, value = converted.Next(key)
			}
			data, err = json.Marshal(obj)
		default:
			err = errInvalidKeys
		}
	default:
		err = invalidTypeError(j.LValue.Type())
	}
	return
}

// Decode converts the JSON encoded data to Lua values.
func Decode(L *lua.LState, data []byte) (lua.LValue, error) {
	var value interface{}
	err := json.Unmarshal(data, &value)
	if err != nil {
		return nil, err
	}
	return DecodeValue(L, value), nil
}

// DecodeValue converts the value to a Lua value.
//
// This function only converts values that the encoding/json package decodes to.
// All other values will return lua.LNil.
func DecodeValue(L *lua.LState, value interface{}) lua.LValue {
	switch converted := value.(type) {
	case bool:
		return lua.LBool(converted)
	case float64:
		return lua.LNumber(converted)
	case string:
		return lua.LString(converted)
	case json.Number:
		return lua.LString(converted)
	case []interface{}:
		arr := L.CreateTable(len(converted), 0)
		for _, item := range converted {
			arr.Append(DecodeValue(L, item))
		}
		return arr
	case map[string]interface{}:
		tbl := L.CreateTable(0, len(converted))
		for key, item := range converted {
			tbl.RawSetH(lua.LString(key), DecodeValue(L, item))
		}
		return tbl
	case nil:
		return lua.LNil
	}

	return lua.LNil
}
