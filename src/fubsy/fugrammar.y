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
	root RootNode
	node ASTNode
	text string
}

%type <root> script
%type <node> element
%type <node> stringlist
%type <node> inline

%token <text> QSTRING INLINE
%token L3BRACE R3BRACE

%%

script:
	script element
	{
		$1.elements = append($1.elements, $2)
		$$ = $1
	}
|	element
	{
		$$ = RootNode{elements: []ASTNode {$1}}
		_ast = &$$
	}

element:
	stringlist
|	inline

stringlist:
	'[' QSTRING ']'
	{
		$$ = ListNode{values: []string {$2}}
	}

inline:
	L3BRACE INLINE R3BRACE
	{
		$$ = InlineNode{content: $2}
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
    switch toktext.token {
	case QSTRING:
		// strip the quotes: they're preserved by the tokenizer,
		// but not part of the string value
		lval.text = toktext.text[1:len(toktext.text)-1]
	case INLINE:
		lval.text = toktext.text
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
