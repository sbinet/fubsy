%{
package dsl

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
	nodelist []ASTNode
	expr ExpressionNode
	text string
	stringlist []string
}

%type <root> script
%type <node> element
%type <node> import
%type <stringlist> dottedname
%type <node> inline
%type <node> phase
%type <nodelist> block
%type <nodelist> statementlist
%type <node> statement
%type <node> assignment
%type <expr> expr
%type <expr> primaryexpr
%type <expr> functioncall
%type <expr> selection
%type <expr> stringlist

%token <text> NAME QSTRING INLINE
%token IMPORT PLUGIN L3BRACE R3BRACE

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
	import
|	inline
|	phase

import:
	IMPORT dottedname ';'
	{
		$$ = ImportNode{plugin: $2}
	}

dottedname:
	dottedname '.' NAME
	{
		$$ = append($1, $3)
	}
|	NAME
	{
		$$ = []string {$1}
	}

inline:
	PLUGIN NAME L3BRACE INLINE R3BRACE
	{
		$$ = InlineNode{lang: $2, content: $4}
	}

phase:
	NAME block
	{
		$$ = PhaseNode{name: $1, statements: $2}
	}

block:
	'{' statementlist '}'
	{
		$$ = $2
	}
|	'{' '}'
	{
		$$ = []ASTNode {}
	}

statementlist:
	statementlist statement
	{
		$$ = append($1, $2)
	}
|	statement
	{
		$$ = []ASTNode {$1}
	}

statement:
	assignment ';'			{ $$ = $1 }
|	expr ';'				{ $$ = $1 }

assignment:
	NAME '=' expr
	{
		$$ = AssignmentNode{target: $1, expr: $3}
	}

expr:
	primaryexpr
|	functioncall
|	selection

primaryexpr:
	'(' expr ')'			{ $$ = $2 }
|	NAME					{ $$ = NameNode{$1}}
|	QSTRING					{ $$ = StringNode{$1}}
|	stringlist				{ $$ = $1}

stringlist:
	'[' QSTRING ']'
	{
		$$ = ListNode{values: []string {$2}}
	}

functioncall:
	expr '(' ')'
	{
		$$ = FunctionCallNode{function: $1, args: []ExpressionNode {}}
	}

selection:
	expr '.' NAME
	{
		$$ = SelectionNode{container: $1, member: $3}
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
	case INLINE, NAME:
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
