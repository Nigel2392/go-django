# Page Signals

The `pages` app emits lifecycle signals using [go-signals](https://github.com/Nigel2392/go-signals) whenever nodes in the page tree are created, updated, moved, or deleted.

## Signal Payload Types

```go
type BaseSignal struct {
	// Ctx stores the context used to perform the operation.
	Ctx context.Context
}

type PageNodeSignal struct {
	BaseSignal
	Node   *PageNode
	Nodes  []*PageNode
	PageID int64
}

type PageSignal struct {
	BaseSignal
	Page Page
}

type PageMovedSignal struct {
	BaseSignal
	Node      *PageNode
	Nodes     []*PageNode
	NewParent *PageNode
	OldParent *PageNode
}
```

Note that `PageSignal` is defined in `contrib/pages`, but no predefined signal currently uses or emits it.

## Predefined Signals

| Variable | Signal Name | Payload Type | Populated Fields |
| --- | --- | --- | --- |
| `SignalRootCreated` | `"pages.root_page_created"` | `*PageNodeSignal` | `Ctx`, `Nodes` (`[]*PageNode`); `Node` is `nil`, `PageID` is `0` |
| `SignalChildCreated` | `"pages.child_page_created"` | `*PageNodeSignal` | `Ctx`, `Nodes` (`[]*PageNode`), `PageID` (parent's `PageID`); `Node` is `nil` |
| `SignalNodeUpdated` | `"pages.node_updated"` | `*PageNodeSignal` | `Ctx`, `Node`, `PageID`; `Nodes` is `nil` |
| `SignalNodeBeforeDelete` | `"pages.node_before_delete"` | `*PageNodeSignal` | `Ctx`, `Node`, `PageID` (once per node and descendant; `Nodes` is `nil`) |
| `SignalNodeMoved` | `"pages.node_moved"` | `*PageMovedSignal` | `Ctx`, `Node`, `Nodes` (descendants), `OldParent`, `NewParent` |

> [!IMPORTANT]
> `SignalRootCreated` and `SignalChildCreated` populate **`Nodes`** (`[]*PageNode`), **not `Node`**. Always iterate `pns.Nodes` when handling creation signals.

Returning a non-nil error from a `SignalNodeBeforeDelete` listener rolls back the delete transaction and aborts deletion.

## Listening to Signals

Register listeners using `signals.Signal[T].Listen(ctx, fn)` where the callback receives `(ctx context.Context, s signals.Signal[T], v T) error`:

```go
package main

import (
	"context"
	"log"

	"github.com/Nigel2392/go-django/contrib/pages"
	"github.com/Nigel2392/go-signals"
)

func registerPageSignals(ctx context.Context) {
	// SignalChildCreated populates pns.Nodes (not pns.Node).
	pages.SignalChildCreated.Listen(ctx, func(ctx context.Context, s signals.Signal[*pages.PageNodeSignal], pns *pages.PageNodeSignal) error {
		for _, node := range pns.Nodes {
			log.Printf("Created child page %q (%s)", node.Title, node.UrlPath)
		}
		return nil
	})

	// SignalNodeUpdated populates pns.Node (and pns.PageID).
	pages.SignalNodeUpdated.Listen(ctx, func(ctx context.Context, s signals.Signal[*pages.PageNodeSignal], pns *pages.PageNodeSignal) error {
		log.Printf("Updated page %q (page_id=%d)", pns.Node.Title, pns.PageID)
		return nil
	})
}
```
