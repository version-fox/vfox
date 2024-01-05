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
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
)

type NetworkProxy struct {
	Url    string `yaml:"url"`
	Enable bool   `yaml:"enable"`
}

const configFilePath = "config.yaml"

func NewNetWorkProxy(configPath string) *NetworkProxy {
	networkProxy := NetworkProxy{}
	return networkProxy.getSingleton(configPath)
}
func (proxyInstance *NetworkProxy) getSingleton(configPath string) *NetworkProxy {
	proxyInstance.getGlobalProxyInfo(configPath)
	return proxyInstance
}
func (proxyInstance *NetworkProxy) getGlobalProxyInfo(configPath string) {
	filePath := filepath.Join(configPath, configFilePath)
	_, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		proxyInstance.Enable = false
		return
	}
	data, _ := os.ReadFile(filePath)
	mpAll := make(map[string]map[string]any)
	nper := yaml.Unmarshal(data, &mpAll)
	if nper != nil {
		proxyInstance.Enable = false
		return
	}
	mpProxyInfo := make(map[string]any)
	mpProxyInfo = mpAll["httpProxy"]
	yamlProxyInfo, _ := yaml.Marshal(mpProxyInfo)
	yaml.Unmarshal(yamlProxyInfo, proxyInstance)

}
func (proxyInstance *NetworkProxy) updateNetworkProxyInfo(configPath string) {
	filePath := filepath.Join(configPath, configFilePath)
	_, err := os.Stat(filePath)
	mpProxyInfo := make(map[string]any)
	mpProxyInfo["url"] = proxyInstance.Url
	mpProxyInfo["enable"] = proxyInstance.Enable
	mpAll := make(map[string]any)
	mpAll["httpProxy"] = mpProxyInfo
	if os.IsNotExist(err) {
		data, _ := yaml.Marshal(mpAll)
		ioutil.WriteFile(filePath, data, 0644)
		return
	}
	actData, _ := yaml.Marshal(mpAll)
	ioutil.WriteFile(filePath, actData, 0644)
	return
}
func (proxyInstance *NetworkProxy) GetByURL(targetUrl string) (resp *http.Response, err error) {
	if proxyInstance.Enable == false {
		return http.Get(targetUrl)
	}
	proxy, err := url.Parse(proxyInstance.Url)
	netTransport := &http.Transport{
		Proxy: http.ProxyURL(proxy),
	}
	httpClient := &http.Client{
		Transport: netTransport,
	}
	req, err := http.NewRequest("GET", targetUrl, nil)
	return httpClient.Do(req)
}

func (proxyInstance *NetworkProxy) SetProxy(proxyUrl string, configPath string) error {
	if len(proxyUrl) == 0 {
		proxyInstance.Enable = false
		proxyInstance.Url = ""
		return nil
	}
	proxyInstance.Url = proxyUrl
	proxyInstance.Enable = true
	proxyInstance.updateNetworkProxyInfo(configPath)
	return nil
}
