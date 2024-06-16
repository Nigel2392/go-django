package pages

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Nigel2392/django/contrib/pages/models"
	models_mysql "github.com/Nigel2392/django/contrib/pages/models-mysql"
	models_postgres "github.com/Nigel2392/django/contrib/pages/models-postgres"
	models_sqlite "github.com/Nigel2392/django/contrib/pages/models-sqlite"
	"github.com/go-sql-driver/mysql"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/mattn/go-sqlite3"

	_ "embed"
)

var _ models.DBQuerier = (*Querier)(nil)

type Querier struct {
	models.Querier
	db *sql.DB
}

func (q *Querier) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	return q.db.BeginTx(ctx, opts)
}

var querySet models.DBQuerier

//go:embed sqlc/schema.mysql.sql
var mySQLCreateTable string

//go:embed sqlc/schema.sqlite3.sql
var sqliteCreateTable string

//go:embed sqlc/schema.postgres.sql
var postgresCreateTable string

func CreateTable(db *sql.DB) error {
	var ctx = context.Background()
	switch db.Driver().(type) {
	case *mysql.MySQLDriver:
		_, err := db.ExecContext(ctx, mySQLCreateTable)
		if err != nil {
			return err
		}
	case *sqlite3.SQLiteDriver:
		_, err := db.ExecContext(ctx, sqliteCreateTable)
		if err != nil {
			return err
		}
	case *stdlib.Driver:
		_, err := db.ExecContext(ctx, postgresCreateTable)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported driver: %T", db.Driver())
	}

	return nil
}

func QuerySet(db *sql.DB) models.DBQuerier {
	if db == nil && querySet != nil {
		return querySet
	}

	if db == nil {
		panic("db is nil")
	}

	var q models.Querier
	switch db.Driver().(type) {
	case *mysql.MySQLDriver:
		q = models_mysql.New(db)
	case *sqlite3.SQLiteDriver:
		q = models_sqlite.New(db)
	case *stdlib.Driver:
		q = models_postgres.New(db)
	default:
		panic(fmt.Sprintf("unsupported driver: %T", db.Driver()))
	}

	if querySet == nil {
		querySet = &Querier{
			Querier: q,
			db:      db,
		}
	}

	return querySet
}

func CreateRootNode(ctx context.Context, q models.Querier, node *models.PageNode) error {
	if node.Path != "" {
		return fmt.Errorf("node path must be empty")
	}

	node.Path = buildPathPart(0)
	node.Depth = 0

	id, err := q.InsertNode(ctx, node.Title, node.Path, node.Depth, node.Numchild, int64(node.StatusFlags), node.PageID, node.Typehash)
	if err != nil {
		return err
	}

	node.ID = id

	return nil
}

func CreateChildNode(ctx context.Context, q models.DBQuerier, parent, child *models.PageNode) error {

	var tx, err = q.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer tx.Rollback()

	var queries = q.WithTx(tx)

	if parent.Path == "" {
		return fmt.Errorf("parent path must not be empty")
	}

	if child.Path != "" {
		return fmt.Errorf("child path must be empty")
	}

	child.Path = parent.Path + buildPathPart(parent.Numchild)
	child.Depth = parent.Depth + 1

	var id int64
	id, err = queries.InsertNode(ctx, child.Title, child.Path, child.Depth, child.Numchild, int64(child.StatusFlags), child.PageID, child.Typehash)
	if err != nil {
		return err
	}
	child.ID = id
	parent.Numchild++
	err = queries.UpdateNode(ctx, parent.Title, parent.Path, parent.Depth, parent.Numchild, int64(parent.StatusFlags), parent.PageID, parent.Typehash, parent.ID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func Specific(p Page) (Page, error) {
	var ctx = context.Background()

	return SpecificInstance(
		ctx, p.Reference(),
	)
}

func ParentNode(p Page) (v models.PageNode, err error) {
	var ref = p.Reference()
	if ref.IsRoot() {
		return v, ErrPageIsRoot
	}
	var parentPath string
	parentPath, err = ancestorPath(
		ref.Path, 1,
	)
	if err != nil {
		return v, err
	}
	var ctx = context.Background()
	return querySet.GetNodeByPath(
		ctx, parentPath,
	)
}

func ChildNodes(p Page) ([]models.PageNode, error) {
	var ref = p.Reference()
	var ctx = context.Background()
	return querySet.GetChildren(
		ctx, ref.Path, ref.Depth,
	)
}

func AncestorNodes(p Page) ([]models.PageNode, error) {
	var ref = p.Reference()
	var paths = make([]string, 0, ref.Depth)
	for i := 1; i < int(ref.Depth); i++ {
		var path, err = ancestorPath(
			ref.Path, int64(i),
		)
		if err != nil {
			return nil, err
		}
		paths = append(paths, path)
	}
	var ctx = context.Background()
	return querySet.GetForPaths(
		ctx, paths,
	)
}

func Descendants(p Page) ([]models.PageNode, error) {
	var ref = p.Reference()
	var ctx = context.Background()
	return querySet.GetDescendants(
		ctx, ref.Path, ref.Depth,
	)
}
