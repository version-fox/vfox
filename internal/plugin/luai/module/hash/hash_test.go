/*
 *    Copyright 2026 Han Li and contributors
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

package hash

import (
	"os"
	"path/filepath"
	"testing"

	lua "github.com/yuin/gopher-lua"
)

func TestHashModule(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "sample-file.txt")
	if err := os.WriteFile(testFile, []byte("hello world"), 0644); err != nil {
		t.Fatal(err)
	}

	const script = `
	local hash = require("hash")
	assert(type(hash) == "table")
	assert(type(hash.sha256_file) == "function")
	assert(type(hash.verify_sha256) == "function")
	assert(type(hash.sum_file) == "function")
	assert(type(hash.verify_file) == "function")

	local sha256, err = hash.sha256_file(testFile)
	assert(err == nil, err)
	assert(sha256 == "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9")

	local genericSha256, err = hash.sum_file(testFile, "sha-256")
	assert(err == nil, err)
	assert(genericSha256 == sha256)

	local ok, err = hash.verify_sha256(testFile, sha256)
	assert(err == nil, err)
	assert(ok == true)

	local ok, err = hash.verify_file(testFile, string.upper(sha256), "sha256")
	assert(err == nil, err)
	assert(ok == true)

	local ok, err = hash.verify_sha256(testFile, "invalid")
	assert(err == nil, err)
	assert(ok == false)
	`

	s := lua.NewState()
	defer s.Close()
	s.SetGlobal("testFile", lua.LString(testFile))
	Preload(s)
	if err := s.DoString(script); err != nil {
		t.Error(err)
	}
}

func TestHashModuleErrors(t *testing.T) {
	const script = `
	local hash = require("hash")

	local sum, err = hash.sum_file("missing-file", "sha256")
	assert(sum == nil)
	assert(type(err) == "string")

	local sum, err = hash.sum_file(testFile, "unknown")
	assert(sum == nil)
	assert(string.find(err, "unsupported hash algorithm"))

	local ok, err = hash.verify_file("missing-file", "expected", "sha256")
	assert(ok == false)
	assert(type(err) == "string")
	`

	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "sample-file.txt")
	if err := os.WriteFile(testFile, []byte("hello world"), 0644); err != nil {
		t.Fatal(err)
	}

	s := lua.NewState()
	defer s.Close()
	s.SetGlobal("testFile", lua.LString(testFile))
	Preload(s)
	if err := s.DoString(script); err != nil {
		t.Error(err)
	}
}
