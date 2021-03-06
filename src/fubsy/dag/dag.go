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
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"code.google.com/p/go-bit/bit"

	//"fubsy/log"
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
type NodeSet bit.Set

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

// Return a slice of all the nodes in this graph. Do not mutate the
// returned slice: it might be shared with the DAG (or it might not).
// It's OK to modify the Node objects in the slice, though; that is
// guaranteed to affect the Nodes in the DAG.
func (self *DAG) Nodes() []Node {
	return self.nodes
}

// Return a deep copy of self, i.e. all nodes in the returned DAG are
// copies of nodes in self. Mainly for test code.
func (self *DAG) copy() *DAG {
	newdag := DAG{
		nodes:   make([]Node, len(self.nodes)),
		index:   make(map[string]int),
		parents: make([]*bit.Set, len(self.parents)),
	}
	for id, node := range self.nodes {
		newdag.nodes[id] = node.copy()
	}
	for name, id := range self.index {
		newdag.index[name] = id
	}
	for id, parentset := range self.parents {
		var copyset bit.Set
		copyset = *parentset
		newdag.parents[id] = &copyset
	}
	return &newdag
}

// Examine the DAG and panic if there are any inconsistencies. Use
// liberally in test code. Should not be necessary to use in
// production code!
func (self *DAG) verify() {
	if len(self.nodes) != len(self.index) {
		panic(fmt.Sprintf(
			"corrupt DAG: self.nodes has %d nodes, but self.index has %d",
			len(self.nodes), len(self.index)))
	}
	if len(self.nodes) != len(self.parents) {
		panic(fmt.Sprintf(
			"corrupt DAG: self.nodes has %d nodes, but self.parents has %d",
			len(self.nodes), len(self.parents)))
	}

	for id, node := range self.nodes {
		if id != node.id() {
			panic(fmt.Sprintf(
				"corrupt DAG: node %s thinks it has id %d, but is in slot %d",
				node, node.id(), id))
		}
	}
	for name, id := range self.index {
		node := self.nodes[id]
		if name != node.Name() {
			panic(fmt.Sprintf(
				"corrupt DAG: index says node %s has id %d, "+
					"but slot %d has node %s",
				name, id, node.Name()))
		}
	}
	maxid := len(self.nodes) - 1
	for id, parentset := range self.parents {
		if !parentset.IsEmpty() && parentset.Max() > maxid {
			panic(fmt.Sprintf(
				"corrupt DAG: max parent of node %d is %d (should be <= %d)",
				id, parentset.Max(), maxid))
		}
	}
}

// Return a NodeSet containing all the nodes passed by name. Panic if
// any of the names do not exist in this DAG. Mainly for test code.
func (self *DAG) MakeNodeSet(names ...string) *NodeSet {
	result := bit.New()
	for _, name := range names {
		id, ok := self.index[name]
		if !ok {
			panic("no such node: " + name)
		}
		result.Add(id)
	}
	return (*NodeSet)(result)
}

