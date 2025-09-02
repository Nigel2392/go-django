package pages

import (
	"fmt"
	"path"
	"slices"
	"strings"

	"github.com/elliotchance/orderedmap/v2"
)

func newNode(ref *PageNode) *Node {
	return &Node{
		Ref:      ref,
		Children: orderedmap.NewOrderedMap[string, *Node](),
	}
}

func NewNodeTree(refs []*PageNode) *Node {
	var refsCpy = make([]*PageNode, len(refs))
	copy(refsCpy, refs)

	slices.SortStableFunc(refsCpy, func(a, b *PageNode) int {
		return strings.Compare(a.Path, b.Path)
	})

	var root = newNode(nil)

	for _, ref := range refsCpy {
		current := root
		for i := 0; i < len(ref.Path); i += STEP_LEN {
			ref := ref
			part := ref.Path[:i+STEP_LEN]
			if child, ok := current.Children.Get(part); ok {
				current = child
				continue
			}
			node := newNode(nil)
			current.Children.Set(part, node)
			current = node
		}
		current.Ref = ref
	}

	return root
}

func PrintTree(node *Node, depth int) {
	if node.Ref != nil {
		fmt.Printf("%s%s\n", strings.Repeat(" ", depth*2), node.Ref.Path)
	}
	for front := node.Children.Front(); front != nil; front = front.Next() {
		child := front.Value
		PrintTree(child, depth+1)
	}
}

type Node struct {
	Ref      *PageNode
	Children *orderedmap.OrderedMap[string, *Node]
}

func (n *Node) FixTree() []*PageNode {
	var i int64 = 0
	var changedList = make([]*PageNode, 0)

	for front := n.Children.Front(); front != nil; front = front.Next() {
		// Fix all root nodes
		var node = front.Value

		if node.Ref == nil {
			continue
		}

		var addToChangedList bool
		if node.Ref.Depth != 0 {
			node.Ref.Depth = 0
			addToChangedList = true
		}

		var urlPath = path.Join("/", node.Ref.Slug)
		if urlPath != node.Ref.UrlPath {
			node.Ref.UrlPath = urlPath
			addToChangedList = true
		}

		if node.Ref.Numchild != int64(node.Children.Len()) {
			node.Ref.Numchild = int64(node.Children.Len())
			addToChangedList = true
		}

		var path = buildPathPart(i)
		if path != node.Ref.Path {
			node.Ref.Path = buildPathPart(i)
			addToChangedList = true
		}

		// Fix all children
		if addToChangedList {
			changedList = append(changedList, node.Ref)
		}

		changedList = append(changedList, node.fixChildren(1)...)
		i++
	}

	return changedList
}

func (n *Node) fixChildren(depth int64) []*PageNode {
	var i int64 = 0
	var changedList = make([]*PageNode, 0)
	for front := n.Children.Front(); front != nil; front = front.Next() {
		var node = front.Value
		if node.Ref == nil {
			continue
		}

		var addToChangedList bool
		if node.Ref.Depth != depth {
			node.Ref.Depth = depth
			addToChangedList = true
		}

		var urlPath = path.Join(n.Ref.UrlPath, node.Ref.Slug)
		if urlPath != node.Ref.UrlPath {
			node.Ref.UrlPath = urlPath
			addToChangedList = true
		}

		if node.Ref.Numchild != int64(node.Children.Len()) {
			node.Ref.Numchild = int64(node.Children.Len())
			addToChangedList = true
		}

		if node.Ref.Path != n.Ref.Path+buildPathPart(i) {
			node.Ref.Path = n.Ref.Path + buildPathPart(i)
			addToChangedList = true
		}

		if addToChangedList {
			changedList = append(changedList, node.Ref)
		}

		changedList = append(changedList, node.fixChildren(depth+1)...)
		i++
	}
	return changedList
}

func (n *Node) FindNode(path string) *Node {
	if len(path) > 0 && n.Ref == nil {
		goto children
	}

	if n.Ref != nil && n.Ref.Path == path {
		return n
	}

	if len(path) == 0 {
		return nil
	}

children:
	for front := n.Children.Front(); front != nil; front = front.Next() {
		if node := front.Value.FindNode(path); node != nil {
			return node
		}
	}

	return nil
}

func (n *Node) ForEach(fn func(*Node, int64) (cancel bool)) (cancelled bool) {
	var ctr int64 = 0
	if n.Ref != nil && fn(n, ctr) {
		return
	}

	if n.Ref != nil {
		ctr++
	}

	for front := n.Children.Front(); front != nil; front = front.Next() {
		if front.Value.forEach(fn, ctr, false) {
			return
		}
	}

	return
}

func (n *Node) forEach(fn func(*Node, int64) (cancel bool), depth int64, execForRoot bool) (cancelled bool) {
	if n.Ref != nil && execForRoot {
		if fn(n, depth) {
			return
		}
	}

	for front := n.Children.Front(); front != nil; front = front.Next() {
		if front.Value.forEach(fn, depth+1, false) {
			return
		}
	}

	return
}

func (n *Node) FlatList() []*PageNode {
	var nodes = make([]*PageNode, 0)
	n.ForEach(func(node *Node, d int64) bool {
		if node.Ref != nil {
			nodes = append(nodes, node.Ref)
		}
		return false
	})
	return nodes

}
