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
	"strings"
	"testing"

	lua "github.com/yuin/gopher-lua"
)

func TestSimple(t *testing.T) {
	const str = `
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

	local json = require("json")
	assert(type(json) == "table", "json is not a table")
	assert(type(json.decode) == "function", "json.decode is not a function")
	assert(type(json.encode) == "function", "json.encode is not a function")

	assert(json.encode(true) == "true", "json.encode(true) is not 'true'")
	assert(json.encode(1) == "1", "json.encode(1) is not '1'")
	assert(json.encode(-10) == "-10", "json.encode(-10) is not '-10'")
	assert(json.encode(nil) == "null", "json.encode(nil) is not 'null'")
	assert(json.encode({}) == "[]", "json.encode({}) is not '[]'")
	assert(json.encode({1, 2, 3}) == "[1,2,3]", "json.encode({1, 2, 3}) is not '[1,2,3]'")

	local _, err = json.encode({1, 2, [10] = 3})
	assert(string.find(err, "sparse array"), "expected sparse array error, got: " .. (err or "nil"))

	local _, err = json.encode({1, 2, 3, name = "Tim"})
	assert(string.find(err, "mixed or invalid key types"), "expected mixed or invalid key types error, got: " .. (err or "nil"))

	local _, err = json.encode({name = "Tim", [false] = 123})
	assert(string.find(err, "mixed or invalid key types"), "expected mixed or invalid key types error, got: " .. (err or "nil"))

	local obj = {"a",1,"b",2,"c",3}
	local jsonStr = json.encode(obj)
	local jsonObj = json.decode(jsonStr)
	for i = 1, #obj do
		assert(obj[i] == jsonObj[i], "obj[" .. i .. "] is not equal to jsonObj[" .. i .. "]")
	end

	local obj = {name="Tim",number=12345}
	local jsonStr = json.encode(obj)
	local jsonObj = json.decode(jsonStr)
	assert(obj.name == jsonObj.name, "obj.name is not equal to jsonObj.name")
	assert(obj.number == jsonObj.number, "obj.number is not equal to jsonObj.number")


	local table1 = json.decode("{\"metadata\":[],\"spec\":{\"containers\":[{\"image\":\"centos:7\",\"name\":\"centos\",\"resources\":[]}]},\"status\":[]}")
	printTable(table1)
	--- assert table1.metadata is a empty table
	assert(type(table1.metadata) == "table", "table1.metadata is not an empty table")
	assert(next(table1.metadata) == nil, "table1.metadata is not an empty table")
	--- assert table1.status is a empty table
	assert(next(table1.status) == nil, "table1.status is not an empty table")
	--- assert table1.spec.containers is a table
	assert(type(table1.spec.containers) == "table", "table1.spec.containers is not a table")
	--- assert table1.spec.containers[1] is a table
	assert(type(table1.spec.containers[1]) == "table", "table1.spec.containers[1] is not a table")
	--- assert table1.spec.containers[1].image is a string
	assert(type(table1.spec.containers[1].image) == "string", "table1.spec.containers[1].image is not a string")
	--- assert table1.spec.containers[1].resources is a table
	assert(type(table1.spec.containers[1].resources) == "table", "table1.spec.containers[1].resources is not a table")

	
	assert(json.decode("null") == nil, "json.decode('null') is not nil")

	assert(json.decode(json.encode({person={name = "tim",}})).person.name == "tim", "json.decode(json.encode({person={name = 'tim',}})).person.name is not 'tim'")

	local obj = {
		abc = 123,
		def = nil,
	}
	local obj2 = {
		obj = obj,
	}
	obj.obj2 = obj2
	assert(json.encode(obj) == nil, "json.encode(obj) is not nil")

	local a = {}
	for i=1, 5 do
		a[i] = i
	end
	assert(json.encode(a) == "[1,2,3,4,5]", "json.encode(a) is not '[1,2,3,4,5]'")
	`
	s := lua.NewState()
	defer s.Close()

	Preload(s)
	if err := s.DoString(str); err != nil {
		t.Error(err)
	}
}

func TestCustomRequire(t *testing.T) {
	const str = `
	local j = require("JSON")
	assert(type(j) == "table", "j is not a table")
	assert(type(j.decode) == "function", "j.decode is not a function")
	assert(type(j.encode) == "function", "j.encode is not a function")
	`
	s := lua.NewState()
	defer s.Close()

	s.PreloadModule("JSON", loader)
	if err := s.DoString(str); err != nil {
		t.Error(err)
	}
}

func TestDecodeValue_jsonNumber(t *testing.T) {
	s := lua.NewState()
	defer s.Close()

	v := DecodeValue(s, json.Number("124.11"))
	if v.Type() != lua.LTString || v.String() != "124.11" {
		t.Fatalf("expecting LString, got %T", v)
	}
}

func TestEncodeSharedSubtableIsNotRecursion(t *testing.T) {
	const str = `
	local json = require("JSON")
	local shared = {1, 2}
	assert(json.encode({a = shared, b = shared}) == '{"a":[1,2],"b":[1,2]}')
	local loop = {}
	loop.self = loop
	local _, err = json.encode(loop)
	assert(string.find(err, "recursively nested"), "expected recursion error, got: " .. (err or "nil"))
	`
	s := lua.NewState()
	defer s.Close()
	s.PreloadModule("JSON", loader)
	if err := s.DoString(str); err != nil {
		t.Error(err)
	}
}

