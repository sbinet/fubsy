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
	"fmt"
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
