package pages

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/Nigel2392/django/contrib/pages/models"
	models_mysql "github.com/Nigel2392/django/contrib/pages/pages-mysql"
	models_postgres "github.com/Nigel2392/django/contrib/pages/pages-postgres"
	models_sqlite "github.com/Nigel2392/django/contrib/pages/pages-sqlite"
	"github.com/go-sql-driver/mysql"
	"github.com/lib/pq"
	"github.com/mattn/go-sqlite3"
)

var querier models.DBQuerier

const STEP_LEN = 3

func Queries(db *sql.DB) models.DBQuerier {
	if db == nil && querier != nil {
		return querier
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
	case *pq.Driver:
		q = models_postgres.New(db)
	default:
		panic(fmt.Sprintf("unsupported driver: %T", db.Driver()))
	}

	if querier == nil {
		querier = q
	}

	return q
}

func checkQuerier() {
	if querier == nil {
		panic("querier is nil")
	}
}

func splitPathParts(path string) ([]int, error) {
	if path == "" {
		return []int{}, nil
	}

	if len(path)%STEP_LEN != 0 {
		return nil, fmt.Errorf("invalid path length: %d", len(path))
	}

	var parts = make([]int, 0, len(path)/STEP_LEN)
	for i := 0; i < len(path); i += STEP_LEN {
		part, err := strconv.Atoi(path[i : i+STEP_LEN])
		if err != nil {
			return nil, err
		}

		parts = append(parts, part)
	}

	return parts, nil
}

func _fmtPathParts(b io.Writer, part int) {
	fmt.Fprintf(b, "%03d", part)
}

func joinPathParts(parts []int) string {
	var b strings.Builder
	for i, part := range parts {
		_fmtPathParts(&b, part)
		if i < len(parts)-1 {
			b.WriteString(".")
		}
	}

	return b.String()
}

func createPathPart(parent *models.PageNode, subPages []models.PageNode) (string, error) {
	var (
		parts = make([]int, 0)
		err   error
	)
	if parent == nil {
		return joinPathParts([]int{1}), nil
	} else {
		parts, err = splitPathParts(parent.Path.String)
		if err != nil {
			return "", err
		}
	}

	if len(subPages) == 0 {
		parts[len(parts)-1]++
		return joinPathParts(parts), nil
	}

	var last = subPages[len(subPages)-1]
	lastParts, err := splitPathParts(last.Path.String)
	if err != nil {
		return "", err
	}

	if len(lastParts) != len(parts)+1 {
		return "", fmt.Errorf("invalid path length: %d", len(lastParts))
	}

	parts = append(parts, lastParts[len(lastParts)-1]+1)
	return joinPathParts(parts), nil
}

func AddRoot(q models.Querier, parent, page models.PageNode, autoCreate bool) error {
	var ctx = context.Background()

	// Check if parent exists
	if parent.ID.Int64 == 0 && autoCreate {
		if err := CreatePage(ctx, q, parent); err != nil {
			return err
		}
	}

	// Check if page exists
	if page.ID.Int64 == 0 && autoCreate {
		if err := CreatePage(ctx, q, page); err != nil {
			return err
		}
	}

	return nil
}

func CreatePage(ctx context.Context, q models.Querier, page models.PageNode) error {
	checkQuerier()

	// Get parent
	var (
		parent models.PageNode
		err    error
	)
	if page.Path.String != "" {
		parentPath := page.Path.String[:len(page.Path.String)-STEP_LEN]
		parent, err = q.GetNodeByPath(ctx, sql.NullString{String: parentPath, Valid: true})
		if err != nil {
			return err
		}
	}

	// Create page
	page.Path.String, err = createPathPart(&parent, []models.PageNode{})
	if err != nil {
		return err
	}

	_, err = q.InsertNode(ctx, page.Title, page.Path, page.Depth, page.Numchild, page.Typehash)
	if err != nil {
		return err
	}

	return nil
}