// Add the same set of parents (source nodes) to many children (target
// nodes).
func (self *DAG) AddManyParents(targets, sources []Node) {
	sourceset := bit.New()
	for _, snode := range sources {
		sourceset.Add(snode.id())
	}

	for _, tnode := range targets {
		tid := tnode.id()
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
		fmt.Fprintf(writer, indent+"%04d: %s %s (state %v)\n",
			id, node.Typename(), node, node.State())
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

// Run NodeExpand() on every node in the graph.
func (self *DAG) ExpandNodes(ns types.Namespace) []error {
	var errs []error
	newindex := make(map[string]int)
	for id, node := range self.nodes {
		err := node.NodeExpand(ns)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		newindex[node.Name()] = id
	}
	if len(errs) > 0 {
		return errs
	}
	self.index = newindex
	return nil
}

// Return the set of nodes in this graph that match the names in
// targets. If targets is nil or empty, return all final targets
// (nodes with no children). Otherwise, for each name in targets, find
// the node with that name. If no such node, find all nodes whose
// names start with that name (in the pathname sense: "foo/bar" starts
// with "foo", but not with "f"). Each target name with no matches
// will result in one error object being appended to the returned
// slice of errors.
func (self *DAG) MatchTargets(targets []string) (*NodeSet, []error) {
	if len(targets) == 0 {
		return self.FindFinalTargets(), nil
	}
	result := bit.New()
	errs := []error{}
	for _, name := range targets {
		name = filepath.Clean(name)
		id, ok := self.index[name]
		if ok {
			if self.parents[id].IsEmpty() {
				errs = append(errs,
					fmt.Errorf("not a target: '%s'", self.nodes[id].Name()))
			} else {
				result.Add(id)
			}
		} else {
			var prefixmatch *bit.Set
			prefixmatch, errs = self.matchPrefix(name, errs)
			result.SetOr(result, prefixmatch)
		}
	}
	return (*NodeSet)(result), errs
}

func (self *DAG) matchPrefix(name string, errs []error) (*bit.Set, []error) {
	prefix := string(append(([]byte)(name), filepath.Separator))
	result := bit.New()
	for _, trynode := range self.nodes {
		if self.HasParents(trynode) &&
			strings.HasPrefix(trynode.Name(), prefix) {
			result.Add(trynode.id())
		}
	}
	if result.IsEmpty() {
		errs = append(errs,
			fmt.Errorf("no targets found matching '%s'", name))
	}
	return result, errs
}

// Return the set of nodes in this graph with no children.
func (self *DAG) FindFinalTargets() *NodeSet {
	//fmt.Println("FindFinalTargets():")
	var targets *bit.Set = bit.New()
	targets.AddRange(0, self.length())
	for _, parents := range self.parents {
		//fmt.Printf("  %d: node=%v, parents=%v\n", id, self.nodes[id], parents)
		targets.SetAndNot(targets, parents)
	}
	//fmt.Printf("  -> targets = %v\n", targets)
	return (*NodeSet)(targets)
}

// Callback function to visit nodes from DFS(). Return a non-nil error
// to abort the traversal and make DFS() return that error. DFS()
// aborted this way does not report dependency cycles.
type DFSVisitor func(node Node) error

// Perform a partial depth-first search of the graph, exploring
// ancestors of all nodes in 'start'. For each node visited, call
// visit() just as we're leaving that node -- i.e. calls to visit()
// are in topological order. visit() can abort the search; see
// DFSVisitor for details.
func (self *DAG) DFS(start *NodeSet, visit DFSVisitor) error {
	colour := make([]byte, len(self.nodes))
	path := make([]int, 0)
	cycles := make([][]int, 0)

	var descend func(id int) error
	descend = func(id int) error {
		path = append(path, id)
		//node := self.nodes[id]
		//indent := fmt.Sprintf("%*s", len(path), "")
		//fmt.Printf("%sentering node %d: %s (path = %v)\n", indent, id, node, path)
		var err error
		parents := self.parents[id]
		for pid, ok := parents.Next(-1); ok; pid, ok = parents.Next(pid) {
			//fmt.Printf("%snode %d child: %d (color = %v)\n", indent, id, pid, colour[pid])
			if colour[pid] == GREY {
				cycle := make([]int, len(path)+1)
				copy(cycle, path)
				cycle[len(path)] = pid
				cycles = append(cycles, cycle)
			} else if colour[pid] == WHITE {
				colour[pid] = GREY
				err = descend(pid)
			}
			if err != nil {
				return err
			}
		}
		path = path[0 : len(path)-1]
		//fmt.Printf("%snode %d: visit(%s)\n", indent, id, self.nodes[id])
		err = visit(self.nodes[id])
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
			if err != nil {
				return err
			}
		}
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
func (self *DAG) Lookup(name string) Node {
	if idx, ok := self.index[name]; ok {
		return self.nodes[idx]
	}
	return nil
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

	if node.id() != -1 {
		panic(fmt.Sprintf(
			"cannot add node %s to the DAG: it already has id %d",
			node, node.id()))
	}
	id := len(self.nodes)
	node.setid(id)
	self.nodes = append(self.nodes, node)
	self.parents = append(self.parents, bit.New())
	self.index[name] = id
	return id, node
}

func (self *DAG) HasParents(node Node) bool {
	return !self.parents[node.id()].IsEmpty()
}

func (self *DAG) ParentNodes(node Node) []Node {
	parents := self.parents[node.id()]
	result := make([]Node, 0, parents.Size())
	for parentid, ok := parents.Next(-1); ok; parentid, ok = parents.Next(parentid) {
		result = append(result, self.nodes[parentid])
	}
	return result
}

func (self *DAG) AddParent(child Node, parent Node) {
	self.parents[child.id()].Add(parent.id())
}

// Return the number of nodes in the DAG.
func (self *DAG) length() int {
	return len(self.nodes)
}

// test-friendly way to format a NodeSet as a string
func (self *NodeSet) String() string {
	set := (*bit.Set)(self)
	result := make([]byte, 1, set.Size()*3)
	result[0] = '{'
	for n, ok := set.Next(-1); ok; n, ok = set.Next(n) {
		result = strconv.AppendInt(result, int64(n), 10)
		result = append(result, ',')
	}
	result[len(result)-1] = '}'
	return string(result)
}

func (self *NodeSet) Length() int {
	return (*bit.Set)(self).Size()
}

// string-based DAG that's easy to construct, and then gets converted
// to the real thing -- for testing only, but public so it can be used
// by other packages' tests
type TestDAG struct {
	nodes   []string
	parents map[string][]string
}

func NewTestDAG() *TestDAG {
	return &TestDAG{parents: make(map[string][]string)}
}

func (self *TestDAG) Add(name string, parent ...string) {
	self.nodes = append(self.nodes, name)
	self.parents[name] = parent
}

func (self *TestDAG) Finish() *DAG {
	dag := NewDAG()
	for _, name := range self.nodes {
		MakeStubNode(dag, name)
	}
	for _, name := range self.nodes {
		node := dag.Lookup(name)
		for _, pname := range self.parents[name] {
			pnode := dag.Lookup(pname)
			if pnode == nil {
				panic(fmt.Sprintf("node %s: invalid parent %s", name, pname))
			}
			dag.AddParent(node, pnode)
		}
	}
	dag.verify()
	return dag
}
