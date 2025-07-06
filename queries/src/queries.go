package queries

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"reflect"

	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/expr"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/core/attrs"

	// Register all schema editors for the migrator package.
	_ "github.com/Nigel2392/go-django/queries/src/migrator/sql/mysql"
	_ "github.com/Nigel2392/go-django/queries/src/migrator/sql/postgres"
	_ "github.com/Nigel2392/go-django/queries/src/migrator/sql/sqlite"
)

const (
	// MetaUniqueTogetherKey is the key used to store the unique together
	// fields in the model's metadata.
	//
	// It is used to determine which fields are unique together in the model
	// and can be used to enforce uniqueness, generate SQL clauses for selections,
	// and to generate unique keys for the model in code.
	MetaUniqueTogetherKey = "unique_together"
)

// CanSetup is an interface that can be implemented by models to indicate that
// the model can be set up with a the given model value.
//
// This is mainly useful for embedded models, where the embedded model needs a
// reference to the parent model to be able to set up the relation properly.
type CanSetup interface {
	// Setup is called to set up the model with the given value.
	Setup(value attrs.Definer) error
}

// Setup allows a model to be setup properly if it adheres to the CanSetup interface.
func setup[T attrs.Definer](model T) (T, error) {
	if setupModel, ok := any(model).(CanSetup); ok {
		if err := setupModel.Setup(model); err != nil {
			return model, fmt.Errorf("error setting up model %T: %w", model, err)
		}
	}
	return model, nil
}

// Validator is a generic interface that can be implemented by objects, fields and more
// to indicate that the object can be validated before being saved to the database.
type ContextValidator interface {
	// Validate is called to validate the model before it is saved to the database.
	// It should return an error if the validation fails.
	Validate(ctx context.Context) error
}

// Validate allows a model to be validated before being saved to the database
func Validate(ctx context.Context, validate any) error {

	if validator, ok := validate.(ContextValidator); ok {
		if err := validator.Validate(ctx); err != nil {
			return fmt.Errorf("error validating %T: %w", validate, err)
		}
	}
	return nil
}

type SaveableField interface {
	attrs.Field

	// Save is called to save the field's value to the database.
	// It should return an error if the save operation fails.
	Save(ctx context.Context) error
}

type SaveableDependantField interface {
	attrs.Field

	// Save is called to save the field's value to the database.
	Save(ctx context.Context, parent attrs.Definer) error
}

// A field can adhere to this interface to indicate that the field should be
// aliased when generating the SQL for the field.
//
// For example: this is used in annotations to alias the field name.
type AliasField = expr.AliasField

// A field can adhere to this interface to indicate that the field should be
// rendered as SQL.
//
// For example: this is used in fields.ExpressionField to render the expression as SQL.
type VirtualField = expr.VirtualField

// RelatedField is an interface that can be implemented by fields to indicate
// that the field is a related field.
//
// For example, this is used in fields.RelationField to determine the column name for the target field.
//
// If `GetTargetField()` returns nil, the primary field of the target model should be used instead.
type RelatedField interface {
	attrs.Field

	// This is used to determine the column name for the field, for example for a through table.
	GetTargetField() attrs.Field

	RelatedName() string
}

func getTargetField(f any, targetDefs attrs.Definitions) attrs.Field {
	if f == nil {
		goto retTarget
	}

	if rf, ok := f.(RelatedField); ok {
		if targetField := rf.GetTargetField(); targetField != nil {
			return targetField
		}
	}

retTarget:
	return targetDefs.Primary()
}

type ClauseTarget struct {
	Model attrs.Definer
	Table Table
	Field attrs.FieldDefinition
}

type ThroughClauseTarget struct {
	Model attrs.Definer
	Table Table
	Left  attrs.FieldDefinition
	Right attrs.FieldDefinition
}

type TargetClauseField interface {
	GenerateTargetClause(qs *QuerySet[attrs.Definer], internals *QuerySetInternals, lhs ClauseTarget, rhs ClauseTarget) JoinDef
}

type TargetClauseThroughField interface {
	GenerateTargetThroughClause(qs *QuerySet[attrs.Definer], internals *QuerySetInternals, lhs ClauseTarget, through ThroughClauseTarget, rhs ClauseTarget) (JoinDef, JoinDef)
}

type ProxyField interface {
	attrs.FieldDefinition
	TargetClauseField
	IsProxy() bool
}

type ProxyThroughField interface {
	attrs.FieldDefinition
	TargetClauseThroughField
	IsProxy() bool
}