func TestEncodeSharedSubtableFanOutIsCapped(t *testing.T) {
	old := maxRepeatValues
	maxRepeatValues = 10000
	defer func() { maxRepeatValues = old }()
	const str = `
	local json = require("JSON")
	local t = {1}
	for i = 1, 40 do t = { a = t, b = t } end
	local out, err = json.encode(t)
	assert(out == nil and string.find(err, "shared tables"), "expected a size error, got: " .. tostring(err))
	-- a large but ordinary list still encodes
	local list = {}
	for i = 1, 5000 do list[i] = i end
	assert(json.encode(list) ~= nil)
	`
	s := lua.NewState()
	defer s.Close()
	s.PreloadModule("JSON", loader)
	if err := s.DoString(str); err != nil {
		t.Error(err)
	}
}

func TestEncodeLimits(t *testing.T) {
	const str = `
	local json = require("JSON")
	local function nest(n)
		local t = {1}
		for i = 2, n do t = { t } end
		return t
	end
	-- plain encode keeps accepting deep nesting, as it always did
	local out = assert(json.encode(nest(300)))
	assert(json.decode(out) ~= nil)
	-- a cycle is reported as a cycle, including a fan-out of self references
	local loop = {}
	loop.a = loop
	loop.b = loop
	local _, err5 = json.encode(loop)
	assert(string.find(err5, "recursively nested"), tostring(err5))
	-- non-strict encode keeps replacing bad bytes, as before
	assert(json.decode(json.encode({"\255"}))[1] == "\239\191\189")
	`
	s := lua.NewState()
	defer s.Close()
	s.PreloadModule("JSON", loader)
	if err := s.DoString(str); err != nil {
		t.Error(err)
	}
}

func TestEncodeLargePlainDataIsNotLimited(t *testing.T) {
	s := lua.NewState()
	defer s.Close()
	l := s.NewTable()
	for i := 0; i < 1100000; i++ {
		l.Append(lua.LNumber(1))
	}
	if _, err := Encode(l); err != nil {
		t.Fatalf("plain list of 1.1M items must encode: %v", err)
	}
	if _, err := EncodeStrict(l); err != nil {
		t.Fatalf("strict list of 1.1M small items is under the byte budget: %v", err)
	}
}

func TestEncodeStrictLimits(t *testing.T) {
	s := lua.NewState()
	defer s.Close()
	nest := func(n int) lua.LValue {
		var v lua.LValue = lua.LNumber(1)
		for i := 0; i < n; i++ {
			t := s.NewTable()
			t.Append(v)
			v = t
		}
		return v
	}
	out, err := EncodeStrict(nest(256))
	if err != nil {
		t.Fatalf("256 levels must encode: %v", err)
	}
	if v, err := Decode(s, out); err != nil || v == nil {
		t.Fatalf("256 levels must decode: %v", err)
	}
	if _, err := EncodeStrict(nest(257)); err == nil || !strings.Contains(err.Error(), "256 levels") {
		t.Fatalf("257 levels must be refused, got %v", err)
	}
	if _, err := EncodeStrict(nest(200000)); err == nil || !strings.Contains(err.Error(), "levels") {
		t.Fatalf("a very deep chain must fail fast, got %v", err)
	}
	// plain encode still accepts what strict refuses
	if _, err := Encode(nest(300)); err != nil {
		t.Fatalf("plain encode of 300 levels must work: %v", err)
	}

	// a shared string counts once per reference against the byte budget
	old := maxBytes
	maxBytes = 1 << 20
	defer func() { maxBytes = old }()
	big := lua.LString(strings.Repeat("x", 100<<10))
	l := s.NewTable()
	for i := 0; i < 5; i++ {
		l.Append(big)
	}
	if _, err := EncodeStrict(l); err != nil {
		t.Fatalf("500 KiB must fit in 1 MiB: %v", err)
	}
	for i := 0; i < 10; i++ {
		l.Append(big)
	}
	if _, err := EncodeStrict(l); err == nil || !strings.Contains(err.Error(), "64 MiB") {
		t.Fatalf("1.5 MiB must be refused, got %v", err)
	}
	if _, err := Encode(l); err != nil {
		t.Fatalf("plain encode has no byte budget: %v", err)
	}
}

func TestEncodeStrictRejectsInvalidUTF8(t *testing.T) {
	s := lua.NewState()
	defer s.Close()
	for name, v := range map[string]lua.LValue{
		"string": lua.LString("a\xffb"),
		"nested": func() lua.LValue { t := s.NewTable(); t.Append(lua.LString("\xff")); return t }(),
		"key": func() lua.LValue {
			t := s.NewTable()
			t.RawSetString("\xff", lua.LNumber(1))
			return t
		}(),
	} {
		if _, err := EncodeStrict(v); err == nil || !strings.Contains(err.Error(), "UTF-8") {
			t.Errorf("%s: expected a UTF-8 error, got %v", name, err)
		}
		if _, err := Encode(v); err != nil {
			t.Errorf("%s: Encode must keep accepting it, got %v", name, err)
		}
	}
	if out, err := EncodeStrict(lua.LString("€")); err != nil || string(out) != `"€"` {
		t.Errorf("valid UTF-8 must pass: %s %v", out, err)
	}
}

func TestEncodeRepeatBudgetBoundary(t *testing.T) {
	old := maxRepeatValues
	maxRepeatValues = 100
	defer func() { maxRepeatValues = old }()
	s := lua.NewState()
	defer s.Close()
	shared := s.NewTable()
	for i := 0; i < 40; i++ {
		shared.Append(lua.LNumber(1))
	}
	// the first reference is free; each later one costs its 41 values
	outer := s.NewTable()
	outer.Append(shared)
	outer.Append(shared)
	outer.Append(shared)
	if _, err := Encode(outer); err != nil {
		t.Fatalf("two repeats (82 values) fit in 100: %v", err)
	}
	outer.Append(shared)
	if _, err := Encode(outer); err == nil || !strings.Contains(err.Error(), "shared tables") {
		t.Fatalf("three repeats must exceed 100, got %v", err)
	}
}
