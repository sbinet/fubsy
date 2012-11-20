package dag

// Fubsy Node types for filesystem objects

import (
	"fmt"
	"code.google.com/p/go-bit/bit"
)

type FileNode struct {
	dag *DAG
	id int
	name string
	parentset bit.Set
}

// Lookup and return the named file node in dag. If it doesn't exist,
// create a new FileNode, add it to dag, and return it. If it does
// exist but isn't a FileNode, panic.
func makeFileNode(dag *DAG, name string) *FileNode {
	node := dag.lookup(name)
	if node == nil {
		fnode := &FileNode{
			id: -1,
			dag: dag,
			name: name,
		}
		fnode.id = dag.addNode(fnode)
		node = fnode
	}
	return node.(*FileNode)		// panic on unexpected type
}

func (self *FileNode) Id() int {
	return self.id
}

func (self *FileNode) Name() string {
	return self.name
}

func (self *FileNode) Equal(other_ Node) bool {
	if other, ok := other_.(*FileNode); ok {
		return other.name == self.name
	}
	return false
}

func (self *FileNode) Parents() []Node {
	result := make([]Node, 0)
	fetch := func(id int) {
		result = append(result, self.dag.nodes[id])
	}
	self.parentset.Do(fetch)
	return result
}

func (self *FileNode) AddParent (node Node) {
	// Bail if node is already in parentset.
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
