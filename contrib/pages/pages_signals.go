package pages

import (
	"context"

	"github.com/Nigel2392/go-signals"
)

type BaseSignal struct {
	// Ctx stores the context used to perform the operation.
	Ctx context.Context
}

type PageNodeSignal struct {
	BaseSignal

	// The current node, a child node or parent node depending on the signal.
	Node  *PageNode
	Nodes []*PageNode

	// The current page ID, the parent page ID or a child's page ID depending on the signal.
	PageID int64
}

type PageSignal struct {
	BaseSignal
	Page Page
}

type PageMovedSignal struct {
	BaseSignal

	// The node being moved.
	Node *PageNode

	// All nodes that are being moved.
	Nodes []*PageNode

	// The new parent node.
	NewParent *PageNode

	// The old parent node.
	OldParent *PageNode
}

var (
	// A signal which is sent when a new root page is created.
	SignalRootCreated = signals.Get[*PageNodeSignal]("pages.root_page_created") // Node is the root node, PageID is zero.

	// A signal which is sent when a new child page is created.
	SignalChildCreated = signals.Get[*PageNodeSignal]("pages.child_page_created") // Node is the child being created, PageID is the parent node's ID.

	// A signal which is sent when a node is updated.
	SignalNodeUpdated = signals.Get[*PageNodeSignal]("pages.node_updated") // Node is the node being updated, PageID is the parent node's ID.

	// A signal which is sent before a node is deleted.
	SignalNodeBeforeDelete = signals.Get[*PageNodeSignal]("pages.node_before_delete") // Node is the node being deleted, PageID is the parent node's ID.

	// A signal which is sent after a node is moved.
	SignalNodeMoved = signals.Get[*PageMovedSignal]("pages.node_moved") // Node is the node being moved, Parent is the new parent, OldParent is the old parent.
)
