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
	"io"
	"os"
	"path/filepath"
	"strings"
)

type Decompressor interface {
	Filename() string
	Decompress(dest string) error
}

type GzipTarDecompressor struct {
	src      string
	filename string
}

func (g *GzipTarDecompressor) Decompress(dest string) error {
	file, err := os.Open(g.src)
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
	first := true
	for {
		header, err := tr.Next()
		switch {
		case err == io.EOF:
			return nil
		case err != nil:
			return err
		case header == nil:
			continue
		}
		target := filepath.Join(dest, header.Name)
		if first && strings.Contains(header.Name, "/") {
			first = false
			g.filename = strings.Split(header.Name, "/")[0]
		}
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
	}
}

func (g *GzipTarDecompressor) Filename() string {
	return g.filename
}

type ZipDecompressor struct {
	src      string
	filename string
}

func (z *ZipDecompressor) Filename() string {
	return z.filename
}

func (z *ZipDecompressor) Decompress(dest string) error {
	r, err := zip.OpenReader(z.src)
	if err != nil {
		return err
	}
	defer r.Close()
	first := true
	for _, f := range r.File {
		err := z.processZipFile(f, dest)
		if err != nil {
			return err
		}
		if first {
			first = false
			z.filename = strings.Split(f.Name, "/")[0]
		}
	}
	return nil
}

func (z *ZipDecompressor) processZipFile(f *zip.File, dest string) error {
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

func NewDecompressor(src string) Decompressor {
	filename := filepath.Base(src)
	if strings.HasSuffix(filename, ".tar.gz") {
		return &GzipTarDecompressor{
			src:      src,
			filename: strings.TrimSuffix(filename, ".tar.gz"),
		}
	}
	if strings.HasSuffix(filename, ".zip") {
		return &ZipDecompressor{
			src:      src,
			filename: strings.TrimSuffix(filename, ".zip"),
		}
	}
	return nil
}
