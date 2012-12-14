// Copyright © 2012, Greg Ward. All rights reserved.
// Use of this source code is governed by a BSD-style license that can
// be found in the LICENSE.txt file.

package dsl

import (
	"fmt"
	"os"
)

type SyntaxError struct {
	badtoken *token
	message  string
}

func (self SyntaxError) Error() string {
	badtok := self.badtoken.id
	badtext := self.badtoken.text
	if badtok == EOF {
		badtext = "EOF"
	} else if badtok == EOL && badtext == "\n" {
		badtext = "EOL"
	} else if badtok == EOL && badtext == "" {
		// synthetic EOL inserted right before EOF -- perhaps this
		// should be reported as EOL too?
		badtext = "EOF"
	} else if badtok == '\'' {
		badtext = "\"'\""
	} else if len(badtext) == 1 {
		badtext = "'" + badtext + "'"
	}
	return fmt.Sprintf("%s%s (near %s)",
		self.badtoken.location, self.message, badtext)
}

func Parse(filename string) (*ASTRoot, []error) {
	infile, err := os.Open(filename)
	if err != nil {
		return nil, []error{err}
	}
	defer infile.Close()
	scanner, err := NewFileScanner(filename, infile)
	if err != nil {
		return nil, []error{err}
	}
	scanner.scan()
	return parseTokens(scanner.tokens)
}

// mainly used in unit tests
func ParseString(filename string, input string) (*ASTRoot, []error) {
	scanner := NewScanner(filename, ([]byte)(input))
	scanner.scan()
	return parseTokens(scanner.tokens)
}

func parseTokens(tokens []token) (*ASTRoot, []error) {
	parser := NewParser(tokens)
	fuParse(parser)
	if parser.syntaxerror != nil {
		return parser.ast, []error{parser.syntaxerror}
	}
	errors := checkAST(parser.ast)
	return parser.ast, errors
}
