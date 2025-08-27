package mixins

import (
	"errors"
	"iter"
)

type Definer[T any] interface {
	Mixins() []T
}

func Mixins[T any](obj T, topDown bool) iter.Seq2[T, int] {
	return func(yield func(T, int) bool) {
		iterMixins(yield, obj, topDown, 0)
	}
}

func iterMixins[T any](yield func(T, int) bool, obj T, topDown bool, depth int) bool {
	if topDown && !yield(obj, depth) {
		return false
	}
	if mixin, ok := any(obj).(Definer[T]); ok {
		for _, m := range mixin.Mixins() {
			if !iterMixins(yield, m, topDown, depth+1) {
				return false
			}
		}
	}
	if !topDown && !yield(obj, depth) {
		return false
	}
	return true
}

type MixinTree[T any] struct {
	Root  T
	Nodes []*MixinTree[T]
}

var ErrStopTreeIter = errors.New("stop tree iteration")

func BuildMixinTree[T any](obj T) *MixinTree[T] {
	var tree = &MixinTree[T]{
		Root: obj,
	}
	if mixin, ok := any(obj).(Definer[T]); ok {
		var mixins = mixin.Mixins()
		tree.Nodes = make([]*MixinTree[T], 0, len(mixins))
		for _, m := range mixins {
			tree.Nodes = append(tree.Nodes, BuildMixinTree(m))
		}
	}
	return tree
}

func (t *MixinTree[T]) ForEach(topDown bool, fn func(node *MixinTree[T], depth int) error) error {
	var treeErr error
	if topDown {
		if err := fn(t, 0); err != nil {
			treeErr = err
			goto checkTreeErr
		}
	}
	for _, child := range t.Nodes {
		var c, err = child.forEach(topDown, fn, 1)
		if err != nil {
			treeErr = err
			goto checkTreeErr
		}
		if !c {
			treeErr = ErrStopTreeIter
			goto checkTreeErr
		}
	}
	if !topDown {
		if err := fn(t, 0); err != nil {
			treeErr = err
			goto checkTreeErr
		}
	}

checkTreeErr:
	if treeErr != nil && errors.Is(treeErr, ErrStopTreeIter) {
		return nil
	}

	return treeErr
}

func (t *MixinTree[T]) forEach(topDown bool, fn func(*MixinTree[T], int) error, depth int) (bool, error) {
	if topDown {
		if err := fn(t, depth); err != nil {
			return false, err
		}
	}
	for _, child := range t.Nodes {
		var c, err = child.forEach(topDown, fn, depth+1)
		if err != nil {
			return false, err
		}
		if !c {
			return false, nil
		}
	}
	if !topDown {
		if err := fn(t, depth); err != nil {
			return false, err
		}
	}
	return true, nil
}
