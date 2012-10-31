// -*- mode: text; tab-width: 4; indent-tabs-mode: t -*-

%{
package dsl

import (
	"fmt"
)

const BADTOKEN = -1
%}

%union {
	root ASTRoot
	node ASTNode
	nodelist []ASTNode
	expr ASTExpression
	exprlist []ASTExpression
	text string
	stringlist []string
}

%type <root> script
%type <nodelist> elementlist
%type <node> element
%type <node> import
%type <stringlist> dottedname
%type <node> global
%type <node> inline
%type <node> phase
%type <nodelist> block
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
%type <stringlist> patternlist

%token <text> NAME QSTRING INLINE FILEPATTERN
%token EOL EOF IMPORT PLUGIN L3BRACE R3BRACE

%%

script:
	elementlist EOF
	{
		$$ = ASTRoot{elements: $1}
		fulex.(*Parser).ast = &$$
	}
|	EOF
	{
		$$ = ASTRoot{}
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
		$$ = ASTImport{plugin: $2}
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

global:
	assignment

inline:
	PLUGIN NAME L3BRACE INLINE R3BRACE
	{
		$$ = ASTInline{lang: $2, content: $4}
	}

phase:
	NAME block
	{
		$$ = ASTPhase{name: $1, statements: $2}
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
|	buildrule EOL			{ $$ = $1 }
|	expr EOL				{ $$ = $1 }

assignment:
	NAME '=' expr
	{
		$$ = ASTAssignment{target: $1, expr: $3}
	}

buildrule:
	expr ':' expr block
	{
		checkActions($4)
		$$ = ASTBuildRule{targets: $1, sources: $3, actions: $4}
	}

expr:
	addexpr

addexpr:
	postfixexpr				{ $$ = $1 }
|	addexpr '+' postfixexpr	{ $$ = ASTAdd{op1: $1, op2: $3} }

postfixexpr:
	primaryexpr
|	functioncall
|	selection

primaryexpr:
	'(' expr ')'			{ $$ = $2 }
|	NAME					{ $$ = ASTName{$1}}
|	QSTRING					{ $$ = ASTString{$1}}
|	filelist				{ $$ = $1}

filelist:
	'<' patternlist '>'
	{
		$$ = ASTFileList{patterns: $2}
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
		$$ = ASTFunctionCall{function: $1, args: []ASTExpression {}}
	}
|	postfixexpr '(' arglist ')'
	{
		$$ = ASTFunctionCall{function: $1, args: $3}
	}
|	postfixexpr '(' arglist ',' ')'
	{
		$$ = ASTFunctionCall{function: $1, args: $3}
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
		$$ = ASTSelection{container: $1, member: $3}
	}

%%

// a token together with its location, text, etc.
type toktext struct {
	filename string
	lineno int
	token int
	text string
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

func (self *Parser) Error(e string) {
	 self.syntaxerror = &SyntaxError{
		badtoken: &self.tokens[self.next-1],
		message: e}
}

// Check if all of the statements in nodes are valid actions for a
// build rule: either a bare string (shell command), a function call,
// or a variable assignment. Return a slice of error objects, one for
// each invalid action.
func checkActions(nodes []ASTNode) []error {
	var errors []error
	for _, node := range nodes {
		_, ok1 := node.(ASTString)
		_, ok2 := node.(ASTFunctionCall)
		_, ok3 := node.(ASTAssignment)
		if !(ok1 || ok2 || ok3) {
			errors = append(errors, SemanticError{
				node: node,
				message: "invalid build action: must be either bare string, function call, or variable assignment"})
		}
	}
	return errors
}
