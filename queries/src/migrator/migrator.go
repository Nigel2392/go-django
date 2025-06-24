package migrator

import (
	"context"
	"database/sql"

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

type CanSQL[T any] interface {
	SQL(T) (string, []any)
}

type CanModelMigrate interface {
	CanMigrate() bool
}

type SchemaEditor interface {
	Setup() error
	StoreMigration(appName string, modelName string, migrationName string) error
	HasMigration(appName string, modelName string, migrationName string) (bool, error)
	RemoveMigration(appName string, modelName string, migrationName string) error

	Execute(ctx context.Context, query string, args ...any) (sql.Result, error)

	CreateTable(table Table, ifNotExists bool) error
	DropTable(table Table, ifExists bool) error
	RenameTable(table Table, newName string) error

	AddIndex(table Table, index Index, ifNotExists bool) error
	DropIndex(table Table, index Index, ifExists bool) error
	RenameIndex(table Table, oldName string, newName string) error

	//	AlterUniqueTogether(table Table, unique bool) error
	//	AlterIndexTogether(table Table, unique bool) error

	AddField(table Table, col Column) error
	AlterField(table Table, old Column, newCol Column) error
	RemoveField(table Table, col Column) error
}

type Table interface {
	TableName() string
	Model() attrs.Definer
	Columns() []*Column
	Comment() string
	Indexes() []Index
}

// Embed this struct in your model to prevent it from being migrated.
type CantMigrate struct{}

func (c *CantMigrate) CanMigrate() bool {
	return false
}

func CanMigrate(obj attrs.Definer) bool {
	var meta = attrs.GetModelMeta(obj)
	if meta == nil {
		return false
	}

	if canMigrator, ok := obj.(CanModelMigrate); ok {
		return canMigrator.CanMigrate()
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
	}
	return false
}
