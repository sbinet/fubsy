// -*- mode: text; tab-width: 4; indent-tabs-mode: t -*-

// Copyright © 2012-2013, Greg Ward. All rights reserved.
// Use of this source code is governed by a BSD-style license that can
// be found in the LICENSE.txt file.

%{
package dsl

import (
	"fmt"
	"strings"
)

const BADTOKEN = -1
%}

%union {
	token token

	root *ASTRoot
	node ASTNode
	nodelist []ASTNode
	expr ASTExpression
	exprlist []ASTExpression
	tokenlist []token
}

%type <root> script
%type <nodelist> elementlist
%type <node> element
%type <node> import
%type <tokenlist> dottedname
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
%type <expr> list
%type <expr> functioncall
%type <exprlist> exprlist
%type <expr> selection
%type <expr> filefinder
%type <tokenlist> patternlist

%token <token> IMPORT PLUGIN INLINE NAME QSTRING FILEPATTERN R3BRACE
%token <token> '(' ')' '[' ']' '<' '>' '{' '}'
%token EOL EOF PLUGIN L3BRACE R3BRACE

%%

script:
	elementlist EOF
	{
		$$ = NewASTRoot($1)
		fulex.(*Parser).ast = $$
	}
|	EOF
	{
		$$ = NewASTRoot([]ASTNode {})
		fulex.(*Parser).ast = $$
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
		$$ = NewASTImport(extractText($2), $1, $2[len($2)-1])
	}

dottedname:
	dottedname '.' NAME
	{
		$$ = append($1, $3)
	}
|	NAME
	{
		$$ = []token {$1}
	}

inline:
	PLUGIN NAME L3BRACE INLINE R3BRACE
	{
		parser := fulex.(*Parser)
		content, err := cleanInlineContent(parser, $4.text)
		if err != nil {
			parser.SetError(err)
		} else {
			$$ = NewASTInline($2.text, content, $1, $5)
		}
	}

phase:
	NAME block
	{
		$$ = NewASTPhase($1.text, $2.(*ASTBlock), $1, $2)
	}

block:
	'{' EOL statementlist '}'
	{
		$$ = NewASTBlock($3, $1, $4)
	}
|	'{' '}'
	{
		$$ = NewASTBlock([]ASTNode {}, $1, $2)
	}
|	'{' EOL '}'
	{
		$$ = NewASTBlock([]ASTNode {}, $1, $3)
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
		$$ = NewASTAssignment($1.text, $3, $1, $3)
	}

buildrule:
	expr ':' expr block
	{
		// some actions could be invalid: we check those in check.go
		// after parsing is done
		$$ = NewASTBuildRule($1, $3, $4.(*ASTBlock))
	}

expr:
	addexpr

addexpr:
	postfixexpr				{ $$ = $1 }
|	addexpr '+' postfixexpr	{ $$ = NewASTAdd($1, $3) }

postfixexpr:
	primaryexpr
|	list
|	functioncall
|	selection

primaryexpr:
	'(' expr ')'			{ $$ = $2 }
|	NAME					{ $$ = NewASTName($1.text, $1) }
|	QSTRING					{ $$ = NewASTString($1.text, $1)}
|	filefinder				{ $$ = $1}

filefinder:
	'<' patternlist '>'
	{
		$$ = NewASTFileFinder(extractText($2), $1, $3)
	}

patternlist:
	patternlist FILEPATTERN
	{
		$$ = append($1, $2)
	}
|	FILEPATTERN
	{
		$$ = []token {$1}
	}

list:
	'[' ']'
	{
		$$ = NewASTList([]ASTExpression{}, $1, $2)
	}
|	'[' exprlist ']'
	{
		$$ = NewASTList($2, $1, $3)
	}
|	'[' exprlist ',' ']'
	{
		$$ = NewASTList($2, $1, $4)
	}

functioncall:
	postfixexpr '(' ')'
	{
		$$ = NewASTFunctionCall($1, []ASTExpression {}, $1, $3)
	}
|	postfixexpr '(' exprlist ')'
	{
		$$ = NewASTFunctionCall($1, $3, $1, $4)
	}
|	postfixexpr '(' exprlist ',' ')'
	{
		$$ = NewASTFunctionCall($1, $3, $1, $5)
	}

exprlist:
	exprlist ',' expr
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
		$$ = NewASTSelection($1, $3.text, $1, $3)
	}

%%

// a token together with its location, text, etc.
type token struct {
	location FileLocation
	id int
	text string
}

// implement the Locatable interface
func (self token) Location() Location {
	return self.location
}

// implement the Token interface defined by ast.go
func (self token) Text() string {
	 return self.text
}


type Parser struct {
	// internal state (fed to parser by Lex() method)
	tokens []token
	next int

	// results for caller to use
	ast *ASTRoot
	syntaxerror *SyntaxError
}

func NewParser(tokens []token) *Parser {
	return &Parser{tokens: tokens}
}

func (self *Parser) Lex(lval *fuSymType) int {
	if self.next >= len(self.tokens) {
		return 0				// eof
	}
	token := self.tokens[self.next]
	self.next++
	lval.token = token
	return token.id
}

func (self *Parser) Error(message string) {
	self.syntaxerror = self.NewSyntaxError(message)
}

func (self *Parser) SetError(err *SyntaxError) {
	 self.syntaxerror = err
}

func (self *Parser) NewSyntaxError(message string) *SyntaxError {
	 return &SyntaxError{
		badtoken: &self.tokens[self.next-1],
		message: message,
	}
}

func extractText(tokens []token) []string {
	text := make([]string, len(tokens))
	for i, token := range tokens {
		text[i] = token.Text()
	}
	return text
}

func cleanInlineContent(parser *Parser, content string) (string, *SyntaxError) {
	length := len(content)
	if length == 0 {
		return content, parser.NewSyntaxError("inline plugin must contain at least a newline")
	}
	if content == "\n" {
		return "", nil
	}

	var err *SyntaxError
	if content[0] != '\n' {
		err = parser.NewSyntaxError("inline plugin must start with a newline")
		return content, err
	} else if content[length-1] != '\n' {
		err = parser.NewSyntaxError("inline plugin must end with a newline")
		return content, err
	}

	content = content[1 : length-1]

	// trim common leading space from each line
	lines := strings.Split(content, "\n")
	minspace := -1
	for _, line := range lines {
		// safe to treat line as bytes when we're only looking for
		// space (ASCII 32), because in UTF-8 bytes < 128 *only*
		// represent the corresponding code point
		for j, byte := range line {
			if byte != ' ' {
				if minspace < 0 || j < minspace {
					minspace = j
				}
				break
			}
		}

		if minspace == 0 {
			// found an unindented line: give up now
			break
		}
	}
	if minspace > 0 {
		for i, line := range lines {
			if len(line) == 0 {
				continue
			} else if len(line) < minspace {
				panic(fmt.Sprintf("line = %#v, but minspace = %d",
					line, minspace))
			}
			lines[i] = line[minspace:]
		}
		content = strings.Join(lines, "\n")
	}
	return content, err
}
