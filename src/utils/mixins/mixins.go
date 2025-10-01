package mixins

import (
	"errors"
	"iter"
)

type Definer[T any] interface {
	Mixins() []T
}

func defaultMixinsFn[T any](obj T, depth int) iter.Seq[T] {
	if mixin, ok := any(obj).(Definer[T]); ok {
		return func(yield func(T) bool) {
			for _, m := range mixin.Mixins() {
				if !yield(m) {
					return
				}
			}
		}
	}
	return nil
}

func Mixins[T any](obj T, topDown bool) iter.Seq2[T, int] {
	return func(yield func(T, int) bool) {
		iterMixins(yield, obj, topDown, 0, defaultMixinsFn)
	}
}

func MixinsFunc[T any](obj T, topDown bool, fn func(obj T, depth int) iter.Seq[T]) iter.Seq2[T, int] {
	return func(yield func(T, int) bool) {
		iterMixins(yield, obj, topDown, 0, fn)
	}
}

func iterMixins[T any](yield func(T, int) bool, obj T, topDown bool, depth int, fn func(obj T, depth int) iter.Seq[T]) bool {
	if topDown && !yield(obj, depth) {
		return false
	}
	if fn == nil {
		panic("cannot get mixins, provided function is nil")
	}
	for m := range fn(obj, depth) {
		if !iterMixins(yield, m, topDown, depth+1, fn) {
			return false
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
	return BuildMixinTreeFunc(obj, defaultMixinsFn)
}

func BuildMixinTreeFunc[T any](obj T, fn func(obj T, depth int) iter.Seq[T]) *MixinTree[T] {
	return buildMixinTreeFunc(obj, 0, fn)
}

func buildMixinTreeFunc[T any](obj T, depth int, fn func(obj T, depth int) iter.Seq[T]) *MixinTree[T] {
	var tree = &MixinTree[T]{
		Root: obj,
	}
	if fn != nil {
		var mixins = fn(obj, depth)
		tree.Nodes = make([]*MixinTree[T], 0)
		for m := range mixins {
			tree.Nodes = append(tree.Nodes, buildMixinTreeFunc(m, depth+1, fn))
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
