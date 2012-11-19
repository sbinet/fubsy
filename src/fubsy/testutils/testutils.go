package testutils

import (
	"testing"
	"os"
	"path"
	"io/ioutil"
)

func AssertNoError(t *testing.T, actual error) {
	if actual != nil {
		t.Fatal("unexpected error:", actual)
	}
}

func AssertNoErrors(t *testing.T, actual []error) {
	if len(actual) != 0 {
		t.Fatalf("expected empty list of errors, but got %v", actual)
	}
}

func AssertError(t *testing.T, expect string, actual error) {
	if actual == nil {
		t.Fatal("expected error, but got nil")
	}
	if actual.Error() != expect {
		t.Errorf("expected error message\n%s\nbut got\n%s",
			expect, actual.Error())
	}
}

func AssertStrings(t *testing.T, expect string, actual string) {
	if expect != actual {
		t.Errorf("expected %#v, but got %#v", expect, actual)
	}
}

// Create a temporary directory. Return the name of the directory and
// a function to clean it up when you're done with it. Panics on any
// error (as does the cleanup function).
func Mktemp() (tmpdir string, cleanup func()) {
	tmpdir, err := ioutil.TempDir("", "fubsytest.")
	if err != nil {
		panic(err)
	}
	cleanup = func() {
		err := os.RemoveAll(tmpdir)
		if err != nil {
			panic(err)
		}
	}
	return
}

// Create a temporary directory and chdir to it. Returns a function
// that chdirs back to your original location and removes the temp
// directory. Panics on any error (as does the goback function).
func Chtemp() (goback func()) {
	tmpdir, cleanup := Mktemp()
	orig, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	err = os.Chdir(tmpdir)
	if err != nil {
		panic(err)
	}

	goback = func() {
		err := os.Chdir(orig)
		if err != nil {
			panic(err)
		}
		cleanup()
	}
	return
}

// Create a file in tmpdir and write data to it. Panics on any error.
func Mkfile(tmpdir string, basename string, data string) string {
	fn := path.Join(tmpdir, basename)
	err := ioutil.WriteFile(fn, []byte(data), 0644)
	if err != nil {
		panic(err)
	}
	return fn
}
