package pages

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"time"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/queries/src/migrator"
	"github.com/Nigel2392/go-django/queries/src/models"
	"github.com/Nigel2392/go-django/src/contrib/revisions"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/mux"
	"github.com/gosimple/slug"
)

var (
	_ Page                     = (*PageNode)(nil)
	_ queries.ActsBeforeCreate = (*PageNode)(nil)
	_ queries.ActsAfterSave    = (*PageNode)(nil)
	_ models.CanControlSaving  = (*PageNode)(nil)
)

type PageNode struct {
	models.Model     `table:"PageNode"`
	PK               int64      `json:"id" attrs:"primary;readonly;column=id"`
	Title            string     `json:"title"`
	Path             string     `json:"path" attrs:"blank"`
	Depth            int64      `json:"depth" attrs:"blank"`
	Numchild         int64      `json:"numchild" attrs:"blank"`
	UrlPath          string     `json:"url_path" attrs:"readonly;blank"`
	Slug             string     `json:"slug"`
	StatusFlags      StatusFlag `json:"status_flags" attrs:"null;blank"`
	PageID           int64      `json:"page_id" attrs:"null;blank"`
	ContentType      string     `json:"content_type" attrs:"null;blank"`
	LatestRevisionID int64      `json:"latest_revision_id"`
	CreatedAt        time.Time  `json:"created_at" attrs:"readonly;label=Created At"`
	UpdatedAt        time.Time  `json:"updated_at" attrs:"readonly;label=Updated At"`

	// PageObject is the specific page object associated with this node.
	//
	// It is used to cache the specific page object for performance optimization.
	//
	// It is also used to save the specific page object when the node is saved.
	PageObject Page `json:"-" attrs:"-"`

	// _parent is used to cache the parent node
	// It is not saved to the database and is only used for performance optimization
	_parent *PageNode `json:"-" attrs:"-"`
}

func (n *PageNode) SetUrlPath(parent *PageNode) (newPath, oldPath string) {
	oldPath = n.UrlPath

	if n.Slug == "" && n.Title != "" {
		n.Slug = slug.Make(n.Title)
	}

	var bufLen = len(n.Slug)
	if parent == nil {
		bufLen++
	} else {
		bufLen += len(parent.UrlPath) + 1
	}

	var buf = make([]byte, 0, bufLen)
	if parent == nil {
		buf = append(buf, '/')
	} else {
		buf = append(buf, parent.UrlPath...)
		buf = append(buf, '/')
	}

	buf = append(buf, n.Slug...)

	n.UrlPath = string(buf)
	return n.UrlPath, oldPath
}

func (n *PageNode) SetSpecificPageObject(p Page) {
	if p == nil {
		n.PageObject = nil
		return
	}

	n.PageObject = p
}

func (n *PageNode) ID() int64 {
	return n.PK
}

func (n *PageNode) Reference() *PageNode {
	return n
}

func (n *PageNode) IsRoot() bool {
	return n.Depth == 0
}

func (n *PageNode) IsPublished() bool {
	return n.StatusFlags&StatusFlagPublished != 0
}

func (n *PageNode) IsHidden() bool {
	return n.StatusFlags&StatusFlagHidden != 0
}

func (n *PageNode) IsDeleted() bool {
	return n.StatusFlags&StatusFlagDeleted != 0
}

func (n *PageNode) BeforeCreate(context.Context) error {
	if n.CreatedAt.IsZero() {
		n.CreatedAt = time.Now()
	}

	if n.UpdatedAt.IsZero() {
		n.UpdatedAt = n.CreatedAt
	}

	return nil
}

func (n *PageNode) BeforeSave(context.Context) error {
	n.UpdatedAt = time.Now()
	return nil
}

