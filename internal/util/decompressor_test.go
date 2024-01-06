/*
 *    Copyright 2024 [lihan aooohan@gmail.com]
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
	"testing"
)

func TestNewDecompressor(t *testing.T) {
	gzipTarDecompressor := NewDecompressor("test.tar.gz")
	if _, ok := gzipTarDecompressor.(*GzipTarDecompressor); !ok {
		t.Errorf("Expected GzipTarDecompressor, got %T", gzipTarDecompressor)
	}

	zipDecompressor := NewDecompressor("test.zip")
	if _, ok := zipDecompressor.(*ZipDecompressor); !ok {
		t.Errorf("Expected ZipDecompressor, got %T", zipDecompressor)
	}

	unknownDecompressor := NewDecompressor("test.unknown")
	if unknownDecompressor != nil {
		t.Errorf("Expected nil, got %T", unknownDecompressor)
	}
}

//func TestDecompress(t *testing.T) {
//	// Create a temporary directory for testing
//	tempDir, err := os.MkdirTemp("", "decompress_test")
//	if err != nil {
//		t.Fatal(err)
//	}
//	defer os.RemoveAll(tempDir)
//
//	// Create a temporary .tar.gz file for testing
//	tempFile, err := os.CreateTemp("", "test.*.tar.gz")
//	if err != nil {
//		t.Fatal(err)
//	}
//	defer os.Remove(tempFile.Name())
//
//	// 创建一个新的gzip.Writer
//	gw := gzip.NewWriter(tempFile)
//	defer gw.Close()
//
//	// 创建一个新的tar.Writer
//	tw := tar.NewWriter(gw)
//	defer tw.Close()
//
//	var files = []struct {
//		Name, Body string
//	}{
//		{"test.txt", "Hello, World!"},
//		{"test2.txt", "This is a test."},
//	}
//
//	// 将内容写入.tar.gz文件
//	for _, file := range files {
//		hdr := &tar.Header{
//			Name: file.Name,
//			Mode: 0600,
//			Size: int64(len(file.Body)),
//		}
//		if err := tw.WriteHeader(hdr); err != nil {
//			t.Fatal(err)
//		}
//		if _, err := io.Copy(tw, strings.NewReader(file.Body)); err != nil {
//			t.Fatal(err)
//		}
//	}
//
//	decompressor := NewDecompressor(tempFile.Name())
//	err = decompressor.Decompress(tempDir)
//	if err != nil {
//		t.Errorf("Failed to decompress: %v", err)
//	}
//	// 比较解压后的文件和原始文件
//	for _, file := range files {
//		decompressedFile, err := os.ReadFile(filepath.Join(tempDir, file.Name))
//		if err != nil {
//			t.Fatal(err)
//		}
//
//		if string(decompressedFile) != file.Body {
//			t.Errorf("Decompressed file content does not match original content")
//		}
//	}
//}
//
//func TestZipDecompressor(t *testing.T) {
//	// Create a temporary directory for testing
//	tempDir, err := os.MkdirTemp("", "decompress_test")
//	if err != nil {
//		t.Fatal(err)
//	}
//	defer os.RemoveAll(tempDir)
//
//	// Create a temporary .zip file for testing
//	tempFile, err := os.CreateTemp("", "test.*.zip")
//	if err != nil {
//		t.Fatal(err)
//	}
//	defer os.Remove(tempFile.Name())
//
//	// Create a new zip.Writer
//	zw := zip.NewWriter(tempFile)
//	defer zw.Close()
//
//	var files = []struct {
//		Name, Body string
//	}{
//		{"a.txt", "aaaa"},
//		{"b.txt", "bbbb"},
//	}
//
//	// Write content to .zip file
//	for _, file := range files {
//		fw, err := zw.Create(file.Name)
//		if err != nil {
//			t.Fatal(err)
//		}
//		_, err = fw.Write([]byte(file.Body))
//		if err != nil {
//			t.Fatal(err)
//		}
//	}
//
//	decompressor := NewDecompressor(tempFile.Name())
//	err = decompressor.Decompress(tempDir)
//	if err != nil {
//		t.Errorf("Failed to decompress: %v", err)
//	}
//
//	// Compare decompressed files with original files
//	for _, file := range files {
//		decompressedFile, err := os.ReadFile(filepath.Join(tempDir, file.Name))
//		if err != nil {
//			t.Fatal(err)
//		}
//
//		if string(decompressedFile) != file.Body {
//			t.Errorf("Decompressed file content does not match original content")
//		}
//	}
//}
