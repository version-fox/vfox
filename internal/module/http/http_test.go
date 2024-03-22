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
	"testing"

	"github.com/version-fox/vfox/internal/config"
	lua "github.com/yuin/gopher-lua"
)

func TestWithConfig(t *testing.T) {
	const str = `
	local http = require("http")
	assert(type(http) == "table")
	assert(type(http.get) == "function")
	local resp, err = http.get({
        url = "http://ip.jsontest.com/"
    })
	print(err)
	assert(err == 'Get "http://ip.jsontest.com/": proxyconnect tcp: dial tcp 127.0.0.1:80: connect: connection refused')
	`
	s := lua.NewState()
	defer s.Close()

	s.PreloadModule("http", NewModule(&config.Proxy{
		Enable: true,
		Url:    "http://127.0.0.1",
	}))
	if err := s.DoString(str); err != nil {
		t.Error(err)
	}
}
func TestGetRequest(t *testing.T) {
	const str = `
	local http = require("http")
	assert(type(http) == "table")
	assert(type(http.get) == "function")
	local resp, err = http.get({
        url = "http://ip.jsontest.com/"
    })
	assert(err == nil)
	assert(resp.status_code == 200)
	assert(resp.headers['Content-Type'] == 'application/json')
	`
	eval(str, t)
}

func TestHeadRequest(t *testing.T) {
	const str = `
	local http = require("http")
	assert(type(http) == "table")
	assert(type(http.get) == "function")
	local resp, err = http.head({
        url = "http://ip.jsontest.com/"
    })
	assert(err == nil)
	assert(resp.status_code == 200)
	assert(resp.content_length ~= 0)
	`
	eval(str, t)
}

func eval(str string, t *testing.T) {
	s := lua.NewState()
	defer s.Close()

	s.PreloadModule("http", NewModule(config.EmptyProxy))
	if err := s.DoString(str); err != nil {
		t.Error(err)
	}
}
