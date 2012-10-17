%{
package fubsy

import (
	"fmt"
)

// Global variables appear to be the only way to get information out
// of the parser.
var _lasttok *toktext
var _ast *RootNode
var _syntaxerror *SyntaxError

const BADTOKEN = -1
%}

%union {
	qstring string
}

%type <node> script

%token <qstring> QSTRING

%%

script:
	'[' QSTRING ']'
	{
		//fmt.Printf("parser: got qstring >%s< in brackets\n", $2)
		values := []string {$2}
		root := RootNode{}
		root.elements = []ASTNode {ListNode{values: values}}
		_ast = &root
	}

%%

// a token together with its location, text, etc.
type toktext struct {
	filename string
	lineno int
	token int
	text string
}

type DummyLexer struct {
	tokens []toktext
	next int
}

func (self *DummyLexer) Lex(lval *fuSymType) int {
	if self.next >= len(self.tokens) {
		return 0				// eof
	}
	toktext := self.tokens[self.next]
	self.next += 1
	if toktext.token == QSTRING {
		lval.qstring = toktext.text[1:len(toktext.text)-1]
	}
	_lasttok = &toktext
	return toktext.token
}

func (self *DummyLexer) Error(e string) {
	_syntaxerror = &SyntaxError{
		line: _lasttok.lineno,
		badtoken: _lasttok.text,
		message: e}
}
