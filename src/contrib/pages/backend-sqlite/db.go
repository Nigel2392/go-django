package models_sqlite

import (
	"context"
	"database/sql"
	"fmt"

	models "github.com/Nigel2392/go-django/src/contrib/pages/page_models"
)

type DBTX interface {
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
	PrepareContext(context.Context, string) (*sql.Stmt, error)
	QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
}

func New(db DBTX) *Queries {
	return &Queries{db: db}
}

func Prepare(ctx context.Context, db DBTX) (*Queries, error) {
	q := Queries{db: db}
	var err error
	if q.countNodesStmt, err = db.PrepareContext(ctx, countNodes); err != nil {
		return nil, fmt.Errorf("error preparing query CountNodes: %w", err)
	}
	if q.countNodesByTypeHashStmt, err = db.PrepareContext(ctx, countNodesByTypeHash); err != nil {
		return nil, fmt.Errorf("error preparing query CountNodesByTypeHash: %w", err)
	}
	if q.countRootNodesStmt, err = db.PrepareContext(ctx, countRootNodes); err != nil {
		return nil, fmt.Errorf("error preparing query CountRootNodes: %w", err)
	}
	if q.decrementNumChildStmt, err = db.PrepareContext(ctx, decrementNumChild); err != nil {
		return nil, fmt.Errorf("error preparing query DecrementNumChild: %w", err)
	}
	if q.deleteDescendantsStmt, err = db.PrepareContext(ctx, deleteDescendants); err != nil {
		return nil, fmt.Errorf("error preparing query DeleteDescendants: %w", err)
	}
	if q.deleteNodeStmt, err = db.PrepareContext(ctx, deleteNode); err != nil {
		return nil, fmt.Errorf("error preparing query DeleteNode: %w", err)
	}
	if q.deleteNodesStmt, err = db.PrepareContext(ctx, deleteNodes); err != nil {
		return nil, fmt.Errorf("error preparing query DeleteNodes: %w", err)
	}
	if q.getChildNodesStmt, err = db.PrepareContext(ctx, getChildNodes); err != nil {
		return nil, fmt.Errorf("error preparing query GetChildNodes: %w", err)
	}
	if q.getDescendantsStmt, err = db.PrepareContext(ctx, getDescendants); err != nil {
		return nil, fmt.Errorf("error preparing query GetDescendants: %w", err)
	}
	if q.getNodeByIDStmt, err = db.PrepareContext(ctx, getNodeByID); err != nil {
		return nil, fmt.Errorf("error preparing query GetNodeByID: %w", err)
	}
	if q.getNodeByPathStmt, err = db.PrepareContext(ctx, getNodeByPath); err != nil {
		return nil, fmt.Errorf("error preparing query GetNodeByPath: %w", err)
	}
	if q.getNodeBySlugStmt, err = db.PrepareContext(ctx, getNodeBySlug); err != nil {
		return nil, fmt.Errorf("error preparing query GetNodeBySlug: %w", err)
	}
	if q.getNodesByDepthStmt, err = db.PrepareContext(ctx, getNodesByDepth); err != nil {
		return nil, fmt.Errorf("error preparing query GetNodesByDepth: %w", err)
	}
	if q.getNodesByIDsStmt, err = db.PrepareContext(ctx, getNodesByIDs); err != nil {
		return nil, fmt.Errorf("error preparing query GetNodesByIDs: %w", err)
	}
	if q.getNodesByPageIDsStmt, err = db.PrepareContext(ctx, getNodesByPageIDs); err != nil {
		return nil, fmt.Errorf("error preparing query GetNodesByPageIDs: %w", err)
	}
	if q.getNodesByTypeHashStmt, err = db.PrepareContext(ctx, getNodesByTypeHash); err != nil {
		return nil, fmt.Errorf("error preparing query GetNodesByTypeHash: %w", err)
	}
	if q.getNodesByTypeHashesStmt, err = db.PrepareContext(ctx, getNodesByTypeHashes); err != nil {
		return nil, fmt.Errorf("error preparing query GetNodesByTypeHashes: %w", err)
	}
	if q.getNodesForPathsStmt, err = db.PrepareContext(ctx, getNodesForPaths); err != nil {
		return nil, fmt.Errorf("error preparing query GetNodesForPaths: %w", err)
	}
	if q.incrementNumChildStmt, err = db.PrepareContext(ctx, incrementNumChild); err != nil {
		return nil, fmt.Errorf("error preparing query IncrementNumChild: %w", err)
	}
	if q.insertNodeStmt, err = db.PrepareContext(ctx, insertNode); err != nil {
		return nil, fmt.Errorf("error preparing query InsertNode: %w", err)
	}
	if q.updateNodeStmt, err = db.PrepareContext(ctx, updateNode); err != nil {
		return nil, fmt.Errorf("error preparing query UpdateNode: %w", err)
	}
	if q.updateNodePathAndDepthStmt, err = db.PrepareContext(ctx, updateNodePathAndDepth); err != nil {
		return nil, fmt.Errorf("error preparing query UpdateNodePathAndDepth: %w", err)
	}
	if q.updateNodeStatusFlagsStmt, err = db.PrepareContext(ctx, updateNodeStatusFlags); err != nil {
		return nil, fmt.Errorf("error preparing query UpdateNodeStatusFlags: %w", err)
	}
	return &q, nil
}