// ForUseInQueriesField is an interface that can be implemented by fields to indicate
// that the field should be included in the query.
//
// For example, this is used in fields.RelationField to exclude the relation from the query,
// otherwise scanning errors will occur.
//
// This is mostly for fields that do not actually exist in the database, I.E. reverse fk, o2o
type ForUseInQueriesField interface {
	attrs.Field
	// ForUseInQueries returns true if the field is for use in queries.
	// This is used to determine if the field should be included in the query.
	// If the field does not implement this method, it is assumed to be for use in queries.
	ForSelectAll() bool
}

// ForSelectAll returns true if the field should be selected in the query.
//
// If the field is nil, it returns false.
//
// If the field is a ForUseInQueriesField, it returns the result of `ForSelectAll()`.
//
// Otherwise, it returns true.
func ForSelectAll(f attrs.FieldDefinition) bool {
	if f == nil {
		return false
	}
	if f.ColumnName() == "" { // dont select fields without a column name
		if _, ok := f.(AliasField); !ok {
			return false
		}
	}
	if f, ok := f.(ForUseInQueriesField); ok {
		return f.ForSelectAll()
	}
	return true
}

// ForDBEditableField returns true if the field should be saved to the database.
// If the field is nil, it returns false.
type ForDBEditableField interface {
	attrs.Field
	AllowDBEdit() bool
}

// ForDBEdit returns true if the field should be saved to the database.
//
// * If the field is nil, it returns false.
// * If the field has no column name, it returns false.
// * If the field is a ForUseInQueriesField, it checks if it is for select all.
// * If the field implements ForDBEditableField, it checks if it is for edit.
func ForDBEdit(f attrs.FieldDefinition) bool {
	if f == nil {
		return false
	}
	if f.ColumnName() == "" { // dont save fields without a column name
		return false
	}
	if f, ok := f.(ForDBEditableField); ok {
		return f.AllowDBEdit()
	}
	return true
}

func ForSelectAllFields[T any](fields any) []T {
	switch fieldsValue := fields.(type) {
	case []attrs.Field:
		var result = make([]T, 0, len(fieldsValue))
		for _, f := range fieldsValue {
			if ForSelectAll(f) {
				result = append(result, f.(T))
			}
		}
		return result
	case []attrs.FieldDefinition:
		var result = make([]T, 0, len(fieldsValue))
		for _, f := range fieldsValue {
			if ForSelectAll(f) {
				result = append(result, f.(T))
			}
		}
		return result
	case attrs.Definer:
		var defs = fieldsValue.FieldDefs()
		var fields = defs.Fields()
		return ForSelectAllFields[T](fields)
	case attrs.Definitions:
		var fields = fieldsValue.Fields()
		return ForSelectAllFields[T](fields)
	case attrs.StaticDefinitions:
		var fields = fieldsValue.Fields()
		return ForSelectAllFields[T](fields)
	default:
		panic(fmt.Errorf("cannot get ForSelectAllFields from %T", fields))
	}
}

// A base interface for relations.
//
// This interface should only be used for OneToOne relations with a through table,
// or for ManyToMany relations with a through table.
//
// It should contain the actual instances of the models involved in the relation,
// and the through model if applicable.
type Relation interface {

	// The target model of the relation.
	Model() attrs.Definer

	// The through model of the relation.
	Through() attrs.Definer
}

// ParentInfo holds information about a relation's parent model instance
// and the field on the parent model that holds the relation.
type ParentInfo struct {
	Object attrs.Definer
	Field  attrs.Field
}

// canPrimaryKey is an interface that can be implemented by a model field's value
type canUniqueKey interface {
	// PrimaryKey returns the primary key of the relation.
	UniqueKey() any
}

// A model field's value can adhere to this interface to indicate that the
// field's relation value can be set or retrieved.
//
// This is used for OneToOne relations without a through table,
// if a through table is specified, the field's value should be of type [ThroughRelationValue]
//
// A default implementation is provided with the [RelO2O] type.
type RelationValue interface {
	attrs.Binder
	ParentInfo() *ParentInfo
	GetValue() (obj attrs.Definer)
	SetValue(instance attrs.Definer)
}

// A model field's value can adhere to this interface to indicate that the
// field's relation values can be set or retrieved.
//
// This is used for OneToMany relations without a through table,
// if a through table is specified, the field's value should be of type [MultiThroughRelationValue]
type MultiRelationValue interface {
	attrs.Binder
	ParentInfo() *ParentInfo
	GetValues() []attrs.Definer
	SetValues(instances []attrs.Definer)
}

