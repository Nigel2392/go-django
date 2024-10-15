package permissions_models

import (
	"context"
	"database/sql"

	"github.com/Nigel2392/go-django/src/models"
)

var backend = models.NewBackendRegistry[Querier]()
var Register = backend.RegisterForDriver
var BackendForDB = backend.BackendForDB

type InsertQuerier interface {
	InsertGroup(ctx context.Context, name string, description string) (int64, error)
	InsertGroupPermission(ctx context.Context, groupID uint64, permissionID uint64) (int64, error)
	InsertPermission(ctx context.Context, name string, description string) (int64, error)
	InsertUserGroup(ctx context.Context, userID uint64, groupID uint64) (int64, error)
}

type UpdateQuerier interface {
	UpdateGroup(ctx context.Context, name string, description string, iD uint64) error
	UpdatePermission(ctx context.Context, name string, description string, iD uint64) error
}

type DeleteQuerier interface {
	DeleteGroup(ctx context.Context, id uint64) error
	DeleteGroupPermission(ctx context.Context, groupID uint64, permissionID uint64) error
	DeletePermission(ctx context.Context, id uint64) error
	DeleteUserGroup(ctx context.Context, userID uint64, groupID uint64) error
}

type Querier interface {
	InsertQuerier
	DeleteQuerier
	UpdateQuerier

	WithTx(*sql.Tx) Querier
	Close() error

	AllGroups(ctx context.Context, limit int32, offset int32) ([]*Group, error)
	AllPermissions(ctx context.Context, limit int32, offset int32) ([]*Permission, error)
	GetGroupByID(ctx context.Context, id uint64) (*Group, error)
	GetPermissionByID(ctx context.Context, id uint64) (*Permission, error)
	PermissionsForUser(ctx context.Context, userID uint64) ([]*Permission, error)
	UserGroups(ctx context.Context, id uint64) (g Group, p []*Permission, err error)
	UserHasPermission(ctx context.Context, userID uint64, permissionName string) (int64, error)
}
