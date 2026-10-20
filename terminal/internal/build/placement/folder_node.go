package placement

type folderNode struct {
	path     string
	files    []string
	names    []string
	children map[string]*folderNode
	count    int
}

func (n *folderNode) subfolders() []*folderNode {
	out := make([]*folderNode, len(n.names))
	for i, name := range n.names {
		out[i] = n.children[name]
	}
	return out
}

func (n *folderNode) allPaths() []string {
	out := append([]string{}, n.files...)
	for _, child := range n.subfolders() {
		out = append(out, child.allPaths()...)
	}
	return out
}
