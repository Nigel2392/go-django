package migrator

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
	"github.com/Nigel2392/go-django/src/core/errs"
	"github.com/Nigel2392/go-django/src/core/filesystem"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/elliotchance/orderedmap/v2"
	"github.com/pkg/errors"
)

const (
	MIGRATION_FILE_SUFFIX = ".mig"

	ErrNoChanges errs.Error = "migrations not created, no changes detected."
)

type Dependency struct {
	AppName   string
	ModelName string
	Name      string
}

func (d *Dependency) MarshalJSON() ([]byte, error) {
	var s = strings.Join([]string{d.AppName, d.ModelName, d.Name}, ":")
	return json.Marshal(s)
}

func (d *Dependency) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return errors.Wrap(err, "failed to unmarshal dependency")
	}

	var parts = strings.SplitN(str, ":", 3)
	if len(parts) != 3 {
		return fmt.Errorf("invalid dependency format: %q", str)
	}

	d.AppName = parts[0]
	d.ModelName = parts[1]
	d.Name = parts[2]
	return nil
}

type MigrationFile struct {
	// The name of the application for this migration.
	//
	// This is used to identify the application that the migration is for.
	AppName string `json:"-"`

	// The name of the model for this migration.
	//
	// This is used to identify the model that the migration is for.
	ModelName string `json:"-"`

	// The name of the migration file.
	//
	// This is used to identify the migration and apply it in the correct order.
	Name string `json:"-"`

	// The order of the migration file.
	//
	// This is used to ensure that the migrations are applied in the correct order.
	Order int `json:"-"`

	// ContentType is the content type for the model of this migration.
	//
	// This is used to identify the model that the migration is for.
	ContentType *contenttypes.BaseContentType[attrs.Definer] `json:"-"`

	// Dependencies are the migration files that this migration depends on.
	//
	// This is used to ensure that the migrations are applied in the correct order.
	// If a migration file has dependencies, it will not be applied until all of its dependencies have been applied.
	Dependencies []Dependency `json:"dependencies,omitempty"`

	// Lazy dependencies exist in case a relation is lazy loaded - these get special handling to ensure
	// proper dependency management, as the model should be considered "generic", i.e. for defining 2 separate user models,
	// but maintaining the same relation.
	//
	// These lazy dependencies will be loaded using contenttypes.LoadModel().
	LazyDependencies []string `json:"lazy_dependencies,omitempty"`

	// The SQL commands to be executed in the
	// migration file.
	//
	// This is used to apply the migration to the database.
	Table *ModelTable `json:"table"`

	// Actions are the actions that have been taken in this migration file.
	//
	// This is used to keep track of the actions that have been taken in the migration file.
	// These actions are used to generate the migration file name, and can be used to
	// migrate the database to a different state.
	Actions []MigrationAction `json:"actions"`
}

func (m *MigrationFile) String() string {
	return fmt.Sprintf("%s:%s:%s", m.AppName, m.ModelName, m.FileName())
}

func (m *MigrationFile) GoString() string {
	return fmt.Sprintf("%s:%s:%s", m.AppName, m.ModelName, m.FileName())
}

func (m *MigrationFile) addDependency(appName, modelName, name string) {
	if m.Dependencies == nil {
		m.Dependencies = make([]Dependency, 0)
	}
	m.Dependencies = append(m.Dependencies, Dependency{
		AppName:   appName,
		ModelName: modelName,
		Name:      name,
	})
}

func (m *MigrationFile) addLazyDependency(modelKey string) {
	if m.LazyDependencies == nil {
		m.LazyDependencies = make([]string, 0)
	}
	if !slices.Contains(m.LazyDependencies, modelKey) {
		m.LazyDependencies = append(m.LazyDependencies, modelKey)
	}
}

func (m *MigrationFile) addAction(actionType ActionType, table *Changed[*ModelTable], column *Changed[*Column], index *Changed[*Index]) {
	if m.Actions == nil {
		m.Actions = make([]MigrationAction, 0)
	}
	m.Actions = append(m.Actions, MigrationAction{
		ActionType: actionType,
		Table:      table,
		Field:      column,
		Index:      index,
	})
}

func (m *MigrationFile) FileName() string {
	return generateMigrationFileName(m)
}

type MigrationLog interface {
	Log(action ActionType, file *MigrationFile, table *Changed[*ModelTable], column *Changed[*Column], index *Changed[*Index])
}

