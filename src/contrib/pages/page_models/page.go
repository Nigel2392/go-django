package page_models

import (
	"time"

	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/gosimple/slug"
)

type PageNode struct {
	PK               int64      `json:"id" attrs:"primary;readonly"`
	Title            string     `json:"title"`
	Path             string     `json:"path"`
	Depth            int64      `json:"depth" attrs:"blank"`
	Numchild         int64      `json:"numchild" attrs:"blank"`
	UrlPath          string     `json:"url_path" attrs:"readonly;blank"`
	Slug             string     `json:"slug"`
	StatusFlags      StatusFlag `json:"status_flags" attrs:"null;blank"`
	PageID           int64      `json:"page_id" attrs:""`
	ContentType      string     `json:"content_type" attrs:""`
	LatestRevisionID int64      `json:"latest_revision_id" attrs:""`
	CreatedAt        time.Time  `json:"created_at" attrs:"readonly;label=Created At"`
	UpdatedAt        time.Time  `json:"updated_at" attrs:"readonly;label=Updated At"`

	// ParentNode is the parent node of this node.
	// It will likely be nil and is not fetched by default.
	ParentNode *PageNode `json:"parent_node" attrs:"readonly"`

	// ChildNodes are the child nodes of this node.
	// It will likely be nil and is not fetched by default.
	ChildNodes []*PageNode `json:"child_nodes" attrs:"readonly"`
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

func (n *PageNode) FieldDefs() attrs.Definitions {
	return attrs.AutoDefinitions(n,
		"PK",
		"Title",
		"Path",
		"Depth",
		"Numchild",
		"UrlPath",
		"Slug",
		"StatusFlags",
		"PageID",
		"ContentType",
		"LatestRevisionID",
		"CreatedAt",
		"UpdatedAt",
	)
}

/// MoveNodeParams contains parameters for moving a node.
//ype MoveNodeParams struct {
//ID          int64
//NewPath     string
//NewDepth    int64
//NewParentID sql.NullInt64
//
/// MoveNode moves a node within the tree.
//unc MoveNode(ctx context.Context, q DBQuerier, params MoveNodeParams) error {
//// Start a transaction
//var tx *sql.Tx
//var err error
//if tx, err = q.BeginTx(ctx); err != nil {
//	return err
//}
//var queries = q.WithTx(tx)
//defer tx.Rollback()
//// Lock the table
//if _, err := tx.ExecContext(ctx, "LOCK TABLE PageNode IN EXCLUSIVE MODE"); err != nil {
//	return err
//}
//// Fetch the node and its children
//nodes, err := queries.GetNodeWithChildren(ctx, params.ID)
//if err != nil {
//	return err
//}
//// Calculate new paths and depths
//oldPath := nodes[0].Path.String
//newPath := params.NewPath
//pathPrefix := oldPath + "."
//for _, node := range nodes {
//	relativePath := node.Path.String[len(pathPrefix):]
//	newChildPath := newPath + "." + relativePath
//	newDepth := params.NewDepth + (node.Depth.Int64 - nodes[0].Depth.Int64)
//	var newPathSQL sql.NullString
//	var newDepthSQL sql.NullInt64
//	if node.ID == params.ID {
//		newPathSQL = sql.NullString{String: newPath, Valid: true}
//		newDepthSQL = sql.NullInt64{Int64: newDepth, Valid: true}
//	} else {
//		newPathSQL = sql.NullString{String: newChildPath, Valid: true}
//		newDepthSQL = sql.NullInt64{Int64: newDepth, Valid: true}
//	}
//	// Update node path and depth
//	if node.ID == params.ID {
//		if err := queries.UpdateNodePathAndDepth(ctx, node.ID, newPathSQL, newDepthSQL, params.NewParentID); err != nil {
//			return err
//		}
//	} else {
//		if err := queries.UpdateChildNode(ctx, node.ID, newPathSQL, newDepthSQL); err != nil {
//			return err
//		}
//	}
//}
//// Commit the transaction
//if err := tx.Commit(); err != nil {
//	return err
//}
//return nil
//
//
