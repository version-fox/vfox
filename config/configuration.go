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
	"github.com/version-fox/vfox/go/pkg/mod/github.com/fatih/structs@v1.1.0"
	lua "github.com/yuin/gopher-lua"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
)

const configFilePath = ".\\vfox-setting.yaml"

// const Verison = "0.1.1";
type httpProxyConfig struct {
	httpProxy NewProxyConfig `json:"httpProxy"`
}
type NewProxyConfig struct {
	proxyUrl    string
	proxyEnable bool
}

const HTTP_PROXY_SETTING = "httpProxySetting"

var configInfo httpProxyConfig

func (p *NewProxyConfig) InitConfigInfo() {
	luaVMInstance := lua.NewState()
	p.readProxySetting()
	json, _ := json.Marshal(configInfo.httpProxy)
	luaVMInstance.SetGlobal(HTTP_PROXY_SETTING, lua.LString((json)))
}
func (p *NewProxyConfig) updateProxySettingFile(config httpProxyConfig) {
	_, err := os.Stat(configFilePath)
	if os.IsNotExist(err) {
		p.createDefaultProxySettingFile()
	}
	data, _ := os.ReadFile(configFilePath)
	proxyInfo := config.httpProxy
	mp := make(map[string]any, 2)
	yaml.Unmarshal(data, mp)
	mp["httpProxy"] = structs.Map(proxyInfo)
	data, _ = yaml.Marshal(mp)
	ioutil.WriteFile(configFilePath, data, 0644)
	configInfo = config
	//
	p.InitConfigInfo()
}
func (p *NewProxyConfig) createDefaultProxySettingFile() {
	proxyInfo := NewProxyConfig{}
	proxyInfo.proxyEnable = false
	config := httpProxyConfig{}
	config.httpProxy = proxyInfo
	data, _ := yaml.Marshal(&config)
	ioutil.WriteFile(configFilePath, data, 0644)
	configInfo = config
	return
}
func (p *NewProxyConfig) readProxySetting() {
	_, err := os.Stat(configFilePath)
	if os.IsNotExist(err) {
		p.createDefaultProxySettingFile()
		p.InitConfigInfo()
		return
	}
	data, _ := os.ReadFile(configFilePath)
	config := httpProxyConfig{}
	mp := make(map[string]any, 2)
	yaml.Unmarshal(data, mp)
	arr, err := json.Marshal(mp)
	err = json.Unmarshal(arr, &config)
	configInfo = config
}