type MigrationEngine struct {
	// BasePath is the path to the migration directory where migration files are stored.
	//
	// This is used to read and write migration files.
	BasePath string

	// Fake indicates whether the migration engine is in fake mode.
	//
	// If true, the migration engine will not apply any migrations to the database,
	// nor will it write any migration files.
	//
	// It is useful for testing purposes or when you want to see what migrations would be applied without actually applying them.
	Fake bool

	// The path to the directory where the migration files are stored.
	//
	// This is used to load the migration files and apply them to the database.
	Paths []string

	// SchemaEditor is the schema editor used to apply the migrations to the database.
	//
	// This is used to execute SQL commands for creating, modifying, and deleting tables and columns.
	SchemaEditor SchemaEditor

	// MigrationFilesystems is the list of migration filesystems used to load the migration files.
	//
	// It is a map of application names to a slice of fs.FileSystem interfaces.
	MigrationFilesystems map[string]fs.FS

	// Migrations is the list of migration files that have been applied to the database.
	//
	// This is used to keep track of the migrations that have been applied and ensure that they are not applied again.
	Migrations map[string]map[string][]*MigrationFile

	// MigrationLog is the migration log used to log the actions taken by the migration engine.
	//
	// This is used to log the actions taken by the migration engine for debugging and auditing purposes.
	MigrationLog MigrationLog

	// dependencies is a map of migration files used for dependency resolution.
	//
	// This is used to ensure that the migrations are applied in the correct order.
	dependencies map[string]map[string][]*MigrationFile

	// apps is an ordered map of applications used for dependency resolution and migrations.
	//
	// The apps contain a slice of models that are used to generate the migration files.
	apps *orderedmap.OrderedMap[string, django.AppConfig]
}

func (m *MigrationEngine) Dirs() []string {
	var dirs = make([]string, 0, len(m.Paths)+1)
	dirs = append(dirs, m.BasePath)
	dirs = append(dirs, m.Paths...)
	return dirs
}

func defaultOptions(options []EngineOption) []EngineOption {
	if len(options) == 0 {
		return []EngineOption{
			EngineOptionApps(),
		}
	}
	return options
}

func NewMigrationEngine(path string, schemaEditor SchemaEditor, opts ...EngineOption) *MigrationEngine {

	var engine = &MigrationEngine{
		BasePath:     path,
		SchemaEditor: schemaEditor,
	}

	for _, opt := range defaultOptions(opts) {
		opt(engine)
	}

	return engine
}

func (m *MigrationEngine) Log(action ActionType, file *MigrationFile, table *Changed[*ModelTable], column *Changed[*Column], index *Changed[*Index]) {
	if m.MigrationLog == nil {
		return
	}
	m.MigrationLog.Log(action, file, table, column, index)
}

// GetLastMigration returns the last applied migration for the given app and model.
func (m *MigrationEngine) GetLastMigration(appName, modelName string) *MigrationFile {
	return latestFromMap(m.Migrations, appName, modelName)
}

func (m *MigrationEngine) Migrate(ctx context.Context, apps ...string) error {

	if err := m.SchemaEditor.Setup(ctx); err != nil {
		return errors.Wrap(err, "failed to setup schema editor")
	}

	var migrations, err = m.ReadMigrations(apps...)
	if err != nil {
		return errors.Wrap(err, "failed to read migrations")
	}

	var migrationInfos = make([]*migrationFileInfo, 0, len(migrations))
	var wereApplied int
	for _, migration := range migrations {
		var hasApplied, err = m.SchemaEditor.HasMigration(
			ctx,
			migration.AppName,
			migration.ModelName,
			migration.FileName(),
		)

		if err != nil {
			return errors.Wrapf(
				err, "failed to check if migration %q has been applied", migration.Name,
			)
		}

		migrationInfos = append(migrationInfos, &migrationFileInfo{
			MigrationFile: migration,
			migrated:      hasApplied,
		})

		if hasApplied {
			wereApplied++
		}
	}

	if wereApplied == len(migrationInfos) {
		return ErrNoChanges
	}

	logger.Debugf("Found %d migrations to apply", len(migrationInfos)-wereApplied)

	m.Migrations = make(map[string]map[string][]*MigrationFile)
	m.dependencies = make(map[string]map[string][]*MigrationFile)
	for _, migration := range migrations {
		m.storeMigration(migration)
	}

	graph, err := m.buildDependencyGraph(migrationInfos)
	if err != nil {
		return err
	}

	for _, n := range graph {

		if n.mig.migrated {
			logger.Debugf("migration %s:%s:%s has already been applied, skipping", n.mig.AppName, n.mig.ModelName, n.mig.FileName())
			continue
		}

		if has, err := m.SchemaEditor.HasMigration(ctx, n.mig.AppName, n.mig.ModelName, n.mig.FileName()); err != nil {
			logger.Errorf("failed to check if migration %q has been applied: %v", n.mig.FileName(), err)
			continue
		} else if has {
			logger.Infof("migration %s has already been applied", n.mig.FileName())
			continue
		}

		var defs = n.mig.Table.Object.FieldDefs()

		if m.Fake {
			logger.Debugf(
				"Skipping migration %q for model %s.%s in fake mode",
				n.mig.FileName(), n.mig.AppName, n.mig.ModelName,
			)
			continue
		}

		for _, action := range n.mig.Actions {
			var err error

			switch action.ActionType {
			case ActionCreateTable:
				err = m.SchemaEditor.CreateTable(ctx, n.mig.Table, false)
			case ActionDropTable:
				err = m.SchemaEditor.DropTable(ctx, action.Table.Old, false)
			case ActionRenameTable:
				err = m.SchemaEditor.RenameTable(ctx, action.Table.Old, action.Table.New.TableName())
			case ActionAddField:
				if !action.Field.New.UseInDB {
					continue
				}
				action.Field.New.Table = n.mig.Table
				action.Field.New.Field, _ = defs.Field(action.Field.New.Name)
				err = m.SchemaEditor.AddField(ctx, n.mig.Table, *action.Field.New)
			case ActionAlterField:
				if !(action.Field.New.UseInDB && action.Field.Old.UseInDB) {
					continue
				}
				action.Field.Old.Table = n.mig.Table
				action.Field.Old.Field, _ = defs.Field(action.Field.Old.Name)
				action.Field.New.Table = n.mig.Table
				action.Field.New.Field, _ = defs.Field(action.Field.New.Name)
				err = m.SchemaEditor.AlterField(ctx, n.mig.Table, *action.Field.Old, *action.Field.New)
			case ActionRemoveField:
				if !action.Field.Old.UseInDB {
					continue
				}
				action.Field.Old.Table = n.mig.Table
				action.Field.Old.Field, _ = defs.Field(action.Field.Old.Name)
				err = m.SchemaEditor.RemoveField(ctx, n.mig.Table, *action.Field.Old)
			case ActionAddIndex:
				action.Index.New.table = n.mig.Table
				err = m.SchemaEditor.AddIndex(ctx, n.mig.Table, *action.Index.New, false)
			case ActionDropIndex:
				action.Index.Old.table = n.mig.Table
				err = m.SchemaEditor.DropIndex(ctx, n.mig.Table, *action.Index.Old, false)
			case ActionRenameIndex:
				action.Index.Old.table = n.mig.Table
				action.Index.New.table = n.mig.Table
				err = m.SchemaEditor.RenameIndex(ctx, n.mig.Table, action.Index.Old.Name(), action.Index.New.Name())
			// case ActionAlterUniqueTogether:
			// 	err = m.SchemaEditor.AlterUniqueTogether(action.Table.New, action.Field.New.Unique)
			// case ActionAlterIndexTogether:
			// 	err = m.SchemaEditor.AlterIndexTogether(action.Table.New, action.Field.New.Index)
			default:
				return fmt.Errorf("unknown action type %d", action.ActionType)
			}

			if err != nil {
				return errors.Wrapf(
					err, "failed to apply migration %q", n.mig.Name,
				)
			}
		}
		err = m.SchemaEditor.StoreMigration(
			ctx,
			n.mig.AppName,
			n.mig.ModelName,
			n.mig.FileName(),
		)
		if err != nil {
			return errors.Wrapf(
				err, "failed to store migration %q", n.mig.Name,
			)
		}
	}

	return nil
}

