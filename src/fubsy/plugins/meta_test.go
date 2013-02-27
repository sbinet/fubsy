package plugins

import (
	"testing"
)

func Test_init(t *testing.T) {
	// test that metaFactory is correctly initialized
	plugin, err := metaFactory["python2"]()
	if err != nil {
		if _, ok := err.(NotAvailableError); ok {
			return
		}
		t.Fatalf("expected either nil error or NotAvailableError; "+
			"got %T: %s",
			err, err.Error())
	}
	_ = plugin.(PythonPlugin)
}
