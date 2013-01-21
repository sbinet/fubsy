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
		if _, ok := obj.(Node); !ok {
			panic(fmt.Sprintf("not a Node: %#v (type %T)", obj, obj))
		}
		names[i] = obj.String()
	}
	name := strings.Join(names, ",")
	node := &ListNode{
		nodebase:     makenodebase(name),
		types.FuList: members,
	}
	return node
}

func MakeListNode(dag *DAG, member ...types.FuObject) *ListNode {
	node := newListNode(member...)
	node = dag.AddNode(node).(*ListNode)
	return node
}

func (self *ListNode) String() string {
	return self.nodebase.String()
}

func (self *ListNode) CommandString() string {
	return self.FuList.CommandString()
}

func (self *ListNode) Equal(other_ types.FuObject) bool {
	other, ok := other_.(*ListNode)
	return ok && reflect.DeepEqual(self, other)
}

func (self *ListNode) Add(other types.FuObject) (types.FuObject, error) {
	otherlist := other.List()
	result := make([]types.FuObject, len(self.FuList)+len(otherlist))
	for i, obj := range self.FuList {
		result[i] = obj
	}
	j := len(self.FuList)
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
	result := make([]Node, len(self.FuList))
	for i, obj := range self.FuList {
		result[i] = obj.(Node)
	}
	return result
}

func (self *ListNode) NodeExpand(ns types.Namespace) (Node, error) {
	result := make(types.FuList, len(self.FuList))
	var err error
	for i, obj := range self.FuList {
		result[i], err = obj.(Node).NodeExpand(ns)
		if err != nil {
			return nil, err
		}
	}
	return newListNode(result...), nil
}

func (self *ListNode) ActionExpand(ns types.Namespace) (types.FuObject, error) {
	result := make(types.FuList, 0, len(self.FuList))
	for _, obj := range self.FuList {
		// ok to panic here: already enforced in newListNode()
		node := obj.(Node)
		exp, err := node.ActionExpand(ns)
		if err != nil {
			return nil, err
		} else if exp == nil {
			result = append(result, node)
		} else {
			result = append(result, exp.List()...)
		}
	}
	return result, nil
}

func (self *ListNode) Exists() (bool, error) {
	panic("ListNode.Exists() not implemented yet")
}

func (self *ListNode) Signature() ([]byte, error) {
	panic("ListNode.Signature() not implemented yet")
}