type NeedsToMigrateInfo struct {
	model *contenttypes.BaseContentType[attrs.Definer]
	mig   *MigrationFile
	app   django.AppConfig
}

func (n *NeedsToMigrateInfo) Model() attrs.Definer {
	if n.model == nil {
		return nil
	}
	return n.model.New()
}

func (n *NeedsToMigrateInfo) Migration() *MigrationFile {
	return n.mig
}

func (n *NeedsToMigrateInfo) App() django.AppConfig {
	return n.app
}

func (m *MigrationEngine) NeedsToMigrate(ctx context.Context, apps ...string) ([]NeedsToMigrateInfo, error) {

	if len(apps) == 0 {
		apps = m.apps.Keys()
	}

	if err := m.SchemaEditor.Setup(ctx); err != nil {
		return nil, errors.Wrap(err, "failed to setup schema editor")
	}

	var migrations, err = m.ReadMigrations(apps...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read migrations")
	}

	m.Migrations = make(map[string]map[string][]*MigrationFile)
	m.dependencies = make(map[string]map[string][]*MigrationFile)
	for _, migration := range migrations {
		m.storeMigration(migration)
	}

	var needsToMigrate = make([]NeedsToMigrateInfo, 0)
	for _, appName := range apps {
		var app, ok = m.apps.Get(appName)
		if !ok || app == nil {
			return nil, fmt.Errorf("app %q not found in migration engines' apps list", appName)
		}

		for _, model := range app.Models() {
			var cType = contenttypes.NewContentType(model)
			var modelName = cType.Model()

			var last = m.GetLastMigration(appName, modelName)
			if last == nil {
				logger.Debugf("No migrations found for %s.%s", appName, modelName)
				continue
			}

			var hasApplied, err = m.SchemaEditor.HasMigration(
				ctx,
				last.AppName,
				last.ModelName,
				last.FileName(),
			)
			if err != nil {
				return nil, errors.Wrapf(
					err, "failed to check if migration %q has been applied", last.Name,
				)
			}

			if !hasApplied {
				needsToMigrate = append(needsToMigrate, NeedsToMigrateInfo{
					model: cType,
					mig:   last,
					app:   app,
				})
			}
		}
	}

	return needsToMigrate, nil
}

