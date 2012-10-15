package fubsy

import (
	"io"
	"fmt"
	"strings"
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

// a list of strings, e.g. ["foo"]
type ListNode struct {
	values []string
}

// argh: why not pointer receiver?
func (self RootNode) Dump(writer io.Writer, indent string) {
	fmt.Fprintln(writer, indent + "RootNode {")
	for _, child := range self.elements {
		child.Dump(writer, indent + "  ")
	}
	fmt.Fprintln(writer, indent + "}")
}

func (self RootNode) Equal(other_ ASTNode) bool {
	if other, ok := other_.(RootNode); ok {
		if len(self.elements) != len(other.elements) {
			return false
		}
		for i := range self.elements {
			if !self.elements[i].Equal(other.elements[i]) {
				return false
			}
		}
		return true
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
	// XXX *very* similar to RootNode.Equal()!
	if other, ok := other_.(ListNode); ok {
		if len(self.values) != len(other.values) {
			return false
		}
		for i := range self.values {
			if self.values[i] != other.values[i] {
				return false
			}
		}
		return true
	}
	return false
}

// hmmm: this is *awfully* similar to BadToken in fulex.go
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
	if filename == "bogus" {
		return nil, ParseError{"that's a bogus filename"}
	}


	return nil, nil
}

type ParseError struct {
	msg string
}

func (self ParseError) Error() string {
	return self.msg
}
