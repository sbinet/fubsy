package fubsy

import (
	"io"
	"os"
	"fmt"
	"strings"
	"reflect"
)


// interface for the whole AST, not just a particular node
// (implemented by RootNode)
type AST interface {
	ListPlugins() []string
	ASTNode
}

// interface for any particular node in the AST (root, internal,
// leaves, whatever)
type ASTNode interface {
	Dump(writer io.Writer, indent string)
	Equal(other ASTNode) bool
}

type RootNode struct {
	elements []ASTNode
}

// a list of strings, e.g. ["foo"]
type ListNode struct {
	values []string
}

// an inline plugin, e.g. "plugin LANG {{{ CONTENT }}}"
type InlineNode struct {
	lang string
	content string
}

// argh: why not pointer receiver?
func (self RootNode) Dump(writer io.Writer, indent string) {
	fmt.Fprintln(writer, indent + "RootNode {")
	if self.elements != nil {
		for _, child := range self.elements {
			child.Dump(writer, indent + "  ")
		}
	}
	fmt.Fprintln(writer, indent + "}")
}

func (self RootNode) Equal(other_ ASTNode) bool {
	if other, ok := other_.(RootNode); ok {
		return reflect.DeepEqual(self, other)
	}
	return false
}

func (self RootNode) ListPlugins() []string {
	return []string {"foo", "bar", "baz"}
}

// argh: why not pointer receiver?
func (self ListNode) Dump(writer io.Writer, indent string) {
	fmt.Fprintln(writer,
		indent + "ListNode[" + strings.Join(self.values, ", ") + "]")
}

func (self ListNode) Equal(other_ ASTNode) bool {
	if other, ok := other_.(ListNode); ok {
		return reflect.DeepEqual(self, other)
	}
	return false
}

func (self InlineNode) Dump(writer io.Writer, indent string) {
	fmt.Fprintf(writer,
		"%sInlineNode[%s] %#v\n", indent, self.lang, self.content)
}

func (self InlineNode) Equal(other_ ASTNode) bool {
	if other, ok := other_.(InlineNode); ok {
		return self == other
	}
	return false
}

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

func Parse(filename string) (AST, error) {
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
