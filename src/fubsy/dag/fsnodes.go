package dag

// Fubsy Node types for filesystem objects

import (
	"fmt"
)

type FileNode struct {
	dag *DAG
	id int
	name string
	parents []Node

	// Bit array: element [0] tells whether nodes 0..7 are members,
	// [1] describes 8..15, etc. Least significant bit of a byte is
	// for the lowest numbered node in that range.
	// Invariant after addParent():
	//     len(parentset) == (dag.length() + 7) / 8
	parentset []byte
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
	return self.parents
}

func (self *FileNode) AddParent (node Node) {
	// make sure parentset is big enough
	self.parentset = bitsetResize(self.parentset, self.dag.length())

	// Bail if node is already in parentset.
	id := node.Id()
	if id < 0 || id >= self.dag.length() {
		panic(fmt.Sprintf(
			"%v has impossible id %d (should be >= 0 && <= %d)",
			node, id, self.dag.length() - 1))
	}
	offset, mask := bitsetCoordinates(id)
	if self.parentset[offset] & mask != 0 {
		return
	}

	self.parents = append(self.parents, node)
	self.parentset[offset] |= mask
}

func bitsetResize(set []byte, length int) []byte {
	needbytes := (length + 7) / 8
	if len(set) < needbytes {
		if cap(set) >= needbytes {
			// no problem: just reslice
			set = set[0:needbytes]
		} else {
			// argh, gotta grow it
			newcap := needbytes * 3 / 2
			newset := make([]byte, needbytes, newcap)
			copy(newset, set)
			set = newset
		}
	}
	return set
}

func bitsetCoordinates(idx int) (offset int, mask byte) {
	offset = idx / 8
	mask = 1 << (uint(idx) % 8)
	return
}
