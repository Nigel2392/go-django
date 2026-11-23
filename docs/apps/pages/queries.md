# Page Queries and Tree Utilities

The `pages` package builds on Go-Django's [`queries`](../../../queries/docs/queryset/queryset.md) ORM with two specialized queryset wrappers—`PageQuerySet` and `SpecificPageQuerySet`—plus package-level tree repair and inspection utilities.

## Constructing a `PageQuerySet`

```go
qs := pages.NewPageQuerySet().WithContext(ctx)
```

Because `PageQuerySet` wraps `queries.QuerySet[*PageNode]`, standard queryset chainers such as `Filter`, `Select`, `OrderBy`, `Offset`, `Limit`, `Count`, `Get`, `First`, and `All` work directly, and calling `.Base()` unwraps the underlying `*queries.QuerySet[*PageNode]`.

## Chainers

- `Specific() *SpecificPageQuerySet`: Switches the queryset to return and operate on concrete [`Page`](./contenttypes.md#the-page-interface) instances.
- `StatusFlags(statusFlags StatusFlag) *PageQuerySet`: Filters nodes matching `statusFlags` via bitwise AND (`StatusFlags__bitand`).
- `Published() *PageQuerySet`: Filters to nodes with `StatusFlagPublished` set.
- `Unpublished() *PageQuerySet`: Filters to nodes without `StatusFlagPublished` set.
- `RootPages() *PageQuerySet`: Filters to root nodes (`Depth == 0`).
- `Types(types ...any) *PageQuerySet`: Filters by `ContentType`; each element may be a `string` (type name or alias), an `attrs.Definer`, or a `contenttypes.ContentType`.
- `Ancestors(path string, depth int64, inclusive ...bool) *PageQuerySet`: Filters to ancestors of the node at `path`/`depth` (includes the node itself if `inclusive` is `true`).
- `Descendants(path string, depth int64, inclusive ...bool) *PageQuerySet`: Filters to descendants of `path`/`depth` (includes the node itself if `inclusive` is `true`).
- `Children(path string, depth int64) *PageQuerySet`: Filters to direct children (`Depth == depth+1`) under `path`.
- `Siblings(path string, depth int64, inclusive ...bool) *PageQuerySet`: Filters to nodes sharing the same parent path and `depth` (excludes `path` unless `inclusive` is `true`).
- `AncestorOf(node *PageNode, inclusive ...bool) *PageQuerySet`: Shorthand for `Ancestors(node.Path, node.Depth, inclusive...)`.
- `DescendantOf(node *PageNode, inclusive ...bool) *PageQuerySet`: Shorthand for `Descendants(node.Path, node.Depth, inclusive...)`.
- `ChildrenOf(node *PageNode) *PageQuerySet`: Shorthand for `Children(node.Path, node.Depth)`.
- `SiblingsOf(node *PageNode, inclusive ...bool) *PageQuerySet`: Shorthand for `Siblings(node.Path, node.Depth, inclusive...)`.
- `Search(query string) *PageQuerySet`: Executes cross-model search across `PageNode` and all registered `PageDefinition` models implementing `search.SearchableModel`, ordering by relevance (implements `search.Searchable` via `BuildSearchQuery`).

## Readers

- `AllNodes() ([]*PageNode, error)`: Executes the queryset and returns all matching `*PageNode` pointers.
- `GetNodeByID(id int64) (*PageNode, error)`: Retrieves a single node by `PK`.
- `GetNodeByPath(path string) (*PageNode, error)`: Retrieves a single node by materialized `Path`.
- `GetNodeBySlug(slug string, depth int64, path string) (*PageNode, error)`: Retrieves a node matching `Depth`, case-insensitive `Slug`, and `Path` prefix.
- `GetNodesByDepth(depth int64, statusFlags StatusFlag, offset int32, limit int32) ([]*PageNode, error)`: Returns paginated nodes at `depth`, optionally filtered by `statusFlags` (ignored if `0`) and ordered by `Path`.
- `GetNodesByIDs(id []int64) ([]*PageNode, error)`: Returns all nodes matching `PK__in`.
- `GetChildNodes(node *PageNode, statusFlags StatusFlag, offset int32, limit int32) ([]*PageNode, error)`: Returns paginated direct children of `node` ordered by `Path` (filters by `statusFlags` when non-zero).
- `GetDescendants(path string, depth int64, statusFlags StatusFlag, offset int32, limit int32) ([]*PageNode, error)`: Returns paginated descendants of `path`/`depth` ordered by `Path` (filters by `statusFlags` when non-zero).
- `GetAncestors(p string, depth int64) ([]*PageNode, error)`: Returns all ancestor nodes of `p`/`depth`.
- `ParentNode(path string, depth int) (*PageNode, error)`: Returns the parent node for `path` at `depth` (returns `ErrPageIsRoot` when `depth == 0`).

## Mutators

- `AddRoot(node *PageNode) error` / `AddRoots(nodes ...*PageNode) error`: Validates empty `Path`, non-empty `Title`, and `DisallowRoot == false`; computes root `Path`, `Depth = 0`, and `UrlPath`; saves any attached `PageObject`; inserts nodes in a transaction; and emits `SignalRootCreated`.
- `AddChildren(parent *PageNode, children ...*PageNode) error`: Validates non-empty `parent.Path`, empty child `Path`, non-empty `Title`, and `ParentPageTypes`/`ChildPageTypes` constraints; assigns `Path`, `Depth`, and `UrlPath`; saves attached `PageObject`s; increments `parent.Numchild`; and emits `SignalChildCreated`.
- `UpdateNode(node *PageNode) error`: Saves attached `PageObject`, recalculates `UrlPath` and descendant `UrlPath` values if `Slug` changed, updates the node in a transaction, and emits `SignalNodeUpdated`.
- `Delete(nodes ...*PageNode) (int64, error)`: When no `WHERE` clause is set on the queryset, runs inside a transaction to gather `nodes` and all descendants, emits `SignalNodeBeforeDelete` for each node, deletes associated concrete `Page` records by `ContentType`, deletes the `PageNode` rows, and decrements `Numchild` on affected parents. (If the queryset already has a `WHERE` clause, `nodes` must be empty and `Base().Delete()` runs directly.)
- `MoveNode(node *PageNode, newParent *PageNode) error`: Prevents moving root nodes, moving onto self, or moving into a descendant; enforces `ParentPageTypes`/`ChildPageTypes`; rewrites `Path`, `Depth`, and `UrlPath` for `node` and up to 1000 descendants; updates `Numchild` on old and new parents; and emits `SignalNodeMoved`.
- `PublishNode(node *PageNode) error`: Sets `StatusFlagPublished` on `node` and updates `StatusFlags` and `UpdatedAt`.
- `UnpublishNode(node *PageNode, unpublishChildren bool) error`: Clears `StatusFlagPublished` on `node` (and on published descendants when `unpublishChildren` is `true`).

## Panicking Raw Write Overrides

To prevent bypassing materialized path, depth, `Numchild`, and `UrlPath` maintenance, `PageQuerySet` overrides raw write methods so calling them directly **panics**:

- `Create(*PageNode) (*PageNode, error)`
- `Update(*PageNode, ...any) (int64, error)`
- `BulkCreate([]*PageNode) ([]*PageNode, error)`
- `BatchCreate([]*PageNode) ([]*PageNode, error)`
- `BulkUpdate(...any) (int64, error)`
- `BatchUpdate(...any) (int64, error)`

Use `AddRoot`, `AddRoots`, `AddChildren`, or `UpdateNode` instead, or call `qs.Base()` explicitly when performing low-level `queries.QuerySet[*PageNode]` writes (such as partial field updates).

## `SpecificPageQuerySet`

Calling `qs.Specific()` returns a `*SpecificPageQuerySet` that queries `PageNode` rows and batch-loads their concrete `Page` records per `ContentType` into `queries.Rows[Page]`:

```go
rows, err := pages.NewPageQuerySet().
	WithContext(ctx).
	Published().
	Specific().
	All()
```

### Chainers and Readers

- Chainers: `StatusFlags`, `Published`, `Unpublished`, `Types`, `AncestorOf`, `DescendantOf`, `ChildrenOf`, `SiblingsOf`, `Search` (plus embedded `WrappedQuerySet` methods such as `Filter`, `OrderBy`, `Offset`, `Limit`, `WithContext`).
- Readers: `All() (queries.Rows[Page], error)`, `Get() (*queries.Row[Page], error)`, `First() (*queries.Row[Page], error)`, `Last() (*queries.Row[Page], error)`.

### Mutators

- `AddRoot(page Page) error` / `AddRoots(pages ...Page) error`
- `AddChildren(parent Page, children ...Page) error`
- `DeletePage(page Page) error`
- `MovePage(page Page, newParent Page) error`
- `PublishPage(page Page) error`
- `UnpublishPage(page Page, unpublishChildren bool) error`

> [!WARNING]
> **`SpecificPageQuerySet.AddChildren` bug**: In `contrib/pages/queryset_page.go`, `SpecificPageQuerySet.AddChildren` prepares each child node inside its loop without appending it to `childNodes`, so `AddChildren` always passes an empty slice and fails with `"no children provided"`. As a workaround, call `PageQuerySet.AddChildren(parent.Reference(), childRef)` after calling `childRef.SetSpecificPageObject(child)`.

## Package-Level Tree Utilities

Tree inspection and repair helpers live at the `pages` package level (not on `PageQuerySet`):

- `pages.FixTree(ctx context.Context) error`: Loads all `PageNode` records inside a transaction, temporarily prefixes every `Path` with `~` to avoid unique-index collisions, rebuilds the hierarchy via `NewNodeTree`, repairs `Path`, `Depth`, `Numchild`, and `UrlPath` via `(*Node).FixTree()`, and bulk-updates all nodes.
- `pages.NewNodeTree(refs []*PageNode) *Node`: Constructs an in-memory tree of `*Node` (`Ref *PageNode`, `Children *orderedmap.OrderedMap[string, *Node]`) sorted by `Path`.
- `pages.PrintTree(node *Node, depth int)`: Prints indented node paths to stdout for debugging.
- `(*Node).FixTree() []*PageNode`: Repairs `Depth`, `UrlPath`, `Numchild`, and `Path` in memory and returns modified nodes.
- `(*Node).FindNode(path string) *Node`: Finds the `*Node` matching `path`.
- `(*Node).ForEach(fn func(*Node, int64) (cancel bool)) (cancelled bool)` and `(*Node).FlatList() []*PageNode`.

> [!NOTE]
> **`Node.ForEach` and `Node.FlatList` caveat**: `ForEach` calls its recursive helper `forEach` with `execForRoot = false`, and `forEach` propagates `false` at every level. As a result, `fn` is never invoked for child or descendant nodes—and on a root container from `NewNodeTree` (where `n.Ref == nil`), `ForEach` visits zero nodes and `FlatList()` returns an empty slice.
