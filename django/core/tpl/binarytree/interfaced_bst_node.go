package binarytree

type IF_BSTNode[T Comparable[T]] struct {
	value T
	left  *IF_BSTNode[T]
	right *IF_BSTNode[T]
}

func (n *IF_BSTNode[T]) Value() T {
	return n.value
}

func (n *IF_BSTNode[T]) insert(v T) (inserted bool) {
	// Equals
	if n.value.Lt(v) {
		if n.right == nil {
			n.right = &IF_BSTNode[T]{value: v}
			return true
		} else {
			return n.right.insert(v)
		}
	} else if v.Lt(n.value) { // Gt(
		if n.left == nil {
			n.left = &IF_BSTNode[T]{value: v}
			return true
		} else {
			return n.left.insert(v)
		}
	} else if !n.value.Lt(v) && !v.Lt(n.value) {
		n.value = v
	}
	return false
}

func (n *IF_BSTNode[T]) search(value T) (v T, ok bool) {
	// if we've reached the end of the tree, the value is not present
	if n == nil {
		return
	}
	// check if we need to traverse further down the tree
	if n.value.Lt(value) {
		return n.right.search(value)
	} else if value.Lt(n.value) {
		return n.left.search(value)
	} else if !n.value.Lt(value) && !value.Lt(n.value) {
		// value is not less than or greater than, so it must be equal
		return n.value, true
	}
	return
}

func (n *IF_BSTNode[T]) traverse(f func(T) bool) bool {
	if n == nil {
		return true
	}

	if !n.left.traverse(f) {
		return false
	}
	if !f(n.value) {
		return false
	}
	if !n.right.traverse(f) {
		return false
	}

	return true
}

func (n *IF_BSTNode[T]) delete(v T) (newRoot *IF_BSTNode[T], deleted bool) {
	if n == nil {
		return nil, false
	}

	if n.value.Lt(v) {
		n.right, deleted = n.right.delete(v)
	} else if v.Lt(n.value) {
		n.left, deleted = n.left.delete(v)
	} else if !n.value.Lt(v) && !v.Lt(n.value) {
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

func (n *IF_BSTNode[T]) deleteIf(predicate func(T) bool) (newRoot *IF_BSTNode[T], deleted int) {
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

func (n *IF_BSTNode[T]) findMin() *IF_BSTNode[T] {
	current := n
	for current.left != nil {
		current = current.left
	}
	return current
}

func (n *IF_BSTNode[T]) getHeight() int {
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
