package dag

// Fubsy Node types for filesystem objects

import (
	"fmt"
	"fubsy/types"
)

type FileNode struct {
	// name: filename (relative to top)
	nodebase
}

type GlobNode struct {
	// name: arbitrary unique string
	nodebase
	glob types.FuObject			// likely FuFileFinder or FuFinderList
}

// Lookup and return the named file node in dag. If it doesn't exist,
// create a new FileNode, add it to dag, and return it. If it does
// exist but isn't a FileNode, panic.
func MakeFileNode(dag *DAG, name string) *FileNode {
	node := dag.lookup(name)
	if node == nil {
		fnode := &FileNode{
			nodebase: makenodebase(dag, name),
		}
		dag.addNode(fnode)
		node = fnode
	}
	return node.(*FileNode)		// panic on unexpected type
}

func (self *FileNode) Equal(other_ Node) bool {
	other, ok := other_.(*FileNode)
	return ok && other.name == self.name
}

func (self *FileNode) Changed() (bool, error) {
	// placeholder until we have persistent build state
	return true, nil
}

func (self *FileNode) addParent(parent Node) {
	self.dag.addParent(self, parent)
}

func MakeGlobNode(dag *DAG, glob_ types.FuObject) *GlobNode {
	var name string
	var globid int
	switch glob :=  glob_.(type) {
	case *types.FuFileFinder:
		globid = glob.Id()
	case *types.FuFinderList:
		globid = glob.Id()
	default:
		panic(fmt.Sprintf("cannot make GlobNode from %T object", glob))
	}
	name = fmt.Sprintf("glob%04d", globid)

	node := dag.lookup(name)
	if node == nil {
		gnode := &GlobNode{
			nodebase: makenodebase(dag, name),
			glob: glob_,
		}
		dag.addNode(gnode)
		node = gnode
	}
	return node.(*GlobNode)		// panic on unexpected type
}

func (self *GlobNode) String() string {
	return self.glob.String()
}

func (self *GlobNode) Equal(other_ Node) bool {
	other, ok := other_.(*GlobNode)
	return ok && self.glob.Equal(other.glob)
}

func (self *GlobNode) Expand() error {
	filenames, err := self.glob.Expand()
	if err != nil {
		return err
	}
	newnodes := []Node {}
	for _, fnobj := range filenames.List() {
		// fnobj had better be a FuString -- panic if not
		fn := fnobj.(types.FuString).Value()
		node := MakeFileNode(self.dag, fn)
		newnodes = append(newnodes, node)
	}
	//self.dag.removeNode(self)
	self.dag.replaceNode(self, newnodes)
	return nil
}

func (self *GlobNode) Changed() (bool, error) {
	panic("Changed() should never be called on a GlobNode " +
		"(graph should have been expanded by this point)")
}

func (self *GlobNode) addParent(parent Node) {
	self.dag.addParent(self, parent)
}
