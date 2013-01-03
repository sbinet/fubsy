// Copyright © 2012-2013, Greg Ward. All rights reserved.
// Use of this source code is governed by a BSD-style license that can
// be found in the LICENSE.txt file.

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
// resource that is derived from other resources by executing actions.
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
	"io"
	"reflect"
	"strings"

	"code.google.com/p/go-bit/bit"

	"fubsy/types"
)

type DAG struct {
	// all nodes in the graph
	nodes []Node

	// map node name to index (offset into nodes)
	index map[string]int

	// the parents of every node
	parents []*bit.Set
}

// an opaque set of integer node IDs (this type deliberately has very
// few methods; it just exists so code in the 'runtime' package can
// get node sets out of the DAG to pass back to other DAG methods that
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
		index: make(map[string]int),
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
func (self *DAG) Dump(writer io.Writer, indent string) {
	for id, node := range self.nodes {
		rule := node.BuildRule()
		desc := node.Name()
		detail := node.String()
		if detail != desc {
			desc += " (" + detail + ")"
		}
		fmt.Fprintf(writer, indent+"%04d: %s (%s, state %v)\n",
			id, desc, node.Typename(), node.State())
		if rule != nil {
			fmt.Fprintf(writer, indent+"  action: %s\n", rule.ActionString())
		}
		parents := self.parents[id]
		if !parents.IsEmpty() {
			fmt.Fprintf(writer, indent+"  parents:\n")
			for parentid, ok := parents.Next(-1); ok; parentid, ok = parents.Next(parentid) {
				pnode := self.nodes[parentid]
				fmt.Fprintf(writer, indent+"    %04d: %s\n", parentid, pnode.Name())
			}
		}
	}
}

// Return the set of nodes in this graph with no children.
func (self *DAG) FindFinalTargets() NodeSet {
	//fmt.Println("FindFinalTargets():")
	var targets *bit.Set = bit.New()
	targets.AddRange(0, self.length())
	for _, parents := range self.parents {
		//fmt.Printf("  %d: node=%v, parents=%v\n", id, self.nodes[id], parents)
		targets.SetAndNot(targets, parents)
	}
	//fmt.Printf("  -> targets = %v\n", targets)
	return NodeSet(targets)
}

// Walk the graph starting from each node in goal to find the set of
// original source nodes, i.e. nodes with no parents that are
// ancestors of any node in goal. Store that set (along with some
// other useful information discovered in the graph walk) in self.
func (self *DAG) FindRelevantNodes(goal NodeSet) NodeSet {
	relevant := bit.New()
	self.DFS(goal, func(id int) error {
		relevant.Add(id)
		return nil
	})

	//fmt.Printf("FindRelevantNodes: %v\n", relevant)
	return NodeSet(relevant)
}

// Callback function to visit nodes from DFS(). Return a non-nil error
// to abort the traversal and make DFS() return that error. DFS()
// aborted this way does not report dependency cycles.
type DFSVisitor func(id int) error

// Perform a partial depth-first search of the graph, exploring
// ancestors of all nodes in 'start'. For each node visited, call
// visit() just as we're leaving that node -- i.e. calls to visit()
// are in topological order. visit() can abort the search; see
// DFSVisitor for details.
func (self *DAG) DFS(start NodeSet, visit DFSVisitor) error {
	colour := make([]byte, len(self.nodes))
	path := make([]int, 0)
	cycles := make([][]int, 0)

	var descend func(id int) error
	descend = func(id int) error {
		path = append(path, id)
		//node := self.nodes[id]
		//fmt.Printf("entering node %d: %s (path = %v)\n", id, node, path)
		var err error
		parents := self.parents[id]
		for pid, ok := parents.Next(-1); ok; pid, ok = parents.Next(pid) {
			if err != nil {
				break
			}
			if colour[pid] == GREY {
				cycle := make([]int, len(path)+1)
				copy(cycle, path)
				cycle[len(path)] = pid
				cycles = append(cycles, cycle)
			} else if colour[pid] == WHITE {
				colour[pid] = GREY
				err = descend(pid)
			}
		}
		if err != nil {
			return err
		}
		path = path[0 : len(path)-1]
		err = visit(id)
		if err != nil {
			return err
		}
		colour[id] = BLACK
		return nil
	}

	var err error
	startbs := (*bit.Set)(start)
	for id, ok := startbs.Next(-1); ok; id, ok = startbs.Next(id) {
		if colour[id] == WHITE {
			colour[id] = GREY
			err = descend(id)
		}
	}
	if err != nil {
		return err
	}
	if len(cycles) > 0 {
		return CycleError{self, cycles}
	}
	return nil
}

