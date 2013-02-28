// Copyright © 2013, Greg Ward. All rights reserved.
// Use of this source code is governed by a BSD-style license that can
// be found in the LICENSE.txt file.

package dag

import (
	"fmt"
	"reflect"
	"strings"

	"fubsy/types"
)

// ListNode is used for the sum of Nodes and other objects. It's very
// similar to FuList, except it also implements Node.

type ListNode struct {
	nodebase
	// Embedding FuList is a bit of a lazy shortcut: really, we want a
	// []Node, not a []FuObject. But if we used []Node, then we'd have
	// to copy all the code in FuList, just changing a lot of FuObject
	// declarations to Node. So instead we allow any FuObject at
	// compile time, but then panic at runtime if a non-Node is
	// detected.
	types.FuList
}

// Create a new ListNode. The members must all be Nodes; everything is
// declared as FuObject for convenience and to avoid repetitive
// coding, but you'll get a panic if you pass in any non-Node objects.
func newListNode(members ...types.FuObject) *ListNode {
	names := make([]string, len(members))
	for i, obj := range members {
		if node, ok := obj.(Node); ok {
			names[i] = node.Name()
		} else {
			panic(fmt.Sprintf("not a Node: %#v (type %T)", obj, obj))
		}
	}
	name := strings.Join(names, ",")
	node := &ListNode{
		nodebase:     makenodebase(name),
		types.FuList: types.MakeFuList(members...),
	}
	return node
}

func ListNodeFromNodes(nodes []Node) *ListNode {
	names := make([]string, len(nodes))
	for i, node := range nodes {
		names[i] = node.Name()
	}
	name := strings.Join(names, ",")
	values := make([]types.FuObject, len(nodes))
	for i, node := range nodes {
		values[i] = node
	}
	return &ListNode{
		nodebase:     makenodebase(name),
		types.FuList: types.MakeFuList(values...),
	}
}

func MakeListNode(dag *DAG, member ...types.FuObject) *ListNode {
	node := newListNode(member...)
	node = dag.AddNode(node).(*ListNode)
	return node
}

func (self *ListNode) Lookup(name string) (types.FuObject, bool) {
	return self.FuList.Lookup(name)
}

func (self *ListNode) String() string {
	return self.FuList.String()
}

func (self *ListNode) ValueString() string {
	return self.nodebase.ValueString()
}

func (self *ListNode) CommandString() string {
	return self.FuList.CommandString()
}

func (self *ListNode) Equal(other_ types.FuObject) bool {
	other, ok := other_.(*ListNode)
	return ok && reflect.DeepEqual(self, other)
}

func (self *ListNode) Add(other types.FuObject) (types.FuObject, error) {
	values := self.FuList.List()
	otherlist := other.List()
	result := make([]types.FuObject, len(values)+len(otherlist))
	for i, obj := range values {
		result[i] = obj
	}
	j := len(values)
	for i, obj := range otherlist {
		if _, ok := obj.(Node); !ok {
			err := fmt.Errorf(
				"unsupported operation: cannot add "+
					"%s %v to %s %v",
				obj, obj.Typename(), self.Typename(), self)
			return nil, err
		}
		result[j+i] = obj
	}
	return newListNode(result...), nil
}

func (self *ListNode) Typename() string {
	return "ListNode"
}

func (self *ListNode) copy() Node {
	var c ListNode = *self
	return &c
}

func (self *ListNode) Nodes() []Node {
	values := self.FuList.List()
	result := make([]Node, len(values))
	for i, obj := range values {
		result[i] = obj.(Node)
	}
	return result
}

func (self *ListNode) NodeExpand(ns types.Namespace) error {
	if self.expanded {
		return nil
	}

	var err error
	for _, obj := range self.FuList.List() {
		err = obj.(Node).NodeExpand(ns)
		if err != nil {
			return err
		}
	}
	self.expanded = true
	return nil
}

func (self *ListNode) ActionExpand(
	ns types.Namespace, ctx *types.ExpandContext) (
	types.FuObject, error) {
	err := self.NodeExpand(ns)
	if err != nil {
		return nil, err
	}
	values := self.FuList.List()
	xvalues := make([]types.FuObject, 0, len(values))
	for _, obj := range values {
		// ok to panic here: already enforced in newListNode()
		node := obj.(Node)
		exp, err := node.ActionExpand(ns, ctx)
		if err != nil {
			return nil, err
		} else if exp == nil {
			xvalues = append(xvalues, node)
		} else {
			xvalues = append(xvalues, exp.List()...)
		}
	}
	return types.MakeFuList(xvalues...), nil
}

func (self *ListNode) Exists() (bool, error) {
	panic("ListNode.Exists() not implemented yet")
}

func (self *ListNode) Signature() ([]byte, error) {
	panic("ListNode.Signature() not implemented yet")
}
