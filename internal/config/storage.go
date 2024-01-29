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

package config

import (
	"fmt"
	"os"
	"path/filepath"
)

type Storage struct {
	SdkPath string `yaml:"sdkPath"`
}

var EmptyStorage = &Storage{
	SdkPath: "",
}

func (s *Storage) Validate() error {
	if s.SdkPath == "" {
		return nil
	}
	stat, err := os.Stat(s.SdkPath)
	if err != nil {
		return err
	}
	if !stat.IsDir() {
		return fmt.Errorf("%s is not a directory", s.SdkPath)
	}
	tmpfn := filepath.Join(s.SdkPath, ".tmpfile")
	f, err := os.OpenFile(tmpfn, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0666)
	if err != nil {
		return err
	}
	defer os.Remove(tmpfn)
	defer f.Close()
	return nil
}
