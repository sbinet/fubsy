// Copyright © 2012, Greg Ward. All rights reserved.
// Use of this source code is governed by a BSD-style license that can
// be found in the LICENSE.txt file.

package dsl

import (
	"fmt"
	"strconv"
	"strings"
)

type fileinfo struct {
	filename string

	// file offset of the first char of each line, plus one more
	// pointing just past EOF -- e.g. for input "foo\n\nbar",
	// lineoffsets = {0, 4, 5, 8}
	lineoffsets []int
}

// physical location of a token or AST node
type FileLocation struct {
	fileinfo *fileinfo
	// [start:end] is a slice into the file contents, i.e.
	// start is the offset of the first byte of this location,
	// and end is *one past* the last byte
	start int
	end   int
}

func newFileLocation(fileinfo *fileinfo) FileLocation {
	return FileLocation{fileinfo, -1, -1}
}

// so FileLocation implements Location
func (self FileLocation) Location() Location {
	return self
}

func (self FileLocation) String() string {
	if self.fileinfo == nil {
		return "(uninitialized)"
	}
	filename := self.fileinfo.filename
	if filename == "" {
		filename = "(unknown)"
	}
	return fmt.Sprintf("%s[%d:%d]", filename, self.start, self.end)
}

func (self FileLocation) ErrorPrefix() string {
	if self.fileinfo == nil {
		// don't panic on uninitialized FileLocation object
		return ""
	}
	var chunks []string
	fn := self.fileinfo.filename
	if fn == "" {
		fn = "(unknown)"
	}
	chunks = append(chunks, fn)
	sline, eline := self.linerange()
	if sline > 0 {
		var lines string
		if sline == eline {
			lines = strconv.Itoa(sline)
		} else {
			lines = fmt.Sprintf("%d-%d", sline, eline)
		}
		chunks = append(chunks, lines)
	}
	return strings.Join(chunks, ":") + ": "
}

// Return a new FileLocation that spans self and other.
func (self FileLocation) merge(other_ Location) Location {
	other := other_.(FileLocation)
	if self.fileinfo == nil {
		return other
	} else if other.fileinfo == nil {
		return self
	}

	if self.fileinfo != other.fileinfo {
		panic(fmt.Sprintf(
			"cannot merge FileLocations from different files"+
				" (self.fileinfo = %#v, other.fileinfo = %#v)",
			self.fileinfo, other.fileinfo))
	}
	result := newFileLocation(self.fileinfo)
	if self.start <= other.end {
		result.start = self.start
		result.end = other.end
	} else {
		result.start = other.start
		result.end = other.end
	}
	return result
}

func (self FileLocation) span() (int, int) {
	return self.start, self.end
}

func (self FileLocation) linerange() (startline int, endline int) {
	// don't try to call this with uninitialized lineoffsets!
	offsets := self.fileinfo.lineoffsets
	if len(offsets) < 2 {
		panic(fmt.Sprintf(
			"invalid lineoffsets array %v: must have at least 2 elements",
			offsets))
	}

	startline = -1
	endline = -1
	if self.start == -1 || self.end == -1 {
		return
	}

	i := 0
	for ; i < len(offsets)-1; i++ {
		if offsets[i] <= self.start && offsets[i+1]-1 >= self.start {
			startline = i + 1
			break
		}
	}
	// special case for empty tokens...
	if self.end == self.start {
		// ...which really only occur as synthetic EOL or EOF tokens,
		// just past the last byte of the file
		if startline == -1 {
			startline = len(offsets) - 1
		}
		endline = startline
	}

	if startline == -1 {
		panic(fmt.Sprintf(
			"unable to determine start line for offset %d (line offsets: %v)",
			self.start, offsets))
	}

	for ; endline == -1 && i < len(offsets)-1; i++ {
		if offsets[i] <= self.end-1 && offsets[i+1]-1 >= self.end-1 {
			endline = i + 1
		}
	}
	if endline == -1 {
		panic(fmt.Sprintf(
			"unable to determine end line for offset %d (line offsets: %v)",
			self.end, offsets))
	}
	return
}
