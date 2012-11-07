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
	Locatable

	Dump(writer io.Writer, indent string)

	// Compare the important structural/semantic/syntactic elements
	// represented by this AST node -- in particular, do not compare
	// location data. (This is because Equal() is really there for the
	// unit tests, and making unit tests worry about location is just
	// too much to ask. Plus it makes intuitive sense that two AST
	// nodes that both mean "foo = bar()" are the same wherever they
	// originated.
	Equal(other ASTNode) bool
}

// implemented by all AST nodes produced by a grammar rule 'expr : ...',
// i.e. anything that has a value
type ASTExpression interface {
	ASTNode
	fmt.Stringer
}

// implemented by every AST node via astbase, and also by toktext
type Locatable interface {
	Location() location
}

// something with a location and text (so this file does not know
// about toktext, which is defined in fugrammar.y -- trying to avoid
// dependency cycles within the package)
type Token interface {
	Locatable
	Text() string
}

type astbase struct {
	location
}

func (self astbase) Location() location {
	return self.location
}

func mergeLocations(loc1 Locatable, loc2 Locatable) location {
	return loc1.Location().merge(loc2.Location())
}

// AST nodes with variable number of children
type children []ASTNode

type ASTRoot struct {
	astbase
	children
}

// import a single plugin, e.g. "import NAME"
type ASTImport struct {
	astbase
	plugin []string				// fully-qualified name split on '.'
}

// an inline plugin, e.g. "plugin LANG {{{ CONTENT }}}"
type ASTInline struct {
	astbase
	lang string
	content string
}

// a build phase, e.g. "NAME { STMTS }"
type ASTPhase struct {
	astbase
	name string
	children
}

// the body of a phase or build rule: { node node ... node }
// (not visible in the final AST; only used to get location info from
// the parser to the ASTPhase or ASTBuildRule node)
type ASTBlock struct {
	astbase
	children
}

// NAME = EXPR (global or local)
type ASTAssignment struct {
	astbase
	target string
	expr ASTNode
}

// TARGETS : SOURCES { ACTIONS }
type ASTBuildRule struct {
	astbase
	targets ASTExpression
	sources ASTExpression
	children
}

// OP1 + OP2 (string/list concatenation)
type ASTAdd struct {
	astbase
	op1 ASTExpression
	op2 ASTExpression
}

// FUNC(arg, arg, ...) (N.B. FUNC is an expr to allow for code like
// "(a.b.c())(stuff))"
type ASTFunctionCall struct {
	astbase
	function ASTExpression
	args []ASTExpression
}

// member selection: CONTAINER.NAME where CONTAINER is any expr
type ASTSelection struct {
	astbase
	container ASTExpression
	member string
}

// a bare name, like foo in "a = foo" or "foo()"
type ASTName struct {
	astbase
	name string
}

// a single string
type ASTString struct {
	astbase
	value string
}

// a list of filename patterns, e.g. [foo*.c **/*.h]
type ASTFileList struct {
	astbase
	patterns []string
}

func (self children) Dump(writer io.Writer, indent string) {
	indent += "  "
	for _, child := range self {
		child.Dump(writer, indent)
	}
}

func (self children) Equal(other children) bool {
	return listsEqual(self, other)
}

func NewASTRoot(children []ASTNode) ASTRoot {
	location := mergeLocations(children[0], children[len(children)-1])
	return ASTRoot{
		astbase: astbase{location},
		children: children}
}

func (self ASTRoot) Dump(writer io.Writer, indent string) {
	fmt.Fprintln(writer, indent + "ASTRoot {")
	self.children.Dump(writer, indent)
	fmt.Fprintln(writer, indent + "}")
}

func (self ASTRoot) Equal(other_ ASTNode) bool {
	if other, ok := other_.(ASTRoot); ok {
		return self.children.Equal(other.children)
	}
	return false
}

func (self ASTRoot) ListPlugins() []string {
	return []string {"foo", "bar", "baz"}
}

func NewASTImport(keyword Locatable, names []Token) ASTImport {
	location := mergeLocations(keyword, names[len(names)-1])
	plugin := extractText(names)
	return ASTImport{
		astbase: astbase{location},
		plugin: plugin}
}

func (self ASTImport) Dump(writer io.Writer, indent string) {
	fmt.Fprintf(writer, "%sASTImport[%s]\n", indent, strings.Join(self.plugin, "."))
}

