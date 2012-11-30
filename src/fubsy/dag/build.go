package dag

import (
	"fmt"
	"os"
	"strings"
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
		err := self.checkChanged(node)
		if err != nil {
			errors = append(errors, err)
		}
	})
	return errors
}

// The heart of Fubsy: keep building stale targets until all relevant
// targets are built (or failed).
func (self *BuildState) BuildStaleTargets() error {

	numtargets := 0				// total number of rebuilt targets
	failed := []Node {}			// targets that failed to build

	for !self.rebuild.IsEmpty() {
		for _, id := range setToSlice(self.rebuild) {
			numtargets++
			node := self.dag.nodes[id]
			node.SetState(BUILDING)
			err := node.Action().Execute()
			self.rebuild.Remove(id)

			if err != nil {
				// normal, everyday build failure
				node.SetState(FAILED)
				self.reportError(err)
				failed = append(failed, node)
				if !self.keepGoing() {
					break
				}
			}

			node.SetState(BUILT)
			err = self.checkChanged(node)
			if err != nil {
				// weird, pathological failure: e.g. a compiler wrote
				// an output file and made it unreadable, or did not
				// write the file it was supposed to write
				return err
			}
		}
	}

	if len(failed) > 0 {
		if self.keepGoing() {
			targets := joinNodes(", ", 10, failed)
			return fmt.Errorf("failed to build %d of %d targets: %s",
				len(failed), numtargets, targets)
		} else {
			// numtargets is meaningless if !keepGoing(), because we
			// broke out of the loop early
			return fmt.Errorf("failed to build target %s", failed[0])
		}
	}
	return nil
}

// (hopefully) temporary, pending acceptance of my patches to go-bit
func setToSlice(set *bit.Set) []int {
	result := make([]int, set.Size())
	j := 0
	set.Do(func(n int) {
		result[j] = n
		j++
	})
	return result
}

func joinNodes(delim string, max int, nodes []Node) string {
	if len(nodes) < max {
		max = len(nodes)
	}
	svalues := make([]string, max)
	for i := 0; i < max; i++ {
		svalues[i] = nodes[i].String()
	}
	if len(nodes) > max {
		svalues[max - 1] = "..."
	}
	return strings.Join(svalues, delim)
}

func (self *BuildState) keepGoing() bool {
	// eventually this should come from command-line option -k
	return true
}

func (self *BuildState) reportError(err error) {
	fmt.Fprintf(os.Stderr, "build failure: %s\n", err)
}

func (self *BuildState) checkChanged(node Node) error {
	changed, err := node.Changed()
	if err != nil {
		return err
	} else if changed  {
		self.children[node.Id()].Do(func (child int) {
			self.rebuild.Add(child)
			self.dag.nodes[child].SetState(STALE)
		})
	}
	return nil
}