type CycleError struct {
	dag    *DAG
	cycles [][]int
}

func (self CycleError) Error() string {
	result := []string{
		fmt.Sprintf("found %d dependency cycles:", len(self.cycles))}
	for _, cycle := range self.cycles {
		names := make([]string, len(cycle))
		for i, id := range cycle {
			names[i] = self.dag.nodes[id].Name()
		}
		result = append(result, "  "+strings.Join(names, " -> "))
	}
	return strings.Join(result, "\n")
}

func (self *DAG) NewBuildState(options BuildOptions) *BuildState {
	return &BuildState{dag: self, options: options}
}

// Build a new DAG that is ready to start building targets. The new
// DAG preserves only relevant nodes and expands all expandable nodes
// in the current DAG (e.g each FinderNode is replaced by FileNodes
// for the files matching the glob's patterns). Any BuildState or
// NodeSet objects derived from the old DAG are invalid with the new
// DAG: throw them away and start over again.
func (self *DAG) Rebuild(relevant *bit.Set, ns types.Namespace) (*DAG, []error) {
	var errors []error
	replacements := make(map[int]*bit.Set)
	newdag := NewDAG()
	for id, node := range self.nodes {
		if !relevant.Contains(id) {
			continue
		}
		expansion, err := node.Expand(ns)
		if err != nil {
			errors = append(errors, err)
		} else if expansion == nil {
			panic("node.Expand() returned nil for node " + node.String())
		}
		repl := bit.New()
		for _, expnode := range expansion.List() {
			// if this type assertion panics, then node.Expand()
			// returned FuObjects that don't implement Node
			newid, _ := newdag.addNode(expnode.(Node))
			repl.Add(newid)
		}
		replacements[id] = repl
	}

	// Second loop to rebuild parent info in the new DAG.
	for oldid := range self.nodes {
		newids := replacements[oldid]
		if newids == nil { // node not relevant (not preserved in new DAG)
			continue
		}

		oldparents := self.parents[oldid]
		newparents := bit.New()
		for oldpid, ok := oldparents.Next(-1); ok; oldpid, ok = oldparents.Next(oldpid) {
			newparents.SetOr(newparents, replacements[oldpid])
		}
		for newid, ok := newids.Next(-1); ok; newid, ok = newids.Next(newid) {
			newdag.parents[newid] = newparents
		}
	}

	return newdag, errors
}

// Inspect every node in the graph and figure out which ones are
// original sources (have no parents). Set their state to SOURCE.
func (self *DAG) MarkSources() {
	for id, node := range self.nodes {
		if self.parents[id].IsEmpty() {
			node.SetState(SOURCE)
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
func (self *DAG) AddNode(node Node) Node {
	_, node = self.addNode(node)
	return node
}

func (self *DAG) addNode(node Node) (int, Node) {
	name := node.Name()
	if id, ok := self.index[name]; ok {
		existing := self.nodes[id]
		newtype := reflect.TypeOf(node)
		oldtype := reflect.TypeOf(existing)
		if newtype != oldtype {
			panic(fmt.Sprintf(
				"cannot add node '%s' (type %s): there is already a node "+
					"with that name, but its type is %s",
				name, newtype, oldtype))
		}
		return id, existing
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
	return id, node
}

func (self *DAG) parentNodes(id int) []Node {
	parents := self.parents[id]
	result := make([]Node, 0, parents.Size())
	for parentid, ok := parents.Next(-1); ok; parentid, ok = parents.Next(parentid) {
		result = append(result, self.nodes[parentid])
	}
	return result
}

func (self *DAG) addParent(child Node, parent Node) {
	childid := self.lookupId(child)
	parentid := self.lookupId(parent)
	self.parents[childid].Add(parentid)
}

// Return the number of nodes in the DAG.
func (self *DAG) length() int {
	return len(self.nodes)
}
