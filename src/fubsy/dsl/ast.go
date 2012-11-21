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
	// Return the list of external plugins imported by this script.
	// Does not include inline plugins. Each plugin is represented as
	// a []string, e.g. "import foo.bar.baz" becomes {"foo", "bar",
	// "baz"}.
	ListPlugins() [][]string

	// Return the AST node for the specified phase, or nil if no such
	// phase in this script.
	FindPhase(name string) *ASTPhase
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

// implemented by every AST node via astbase, and also by token
type Locatable interface {
	Location() Location
}

// something with a Location and text (so this file does not know
// about token, which is defined in fugrammar.y -- trying to avoid
// dependency cycles within the package)
type Token interface {
	Locatable
	Text() string
}

type astbase struct {
	location Location
}

func (self astbase) Location() Location {
	return self.location
}

func mergeLocations(loc1 Locatable, loc2 Locatable) Location {
	if loc1 == nil || loc2 == nil {
		// so lazy test code can get away with not creating real
		// Location objects
		return newLocation(nil)
	}
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
	expr ASTExpression
}

// TARGETS : SOURCES { ACTIONS }
// (children is a list of ASTNode, one per action)
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

func (self children) Children() []ASTNode {
	return ([]ASTNode)(self)
}

func NewASTRoot(children []ASTNode) *ASTRoot {
	location := mergeLocations(children[0], children[len(children)-1])
	return &ASTRoot{
		astbase: astbase{location},
		children: children}
}

func (self *ASTRoot) Dump(writer io.Writer, indent string) {
	fmt.Fprintln(writer, indent + "ASTRoot {")
	self.children.Dump(writer, indent)
	fmt.Fprintln(writer, indent + "}")
}

func (self *ASTRoot) Equal(other_ ASTNode) bool {
	if other, ok := other_.(*ASTRoot); ok {
		return other != nil && self.children.Equal(other.children)
	}
	return false
}

func (self *ASTRoot) ListPlugins() [][]string {
	result := make([][]string, 0)
	for _, node_ := range self.children {
		if node, ok := node_.(*ASTImport); ok {
			result = append(result, node.plugin)
		}
	}
	return result
}

func (self *ASTRoot) FindPhase(name string) *ASTPhase {
	for _, node_ := range self.children {
		if node, ok := node_.(*ASTPhase); ok && node.name == name {
			return node
		}
	}
	return nil
}

func NewASTImport(keyword Locatable, names []Token) *ASTImport {
	location := mergeLocations(keyword, names[len(names)-1])
	plugin := extractText(names)
	return &ASTImport{
		astbase: astbase{location},
		plugin: plugin}
}

func (self *ASTImport) Dump(writer io.Writer, indent string) {
	fmt.Fprintf(writer, "%sASTImport[%s]\n", indent, strings.Join(self.plugin, "."))
}

func (self *ASTImport) Equal(other_ ASTNode) bool {
	if other, ok := other_.(*ASTImport); ok {
		return other != nil && reflect.DeepEqual(self.plugin, other.plugin)
	}
	return false
}

func NewASTInline(keyword Locatable, lang Token, content Token) *ASTInline {
	location := mergeLocations(keyword, content)
	return &ASTInline{
		astbase: astbase{location},
		lang: lang.Text(),
		content: content.Text()}
}

