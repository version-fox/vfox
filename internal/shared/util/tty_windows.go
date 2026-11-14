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

package util

import (
	"encoding/binary"
	"strings"
	"unicode/utf16"

	"golang.org/x/sys/windows"
)

// maxPathUnits is the length of a file name buffer in UTF-16 units, MAX_PATH.
const maxPathUnits = 260

// canReadKeysFrom reports whether the key reader can read fd, which needs a console.
func canReadKeysFrom(fd uintptr) bool {
	var mode uint32
	return windows.GetConsoleMode(windows.Handle(fd), &mode) == nil
}

// isCygwinTerminal reports whether fd is a pty of Cygwin or MSYS2, such as the
// one Git Bash in mintty runs in. Native programs get that pty as a named pipe
// called \msys-<id>-pty<N>-to-master (or -from-master), not as a console.
func isCygwinTerminal(fd uintptr) bool {
	handle := windows.Handle(fd)
	if fileType, err := windows.GetFileType(handle); err != nil || fileType != windows.FILE_TYPE_PIPE {
		return false
	}
	name, err := pipeName(handle)
	return err == nil && isCygwinPipeName(name)
}

// pipeName returns the name of a pipe handle as FileNameInfo reports it: a byte
// count, then the name in UTF-16.
func pipeName(handle windows.Handle) (string, error) {
	buf := make([]byte, 4+2*maxPathUnits)
	if err := windows.GetFileInformationByHandleEx(handle, windows.FileNameInfo, &buf[0], uint32(len(buf))); err != nil {
		return "", err
	}
	size := int(binary.LittleEndian.Uint32(buf[:4]))
	if size > len(buf)-4 {
		size = len(buf) - 4
	}
	units := make([]uint16, size/2)
	for i := range units {
		units[i] = binary.LittleEndian.Uint16(buf[4+2*i:])
	}
	return string(utf16.Decode(units)), nil
}

// isCygwinPipeName matches the names of Cygwin and MSYS2 ptys, the way go-isatty
// does: \msys-<id>-pty<N>-(from|to)-master, or the same with cygwin for msys.
func isCygwinPipeName(name string) bool {
	token := strings.Split(name, "-")
	if len(token) < 5 {
		return false
	}
	switch token[0] {
	case `\msys`, `\cygwin`, `\Device\NamedPipe\msys`, `\Device\NamedPipe\cygwin`:
	default:
		return false
	}
	return token[1] != "" &&
		strings.HasPrefix(token[2], "pty") &&
		(token[3] == "from" || token[3] == "to") &&
		token[4] == "master"
}
