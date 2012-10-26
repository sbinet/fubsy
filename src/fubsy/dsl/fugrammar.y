// -*- mode: text; tab-width: 4; indent-tabs-mode: t -*-

%{
package dsl

import (
	"fmt"
)

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
%type <expr> addexpr
%type <expr> postfixexpr
%type <expr> primaryexpr
%type <expr> functioncall
%type <exprlist> arglist
%type <expr> selection
%type <expr> filelist
%type <stringlist> patternlist

%token <text> NAME QSTRING INLINE FILEPATTERN
%token EOL EOF IMPORT PLUGIN L3BRACE R3BRACE

%%

script:
	elementlist EOF
	{
		$$ = RootNode{elements: $1}
		fulex.(*Lexer).ast = &$$
	}
|	EOF
	{
		$$ = RootNode{}
		fulex.(*Lexer).ast = &$$
	}

elementlist:
	elementlist element EOL
	{
		$$ = append($1, $2)
	}
|	element EOL
	{
		$$ = []ASTNode {$1}
	}

element:
	import
|	inline
|	phase

import:
	IMPORT dottedname
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
	'{' EOL statementlist '}'
	{
		$$ = $3
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
	assignment EOL			{ $$ = $1 }
|	expr EOL				{ $$ = $1 }

assignment:
	NAME '=' expr
	{
		$$ = AssignmentNode{target: $1, expr: $3}
	}

expr:
	addexpr

addexpr:
	postfixexpr				{ $$ = $1 }
|	addexpr '+' postfixexpr	{ $$ = AddNode{op1: $1, op2: $3} }

postfixexpr:
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
	postfixexpr '(' ')'
	{
		$$ = FunctionCallNode{function: $1, args: []ExpressionNode {}}
	}
|	postfixexpr '(' arglist ')'
	{
		$$ = FunctionCallNode{function: $1, args: $3}
	}
|	postfixexpr '(' arglist ',' ')'
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
	postfixexpr '.' NAME
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
	// internal state
	tokens []toktext
	next int

	// results for caller to use
	ast *RootNode
	syntaxerror *SyntaxError
}

func NewLexer(tokens []toktext) *Lexer {
	return &Lexer{tokens: tokens}
}

func (self *Lexer) Lex(lval *fuSymType) int {
	if self.next >= len(self.tokens) {
		return 0				// eof
	}
	toktext := self.tokens[self.next]
	self.next++
	switch toktext.token {
	case QSTRING:
		// strip the quotes: they're preserved by the tokenizer,
		// but not part of the string value
		lval.text = toktext.text[1:len(toktext.text)-1]
	case INLINE, NAME, FILEPATTERN:
		lval.text = toktext.text
	}
	return toktext.token
}

func (self *Lexer) Error(e string) {
	 self.syntaxerror = &SyntaxError{
		badtoken: &self.tokens[self.next-1],
		message: e}
}
