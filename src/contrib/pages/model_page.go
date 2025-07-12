package pages

import (
	"context"
	"reflect"
	"time"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/migrator"
	"github.com/Nigel2392/go-django/queries/src/models"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/gosimple/slug"
)

var (
	_ queries.ActsBeforeCreate = (*PageNode)(nil)
	_ queries.ActsAfterSave    = (*PageNode)(nil)
)

type PageNode struct {
	models.Model     `table:"PageNode"`
	PK               int64      `json:"id" attrs:"primary;readonly;column=id"`
	Title            string     `json:"title"`
	Path             string     `json:"path"`
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

func (n *PageNode) ID() int64 {
	return n.PK
}

func (n *PageNode) Reference() *PageNode {
	return n
}

func (n *PageNode) IsRoot() bool {
	return n.Depth == 0
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
	if !n.CreatedAt.IsZero() && n.UpdatedAt.IsZero() {
		n.UpdatedAt = n.CreatedAt
	} else {
		n.UpdatedAt = time.Now()
	}
	return nil
}

func (n *PageNode) DatabaseIndexes(obj attrs.Definer) []migrator.Index {
	if reflect.TypeOf(obj) != reflect.TypeOf(n) {
		return nil
	}
	return []migrator.Index{
		{Fields: []string{"Path"}, Unique: true},
		{Fields: []string{"PageID", "ContentType"}, Unique: true},
		{Fields: []string{"Slug", "Depth"}, Unique: true},
	}
}

func (n *PageNode) Specific(ctx context.Context) (Page, error) {
	return Specific(ctx, n)
}

func (n *PageNode) Children(ctx context.Context) ([]*PageNode, error) {
	var qs = NewPageQuerySet().WithContext(ctx)
	return qs.GetChildNodes(n, StatusFlagNone, 0, 1000)
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

func (n *PageNode) FieldDefs() attrs.Definitions {
	return n.Model.Define(n, func(d attrs.Definer) []attrs.Field {

		//	var relForeignKey attrs.Relation
		//	if django.AppInstalled != nil && django.AppInstalled("revisions") {
		//		relForeignKey = attrs.Relate(
		//			&revisions.Revision{},
		//			"", nil,
		//		)
		//	}

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
