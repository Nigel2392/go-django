package binarytree

import (
	"fmt"
	"math"
	"strings"

	"golang.org/x/exp/slices"
)

type Comparable[T any] interface {
	Lt(other T) bool
}

// A binary search tree implementation which works with any type that implements the Comparable[T] interface.
type InterfacedBST[T Comparable[T]] struct {
	root   *IF_BSTNode[T]
	len    int
	height int
}

// Return the binary search tree as a string.
func (t *InterfacedBST[T]) String() string {
	if t.root == nil {
		return ""
	}

	height := t.root.getHeight()
	IF_BSTNodes := make([][]string, height)

	fillIF_BSTNodes(IF_BSTNodes, t.root, 0)

	var b strings.Builder
	padding := int(math.Pow(2, float64(height)) - 1)

	for i, level := range IF_BSTNodes {
		if i == 0 {
			paddingStr := strings.Repeat(" ", (padding/2)+1)
			b.WriteString(paddingStr)
		} else {
			paddingStr := strings.Repeat(" ", padding/2)
			b.WriteString(paddingStr)
		}

		for j, IF_BSTNode := range level {
			b.WriteString(IF_BSTNode)
			if j != len(level)-1 {
				b.WriteString(strings.Repeat(" ", padding))
			}
		}

		padding /= 2
		b.WriteString("\n")
	}

	return b.String()
}

// Initialize a new binary search tree with the given initial value.
func NewInterfaced[T Comparable[T]](initial T) *InterfacedBST[T] {
	return &InterfacedBST[T]{
		root: &IF_BSTNode[T]{value: initial}}
}

// Insert a value into the binary search tree.
func (t *InterfacedBST[T]) Insert(value T) (inserted bool) {
	if t.root == nil {
		t.root = &IF_BSTNode[T]{value: value}
		t.len++
		return true
	}
	inserted = t.root.insert(value)
	if inserted {
		t.len++
	}
	return inserted
}

// Search for, and return, a value in the binary search tree.
func (t *InterfacedBST[T]) Search(value T) (v T, ok bool) {
	if t.root == nil {
		return
	}
	return t.root.search(value)
}

// Delete a value from the binary search tree.
func (t *InterfacedBST[T]) Delete(value T) (deleted bool) {
	if t.root == nil {
		return false
	}
	t.root, deleted = t.root.delete(value)
	if deleted {
		t.len--
	}
	return deleted
}

// Delete all values from the binary search tree that match the given predicate.
func (t *InterfacedBST[T]) DeleteIf(predicate func(T) bool) (deleted int) {
	if t.root == nil {
		return 0
	}
	t.root, deleted = t.root.deleteIf(predicate)
	t.len -= int(deleted)
	return deleted
}

// Traverse the binary search tree in-order.
func (t *InterfacedBST[T]) Traverse(f func(T) bool) bool {
	if t.root == nil {
		return false
	}
	return t.root.traverse(f)
}

// Return the number of values in the binary search tree.
func (t *InterfacedBST[T]) Len() int {
	return t.len
}

// Return the height of the binary search tree.
func (t *InterfacedBST[T]) Height() int {
	if t.root == nil {
		return 0
	}
	return t.root.getHeight()
}

// Clear the binary search tree.
func (t *InterfacedBST[T]) Clear() {
	t.root = nil
	t.len = 0
}

func fillIF_BSTNodes[T Comparable[T]](IF_BSTNodes [][]string, n *IF_BSTNode[T], depth int) {
	if n == nil {
		return
	}

	IF_BSTNodes[depth] = append(IF_BSTNodes[depth], fmt.Sprintf("%v", n.value))
	fillIF_BSTNodes(IF_BSTNodes, n.left, depth+1)
	fillIF_BSTNodes(IF_BSTNodes, n.right, depth+1)
}

// Create a new binary search tree from an array.
func SliceToInterfacedBST[T Comparable[T]](items []T, sorted bool) *InterfacedBST[T] {
	if !sorted {
		slices.SortFunc(items, func(i, j T) int {
			var ltA, ltB = i.Lt(j), j.Lt(i)
			if ltA && !ltB {
				return -1
			}
			if !ltA && ltB {
				return 1
			}
			return 0
		})
	}
	var bst InterfacedBST[T]
	bst.root = constructInterfacedBSTFromSortedSlice(items, 0, len(items))
	return &bst
}

func constructInterfacedBSTFromSortedSlice[T Comparable[T]](items []T, start, end int) *IF_BSTNode[T] {
	if start == end {
		return nil
	}
	mid := start + (end-start)/2
	return &IF_BSTNode[T]{
		value: items[mid],
		left:  constructInterfacedBSTFromSortedSlice(items, start, mid),
		right: constructInterfacedBSTFromSortedSlice(items, mid+1, end),
	}
}