func (q *Queries) Close() error {
	var err error
	if q.countNodesStmt != nil {
		if cerr := q.countNodesStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing countNodesStmt: %w", cerr)
		}
	}
	if q.countNodesByTypeHashStmt != nil {
		if cerr := q.countNodesByTypeHashStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing countNodesByTypeHashStmt: %w", cerr)
		}
	}
	if q.countRootNodesStmt != nil {
		if cerr := q.countRootNodesStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing countRootNodesStmt: %w", cerr)
		}
	}
	if q.decrementNumChildStmt != nil {
		if cerr := q.decrementNumChildStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing decrementNumChildStmt: %w", cerr)
		}
	}
	if q.deleteDescendantsStmt != nil {
		if cerr := q.deleteDescendantsStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing deleteDescendantsStmt: %w", cerr)
		}
	}
	if q.deleteNodeStmt != nil {
		if cerr := q.deleteNodeStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing deleteNodeStmt: %w", cerr)
		}
	}
	if q.deleteNodesStmt != nil {
		if cerr := q.deleteNodesStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing deleteNodesStmt: %w", cerr)
		}
	}
	if q.getChildNodesStmt != nil {
		if cerr := q.getChildNodesStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing getChildNodesStmt: %w", cerr)
		}
	}
	if q.getDescendantsStmt != nil {
		if cerr := q.getDescendantsStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing getDescendantsStmt: %w", cerr)
		}
	}
	if q.getNodeByIDStmt != nil {
		if cerr := q.getNodeByIDStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing getNodeByIDStmt: %w", cerr)
		}
	}
	if q.getNodeByPathStmt != nil {
		if cerr := q.getNodeByPathStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing getNodeByPathStmt: %w", cerr)
		}
	}
	if q.getNodeBySlugStmt != nil {
		if cerr := q.getNodeBySlugStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing getNodeBySlugStmt: %w", cerr)
		}
	}
	if q.getNodesByDepthStmt != nil {
		if cerr := q.getNodesByDepthStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing getNodesByDepthStmt: %w", cerr)
		}
	}
	if q.getNodesByIDsStmt != nil {
		if cerr := q.getNodesByIDsStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing getNodesByIDsStmt: %w", cerr)
		}
	}
	if q.getNodesByPageIDsStmt != nil {
		if cerr := q.getNodesByPageIDsStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing getNodesByPageIDsStmt: %w", cerr)
		}
	}
	if q.getNodesByTypeHashStmt != nil {
		if cerr := q.getNodesByTypeHashStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing getNodesByTypeHashStmt: %w", cerr)
		}
	}
	if q.getNodesByTypeHashesStmt != nil {
		if cerr := q.getNodesByTypeHashesStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing getNodesByTypeHashesStmt: %w", cerr)
		}
	}
	if q.getNodesForPathsStmt != nil {
		if cerr := q.getNodesForPathsStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing getNodesForPathsStmt: %w", cerr)
		}
	}
	if q.incrementNumChildStmt != nil {
		if cerr := q.incrementNumChildStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing incrementNumChildStmt: %w", cerr)
		}
	}
	if q.insertNodeStmt != nil {
		if cerr := q.insertNodeStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing insertNodeStmt: %w", cerr)
		}
	}
	if q.updateNodeStmt != nil {
		if cerr := q.updateNodeStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing updateNodeStmt: %w", cerr)
		}
	}
	if q.updateNodePathAndDepthStmt != nil {
		if cerr := q.updateNodePathAndDepthStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing updateNodePathAndDepthStmt: %w", cerr)
		}
	}
	if q.updateNodeStatusFlagsStmt != nil {
		if cerr := q.updateNodeStatusFlagsStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing updateNodeStatusFlagsStmt: %w", cerr)
		}
	}
	return err
}

