package page_models

import (
	"context"
	"database/sql"
)

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
	AllNodes(ctx context.Context, statusFlags StatusFlag, offset int32, limit int32, orderings ...string) ([]PageNode, error)
	CountNodes(ctx context.Context, statusFlags StatusFlag) (int64, error)
	CountNodesByTypeHash(ctx context.Context, contentType string) (int64, error)
	CountRootNodes(ctx context.Context, statusFlags StatusFlag) (int64, error)
	DecrementNumChild(ctx context.Context, id int64) (PageNode, error)
	DeleteDescendants(ctx context.Context, path string, depth int64) error
	DeleteNode(ctx context.Context, id int64) error
	DeleteNodes(ctx context.Context, id []int64) error
	GetChildNodes(ctx context.Context, path string, depth int64, statusFlags StatusFlag, offset int32, limit int32) ([]PageNode, error)
	GetDescendants(ctx context.Context, path string, depth int64, statusFlags StatusFlag, offset int32, limit int32) ([]PageNode, error)
	GetNodeByID(ctx context.Context, id int64) (PageNode, error)
	GetNodeByPath(ctx context.Context, path string) (PageNode, error)
	GetNodeBySlug(ctx context.Context, slug string, depth int64, path string) (PageNode, error)
	GetNodesByDepth(ctx context.Context, depth int64, statusFlags StatusFlag, offset int32, limit int32) ([]PageNode, error)
	GetNodesByIDs(ctx context.Context, id []int64) ([]PageNode, error)
	GetNodesByPageIDs(ctx context.Context, pageID []int64) ([]PageNode, error)
	GetNodesByTypeHash(ctx context.Context, contentType string, offset int32, limit int32) ([]PageNode, error)
	GetNodesByTypeHashes(ctx context.Context, contentType []string, offset int32, limit int32) ([]PageNode, error)
	GetNodesForPaths(ctx context.Context, path []string) ([]PageNode, error)
	IncrementNumChild(ctx context.Context, id int64) (PageNode, error)
	InsertNode(ctx context.Context, title string, path string, depth int64, numchild int64, urlPath string, slug string, statusFlags int64, pageID int64, contentType string, latestRevisionID int64) (int64, error)
	UpdateNode(ctx context.Context, title string, path string, depth int64, numchild int64, urlPath string, slug string, statusFlags int64, pageID int64, contentType string, latestRevisionID int64, iD int64) error
	UpdateNodes(ctx context.Context, nodes []*PageNode) error
	UpdateNodePathAndDepth(ctx context.Context, path string, depth int64, iD int64) error
	UpdateNodeStatusFlags(ctx context.Context, statusFlags int64, iD int64) error
	UpdateDescendantPaths(ctx context.Context, oldUrlPath, newUrlPath, pageNodePath string, id int64) error
}
