package migrator

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/drivers/dbtype"
	"github.com/Nigel2392/go-django/src/core/attrs"
)

const (
	// Keys for attrs.Field
	AttrDBTypeKey   = "migrator.db_type"
	AttrUseInDBKey  = "migrator.use_in_db"
	AttrOnDeleteKey = "migrator.on_delete"
	AttrOnUpdateKey = "migrator.on_update"

	// Keys for attrs.ModelMeta
	MetaAllowMigrateKey = "migrator.allow_migrate"

	// The directory where the migration files are stored
	APPVAR_MIGRATION_DIR = "migrator.migration_dir"
)

type CanColumnDBType interface {
	DBType(*Column) dbtype.Type
}

type CanSQL[T any] interface {
	SQL(T) (string, []any)
}

type CanMigrate interface {
	CanMigrate() bool
}

type SchemaEditor interface {
	Setup(ctx context.Context) error
	StartTransaction(ctx context.Context) (drivers.Transaction, error)
	StoreMigration(ctx context.Context, appName string, modelName string, migrationName string) error
	HasMigration(ctx context.Context, appName string, modelName string, migrationName string) (bool, error)
	RemoveMigration(ctx context.Context, appName string, modelName string, migrationName string) error

	Execute(ctx context.Context, query string, args ...any) (sql.Result, error)

	CreateTable(ctx context.Context, table Table, ifNotExists bool) error
	DropTable(ctx context.Context, table Table, ifExists bool) error
	RenameTable(ctx context.Context, table Table, newName string) error

	AddIndex(ctx context.Context, table Table, index Index, ifNotExists bool) error
	DropIndex(ctx context.Context, table Table, index Index, ifExists bool) error
	RenameIndex(ctx context.Context, table Table, oldName string, newName string) error

	//	AlterUniqueTogether(table Table, unique bool) error
	//	AlterIndexTogether(table Table, unique bool) error

	AddField(ctx context.Context, table Table, col Column) error
	AlterField(ctx context.Context, table Table, old Column, newCol Column) error
	RemoveField(ctx context.Context, table Table, col Column) error
}

type Table interface {
	TableName() string
	Model() attrs.Definer
	Columns() []*Column
	Comment() string
	Indexes() []Index
}

// Embed this struct in your model or fields to indicate that it cannot be migrated.
type CantMigrate struct{}

func (c *CantMigrate) CanMigrate() bool {
	return false
}

func CheckCanMigrate(obj any) bool {

	if canMigrator, ok := obj.(CanMigrate); ok {
		return canMigrator.CanMigrate()
	}

	if obj, ok := obj.(attrs.Definer); ok {
		var meta = attrs.GetModelMeta(obj)
		if meta == nil {
			return false
		}

		var canMigrate, ok = meta.Storage(MetaAllowMigrateKey)
		if !ok {
			return true
		}

		switch canMigrate := canMigrate.(type) {
		case bool:
			return canMigrate
		case int, int8, int16, int32, int64,
			uint, uint8, uint16, uint32, uint64,
			float32, float64:
			return (!EqualDefaultValue(0, canMigrate)) // if canMigrate != 0
		default:
			panic(fmt.Errorf(
				"CheckCanMigrate: unexpected type %T for MetaAllowMigrateKey, expected bool or numeric type",
				canMigrate,
			))
		}
	}

	return true
}
