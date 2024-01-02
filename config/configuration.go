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

package config

import (
	"encoding/json"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
)

type NetworkProxy struct {
	proxyUrl    string `json:"proxyUrl"`
	enableProxy bool   `json:"enableProxy"`
}

const configFilePath = ".\\vfox-setting.yaml"

func NewNetWorkProxy() *NetworkProxy {
	networkProxy := NetworkProxy{}
	return networkProxy.getInstance()
}
func (proxyInstance *NetworkProxy) getInstance() *NetworkProxy {
	proxyInstance.getGlobalProxyInfo()
	return proxyInstance
}
func (proxyInstance *NetworkProxy) getGlobalProxyInfo() {
	_, err := os.Stat(configFilePath)
	if os.IsNotExist(err) {
		proxyInstance.enableProxy = false
		return
	}
	data, _ := os.ReadFile(configFilePath)
	mp := make(map[string]any, 2)
	yaml.Unmarshal(data, mp)
	arr, err := json.Marshal(mp)
	err = json.Unmarshal(arr, &proxyInstance)
}
func (proxyInstance *NetworkProxy) updateNetworkProxyInfo() {
	_, err := os.Stat(configFilePath)
	if os.IsNotExist(err) {
		data, _ := yaml.Marshal(proxyInstance)
		ioutil.WriteFile(configFilePath, data, 0644)
		return
	}
	mp := make(map[string]any, 2)
	data, _ := os.ReadFile(configFilePath)
	yaml.Unmarshal(data, mp)
	//mp["httpProxy"] = structs.Map(proxyInfo)
	//mp["httpProxy"] = structs.Map(proxyInfo)
	mpProxyInfo := make(map[string]any, 2)
	proxyInfoJSON, _ := yaml.Marshal(proxyInstance)
	mp["httpProxyConfig"] = yaml.Unmarshal(proxyInfoJSON, &mpProxyInfo)
	data, _ = yaml.Marshal(mp)
	ioutil.WriteFile(configFilePath, data, 0644)
	return
}
func (proxyInstance *NetworkProxy) GetByURL(targetUrl string) (resp *http.Response, err error) {
	if proxyInstance.enableProxy == false {
		return http.Get(targetUrl)
	}
	proxy, err := url.Parse(proxyInstance.proxyUrl)
	netTransport := &http.Transport{
		Proxy: http.ProxyURL(proxy),
	}
	httpClient := &http.Client{
		Transport: netTransport,
	}
	req, err := http.NewRequest("GET", targetUrl, nil)
	return httpClient.Do(req)
}

func (proxyInstance *NetworkProxy) SetProxy(proxyUrl string) error {
	if len(proxyUrl) == 0 {
		proxyInstance.enableProxy = false
		proxyInstance.proxyUrl = ""
		return nil
	}
	proxyInstance.proxyUrl = proxyUrl
	proxyInstance.enableProxy = true
	return nil
}