func (m *MigrationEngine) NeedsToMakeMigrations(ctx context.Context, apps ...string) ([]*contenttypes.BaseContentType[attrs.Definer], error) {

	if len(apps) == 0 {
		apps = m.apps.Keys()
	}

	if err := m.SchemaEditor.Setup(ctx); err != nil {
		return nil, errors.Wrap(err, "failed to setup schema editor")
	}

	var migrations, err = m.ReadMigrations(apps...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read migrations")
	}

	m.Migrations = make(map[string]map[string][]*MigrationFile)
	m.dependencies = make(map[string]map[string][]*MigrationFile)
	for _, migration := range migrations {
		m.storeMigration(migration)
	}

	var needsToMigrate = make([]*contenttypes.BaseContentType[attrs.Definer], 0)
	for _, appName := range apps {
		var app, ok = m.apps.Get(appName)
		if !ok || app == nil {
			return nil, fmt.Errorf("app %q not found in migration engines' apps list", appName)
		}

		for _, model := range app.Models() {
			var cType = contenttypes.NewContentType(model)
			var modelName = cType.Model()

			// Build current table state
			var currTable = NewModelTable(cType.New())

			// Compare to last migration
			var mig, err = m.NewMigration(appName, modelName, currTable, cType)
			if err != nil {
				return nil, fmt.Errorf("MakeMigrations: failed to generate migration for %s: %w", modelName, err)
			}

			var last = m.GetLastMigration(
				mig.AppName, mig.ModelName,
			)
			var newMigrationNeeded bool = true
			newMigrationNeeded = m.makeMigrationDiff(
				mig, last, mig.Table,
			)

			if newMigrationNeeded {
				needsToMigrate = append(needsToMigrate, cType)
			}
		}
	}

	return needsToMigrate, nil
}

func (m *MigrationEngine) MakeMigrations(ctx context.Context, apps ...string) error {

	if len(apps) == 0 {
		apps = m.apps.Keys()
	}

	if err := m.SchemaEditor.Setup(ctx); err != nil {
		return errors.Wrap(err, "failed to setup schema editor")
	}

	var migrations, err = m.ReadMigrations(apps...)
	if err != nil {
		return errors.Wrap(err, "failed to read migrations")
	}

	m.Migrations = make(map[string]map[string][]*MigrationFile)
	m.dependencies = make(map[string]map[string][]*MigrationFile)
	for _, migration := range migrations {
		m.storeMigration(migration)
	}

	var (
		migrationList   = make([]*MigrationFile, 0)
		dependencies    = make(map[string]map[string]*MigrationFile)
		migrationsFound = false
	)

	for _, appName := range apps {
		var app, ok = m.apps.Get(appName)
		if !ok || app == nil {
			return fmt.Errorf("app %q not found in migration engines' apps list", appName)
		}

		for _, model := range app.Models() {

			if !CheckCanMigrate(model) {
				logger.Debugf("Skipping model %T, migrations are disabled", model)
				continue
			}

			var cType = contenttypes.NewContentType(model)
			var modelName = cType.Model()

			var appLabel = appName
			var model = modelName

			// Build current table state
			var currTable = NewModelTable(cType.New())

			// Compare to last migration
			var mig, err = m.NewMigration(appLabel, model, currTable, cType)
			if err != nil {
				return fmt.Errorf("MakeMigrations: failed to generate migration for %s: %w", modelName, err)
			}

			var last = m.GetLastMigration(mig.AppName, mig.ModelName)
			if !m.makeMigrationDiff(mig, last, mig.Table) {
				logger.Debugf(
					"No changes detected for model %s.%s, (checked %s), skipping migration",
					mig.AppName, mig.ModelName, last.FileName(),
				)
				continue
			}

			migrationsFound = true

			mig.Name = generateMigrationFileName(mig)

			logger.Debugf(
				"Creating migration file %q for model \"%s.%s\"",
				mig.FileName(), mig.AppName, mig.ModelName,
			)

			m.storeDependency(mig)

			migrationList = append(migrationList, mig)
			if dependencies[mig.AppName] == nil {
				dependencies[mig.AppName] = make(map[string]*MigrationFile)
			}
			dependencies[mig.AppName][mig.ModelName] = mig
		}
	}

	if !migrationsFound {
		return ErrNoChanges
	}

	// Check for dependencies and write migration files
	for _, mig := range migrationList {

		var cols = mig.Table.Columns()
		if len(cols) == 0 && len(mig.Actions) == 0 {
			continue
		}

		// Check for dependencies
	colLoop:
		for _, col := range cols {

			if col.Rel != nil {

				if col.Rel.IsLazy() { // translates to `m.TargetModel.LazyModelKey != ""`
					var relModelKey = col.Rel.TargetModel.LazyModelKey
					mig.addLazyDependency(relModelKey)
				} else {
					var relApp = getModelApp(col.Rel.TargetModel.Object())
					if relApp == nil {
						//	logger.Warnf(
						//		"Could not find app for model %T, related to migration %q/%q",
						//		col.Rel.TargetModel.New(), mig.AppName, mig.ModelName,
						//	)
						//	continue colLoop
						return fmt.Errorf(
							"could not find app for model %T, related to migration %q/%q",
							col.Rel.TargetModel.Object(), mig.AppName, mig.ModelName,
						)
					}

					var (
						relAppName  = relApp.Name()
						relModel    = col.Rel.TargetModel.ContentType().Model()
						depMigs, ok = m.dependencies[relAppName][relModel]
						// depMig, ok = dependencies[relAppName][relModel]
					)
					if !ok || len(depMigs) == 0 {
						logger.Warnf(
							"Dependency %q/%q not found for migration %q/%q",
							relAppName, relModel, mig.AppName, mig.ModelName,
						)
						continue
					}

					// walk old migrations and check if the latest dependency
					// migration is already in the list of dependencies of
					// one of the older migrations
					var (
						depMig    = depMigs[len(depMigs)-1]
						appMigs   map[string][]*MigrationFile
						modelMigs []*MigrationFile
					)

					appMigs, ok = m.dependencies[mig.AppName]
					if !ok {
						goto addDep
					}

					modelMigs, ok = appMigs[mig.ModelName]
					if !ok {
						goto addDep
					}

					// check if the dependency migration is already in the list of dependencies
					// of the current models' migration files
					if len(modelMigs) >= 2 {
						var modelMigs = modelMigs[:len(modelMigs)-1]
						for i := len(modelMigs) - 1; i >= 0; i-- {
							var mig = modelMigs[i]
							for _, oldDep := range mig.Dependencies {
								if oldDep.AppName == relAppName && oldDep.ModelName == relModel && oldDep.Name == depMig.FileName() {
									continue colLoop
								}
							}
						}
					}

					// add dependency to the migration file
				addDep:
					logger.Debugf(
						"Adding dependency \"%s/%s\" to migration \"%s/%s\"",
						depMig.ModelName, depMig.FileName(), mig.ModelName, mig.FileName(),
					)
					mig.addDependency(relAppName, relModel, depMig.FileName())
				}
			}
		}

		// Write the migration file
		if m.Fake {
			logger.Debugf(
				"Skipping writing migration file %q for model %s.%s in fake mode",
				mig.FileName(), mig.AppName, mig.ModelName,
			)
			continue
		}

		if err := m.WriteMigration(mig); err != nil {
			return err
		}
	}

	return nil
}