func (self ASTImport) Equal(other_ ASTNode) bool {
	if other, ok := other_.(ASTImport); ok {
		return reflect.DeepEqual(self.plugin, other.plugin)
	}
	return false
}

func NewASTInline(keyword Locatable, lang Token, content Token) ASTInline {
	location := mergeLocations(keyword, content)
	return ASTInline{
		astbase: astbase{location},
		lang: lang.Text(),
		content: content.Text()}
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
		return self.lang == other.lang &&
			self.content == other.content
	}
	return false
}

func NewASTPhase(name Token, block ASTBlock) ASTPhase {
	location := mergeLocations(name, block)
	return ASTPhase{
		astbase: astbase{location},
		name: name.Text(),
		children: block.children}
}

func (self ASTPhase) Dump(writer io.Writer, indent string) {
	fmt.Fprintf(writer, "%sASTPhase[%s] {\n", indent, self.name)
	self.children.Dump(writer, indent)
	fmt.Fprintln(writer, indent + "}")
}

func (self ASTPhase) Equal(other_ ASTNode) bool {
	if other, ok := other_.(ASTPhase); ok {
		return self.name == other.name &&
			self.children.Equal(other.children)
	}
	return false
}

func NewASTBlock(opener Token, children []ASTNode, closer Token) ASTBlock {
	location := mergeLocations(opener, closer)
	return ASTBlock{
		astbase: astbase{location},
		children: children}
}

func (self ASTBlock) Dump(writer io.Writer, indent string) {
	panic("Dump() not supported for ASTBlock (transient node type)")
}

func (self ASTBlock) Equal(other ASTNode) bool {
	panic("Equal() not supported for ASTBlock (transient node type)")
}

func NewASTAssignment(target Token, expr ASTExpression) ASTAssignment {
	location := mergeLocations(target, expr)
	return ASTAssignment{
		astbase: astbase{location},
		target: target.Text(),
		expr: expr}
}

func (self ASTAssignment) Dump(writer io.Writer, indent string) {
	fmt.Fprintf(writer, "%sASTAssignment[%s]\n", indent, self.target)
	self.expr.Dump(writer, indent + "  ")
}

func (self ASTAssignment) Equal(other_ ASTNode) bool {
	if other, ok := other_.(ASTAssignment); ok {
		return self.target == other.target &&
			self.expr.Equal(other.expr)
	}
	return false
}

func NewASTBuildRule(
	targets ASTExpression,
	sources ASTExpression,
	block ASTBlock) ASTBuildRule {
	location := mergeLocations(targets, block)
	return ASTBuildRule{
		astbase: astbase{location},
		targets: targets,
		sources: sources,
		children: block.children}
}

func (self ASTBuildRule) Dump(writer io.Writer, indent string) {
	fmt.Fprintf(writer, "%sASTBuildRule {\n", indent)
	fmt.Fprintf(writer, "%stargets:\n", indent)
	self.targets.Dump(writer, indent + "  ")
	fmt.Fprintf(writer, "%ssources:\n", indent)
	self.sources.Dump(writer, indent + "  ")
	fmt.Fprintf(writer, "%sactions:\n", indent)
	self.children.Dump(writer, indent)
	fmt.Fprintf(writer, "%s}\n", indent)
}

func (self ASTBuildRule) Equal(other_ ASTNode) bool {
	if other, ok := other_.(ASTBuildRule); ok {
		return self.targets.Equal(other.targets) &&
			self.sources.Equal(other.sources) &&
			self.children.Equal(other.children)
	}
	return false
}

func NewASTAdd(op1 ASTExpression, op2 ASTExpression) ASTAdd {
	location := mergeLocations(op1, op2)
	return ASTAdd{
		astbase: astbase{location},
		op1: op1,
		op2: op2}
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
		return self.op1.Equal(other.op1) && self.op2.Equal(other.op2)
	}
	return false
}

func (self ASTAdd) String() string {
	return fmt.Sprintf("%s + %s", self.op1, self.op2)
}

func NewASTFunctionCall(
	function ASTExpression,
	args []ASTExpression,
	closer Locatable) ASTFunctionCall {
	location := mergeLocations(function, closer)
	return ASTFunctionCall{
		astbase: astbase{location},
		function: function,
		args: args}
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
		return self.function.Equal(other.function) &&
			exprlistsEqual(self.args, other.args)
	}
	return false
}