func (self *ASTInline) Dump(writer io.Writer, indent string) {
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

func (self *ASTInline) Equal(other_ ASTNode) bool {
	if other, ok := other_.(*ASTInline); ok {
		return other != nil &&
			self.lang == other.lang &&
			self.content == other.content
	}
	return false
}

func NewASTPhase(name Token, block *ASTBlock) *ASTPhase {
	location := mergeLocations(name, block)
	return &ASTPhase{
		astbase: astbase{location},
		name: name.Text(),
		children: block.children}
}

func (self *ASTPhase) Dump(writer io.Writer, indent string) {
	fmt.Fprintf(writer, "%sASTPhase[%s] {\n", indent, self.name)
	self.children.Dump(writer, indent)
	fmt.Fprintln(writer, indent + "}")
}

func (self *ASTPhase) Equal(other_ ASTNode) bool {
	// switch other := other_.(type) {
	// case ASTPhase:
	// 	return self.name == other.name && self.children.Equal(other.children)
	// case *ASTPhase:
	// 	return self.name == other.name && self.children.Equal(other.children)
	// }
	if other, ok := other_.(*ASTPhase); ok {
		return other != nil &&
			self.name == other.name &&
			self.children.Equal(other.children)
	}
	return false
}

func NewASTBlock(opener Token, children []ASTNode, closer Token) *ASTBlock {
	location := mergeLocations(opener, closer)
	return &ASTBlock{
		astbase: astbase{location},
		children: children}
}

func (self *ASTBlock) Dump(writer io.Writer, indent string) {
	panic("Dump() not supported for ASTBlock (transient node type)")
}

func (self *ASTBlock) Equal(other ASTNode) bool {
	panic("Equal() not supported for ASTBlock (transient node type)")
}

func NewASTAssignment(target Token, expr ASTExpression) *ASTAssignment {
	location := mergeLocations(target, expr)
	return &ASTAssignment{
		astbase: astbase{location},
		target: target.Text(),
		expr: expr}
}

func (self *ASTAssignment) Dump(writer io.Writer, indent string) {
	fmt.Fprintf(writer, "%sASTAssignment[%s]\n", indent, self.target)
	self.expr.Dump(writer, indent + "  ")
}

func (self *ASTAssignment) Equal(other_ ASTNode) bool {
	if other, ok := other_.(*ASTAssignment); ok {
		return other != nil &&
			self.target == other.target &&
			self.expr.Equal(other.expr)
	}
	return false
}

// return the name of the variable that is the target of this assignment
func (self *ASTAssignment) Target() string {
	return self.target
}

// return the expression node that is the RHS (right-hand side) of
// this assignment
func (self *ASTAssignment) Expression() ASTExpression {
	return self.expr
}

func NewASTBuildRule(
	targets ASTExpression,
	sources ASTExpression,
	block *ASTBlock) *ASTBuildRule {
	location := mergeLocations(targets, block)
	return &ASTBuildRule{
		astbase: astbase{location},
		targets: targets,
		sources: sources,
		children: block.children}
}

func (self *ASTBuildRule) Dump(writer io.Writer, indent string) {
	fmt.Fprintf(writer, "%sASTBuildRule {\n", indent)
	fmt.Fprintf(writer, "%stargets:\n", indent)
	self.targets.Dump(writer, indent + "  ")
	fmt.Fprintf(writer, "%ssources:\n", indent)
	self.sources.Dump(writer, indent + "  ")
	fmt.Fprintf(writer, "%sactions:\n", indent)
	self.children.Dump(writer, indent)
	fmt.Fprintf(writer, "%s}\n", indent)
}

func (self *ASTBuildRule) Equal(other_ ASTNode) bool {
	if other, ok := other_.(*ASTBuildRule); ok {
		return other != nil &&
			self.targets.Equal(other.targets) &&
			self.sources.Equal(other.sources) &&
			self.children.Equal(other.children)
	}
	return false
}

func (self *ASTBuildRule) Targets() ASTExpression {
	return self.targets
}

func (self *ASTBuildRule) Sources() ASTExpression {
	return self.sources
}

func (self *ASTBuildRule) Actions() []ASTNode {
	return self.children
}

func NewASTAdd(op1 ASTExpression, op2 ASTExpression) *ASTAdd {
	location := mergeLocations(op1, op2)
	return &ASTAdd{
		astbase: astbase{location},
		op1: op1,
		op2: op2}
}

func (self *ASTAdd) Dump(writer io.Writer, indent string) {
	fmt.Fprintf(writer, "%sASTAdd\n", indent)
	fmt.Fprintf(writer, "%sop1:\n", indent)
	self.op1.Dump(writer, indent + "  ")
	fmt.Fprintf(writer, "%sop2:\n", indent)
	self.op2.Dump(writer, indent + "  ")
}

func (self *ASTAdd) Equal(other_ ASTNode) bool {
	if other, ok := other_.(*ASTAdd); ok {
		return other != nil &&
			self.op1.Equal(other.op1) &&
			self.op2.Equal(other.op2)
	}
	return false
}

func (self *ASTAdd) String() string {
	return fmt.Sprintf("%s + %s", self.op1, self.op2)
}

func (self *ASTAdd) Operands() (ASTExpression, ASTExpression) {
	return self.op1, self.op2
}

func NewASTFunctionCall(
	function ASTExpression,
	args []ASTExpression,
	closer Locatable) *ASTFunctionCall {
	location := mergeLocations(function, closer)
	return &ASTFunctionCall{
		astbase: astbase{location},
		function: function,
		args: args}
}

func (self *ASTFunctionCall) Dump(writer io.Writer, indent string) {
	fmt.Fprintf(writer, "%sASTFunctionCall[%s] (%d args)\n",
		indent, self.function, len(self.args))
	for _, arg := range self.args {
		arg.Dump(writer, indent + "  ")
	}
}

func (self *ASTFunctionCall) Equal(other_ ASTNode) bool {
	if other, ok := other_.(*ASTFunctionCall); ok {
		return other != nil &&
			self.function.Equal(other.function) &&
			exprlistsEqual(self.args, other.args)
	}
	return false
}

func (self *ASTFunctionCall) String() string {
	// args := make([]string, len(self.args))
	// for i, arg := range self.args {
	// 	args[i] = arg.String()
	// }
	// return fmt.Sprintf("%s(%s)", self.function, strings.Join(args, ", "))
	return fmt.Sprintf("%s(%d args)", self.function, len(self.args))
}

func NewASTSelection(container ASTExpression, member Token) *ASTSelection {
	location := mergeLocations(container, member)
	return &ASTSelection{
		astbase: astbase{location},
		container: container,
		member: member.Text()}
}

func (self *ASTSelection) Dump(writer io.Writer, indent string) {
	fmt.Fprintf(writer, "%sASTSelection[%s: %s]\n",
		indent, self.container, self.member)
}

func (self *ASTSelection) Equal(other_ ASTNode) bool {
	if other, ok := other_.(*ASTSelection); ok {
		return other != nil &&
			self.container.Equal(other.container) &&
			self.member == other.member
	}
	return false
}

func (self *ASTSelection) String() string {
	return fmt.Sprintf("%s.%s", self.container, self.member)
}

func NewASTName(tok Token) *ASTName {
	return &ASTName{
		astbase: astbase{tok.Location()},
		name: tok.Text()}
}

func (self *ASTName) Dump(writer io.Writer, indent string) {
	fmt.Fprintf(writer, "%sASTName[%s]\n", indent, self.name)
}

func (self *ASTName) Equal(other_ ASTNode) bool {
	if other, ok := other_.(*ASTName); ok {
		return other != nil && self.name == other.name
	}
	return false
}

func (self *ASTName) String() string {
	return self.name
}

func (self *ASTName) Name() string {
	return self.name
}

func NewASTString(value Token) *ASTString {
	// strip the quotes: they're preserved by the tokenizer, but not
	// part of the string value (but note that the node location still
	// encompasses the quotes!)
	text := value.Text()
	text = text[1:len(text)-1]
	return &ASTString{
		astbase: astbase{value.Location()},
		value: text}
}

func (self *ASTString) Dump(writer io.Writer, indent string) {
	fmt.Fprintln(writer,
		indent + "ASTString[" + self.value + "]")
}

func (self *ASTString) Equal(other_ ASTNode) bool {
	if other, ok := other_.(*ASTString); ok {
		return other != nil && self.value == other.value
	}
	return false
}

func (self *ASTString) String() string {
	// this assumes that Go syntax for strings is Fubsy syntax!
	return fmt.Sprintf("%#v", self.value)
}

func (self *ASTString) Value() string {
	return self.value
}

func NewASTFileList(
	opener Token,
	patterns []Token,
	closer Token) *ASTFileList {
	location := mergeLocations(opener, closer)
	return &ASTFileList{
		astbase: astbase{location},
		patterns: extractText(patterns)}
}

func (self *ASTFileList) Dump(writer io.Writer, indent string) {
	fmt.Fprintln(writer,
		indent + "ASTFileList[" + strings.Join(self.patterns, " ") + "]")
}

func (self *ASTFileList) Equal(other_ ASTNode) bool {
	if other, ok := other_.(*ASTFileList); ok {
		return other != nil && reflect.DeepEqual(self.patterns, other.patterns)
	}
	return false
}

func (self *ASTFileList) String() string {
	return "[" + strings.Join(self.patterns, " ") + "]"
}

func (self *ASTFileList) Patterns() []string {
	return self.patterns
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
