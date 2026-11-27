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
	"math"
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
	errNested         = errors.New("cannot encode recursively nested tables to JSON")
	errSparseArray    = errors.New("cannot encode sparse array")
	errInvalidKeys    = errors.New("cannot encode mixed or invalid key types")
	errTooManyRepeats = errors.New("cannot encode: shared tables are referenced too many times")
	errTooLarge       = errors.New("cannot encode more than 64 MiB of JSON")
	errTooDeep        = errors.New("cannot encode tables nested more than 256 levels deep to JSON")
	errNotUTF8        = errors.New("cannot encode a string that is not valid UTF-8 to JSON")
	errNotFinite      = errors.New("cannot encode a number that is not finite (NaN or infinity) to JSON")
)

// maxRepeatValues caps the values written for tables that were already encoded once
// in the same call. A table referenced several times is encoded once per reference,
// so a few levels of sharing can expand into a tree with exponentially many values.
// Data without shared tables never counts against it. It is a variable only so tests
// can lower it.
var maxRepeatValues = 1 << 20

// maxBytes caps the output of EncodeStrict. A shared string counts once per
// reference, so a small table can otherwise expand into an output of any size.
var maxBytes = 64 << 20

// maxDepth caps table nesting in EncodeStrict. Each level re-validates the output of
// the level below it, so the cost grows with depth, and decode refuses data nested
// more than 10000 levels, so a deeper value could be stored but never read back.
const maxDepth = 256

type invalidTypeError lua.LValueType

func (i invalidTypeError) Error() string {
	return `cannot encode ` + lua.LValueType(i).String() + ` to JSON`
}

// Encode returns the JSON encoding of value. A string that is not valid UTF-8 is
// written with each bad byte replaced by U+FFFD, as encoding/json does. A table that
// is referenced more than once is written once per reference, and the values written
// for such repeat references are capped.
func Encode(value lua.LValue) ([]byte, error) {
	return encode(value, false)
}

// EncodeStrict is Encode for data that is stored and read back. It also returns an
// error for any string or table key that is not valid UTF-8, for tables nested more
// than 256 levels deep, for NaN and infinity, and for output larger than 64 MiB. Its
// errors carry the encoder's own message, without encoding/json's wrapping.
func EncodeStrict(value lua.LValue) ([]byte, error) {
	return encode(value, true)
}

func encode(value lua.LValue, strict bool) ([]byte, error) {
	e := &encoder{
		visited: make(map[*lua.LTable]bool),
		done:    make(map[*lua.LTable]bool),
		strict:  strict,
	}
	data, err := json.Marshal(jsonValue{LValue: value, enc: e})
	if err != nil && strict {
		// encoding/json wraps an error from a nested MarshalJSON once per level. Strict
		// callers get the original message, without those prefixes.
		var wrapped *json.MarshalerError
		for errors.As(err, &wrapped) && wrapped.Err != nil {
			err = wrapped.Err
		}
	}
	return data, err
}

// encoder holds the state shared by every value of one encode call.
type encoder struct {
	visited map[*lua.LTable]bool // tables on the current path, to find cycles
	done    map[*lua.LTable]bool // tables already written once
	repeats int                  // values written inside repeat references
	bytes   int                  // estimated output size, strict only
	strict  bool
}

// add charges n bytes of output against the budget. EncodeStrict charges every
// value. Plain Encode charges only values written inside a repeat reference, so
// ordinary data is never limited but a shared table cannot expand without bound.
func (e *encoder) add(n int, repeat bool) error {
	if !e.strict && !repeat {
		return nil
	}
	if e.bytes += n; e.bytes > maxBytes {
		return errTooLarge
	}
	return nil
}

type jsonValue struct {
	lua.LValue
	enc    *encoder
	depth  int
	repeat bool // inside a table that was already written once
}

func (j jsonValue) child(v lua.LValue) jsonValue {
	j.LValue = v
	j.depth++
	return j
}

func (j jsonValue) MarshalJSON() (data []byte, err error) {
	e := j.enc
	if j.repeat {
		if e.repeats++; e.repeats > maxRepeatValues {
			return nil, errTooManyRepeats
		}
	}
	switch converted := j.LValue.(type) {
	case lua.LBool:
		data, err = json.Marshal(bool(converted))
		if err == nil {
			err = e.add(len(data), j.repeat)
		}
	case lua.LNumber:
		if e.strict && (math.IsNaN(float64(converted)) || math.IsInf(float64(converted), 0)) {
			return nil, errNotFinite
		}
		data, err = json.Marshal(float64(converted))
		if err == nil {
			err = e.add(len(data), j.repeat)
		}
	case *lua.LNilType:
		data = []byte(`null`)
		err = e.add(len(data), j.repeat)
	case lua.LString:
		if e.strict && !utf8.ValidString(string(converted)) {
			return nil, errNotUTF8
		}
		// charged before encoding, so a huge string is refused without copying it
		if e.strict || j.repeat {
			if err = e.add(quotedLen(string(converted)), j.repeat); err != nil {
				return nil, err
			}
		}
		data, err = json.Marshal(string(converted))
	case *lua.LTable:
		if e.visited[converted] {
			return nil, errNested
		}
		if e.strict && j.depth >= maxDepth {
			return nil, errTooDeep
		}
		e.visited[converted] = true
		// only a table that contains itself is recursive; the same table twice is not
		defer func() {
			delete(e.visited, converted)
			e.done[converted] = true
		}()
		if e.done[converted] {
			j.repeat = true
		}
		if err = e.add(2, j.repeat); err != nil {
			return nil, err
		}

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
				if err = e.add(1, j.repeat); err != nil {
					return nil, err
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
				if e.strict && !utf8.ValidString(key.String()) {
					err = errNotUTF8
					return
				}
				if err = e.add(quotedLen(key.String())+2, j.repeat); err != nil {
					return nil, err
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

// quotedLen returns an upper bound for the size of s once it is written as a JSON
// string: the quotes, plus six bytes for every character that encoding/json
// escapes as \\uXXXX (<, >, &, U+2028, U+2029, control bytes and invalid UTF-8),
// two for a quote, a backslash or a short escape, and the raw length otherwise.
func quotedLen(s string) int {
	n := 2
	for i := 0; i < len(s); {
		b := s[i]
		if b < utf8.RuneSelf {
			switch {
			case b == '"' || b == '\\' || b == '\n' || b == '\r' || b == '\t':
				n += 2
			case b < 0x20 || b == '<' || b == '>' || b == '&':
				n += 6
			default:
				n++
			}
			i++
			continue
		}
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError && size == 1 || r == '\u2028' || r == '\u2029' {
			n += 6
		} else {
			n += size
		}
		i += size
	}
	return n
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