// A model field's value can adhere to this interface to indicate that the
// field's relation value can be set or retrieved.
//
// This is used for OneToOne relations with a through table,
// if no through table is specified, the field's value should be of type [attrs.Definer]
//
// A default implementation is provided with the [RelO2O] type.
type ThroughRelationValue interface {
	attrs.Binder
	ParentInfo() *ParentInfo
	GetValue() (obj attrs.Definer, through attrs.Definer)
	SetValue(instance attrs.Definer, through attrs.Definer)
}

// A model field's value can adhere to this interface to indicate that the
// field's relation values can be set or retrieved.
//
// This is used for ManyToMany relations with a through table,
// a through table is required for ManyToMany relations.
//
// A default implementation is provided with the [RelM2M] type.
type MultiThroughRelationValue interface {
	attrs.Binder
	ParentInfo() *ParentInfo
	SetValues(instances []Relation)
	GetValues() []Relation
}

// Annotations from the database are stored in the `Row` struct, and if the
// model has a `ModelDataStore()` method that implements this interface,
// annotated values will be stored there too.
//
// Relations are also stored in the model's data store.
type ModelDataStore interface {
	HasValue(key string) bool
	GetValue(key string) (any, bool)
	SetValue(key string, value any) error
	DeleteValue(key string) error
}

// A model can adhere to this interface to indicate that the queries package
// should use the model to store and retrieve annotated values.
//
// Relations will also be stored here.
//
// Annotations will also be stored using the [Annotator] interface.
type DataModel interface {
	DataStore() ModelDataStore
}

// A model should adhere to this interface to indicate that it can store
// and retrieve annotated values from the database.
type Annotator interface {
	Annotate(annotations map[string]any)
}

// A model can adhere to this interface to indicate that it can receive
// a through model for a relation.
//
// This is used for OneToOne relations with a through table, or for ManyToMany
// relations with a through table.
//
// The through model will be set on the target end of the relation.
type ThroughModelSetter interface {
	SetThroughModel(throughModel attrs.Definer)
}

// A model can adhere to this interface to indicate that the queries package
// should not automatically save or delete the model to/from the database when
// `django/models.SaveObject()` or `django/models.DeleteObject()` is called.
type ForUseInQueries interface {
	attrs.Definer
	ForUseInQueries() bool
}

// A model can adhere to this interface to indicate fields which are
// unique together.
type UniqueTogetherDefiner interface {
	attrs.Definer
	UniqueTogether() [][]string
}

// A model can adhere to this interface to indicate that the queries package
// should use the queryset returned by `GetQuerySet()` to execute the query.
//
// Calling `queries.Objects()` with a model that implements this interface will
// return the queryset returned by `GetQuerySet()`.
type QuerySetDefiner interface {
	attrs.Definer

	GetQuerySet() *QuerySet[attrs.Definer]
}

// QuerySetChanger is an interface that can be implemented by models to indicate
// that the queryset should be changed when the model is used in a queryset.
type QuerySetChanger interface {
	attrs.Definer

	// ChangeQuerySet is called when the model is used in a queryset.
	// It should return a new queryset that will be used to execute the query.
	ChangeQuerySet(qs *QuerySet[attrs.Definer]) *QuerySet[attrs.Definer]
}

// A model can adhere to this interface to indicate that the queries package
// should use the database returned by `QuerySetDatabase()` to execute the query.
//
// The database should be retrieved from the django.Global.Settings object using the returned key.
type QuerySetDatabaseDefiner interface {
	attrs.Definer

	QuerySetDatabase() string
}

// OrderByDefiner is an interface that can be implemented by models to indicate
// that the model has a default ordering that should be used when executing queries.
type OrderByDefiner interface {
	attrs.Definer
	OrderBy() []string
}

// DatabaseSpecificTransaction is an interface for transactions that are specific to a database.
type DatabaseSpecificTransaction interface {
	drivers.Transaction
	DatabaseName() string
}

// A QueryInfo interface is used to retrieve information about a query.
//
// It is possible to introspect the queries' SQL, arguments, model, and compiler.
type QueryInfo interface {
	SQL() string
	Args() []any
	Model() attrs.Definer
	Compiler() QueryCompiler
}

// A CompiledQuery interface is used to execute a query.
//
// It is possible to execute the query and retrieve the results.
//
// The compiler will generally return a CompiledQuery interface,
// which the queryset will then store to be used as result on `LatestQuery()`.
type CompiledQuery[T1 any] interface {
	QueryInfo
	Exec() (T1, error)
}

// A compiledQuery which returns the number of rows affected by the query.
type CompiledCountQuery CompiledQuery[int64]

