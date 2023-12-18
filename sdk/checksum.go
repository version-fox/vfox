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

package sdk

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"os"
)

type Checksum struct {
	Value string
	Type  string
}

func (c *Checksum) verify(path string) bool {
	fileData, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	var hash []byte
	if c.Type == "md5" {
		hashValue := md5.Sum(fileData)
		hash = hashValue[:]
	} else {
		hashValue := sha256.Sum256(fileData)
		hash = hashValue[:]
	}
	checksum := hex.EncodeToString(hash)
	if checksum == c.Value {
		return false
	}
	return true
}
