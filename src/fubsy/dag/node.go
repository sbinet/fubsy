// Copyright © 2012-2013, Greg Ward. All rights reserved.
// Use of this source code is governed by a BSD-style license that can
// be found in the LICENSE.txt file.

package dag

import (
	"bytes"
	"errors"

	"fubsy/types"
)

// hmmm: this is really the *build* state of a given node
type NodeState byte

const (
	// the default state for all nodes
	UNKNOWN NodeState = iota

	// this is a source node, so it doesn't make sense to have a build
	// state; this state just exists because it looks smarter than
	// UNKNOWN. not sure if we really need it...
	SOURCE

	// this node is currently building
	BUILDING

	// the attempt to build this node failed
	FAILED

	// we did not try to build this node, because building one of its
	// ancestors failed
	TAINTED

	// this node was successfully built
	BUILT
)

var statenames []string

func init() {
	statenames = []string{
		"UNKNOWN", "SOURCE", "BUILDING", "FAILED", "TAINTED", "BUILT"}
}

func (self NodeState) String() string {
	return statenames[int(self)]
}

// This interface must not betray anything about the filesystem,
// otherwise we'll have no hope of making non-file nodes. (They're not
// needed often, but when you need them you *really* need them.)
type Node interface {
	types.FuObject

	setid(id int)
	id() int
	copy() Node

	// brief human-readable string representation of this node (must
	// be unique in this graph)
	Name() string

	// N.B. it's perfectly reasonable for String() and Name() to
	// return the same thing, but some Node types have to resort to
	// using cryptic unique IDs as their Name(), so String() can be
	// helpful in presenting those nodes to the user -- the important
	// thing is that String() does not *necessarily* return a unique
	// representation, whereas Name() does

	// Set the BuildRule that tells how to build this node (i.e. for
	// which this node is a target). Non-target nodes (original
	// sources) have no BuildRule.
	SetBuildRule(rule BuildRule)

	// Return the build rule previously passed to SetBuildRule() (nil
	// if no rule has ever been set, which implies that this is an
	// original source node).
	BuildRule() BuildRule

	// Transform a node in-place from its initial representation
	// (computed in the main phase) to something that can be used to
	// find/read/write actual resources in the real world. The
	// canonical example is expanding variable references, e.g. a
	// FinderNode <$src/$app/*.c> might expand to <some/deep/dir/*.c>,
	// depending on the values of 'src' and 'app' -- but it remains a
	// FinderNode. This is done to every node in the DAG early in the
	// build phase, before selecting targets to build.
	NodeExpand(ns types.Namespace) error

	// return true if the resource represented by this node already
	// exists -- we don't care if it's stale or up-to-date, or whether
	// it has changed or not... simply, does it exist? (non-existence
	// is a short-circuit that means we don't have to check if parent
	// nodes have changed, because of course we have to rebuild this
	// node)
	Exists() (bool, error)

	// return a brief byte sequence that summarizes the content of
	// this node, and can be used to determine if it has changed since
	// a previous build (e.g. file modification time, content hash, or
	// a combination of such data)
	Signature() ([]byte, error)

	// return true if the two signatures are different, which
	// indicates that this node has changed since the previous build
	// of some unspecified target
	Changed(oldsig, newsig []byte) bool

	SetState(state NodeState)

	State() NodeState
}

// a build rule relates source(s) to target(s) by way of action(s)
type BuildRule interface {
	// Run this rule's action(s) to build its targets from their
	// sources. Return the rule's list of target nodes, whether they
	// built successfully or not, and a list of errors. (Multiple
	// errors are possible because of builtin functions like mkdir()
	// and remove(), which keep going after errors and thus must
	// report all errors that they encounter.) Caller must not mutate
	// targets, since it may be internal state of the BuildRule.
	Execute() (targets []Node, errs []error)

	// Return a string describing this rule's action(s).
	ActionString() string
}

// Convenient base type for Node implementations -- provides the
// basics right out of the box. Real Node implementations still have
// to take care of:
//   Equal()
//   Exists()
//   Signature()

type nodebase struct {
	_id   int
	name  string
	rule  BuildRule
	state NodeState

	// make sure we only call NodeExpand() once (imagine expanding
	// "$a$b" where a = "$" and b = "x": if we call NodeExpand() a
	// second time, then we'll try to expand "$x", which would be
	// insane)
	expanded bool

	// implement Lookup() for attributes
	types.ValueMap
}