// A compiledQuery which returns a boolean indicating if any rows were affected by the query.
type CompiledExistsQuery CompiledQuery[bool]

// A compiledQuery which returns a list of values from the query.
type CompiledValuesListQuery CompiledQuery[[][]any]

type UpdateInfo struct {
	FieldInfo[attrs.Field]
	Where  []expr.ClauseExpression
	Joins  []JoinDef
	Values []any
}

// A QueryCompiler interface is used to compile a query.
//
// It should be able to generate SQL queries and execute them.
//
// It does not need to know about the model nor its field types.
type QueryCompiler interface {
	// DatabaseName returns the name of the database connection used by the query compiler.
	//
	// This is the name of the database connection as defined in the django.Global.Settings object.
	DatabaseName() string

	// DB returns the database connection used by the query compiler.
	//
	// If a transaction was started, it will return the transaction instead of the database connection.
	DB() drivers.DB

	// ExpressionInfo returns a usable [expr.ExpressionInfo] for the compiler.
	//
	// This is used to parse raw queries inside of [QuerySet.Rows], [QuerySet.Row] and [QuerySet.Exec].
	//
	// Allowing for the use of GO field names in a raw SQL query.
	ExpressionInfo(qs *QuerySet[attrs.Definer], internals *QuerySetInternals) *expr.ExpressionInfo

	// Quote returns the quotes used by the database.
	//
	// This is used to quote table and field names.
	// For example, MySQL uses backticks (`) and PostgreSQL uses double quotes (").
	Quote() (front string, back string)

	// Placeholder returns the placeholder used by the database for query parameters.
	// This is used to format query parameters in the SQL query.
	// For example, MySQL uses `?` and PostgreSQL uses `$1`, `$2` (but can support `?` as well).
	Placeholder() string

	// FormatColumn formats the given field column to be used in a query.
	// It should return the column name with the quotes applied.
	// Expressions should use this method to format the column name.
	FormatColumn(tableColumn *expr.TableColumn) (string, []any)

	// SupportsReturning returns the type of returning supported by the database.
	// It can be one of the following:
	//
	// - SupportsReturningNone: no returning supported
	// - SupportsReturningLastInsertId: last insert id supported
	// - SupportsReturningColumns: returning columns supported
	SupportsReturning() drivers.SupportsReturningType

	// StartTransaction starts a new transaction.
	StartTransaction(ctx context.Context) (drivers.Transaction, error)

	// WithTransaction wraps the transaction and binds it to the compiler.
	WithTransaction(tx drivers.Transaction) (drivers.Transaction, error)

	// Transaction returns the current transaction if one is active.
	Transaction() drivers.Transaction

	// InTransaction returns true if the current query compiler is in a transaction.
	InTransaction() bool

	// PrepForLikeQuery prepares a value for a LIKE query.
	//
	// It should return the value as a string, with values like `%` and `_` escaped
	// according to the database's LIKE syntax.
	PrepForLikeQuery(v any) string

	// BuildSelectQuery builds a select query with the given parameters.
	BuildSelectQuery(
		ctx context.Context,
		qs *QuerySet[attrs.Definer],
		internals *QuerySetInternals,
	) CompiledQuery[[][]interface{}]

	// BuildCountQuery builds a count query with the given parameters.
	BuildCountQuery(
		ctx context.Context,
		qs *QuerySet[attrs.Definer],
		internals *QuerySetInternals,
	) CompiledQuery[int64]

	// BuildCreateQuery builds a create query with the given parameters.
	BuildCreateQuery(
		ctx context.Context,
		qs *QuerySet[attrs.Definer],
		internals *QuerySetInternals,
		objects []UpdateInfo,
	) CompiledQuery[[][]interface{}]

	BuildUpdateQuery(
		ctx context.Context,
		qs *GenericQuerySet,
		internals *QuerySetInternals,
		objects []UpdateInfo,
	) CompiledQuery[int64]

	// BuildUpdateQuery builds an update query with the given parameters.
	BuildDeleteQuery(
		ctx context.Context,
		qs *QuerySet[attrs.Definer],
		internals *QuerySetInternals,
	) CompiledQuery[int64]
}

// RebindCompiler is an interface that can be implemented by compilers to indicate
// that the compiler can rebind queries to a different database.
//
// In simple terms, it allows the compiler to change the placeholder syntax
// for query parameters when the query is executed on a different database.
//
// If the compiler does not implement this interface, it will not be able to rebind queries,
// rebind functionality will not be available when calling [QuerySet.Rows], [QuerySet.Row] and [QuerySet.Exec].
type RebindCompiler interface {
	QueryCompiler
	Rebind(s string) string
}

