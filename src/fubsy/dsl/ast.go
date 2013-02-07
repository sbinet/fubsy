// Copyright © 2012-2013, Greg Ward. All rights reserved.
// Use of this source code is governed by a BSD-style license that can
// be found in the LICENSE.txt file.

package dsl

import (
	"fmt"
	"io"
	"reflect"
	"strings"
)

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
	// originated.)
	Equal(other ASTNode) bool
}

// implemented by all AST nodes produced by a grammar rule 'expr : ...',
// i.e. anything that has a value
type ASTExpression interface {
	ASTNode
	fmt.Stringer
}

// describe the physical location (e.g. filename and line number(s))
// of a piece of text, for use in error reporting
type Location interface {
	String() string
	ErrorPrefix() string
	merge(other Location) Location
}

// something with a location: every token and AST node implements this
type Locatable interface {
	Location() Location
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
		return newFileLocation(nil)
	}
	return loc1.Location().merge(loc2.Location())
}

// AST nodes with variable number of children
type children []ASTNode

type ASTRoot struct {
	astbase
	children
	eof Locatable
}

// import a single plugin, e.g. "import NAME"
type ASTImport struct {
	astbase
	plugin []string // fully-qualified name split on '.'
}

// an inline plugin, e.g. "plugin LANG {{{ CONTENT }}}"
type ASTInline struct {
	astbase
	lang    string
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
	expr   ASTExpression
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

// [expr, expr, ...]
type ASTList struct {
	astbase
	elements []ASTExpression
}

// FUNC(arg, arg, ...) (N.B. FUNC is an expr to allow for code like
// "(a.b.c())(stuff))"
type ASTFunctionCall struct {
	astbase
	function ASTExpression
	args     []ASTExpression
}

// member selection: CONTAINER.NAME where CONTAINER is any expr
type ASTSelection struct {
	astbase
	container ASTExpression
	member    string
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
type ASTFileFinder struct {
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
	var location Location
	if len(children) > 0 {
		location = mergeLocations(children[0], children[len(children)-1])
	}
	return &ASTRoot{
		astbase:  astbase{location},
		children: children}
}

func (self *ASTRoot) Dump(writer io.Writer, indent string) {
	fmt.Fprintln(writer, indent+"ASTRoot {")
	self.children.Dump(writer, indent)
	fmt.Fprintln(writer, indent+"}")
}

func (self *ASTRoot) Equal(other_ ASTNode) bool {
	if other, ok := other_.(*ASTRoot); ok {
		return other != nil && self.children.Equal(other.children)
	}
	return false
}

func (self *ASTRoot) EOF() Locatable {
	return self.eof
}

func (self *ASTRoot) FindImports() [][]string {
	result := make([][]string, 0)
	for _, node_ := range self.children {
		if node, ok := node_.(*ASTImport); ok {
			result = append(result, node.plugin)
		}
	}
	return result
}

func (self *ASTRoot) FindInlinePlugins() []*ASTInline {
	var result []*ASTInline
	for _, node := range self.children {
		if node, ok := node.(*ASTInline); ok {
			result = append(result, node)
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

func NewASTImport(dottedname []string, location ...Locatable) *ASTImport {
	return &ASTImport{
		astbase: astLocation(location),
		plugin:  dottedname}
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

func NewASTInline(lang string, content string, location ...Locatable) *ASTInline {
	return &ASTInline{
		astbase: astLocation(location),
		lang:    lang,
		content: content}
}

func (self *ASTInline) Dump(writer io.Writer, indent string) {
	fmt.Fprintf(writer, "%sASTInline[%s] {{{", indent, self.lang)
	if len(self.content) > 0 {
		replace := -1 // indent all lines by default
		if strings.HasSuffix(self.content, "\n") {
			// last line doesn't really exist, so don't indent it
			replace = strings.Count(self.content, "\n") - 1
		}
		content := strings.Replace(
			self.content, "\n", "\n"+indent+"  ", replace)
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

func (self *ASTInline) Language() string {
	return self.lang
}

func (self *ASTInline) Content() string {
	return self.content
}

func NewASTPhase(name string, block *ASTBlock, location ...Locatable) *ASTPhase {
	return &ASTPhase{
		astbase:  astLocation(location),
		name:     name,
		children: block.children}
}

func (self *ASTPhase) Dump(writer io.Writer, indent string) {
	fmt.Fprintf(writer, "%sASTPhase[%s] {\n", indent, self.name)
	self.children.Dump(writer, indent)
	fmt.Fprintln(writer, indent+"}")
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

func NewASTBlock(children []ASTNode, location ...Locatable) *ASTBlock {
	return &ASTBlock{
		astbase:  astLocation(location),
		children: children}
}

func (self *ASTBlock) Dump(writer io.Writer, indent string) {
	panic("Dump() not supported for ASTBlock (transient node type)")
}

func (self *ASTBlock) Equal(other ASTNode) bool {
	panic("Equal() not supported for ASTBlock (transient node type)")
}

func NewASTAssignment(name string, expr ASTExpression, location ...Locatable) *ASTAssignment {
	return &ASTAssignment{
		astbase: astLocation(location),
		target:  name,
		expr:    expr}
}

func (self *ASTAssignment) Dump(writer io.Writer, indent string) {
	fmt.Fprintf(writer, "%sASTAssignment[%s]\n", indent, self.target)
	self.expr.Dump(writer, indent+"  ")
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
		astbase:  astbase{location},
		targets:  targets,
		sources:  sources,
		children: block.children}
}

func (self *ASTBuildRule) Dump(writer io.Writer, indent string) {
	fmt.Fprintf(writer, "%sASTBuildRule {\n", indent)
	fmt.Fprintf(writer, "%stargets:\n", indent)
	self.targets.Dump(writer, indent+"  ")
	fmt.Fprintf(writer, "%ssources:\n", indent)
	self.sources.Dump(writer, indent+"  ")
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
		op1:     op1,
		op2:     op2}
}

func (self *ASTAdd) Dump(writer io.Writer, indent string) {
	fmt.Fprintf(writer, "%sASTAdd\n", indent)
	fmt.Fprintf(writer, "%sop1:\n", indent)
	self.op1.Dump(writer, indent+"  ")
	fmt.Fprintf(writer, "%sop2:\n", indent)
	self.op2.Dump(writer, indent+"  ")
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

func NewASTList(elements []ASTExpression, location ...Locatable) *ASTList {
	result := &ASTList{
		astbase:  astLocation(location),
		elements: elements,
	}
	return result
}

func (self *ASTList) Dump(writer io.Writer, indent string) {
	fmt.Fprintf(writer, "%sASTList (%d elements)\n", indent, len(self.elements))
	for _, elem := range self.elements {
		elem.Dump(writer, indent+"  ")
	}
}

func (self *ASTList) Equal(other_ ASTNode) bool {
	other, ok := other_.(*ASTList)
	return ok && exprlistsEqual(self.elements, other.elements)
}

func (self *ASTList) String() string {
	elements := toStrings(self.elements)
	return "[" + strings.Join(elements, ", ") + "]"
}

func NewASTFunctionCall(
	function ASTExpression,
	args []ASTExpression,
	location ...Locatable) *ASTFunctionCall {
	return &ASTFunctionCall{
		astbase:  astLocation(location),
		function: function,
		args:     args}
}

func (self *ASTFunctionCall) Dump(writer io.Writer, indent string) {
	fmt.Fprintf(writer, "%sASTFunctionCall[%s] (%d args)\n",
		indent, self.function, len(self.args))
	for _, arg := range self.args {
		arg.Dump(writer, indent+"  ")
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
	args := toStrings(self.args)
	return fmt.Sprintf("%s(%s)", self.function, strings.Join(args, ", "))
}

func (self *ASTFunctionCall) Function() ASTExpression {
	return self.function
}

func (self *ASTFunctionCall) Args() []ASTExpression {
	return self.args
}

func NewASTSelection(container ASTExpression, member string, location ...Locatable) *ASTSelection {
	return &ASTSelection{
		astbase:   astLocation(location),
		container: container,
		member:    member}
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

func (self *ASTSelection) Container() ASTExpression {
	return self.container
}

func (self *ASTSelection) Name() string {
	return self.member
}

func NewASTName(name string, location ...Locatable) *ASTName {
	return &ASTName{
		astbase: astLocation(location),
		name:    name}
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

func NewASTString(toktext string, location ...Locatable) *ASTString {
	// strip the quotes: they're preserved by the tokenizer, but not
	// part of the string value (but note that the node location still
	// encompasses the quotes!)
	value := toktext[1 : len(toktext)-1]
	return &ASTString{
		astbase: astLocation(location),
		value:   value}
}

func astLocation(locations []Locatable) astbase {
	switch len(locations) {
	case 0:
		return astbase{FileLocation{}}
	case 1:
		return astbase{locations[0].Location()}
	case 2:
		//return astbase{locations[0].merge(locations[1])}
		return astbase{mergeLocations(locations[0], locations[1])}
	default:
		panic("too many Locations passed to AST constructor (max 2)")
	}
	panic("unreachable code")
}

func (self *ASTString) Dump(writer io.Writer, indent string) {
	fmt.Fprintln(writer,
		indent+"ASTString["+self.value+"]")
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

func NewASTFileFinder(patterns []string, location ...Locatable) *ASTFileFinder {
	return &ASTFileFinder{
		astbase:  astLocation(location),
		patterns: patterns}
}

func (self *ASTFileFinder) Dump(writer io.Writer, indent string) {
	fmt.Fprintln(writer,
		indent+"ASTFileFinder["+strings.Join(self.patterns, " ")+"]")
}

func (self *ASTFileFinder) Equal(other_ ASTNode) bool {
	if other, ok := other_.(*ASTFileFinder); ok {
		return other != nil && reflect.DeepEqual(self.patterns, other.patterns)
	}
	return false
}

func (self *ASTFileFinder) String() string {
	return "[" + strings.Join(self.patterns, " ") + "]"
}

func (self *ASTFileFinder) Patterns() []string {
	return self.patterns
}

func toStrings(expressions []ASTExpression) []string {
	result := make([]string, len(expressions))
	for i, expr := range expressions {
		result[i] = expr.String()
	}
	return result
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

// for tests in other packages
type StubLocation struct {
	message string
}

func NewStubLocation(message string) StubLocation {
	return StubLocation{message}
}

func (self StubLocation) Location() Location {
	return self
}

func (self StubLocation) String() string {
	return self.message
}

func (self StubLocation) ErrorPrefix() string {
	return self.message + ": "
}

func (self StubLocation) merge(other Location) Location {
	return NewStubLocation(self.message + other.(StubLocation).message)
}

type StubLocatable struct {
	location Location
}

func NewStubLocatable(location Location) StubLocatable {
	return StubLocatable{location: location}
}

func (self StubLocatable) Location() Location {
	return self.location
}
