package models

import (
	"context"
	"database/sql"
	"fmt"

	_ "embed"

	permissions_models "github.com/Nigel2392/go-django/src/contrib/auth/auth-permissions/permissions-models"
	dj_models "github.com/Nigel2392/go-django/src/models"
	"github.com/go-sql-driver/mysql"
)

var _ permissions_models.Querier = (*Queries)(nil)

//go:embed schema.mysql.sql
var mysql_schema string

func init() {
	permissions_models.Register(
		mysql.MySQLDriver{}, &dj_models.BaseBackend[permissions_models.Querier]{
			CreateTableQuery: mysql_schema,
			NewQuerier: func(d *sql.DB) (permissions_models.Querier, error) {
				return New(d), nil
			},
			PreparedQuerier: func(ctx context.Context, d *sql.DB) (permissions_models.Querier, error) {
				return Prepare(ctx, nil)
			},
		},
	)
}

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
	if q.allGroupsStmt, err = db.PrepareContext(ctx, allGroups); err != nil {
		return nil, fmt.Errorf("error preparing query AllGroups: %w", err)
	}
	if q.allPermissionsStmt, err = db.PrepareContext(ctx, allPermissions); err != nil {
		return nil, fmt.Errorf("error preparing query AllPermissions: %w", err)
	}
	if q.deleteGroupStmt, err = db.PrepareContext(ctx, deleteGroup); err != nil {
		return nil, fmt.Errorf("error preparing query DeleteGroup: %w", err)
	}
	if q.deleteGroupPermissionStmt, err = db.PrepareContext(ctx, deleteGroupPermission); err != nil {
		return nil, fmt.Errorf("error preparing query DeleteGroupPermission: %w", err)
	}
	if q.deletePermissionStmt, err = db.PrepareContext(ctx, deletePermission); err != nil {
		return nil, fmt.Errorf("error preparing query DeletePermission: %w", err)
	}
	if q.deleteUserGroupStmt, err = db.PrepareContext(ctx, deleteUserGroup); err != nil {
		return nil, fmt.Errorf("error preparing query DeleteUserGroup: %w", err)
	}
	if q.getGroupByIDStmt, err = db.PrepareContext(ctx, getGroupByID); err != nil {
		return nil, fmt.Errorf("error preparing query GetGroupByID: %w", err)
	}
	if q.getPermissionByIDStmt, err = db.PrepareContext(ctx, getPermissionByID); err != nil {
		return nil, fmt.Errorf("error preparing query GetPermissionByID: %w", err)
	}
	if q.insertGroupStmt, err = db.PrepareContext(ctx, insertGroup); err != nil {
		return nil, fmt.Errorf("error preparing query InsertGroup: %w", err)
	}
	if q.insertGroupPermissionStmt, err = db.PrepareContext(ctx, insertGroupPermission); err != nil {
		return nil, fmt.Errorf("error preparing query InsertGroupPermission: %w", err)
	}
	if q.insertPermissionStmt, err = db.PrepareContext(ctx, insertPermission); err != nil {
		return nil, fmt.Errorf("error preparing query InsertPermission: %w", err)
	}
	if q.insertUserGroupStmt, err = db.PrepareContext(ctx, insertUserGroup); err != nil {
		return nil, fmt.Errorf("error preparing query InsertUserGroup: %w", err)
	}
	if q.permissionsForUserStmt, err = db.PrepareContext(ctx, permissionsForUser); err != nil {
		return nil, fmt.Errorf("error preparing query PermissionsForUser: %w", err)
	}
	if q.updateGroupStmt, err = db.PrepareContext(ctx, updateGroup); err != nil {
		return nil, fmt.Errorf("error preparing query UpdateGroup: %w", err)
	}
	if q.updatePermissionStmt, err = db.PrepareContext(ctx, updatePermission); err != nil {
		return nil, fmt.Errorf("error preparing query UpdatePermission: %w", err)
	}
	if q.userGroupsStmt, err = db.PrepareContext(ctx, userGroups); err != nil {
		return nil, fmt.Errorf("error preparing query UserGroups: %w", err)
	}
	if q.userHasPermissionStmt, err = db.PrepareContext(ctx, userHasPermission); err != nil {
		return nil, fmt.Errorf("error preparing query UserHasPermission: %w", err)
	}
	return &q, nil
}

