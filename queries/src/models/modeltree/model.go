package modeltree

import (
	"context"
	"strings"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/queries/src/models"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/trans"
)

const ROOT_NODES_KEY = "modeltree.root_nodes"

var (
	_ queries.ActsBeforeSave = (*TreeNode)(nil)
)

type Node interface {
	attrs.Definer
	Node() *TreeNode
}

type TreeNode struct {
	models.Model
	Path     string `json:"path" attrs:"blank"`
	Depth    int64  `json:"depth" attrs:"blank"`
	Numchild int64  `json:"numchild" attrs:"blank"`

	// Object is the object that this node represents.
	// It is used to store the actual data of the node.
	// It is used to cache the object so that it can be reused
	// without having to query the database again.
	Object Node `json:"object" attrs:"-"`

	// Parent is used to cache the parent node
	// If it is not nil and the node is being created,
	// it will be used to set the path and depth of the node.
	Parent *TreeNode `json:"-" attrs:"-"`
}

func (n *TreeNode) BindToEmbedder(embedder attrs.Definer) error {
	n.Object = embedder.(Node)
	return nil
}

func (n *TreeNode) Node() *TreeNode {
	return n
}

func (n *TreeNode) Validate(ctx context.Context) error {
	if err := n.Model.Validate(ctx); err != nil {
		return err
	}

	if n.Depth == 0 && n.Parent == nil && n.Path != "" {
		var validatorContextVal = ctx.Value(queries.ValidationContext{})
		var validatorCtx, ok = validatorContextVal.(*queries.ValidationContext)
		if !ok {
			return errors.TypeMismatch.Wrapf(
				"expected %T, got %T (%v)",
				&queries.ValidationContext{},
				validatorContextVal,
				validatorContextVal,
			)
		}

		var rootCount *int64
		val, ok := validatorCtx.Data[ROOT_NODES_KEY]
		if !ok {
			var nodeCount, err = NewNodeQuerySet(n.Object).
				WithContext(ctx).
				RootNodes().
				Count()
			if err != nil {
				return errors.Wrapf(
					err, "failed to count root nodes for %T", n.Object,
				)
			}

			rootCount = &nodeCount

			validatorCtx.SetValue(
				ROOT_NODES_KEY,
				rootCount,
			)
		} else {
			rootCount = val.(*int64)
		}

		var pathSuffix = BuildNextPathPart(*rootCount)
		if !strings.HasSuffix(n.Path, pathSuffix) {
			return errors.ValueError.Wrapf(
				"invalid path %s for root node, expected %s",
				n.Path, pathSuffix,
			)
		}

		*rootCount++
	}

	if n.Path == "" {
		return errors.ValueError.Wrap("path cannot be empty")
	}

	if n.Depth < 0 {
		return errors.ValueError.Wrapf(
			"invalid depth %d for path %s, expected >= 0",
			n.Depth, n.Path,
		)
	}

	if n.Depth != (int64(len(n.Path))/STEP_LEN)-1 {
		return errors.ValueError.Wrapf(
			"invalid path length %d for path %s, expected %d",
			len(n.Path), n.Path, n.Depth*STEP_LEN,
		)
	}

	return nil
}

func (n *TreeNode) BeforeSave(ctx context.Context) error {

	if n.Parent != nil && n.Path == "" {
		n.Path = n.Parent.Path + BuildNextPathPart(n.Parent.Numchild)
		n.Depth = n.Parent.Depth + 1
	}

	return nil
}

func (n *TreeNode) TreeNodeFields(def attrs.Definer) []attrs.Field {
	return []attrs.Field{
		attrs.NewField(def, "Path", &attrs.FieldConfig{
			Label:    trans.S("Path"),
			HelpText: trans.S("The path of the node in the tree, used to determine its position."),
		}),
		attrs.NewField(def, "Depth", &attrs.FieldConfig{
			Blank:    true,
			Label:    trans.S("Depth"),
			HelpText: trans.S("The depth of the node in the tree."),
		}),
		attrs.NewField(def, "Numchild", &attrs.FieldConfig{
			Blank:    true,
			Label:    trans.S("Numchild"),
			HelpText: trans.S("The number of children of the node."),
		}),
	}
}

func (n *TreeNode) Define(def attrs.Definer, fields ...any) attrs.Definitions {
	n.Object = def.(Node)
	return n.Model.Define(def, fields...)
}

func (n *TreeNode) AddChild(ctx context.Context, child Node, save ...bool) error {
	var childNode = child.Node()
	childNode.Parent = n
	childNode.Depth = n.Depth + 1
	childNode.Path = n.Path + BuildNextPathPart(n.Numchild)
	n.Numchild++

	if len(save) > 0 && save[0] {
		return childNode.Save(ctx)
	}
	return nil
}
