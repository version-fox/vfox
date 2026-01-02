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
