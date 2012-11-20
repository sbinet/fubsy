package dag

// Fubsy Node types for filesystem objects

type FileNode struct {
	dag *DAG
	id int
	name string
	parents []Node
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

// XXX this is O(N) in the number of parents, so O(N^2) when adding
// many parents!
func (self *FileNode) AddParent (node Node) {
	for _, parent := range self.parents {
		if node.Equal(parent) {
			return
		}
	}
	self.parents = append(self.parents, node)
}
