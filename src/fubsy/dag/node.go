package dag

import (
	"fmt"
	"code.google.com/p/go-bit/bit"
)

type NodeState byte

const (
	UNKNOWN NodeState = iota
	STALE
	BUILDING
	BUILT
)

// This interface must not betray anything about the filesystem,
// otherwise we'll have no hope of making non-file nodes. (They're not
// needed often, but when you need them you *really* need them.)
type Node interface {
	// brief human-readable string representation of this node (must
	// be unique in this graph)
	Name() string

	// human-readable string representation of this node that is not
	// necessarily unique; it's perfectly reasonable for String() and
	// Name() to return the same thing, but some Node types have to
	// resort to cryptic unique IDs Name(), so String() can be helpful
	// in presenting those nodes to the user
	String() string

	// unique integer identifier for this node
	Id() int

	// return true if this node and other describe the same resource
	// (should be sufficient to compare names)
	Equal(other Node) bool

	// return the set of node IDs for the nodes that this node depends on
	ParentSet() NodeSet

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

	// return true if this node has changed since the last build where
	// it was relevant
	Changed() (bool, error)

	SetState(state NodeState)

	State() NodeState
}


// Convenient base type for Node implementations -- provides the
// basics right out of the box.

type nodebase struct {
	dag *DAG
	id int
	name string
	parentset bit.Set
	action Action
	state NodeState
}

func makenodebase(dag *DAG, id int, name string) nodebase {
	return nodebase{
		dag: dag,
		id: id,
		name: name,
	}
}

func (self *nodebase) Id() int {
	return self.id
}

func (self *nodebase) Name() string {
	return self.name
}

func (self *nodebase) String() string {
	return self.name
}

func (self *nodebase) ParentSet() NodeSet {
	return NodeSet(&self.parentset)
}

func (self *nodebase) Parents() []Node {
	result := make([]Node, 0)
	fetch := func(id int) {
		result = append(result, self.dag.nodes[id])
	}
	self.parentset.Do(fetch)
	return result
}

func (self *nodebase) AddParent(node Node) {
	id := node.Id()
	if id < 0 || id >= self.dag.length() {
		panic(fmt.Sprintf(
			"%v has impossible id %d (should be >= 0 && <= %d)",
			node, id, self.dag.length() - 1))
	}
	if self.parentset.Contains(id) {
		return
	}
	self.parentset.Add(id)
}

func (self *nodebase) SetAction(action Action) {
	self.action = action
}

func (self *nodebase) Action() Action {
	return self.action
}

func (self *nodebase) SetState(state NodeState) {
	self.state = state
}

func (self *nodebase) State() NodeState {
	return self.state
}