func (n *PageNode) DatabaseIndexes(obj attrs.Definer) []migrator.Index {
	if reflect.TypeOf(obj) != reflect.TypeOf(n) {
		return nil
	}
	return []migrator.Index{
		{Fields: []string{"Path"}, Unique: true},
		{Fields: []string{"UrlPath"}, Unique: true},
		{Fields: []string{"PageID"}, Unique: false},
		{Fields: []string{"ContentType"}, Unique: false},
		{Fields: []string{"PageID", "ContentType"}, Unique: true},
	}
}

func (n *PageNode) TargetContentTypeField() attrs.FieldDefinition {
	var defs = n.FieldDefs()
	var f, _ = defs.Field("ContentType")
	return f
}

func (n *PageNode) TargetPrimaryField() attrs.FieldDefinition {
	var defs = n.FieldDefs()
	var f, _ = defs.Field("PageID")
	return f
}

func (n *PageNode) ControlsEmbedderSaving() bool {
	return true
}

func (n *PageNode) SaveObject(ctx context.Context, cnf models.SaveConfig) (err error) {
	var creating = n.PK == 0 || cnf.ForceCreate
	var querySet = NewPageQuerySet().WithContext(ctx)
	if creating {
		err = querySet.AddRoot(n)
	} else {
		err = querySet.UpdateNode(n)
	}
	if err != nil {
		return fmt.Errorf("failed to save page node: %w", err)
	}
	return nil
}

func (n *PageNode) DeleteObject(ctx context.Context) error {
	var querySet = NewPageQuerySet().WithContext(ctx)
	_, err := querySet.Delete(n)
	if err != nil {
		return fmt.Errorf("failed to delete page node: %w", err)
	}
	return nil
}

func (n *PageNode) FieldDefs() attrs.Definitions {
	return n.Model.Define(n, func(d attrs.Definer) []attrs.Field {
		return []attrs.Field{
			attrs.NewField(n, "PK"),
			attrs.NewField(n, "Title"),
			attrs.NewField(n, "Path"),
			attrs.NewField(n, "Depth"),
			attrs.NewField(n, "Numchild"),
			attrs.NewField(n, "UrlPath"),
			attrs.NewField(n, "Slug"),
			attrs.NewField(n, "StatusFlags"),
			attrs.NewField(n, "PageID"),
			attrs.NewField(n, "ContentType"),
			attrs.NewField(n, "LatestRevisionID", &attrs.FieldConfig{
				Null:   true,
				Blank:  true,
				Column: "latest_revision_id",
				//	RelForeignKey: relForeignKey,
			}),
			attrs.NewField(n, "CreatedAt"),
			attrs.NewField(n, "UpdatedAt"),
		}
	})
}

var ErrRouteNotFound = mux.ErrRouteNotFound

type RoutablePage interface {
	Route(r *http.Request, pathComponents []string) (Page, error)
}

func (n *PageNode) Route(r *http.Request, pathComponents []string) (Page, error) {

	if len(pathComponents) == 0 {
		if n.StatusFlags.Is(StatusFlagDeleted) ||
			n.StatusFlags.Is(StatusFlagHidden) ||
			!n.StatusFlags.Is(StatusFlagPublished) {
			return nil, fmt.Errorf("page is not published: %w", ErrRouteNotFound)
		}

		return n, nil
	}

	var pageRow, err = n.Children().
		WithContext(r.Context()).
		Filter("Slug", pathComponents[0]).
		Get()
	if err != nil {
		if !errors.Is(err, errors.NoRows) {
			return nil, fmt.Errorf("failed to get page by path %q: %w", pathComponents, err)
		}
		return nil, fmt.Errorf("page not found for path %q: %w", pathComponents, ErrRouteNotFound)
	}

	if pageRow.Object == nil {
		return nil, fmt.Errorf("page not found for path %q: %w", pathComponents, ErrRouteNotFound)
	}

	var cType = DefinitionForType(pageRow.Object.ContentType)
	if cType != nil {
		if _, ok := cType.Object().(RoutablePage); !ok {
			goto routePageNode
		}

		var rTyp = reflect.TypeOf(cType.Object())
		if isPromoted(rTyp, "Route") {
			goto routePageNode
		}

		var specific, err = pageRow.Object.Specific(r.Context())
		if err != nil {
			return nil, fmt.Errorf(
				"failed to get specific page for node %d: %w",
				pageRow.Object.PK, err,
			)
		}

		var ref = specific.Reference()
		ref.SetSpecificPageObject(pageRow.Object)
		ref._parent = n

		return specific.(RoutablePage).Route(r, pathComponents[1:])
	}

routePageNode:
	return pageRow.Object.Route(r, pathComponents[1:])
}

