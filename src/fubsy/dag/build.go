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

// Compute the initial rebuild set, i.e. nodes that are 1) children of
// the original sources, 2) relevant (ancestors of a goal node), and
// 3) stale.
func FindStaleTargets(dag *DAG) (NodeSet, []error) {
	//fmt.Printf("FindStaleTargets():\n")
	if dag.children == nil {
		panic("dag.children == nil: did you forget to call dag.ComputeChildren()?")
	}

	errors := make([]error, 0)
	stale := bit.New()
	for id, node := range dag.nodes {
		if !dag.parents[id].IsEmpty() {
			// node has parents, so it's not an original source
			continue
		}
		err := checkChanged(dag, id, node, stale)
		if err != nil {
			errors = append(errors, err)
		}
	}

	//fmt.Printf("FindStaleTargets(): initial stale set = %v\n", stale)
	return NodeSet(stale), errors
}

// The heart of Fubsy: keep building stale targets until all relevant
// targets are built (or failed).
func (self *BuildState) BuildStaleTargets(stale_ NodeSet) error {

	stale := (*bit.Set)(stale_)
	attempted := 0				// number of targets we tried to build
	failed := []Node {}			// targets that failed to build

	for !stale.IsEmpty() {
		for _, id := range setToSlice(stale) {
			attempted++
			node := self.dag.nodes[id]
			node.SetState(BUILDING)
			err := node.Action().Execute()
			stale.Remove(id)

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
			err = checkChanged(self.dag, id, node, stale)
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
				len(failed), attempted, targets)
		} else {
			// attempted is meaningless if !keepGoing(), because we
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
		dag.children[id].Do(func (childid int) {
			stale.Add(childid)
			dag.nodes[childid].SetState(STALE)
		})
	}
	return nil
}
