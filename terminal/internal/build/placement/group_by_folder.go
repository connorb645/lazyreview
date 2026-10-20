package placement

import (
	"math"
	"slices"
	"sort"
	"strings"

	"github.com/connorb645/lazyreview/terminal/internal/utils"
)

// GroupByFolder groups paths by folder when nothing names their
// components. It starts from the deepest folder that holds every path and
// splits the largest group into its subfolders while the result stays
// within the grouping's limit. One exception: a folder where one subfolder
// holds most of the paths, and that subfolder would itself split, splits
// into that subfolder and one group of everything else, so a change inside
// one part of a large repository is still divided by its parts. The files
// directly in a split folder stay together as one group of their own. The
// groups come in folder order, the group of everything else last.
func GroupByFolder(paths []string, g Grouping) []FolderGroup {
	root := buildTree(paths)
	groups := []*folderPiece{collapse(root)}
	if top := groups[0].node; g.SplitTop && len(top.names) > 0 && len(parts(top)) > 1 && len(parts(top)) <= g.MaxGroups {
		groups = parts(top)
	}
	pieces := func(p *folderPiece) []*folderPiece {
		if !g.Balanced || isBalanced(p.node) {
			return parts(p.node)
		}
		rest := dominant(p.node, g.SplitAbove)
		return []*folderPiece{collapse(rest), {node: p.node, rest: rest}}
	}
	for {
		var candidates []*folderPiece
		for _, p := range groups {
			if !p.direct && p.rest == nil && len(p.node.names) > 0 && float64(p.node.count) > g.SplitAbove && (!g.Balanced || isBalanced(p.node) || dominant(p.node, g.SplitAbove) != nil) {
				candidates = append(candidates, p)
			}
		}
		sort.SliceStable(candidates, func(a, b int) bool { return candidates[a].node.count > candidates[b].node.count })
		var pick *folderPiece
		for _, p := range candidates {
			if len(groups)-1+len(pieces(p)) <= g.MaxGroups {
				pick = p
				break
			}
		}
		if pick == nil {
			break
		}
		var next []*folderPiece
		for _, p := range groups {
			if p == pick {
				next = append(next, pieces(p)...)
			} else {
				next = append(next, p)
			}
		}
		groups = next
	}
	out := make([]FolderGroup, len(groups))
	for i, p := range groups {
		switch {
		case p.rest != nil:
			paths := append([]string{}, p.node.files...)
			for _, child := range p.node.subfolders() {
				if child != p.rest {
					paths = append(paths, child.allPaths()...)
				}
			}
			out[i] = FolderGroup{Others: true, Paths: paths}
		case p.direct:
			out[i] = FolderGroup{Folder: p.node.path, Paths: append([]string{}, p.node.files...), Direct: true}
		default:
			out[i] = FolderGroup{Folder: p.node.path, Paths: p.node.allPaths()}
		}
	}
	slices.SortStableFunc(out, func(a, b FolderGroup) int {
		if a.Others != b.Others {
			if a.Others {
				return 1
			}
			return -1
		}
		if c := utils.LocaleCompare(a.Folder, b.Folder); c != 0 {
			return c
		}
		switch {
		case a.Direct == b.Direct:
			return 0
		case a.Direct:
			return -1
		}
		return 1
	})
	return out
}

func buildTree(paths []string) *folderNode {
	root := newFolderNode("")
	for _, path := range paths {
		segments := strings.Split(path, "/")
		at := root
		at.count++
		for _, segment := range segments[:len(segments)-1] {
			next := segment
			if at.path != "" {
				next = at.path + "/" + segment
			}
			child, held := at.children[segment]
			if !held {
				child = newFolderNode(next)
				at.children[segment] = child
				at.names = append(at.names, segment)
			}
			at = child
			at.count++
		}
		at.files = append(at.files, path)
	}
	return root
}

func newFolderNode(path string) *folderNode {
	return &folderNode{path: path, children: map[string]*folderNode{}}
}

func collapse(n *folderNode) *folderPiece {
	at := n
	for len(at.files) == 0 && len(at.names) == 1 {
		at = at.children[at.names[0]]
	}
	return &folderPiece{node: at}
}

func parts(n *folderNode) []*folderPiece {
	var out []*folderPiece
	for _, child := range n.subfolders() {
		out = append(out, collapse(child))
	}
	if len(n.files) > 0 {
		out = append(out, &folderPiece{node: n, direct: true})
	}
	return out
}

func dominant(n *folderNode, splitAbove float64) *folderNode {
	for _, child := range n.subfolders() {
		inner := collapse(child).node
		if float64(child.count) >= float64(n.count)*0.75 && len(inner.names) > 0 && float64(inner.count) > splitAbove && isBalanced(inner) {
			return child
		}
	}
	return nil
}

func isBalanced(n *folderNode) bool {
	share := math.Max(2, float64(n.count)*0.12)
	big := 0
	for _, child := range n.subfolders() {
		if float64(child.count) >= share {
			big++
		}
	}
	if float64(len(n.files)) >= share {
		big++
	}
	return big >= 2
}
