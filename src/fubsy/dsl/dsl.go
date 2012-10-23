package dsl

import (
	"os"
	"fmt"
)

type SyntaxError struct {
	filename string
	line int
	message string
	badtoken string
}

func (self SyntaxError) Error() string {
	return fmt.Sprintf("%s:%d: %s (near %v)",
		self.filename, self.line, self.message, self.badtoken)
}

func Parse(filename string) (*RootNode, error) {
	infile, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer infile.Close()
	scanner, err := NewFileScanner(filename, infile)
	if err != nil {
		return nil, err
	}
	scanner.scan()
/*
	fmt.Printf("scanner found %d tokens:\n", len(scanner.tokens))
	for i, toktext := range scanner.tokens {
		if toktext.token == BADTOKEN {
			fmt.Fprintf(os.Stderr, "%s, line %d: invalid input: %s (ignored)\n",
				toktext.filename, toktext.lineno, toktext.text)
		} else {
			fmt.Printf("[%d] %#v\n", i, toktext)
		}
	}
*/

	lexer := NewLexer(scanner.tokens)
	err = nil
	fuParse(lexer)
	if _syntaxerror != nil {
		err = _syntaxerror
	}
	return _ast, err
}