type node struct {
	mig      *migrationFileInfo
	deps     []*node
	visited  bool
	visiting bool
}

type migrationFileInfo struct {
	*MigrationFile
	migrated bool
}

func (m *MigrationEngine) buildDependencyGraph(migrations []*migrationFileInfo) ([]*node, error) {
	var nodeMap = make(map[string]*node)

	// helper to create a unique key for lookup
	var key = func(m *migrationFileInfo) string {
		return fmt.Sprintf("%s:%s:%s", m.AppName, m.ModelName, m.FileName())
	}

	// Step 1: Create node for each migration
	for _, m := range migrations {
		nodeMap[key(m)] = &node{mig: m}
	}

	// Step 2: Link dependencies
	for _, n := range nodeMap {
		// dependencyLoop:
		for _, dep := range n.mig.Dependencies {
			depKey := fmt.Sprintf("%s:%s:%s", dep.AppName, dep.ModelName, dep.Name)
			depNode, ok := nodeMap[depKey]
			if !ok {

				//var appMigs, ok = m.dependencies[n.mig.AppName]
				//if !ok {
				//	return nil, fmt.Errorf("dependency %q not found", depKey)
				//}

				//modelMigs, ok := appMigs[n.mig.ModelName]
				//if ok && len(modelMigs) >= 2 {
				//	var modelMigs = modelMigs[:len(modelMigs)-1]
				//	for i := len(modelMigs) - 1; i >= 0; i-- {
				//		var mig = modelMigs[i]
				//		for _, oldDep := range mig.Dependencies {
				//			if oldDep.AppName == dep.AppName && oldDep.ModelName == dep.ModelName {
				//				continue dependencyLoop
				//			}
				//		}
				//	}
				//}

				// ------- comment this out to ignore missing dependencies
				//deps, ok := m.dependencies[dep.AppName]
				//if !ok {
				//	return nil, fmt.Errorf("dependency %q not found", depKey)
				//}
				//depMigs, ok := deps[dep.ModelName]
				//if !ok || len(depMigs) == 0 {
				return nil, fmt.Errorf("dependency %q not found for migration %q/%q: %+v", depKey, n.mig.AppName, n.mig.ModelName, nodeMap)
				//}
				//depNode = &node{mig: depMigs[len(depMigs)-1]}
				// ------- comment this out to ignore missing dependencies
			}
			n.deps = append(n.deps, depNode)
		}

		for _, lazyDep := range n.mig.LazyDependencies {
			// lazy dependencies are loaded using contenttypes.LoadModel()
			var lazyModel = contenttypes.LoadModel(lazyDep)
			var app = getModelApp(lazyModel.Object())
			if app == nil {
				return nil, fmt.Errorf("could not find app for lazy model %T, related to migration %q/%q", lazyModel.Object(), n.mig.AppName, n.mig.ModelName)
			}

			var cType = contenttypes.NewContentType(lazyModel.Object())
			if cType == nil {
				return nil, fmt.Errorf("could not create content type for lazy model %T, related to migration %q/%q", lazyModel.Object(), n.mig.AppName, n.mig.ModelName)
			}

			var lastMigration = m.GetLastMigration(app.Name(), cType.Model())
			var depKey = fmt.Sprintf("%s:%s:%s", app.Name(), cType.Model(), lastMigration.FileName())
			depNode, ok := nodeMap[depKey]
			if !ok {
				return nil, fmt.Errorf("lazy dependency %q not found for migration %q/%q", depKey, n.mig.AppName, n.mig.ModelName)
			}
			n.deps = append(n.deps, depNode)
		}
	}

	// Step 3: Topological sort
	var ordered []*node
	var visit func(n *node) error
	visit = func(n *node) error {
		if n.visited {
			return nil
		}
		if n.visiting {
			// return nil
			return fmt.Errorf("cyclic dependency detected for migration: %s", key(n.mig))
		}
		n.visiting = true
		for _, dep := range n.deps {
			if err := visit(dep); err != nil {
				return err
			}
		}
		n.visited = true
		n.visiting = false
		ordered = append(ordered, n)
		return nil
	}

	for _, n := range nodeMap {
		if !n.visited {
			if err := visit(n); err != nil {
				return nil, err
			}
		}
	}

	return ordered, nil
}

