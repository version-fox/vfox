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

package util

import (
	"encoding/json"
	"github.com/version-fox/vfox/config"
	"github.com/version-fox/vfox/go/pkg/mod/github.com/fatih/structs@v1.1.0"
	lua "github.com/yuin/gopher-lua"
	"net/http"
	"net/url"
)

const HTTP_PROXY_SETTING = "httpProxySetting"

type NewProxyStruct struct {
	proxyUrl    string
	proxyEnable bool
}

func (s *NewProxyStruct) GetByURL(targetUrl string) (resp *http.Response, err error) {
	s.initProxySetting()
	if s.proxyEnable != true {
		return http.Get(targetUrl)
	}
	proxyAddr := s.proxyUrl
	proxy, err := url.Parse(proxyAddr)
	netTransport := &http.Transport{
		Proxy: http.ProxyURL(proxy),
	}
	//httpClient := http.Client{
	//	Transport: netTransport,
	//}
	//res, err := http.NewRequest("GET", targetUrl)
	return nil
}
func (s *NewProxyStruct) initProxySetting() {
	luaInstance := lua.NewState()
	proxyConfig := config.NewProxyConfig{}
	proxyConfig.InitConfigInfo()
	//
	jsonLValue := luaInstance.GetGlobal(HTTP_PROXY_SETTING)
	mp := structs.Map(jsonLValue)
	arr, _ := json.Marshal(mp)
	json.Unmarshal(arr, &s)
}
