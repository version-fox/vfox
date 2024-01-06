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
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"github.com/ulikunitz/xz"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type Decompressor interface {
	Decompress(dest string) error
}

type GzipTarDecompressor struct {
	src string
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
		// Split the file name into a slice
		parts := strings.Split(header.Name, "/")
		if len(parts) > 1 {
			// Remove the first element
			parts = parts[1:]
		}
		// Join the remaining elements to get the new file name
		fname := strings.Join(parts, "/")

		target := filepath.Join(dest, fname)
		switch header.Typeflag {
		case tar.TypeDir:
			if _, err := os.Stat(target); err != nil {
				if err := os.MkdirAll(target, 0755); err != nil {
					return err
				}
			}
		case tar.TypeReg:
			_ = os.MkdirAll(filepath.Dir(target), 0755)
			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}

			if _, err := io.Copy(f, tr); err != nil {
				return err
			}

			f.Close()
		case tar.TypeSymlink:
			err := os.Symlink(header.Linkname, target)
			if err != nil {
				return err
			}

		}
	}
}

type XZTarDecompressor struct {
	src string
}

func (g *XZTarDecompressor) Decompress(dest string) error {
	file, err := os.Open(g.src)
	if err != nil {
		return err
	}
	defer file.Close()
	gzr, err := xz.NewReader(file)
	if err != nil {
		return err
	}

	tr := tar.NewReader(gzr)
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
		// Split the file name into a slice
		parts := strings.Split(header.Name, "/")
		if len(parts) > 1 {
			// Remove the first element
			parts = parts[1:]
		}
		// Join the remaining elements to get the new file name
		fname := strings.Join(parts, "/")

		target := filepath.Join(dest, fname)
		switch header.Typeflag {
		case tar.TypeDir:
			if _, err := os.Stat(target); err != nil {
				if err := os.MkdirAll(target, 0755); err != nil {
					return err
				}
			}
		case tar.TypeReg:
			_ = os.MkdirAll(filepath.Dir(target), 0755)
			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}

			if _, err := io.Copy(f, tr); err != nil {
				return err
			}

			f.Close()
		case tar.TypeSymlink:
			err := os.Symlink(header.Linkname, target)
			if err != nil {
				return err
			}

		}
	}
}

type ZipDecompressor struct {
	src string
}

func (z *ZipDecompressor) Decompress(dest string) error {
	r, err := zip.OpenReader(z.src)
	if err != nil {
		return err
	}
	defer r.Close()
	for _, f := range r.File {
		err := z.processZipFile(f, dest)
		if err != nil {
			return err
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

	// Split the file name into a slice
	parts := strings.Split(f.Name, "/")
	if len(parts) > 1 {
		// Remove the first element
		parts = parts[1:]
	}
	// Join the remaining elements to get the new file name
	fname := strings.Join(parts, "/")

	fpath := filepath.Join(dest, fname)
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
	if strings.HasSuffix(filename, ".tar.gz") || strings.HasSuffix(filename, ".tgz") {
		return &GzipTarDecompressor{
			src: src,
		}
	}
	if strings.HasSuffix(filename, ".tar.xz") {
		return &XZTarDecompressor{
			src: src,
		}
	}
	if strings.HasSuffix(filename, ".zip") {
		return &ZipDecompressor{
			src: src,
		}
	}
	return nil
}
