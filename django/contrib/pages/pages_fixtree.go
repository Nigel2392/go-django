package pages

import (
	"fmt"
	"slices"
	"strings"

	"github.com/Nigel2392/django/contrib/pages/models"
	"github.com/elliotchance/orderedmap/v2"
)

func fixTree(n *Node, depth int64) (cancel bool) {
	if n.Ref == nil {
		return
	}

	n.Ref.Numchild = int64(n.Children.Len())
	n.Ref.Depth = depth

	var i = 0
	for front := n.Children.Front(); front != nil; front = front.Next() {
		if front.Value.Ref == nil {
			continue
		}
		front.Value.Ref.Path = n.Ref.Path + buildPathPart(int64(i))
		front.Value.Ref.Depth = depth + 1
		i++
	}

	return
}

func newNode(ref *models.PageNode) *Node {
	return &Node{
		Ref:      ref,
		Children: orderedmap.NewOrderedMap[string, *Node](),
	}
}

func NewNodeTree(refs []*models.PageNode) *Node {
	var refsCpy = make([]*models.PageNode, len(refs))
	copy(refsCpy, refs)

	slices.SortStableFunc(refsCpy, func(a, b *models.PageNode) int {
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
	Ref      *models.PageNode
	Children *orderedmap.OrderedMap[string, *Node]
}

func (n *Node) FixTree() {
	n.ForEach(fixTree)
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
		if front.Value.forEach(fn, ctr) {
			return
		}
	}

	return
}

func (n *Node) forEach(fn func(*Node, int64) (cancel bool), depth int64) (cancelled bool) {
	if fn(n, depth) {
		return
	}

	for front := n.Children.Front(); front != nil; front = front.Next() {
		if front.Value.forEach(fn, depth+1) {
			return
		}
	}

	return
}

func (n *Node) FlatList() []*models.PageNode {
	var nodes = make([]*models.PageNode, 0)
	n.ForEach(func(node *Node, d int64) bool {
		if node.Ref != nil {
			nodes = append(nodes, node.Ref)
		}
		return false
	})
	return nodes

}