func (q *Queries) exec(ctx context.Context, stmt *sql.Stmt, query string, args ...interface{}) (sql.Result, error) {
	switch {
	case stmt != nil && q.tx != nil:
		return q.tx.StmtContext(ctx, stmt).ExecContext(ctx, args...)
	case stmt != nil:
		return stmt.ExecContext(ctx, args...)
	default:
		return q.db.ExecContext(ctx, query, args...)
	}
}

func (q *Queries) query(ctx context.Context, stmt *sql.Stmt, query string, args ...interface{}) (*sql.Rows, error) {
	switch {
	case stmt != nil && q.tx != nil:
		return q.tx.StmtContext(ctx, stmt).QueryContext(ctx, args...)
	case stmt != nil:
		return stmt.QueryContext(ctx, args...)
	default:
		return q.db.QueryContext(ctx, query, args...)
	}
}

func (q *Queries) queryRow(ctx context.Context, stmt *sql.Stmt, query string, args ...interface{}) *sql.Row {
	switch {
	case stmt != nil && q.tx != nil:
		return q.tx.StmtContext(ctx, stmt).QueryRowContext(ctx, args...)
	case stmt != nil:
		return stmt.QueryRowContext(ctx, args...)
	default:
		return q.db.QueryRowContext(ctx, query, args...)
	}
}

type Queries struct {
	db                         DBTX
	tx                         *sql.Tx
	countNodesStmt             *sql.Stmt
	countNodesByTypeHashStmt   *sql.Stmt
	countRootNodesStmt         *sql.Stmt
	decrementNumChildStmt      *sql.Stmt
	deleteDescendantsStmt      *sql.Stmt
	deleteNodeStmt             *sql.Stmt
	deleteNodesStmt            *sql.Stmt
	getChildNodesStmt          *sql.Stmt
	getDescendantsStmt         *sql.Stmt
	getNodeByIDStmt            *sql.Stmt
	getNodeByPathStmt          *sql.Stmt
	getNodeBySlugStmt          *sql.Stmt
	getNodesByDepthStmt        *sql.Stmt
	getNodesByIDsStmt          *sql.Stmt
	getNodesByPageIDsStmt      *sql.Stmt
	getNodesByTypeHashStmt     *sql.Stmt
	getNodesByTypeHashesStmt   *sql.Stmt
	getNodesForPathsStmt       *sql.Stmt
	incrementNumChildStmt      *sql.Stmt
	insertNodeStmt             *sql.Stmt
	updateNodeStmt             *sql.Stmt
	updateNodePathAndDepthStmt *sql.Stmt
	updateNodeStatusFlagsStmt  *sql.Stmt
}

func (q *Queries) WithTx(tx *sql.Tx) models.Querier {
	return &Queries{
		db:                         tx,
		tx:                         tx,
		countNodesStmt:             q.countNodesStmt,
		countNodesByTypeHashStmt:   q.countNodesByTypeHashStmt,
		countRootNodesStmt:         q.countRootNodesStmt,
		decrementNumChildStmt:      q.decrementNumChildStmt,
		deleteDescendantsStmt:      q.deleteDescendantsStmt,
		deleteNodeStmt:             q.deleteNodeStmt,
		deleteNodesStmt:            q.deleteNodesStmt,
		getChildNodesStmt:          q.getChildNodesStmt,
		getDescendantsStmt:         q.getDescendantsStmt,
		getNodeByIDStmt:            q.getNodeByIDStmt,
		getNodeByPathStmt:          q.getNodeByPathStmt,
		getNodeBySlugStmt:          q.getNodeBySlugStmt,
		getNodesByDepthStmt:        q.getNodesByDepthStmt,
		getNodesByIDsStmt:          q.getNodesByIDsStmt,
		getNodesByPageIDsStmt:      q.getNodesByPageIDsStmt,
		getNodesByTypeHashStmt:     q.getNodesByTypeHashStmt,
		getNodesByTypeHashesStmt:   q.getNodesByTypeHashesStmt,
		getNodesForPathsStmt:       q.getNodesForPathsStmt,
		incrementNumChildStmt:      q.incrementNumChildStmt,
		insertNodeStmt:             q.insertNodeStmt,
		updateNodeStmt:             q.updateNodeStmt,
		updateNodePathAndDepthStmt: q.updateNodePathAndDepthStmt,
		updateNodeStatusFlagsStmt:  q.updateNodeStatusFlagsStmt,
	}
}
