package dsl

import (
	"strconv"
	"strings"
	"fmt"
)

type fileinfo struct {
	filename string

	// file offset of the first char of each line, plus one more
	// pointing just past EOF -- e.g. for input "foo\n\nbar",
	// lineoffsets = {0, 4, 5, 8}
	lineoffsets []int
}

// physical location of a token or AST node
type location struct {
	fileinfo *fileinfo
	// [start:end] is a slice into the file contents, i.e.
	// start is the offset of the first byte of this location,
	// and end is *one past* the last byte
	start int
	end int
}

func newlocation(fileinfo *fileinfo) location {
	return location{fileinfo, -1, -1}
}

func (self location) String() string {
	fn := self.fileinfo.filename
	if fn == "" {
		fn = "(unknown)"
	}
	chunks := []string {fn}
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

func (self location) linerange() (startline int, endline int) {
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
	for ; i < len(offsets) - 1; i++ {
		if offsets[i] <= self.start && offsets[i+1]-1 >=  self.start  {
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

	for ; endline == -1 && i < len(offsets) - 1; i++ {
		if offsets[i] <= self.end - 1 && offsets[i+1]-1 >= self.end - 1 {
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
