package testutils

import (
	"testing"
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
