package dsl

import (
	"testing"
	"fubsy/testutils"
)

func Test_linerange_basic(t *testing.T) {
	// Sample input:
	//   "foo\n\n0123456789a\nyoyoyoyo\n\n"
	// (This file has 5 newlines, therefore it is considered to have 5
	// lines: each newline counts as part of the line it terminates.
	// lineoffsets for this file has 6 elements, the last one pointing
	// just past EOF, as a convenience.)
	fi := &fileinfo{lineoffsets: []int {0, 4, 5, 17, 26, 27}}
	loc := location{fi, 0, 0}	// empty token at start of line 1
	assertLines(t, 1, 1, loc)

	loc.end = 3				// still entirely in line 1
	assertLines(t, 1, 1, loc)

	loc.end = 4				// include newline in the token
	assertLines(t, 1, 1, loc)

	loc.start = 3			// newline *is* the token
	assertLines(t, 1, 1, loc)

	loc.start = 4			// line 2 is just a newline
	loc.end = 5
	assertLines(t, 2, 2, loc)

	loc.start = 5			// start of line 3 ("0123456789a")
	loc.end = 15				// still in line 3
	assertLines(t, 3, 3, loc)

	loc.start = 8			// not at start, but still line 3
	assertLines(t, 3, 3, loc)

	loc.end = 18				// include first char of line 4
	assertLines(t, 3, 4, loc)

	loc.end = 21				// middle of line 4
	assertLines(t, 3, 4, loc)

	loc.end = 24				// include last non-newline char of line 4
	assertLines(t, 3, 4, loc)

	loc.end = 25				// newline at end of line 4
	assertLines(t, 3, 4, loc)

	loc.end = 27				// newline at end of line 5 (blank line)
	assertLines(t, 3, 5, loc)
}

func Test_linerange_oneline(t *testing.T) {
	// sample input:
	//   "foobar"
	// (1 line, no newlines)
	fi := &fileinfo{lineoffsets: []int {0, 6}}
	loc := location{fi, 0, 0}	// empty token at start of line 1
	assertLines(t, 1, 1, loc)

	loc.end = 6				// span all of line 1
	assertLines(t, 1, 1, loc)

	loc.start = 5			// last char of line 1 (and of the file)
	assertLines(t, 1, 1, loc)

	// this is how we represent empty tokens, like the synthetic EOL
	// added when there is no \n at EOF
	loc.start = 6
	loc.end = 6
	assertLines(t, 1, 1, loc)
}

func Test_linerange_panic_lineoffsets(t *testing.T) {
	fi := &fileinfo{lineoffsets: []int {}}
	location := newlocation(fi)
	defer wantpanic(t)
	location.linerange()
}

func Test_linerange_panic_aftereof_1(t *testing.T) {
	fi := &fileinfo{lineoffsets: []int {0, 10}}
	location := newlocation(fi)
	location.start = 10
	location.end = 15
	defer wantpanic(t)
	location.linerange()
}

func Test_linerange_panic_aftereof_2(t *testing.T) {
	fi := &fileinfo{lineoffsets: []int {0, 10}}
	location := newlocation(fi)
	location.start = 5
	location.end = 11
	defer wantpanic(t)
	location.linerange()
}

func Test_location_String(t *testing.T) {
	fi := &fileinfo{lineoffsets: []int {0, 4, 5, 17, 26, 27}}
	loc := newlocation(fi)
	testutils.AssertStrings(t, "(unknown): ", loc.String())

	fi.filename = "foo.txt"
	testutils.AssertStrings(t, "foo.txt: ", loc.String())

	loc.start = 2
	loc.end = 3
	testutils.AssertStrings(t, "foo.txt:1: ", loc.String())

	loc.end = 6
	testutils.AssertStrings(t, "foo.txt:1-3: ", loc.String())

	fi.filename = ""
	testutils.AssertStrings(t, "(unknown):1-3: ", loc.String())
}

func assertLines(t *testing.T, start int, end int, location location) {
	actualstart, actualend := location.linerange()
	if !(start == actualstart && end == actualend) {
		t.Errorf(
			"bad location.linerange(): " +
			"expected (%d, %d) but got (%d, %d)",
			start, end, actualstart, actualend)
	}
}

func wantpanic(t *testing.T) {
	if err := recover(); err == nil {
		t.Error("expected panic")
	} else {
		//t.Log("got expected panic:", err)
	}
}
