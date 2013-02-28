// Copyright © 2013, Greg Ward. All rights reserved.
// Use of this source code is governed by a BSD-style license that can
// be found in the LICENSE.txt file.

package dag

import (
	"testing"

	"github.com/stretchrcom/testify/assert"

	"fubsy/testutils"
	"fubsy/types"
)

func Test_ListNode_basics(t *testing.T) {
	node0 := NewStubNode("foo")
	node1 := NewStubNode("bar baz")
	node2 := NewStubNode("qux")
	list1 := newListNode(node0, node1, node2)
	assert.Equal(t, "ListNode", list1.Typename())
	assert.Equal(t, "foo,bar baz,qux", list1.Name())
	assert.Equal(t, `["foo", "bar baz", "qux"]`, list1.String())
	assert.Equal(t, "foo,bar baz,qux", list1.ValueString())
	assert.Equal(t, "foo 'bar baz' qux", list1.CommandString())

	expect1 := []types.FuObject{node0, node1, node2}
	assert.Equal(t, expect1, list1.List())

	expect2 := []Node{node0, node1, node2}
	assert.Equal(t, expect2, list1.Nodes())

	list2 := newListNode(node0, node1, node2)
	assert.True(t, list1.Equal(list2))

	list3 := newListNode(node1, node0, node2)
	assert.False(t, list1.Equal(list3))

	list4 := list3.copy().(*ListNode)
	assert.False(t, list3 == list4)
	assert.True(t, list3.Equal(list4))
}

func Test_ListNodeFromNodes(t *testing.T) {
	nodes := []Node{}
	list := ListNodeFromNodes(nodes)
	assert.Equal(t, []types.FuObject{}, list.List())
	assert.Equal(t, nodes, list.Nodes())

	node0 := NewStubNode("foo")
	node1 := NewStubNode("bar baz")
	node2 := NewStubNode("qux")
	nodes = []Node{node0, node1, node2}

	list = ListNodeFromNodes(nodes)
	assert.Equal(t, []types.FuObject{node0, node1, node2}, list.List())
	assert.Equal(t, nodes, list.Nodes())
}

func Test_MakeListNode(t *testing.T) {
	graph := NewDAG()
	list1 := MakeListNode(graph, NewStubNode("blah"))
	list2 := MakeListNode(graph, NewStubNode("blah"))
	assert.Equal(t, &list1, &list2)
}

func Test_ListNode_Add(t *testing.T) {
	nodelist := newListNode()
	otherlist := types.MakeFuList()

	actual, err := nodelist.Add(otherlist)
	testutils.NoError(t, err)
	if _, ok := actual.(*ListNode); !ok {
		t.Fatalf("expected object of type *ListNode, but got %T", actual)
	}
	assert.Equal(t, 0, len(actual.List()))

	node0 := NewStubNode("bla")
	node1 := NewStubNode("pog")
	otherlist = types.MakeFuList(node0, node1)
	actual, err = nodelist.Add(otherlist)
	testutils.NoError(t, err)
	nodes := actual.(*ListNode).List()
	if len(nodes) != 2 {
		t.Errorf("expected ListNode with 2 nodes, but got %v", actual)
	} else {
		assert.Equal(t, []types.FuObject{node0, node1}, nodes)
	}

	otherlist = types.MakeStringList("foo")
	actual, err = nodelist.Add(otherlist)
	assert.Nil(t, actual)
	assert.Equal(t,
		"unsupported operation: cannot add list to ListNode "+
			"(second operand contains string)",
		err.Error())
}

func Test_ListNode_ActionExpand(t *testing.T) {
	ns := types.NewValueMap()
	assertExpand := func(expect []Node, list *ListNode) {
		actualobj, err := list.ActionExpand(ns, nil)
		assert.Nil(t, err)
		actual := make([]Node, len(actualobj.List()))
		for i, obj := range actualobj.List() {
			actual[i] = obj.(Node)
		}
		if len(expect) == len(actual) {
			for i, enode := range expect {
				anode := actual[i]
				if !enode.Equal(anode) {
					t.Errorf("ListNode[%d]: expected <%T: %s> but got <%T: %s>",
						i, enode, enode, anode, anode)
				}
			}
		} else {
			t.Errorf(
				"ListNode %v: expected ActionExpand() to return %d Nodes, "+
					"but got %d: %v",
				list, len(expect), len(actual), actual)
		}
	}

	// a single empty ListNode yields an empty slice of Nodes
	list := newListNode()
	assertExpand([]Node{}, list)

	// a ListNode containing boring ordinary non-expanding nodes just
	// returns them
	node0 := NewStubNode("0")
	node1 := NewStubNode("1")
	list = newListNode(node0, node1)
	assertExpand([]Node{node0, node1}, list)

	// a ListNode with expanding nodes expands them (and flattens the
	// resulting list)
	list = newListNode(node1, list, node0)
	assertExpand([]Node{node1, node0, node1, node0}, list)

	ns.Assign("a", types.MakeFuString("argghh"))
	list = newListNode(node1, NewStubNode("say $a"), node0)
	assertExpand([]Node{node1, NewStubNode("say argghh"), node0}, list)
}

func Test_ListNode_expand_cycle(t *testing.T) {
	ns := types.NewValueMap()
	ns.Assign("a", types.MakeFuString("$b"))
	ns.Assign("b", types.MakeFuString("$a"))

	var err error
	inner := NewStubNode("foo/$a")
	list := newListNode(inner)

	// XXX ooops, ActionExpand() does not expand variable refs!
	// _, err = list.ActionExpand(ns, nil)
	// assert.Equal(t, "cyclic variable reference: a -> b -> a", err.Error())

	err = list.NodeExpand(ns)
	assert.Equal(t, "cyclic variable reference: a -> b -> a", err.Error())
}
