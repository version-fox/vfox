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

package json

import (
	"encoding/json"
	"testing"

	"github.com/yuin/gopher-lua"
)

func TestSimple(t *testing.T) {
	const str = `
	local json = require("json")
	assert(type(json) == "table")
	assert(type(json.decode) == "function")
	assert(type(json.encode) == "function")

	assert(json.encode(true) == "true")
	assert(json.encode(1) == "1")
	assert(json.encode(-10) == "-10")
	assert(json.encode(nil) == "null")
	assert(json.encode({}) == "[]")
	assert(json.encode({1, 2, 3}) == "[1,2,3]")

	local _, err = json.encode({1, 2, [10] = 3})
	assert(string.find(err, "sparse array"))

	local _, err = json.encode({1, 2, 3, name = "Tim"})
	assert(string.find(err, "mixed or invalid key types"))

	local _, err = json.encode({name = "Tim", [false] = 123})
	assert(string.find(err, "mixed or invalid key types"))

	local obj = {"a",1,"b",2,"c",3}
	local jsonStr = json.encode(obj)
	local jsonObj = json.decode(jsonStr)
	for i = 1, #obj do
		assert(obj[i] == jsonObj[i])
	end

	local obj = {name="Tim",number=12345}
	local jsonStr = json.encode(obj)
	local jsonObj = json.decode(jsonStr)
	assert(obj.name == jsonObj.name)
	assert(obj.number == jsonObj.number)

	assert(json.decode("null") == nil)

	assert(json.decode(json.encode({person={name = "tim",}})).person.name == "tim")

	local obj = {
		abc = 123,
		def = nil,
	}
	local obj2 = {
		obj = obj,
	}
	obj.obj2 = obj2
	assert(json.encode(obj) == nil)

	local a = {}
	for i=1, 5 do
		a[i] = i
	end
	assert(json.encode(a) == "[1,2,3,4,5]")
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
	assert(type(j) == "table")
	assert(type(j.decode) == "function")
	assert(type(j.encode) == "function")
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
