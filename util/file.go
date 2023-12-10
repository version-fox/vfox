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

package util

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"github.com/schollz/progressbar/v3"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func DecompressGzipTar(src, dest string) error {
	file, err := os.Open(src)
	if err != nil {
		return err
	}
	defer file.Close()
	gzr, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	bar := progressbar.NewOptions(-1,
		progressbar.OptionSetRenderBlankState(true),
		progressbar.OptionSetDescription("Decompressing..."),
		progressbar.OptionFullWidth(),
	)

	for {
		header, err := tr.Next()
		switch {
		case err == io.EOF:
			fmt.Println()
			return nil
		case err != nil:
			return err
		case header == nil:
			continue
		}
		target := filepath.Join(dest, header.Name)
		switch header.Typeflag {
		case tar.TypeDir:
			if _, err := os.Stat(target); err != nil {
				if err := os.MkdirAll(target, 0755); err != nil {
					return err
				}
			}
		case tar.TypeReg:
			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}

			if _, err := io.Copy(f, tr); err != nil {
				return err
			}

			f.Close()

		}
		_ = bar.Add(1)
	}
}

func DecompressZip(src string, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()
	bar := progressbar.Default(int64(len(r.File)), "Decompressing...")
	for _, f := range r.File {
		err := processZipFile(f, dest)
		if err != nil {
			return err
		}
		_ = bar.Add(1)
	}
	return nil
}

func processZipFile(f *zip.File, dest string) error {
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	fpath := filepath.Join(dest, f.Name)
	if f.FileInfo().IsDir() {
		err := os.MkdirAll(fpath, os.ModePerm)
		if err != nil {
			return err
		}
	} else {
		var fdir string
		if lastIndex := strings.LastIndex(fpath, string(os.PathSeparator)); lastIndex > -1 {
			fdir = fpath[:lastIndex]
		}

		err = os.MkdirAll(fdir, os.ModePerm)
		if err != nil {
			return err
		}
		f, err := os.OpenFile(
			fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}
		defer f.Close()

		_, err = io.Copy(f, rc)
		if err != nil {
			return err
		}
	}
	return nil
}

func FileExists(filename string) bool {
	_, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return err == nil
}
