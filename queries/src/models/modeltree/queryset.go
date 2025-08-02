package modeltree

import (
	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/queries/src/expr"
)

var (
	// _ queries.QuerySetCanBeforeExec = (*NodeQuerySet)(nil)
	_ queries.QuerySetCanClone[Node, *NodeQuerySet[Node], *queries.QuerySet[Node]] = (*NodeQuerySet[Node])(nil)
)

type NodeQuerySet[T Node] struct {
	*queries.WrappedQuerySet[T, *NodeQuerySet[T], *queries.QuerySet[T]]
}

func variableBool(inclusive ...bool) bool {
	if len(inclusive) > 0 {
		return inclusive[0]
	}
	return false
}

func NewNodeQuerySet[T Node](forObj T) *NodeQuerySet[T] {
	var nodeQuerySet = &NodeQuerySet[T]{}
	nodeQuerySet.WrappedQuerySet = queries.WrapQuerySet(
		queries.GetQuerySet(forObj),
		nodeQuerySet,
	)
	return nodeQuerySet
}

func (qs *NodeQuerySet[T]) CloneQuerySet(wrapped *queries.WrappedQuerySet[T, *NodeQuerySet[T], *queries.QuerySet[T]]) *NodeQuerySet[T] {
	return &NodeQuerySet[T]{
		WrappedQuerySet: wrapped,
	}
}

func (qs *NodeQuerySet[T]) RootNodes() *NodeQuerySet[T] {
	return qs.Filter("Depth", 0)
}

func (qs *NodeQuerySet[T]) Ancestors(path string, depth int64, inclusive ...bool) *NodeQuerySet[T] {
	depth++

	var incl = variableBool(inclusive...)
	var paths = make([]string, depth)
	var start = 0
	if !incl {
		start = 1
	}
	for i := start; i < int(depth); i++ {
		var path, err = ParentPath(
			path, int64(i),
		)
		if err != nil {
			panic(errors.Wrapf(
				err, "failed to get ancestor path for %s at depth %d",
				path, i,
			))
		}
		paths[i] = path
	}

	return qs.Filter("Path__in", paths)
}

func (qs *NodeQuerySet[T]) Descendants(path string, depth int64, inclusive ...bool) *NodeQuerySet[T] {
	var incl = variableBool(inclusive...)
	var exp = expr.And(
		expr.Expr("Path", expr.LOOKUP_STARTSWITH, path),
		expr.Expr("Depth", expr.LOOKUP_GT, depth),
	)

	if incl {
		exp = expr.Or(
			exp,
			expr.Expr("Path", expr.LOOKUP_EXACT, path),
		)
	}

	return qs.Filter(exp)
}

func (qs *NodeQuerySet[T]) Children(path string, depth int64) *NodeQuerySet[T] {
	return qs.Filter(
		expr.Expr("Path", expr.LOOKUP_STARTSWITH, path),
		expr.Expr("Depth", expr.LOOKUP_EXACT, depth+1),
	)
}

func (qs *NodeQuerySet[T]) Siblings(path string, depth int64, inclusive ...bool) *NodeQuerySet[T] {
	var incl = variableBool(inclusive...)
	var parentPath, err = ParentPath(path, 1)
	if err != nil {
		panic(errors.Wrapf(err, "failed to get parent path for %s", path))
	}

	qs = qs.Filter(
		expr.Expr("Path", expr.LOOKUP_STARTSWITH, parentPath),
		expr.Expr("Depth", expr.LOOKUP_EXACT, depth),
	)

	if !incl {
		qs = qs.Filter(
			expr.Expr("Path", expr.LOOKUP_EXACT, path).Not(true),
		)
	}

	return qs
}

func (qs *NodeQuerySet[T]) AncestorOf(treeNode Node, inclusive ...bool) *NodeQuerySet[T] {
	var node = treeNode.Node()
	return qs.Ancestors(node.Path, node.Depth, inclusive...)
}

func (qs *NodeQuerySet[T]) DescendantOf(treeNode Node, inclusive ...bool) *NodeQuerySet[T] {
	var node = treeNode.Node()
	return qs.Descendants(node.Path, node.Depth, inclusive...)
}

func (qs *NodeQuerySet[T]) ChildrenOf(treeNode Node) *NodeQuerySet[T] {
	var node = treeNode.Node()
	return qs.Children(node.Path, node.Depth)
}

func (qs *NodeQuerySet[T]) SiblingsOf(treeNode Node, inclusive ...bool) *NodeQuerySet[T] {
	var node = treeNode.Node()
	return qs.Siblings(node.Path, node.Depth, inclusive...)
}
