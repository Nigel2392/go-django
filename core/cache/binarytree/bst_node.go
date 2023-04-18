package binarytree

type BSTNode[T Comparable[T]] struct {
	value T
	left  *BSTNode[T]
	right *BSTNode[T]
}

func (n *BSTNode[T]) Value() T {
	return n.value
}

func (n *BSTNode[T]) insert(v T) (inserted bool) {
	if n.value.Lt(v) {
		if n.right == nil {
			n.right = &BSTNode[T]{value: v}
			return true
		} else {
			return n.right.insert(v)
		}
	} else if n.value.Gt(v) {
		if n.left == nil {
			n.left = &BSTNode[T]{value: v}
			return true
		} else {
			return n.left.insert(v)
		}
	}

	n.value = v

	return false
}

func (n *BSTNode[T]) search(value T) (v T, ok bool) {
	// if we've reached the end of the tree, the value is not present
	if n == nil {
		return
	}

	// check if we need to traverse further down the tree
	if n.value.Lt(value) {
		return n.right.search(value)
	} else if n.value.Gt(value) {
		return n.left.search(value)
	}

	// value is not less than or greater than, so it must be equal
	return n.value, true
}

func (n *BSTNode[T]) traverse(f func(T)) {
	if n == nil {
		return
	}

	n.left.traverse(f)
	f(n.value)
	n.right.traverse(f)
}

func (n *BSTNode[T]) delete(v T) (newRoot *BSTNode[T], deleted bool) {
	if n == nil {
		return nil, false
	}

	if v.Lt(n.value) {
		n.left, deleted = n.left.delete(v)
	} else if v.Gt(n.value) {
		n.right, deleted = n.right.delete(v)
	} else {
		deleted = true
		if n.left == nil {
			return n.right, deleted
		} else if n.right == nil {
			return n.left, deleted
		}

		minRight := n.right.findMin()
		minRight.right, _ = n.right.delete(minRight.value)
		minRight.left = n.left
		return minRight, deleted
	}

	return n, deleted
}

func (n *BSTNode[T]) deleteIf(predicate func(T) bool) (newRoot *BSTNode[T], deleted int) {
	if n == nil {
		return nil, 0
	}

	n.left, deleted = n.left.deleteIf(predicate)
	n.right, deleted = n.right.deleteIf(predicate)

	if predicate(n.value) {
		deleted++
		if n.left == nil {
			return n.right, deleted
		} else if n.right == nil {
			return n.left, deleted
		}

		minRight := n.right.findMin()
		minRight.right, _ = n.right.delete(minRight.value)
		minRight.left = n.left
		return minRight, deleted
	}

	return n, deleted
}

func (n *BSTNode[T]) findMin() *BSTNode[T] {
	current := n
	for current.left != nil {
		current = current.left
	}
	return current
}

func (n *BSTNode[T]) getHeight() int {
	if n == nil {
		return 0
	}

	leftHeight := n.left.getHeight()
	rightHeight := n.right.getHeight()

	if leftHeight > rightHeight {
		return leftHeight + 1
	}

	return rightHeight + 1
}
