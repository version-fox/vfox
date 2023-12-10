/*
 *    Copyright 2023 [lihan aooohan@gmail.com]
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

package lua

import (
	lua "github.com/yuin/gopher-lua"
	"io"
	"net/http"
)

const metatableName = "request_metatable"

func RegisterHttpClient(L *lua.LState) {
	mt := L.NewTypeMetatable(metatableName)
	L.SetGlobal("http_client", mt)
	L.SetField(mt, "__index", L.SetFuncs(L.NewTable(), map[string]lua.LGFunction{
		"get":  getRequest,
		"post": getRequest,
	}))
}

//	http_client:get({
//		url = "http://www.baidu.com",j
//		headers = {
//			["User-Agent"] = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) "
//	  }
//	}) returns (response, error)
//
//	response: {
//	  code = http_code (200, 201, ..., 500, ...),
//	  body = string
//	  headers = table
//	}
func getRequest(L *lua.LState) int {
	param := L.CheckTable(1)
	url := param.RawGetString("url")
	if url == lua.LNil {
		L.Push(lua.LNil)
		L.Push(lua.LString("url is required"))
	}
	client := &http.Client{}
	req, err := http.NewRequest(url.String(), url.String(), nil)
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(err.Error()))
		return 2
	}

	headersTable := param.RawGetString("headers")
	if headersTable != lua.LNil {
		if table, ok := headersTable.(*lua.LTable); ok {
			table.ForEach(func(key lua.LValue, value lua.LValue) {
				req.Header.Add(key.String(), value.String())
			})
		}
	}
	resp, err := client.Do(req)
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(err.Error()))
		return 2
	}
	defer resp.Body.Close()
	headers := L.NewTable()
	for k, v := range resp.Header {
		if len(v) > 0 {
			headers.RawSetString(k, lua.LString(v[0]))
		}
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(err.Error()))
		return 2
	}
	result := L.NewTable()
	L.SetField(result, "body", lua.LString(body))
	L.SetField(result, "status_code", lua.LNumber(resp.StatusCode))
	L.SetField(result, "headers", headers)
	L.Push(result)
	return 1
}
