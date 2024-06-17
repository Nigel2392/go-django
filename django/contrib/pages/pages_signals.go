package pages

import (
	"context"

	"github.com/Nigel2392/django/contrib/pages/models"
	"github.com/Nigel2392/go-signals"
)

type PageSignal struct {
	// Querier is the Querier used to perform the operation.
	//
	// This can be used to perform additional queries or operations
	Querier models.Querier

	// Ctx stores the context used to perform the operation.
	Ctx context.Context

	// The current node, a child node or parent node depending on the signal.
	Node *models.PageNode

	// The current page ID, the parent page ID or a child's page ID depending on the signal.
	PageID int64
}

var (
	signalRegistry         = signals.NewPool[*PageSignal]()
	SignalRootCreated      = signalRegistry.Get("pages.root_page_created")  // Node is the root node, PageID is zero.
	SignalChildCreated     = signalRegistry.Get("pages.child_page_created") // Node is the child being created, PageID is the parent node's ID.
	SignalNodeUpdated      = signalRegistry.Get("pages.node_updated")       // Node is the node being updated, PageID is the parent node's ID.
	SignalNodeBeforeDelete = signalRegistry.Get("pages.node_before_delete") // Node is the node being deleted, PageID is the parent node's ID.
)
