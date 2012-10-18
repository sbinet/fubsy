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
%token L3BRACE INLINE R3BRACE

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
	|
        L3BRACE INLINE R3BRACE
        {
		_ast = &RootNode{}
	}

%%

// a token together with its location, text, etc.
type toktext struct {
	filename string
	lineno int
	token int
	text string
}

// This isn't really the lexer; it's just a list of tokens provided by
// the real lexer (a Scanner object from fulex.l). But it implements
// the fuLexer inteface specified by the generated parser, so stick
// with that terminology.
type Lexer struct {
	tokens []toktext
	next int
}

func NewLexer(tokens []toktext) *Lexer {
	return &Lexer{tokens: tokens}
}

func (self *Lexer) Lex(lval *fuSymType) int {
	if self.next >= len(self.tokens) {
		return 0				// eof
	}
	toktext := self.tokens[self.next]
	self.next += 1
	if toktext.token == QSTRING {
		// strip the quotes: they're preserved by the tokenizer,
		// but not part of the string value
		lval.qstring = toktext.text[1:len(toktext.text)-1]
	}
	_lasttok = &toktext
	return toktext.token
}

func (self *Lexer) Error(e string) {
	_syntaxerror = &SyntaxError{
		filename: _lasttok.filename,
		line: _lasttok.lineno,
		badtoken: _lasttok.text,
		message: e}
}
