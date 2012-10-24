// -*- mode: text; tab-width: 4; indent-tabs-mode: t -*-

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
	exprlist []ExpressionNode
	text string
	stringlist []string
}

%type <root> script
%type <nodelist> elementlist
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
%type <exprlist> arglist
%type <expr> selection
%type <expr> filelist
%type <stringlist> patternlist

%token <text> NAME QSTRING INLINE FILEPATTERN
%token EOL IMPORT PLUGIN L3BRACE R3BRACE

%%

script:
	eolopt elementlist
	{
		$$ = RootNode{elements: $2}
		_ast = &$$
	}

elementlist:
	elementlist element
	{
		$$ = append($1, $2)
	}
|	element
	{
		$$ = []ASTNode {$1}
	}

element:
	import
|	inline
|	phase

import:
	IMPORT dottedname eol
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
	PLUGIN NAME L3BRACE INLINE R3BRACE eol
	{
		$$ = InlineNode{lang: $2, content: $4}
	}

phase:
	NAME block
	{
		$$ = PhaseNode{name: $1, statements: $2}
	}

block:
	'{' eol statementlist '}' eolopt
	{
		$$ = $3
	}
|	'{' '}' eolopt
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
	assignment eol			{ $$ = $1 }
|	expr eol				{ $$ = $1 }

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
|	filelist				{ $$ = $1}

filelist:
	'<' patternlist '>'
	{
		$$ = FileListNode{patterns: $2}
	}

patternlist:
	patternlist FILEPATTERN
	{
		$$ = append($1, $2)
	}
|	FILEPATTERN
	{
		$$ = []string {$1}
	}

functioncall:
	expr '(' ')'
	{
		$$ = FunctionCallNode{function: $1, args: []ExpressionNode {}}
	}
|	expr '(' arglist ')'
	{
		$$ = FunctionCallNode{function: $1, args: $3}
	}
|	expr '(' arglist ',' ')'
	{
		$$ = FunctionCallNode{function: $1, args: $3}
	}

arglist:
	arglist ',' expr
	{
		$$ = append($1, $3)
	}
|	expr
	{
		$$ = []ExpressionNode{$1}
	}

selection:
	expr '.' NAME
	{
		$$ = SelectionNode{container: $1, member: $3}
	}

// sequence of one or more newlines
eol:
	eol EOL
|	EOL

// sequence of zero or more newlines
eolopt:
	eol
|

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
	case INLINE, NAME, FILEPATTERN:
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
