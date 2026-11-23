# `PageNode` Model and Status Flags

`PageNode`, `StatusFlag`, and all page-tree errors live directly in package `pages` (`github.com/Nigel2392/go-django/contrib/pages`). Each `PageNode` represents one entry in the hierarchical page tree, storing tree metadata alongside optional references to a concrete `Page` model.

## `StatusFlag`

`StatusFlag` is a bitmask representing a page node's publication and visibility state:

```go
type StatusFlag int64

const (
	// Page is published.
	StatusFlagPublished StatusFlag = 1 << iota

	// Page is hidden.
	StatusFlagHidden

	// Page is marked as deleted.
	StatusFlagDeleted

	// No status flag (matches all nodes in flag-filtered queries).
	StatusFlagNone StatusFlag = 0
)
```

- `Is(flag StatusFlag) bool`: Returns `true` when all bits in `flag` are set (`f&flag == flag`).
- `Value() (driver.Value, error)`: Implements `driver.Valuer`, returning `int64(f)`.

## `PageNode`

```go
type PageNode struct {
	models.Model            `table:"PageNode"`
	PK                      int64      `json:"id" attrs:"primary;readonly;column=id"`
	Title                   string     `json:"title"`
	Path                    string     `json:"path" attrs:"blank"`
	Depth                   int64      `json:"depth" attrs:"blank;default=0"`
	Numchild                int64      `json:"numchild" attrs:"blank;default=0"`
	UrlPath                 string     `json:"url_path" attrs:"readonly;blank"`
	Slug                    string     `json:"slug"`
	StatusFlags             StatusFlag `json:"status_flags" attrs:"null;blank;default=0"`
	PageID                  int64      `json:"page_id" attrs:"null;blank"`
	ContentType             string     `json:"content_type" attrs:"null;blank"`
	PublishedAt             time.Time  `json:"published_at" attrs:"null;blank"`
	LatestRevisionCreatedAt time.Time  `json:"latest_revision_created_at" attrs:"readonly;label=Latest Revision Created At"`
	CreatedAt               time.Time  `json:"created_at" attrs:"readonly;label=Created At"`
	UpdatedAt               time.Time  `json:"updated_at" attrs:"readonly;label=Updated At"`

	PageObject Page `json:"-" attrs:"-"`

	_parent *PageNode `json:"-" attrs:"-"`
}
```

- `FieldDefs(ctx context.Context) attrs.Definitions`: Defines all fields for `PageNode` using `models.Model.Define`.

## `PageNode` Methods

### Identity, Status, and Path Helpers

- `ID() int64`: Returns `n.PK`.
- `Reference() *PageNode`: Returns `n` itself, implementing `pages.Page`.
- `IsRoot() bool`: Returns `true` when `n.Depth == 0`.
- `IsPublished() bool`: Returns `true` if `StatusFlagPublished` is set.
- `IsHidden() bool`: Returns `true` if `StatusFlagHidden` is set.
- `IsDeleted() bool`: Returns `true` if `StatusFlagDeleted` is set.
- `SetUrlPath(parent *PageNode) (newPath, oldPath string)`: Auto-generates `Slug` from `Title` if `Slug` is empty, computes `UrlPath` from `parent.UrlPath` (or `"/"` for root nodes) and `Slug`, and returns the new and previous `UrlPath`.
- `SetSpecificPageObject(p Page)`: Caches the concrete `Page` object on `n.PageObject` (clears it if `p == nil`).

### Tree Queries and Routing

