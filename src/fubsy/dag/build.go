package dag

import (
	"fmt"
	"code.google.com/p/go-bit/bit"
)

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
		parents := self.dag.parents[id]
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

func (self *BuildState) ExpandDAG() []error {
	return self.dag.Expand(self.relevant)
}

// Compute the initial rebuild set, i.e. nodes that are 1) children of
// the original sources, 2) relevant (ancestors of a goal node), and
// 3) stale.
func (self *BuildState) FindStaleTargets() []error {
	errors := make([]error, 0)
	self.rebuild = bit.New()
	self.sources.Do(func (id int) {
		node := self.dag.nodes[id]
		if node == nil {
			return
		}
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
