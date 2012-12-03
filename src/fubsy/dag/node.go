package dag

import (
)

type NodeState byte

const (
	UNKNOWN NodeState = iota
	STALE
	BUILDING
	FAILED
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

	// return true if this node and other describe the same resource
	// (should be sufficient to compare names)
	Equal(other Node) bool

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

	// Generate new nodes that represent the same resources as this
	// node, only more concretely. Return nil if no expansion is
	// necessary (the normal case). Canonical use case: expanding
	// wildcards by replacing one GlobNode with zero or more
	// FileNodes. Note that Node.Expand() implementations are
	// responsible for creating new Node objects and thereby adding
	// them to the DAG, but not for rewiring existing parent/child
	// relationships. That happens in DAG.Rebuild(). Also, the
	// existing node is dropped by default; implemenations that want
	// to preserve themselves must include self in the return list of
	// replacement Nodes.
	Expand(dag *DAG) ([]Node, error)

	// return true if this node has changed since the last build where
	// it was relevant
	Changed() (bool, error)

	SetState(state NodeState)

	State() NodeState
}


// Convenient base type for Node implementations -- provides the
// basics right out of the box.

type nodebase struct {
	name string
	action Action
	state NodeState
}

func makenodebase(name string) nodebase {
	return nodebase{
		name: name,
	}
}

func (self *nodebase) Name() string {
	return self.name
}

func (self *nodebase) String() string {
	return self.name
}

func (self *nodebase) SetAction(action Action) {
	self.action = action
}

func (self *nodebase) Action() Action {
	return self.action
}

func (self *nodebase) Expand(dag *DAG) ([]Node, error) {
	return nil, nil
}

func (self *nodebase) SetState(state NodeState) {
	self.state = state
}

func (self *nodebase) State() NodeState {
	return self.state
}