type ModelMeta interface {
	Model() attrs.Definer
	TableName() string
	PrimaryKey() attrs.FieldDefinition
	OrderBy() []string
}

// A queryset is a collection of queries that can be executed against a database.
//
// It is used to retrieve, create, update, and delete objects from the database.
// It is also used to filter, order, and annotate the objects in the database.
type BaseQuerySet[T attrs.Definer, QS any] interface {
	expr.ExpressionBuilder

	Clone() QS
	Distinct() QS
	Select(fields ...any) QS
	Filter(key interface{}, vals ...interface{}) QS
	GroupBy(fields ...any) QS
	Limit(n int) QS
	Offset(n int) QS
	OrderBy(fields ...string) QS
	Reverse() QS
	ExplicitSave() QS
	Annotate(aliasOrAliasMap interface{}, exprs ...expr.Expression) QS

	// Read operations
	All() (Rows[T], error)
	Exists() (bool, error)
	Count() (int64, error)
	First() (*Row[T], error)
	Last() (*Row[T], error)
	Get() (*Row[T], error)
	Values(fields ...any) ([]map[string]any, error)
	ValuesList(fields ...any) ([][]interface{}, error)
	Aggregate(annotations map[string]expr.Expression) (map[string]any, error)

	// Write, update, and delete operations
	Create(value T) (T, error)
	Update(value T, expressions ...any) (int64, error)
	GetOrCreate(value T) (T, bool, error)
	BatchCreate(objects []T) ([]T, error)
	BatchUpdate(objects []T, exprs ...any) (int64, error)
	BulkCreate(objects []T) ([]T, error)
	BulkUpdate(objects []T, expressions ...any) (int64, error)
	Delete(objects ...T) (int64, error)

	// Raw SQL operations
	Row(sqlStr string, args ...interface{}) drivers.SQLRow
	Rows(sqlStr string, args ...interface{}) (drivers.SQLRows, error)
	Exec(sqlStr string, args ...interface{}) (sql.Result, error)

	// Transactions
	GetOrCreateTransaction() (tx drivers.Transaction, err error)
	StartTransaction(ctx context.Context) (drivers.Transaction, error)
	WithTransaction(tx drivers.Transaction) (drivers.Transaction, error)

	// Generic queryset methods
	ForUpdate() QS
	Prefix(prefix string) QS
	DB() drivers.DB
	Meta() ModelMeta
	Compiler() QueryCompiler
	Having(key interface{}, vals ...interface{}) QS
	LatestQuery() QueryInfo
	Context() context.Context
	WithContext(ctx context.Context) QS

	// Lazy methods for retrieving queries
	QueryAll(fields ...any) CompiledQuery[[][]interface{}]
	QueryAggregate() CompiledQuery[[][]interface{}]
	QueryCount() CompiledQuery[int64]
}

var compilerRegistry = make(map[reflect.Type]func(defaultDB string) QueryCompiler)

// RegisterCompiler registers a compiler for a given driver.
//
// It should be used in the init() function of the package that implements the compiler.
//
// The compiler function should take a model and a default database name as arguments,
// and return a QueryCompiler.
//
// The default database name is used to determine the database connection to use and
// retrieve from the django.Global.Settings object.
func RegisterCompiler(driver driver.Driver, compiler func(defaultDB string) QueryCompiler) {
	var driverType = reflect.TypeOf(driver)
	if driverType == nil {
		panic("driver is nil")
	}

	compilerRegistry[driverType] = compiler
}

// Compiler returns a QueryCompiler for the given model and default database name.
//
// If the default database name is empty, it will use the APPVAR_DATABASE setting.
//
// If the database is not found in the settings, it will panic.
func Compiler(defaultDB string) QueryCompiler {
	if defaultDB == "" {
		defaultDB = django.APPVAR_DATABASE
	}

	if django.Global == nil || django.Global.Settings == nil {
		panic("django.Global or django.Global.Settings is nil")
	}

	var db = django.ConfigGet[interface{ Driver() driver.Driver }](
		django.Global.Settings,
		defaultDB,
	)
	if db == nil {
		panic(fmt.Errorf(
			"no database connection found for %q",
			defaultDB,
		))
	}

	var driverType = reflect.TypeOf(db.Driver())
	if driverType == nil {
		panic("driver is nil")
	}

	var compiler, ok = compilerRegistry[driverType]
	if !ok {
		panic(fmt.Errorf("no compiler registered for driver %T", db.Driver()))
	}

	return compiler(defaultDB)
}