// makeMigrationDiff diffs the last migration with the current table state and returns true if a migration is needed.
func (m *MigrationEngine) makeMigrationDiff(migration *MigrationFile, last *MigrationFile, table *ModelTable) (shouldMigrate bool) {
	if last == nil || last.Table == nil {
		migration.addAction(ActionCreateTable, nil, nil, nil)
		m.Log(ActionCreateTable, migration, unchanged(table), nil, nil)

		for _, idx := range table.Indexes() {
			migration.addAction(ActionAddIndex, nil, nil, changed(nil, &idx))
			m.Log(ActionAddIndex, migration, unchanged(table), nil, unchanged(&idx))
		}

		return true
	}

	var lastAppliedTable = last.Table
	if table == nil {
		migration.addAction(ActionDropTable, changed(lastAppliedTable, nil), nil, nil)
		m.Log(ActionDropTable, migration, unchanged(lastAppliedTable), nil, nil)
		return true
	}

	if lastAppliedTable.TableName() != table.TableName() {
		migration.addAction(ActionRenameTable, changed(lastAppliedTable, table), nil, nil)
		m.Log(ActionRenameTable, migration, changed(lastAppliedTable, table), nil, nil)
		shouldMigrate = true
	}

	var added, removed, diffs = table.Diff(lastAppliedTable)

	for _, col := range added {
		migration.addAction(ActionAddField, nil, unchanged(&col), nil)
		m.Log(ActionAddField, migration, unchanged(table), unchanged(&col), nil)
		shouldMigrate = true
	}

	for _, col := range removed {
		migration.addAction(ActionRemoveField, nil, changed(&col, nil), nil)
		m.Log(ActionRemoveField, migration, unchanged(table), changed(&col, nil), nil)
		shouldMigrate = true
	}

	for _, col := range diffs {
		migration.addAction(ActionAlterField, nil, changed(&col.Old, &col.New), nil)
		m.Log(ActionAlterField, migration, unchanged(table), changed(&col.Old, &col.New), nil)
		shouldMigrate = true
	}

	var (
		oldIndexes = lastAppliedTable.Indexes()
		newIndexes = table.Indexes()
		oldMap     = make(map[string]Index, len(oldIndexes))
		newMap     = make(map[string]Index, len(newIndexes))
	)

	for _, idx := range oldIndexes {
		oldMap[idx.Name()] = idx
	}
	for _, idx := range newIndexes {
		newMap[idx.Name()] = idx
	}

	// Drop removed or changed indexes
	for name, oldIdx := range oldMap {
		var newIdx, exists = newMap[name]
		if !exists || !indexesEqual(oldIdx, newIdx) {
			migration.addAction(ActionDropIndex, nil, nil, changed(&oldIdx, nil))
			m.Log(ActionDropIndex, migration, unchanged(table), nil, changed(&oldIdx, nil))
			shouldMigrate = true
		}
	}

	// Add new or changed indexes
	for name, newIdx := range newMap {
		var oldIdx, exists = oldMap[name]
		if !exists || !indexesEqual(oldIdx, newIdx) {
			migration.addAction(ActionAddIndex, nil, nil, changed(nil, &newIdx))
			m.Log(ActionAddIndex, migration, unchanged(table), nil, unchanged(&newIdx))
			shouldMigrate = true
		}
	}

	// Detect and rename matching indexes with different names
	for oldName, oldIdx := range oldMap {
		for newName, candidate := range newMap {
			if indexesEqual(oldIdx, candidate) && oldName != newName {
				delete(oldMap, oldName)
				delete(newMap, newName)
				migration.addAction(ActionRenameIndex, nil, nil, changed(&oldIdx, &candidate))
				m.Log(ActionRenameIndex, migration, unchanged(table), nil, changed(&oldIdx, &candidate))
				shouldMigrate = true
				break
			}
		}
	}

	return shouldMigrate
}

