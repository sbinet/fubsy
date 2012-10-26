package dsl

import (
	"io"
	"fmt"
	"strings"
	"reflect"
)

// interface for the whole AST, not just a particular node
// (implemented by ASTRoot)
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
type ASTExpression interface {
	ASTNode
	fmt.Stringer
}

type ASTRoot struct {
	elements []ASTNode
}

// import a single plugin, e.g. "import NAME"
type ASTImport struct {
	plugin []string				// fully-qualified name split on '.'
}

// an inline plugin, e.g. "plugin LANG {{{ CONTENT }}}"
type ASTInline struct {
	lang string
	content string
}

// a build phase, e.g. "NAME { STMTS }"
type ASTPhase struct {
	name string
	statements []ASTNode
}

// NAME = EXPR (global or local)
type ASTAssignment struct {
	target string
	expr ASTNode
}

// OP1 + OP2 (string/list concatenation)
type ASTAdd struct {
	op1 ASTExpression
	op2 ASTExpression
}

// FUNC(arg, arg, ...) (N.B. FUNC is really an expr, to allow
// for code like "(a.b.c())(stuff))"
type ASTFunctionCall struct {
	function ASTExpression
	args []ASTExpression
}

// member selection: CONTAINER.NAME where CONTAINER is any expr
type ASTSelection struct {
	container ASTExpression
	member string
}

// a bare name, like foo in "a = foo" or "foo()"
type ASTName struct {
	name string
}

// a single string
type ASTString struct {
	value string
}

// a list of filename patterns, e.g. [foo*.c **/*.h]
type ASTFileList struct {
	patterns []string
}

func (self ASTRoot) Dump(writer io.Writer, indent string) {
	fmt.Fprintln(writer, indent + "ASTRoot {")
	if self.elements != nil {
		for _, child := range self.elements {
			child.Dump(writer, indent + "  ")
		}
	}
	fmt.Fprintln(writer, indent + "}")
}

func (self ASTRoot) Equal(other_ ASTNode) bool {
	if other, ok := other_.(ASTRoot); ok {
		return reflect.DeepEqual(self, other)
	}
	return false
}

func (self ASTRoot) ListPlugins() []string {
	return []string {"foo", "bar", "baz"}
}

func (self ASTImport) Dump(writer io.Writer, indent string) {
	fmt.Fprintf(writer, "%sASTImport[%s]\n", indent, strings.Join(self.plugin, "."))
}

func (self ASTImport) Equal(other_ ASTNode) bool {
	if other, ok := other_.(ASTImport); ok {
		return reflect.DeepEqual(self, other)
	}
	return false
}

func (self ASTInline) Dump(writer io.Writer, indent string) {
	fmt.Fprintf(writer, "%sASTInline[%s] {{{", indent, self.lang)
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

func (self ASTInline) Equal(other_ ASTNode) bool {
	if other, ok := other_.(ASTInline); ok {
		return self == other
	}
	return false
}

func (self ASTPhase) Dump(writer io.Writer, indent string) {
	fmt.Fprintf(writer, "%sASTPhase[%s] {\n", indent, self.name)
	for _, node := range self.statements {
		node.Dump(writer, indent + "  ")
	}
	fmt.Fprintln(writer, indent + "}")
}

func (self ASTPhase) Equal(other_ ASTNode) bool {
	if other, ok := other_.(ASTPhase); ok {
		return reflect.DeepEqual(self, other)
	}
	return false
}

func (self ASTAssignment) Dump(writer io.Writer, indent string) {
	fmt.Fprintf(writer, "%sASTAssignment[%s]\n", indent, self.target)
	self.expr.Dump(writer, indent + "  ")
}

func (self ASTAssignment) Equal(other_ ASTNode) bool {
	if other, ok := other_.(ASTAssignment); ok {
		return self == other
	}
	return false
}

func (self ASTAdd) Dump(writer io.Writer, indent string) {
	fmt.Fprintf(writer, "%sASTAdd\n", indent)
	fmt.Fprintf(writer, "%sop1:\n", indent)
	self.op1.Dump(writer, indent + "  ")
	fmt.Fprintf(writer, "%sop2:\n", indent)
	self.op2.Dump(writer, indent + "  ")
}

func (self ASTAdd) Equal(other_ ASTNode) bool {
	if other, ok := other_.(ASTAdd); ok {
		return reflect.DeepEqual(self, other)
	}
	return false
}

func (self ASTAdd) String() string {
	return fmt.Sprintf("%s + %s", self.op1, self.op2)
}

func (self ASTFunctionCall) Dump(writer io.Writer, indent string) {
	fmt.Fprintf(writer, "%sASTFunctionCall[%s] (%d args)\n",
		indent, self.function, len(self.args))
	for _, arg := range self.args {
		arg.Dump(writer, indent + "  ")
	}
}

func (self ASTFunctionCall) Equal(other_ ASTNode) bool {
	if other, ok := other_.(ASTFunctionCall); ok {
		return reflect.DeepEqual(self, other)
	}
	return false
}

func (self ASTFunctionCall) String() string {
	args := make([]string, len(self.args))
	for i, arg := range self.args {
		args[i] = arg.String()
	}
	return fmt.Sprintf("%s(%s)", self.function, strings.Join(args, ", "))
}

func (self ASTSelection) Dump(writer io.Writer, indent string) {
	fmt.Fprintf(writer, "%sASTSelection[%s: %s]\n",
		indent, self.container, self.member)
}

func (self ASTSelection) Equal(other_ ASTNode) bool {
	if other, ok := other_.(ASTSelection); ok {
		return self == other
	}
	return false
}

func (self ASTSelection) String() string {
	return fmt.Sprintf("%s.%s", self.container, self.member)
}

func (self ASTName) Dump(writer io.Writer, indent string) {
	fmt.Fprintf(writer, "%sASTName[%s]\n", indent, self.name)
}

func (self ASTName) Equal(other_ ASTNode) bool {
	if other, ok := other_.(ASTName); ok {
		return self == other
	}
	return false
}

func (self ASTName) String() string {
	return self.name
}

func (self ASTString) Dump(writer io.Writer, indent string) {
	fmt.Fprintln(writer,
		indent + "ASTString[" + self.value + "]")
}

func (self ASTString) Equal(other_ ASTNode) bool {
	if other, ok := other_.(ASTString); ok {
		return self == other
	}
	return false
}

func (self ASTString) String() string {
	// this assumes that Go syntax for strings is Fubsy syntax!
	return fmt.Sprintf("%#v", self.value)
}

func (self ASTFileList) Dump(writer io.Writer, indent string) {
	fmt.Fprintln(writer,
		indent + "ASTFileList[" + strings.Join(self.patterns, " ") + "]")
}

func (self ASTFileList) Equal(other_ ASTNode) bool {
	if other, ok := other_.(ASTFileList); ok {
		return reflect.DeepEqual(self, other)
	}
	return false
}

func (self ASTFileList) String() string {
	return "[" + strings.Join(self.patterns, " ") + "]"
}
