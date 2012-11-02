package dsl

import (
	"os"
	"fmt"
)

type SyntaxError struct {
	badtoken *toktext
	message string
}

func (self SyntaxError) Error() string {
	badtok := self.badtoken.token
	badtext := self.badtoken.text
	if badtok == EOF {
		badtext = "EOF"
	} else if badtok == EOL && badtext == "\n" {
		badtext = "EOL"
	} else 	if badtok == EOL && badtext == "" {
		// synthetic EOL inserted right before EOF -- perhaps this
		// should be reported as EOL too?
		badtext = "EOF"
	} else if badtok == '\'' {
		badtext = "\"'\""
	} else if len(badtext) == 1 {
		badtext = "'" + badtext + "'"
	}
	return fmt.Sprintf("%s: %s (near %s)",
		self.badtoken.location, self.message, badtext)
}

func Parse(filename string) (*ASTRoot, []error) {
	infile, err := os.Open(filename)
	if err != nil {
		return nil, []error {err}
	}
	defer infile.Close()
	scanner, err := NewFileScanner(filename, infile)
	if err != nil {
		return nil, []error {err}
	}
	scanner.scan()

	parser := NewParser(scanner.tokens)
	fuParse(parser)
	if parser.syntaxerror != nil {
		return parser.ast, []error {parser.syntaxerror}
	}
	errors := checkAST(parser.ast)
	return parser.ast, errors
}
