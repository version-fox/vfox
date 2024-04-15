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

package util

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"fmt"
	"github.com/bodgit/sevenzip"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/ulikunitz/xz"
)

type Decompressor interface {
	Decompress(dest string) error
}

type symlink struct {
	oldname, newname string
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
	var symlinks []symlink
loop:
	for {
		header, err := tr.Next()
		switch {
		case err == io.EOF:
			break loop
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
			symlinks = append(symlinks, symlink{header.Linkname, target})
		}
	}
	for _, s := range symlinks {
		dir := filepath.Dir(s.newname)
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return err
			}
		}
		if err = os.Symlink(s.oldname, s.newname); err != nil {
			return err
		}
	}
	return nil
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
	var symlinks []symlink
loop:
	for {
		header, err := tr.Next()
		switch {
		case err == io.EOF:
			break loop
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
			symlinks = append(symlinks, symlink{header.Linkname, target})
		}
	}
	for _, s := range symlinks {
		dir := filepath.Dir(s.newname)
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return err
			}
		}
		if err = os.Symlink(s.oldname, s.newname); err != nil {
			return err
		}
	}
	return nil
}

type ZipDecompressor struct {
	src string
}

func (z *ZipDecompressor) Decompress(dest string) error {
	rootFolderInZip := findRootFolderInZip(z.src)
	r, err := zip.OpenReader(z.src)
	if err != nil {
		return err
	}
	defer r.Close()
	for _, f := range r.File {
		err := z.processZipFile(f, dest, rootFolderInZip)
		if err != nil {
			return err
		}
	}
	return nil
}

func findRootFolderInZip(zipFilePath string) string {
	r, err := zip.OpenReader(zipFilePath)
	if err != nil {
		return ""
	}
	defer r.Close()

	var firstElement string

	for _, f := range r.File {
		normalizedPath := strings.ReplaceAll(f.Name, "\\", "/")

		currentFirstElement := strings.Split(normalizedPath, "/")[0]

		if firstElement != "" && firstElement != currentFirstElement {
			return ""
		}

		if firstElement == "" {
			firstElement = currentFirstElement
		}
	}
	return firstElement
}

func isDir(path string) bool {
	return strings.HasSuffix(path, "/")
}

func (z *ZipDecompressor) processZipFile(f *zip.File, dest string, rootFolderInZip string) error {
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	normalizedPath := strings.ReplaceAll(f.Name, "\\", "/")
	// Split the file name into a slice
	parts := strings.Split(normalizedPath, "/")
	if len(parts) > 1 && rootFolderInZip != "" {
		// Remove the first element
		parts = parts[1:]
	}
	// Join the remaining elements to get the new file name
	fname := strings.Join(parts, "/")

	fpath := filepath.Join(dest, fname)
	if f.FileInfo().IsDir() || isDir(fname) {
		err := os.MkdirAll(fpath, os.ModePerm)
		if err != nil {
			return err
		}
	} else if isSymlink(f.FileInfo()) {
		// symlink target is the contents of the file
		buf := new(bytes.Buffer)
		_, err := io.Copy(buf, rc)
		if err != nil {
			return fmt.Errorf("%s: reading symlink target: %v", f.FileHeader.Name, err)
		}
		return writeNewSymbolicLink(fpath, strings.TrimSpace(buf.String()))
	} else {
		err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm)
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

func isSymlink(fi os.FileInfo) bool {
	return fi.Mode()&os.ModeSymlink != 0
}

func writeNewSymbolicLink(fpath string, target string) error {
	err := os.MkdirAll(filepath.Dir(fpath), 0755)
	if err != nil {
		return fmt.Errorf("%s: making directory for file: %v", fpath, err)
	}

	_, err = os.Lstat(fpath)
	if err == nil {
		err = os.Remove(fpath)
		if err != nil {
			return fmt.Errorf("%s: failed to unlink: %+v", fpath, err)
		}
	}

	err = os.Symlink(target, fpath)
	if err != nil {
		return fmt.Errorf("%s: making symbolic link for: %v", fpath, err)
	}
	return nil
}

type SevenZipDecompressor struct {
	src string
}

func findRootFolderIn7Zip(zipFilePath string) string {
	r, err := sevenzip.OpenReader(zipFilePath)
	if err != nil {
		return ""
	}
	defer r.Close()

	var firstElement string

	for _, f := range r.File {

		normalizedPath := strings.ReplaceAll(f.Name, "\\", "/")

		currentFirstElement := strings.Split(normalizedPath, "/")[0]

		if firstElement != "" && firstElement != currentFirstElement {
			return ""
		}

		if firstElement == "" {
			firstElement = currentFirstElement
		}
	}
	return firstElement
}

func (s *SevenZipDecompressor) Decompress(dest string) error {
	rootFolderInZip := findRootFolderIn7Zip(s.src)
	r, err := sevenzip.OpenReader(s.src)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		if err = s.extractFile(f, dest, rootFolderInZip); err != nil {
			return err
		}
	}

	return nil
}

func (s *SevenZipDecompressor) extractFile(f *sevenzip.File, dest string, rootFolderInZip string) error {
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	normalizedPath := strings.ReplaceAll(f.Name, "\\", "/")
	// Split the file name into a slice
	parts := strings.Split(normalizedPath, "/")
	if len(parts) > 1 && rootFolderInZip != "" {
		// Remove the first element
		parts = parts[1:]
	}
	// Join the remaining elements to get the new file name
	fname := strings.Join(parts, "/")

	fpath := filepath.Join(dest, fname)
	if f.FileInfo().IsDir() || isDir(fname) {
		err := os.MkdirAll(fpath, os.ModePerm)
		if err != nil {
			return err
		}
	} else if isSymlink(f.FileInfo()) {
		// symlink target is the contents of the file
		buf := new(bytes.Buffer)
		_, err := io.Copy(buf, rc)
		if err != nil {
			return fmt.Errorf("%s: reading symlink target: %v", f.FileHeader.Name, err)
		}
		return writeNewSymbolicLink(fpath, strings.TrimSpace(buf.String()))
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
	if strings.HasSuffix(filename, ".7z") {
		return &SevenZipDecompressor{
			src: src,
		}
	}
	return nil
}