func (e *MigrationEngine) NewMigration(appName, modelName string, newTable *ModelTable, def *contenttypes.BaseContentType[attrs.Definer]) (*MigrationFile, error) {
	// load latest applied migration if it exists
	var last = e.GetLastMigration(appName, modelName)

	// Get last order
	var nextOrder = 1
	if last != nil {
		nextOrder = last.Order + 1
	}

	// Name this migration something useful later on
	var name = "auto_generated"

	// Build tables map
	return &MigrationFile{
		AppName:     appName,
		ModelName:   modelName,
		ContentType: def,
		Name:        name,
		Order:       nextOrder,
		Table:       newTable,
	}, nil
}

// store a dependency in the migration map
//
// this is used to keep track of the dependencies between migration files
// so that they can be applied in the correct order
func (m *MigrationEngine) storeDependency(mig *MigrationFile) {
	if m.dependencies == nil {
		m.dependencies = make(map[string]map[string][]*MigrationFile)
	}

	storeInMap(m.dependencies, mig)
}

// storeMigration stores the migration file in the migration map.
//
// it will also automatically store a copy of the migration file in the dependencies
func (m *MigrationEngine) storeMigration(mig *MigrationFile) {
	if m.Migrations == nil {
		m.Migrations = make(map[string]map[string][]*MigrationFile)
	}

	storeInMap(m.Migrations, mig)

	m.storeDependency(mig)
}

func latestFromMap(m map[string]map[string][]*MigrationFile, appName, modelName string) *MigrationFile {
	if m == nil {
		return nil
	}

	var appMigrations, ok = m[appName]
	if !ok {
		return nil
	}

	modelMigrations, ok := appMigrations[modelName]
	if !ok || len(modelMigrations) == 0 {
		return nil
	}

	return modelMigrations[len(modelMigrations)-1]
}

func storeInMap(m map[string]map[string][]*MigrationFile, mig *MigrationFile) {
	var appMigrations, ok = m[mig.AppName]
	if !ok {
		appMigrations = make(map[string][]*MigrationFile)
		m[mig.AppName] = appMigrations
	}

	modelMigrations, ok := appMigrations[mig.ModelName]
	if !ok {
		modelMigrations = make([]*MigrationFile, 0)
		appMigrations[mig.ModelName] = modelMigrations
	}

	modelMigrations = append(modelMigrations, mig)
	appMigrations[mig.ModelName] = modelMigrations
}

func indexesEqual(a, b Index) bool {
	if a.Name() != b.Name() || a.Unique != b.Unique || a.Type != b.Type {
		return false
	}

	if len(a.Fields) != len(b.Fields) {
		return false
	}

	for i := range a.Fields {
		if a.Fields[i] != b.Fields[i] {
			return false
		}
	}

	return true
}

// WriteMigration writes the migration file to the specified path.
//
// The migration file is used to apply the migrations to the database.
func (e *MigrationEngine) WriteMigration(migration *MigrationFile) error {
	var filePath = filepath.Join(e.BasePath, migration.AppName, migration.ModelName, migration.FileName())

	if _, err := os.Stat(filePath); err == nil {
		return fmt.Errorf("migration file %q already exists", filePath)
	}

	var data, err = json.MarshalIndent(migration, "", "  ")
	if err != nil {
		return errors.Wrapf(err, "failed to marshal migration file %q", filePath)
	}

	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return errors.Wrapf(err, "failed to create directory %q", filepath.Dir(filePath))
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return errors.Wrapf(err, "failed to write migration file %q", filePath)
	}

	return nil
}