func (n *PageNode) Ancestors(inclusive ...bool) *PageQuerySet {
	return NewPageQuerySet().Ancestors(n.Path, n.Depth, inclusive...)
}

func (n *PageNode) Descendants(inclusive ...bool) *PageQuerySet {
	return NewPageQuerySet().Descendants(n.Path, n.Depth, inclusive...)
}

func (n *PageNode) Children() *PageQuerySet {
	return NewPageQuerySet().Children(n.Path, n.Depth)
}

func (n *PageNode) Specific(ctx context.Context, refresh ...bool) (Page, error) {
	var refreshFlag bool
	if len(refresh) > 0 {
		refreshFlag = refresh[0]
	}

	if !refreshFlag && n.PageObject != nil {
		return n.PageObject, nil
	}

	var p, err = Specific(ctx, n, true)
	if err != nil {
		return nil, fmt.Errorf("failed to get specific page for node %d: %w", n.PK, err)
	}

	n.PageObject = p
	return p, nil
}

func (n *PageNode) Parent(ctx context.Context, refresh ...bool) (parent *PageNode, err error) {
	if n.Depth == 0 {
		return nil, nil
	}

	var refreshFlag bool
	if len(refresh) > 0 {
		refreshFlag = refresh[0]
	}

	if !refreshFlag && n._parent != nil {
		return n._parent, nil
	}

	parent, err = NewPageQuerySet().
		WithContext(ctx).
		ParentNode(n.Path, int(n.Depth))
	if err != nil {
		return nil, err
	}

	n._parent = parent
	return parent, nil
}

func (n *PageNode) AddChild(ctx context.Context, child *PageNode) error {
	if child == nil {
		return fmt.Errorf("child node cannot be nil")
	}

	if ctx == nil {
		ctx = context.Background()
	}

	var err = NewPageQuerySet().
		WithContext(ctx).
		AddChildren(n, child)
	if err != nil {
		return fmt.Errorf("failed to add child node: %w", err)
	}

	child._parent = n
	return nil
}

func (n *PageNode) Move(ctx context.Context, newParent *PageNode) error {
	if ctx == nil {
		ctx = context.Background()
	}

	var err = NewPageQuerySet().
		WithContext(ctx).
		MoveNode(n, newParent)
	if err != nil {
		return fmt.Errorf("failed to move node: %w", err)
	}

	n._parent = newParent
	return nil
}

func (n *PageNode) Publish(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}

	return NewPageQuerySet().
		WithContext(ctx).
		PublishNode(n)
}

func (n *PageNode) Unpublish(ctx context.Context, unpublishChildren bool) error {
	if ctx == nil {
		ctx = context.Background()
	}

	return NewPageQuerySet().
		WithContext(ctx).
		UnpublishNode(n, unpublishChildren)
}

func (n *PageNode) CreateRevision(ctx context.Context, save bool) (*revisions.Revision, error) {
	var revision, err = revisions.NewRevision(n)
	if err != nil {
		return nil, fmt.Errorf("failed to create revision for page node %d: %w", n.PK, err)
	}

	if save {
		_, err = revisions.CreateRevision(ctx, revision)
		if err != nil {
			return nil, fmt.Errorf("failed to save revision for page node %d: %w", n.PK, err)
		}
		n.LatestRevisionID = revision.ID
	}

	return revision, nil
}
