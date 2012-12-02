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
	"reflect"
	"code.google.com/p/go-bit/bit"
)

type DAG struct {
	// all nodes in the graph
	nodes []Node

	// map node name to index (offset into nodes)
	index map[string] int

	// the parents of every node
	parents []*bit.Set
}

// an opaque set of integer node IDs (this type deliberately has no
// methods; it just exists so code in the 'runtime' package can get
// node sets out of the DAG to pass back to other DAG methods that
// then do further processing)
type NodeSet *bit.Set

// graph-walking state: a white node is one we haven't visited yet,
// grey is one we're currently processing, and black is one we're done
// with
const (
	WHITE = iota
	GREY
	BLACK
)

func NewDAG() *DAG {
	return &DAG{
		nodes: make([]Node, 0),
		index: make(map[string] int),
	}
}

// Add the same set of parents (source nodes) to many children (target
// nodes).
func (self *DAG) AddManyParents(targets, sources []Node) {
	sourceset := bit.New()
	for _, snode := range sources {
		sourceset.Add(self.lookupId(snode))
	}

	for _, tnode := range targets {
		tid := self.lookupId(tnode)
		self.parents[tid].SetOr(self.parents[tid], sourceset)
	}
}

// Write a compact, human-readable representation of the entire DAG to
// writer. This faithfully represents the data structure as it exists
// in memory; it doesn't try to make a fancy recursive tree-like
// structure.
func (self *DAG) Dump(writer io.Writer) {
	for id, node := range self.nodes {
		if node == nil {
			continue
		}
		action := node.Action()
		desc := node.Name()
		detail := node.String()
		if detail != desc {
			desc += " (" + detail + ")"
		}
		fmt.Fprintf(writer, "%04d: %s\n", id, desc)
		if action != nil {
			fmt.Fprintf(writer, "  action: %v\n", action)
		}
		parents := self.parents[id]
		if !parents.IsEmpty() {
			fmt.Fprintf(writer, "  parents:\n")
			parents.Do(func (parentid int) {
				pnode := self.nodes[parentid]
				fmt.Fprintf(writer, "    %04d: %s\n", parentid, pnode.Name())
			})
		}
	}
}

// Return the set of nodes in this graph with no children.
func (self *DAG) FindFinalTargets() NodeSet {
	fmt.Println("FindFinalTargets():")
	var targets *bit.Set = bit.New()
	targets.AddRange(0, self.length())
	for id, parents := range self.parents {
		//fmt.Printf("  %d: node=%v, parents=%v\n", id, self.nodes[id], parents)
		if parents == nil {
			targets.Remove(id)
		} else {
			targets.SetAndNot(targets, parents)
		}
	}
	fmt.Printf("  -> targets = %v\n", targets)
	return NodeSet(targets)
}

func (self *DAG) NewBuildState() *BuildState {
	return &BuildState{dag: self}
}

// Iterate over the graph, expanding all relevant expandable nodes.
// That is, for each expandable node that is a member of relevant, add
// zero or more nodes that represent the same resource(s), but more
// concretely. The original node is typically removed. The canonical
// use case for this is that each GlobNode is replaced by FileNodes
// for the files matching the glob's patterns.
func (self *DAG) Expand(relevant *bit.Set) []error {
	var errors []error
	var err error
	for id, node := range self.nodes {
		if node != nil && relevant.Contains(id) {
			err = node.Expand(self)
			if err != nil {
				errors = append(errors, err)
			}
		}
	}
	return errors
}

// Return the node with the specified name, or nil if no such node.
func (self *DAG) lookup(name string) Node {
	if idx, ok := self.index[name]; ok {
		return self.nodes[idx]
	}
	return nil
}

// Return the ID of node, or -1 if node is not in the DAG.
func (self *DAG) lookupId(node Node) int {
	if idx, ok := self.index[node.Name()]; ok {
		return idx
	}
	return -1
}

// Either add node to the DAG, or ensure that another node just like
// it is already there. Specifically: if there is already a node with
// the same name and type as node, do nothing; if there is no node
// with the same name, add node; if there is a same-named node but it
// has different type, panic. (Thus, while the static return type of
// this method is Node, the runtime type of the return value is
// guaranteed to be the same runtime type as the node passed in.)
func (self *DAG) addNode(node Node) Node {
	name := node.Name()
	if id, ok := self.index[name]; ok {
		existing := self.nodes[id]
		newtype := reflect.TypeOf(node)
		oldtype := reflect.TypeOf(existing)
		if newtype != oldtype {
			panic(fmt.Sprintf(
				"cannot add node '%s' (type %s): there is already a node " +
				"with that name, but its type is %s",
				name, newtype, oldtype))
		}
		return existing
	}
	if len(self.nodes) != len(self.parents) {
		panic(fmt.Sprintf(
			"inconsistent DAG: len(nodes) = %d, len(parents) = %d",
			len(self.nodes), len(self.parents)))
	}

	id := len(self.nodes)
	self.nodes = append(self.nodes, node)
	self.parents = append(self.parents, bit.New())
	self.index[name] = id
	return node
}

func (self *DAG) parentNodes(node Node) []Node {
	id := self.lookupId(node)
	result := make([]Node, 0)
	self.parents[id].Do(func (parentid int) {
		pnode := self.nodes[parentid]
		if pnode == nil {
			// parents were not correctly adjusted by replaceNode()
			panic(fmt.Sprintf(
				"dag.parents[%d] includes nil node pointer (%d)", id, parentid))
		}
		result = append(result, pnode)
	})
	return result
}

func (self *DAG) addParent(child Node, parent Node) {
	childid := self.lookupId(child)
	parentid := self.lookupId(parent)
	self.parents[childid].Add(parentid)
}

// Remove the specified node from the DAG, updating references to it
// with replacements. Panic if it's not in the DAG.
func (self *DAG) replaceNode(remove Node, replacements []Node) {
	removeid := self.lookupId(remove)
	name := remove.Name()
	match := self.nodes[removeid]
	if match != remove {
		panic(fmt.Sprintf(
			"cannot remove node %v from the DAG (slot %d is used by %v)",
			remove, removeid, match))
	}

	// replace any occurences of node in any other node's parent set
	// with replacements
	replacementset := bit.New()
	for _, node := range replacements {
		replacementset.Add(self.lookupId(node))
	}
	for _, parents := range self.parents {
		if parents == nil {
			continue
		}
		if parents.Contains(removeid) {
			parents.Remove(removeid)
			parents.SetOr(parents, replacementset)
		}
	}

	// and remove node from the DAG (leaving a hole)
	self.nodes[removeid] = nil
	self.parents[removeid] = nil
	delete(self.index, name)
}

// Return the number of nodes in the DAG.
func (self *DAG) length() int {
	return len(self.nodes)
}