func (q *Queries) Close() error {
	var err error
	if q.allGroupsStmt != nil {
		if cerr := q.allGroupsStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing allGroupsStmt: %w", cerr)
		}
	}
	if q.allPermissionsStmt != nil {
		if cerr := q.allPermissionsStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing allPermissionsStmt: %w", cerr)
		}
	}
	if q.deleteGroupStmt != nil {
		if cerr := q.deleteGroupStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing deleteGroupStmt: %w", cerr)
		}
	}
	if q.deleteGroupPermissionStmt != nil {
		if cerr := q.deleteGroupPermissionStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing deleteGroupPermissionStmt: %w", cerr)
		}
	}
	if q.deletePermissionStmt != nil {
		if cerr := q.deletePermissionStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing deletePermissionStmt: %w", cerr)
		}
	}
	if q.deleteUserGroupStmt != nil {
		if cerr := q.deleteUserGroupStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing deleteUserGroupStmt: %w", cerr)
		}
	}
	if q.getGroupByIDStmt != nil {
		if cerr := q.getGroupByIDStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing getGroupByIDStmt: %w", cerr)
		}
	}
	if q.getPermissionByIDStmt != nil {
		if cerr := q.getPermissionByIDStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing getPermissionByIDStmt: %w", cerr)
		}
	}
	if q.insertGroupStmt != nil {
		if cerr := q.insertGroupStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing insertGroupStmt: %w", cerr)
		}
	}
	if q.insertGroupPermissionStmt != nil {
		if cerr := q.insertGroupPermissionStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing insertGroupPermissionStmt: %w", cerr)
		}
	}
	if q.insertPermissionStmt != nil {
		if cerr := q.insertPermissionStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing insertPermissionStmt: %w", cerr)
		}
	}
	if q.insertUserGroupStmt != nil {
		if cerr := q.insertUserGroupStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing insertUserGroupStmt: %w", cerr)
		}
	}
	if q.permissionsForUserStmt != nil {
		if cerr := q.permissionsForUserStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing permissionsForUserStmt: %w", cerr)
		}
	}
	if q.updateGroupStmt != nil {
		if cerr := q.updateGroupStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing updateGroupStmt: %w", cerr)
		}
	}
	if q.updatePermissionStmt != nil {
		if cerr := q.updatePermissionStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing updatePermissionStmt: %w", cerr)
		}
	}
	if q.userGroupsStmt != nil {
		if cerr := q.userGroupsStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing userGroupsStmt: %w", cerr)
		}
	}
	if q.userHasPermissionStmt != nil {
		if cerr := q.userHasPermissionStmt.Close(); cerr != nil {
			err = fmt.Errorf("error closing userHasPermissionStmt: %w", cerr)
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
	db                        DBTX
	tx                        *sql.Tx
	allGroupsStmt             *sql.Stmt
	allPermissionsStmt        *sql.Stmt
	deleteGroupStmt           *sql.Stmt
	deleteGroupPermissionStmt *sql.Stmt
	deletePermissionStmt      *sql.Stmt
	deleteUserGroupStmt       *sql.Stmt
	getGroupByIDStmt          *sql.Stmt
	getPermissionByIDStmt     *sql.Stmt
	insertGroupStmt           *sql.Stmt
	insertGroupPermissionStmt *sql.Stmt
	insertPermissionStmt      *sql.Stmt
	insertUserGroupStmt       *sql.Stmt
	permissionsForUserStmt    *sql.Stmt
	updateGroupStmt           *sql.Stmt
	updatePermissionStmt      *sql.Stmt
	userGroupsStmt            *sql.Stmt
	userHasPermissionStmt     *sql.Stmt
}

func (q *Queries) WithTx(tx *sql.Tx) permissions_models.Querier {
	return &Queries{
		db:                        tx,
		tx:                        tx,
		allGroupsStmt:             q.allGroupsStmt,
		allPermissionsStmt:        q.allPermissionsStmt,
		deleteGroupStmt:           q.deleteGroupStmt,
		deleteGroupPermissionStmt: q.deleteGroupPermissionStmt,
		deletePermissionStmt:      q.deletePermissionStmt,
		deleteUserGroupStmt:       q.deleteUserGroupStmt,
		getGroupByIDStmt:          q.getGroupByIDStmt,
		getPermissionByIDStmt:     q.getPermissionByIDStmt,
		insertGroupStmt:           q.insertGroupStmt,
		insertGroupPermissionStmt: q.insertGroupPermissionStmt,
		insertPermissionStmt:      q.insertPermissionStmt,
		insertUserGroupStmt:       q.insertUserGroupStmt,
		permissionsForUserStmt:    q.permissionsForUserStmt,
		updateGroupStmt:           q.updateGroupStmt,
		updatePermissionStmt:      q.updatePermissionStmt,
		userGroupsStmt:            q.userGroupsStmt,
		userHasPermissionStmt:     q.userHasPermissionStmt,
	}
}
