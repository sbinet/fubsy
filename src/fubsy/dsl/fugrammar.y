// -*- mode: text; tab-width: 4; indent-tabs-mode: t -*-

%{
package dsl

import (
	"fmt"
)

const BADTOKEN = -1
%}

%union {
	token toktext

	root ASTRoot
	node ASTNode
	nodelist []ASTNode
	expr ASTExpression
	exprlist []ASTExpression
	tokenlist []Token
}

%type <root> script
%type <nodelist> elementlist
%type <node> element
%type <node> import
%type <tokenlist> dottedname
%type <node> global
%type <node> inline
%type <node> phase
%type <node> block
%type <nodelist> statementlist
%type <node> statement
%type <node> assignment
%type <node> buildrule
%type <expr> expr
%type <expr> addexpr
%type <expr> postfixexpr
%type <expr> primaryexpr
%type <expr> functioncall
%type <exprlist> arglist
%type <expr> selection
%type <expr> filelist
%type <tokenlist> patternlist

%token <token> IMPORT PLUGIN INLINE NAME QSTRING FILEPATTERN
%token <token> '(' ')' '<' '>' '{' '}'
%token EOL EOF PLUGIN L3BRACE R3BRACE

%%

script:
	elementlist EOF
	{
		$$ = NewASTRoot($1)
		fulex.(*Parser).ast = &$$
	}
|	EOF
	{
		$$ = NewASTRoot([]ASTNode {})
		fulex.(*Parser).ast = &$$
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
|	global
|	inline
|	phase

import:
	IMPORT dottedname
	{
		$$ = NewASTImport($1, $2)
	}

dottedname:
	dottedname '.' NAME
	{
		$$ = append($1, $3)
	}
|	NAME
	{
		$$ = []Token {$1}
	}

global:
	assignment

inline:
	PLUGIN NAME L3BRACE INLINE R3BRACE
	{
		$$ = NewASTInline($1, $2, $4)
	}

phase:
	NAME block
	{
		$$ = NewASTPhase($1, $2.(ASTBlock))
	}

block:
	'{' EOL statementlist '}'
	{
		$$ = NewASTBlock($1, $3, $4)
	}
|	'{' '}'
	{
		$$ = NewASTBlock($1, []ASTNode {}, $2)
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
|	buildrule EOL			{ $$ = $1 }
|	expr EOL				{ $$ = $1 }

assignment:
	NAME '=' expr
	{
		$$ = NewASTAssignment($1, $3)
	}

buildrule:
	expr ':' expr block
	{
		// some actions could be invalid: we check those in check.go
		// after parsing is done
		$$ = NewASTBuildRule($1, $3, $4.(ASTBlock))
	}

expr:
	addexpr

addexpr:
	postfixexpr				{ $$ = $1 }
|	addexpr '+' postfixexpr	{ $$ = NewASTAdd($1, $3) }

postfixexpr:
	primaryexpr
|	functioncall
|	selection

primaryexpr:
	'(' expr ')'			{ $$ = $2 }
|	NAME					{ $$ = NewASTName($1) }
|	QSTRING					{ $$ = NewASTString($1)}
|	filelist				{ $$ = $1}

filelist:
	'<' patternlist '>'
	{
		$$ = NewASTFileList($1, $2, $3)
	}

patternlist:
	patternlist FILEPATTERN
	{
		$$ = append($1, $2)
	}
|	FILEPATTERN
	{
		$$ = []Token {$1}
	}

functioncall:
	postfixexpr '(' ')'
	{
		$$ = NewASTFunctionCall($1, []ASTExpression {}, $3)
	}
|	postfixexpr '(' arglist ')'
	{
		$$ = NewASTFunctionCall($1, $3, $4)
	}
|	postfixexpr '(' arglist ',' ')'
	{
		$$ = NewASTFunctionCall($1, $3, $5)
	}

arglist:
	arglist ',' expr
	{
		$$ = append($1, $3)
	}
|	expr
	{
		$$ = []ASTExpression{$1}
	}

selection:
	postfixexpr '.' NAME
	{
		$$ = NewASTSelection($1, $3)
	}

%%

// a token together with its location, text, etc.
type toktext struct {
	location location
	token int
	text string
}

// implement the Locatable interface
func (self toktext) Location() location {
	return self.location
}

// implement the Token interface defined by ast.go
func (self toktext) Text() string {
	 return self.text
}


type Parser struct {
	// internal state (fed to parser by Lex() method)
	tokens []toktext
	next int

	// results for caller to use
	ast *ASTRoot
	syntaxerror *SyntaxError
}

func NewParser(tokens []toktext) *Parser {
	return &Parser{tokens: tokens}
}

func (self *Parser) Lex(lval *fuSymType) int {
	if self.next >= len(self.tokens) {
		return 0				// eof
	}
	toktext := self.tokens[self.next]
	self.next++
	lval.token = toktext
	return toktext.token
}

func (self *Parser) Error(e string) {
	 self.syntaxerror = &SyntaxError{
		badtoken: &self.tokens[self.next-1],
		message: e}
}
