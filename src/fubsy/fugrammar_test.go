package fubsy

import (
	"testing"
	"os"
	"fmt"
	"bytes"
)

func TestParse_valid(t *testing.T) {
	parser := NewParser()
	defer parser.Dispose()

	expect := RootNode{elements: []ASTNode {ListNode{values: []string {"foo"}}}}
	parser.Parse(LBRACKET, ptok("lbracket", "["))
	parser.Parse(QSTRING,  ptok("qstring",  "\"foo\""))
	parser.Parse(RBRACKET, ptok("rbracket", "]"))
	parser.Parse(0, nil)
	if false { _ast.Dump(os.Stdout, "") }
	checkASTEquals(t, &expect, _ast)
}

func TestParse_invalid(t *testing.T) {
	parser := NewParser()
	defer parser.Dispose()

	// hmmm: we just print syntax errors to stderr -- can't test that!
	parser.Parse(QSTRING, ptok("qstring", "\"boo\""))
	parser.Parse(LBRACKET, ptok("lbracket", "["))
	parser.Parse(0, nil)

	assertNotNil(t, "_syntaxerror", _syntaxerror)
	if _syntaxerror.badtoken != "\"boo\"" {
		t.Errorf("_syntaxerror.badtoken = %v (expected \"boo\")",
			_syntaxerror.badtoken)
	}
	expect := ":0: syntax error near \"boo\""
	actual := _syntaxerror.Error()
	if actual != expect {
		t.Errorf("bad syntax error message: %s\n(expected: %s)",
			actual, expect)
	}
}

func checkASTEquals(t *testing.T, expect *RootNode, actual *RootNode) {
	if ! expect.Equal(*actual) {
		expectbuf := new(bytes.Buffer)
		actualbuf := new(bytes.Buffer)
		expect.Dump(expectbuf, "")
		actual.Dump(actualbuf, "")
		fmt.Printf("expect.elements[0] = %#v\n", expect.elements[0])
		fmt.Printf("actual.elements[0] = %#v\n", actual.elements[0])
		t.Errorf("expected AST node:\n%sbut got:\n%s", expectbuf, actualbuf)
	}
}

func assertNotNil(t *testing.T, name string, p *SyntaxError) {
	if p == nil {
		t.Fatal(fmt.Sprintf("%s == nil (expected non-nil)", name))
	}
}

// return a token in the form expected by parser.Parse()
func ptok(name string, value string) ParseTOKENTYPE {
	tok := ASTNode(ttok(name, value)) // ttok() is in fulex_test.go
	return ParseTOKENTYPE(&tok)
}
