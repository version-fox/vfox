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

package http

import (
	"github.com/version-fox/vfox/internal/config"
	lua "github.com/yuin/gopher-lua"
	"io"
	"net/http"
	"net/url"
)

type Module struct {
	proxy  *config.Proxy
	client *http.Client
}

// Get performs a http get request
// @param url string
// @param headers table
// @return resp table
// @return err string
// local http = require("http")
//
//	http.get({
//	    url = "http://ip.jsontest.com/"
//	}) return (response, error)
//
//	response : {
//	    body = "",
//	    status_code = 200,
//	    headers = table
//	}
func (m *Module) Get(L *lua.LState) int {
	param := L.CheckTable(1)
	urlStr := param.RawGetString("url")
	if urlStr == lua.LNil {
		L.Push(lua.LNil)
		L.Push(lua.LString("url is required"))
	}

	req, err := http.NewRequest("GET", urlStr.String(), nil)
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
	resp, err := m.client.Do(req)
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
	L.SetField(result, "content_length", lua.LNumber(resp.ContentLength))
	L.Push(result)
	return 1
}

func (m *Module) Head(L *lua.LState) int {
	param := L.CheckTable(1)
	urlStr := param.RawGetString("url")
	if urlStr == lua.LNil {
		L.Push(lua.LNil)
		L.Push(lua.LString("url is required"))
	}

	req, err := http.NewRequest("HEAD", urlStr.String(), nil)
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
	resp, err := m.client.Do(req)
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(err.Error()))
		return 2
	}
	headers := L.NewTable()
	for k, v := range resp.Header {
		if len(v) > 0 {
			headers.RawSetString(k, lua.LString(v[0]))
		}
	}
	result := L.NewTable()
	L.SetField(result, "status_code", lua.LNumber(resp.StatusCode))
	L.SetField(result, "headers", headers)
	L.SetField(result, "content_length", lua.LNumber(resp.ContentLength))
	L.Push(result)
	return 1
}

func (m *Module) luaMap() map[string]lua.LGFunction {
	return map[string]lua.LGFunction{
		"get":  m.Get,
		"head": m.Head,
	}
}

func NewModule(proxy *config.Proxy) lua.LGFunction {
	return func(L *lua.LState) int {
		client := &http.Client{}
		if proxy.Enable {
			uri, err := url.Parse(proxy.Url)
			if err == nil {
				transPort := &http.Transport{
					Proxy: http.ProxyURL(uri),
				}
				client = &http.Client{
					Transport: transPort,
				}
			}
		}
		m := &Module{proxy: proxy, client: client}
		t := L.NewTable()
		L.SetFuncs(t, m.luaMap())
		L.Push(t)
		return 1
	}
}
