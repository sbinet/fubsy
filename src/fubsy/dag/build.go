package dag

import (
	"fmt"
	"os"
	"strings"
	"code.google.com/p/go-bit/bit"
)

type BuildState struct {
	dag *DAG

	// not much else here now, but at some point we're going to need a
	// place to put user options -- how do we transfer "--keep-going"
	// from the command line to keepGoing() below?
}

// The heart of Fubsy: find the initial set of stale targets, then
// keep building stale targets until all are built (or failed).
func (self *BuildState) BuildStaleTargets() []error {
	stale, errors := self.findStaleTargets()
	if errors != nil {
		return errors
	}

	attempted := 0				// number of targets we tried to build
	failed := []Node {}			// targets that failed to build

	for !stale.IsEmpty() {
		for _, id := range setToSlice(stale) {
			node := self.dag.nodes[id]
			if node.State() == TAINTED {
				// one of this node's parents failed to build: skip it
				stale.Remove(id)
				continue
			}

			attempted++
			node.SetState(BUILDING)
			err := node.Action().Execute()
			stale.Remove(id)
			//fmt.Printf("%s: err = %v\n", node.Action(), err)

			if err != nil {
				// normal, everyday build failure
				node.SetState(FAILED)
				self.reportError(err)
				failed = append(failed, node)
				if !self.keepGoing() {
					break
				}
				taintChildren(self.dag, id, node)
				continue		// to the next stale target
			}

			node.SetState(BUILT)
			err = checkChanged(self.dag, id, node, stale)
			if err != nil {
				// weird, pathological failure: e.g. a compiler wrote
				// an output file and made it unreadable, or did not
				// write the file it was supposed to write
				return []error {err}
			}
		}
	}

	// Collapse possibly many build failures down to a single error
	// object, because build failure are reported as they happen. At
	// this point we just need a single error object to summarize the
	// whole failure and make the process terminate soon.
	err := self.explainFailures(attempted, failed) // nil if no failures
	if err != nil {
		return []error {err}
	}
	return nil
}

// Compute the initial set set of stale nodes, i.e. nodes that are 1)
// children of the original sources, 2) relevant (ancestors of a goal
// node), and 3) stale.
func (self *BuildState) findStaleTargets() (*bit.Set, []error) {
	//fmt.Printf("FindStaleTargets():\n")
	if self.dag.children == nil {
		panic("dag.children == nil: did you forget to call dag.ComputeChildren()?")
	}

	var errors []error
	stale := bit.New()
	for id, node := range self.dag.nodes {
		if !self.dag.parents[id].IsEmpty() {
			// node has parents, so it's not an original source
			continue
		}
		err := checkChanged(self.dag, id, node, stale)
		if err != nil {
			errors = append(errors, err)
		}
	}

	//fmt.Printf("FindStaleTargets(): initial stale set = %v\n", stale)
	return stale, errors
}

func (self *BuildState) explainFailures(attempted int, failed []Node) error {
	if len(failed) == 0 {
		return nil
	}

	var err error
	if self.keepGoing() {
		targets := joinNodes(", ", 10, failed)
		err = fmt.Errorf("failed to build %d of %d targets: %s",
			len(failed), attempted, targets)
	} else {
		// attempted is meaningless if !keepGoing(), because we
		// broke out of the main loop early
		err = fmt.Errorf("failed to build target %s", failed[0])
	}
	return err
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

func taintChildren(dag *DAG, id int, node Node) {
	dag.children[id].Do(func(childid int) {
		dag.nodes[childid].SetState(TAINTED)
	})
}

func checkChanged(
	dag *DAG, id int, node Node, stale *bit.Set) error {

	changed, err := node.Changed()
	if err != nil {
		return err
	} else if changed  {
		if dag.children[id] == nil {
			// temporary, until dag.ComputeChildren() implemented
			panic(fmt.Sprintf(
				"BuildState: no children known for node %d (%v)", id, node))
		}
		dag.children[id].Do(func(childid int) {
			child := dag.nodes[childid]
			if child.State() != TAINTED {
				// XXX this might be premature: I think we should not add a
				// node to the stale set until *all* of its parents are
				// built! (that might fix the whole "tainted" thing too)
				stale.Add(childid)
				child.SetState(STALE)
			}
		})
	}
	return nil
}