- `Route(r *http.Request, pathComponents []string) (Page, error)`: Resolves URL path segments recursively across child nodes (returning `ErrRouteNotFound` if unpublished, hidden, deleted, or missing), delegating to concrete `RoutablePage` implementations when present (ignoring promoted `Route` methods).
- `Ancestors(inclusive ...bool) *PageQuerySet`: Returns a queryset scoped to `n`'s ancestors.
- `Descendants(inclusive ...bool) *PageQuerySet`: Returns a queryset scoped to `n`'s descendants.
- `Children() *PageQuerySet`: Returns a queryset scoped to `n`'s immediate children.
- `Specific(ctx context.Context, refresh ...bool) (Page, error)`: Loads and caches the concrete `Page` via `pages.Specific(ctx, n, true)` (pass `true` for `refresh` to bypass cached `PageObject`).
- `Parent(ctx context.Context, refresh ...bool) (*PageNode, error)`: Returns the parent `*PageNode` (or `nil, nil` for root nodes), caching the result in `_parent` unless `refresh` is `true`.

### Tree Mutations and Revisions

- `AddChild(ctx context.Context, child *PageNode) error`: Appends `child` beneath `n` via `PageQuerySet.AddChildren` and sets `child._parent = n`.
- `Move(ctx context.Context, newParent *PageNode) error`: Moves `n` and its descendants under `newParent` via `PageQuerySet.MoveNode` and updates `_parent`.
- `Publish(ctx context.Context) error`: Publishes `n` via `PageQuerySet.PublishNode`.
- `Unpublish(ctx context.Context, unpublishChildren bool) error`: Unpublishes `n` (and optionally all descendants) via `PageQuerySet.UnpublishNode`.
- `CreateRevision(ctx context.Context, user users.User, save bool) (*revisions.Revision, error)`: Builds a `revisions.Revision` for `n`, updates `LatestRevisionCreatedAt`, and persists the revision via `revisions.CreateDatedRevision` when `save` is `true`.

### ORM, Search, Migrations, and Revision Hooks

- `SaveObject(ctx context.Context, cnf models.SaveConfig) error` and `DeleteObject(ctx context.Context) error` (along with `ControlsEmbedderSaving() bool`): Delegate ORM saves (`AddRoot` when `PK == 0 || cnf.ForceCreate`, otherwise `UpdateNode`) and deletes through `PageQuerySet`.
- `DatabaseIndexes(obj attrs.Definer) []migrator.Index`: Defines unique indexes on `Path`, `UrlPath`, and `(PageID, ContentType)`, plus non-unique indexes on `PageID` and `ContentType`.
- `SearchableFields() []search.SearchField`: Exposes weighted `icontains` search fields for `Title` (weight 5), `UrlPath` (4), `Slug` (2), and `ContentType` (1).
- `GetRevisionInfo(ctx context.Context) (pk any, contentType string, err error)`: Returns `(n.PageID, n.ContentType, nil)` for typed pages or `(n.PK, pageNodeTypeName, nil)` for untyped nodes.
- `ExcludeFromRevisionData() []string`: Excludes `"Path"`, `"Depth"`, `"StatusFlags"`, `"PageID"`, `"ContentType"`, `"PublishedAt"`, `"LatestRevisionCreatedAt"`, and `"SiteSet"` from revision snapshots.
- `TargetContentTypeField() attrs.FieldDefinition` / `TargetPrimaryField() attrs.FieldDefinition`: Return the `ContentType` and `PageID` field definitions for generic relation mapping.
- `BeforeCreate(context.Context) error` / `BeforeSave(context.Context) error`: Maintain `CreatedAt` and `UpdatedAt` timestamps.

## Sentinel Errors

- `ErrNoPage` (`ErrCodeNoPage`): Returned when a `PageNode` has no associated concrete `Page`.
- `ErrNoPageID` (`ErrCodeNoPageID`, wraps `ErrNoPage`): Returned when `PageNode.PageID == 0`.
- `ErrInvalidPathLength`: Returned when a materialized `Path` is not a multiple of `STEP_LEN` (`3`).
- `ErrTooLittleAncestors`: Returned when a negative ancestor depth is requested.
- `ErrTooManyAncestors`: Returned when requesting more ancestors than the path depth allows.
- `ErrPageIsRoot`: Returned by `ParentNode` when called on a root node (`depth == 0`).
- `ErrRouteNotFound`: Alias for `mux.ErrRouteNotFound`, returned when routing fails or targets an unpublished/hidden/deleted node.
