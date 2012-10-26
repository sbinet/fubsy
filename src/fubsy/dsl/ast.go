package dsl

import (
	"io"
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

// implemented by all AST nodes produced by a grammar rule 'expr : ...',
// i.e. anything that has a value
type ExpressionNode interface {
	ASTNode
	fmt.Stringer
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

// NAME = EXPR (global or local)
type AssignmentNode struct {
	target string
	expr ASTNode
}

// OP1 + OP2 (string/list concatenation)
type AddNode struct {
	op1 ExpressionNode
	op2 ExpressionNode
}

// FUNC(arg, arg, ...) (N.B. FUNC is really an expr, to allow
// for code like "(a.b.c())(stuff))"
type FunctionCallNode struct {
	function ExpressionNode
	args []ExpressionNode
}

// member selection: CONTAINER.NAME where CONTAINER is any expr
type SelectionNode struct {
	container ExpressionNode
	member string
}

// a bare name, like foo in "a = foo" or "foo()"
type NameNode struct {
	name string
}

// a single string
type StringNode struct {
	value string
}

// a list of filename patterns, e.g. [foo*.c **/*.h]
type FileListNode struct {
	patterns []string
}

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
	fmt.Fprintf(writer, "%sImportNode[%s]\n", indent, strings.Join(self.plugin, "."))
}

func (self ImportNode) Equal(other_ ASTNode) bool {
	if other, ok := other_.(ImportNode); ok {
		return reflect.DeepEqual(self, other)
	}
	return false
}

func (self InlineNode) Dump(writer io.Writer, indent string) {
	fmt.Fprintf(writer, "%sInlineNode[%s] {{{", indent, self.lang)
	if len(self.content) > 0 {
		replace := -1			// indent all lines by default
		if strings.HasSuffix(self.content, "\n") {
			// last line doesn't really exist, so don't indent it
			replace = strings.Count(self.content, "\n") - 1
		}
		content := strings.Replace(
			self.content, "\n", "\n" + indent + "  ", replace)
		fmt.Fprintf(writer, content)
	}
	fmt.Fprintf(writer, "%s}}}\n", indent)
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

func (self AssignmentNode) Dump(writer io.Writer, indent string) {
	fmt.Fprintf(writer, "%sAssignmentNode[%s]\n", indent, self.target)
	self.expr.Dump(writer, indent + "  ")
}

func (self AssignmentNode) Equal(other_ ASTNode) bool {
	if other, ok := other_.(AssignmentNode); ok {
		return self == other
	}
	return false
}

func (self AddNode) Dump(writer io.Writer, indent string) {
	fmt.Fprintf(writer, "%sAddNode\n", indent)
	fmt.Fprintf(writer, "%sop1:\n", indent)
	self.op1.Dump(writer, indent + "  ")
	fmt.Fprintf(writer, "%sop2:\n", indent)
	self.op2.Dump(writer, indent + "  ")
}

func (self AddNode) Equal(other_ ASTNode) bool {
	if other, ok := other_.(AddNode); ok {
		return reflect.DeepEqual(self, other)
	}
	return false
}

func (self AddNode) String() string {
	return fmt.Sprintf("%s + %s", self.op1, self.op2)
}

func (self FunctionCallNode) Dump(writer io.Writer, indent string) {
	fmt.Fprintf(writer, "%sFunctionCallNode[%s] (%d args)\n",
		indent, self.function, len(self.args))
	for _, arg := range self.args {
		arg.Dump(writer, indent + "  ")
	}
}

func (self FunctionCallNode) Equal(other_ ASTNode) bool {
	if other, ok := other_.(FunctionCallNode); ok {
		return reflect.DeepEqual(self, other)
	}
	return false
}

func (self FunctionCallNode) String() string {
	args := make([]string, len(self.args))
	for i, arg := range self.args {
		args[i] = arg.String()
	}
	return fmt.Sprintf("%s(%s)", self.function, strings.Join(args, ", "))
}

func (self SelectionNode) Dump(writer io.Writer, indent string) {
	fmt.Fprintf(writer, "%sSelectionNode[%s: %s]\n",
		indent, self.container, self.member)
}

func (self SelectionNode) Equal(other_ ASTNode) bool {
	if other, ok := other_.(SelectionNode); ok {
		return self == other
	}
	return false
}

func (self SelectionNode) String() string {
	return fmt.Sprintf("%s.%s", self.container, self.member)
}

func (self NameNode) Dump(writer io.Writer, indent string) {
	fmt.Fprintf(writer, "%sNameNode[%s]\n", indent, self.name)
}

func (self NameNode) Equal(other_ ASTNode) bool {
	if other, ok := other_.(NameNode); ok {
		return self == other
	}
	return false
}

func (self NameNode) String() string {
	return self.name
}

func (self StringNode) Dump(writer io.Writer, indent string) {
	fmt.Fprintln(writer,
		indent + "StringNode[" + self.value + "]")
}

func (self StringNode) Equal(other_ ASTNode) bool {
	if other, ok := other_.(StringNode); ok {
		return self == other
	}
	return false
}

func (self StringNode) String() string {
	// this assumes that Go syntax for strings is Fubsy syntax!
	return fmt.Sprintf("%#v", self.value)
}

func (self FileListNode) Dump(writer io.Writer, indent string) {
	fmt.Fprintln(writer,
		indent + "FileListNode[" + strings.Join(self.patterns, " ") + "]")
}

func (self FileListNode) Equal(other_ ASTNode) bool {
	if other, ok := other_.(FileListNode); ok {
		return reflect.DeepEqual(self, other)
	}
	return false
}

func (self FileListNode) String() string {
	return "[" + strings.Join(self.patterns, " ") + "]"
}
