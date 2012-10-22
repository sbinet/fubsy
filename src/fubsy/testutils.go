package fubsy

import (
	"testing"
)

func assertNoError(t *testing.T, actual error) {
	if actual != nil {
		t.Fatal("unexpected error:", actual)
	}
}

func assertError(t *testing.T, expect string, actual error) {
	if actual == nil {
		t.Fatal("expected error, but got nil")
	}
	if actual.Error() != expect {
		t.Errorf("expected error message\n%s\nbut got\n%s",
			expect, actual.Error())
	}
}
