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

// import a single plugin, e.g. "import NAME"
type ImportNode struct {
	plugin []string				// fully-qualified name split on '.'
}

// an inline plugin, e.g. "plugin LANG {{{ CONTENT }}}"
type InlineNode struct {
	lang string
	content string
}

// a build phase, e.g. "NAME { STMTS }"
type PhaseNode struct {
	name string
	statements []ASTNode
}

// a list of strings, e.g. ["foo"]
type ListNode struct {
	values []string
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

func (self ImportNode) Dump(writer io.Writer, indent string) {
	fmt.Fprintf(writer, "%sImportNode[%s]\n", indent, self.plugin)
}

func (self ImportNode) Equal(other_ ASTNode) bool {
	if other, ok := other_.(ImportNode); ok {
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

func (self PhaseNode) Dump(writer io.Writer, indent string) {
	fmt.Fprintf(writer, "%sPhaseNode[%s] {\n", indent, self.name)
	for _, node := range self.statements {
		node.Dump(writer, indent + "  ")
	}
	fmt.Fprintln(writer, indent + "}")
}

func (self PhaseNode) Equal(other_ ASTNode) bool {
	if other, ok := other_.(PhaseNode); ok {
		return reflect.DeepEqual(self, other)
	}
	return false
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
