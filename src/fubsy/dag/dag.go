package dag

// Fubsy's DAG (directed acyclic graph) of dependencies. In theory,
// nodes in the graph represent abstract resources; in pratice, each
// node is typically one file: maybe a source file, maybe a target
// file built from source files.
//
// A node's *parents* are the nodes that it depends on, i.e. from
// which it is built. A node with no parents is an original source
// node, most commonly a source file kept under version control and
// directly edited by developers. A node with parents is a target
// node. It might be generated source code (e.g. C from a yacc
// grammar), binary machine code (.o from .c), bytecode (.pyc from
// .py, .class from .java), an archive (.jar, lib*.a), or in fact any
// resource that is derived from other resourcs by executing actions.
//
// A node's *children* are the nodes built from it. A node with no
// children is called a final target.
//
// From the above definitions, it's perfectly sensible for a node to
// be both a source and a target node. E.g. foo.o might be a target
// node derived from foo.c and foo.h, and a source node for libstuff.a
// and libstuff.so.
//
// Just to clarify: the direct relationships between nodes are always
// stated as "parent" and "child": foo.o is a child of foo.c and a
// parent of libstuff.a. Indirect relationships are described as
// "ancestor" and "descendant": foo.c is an ancestor of libstuff.a,
// and libstuff.so is a descendant of foo.h.

import (
	"io"
	"fmt"
	"code.google.com/p/go-bit/bit"
)

// This interface must not betray anything about the filesystem,
// otherwise we'll have no hope of making non-file nodes. (They're not
// needed often, but when you need them you *really* need them.)
type Node interface {
	// human-readable string representation of this node (must be
	// unique in this graph)
	Name() string

	// unique integer identifier for this node
	Id() int

	// return true if this node and other describe the same resource
	// (should be sufficient to compare names)
	Equal(other Node) bool

	// return the parent nodes that this node depends on
	Parents() []Node

	// add node to this node's parent list (do nothing if it's already there)
	AddParent(node Node)

	// return the child nodes that depend on this node
	//Children() []Node

	// Set the action that must be executed to build this node from
	// its parents. (This is a single Action because actions can be
	// compound: in particular, SequenceAction is an implementation of
	// Action that is just a sequence of other Actions.)
	SetAction(action Action)

	// Return the action previously passed to SetAction() (nil if no
	// action has ever been set, which implies that this is an
	// original source node).
	Action() Action
}

type DAG struct {
	// all nodes in the graph
	nodes []Node

	// map node name to index (offset into nodes)
	index map[string] int
}

func NewDAG() *DAG {
	return &DAG{
		nodes: make([]Node, 0),
		index: make(map[string] int),
	}
}

// Add the same set of parents (source nodes) to many children (target
// nodes).
func (self *DAG) AddManyParents(targets, sources []Node) {
	// This could be optimized to take advantage of bitsets: collapse
	// sources to a bitmask and OR that onto the parent set of each
	// target.
	for _, target := range targets {
		for _, source := range sources {
			target.AddParent(source)
		}
	}
}

// Write a compact, human-readable representation of the entire DAG to
// writer. This faithfully represents the data structure as it exists
// in memory; it doesn't try to make a fancy recursive tree-like
// structure.
func (self *DAG) Dump(writer io.Writer) {
	for i, node := range self.nodes {
		action := node.Action()
		parents := node.Parents()
		fmt.Fprintf(writer, "%04d: %s\n", i, node.Name())
		if action != nil {
			fmt.Fprintf(writer, "  action: %v\n", action)
		}
		if len(parents) > 0 {
			fmt.Fprintf(writer, "  parents:\n")
			for _, pnode := range parents {
				fmt.Fprintf(writer, "    %04d: %s\n", pnode.Id(), pnode.Name())
			}
		}
	}
}

// Return the node with the specified name, or nil if no such node.
func (self *DAG) lookup(name string) Node {
	if idx, ok := self.index[name]; ok {
		return self.nodes[idx]
	}
	return nil
}

// Add the specified node to the DAG. Return the node ID. Panic if a
// node with the same name already exists.
func (self *DAG) addNode(node Node) int {
	name := node.Name()
	if _, ok := self.index[name]; ok {
		panic(fmt.Sprintf("node with name '%s' already exists", name))
	}
	if node.Id() != -1 {
		panic(fmt.Sprintf("node '%s' has id %d: is it already in the DAG?",
			name, node.Id()))
	}
	id := len(self.nodes)
	self.nodes = append(self.nodes, node)
	self.index[name] = id
	return id
}

// Return the number of nodes in the DAG.
func (self *DAG) length() int {
	return len(self.nodes)
}

// Convenient base type for Node implementations -- provides the
// basics right out of the box.

type nodebase struct {
	dag *DAG
	id int
	name string
	parentset bit.Set
	action Action
}

func makenodebase(dag *DAG, id int, name string) nodebase {
	return nodebase{
		dag: dag,
		id: id,
		name: name,
	}
}

func (base *nodebase) Id() int {
	return base.id
}

func (base *nodebase) Name() string {
	return base.name
}

func (base *nodebase) Parents() []Node {
	result := make([]Node, 0)
	fetch := func(id int) {
		result = append(result, base.dag.nodes[id])
	}
	base.parentset.Do(fetch)
	return result
}

func (base *nodebase) AddParent (node Node) {
	id := node.Id()
	if id < 0 || id >= base.dag.length() {
		panic(fmt.Sprintf(
			"%v has impossible id %d (should be >= 0 && <= %d)",
			node, id, base.dag.length() - 1))
	}
	if base.parentset.Contains(id) {
		return
	}
	base.parentset.Add(id)
}

func (base *nodebase) SetAction (action Action) {
	base.action = action
}

func (base *nodebase) Action() Action {
	return base.action
}
