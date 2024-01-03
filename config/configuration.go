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
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
)

type NetworkProxy struct {
	ProxyUrl    string `yaml:"ProxyUrl"`
	EnableProxy bool   `yaml:"EnableProxy"`
}

const configFilePath = ".\\vfox-proxy-setting.yaml"

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
		proxyInstance.EnableProxy = false
		return
	}
	data, _ := os.ReadFile(configFilePath)
	nper := yaml.Unmarshal(data, proxyInstance)
	if nper != nil {
		fmt.Printf("nper")
	}
}
func (proxyInstance *NetworkProxy) updateNetworkProxyInfo() {
	_, err := os.Stat(configFilePath)
	mpProxyInfo := make(map[string]any)
	mpProxyInfo["ProxyUrl"] = proxyInstance.ProxyUrl
	mpProxyInfo["EnableProxy"] = proxyInstance.EnableProxy
	if os.IsNotExist(err) {
		data, _ := yaml.Marshal(mpProxyInfo)
		ioutil.WriteFile(configFilePath, data, 0644)
		return
	}
	actData, _ := yaml.Marshal(mpProxyInfo)
	ioutil.WriteFile(configFilePath, actData, 0644)
	return
}
func (proxyInstance *NetworkProxy) GetByURL(targetUrl string) (resp *http.Response, err error) {
	if proxyInstance.EnableProxy == false {
		return http.Get(targetUrl)
	}
	proxy, err := url.Parse(proxyInstance.ProxyUrl)
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
		proxyInstance.EnableProxy = false
		proxyInstance.ProxyUrl = ""
		return nil
	}
	proxyInstance.ProxyUrl = proxyUrl
	proxyInstance.EnableProxy = true
	proxyInstance.updateNetworkProxyInfo()
	return nil
}
