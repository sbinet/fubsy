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

type DAG struct {
	// all nodes in the graph
	nodes []Node

	// map node name to index (offset into nodes)
	index map[string] int
}

type BuildState struct {
	// establishes the meaning of the integer node IDs in all the
	// other fields
	dag *DAG

	// the target nodes requested by the user (default: all final targets)
	goal *bit.Set

	// the original sources for goal, i.e. the set of ancestors of
	// goal that have no parents
	sources *bit.Set

	// the set of all ancestors of goal (relevant contains both goal
	// and source)
	relevant *bit.Set

	// the set of all nodes that need to be rebuilt (changes as a
	// build proceeds)
	rebuild *bit.Set

	// the relevant children of all relevant nodes
	children map[int] *bit.Set
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
		desc := node.Name()
		detail := node.String()
		if detail != desc {
			desc += " (" + detail + ")"
		}
		fmt.Fprintf(writer, "%04d: %s\n", i, desc)
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

// Return the set of nodes in this graph with no children.
func (self *DAG) FindFinalTargets() NodeSet {
	var targets *bit.Set = bit.New()
	targets.AddRange(0, self.length())
	for _, node := range self.nodes {
		parents := (*bit.Set)(node.ParentSet())
		targets.SetAndNot(targets, parents)
	}
	return NodeSet(targets)
}

func (self *DAG) NewBuildState() *BuildState {
	return &BuildState{dag: self}
}

func (self *BuildState) SetGoal(goal NodeSet) {
	self.goal = (*bit.Set)(goal)
}

// Walk the graph starting from each node in goal to find the set of
// original source nodes, i.e. nodes with no parents that are
// ancestors of any node in goal. Store that set (along with some
// other useful information discovered in the graph walk) in self.
func (self *BuildState) FindOriginalSources() {
	nodes := self.dag.nodes
	colour := make([]byte, len(nodes))

	self.relevant = bit.New()
	self.sources = bit.New()
	self.children = make(map[int] *bit.Set)

	var visit func(id int)
	visit = func(id int) {
		node := nodes[id]
		//fmt.Printf("visiting node %d (%s)\n", id, node)
		self.children[id] = bit.New()
		parents := (*bit.Set)(node.ParentSet())
		parents.Do(func(parent int) {
			if colour[parent] == GREY {
				// we can do a better job of reporting this!
				panic(fmt.Sprintf("dependency cycle! (..., %s, %s)",
					node, nodes[parent]))
			}
			if colour[parent] == WHITE {
				colour[parent] = GREY
				visit(parent)
			}
			self.children[parent].Add(id)
		})
		self.relevant.Add(id)
		if parents.IsEmpty() {
			self.sources.Add(id)
		}
		colour[id] = BLACK
	}

	self.goal.Do(func(id int) {
		if colour[id] == WHITE {
			colour[id] = GREY
			visit(id)
		}
	})
}

// Compute the initial rebuild set, i.e. nodes that are 1) children of
// the original sources, 2) relevant (ancestors of a goal node), and
// 3) stale.
func (self *BuildState) FindStaleTargets() []error {
	errors := make([]error, 0)
	self.rebuild = bit.New()
	self.sources.Do(func (id int) {
		node := self.dag.nodes[id]
		changed, err := node.Changed()
		if err != nil {
			errors = append(errors, err)
		} else if changed  {
			self.children[id].Do(func (child int) {
				self.rebuild.Add(child)
				self.dag.nodes[child].SetState(STALE)
			})
		}
	})
	return errors
}

func (self *BuildState) BuildStaleTargets() []error {
	return nil
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
