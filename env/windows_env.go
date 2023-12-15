//go:build windows

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

package env

type windowsEnvManager struct {
}

func (w *windowsEnvManager) Flush() {
	//TODO implement me

	panic("implement me")
}

func (w *windowsEnvManager) Load(kvs []*KV) error {
	//TODO implement me
	panic("implement me")
}

func (w *windowsEnvManager) Get(key string) (string, error) {
	//TODO implement me
	panic("implement me")
}

func (w *windowsEnvManager) Remove(key string) error {
	//TODO implement me
	panic("implement me")
}

func (w *windowsEnvManager) ReShell() error {
	//TODO implement me
	panic("implement me")
}

func NewEnvManager(vfConfigPath string) (Manager, error) {
	manager := &windowsEnvManager{}
	return manager, nil
}