// ReadMigrations reads the migration files from the specified path and returns a list of migration files.
//
// These migration files are used to apply the migrations to the database.
func (e *MigrationEngine) ReadMigrations(apps ...string) ([]*MigrationFile, error) {

	os.MkdirAll(e.BasePath, 0755)

	if len(apps) == 0 {
		apps = e.apps.Keys()
	}

	var migrations = make([]*MigrationFile, 0)
	for _, appName := range apps {
		app, ok := e.apps.Get(appName)
		if !ok || app == nil {
			return nil, fmt.Errorf("app %q not found in migration engines' apps list", appName)
		}

		fSys, ok := e.MigrationFilesystems[appName]
		if !ok {
			// no actual issue, just means no special filesystem for this app
			//	logger.Debugf(
			//		"Skip reading migrations FS for app %q, no migration filesystem found", appName,
			//	)
			continue
		}

		var models = app.Models()
		for _, model := range models {

			var cType = contenttypes.NewContentType(model)
			var modelMigrationPath = cType.Model()

			modelMigrationPath = filepath.FromSlash(modelMigrationPath)
			modelMigrationPath = filepath.ToSlash(modelMigrationPath)

			var modelMigrationDir, err = fs.ReadDir(fSys, modelMigrationPath)
			if err != nil && !errors.Is(err, fs.ErrNotExist) {
				return nil, errors.Wrapf(
					err, "failed to read migration directory %q", modelMigrationPath,
				)
			} else if err != nil || len(modelMigrationDir) == 0 {
				continue
			}

			migrationFiles, err := e.readMigrationDirFS(
				fSys, modelMigrationPath, appName, cType.Model(),
			)
			if err != nil {
				return nil, errors.Wrapf(
					err, "failed to read app's migration directory %q", modelMigrationPath,
				)
			}

			migrations = append(migrations, migrationFiles...)
		}
	}

	var mfs = filesystem.NewMultiFS()
	for _, dir := range e.Dirs() {
		var path = filepath.FromSlash(dir)
		path = filepath.ToSlash(path)
		mfs.Add(os.DirFS(path), nil)
	}

	var directories, err = fs.ReadDir(mfs, ".")
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return []*MigrationFile{}, nil
	} else if err != nil {
		return nil, errors.Wrapf(
			err, "failed to read migration directories %v", e.Dirs(),
		)
	}

	for _, appMigrationDir := range directories {
		if !appMigrationDir.IsDir() {
			continue
		}

		var workingPath = appMigrationDir.Name()
		if _, err = fs.Stat(mfs, workingPath); err != nil && !errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, errors.Wrapf(
				err, "failed to stat migration directory %q", workingPath,
			)
		}

		var files, err = fs.ReadDir(mfs, workingPath)
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			continue
		} else if err != nil {
			return nil, errors.Wrapf(
				err, "failed to read migration directory %q", workingPath,
			)
		}

		if _, ok := e.apps.Get(appMigrationDir.Name()); !ok {
			logger.Debugf(
				"Skip reading migrations for app %q, not found in migration engines' apps list", appMigrationDir.Name(),
			)
			// no actual issue, just means no migrations for this app, or the app is currently not registered
			// in the migration engine, so we skip it
			continue
			// panic(fmt.Sprintf("app %q not found in migration engines' apps list", appMigrationDir.Name()))
		}

		for _, modelMigrationDir := range files {
			if !modelMigrationDir.IsDir() {
				continue
			}

			var model, ok = getModelByApp(appMigrationDir.Name(), modelMigrationDir.Name())
			if model == nil || !ok {
				logger.Debugf(
					"Skip reading migrations for model %q.%q, not found in migration engines' apps list",
					appMigrationDir.Name(), modelMigrationDir.Name(),
				)
				continue
			}

			var filesDir = filepath.Join(workingPath, modelMigrationDir.Name())
			migrationFiles, err := e.readMigrationDirFS(
				mfs, filesDir, appMigrationDir.Name(), modelMigrationDir.Name(),
			)
			if err != nil {
				return nil, errors.Wrapf(
					err, "failed to read migration directory %q for app %q", filesDir,
					appMigrationDir.Name(),
				)
			}

			migrations = append(migrations, migrationFiles...)
		}
	}

	slices.SortStableFunc(migrations, func(a, b *MigrationFile) int {
		if a.Order < b.Order {
			return -1
		}
		if a.Order > b.Order {
			return 1
		}
		return 0
	})

	return migrations, nil
}

func (e *MigrationEngine) readMigrationDirFS(dir fs.FS, dirPath, appName, modelName string) ([]*MigrationFile, error) {
	dirPath = filepath.FromSlash(dirPath)
	dirPath = filepath.ToSlash(dirPath)

	var files, err = fs.ReadDir(dir, dirPath)
	if err != nil && errors.Is(err, fs.ErrNotExist) {
		return []*MigrationFile{}, nil
	} else if err != nil {
		return nil, errors.Wrapf(
			err, "failed to read migration directory %q", dirPath,
		)
	}

	var migrations = make([]*MigrationFile, 0)
	for _, file := range files {
		var filePath = filepath.Join(
			dirPath, file.Name(),
		)

		if file.IsDir() || filepath.Ext(file.Name()) != MIGRATION_FILE_SUFFIX {
			continue
		}

		filePath = filepath.FromSlash(filePath)
		filePath = filepath.ToSlash(filePath)

		if _, err = fs.Stat(dir, filePath); err != nil && !errors.Is(err, fs.ErrNotExist) {
			continue
		}

		var migrationFileBytes, err = fs.ReadFile(dir, filePath)
		if err != nil {
			return nil, errors.Wrapf(
				err, "failed to read migration file %q", filePath,
			)
		}

		var migrationFile = new(MigrationFile)
		if err := json.Unmarshal(migrationFileBytes, &migrationFile); err != nil {
			return nil, errors.Wrapf(
				err, "failed to unmarshal migration file %q", filePath,
			)
		}

		orderNum, name, err := parseMigrationFileName(file.Name())
		if err != nil {
			return nil, errors.Wrapf(
				err, "failed to parse migration file name %q", file.Name(),
			)
		}

		migrations = append(migrations, &MigrationFile{
			Name:             name,
			AppName:          appName,
			ModelName:        modelName,
			Order:            orderNum,
			Table:            migrationFile.Table,
			Actions:          migrationFile.Actions,
			Dependencies:     migrationFile.Dependencies,
			LazyDependencies: migrationFile.LazyDependencies,
			ContentType:      contenttypes.NewContentType(migrationFile.Table.Object),
		})
	}

	return migrations, nil
}