func makenodebase(name string) nodebase {
	return nodebase{
		_id:  -1,
		name: name,
	}
}

func (self *nodebase) Name() string {
	return self.name
}

func (self *nodebase) setid(id int) {
	self._id = id
}

func (self *nodebase) id() int {
	return self._id
}

func (self *nodebase) SetBuildRule(rule BuildRule) {
	self.rule = rule
}

func (self *nodebase) BuildRule() BuildRule {
	return self.rule
}

func (self *nodebase) NodeExpand(ns types.Namespace) error {
	if self.expanded {
		return nil
	}
	_, name, err := types.ExpandString(self.name, ns, nil)
	if err != nil {
		return err
	}
	self.name = name
	self.expanded = true
	return nil
}

func (self *nodebase) Changed(oldsig, newsig []byte) bool {
	if oldsig == nil || newsig == nil {
		panic("node signatures must not be nil")
	}
	return !bytes.Equal(oldsig, newsig)
}

func (self *nodebase) SetState(state NodeState) {
	self.state = state
}

func (self *nodebase) State() NodeState {
	return self.state
}

// some methods to implement FuObject

func (self *nodebase) String() string {
	return "\"" + self.name + "\""
}

func (self *nodebase) ValueString() string {
	return self.name
}

func (self *nodebase) CommandString() string {
	return types.ShellQuote(self.name)
}

func defaultNodeAdd(self Node, other types.FuObject) (types.FuObject, error) {
	otherlist := other.List()
	values := make([]types.FuObject, 0, 1+len(otherlist))
	values = append(values, self)
	values = append(values, otherlist...)
	return types.MakeFuList(values...), nil
}

func defaultNodeActionExpand(
	self Node, ns types.Namespace) (
	types.FuObject, error) {
	err := self.NodeExpand(ns)
	if err != nil {
		return nil, err
	}
	return self, nil
}

// StubNode is test code only, but it's used by tests in other
// packages, so cannot be in node_test.go.

type StubNode struct {
	nodebase
	exists bool
	sig    []byte
}

func (self *StubNode) SetExists(exists bool) {
	self.exists = exists
}

func (self *StubNode) SetSignature(sig []byte) {
	self.sig = sig
}

func (self *StubNode) Typename() string {
	return "StubNode"
}

func (self *StubNode) copy() Node {
	var c StubNode = *self
	return &c
}

func (self *StubNode) Equal(other_ types.FuObject) bool {
	other, ok := other_.(*StubNode)
	return ok && self.name == other.name
}

func (self *StubNode) Exists() (bool, error) {
	return self.exists, nil
}

func (self *StubNode) ActionExpand(ns types.Namespace, ctx *types.ExpandContext) (types.FuObject, error) {
	return self, nil
}

func (self *StubNode) Signature() ([]byte, error) {
	return self.sig, nil
}

func (self *StubNode) Add(other types.FuObject) (types.FuObject, error) {
	panic("should be unused in tests")
}

func (self *StubNode) List() []types.FuObject {
	return []types.FuObject{self}
}

func NewStubNode(name string) *StubNode {
	return &StubNode{
		nodebase: makenodebase(name),
		sig:      []byte{},
	}
}

func MakeStubNode(dag *DAG, name string) *StubNode {
	_, node := dag.addNode(NewStubNode(name))
	return node.(*StubNode)
}

// stub implementation of BuildRule for use in unit tests (similar to
// StubNode, this has to be public so it can be used in other
// packages' tests)
type StubRule struct {
	// takes name of first target -- used for recording order in which
	// targets are built
	callback func(string)

	targets  []Node
	fail     bool
	executed bool
}

func MakeStubRule(callback func(string), target ...Node) *StubRule {
	return &StubRule{
		callback: callback,
		targets:  target,
	}
}

func (self *StubRule) SetFail(fail bool) {
	self.fail = fail
}

func (self *StubRule) Execute() ([]Node, []error) {
	self.callback(self.targets[0].Name())
	errs := []error{}
	if self.fail {
		errs = append(errs, errors.New("action failed"))
	}
	return self.targets, errs
}

func (self *StubRule) ActionString() string {
	return "build " + self.targets[0].String()
}
