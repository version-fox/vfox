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
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	gohash "hash"
	"io"
	"os"
	"strings"

	lua "github.com/yuin/gopher-lua"
)

func Preload(L *lua.LState) {
	L.PreloadModule("hash", loader)
}

func loader(L *lua.LState) int {
	t := L.NewTable()
	L.SetFuncs(t, api)
	L.Push(t)
	return 1
}

var api = map[string]lua.LGFunction{
	"sum_file":      SumFile,
	"verify_file":   VerifyFile,
	"md5_file":      Md5File,
	"sha1_file":     Sha1File,
	"sha256_file":   Sha256File,
	"sha512_file":   Sha512File,
	"verify_md5":    VerifyMd5,
	"verify_sha1":   VerifySha1,
	"verify_sha256": VerifySha256,
	"verify_sha512": VerifySha512,
}

func SumFile(L *lua.LState) int {
	filepath := L.CheckString(1)
	algorithm := L.CheckString(2)
	return pushFileDigest(L, filepath, algorithm)
}

func VerifyFile(L *lua.LState) int {
	filepath := L.CheckString(1)
	expected := L.CheckString(2)
	algorithm := L.CheckString(3)
	return pushFileVerify(L, filepath, expected, algorithm)
}

func Md5File(L *lua.LState) int {
	return pushFileDigest(L, L.CheckString(1), "md5")
}

func Sha1File(L *lua.LState) int {
	return pushFileDigest(L, L.CheckString(1), "sha1")
}

func Sha256File(L *lua.LState) int {
	return pushFileDigest(L, L.CheckString(1), "sha256")
}

func Sha512File(L *lua.LState) int {
	return pushFileDigest(L, L.CheckString(1), "sha512")
}

func VerifyMd5(L *lua.LState) int {
	return pushFileVerify(L, L.CheckString(1), L.CheckString(2), "md5")
}

func VerifySha1(L *lua.LState) int {
	return pushFileVerify(L, L.CheckString(1), L.CheckString(2), "sha1")
}

func VerifySha256(L *lua.LState) int {
	return pushFileVerify(L, L.CheckString(1), L.CheckString(2), "sha256")
}

func VerifySha512(L *lua.LState) int {
	return pushFileVerify(L, L.CheckString(1), L.CheckString(2), "sha512")
}

func pushFileDigest(L *lua.LState, filepath, algorithm string) int {
	sum, err := DigestFile(filepath, algorithm)
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(err.Error()))
		return 2
	}
	L.Push(lua.LString(sum))
	return 1
}

func pushFileVerify(L *lua.LState, filepath, expected, algorithm string) int {
	sum, err := DigestFile(filepath, algorithm)
	if err != nil {
		L.Push(lua.LFalse)
		L.Push(lua.LString(err.Error()))
		return 2
	}
	L.Push(lua.LBool(strings.EqualFold(sum, strings.TrimSpace(expected))))
	return 1
}

func DigestFile(filepath, algorithm string) (string, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hasher, err := newHash(algorithm)
	if err != nil {
		return "", err
	}
	if _, err = io.Copy(hasher, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func newHash(algorithm string) (gohash.Hash, error) {
	switch normalizeAlgorithm(algorithm) {
	case "md5":
		return md5.New(), nil
	case "sha1":
		return sha1.New(), nil
	case "sha256":
		return sha256.New(), nil
	case "sha512":
		return sha512.New(), nil
	default:
		return nil, fmt.Errorf("unsupported hash algorithm: %s", algorithm)
	}
}

func normalizeAlgorithm(algorithm string) string {
	algorithm = strings.ToLower(strings.TrimSpace(algorithm))
	return strings.ReplaceAll(algorithm, "-", "")
}