func (self ASTFunctionCall) String() string {
	// args := make([]string, len(self.args))
	// for i, arg := range self.args {
	// 	args[i] = arg.String()
	// }
	// return fmt.Sprintf("%s(%s)", self.function, strings.Join(args, ", "))
	return fmt.Sprintf("%s(%d args)", self.function, len(self.args))
}

func NewASTSelection(container ASTExpression, member Token) ASTSelection {
	location := mergeLocations(container, member)
	return ASTSelection{
		astbase: astbase{location},
		container: container,
		member: member.Text()}
}

func (self ASTSelection) Dump(writer io.Writer, indent string) {
	fmt.Fprintf(writer, "%sASTSelection[%s: %s]\n",
		indent, self.container, self.member)
}

func (self ASTSelection) Equal(other_ ASTNode) bool {
	if other, ok := other_.(ASTSelection); ok {
		return self.container.Equal(other.container) &&
			self.member == other.member
	}
	return false
}

func (self ASTSelection) String() string {
	return fmt.Sprintf("%s.%s", self.container, self.member)
}

func NewASTName(tok Token) ASTName {
	return ASTName{
		astbase: astbase{tok.Location()},
		name: tok.Text()}
}

func (self ASTName) Dump(writer io.Writer, indent string) {
	fmt.Fprintf(writer, "%sASTName[%s]\n", indent, self.name)
}

func (self ASTName) Equal(other_ ASTNode) bool {
	if other, ok := other_.(ASTName); ok {
		return self.name == other.name
	}
	return false
}

func (self ASTName) String() string {
	return self.name
}

func NewASTString(value Token) ASTString {
	// strip the quotes: they're preserved by the tokenizer, but not
	// part of the string value (but note that the node location still
	// encompasses the quotes!)
	text := value.Text()
	text = text[1:len(text)-1]
	return ASTString{
		astbase: astbase{value.Location()},
		value: text}
}

func (self ASTString) Dump(writer io.Writer, indent string) {
	fmt.Fprintln(writer,
		indent + "ASTString[" + self.value + "]")
}

func (self ASTString) Equal(other_ ASTNode) bool {
	if other, ok := other_.(ASTString); ok {
		return self.value == other.value
	}
	return false
}

func (self ASTString) String() string {
	// this assumes that Go syntax for strings is Fubsy syntax!
	return fmt.Sprintf("%#v", self.value)
}

func NewASTFileList(
	opener Token,
	patterns []Token,
	closer Token) ASTFileList {
	location := mergeLocations(opener, closer)
	return ASTFileList{
		astbase: astbase{location},
		patterns: extractText(patterns)}
}

func (self ASTFileList) Dump(writer io.Writer, indent string) {
	fmt.Fprintln(writer,
		indent + "ASTFileList[" + strings.Join(self.patterns, " ") + "]")
}

func (self ASTFileList) Equal(other_ ASTNode) bool {
	if other, ok := other_.(ASTFileList); ok {
		return reflect.DeepEqual(self.patterns, other.patterns)
	}
	return false
}

func (self ASTFileList) String() string {
	return "[" + strings.Join(self.patterns, " ") + "]"
}

func extractText(tokens []Token) []string {
	text := make([]string, len(tokens))
	for i, token := range tokens {
		text[i] = token.Text()
	}
	return text
}

func listsEqual(alist []ASTNode, blist []ASTNode) bool {
	//fmt.Printf("comparing node lists:\n  alist=%v\n  blist=%v\n", alist, blist)
	if len(alist) != len(blist) {
		//fmt.Println("list lengths differ")
		return false
	}
	for i, anode := range alist {
		bnode := blist[i]
		if !anode.Equal(bnode) {
			//fmt.Printf("alist[%d] != blist[%d]\n", i, i)
			return false
		}
	}
	//fmt.Println("lists are equal")
	return true
}

func exprlistsEqual(aexprs []ASTExpression, bexprs []ASTExpression) bool {
	// argh, this is stupid: ASTExpression embeds ASTNode, so anything
	// that implements ASTExpression also implements ASTNode, so why
	// can't we treat []ASTExpression more like []ASTNode?
	nodes := func(exprs []ASTExpression) []ASTNode {
		nodes := make([]ASTNode, len(exprs))
		for i, expr := range exprs {
			nodes[i] = ASTNode(expr)
		}
		return nodes
	}

	return listsEqual(nodes(aexprs), nodes(bexprs))
}
