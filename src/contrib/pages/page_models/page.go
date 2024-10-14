package page_models

import (
	"context"
	"database/sql"
	"time"

	"github.com/Nigel2392/go-django/src/core/attrs"
)

type StatusFlag int64

const (
	// StatusFlagPublished is the status flag for published pages.
	StatusFlagPublished StatusFlag = 1 << iota

	// StatusFlagHidden is the status flag for hidden pages.
	StatusFlagHidden

	// StatusFlagDeleted is the status flag for deleted pages.
	StatusFlagDeleted
)

func (f StatusFlag) Is(flag StatusFlag) bool {
	return f&flag == flag
}

type PageNode struct {
	PK          int64      `json:"id" attrs:"primary;readonly"`
	Title       string     `json:"title"`
	Path        string     `json:"path"`
	Depth       int64      `json:"depth" attrs:"blank"`
	Numchild    int64      `json:"numchild" attrs:"blank"`
	UrlPath     string     `json:"url_path" attrs:"readonly;blank"`
	Slug        string     `json:"slug"`
	StatusFlags StatusFlag `json:"status_flags" attrs:"null;blank"`
	PageID      int64      `json:"page_id" attrs:""`
	ContentType string     `json:"content_type" attrs:""`
	CreatedAt   time.Time  `json:"created_at" attrs:"readonly;label=Created At"`
	UpdatedAt   time.Time  `json:"updated_at" attrs:"readonly;label=Updated At"`

	// ParentNode is the parent node of this node.
	// It will likely be nil and is not fetched by default.
	ParentNode *PageNode `json:"parent_node" attrs:"readonly"`

	// ChildNodes are the child nodes of this node.
	// It will likely be nil and is not fetched by default.
	ChildNodes []*PageNode `json:"child_nodes" attrs:"readonly"`
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
		"CreatedAt",
		"UpdatedAt",
	)
}

type DBTX interface {
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
	PrepareContext(context.Context, string) (*sql.Stmt, error)
	QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
}

type DBQuerier interface {
	Querier
	DB() *sql.DB
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
}

type Querier interface {
	Close() error
	WithTx(tx *sql.Tx) Querier
	AllNodes(ctx context.Context, limit int32, offset int32) ([]PageNode, error)
	CountNodes(ctx context.Context) (int64, error)
	CountRootNodes(ctx context.Context) (int64, error)
	CountNodesByTypeHash(ctx context.Context, contentType string) (int64, error)
	DecrementNumChild(ctx context.Context, id int64) (PageNode, error)
	DeleteDescendants(ctx context.Context, path interface{}, depth int64) error
	DeleteNode(ctx context.Context, id int64) error
	DeleteNodes(ctx context.Context, id []int64) error
	GetChildNodes(ctx context.Context, path interface{}, depth interface{}, limit int32, offset int32) ([]PageNode, error)
	GetDescendants(ctx context.Context, path interface{}, depth int64, limit int32, offset int32) ([]PageNode, error)
	GetNodeByID(ctx context.Context, id int64) (PageNode, error)
	GetNodeByPath(ctx context.Context, path string) (PageNode, error)
	GetNodeBySlug(ctx context.Context, slug string, depth int64, path interface{}) (PageNode, error)
	GetNodesByDepth(ctx context.Context, depth int64, limit int32, offset int32) ([]PageNode, error)
	GetNodesByIDs(ctx context.Context, id []int64) ([]PageNode, error)
	GetNodesByPageIDs(ctx context.Context, pageID []int64) ([]PageNode, error)
	GetNodesByTypeHash(ctx context.Context, contentType string, limit int32, offset int32) ([]PageNode, error)
	GetNodesByTypeHashes(ctx context.Context, contentType []string, limit int32, offset int32) ([]PageNode, error)
	GetNodesForPaths(ctx context.Context, path []string) ([]PageNode, error)
	IncrementNumChild(ctx context.Context, id int64) (PageNode, error)
	InsertNode(ctx context.Context, title string, path string, depth int64, numchild int64, urlPath string, slug string, statusFlags int64, pageID int64, contentType string) (int64, error)
	UpdateNode(ctx context.Context, title string, path string, depth int64, numchild int64, urlPath string, slug string, statusFlags int64, pageID int64, contentType string, iD int64) error
	UpdateNodes(ctx context.Context, nodes []*PageNode) error
	UpdateNodePathAndDepth(ctx context.Context, path string, depth int64, iD int64) error
	UpdateNodeStatusFlags(ctx context.Context, statusFlags int64, iD int64) error
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
