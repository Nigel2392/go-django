package queries

import (
	"context"
	"database/sql"
	"fmt"
	"iter"
	"maps"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"github.com/Nigel2392/go-django/queries/internal"
	"github.com/Nigel2392/go-django/queries/src/alias"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/queries/src/expr"
	"github.com/Nigel2392/go-django/queries/src/migrator"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/Nigel2392/go-django/src/models"
	"github.com/elliotchance/orderedmap/v2"

	_ "unsafe"
)

//QUERYSET EEN INTERFACE MAKEN
//Objects() RETURNS NIET INTERFACE
//GetQuerySet() RETURNS WEL INTERFACE

const (
	// Maximum number of results to return when using the `Get` method.
	//
	// Also the maximum number of results to return when querying the database
	// inside of the `String` method.
	MAX_GET_RESULTS = 21

	// Maximum default number of results to return when using the `All` method.
	//
	// This can be overridden by the `Limit` method.
	MAX_DEFAULT_RESULTS = 1000
)

// QUERYSET_USE_CACHE_DEFAULT is the default value for the useCache field in the QuerySet.
//
// It is used to determine whether the QuerySet should cache the results of the
// latest query until the QuerySet is modified.
var QUERYSET_USE_CACHE_DEFAULT = true

// CREATE_IMPLICIT_TRANSACTION is a constant that determines whether
// the QuerySet should create a transaction implicitly each time a query is executed.
//
// If false, the queryset will not automatically start a transaction
// on each write / update / delete operation to the database.
var QUERYSET_CREATE_IMPLICIT_TRANSACTION = true

// Basic information about the model used in the QuerySet.
// It contains the model's meta information, primary key field, all fields,
// and the table name.
type modelInfo struct {
	Primary  attrs.FieldDefinition
	Object   attrs.Definer
	Fields   []attrs.FieldDefinition
	Table    string
	Ordering []string
}

func (m modelInfo) Model() attrs.Definer {
	return m.Object
}

func (m modelInfo) TableName() string {
	return m.Table
}

func (m modelInfo) PrimaryKey() attrs.FieldDefinition {
	return m.Primary
}

func (m modelInfo) OrderBy() []string {
	return m.Ordering
}

type PreloadResults struct {
	rowsRaw Rows[attrs.Definer]

	// RowsMap is a map of primary key to a slice of rows.
	//
	// The primary key is the value of the primary key field
	// of the row itself, not the value of the field
	// that was used to preload the relation.
	rowsMap map[any][]*Row[attrs.Definer]
}

type Preload struct {
	FieldName  string
	Path       string
	ParentPath string
	Chain      []string
	Rel        attrs.Relation
	Primary    attrs.FieldDefinition
	Model      attrs.Definer
	QuerySet   *QuerySet[attrs.Definer]
	Field      attrs.FieldDefinition
	Results    *PreloadResults
}

type QuerySetPreloads struct {
	Preloads []*Preload
	mapping  map[string]*Preload
}

func (p *QuerySetPreloads) Copy() *QuerySetPreloads {
	if p == nil {
		return nil
	}
	return &QuerySetPreloads{
		Preloads: slices.Clone(p.Preloads),
		mapping:  maps.Clone(p.mapping),
	}
}

// Internals contains the internal state of the QuerySet.
//
// It includes all nescessary information for
// the compiler to build a query out of.
type QuerySetInternals struct {
	Model       modelInfo
	Annotations *orderedmap.OrderedMap[string, attrs.Field]
	Fields      []*FieldInfo[attrs.FieldDefinition]
	Preload     *QuerySetPreloads
	Where       []expr.ClauseExpression
	Having      []expr.ClauseExpression
	Joins       []JoinDef
	GroupBy     []FieldInfo[attrs.FieldDefinition]
	OrderBy     []OrderBy
	Limit       int
	Offset      int
	ForUpdate   bool
	Distinct    bool

	fieldsMap map[string]*FieldInfo[attrs.FieldDefinition]
	joinsMap  map[string]struct{}
	proxyMap  map[string]struct{}

	// a pointer to the annotations field info
	// to avoid having to create a new one every time
	// an annotation is added
	//
	// this is not cloned to prevent
	// a clone from changing the annotations
	annotations *FieldInfo[attrs.FieldDefinition]
}

func (i *QuerySetInternals) AddJoin(join JoinDef) {
	if i.joinsMap == nil {
		i.joinsMap = make(map[string]struct{})
	}

	var key = join.JoinDefCondition.String()
	if _, exists := i.joinsMap[key]; !exists {
		i.joinsMap[key] = struct{}{}
		i.Joins = append(i.Joins, join)
	}
}

func (i *QuerySetInternals) AddField(field *FieldInfo[attrs.FieldDefinition]) {
	if i.fieldsMap == nil {
		i.fieldsMap = make(map[string]*FieldInfo[attrs.FieldDefinition])
	}

	var key string
	if len(field.Fields) == 1 {
		var fld = field.Fields[0]
		var fieldName = fld.Name()
		if aliasField, ok := fld.(AliasField); ok {
			var alias = aliasField.Alias()
			if alias != "" {
				fieldName = alias
			}
		}

		if len(field.Chain) > 0 {
			key = fmt.Sprintf("%s.%s", strings.Join(field.Chain, "."), fieldName)
		} else {
			key = fieldName
		}
	} else {
		key = strings.Join(field.Chain, ".")
	}

	if info, exists := i.fieldsMap[key]; !exists {
		i.fieldsMap[key] = field
		i.Fields = append(i.Fields, field)
	} else {
		logger.Warnf(
			"QuerySetInternals.AddField: field %q already exists in the queryset, skipping: %+v",
			key, info,
		)
	}
}

// ObjectsFunc is a function type that takes a model of type T and returns a QuerySet for that model.
//
// It is used to create a new QuerySet for a model which is automatically initialized with a transaction.
//
// See [RunInTransaction] for more details.
type ObjectsFunc[T attrs.Definer] func(model T) *QuerySet[T]

// QuerySet is a struct that represents a query set in the database.
//
// It contains methods to filter, order, and limit the results of a query.
//
// It is used to build and execute queries against the database.
//
// Every method on the queryset returns a new queryset, so that the original queryset is not modified.
//
// It also has a chainable api, so that you can easily build complex queries by chaining methods together.
//
// Queries are built internally with the help of the QueryCompiler interface, which is responsible for generating the SQL queries for the database.
type GenericQuerySet = QuerySet[attrs.Definer]

// QuerySet is a struct that represents a query set in the database.
//
// It contains methods to filter, order, and limit the results of a query.
//
// It is used to build and execute queries against the database.
//
// Every method on the queryset returns a new queryset, so that the original queryset is not modified.
//
// It also has a chainable api, so that you can easily build complex queries by chaining methods together.
//
// Queries are built internally with the help of the QueryCompiler interface, which is responsible for generating the SQL queries for the database.
type QuerySet[T attrs.Definer] struct {
	context      context.Context
	internals    *QuerySetInternals
	compiler     QueryCompiler
	AliasGen     *alias.Generator
	forEachRow   func(qs *QuerySet[T], row *Row[T]) error
	explicitSave bool
	latestQuery  QueryInfo
	useCache     bool
	cached       any
}

// GetQuerySet creates a new QuerySet for the given model.
//
// If the model implements the QuerySetDefiner interface,
// it will use the GetQuerySet method to get the initial QuerySet.
//
// A model should use Objects[T](model) to get the default QuerySet inside of it's
// GetQuerySet method. If not, it will recursively call itself.
//
// See [Objects] for more details.
func GetQuerySet[T attrs.Definer](model T) *QuerySet[T] {
	if m, ok := any(model).(QuerySetDefiner); ok {
		_ = m.FieldDefs() // ensure the model is initialized
		var qs = m.GetQuerySet()
		qs = qs.clone()
		return ChangeObjectsType[attrs.Definer, T](qs)
	}

	return Objects(model)
}

// GetQuerySetWithContext creates a new QuerySet for the given model
// with the given context bound to it.
//
// For more information, see [GetQuerySet].
func GetQuerySetWithContext[T attrs.Definer](ctx context.Context, model T) *QuerySet[T] {
	if ctx == nil {
		panic("GetQuerySetWithContext: context cannot be nil")
	}
	return GetQuerySet(model).WithContext(ctx)
}

// Objects creates a new QuerySet for the given model.
//
// This function should only be called in a model's GetQuerySet method.
//
// In other places the [GetQuerySet] function should be used instead.
//
// It panics if:
// - the model is nil
// - the base query info cannot be retrieved
//
// It returns a pointer to a new QuerySet.
//
// The model must implement the Definer interface.
func Objects[T attrs.Definer](model T, database ...string) *QuerySet[T] {
	model = attrs.NewObject[T](model)
	var modelV = reflect.ValueOf(model)

	if !modelV.IsValid() {
		panic("QuerySet: model is not a valid value")
	}

	if modelV.IsNil() {
		panic("QuerySet: model is nil")
	}

	if len(database) > 1 {
		panic("QuerySet: too many databases provided")
	}

	var (
		defaultDb   = getDatabaseName(model, database...)
		meta        = attrs.GetModelMeta(model)
		definitions = meta.Definitions()
		primary     = definitions.Primary()
		tableName   = definitions.TableName()
	)

	if tableName == "" {
		panic(errors.NoTableName.WithCause(fmt.Errorf(
			"model %T has no table name", model,
		)))
	}

	var orderBy []string
	if ord, ok := any(model).(OrderByDefiner); ok {
		orderBy = ord.OrderBy()
	}

	var qs = &QuerySet[T]{
		AliasGen: alias.NewGenerator(),
		context:  context.Background(),
		internals: &QuerySetInternals{
			Model: modelInfo{
				Primary:  primary,
				Object:   model,
				Fields:   definitions.Fields(),
				Table:    tableName,
				Ordering: orderBy,
			},
			Annotations: orderedmap.NewOrderedMap[string, attrs.Field](),
			Where:       make([]expr.ClauseExpression, 0),
			Having:      make([]expr.ClauseExpression, 0),
			Joins:       make([]JoinDef, 0),
			GroupBy:     make([]FieldInfo[attrs.FieldDefinition], 0),
			OrderBy:     make([]OrderBy, 0),
			Limit:       MAX_DEFAULT_RESULTS,
			Offset:      0,
		},

		// enable queryset caching by default
		// this can result in race conditions in some rare edge cases
		// but is generally safe to use
		useCache: QUERYSET_USE_CACHE_DEFAULT,
	}

	qs.compiler = Compiler(defaultDb)

	// Add default ordering to the QuerySet if the model implements OrderByDefiner
	if len(orderBy) > 0 {
		qs.internals.OrderBy = qs.compileOrderBy(orderBy...)
	}

	// Allow the model to change the QuerySet
	if c, ok := any(model).(QuerySetChanger); ok {
		var changed = c.ChangeQuerySet(
			ChangeObjectsType[T, attrs.Definer](qs),
		)
		qs = ChangeObjectsType[attrs.Definer, T](changed)
	}

	return qs
}

// Change the type of the objects in the QuerySet.
//
// Mostly used to change the type of the QuerySet
// from the generic QuerySet[attrs.Definer] to a concrete non-interface type
//
// Some things to note:
// - This does not clone the QuerySet
// - If the type mismatches and is not assignable, it will panic.
func ChangeObjectsType[OldT, NewT attrs.Definer](qs *QuerySet[OldT]) *QuerySet[NewT] {
	if _, ok := qs.internals.Model.Object.(NewT); !ok {
		panic(fmt.Errorf(
			"ChangeObjectsType: cannot change QuerySet type from %T to %T: %w",
			qs.internals.Model.Object, new(NewT),
			errors.TypeMismatch,
		))
	}

	return &QuerySet[NewT]{
		AliasGen:     qs.AliasGen,
		compiler:     qs.compiler,
		explicitSave: qs.explicitSave,
		useCache:     qs.useCache,
		cached:       qs.cached,
		internals:    qs.internals,
		context:      qs.context,
		forEachRow: func(_ *QuerySet[NewT], row *Row[NewT]) error {
			if qs.forEachRow != nil {
				return qs.forEachRow(qs, &Row[OldT]{
					Object:          any(row.Object).(OldT),
					ObjectFieldDefs: row.ObjectFieldDefs,
					Through:         row.Through,
					Annotations:     row.Annotations,
				})
			}
			return nil
		},
	}
}

// Return the underlying database which the compiler is using.
func (qs *QuerySet[T]) DB() drivers.DB {
	return qs.compiler.DB()
}

// Meta returns the model meta information for the QuerySet.
func (qs *QuerySet[T]) Meta() ModelMeta {
	// return a new model meta object with the model information
	return &modelInfo{
		Primary:  qs.internals.Model.Primary,
		Object:   qs.internals.Model.Object,
		Fields:   qs.internals.Model.Fields,
		Table:    qs.internals.Model.Table,
		Ordering: qs.internals.Model.Ordering,
	}
}

// Return the compiler which the queryset is using.
func (qs *QuerySet[T]) Compiler() QueryCompiler {
	return qs.compiler
}

// LatestQuery returns the latest query that was executed on the queryset.
func (qs *QuerySet[T]) LatestQuery() QueryInfo {
	return qs.latestQuery
}

// Context returns the context of the QuerySet.
//
// It is used to pass a context to the QuerySet, which is mainly used
// for transaction management.
func (qs *QuerySet[T]) Context() context.Context {
	if qs.context == nil {
		qs.context = context.Background()
	}
	return qs.context
}

// WithContext sets the context for the QuerySet.
//
// If a transaction is present in the context for the current database,
// it will be used for the QuerySet.
//
// It panics if the context is nil.
// This is used to pass a context to the QuerySet, which is mainly used
// for transaction management.
func (qs *QuerySet[T]) WithContext(ctx context.Context) *QuerySet[T] {
	if ctx == nil {
		panic("QuerySet: context cannot be nil")
	}

	var tx, dbName, ok = transactionFromContext(ctx)
	if ok && dbName == qs.compiler.DatabaseName() && !qs.compiler.InTransaction() {
		// if the context already has a transaction, use it
		_, err := qs.WithTransaction(tx)
		if err != nil {
			panic(errors.Wrap(err, "WithContext: failed to bind transaction to QuerySet"))
		}
	}

	qs.context = ctx
	return qs
}

// StartTransaction starts a transaction on the underlying database.
//
// It returns a transaction object which can be used to commit or rollback the transaction.
func (qs *QuerySet[T]) StartTransaction(ctx context.Context) (drivers.Transaction, error) {
	var tx, err = qs.compiler.StartTransaction(ctx)
	if err != nil {
		return nil, errors.FailedStartTransaction.WithCause(err).Wrapf(
			"failed to start transaction for QuerySet %T", qs.internals.Model.Object,
		)
	}

	// bind the transaction to the queryset context
	ctx = transactionToContext(ctx, tx, qs.compiler.DatabaseName())
	qs.context = ctx
	return tx, nil
}

// WithTransaction wraps the transaction and binds it to the QuerySet compiler.
func (qs *QuerySet[T]) WithTransaction(tx drivers.Transaction) (drivers.Transaction, error) {
	var err error
	tx, err = qs.compiler.WithTransaction(tx)
	if err != nil {
		return nil, errors.Wrap(err, "WithTransaction: failed to bind transaction to QuerySet")
	}
	// bind the transaction to the queryset context
	ctx := transactionToContext(qs.context, tx, qs.compiler.DatabaseName())
	qs.context = ctx
	return tx, nil
}

// GetOrCreateTransaction returns the current transaction if one exists,
// or starts a new transaction if the QuerySet is not already in a transaction and QUERYSET_CREATE_IMPLICIT_TRANSACTION is true.
func (qs *QuerySet[T]) GetOrCreateTransaction() (tx drivers.Transaction, err error) {
	// Check if we need to start a transaction
	var inTransaction = qs.compiler.InTransaction()
	if !inTransaction && QUERYSET_CREATE_IMPLICIT_TRANSACTION {
		return qs.StartTransaction(qs.context)
	}

	// It doesn't really matter if we are in a transaction or not,
	// the DB interface could be a transaction under the hood
	// but we don't do anything with it - it was started out of our control.
	tx = &nullTransaction{qs.compiler.DB()}

	// if we are in a transaction, we need to bind it to the QuerySet context
	if inTransaction {
		qs.context = transactionToContext(
			qs.context, tx, qs.compiler.DatabaseName(),
		)
	}

	return tx, nil
}

func (qs *QuerySet[T]) clone() *QuerySet[T] {
	return &QuerySet[T]{
		AliasGen: qs.AliasGen.Clone(),
		internals: &QuerySetInternals{
			Model:       qs.internals.Model,
			Preload:     qs.internals.Preload.Copy(),
			Annotations: qs.internals.Annotations.Copy(),
			Fields:      slices.Clone(qs.internals.Fields),
			Where:       slices.Clone(qs.internals.Where),
			Having:      slices.Clone(qs.internals.Having),
			Joins:       slices.Clone(qs.internals.Joins),
			GroupBy:     slices.Clone(qs.internals.GroupBy),
			OrderBy:     slices.Clone(qs.internals.OrderBy),
			Limit:       qs.internals.Limit,
			Offset:      qs.internals.Offset,
			ForUpdate:   qs.internals.ForUpdate,
			Distinct:    qs.internals.Distinct,

			fieldsMap: maps.Clone(qs.internals.fieldsMap),
			joinsMap:  maps.Clone(qs.internals.joinsMap),
			proxyMap:  maps.Clone(qs.internals.proxyMap),

			// annotations are not cloned
			// this is to prevent the previous annotations
			// from being modified by the cloned QuerySet
		},
		forEachRow:   qs.forEachRow,
		explicitSave: qs.explicitSave,
		useCache:     qs.useCache,
		compiler:     qs.compiler,
		context:      qs.context,

		// do not copy the cached value
		// changing the queryset should automatically
		// invalidate the cache
	}
}

// Clone creates a new QuerySet with the same parameters as the original one.
//
// It is used to create a new QuerySet with the same parameters as the original one, so that the original one is not modified.
//
// It is a shallow clone, underlying values like `*queries.Expr` are not cloned and have built- in immutability.
func (qs *QuerySet[T]) Clone() *QuerySet[T] {
	return qs.clone()
}

// Prefix sets the prefix for the alias generator
func (qs *QuerySet[T]) Prefix(prefix string) *QuerySet[T] {
	qs.AliasGen.Prefix = prefix
	return qs
}

// Return the string representation of the QuerySet.
//
// It shows a truncated list of the first 20 results, or an error if one occurred.
//
// This method WILL query the database!
func (qs *QuerySet[T]) String() string {
	var sb = strings.Builder{}
	sb.WriteString("QuerySet{")

	qs = qs.clone()
	qs.internals.Limit = MAX_GET_RESULTS

	var rows, err = qs.All()
	if err != nil {
		sb.WriteString("Error: ")
		sb.WriteString(err.Error())
		sb.WriteString("}")
		return sb.String()
	}

	if len(rows) == 0 {
		sb.WriteString("<empty>")
		sb.WriteString("}")
		return sb.String()
	}

	for i, row := range rows {
		if i == MAX_GET_RESULTS {
			sb.WriteString("... (truncated)")
			break
		}

		if i > 0 {
			sb.WriteString(", ")
		}

		sb.WriteString(
			attrs.ToString(row.Object),
		)
	}

	sb.WriteString("}")
	return sb.String()
}

// Return a detailed string representation of the QuerySet.
func (qs *QuerySet[T]) GoString() string {
	var sb = strings.Builder{}
	sb.WriteString("QuerySet{")
	sb.WriteString("\n\tmodel: ")
	sb.WriteString(fmt.Sprintf("%T", qs.internals.Model.Object))
	sb.WriteString(",\n\tfields: [")
	var written bool
	for _, field := range qs.internals.Fields {
		for _, f := range field.Fields {
			if written {
				sb.WriteString(", ")
			}

			sb.WriteString("\n\t\t")
			if len(field.Chain) > 0 {
				sb.WriteString(strings.Join(
					field.Chain, ".",
				))
				sb.WriteString(".")
			}

			sb.WriteString(f.Name())
			written = true
		}
	}
	sb.WriteString("\n\t],")

	if len(qs.internals.Where) > 0 {
		sb.WriteString("\n\twhere: [")
		for _, expr := range qs.internals.Where {
			fmt.Fprintf(&sb, "\n\t\t%T: %#v", expr, expr)
		}
		sb.WriteString("\n\t],")
	}

	if len(qs.internals.Joins) > 0 {
		sb.WriteString("\n\tjoins: [")
		for _, join := range qs.internals.Joins {
			sb.WriteString("\n\t\t")
			sb.WriteString(string(join.TypeJoin))
			sb.WriteString(" ")
			if join.Table.Alias == "" {
				sb.WriteString(join.Table.Name)
			} else {
				sb.WriteString(join.Table.Alias)
			}
			sb.WriteString(" ON ")
			var cond = join.JoinDefCondition
			for cond != nil {

				sb.WriteString(cond.String())

				cond = cond.Next

				if cond != nil {
					sb.WriteString(" AND ")
				}
			}
		}
		sb.WriteString("\n\t],")
	}

	sb.WriteString("\n}")
	return sb.String()
}

// The core function used to convert a list of fields to a list of FieldInfo.
//
// This function will make sure to map each provided field name to a model field.
//
// Relations are also respected, joins are automatically added to the query.
func (qs *QuerySet[T]) unpackFields(fields ...any) (infos []FieldInfo[attrs.FieldDefinition], hasRelated bool) {
	infos = make([]FieldInfo[attrs.FieldDefinition], 0, len(qs.internals.Fields))
	var info = FieldInfo[attrs.FieldDefinition]{
		Table: Table{
			Name: qs.internals.Model.Table,
		},
		Fields: make([]attrs.FieldDefinition, 0),
	}

	if len(fields) == 0 || len(fields) == 1 && fields[0] == "*" {
		fields = make([]any, 0, len(qs.internals.Model.Fields))
		for _, field := range qs.internals.Model.Fields {
			fields = append(fields, field.Name())
		}
	}

	for _, selectedFieldObj := range fields {

		var selectedField string
		switch v := selectedFieldObj.(type) {
		case string:
			selectedField = v
		case expr.NamedExpression:
			selectedField = v.FieldName()

			if selectedField == "" {
				panic(fmt.Errorf("Select: empty field name for %T", v))
			}
		default:
			panic(fmt.Errorf("Select: invalid field type %T, can be one of [string, NamedExpression]", v))
		}

		var current, parent, field, chain, aliases, isRelated, err = internal.WalkFields(
			qs.internals.Model.Object, selectedField, qs.AliasGen,
		)
		if err != nil {
			field, ok := qs.internals.Annotations.Get(selectedField)
			if ok {
				infos = append(infos, FieldInfo[attrs.FieldDefinition]{
					Table: Table{
						Name: qs.internals.Model.Table,
					},
					Fields: []attrs.FieldDefinition{field},
				})
				continue
			}

			panic(err)
		}

		if expr, ok := selectedFieldObj.(expr.NamedExpression); ok {
			field = &exprField{
				Field: field,
				expr:  expr,
			}
		}

		// The field might be a relation
		var rel = field.Rel()

		if (rel != nil) || (len(chain) > 0 || isRelated) {
			var relType attrs.RelationType
			if rel != nil {
				relType = rel.Type()
			} else {
				var parentMeta = attrs.GetModelMeta(parent)
				var parentDefs = parentMeta.Definitions()
				var parentField, ok = parentDefs.Field(chain[len(chain)-1])
				if !ok {
					panic(fmt.Errorf("field %q not found in %T", chain[len(chain)-1], parent))
				}
				relType = parentField.Rel().Type()
			}

			var relMeta = attrs.GetModelMeta(current)
			var relDefs = relMeta.Definitions()
			var tableName = relDefs.TableName()
			infos = append(infos, FieldInfo[attrs.FieldDefinition]{
				SourceField: field,
				Model:       current,
				RelType:     relType,
				Table: Table{
					Name:  tableName,
					Alias: aliases[len(aliases)-1],
				},
				Fields: relDefs.Fields(),
				Chain:  chain,
			})

			continue
		}

		info.Fields = append(info.Fields, field)
	}

	if len(info.Fields) > 0 {
		infos = append(infos, info)
	}

	return infos, hasRelated
}

// Select is used to select specific fields from the model.
//
// It takes a list of field names as arguments and returns a new QuerySet with the selected fields.
//
// If no fields are provided, it selects all fields from the model.
//
// If the first field is "*", it selects all fields from the model,
// extra fields (i.e. relations) can be provided thereafter - these will also be added to the selection.
//
// How to call Select:
//
// `Select("*")`
// `Select("Field1", "Field2")`
// `Select("Field1", "Field2", "Relation.*")`
// `Select("*", "Relation.*")`
// `Select("Relation.*")`
// `Select("*", "Relation.Field1", "Relation.Field2", "Relation.Nested.*")`
func (qs *QuerySet[T]) Select(fields ...any) *QuerySet[T] {
	qs = qs.clone()

	qs.internals.Fields = make([]*FieldInfo[attrs.FieldDefinition], 0)
	qs.internals.fieldsMap = make(map[string]*FieldInfo[attrs.FieldDefinition], 0)

	if qs.internals.joinsMap == nil {
		qs.internals.joinsMap = make(map[string]struct{}, len(qs.internals.Joins))
	}

	if qs.internals.proxyMap == nil {
		qs.internals.proxyMap = make(map[string]struct{}, 0)
	}

	if len(fields) == 0 {
		fields = []any{"*"}
	}

	var selectedFieldsList = make([]string, 0, len(fields))
	var namedExprs = make(map[string]expr.NamedExpression, 0)
	for _, field := range fields {
		var selectedField string
		switch v := field.(type) {
		case string:
			selectedField = v
		case expr.NamedExpression:
			selectedField = v.FieldName()

			if selectedField == "" {
				panic(fmt.Errorf("Select: empty field name for %T", v))
			}

			namedExprs[selectedField] = v
		case *FieldInfo[attrs.FieldDefinition]:
			qs.internals.AddField(v)
			continue
		default:
			panic(fmt.Errorf("Select: invalid field type %T, can be one of [string, NamedExpression]", v))
		}

		selectedFieldsList = append(
			selectedFieldsList, selectedField,
		)
	}

fieldsLoop:
	for _, selectedField := range selectedFieldsList {

		var allFields bool
		if strings.HasSuffix(selectedField, "*") {
			selectedField = strings.TrimSuffix(selectedField, "*")
			selectedField = strings.TrimSuffix(selectedField, ".")
			allFields = true
		}

		if selectedField == "" && !allFields {
			panic(fmt.Errorf("Select: empty field path, cannot select field \"\""))
		}

		if selectedField == "" && allFields {
			var currInfo = &FieldInfo[attrs.FieldDefinition]{
				Model: qs.internals.Model.Object,
				Table: Table{
					Name: qs.internals.Model.Table,
				},
				Fields: ForSelectAllFields[attrs.FieldDefinition](qs.internals.Model.Object),
			}

			// Add the current model fields to the queryset selection
			qs.internals.AddField(currInfo)

			// add proxy chain for the model
			var subInfos, subJoins = addProxyChain(
				qs, []string{}, qs.internals.proxyMap, qs.internals.joinsMap, currInfo, []string{},
			)
			for _, info := range subInfos {
				qs.internals.AddField(info)
			}
			for _, join := range subJoins {
				qs.internals.AddJoin(join)
			}

			// reselect all annotations
			// this is needed to ensure that annotations are not lost
			// on selecting all fields
			if qs.internals.Annotations.Len() > 0 {
				var info = &FieldInfo[attrs.FieldDefinition]{
					Table:  Table{Name: qs.internals.Model.Table},
					Fields: make([]attrs.FieldDefinition, 0, qs.internals.Annotations.Len()),
				}

				for head := qs.internals.Annotations.Front(); head != nil; head = head.Next() {
					info.Fields = append(info.Fields, head.Value)
				}

				qs.internals.Fields = append(qs.internals.Fields, info)
			}

			continue fieldsLoop
		}

		var flags = WalkFlagAddJoins | WalkFlagAddProxies
		if allFields {
			flags |= WalkFlagAllFields
		}

		var res, err = qs.WalkField(selectedField, namedExprs, flags)
		if err != nil {
			panic(fmt.Errorf("failed to walk field %q: %w", selectedField, err))
		}

		if res.Annotation != nil {
			qs.internals.AddField(&FieldInfo[attrs.FieldDefinition]{
				Table: Table{
					Name: qs.internals.Model.Table,
				},
				Fields: []attrs.FieldDefinition{res.Annotation},
			})
			continue fieldsLoop
		}

		for _, info := range res.Fields {
			qs.internals.AddField(info)
		}

		for _, join := range res.Joins {
			qs.internals.AddJoin(join)
		}
	}

	return qs
}

type WalkFieldResult[T attrs.Definer] struct {
	Chain      *attrs.RelationChain
	Aliases    []string
	Annotation attrs.Field
	Fields     []*FieldInfo[attrs.FieldDefinition]
	Joins      []JoinDef

	qs *QuerySet[T]
}

type WalkFlag int

const (
	WalkFlagNone      WalkFlag = 0
	WalkFlagAllFields WalkFlag = 1 << iota
	WalkFlagAddJoins
	WalkFlagSelectSubFields
	WalkFlagAddProxies
)

var _ expr.FieldResolver = (*QuerySet[attrs.Definer])(nil)

func (qs *QuerySet[T]) Alias() *alias.Generator {
	if qs.AliasGen == nil {
		qs.AliasGen = alias.NewGenerator()
	}
	return qs.AliasGen
}

func (qs *QuerySet[T]) Resolve(fieldName string, inf *expr.ExpressionInfo) (attrs.Definer, attrs.FieldDefinition, *expr.TableColumn, error) {
	var res, err = qs.WalkField(fieldName, nil, WalkFlagAddJoins)
	if err != nil {
		panic(err)
	}

	var field attrs.FieldDefinition = res.Annotation
	if field == nil {
		if res.Chain == nil || res.Chain.Final == nil {
			return nil, nil, nil, fmt.Errorf(
				"Resolve: field %q not found in model %T", fieldName, inf.Model,
			)
		}
		field = res.Chain.Final.Field
	}

	var model = qs.internals.Model.Object
	if res.Chain != nil && res.Chain.Final != nil {
		model = res.Chain.Final.Model
	}

	var col = &expr.TableColumn{}
	var args []any
	if (!inf.ForUpdate) || (len(res.Chain.Chain) > 0) {
		var aliasStr string
		switch {
		case len(res.Aliases) > 0:
			aliasStr = res.Aliases[len(res.Aliases)-1]
		case res.Chain != nil && res.Chain.Final != nil:
			var meta = attrs.GetModelMeta(res.Chain.Final.Model)
			var resDefs = meta.Definitions()
			aliasStr = resDefs.TableName()
			// aliasStr = res.Chain.Final.FieldDefs().TableName()
		default:
			aliasStr = qs.internals.Model.Table
		}

		if s, ok := field.(VirtualField); ok && !inf.SupportsWhereAlias {
			// If the field is a virtual field and the database does not support
			// WHERE expressions with aliases, we need to use the raw SQL of the
			// virtual field.
			var sql string
			sql, args = s.SQL(inf)
			col.RawSQL = sql
			col.Values = args
			goto newField
		}

		if s, ok := field.(AliasField); ok {
			// If the field is an alias field, we need to use the alias of the field.
			col.FieldAlias = qs.AliasGen.GetFieldAlias(
				aliasStr, s.Alias(),
			)
			goto newField
		}

		col.TableOrAlias = aliasStr
		col.FieldColumn = field

	newField:
		return model, field, col, nil
	}

	col.FieldColumn = field
	return model, field, col, nil
}

func (qs *QuerySet[T]) WalkField(selectedField string, namedExprs map[string]expr.NamedExpression, flag ...WalkFlag) (res *WalkFieldResult[T], err error) {

	var flags = WalkFlagNone
	for _, f := range flag {
		flags |= f
	}

	// When adding proxies, it is required to also add joins.
	if flags&WalkFlagAddProxies != 0 && flags&WalkFlagAddJoins == 0 {
		panic(fmt.Errorf(
			"WalkField: WalkFlagAddProxies requires WalkFlagAddJoins to be set if WalkFlagAddProxies is set",
		))
	}

	res = &WalkFieldResult[T]{
		qs: qs,
	}

	if annotation, ok := qs.internals.Annotations.Get(selectedField); ok {

		if namedExpr, ok := namedExprs[selectedField]; ok {
			annotation = &exprField{
				Field: annotation,
				expr:  namedExpr,
			}
		}

		res.Annotation = annotation
		return res, nil
	}

	if qs.internals.fieldsMap == nil {
		qs.internals.fieldsMap = make(map[string]*FieldInfo[attrs.FieldDefinition], 0)
	}

	if qs.internals.joinsMap == nil {
		qs.internals.joinsMap = make(map[string]struct{})
	}

	if qs.internals.proxyMap == nil {
		qs.internals.proxyMap = make(map[string]struct{}, 0)
	}

	fieldPath := strings.Split(selectedField, ".")
	relatrionChain, err := attrs.WalkRelationChain(
		qs.internals.Model.Object, flags&WalkFlagAllFields != 0, fieldPath,
	)
	if err != nil {
		return nil, err
	}

	var partIdx = 0
	var curr = relatrionChain.Root

	res.Chain = relatrionChain
	res.Aliases = make([]string, 0, len(relatrionChain.Chain))

	for curr != nil {
		var (
			fields      []*FieldInfo[attrs.FieldDefinition]
			joins       []JoinDef
			meta        = attrs.GetModelMeta(curr.Model)
			defs        = meta.Definitions()
			preloadPath = strings.Join(relatrionChain.Chain[:partIdx], ".")
		)

		var exprKey = curr.Field.Name()
		if len(preloadPath) > 0 {
			exprKey = fmt.Sprintf("%s.%s", preloadPath, exprKey)
		}

		if namedExpr, ok := namedExprs[exprKey]; ok {
			curr.Field = &exprField{
				Field: curr.Field.(attrs.Field),
				expr:  namedExpr,
			}
		}

		var alias = qs.AliasGen.GetTableAlias(
			defs.TableName(), preloadPath,
		)
		res.Aliases = append(res.Aliases, alias)

		if partIdx == 0 && curr.Next == nil {
			var fields = []attrs.FieldDefinition{curr.Field}
			if flags&WalkFlagAllFields != 0 {
				fields = ForSelectAllFields[attrs.FieldDefinition](
					defs.Fields(),
				)
			}

			var base = &FieldInfo[attrs.FieldDefinition]{
				Model: curr.Model,
				Table: Table{
					Name: defs.TableName(),
					// Alias: alias,
				},
				Chain:  []string{},
				Fields: fields,
			}

			res.Fields = append(
				res.Fields, base,
			)
		}

		if flags&WalkFlagAddJoins == 0 || partIdx == 0 { // skip the root model
			goto nextIter
		}

		// Build information for joining tables
		// This is required to still correctly handle where clauses
		// and other query modifications which might require the related fields.
		if fields, joins, err = qs.addRelationChainPart(curr.Prev, curr, res.Aliases, flags); err != nil {
			return nil, fmt.Errorf(
				"WalkField: failed to add relation chain part for %q / %q: %w",
				preloadPath, selectedField, err,
			)
		}

		if curr.Next == nil || flags&WalkFlagSelectSubFields != 0 {
			res.Fields = append(
				res.Fields,
				fields...,
			)
		}

		res.Joins = append(
			res.Joins,
			joins...,
		)

	nextIter:
		curr = curr.Next
		partIdx++
	}

	return res, nil
}

func (qs *QuerySet[T]) addRelationChainPart(prev, curr *attrs.RelationChainPart, aliasList []string, flags WalkFlag) ([]*FieldInfo[attrs.FieldDefinition], []JoinDef, error) {
	if curr == nil {
		return nil, nil, fmt.Errorf("addRelationChainPart: curr is nil")
	}

	if curr.Model == nil {
		return nil, nil, fmt.Errorf("addRelationChainPart: curr.FieldRel is nil for %q", curr.Field.Name())
	}

	var (
		relType    = prev.Field.Rel().Type()
		relThrough = prev.Field.Rel().Through()

		info  *FieldInfo[attrs.FieldDefinition]
		infos = make([]*FieldInfo[attrs.FieldDefinition], 0, 1)
		joins = make([]JoinDef, 0, 1)
		meta  = attrs.GetModelMeta(curr.Model)
		defs  = meta.Definitions()
	)

	// Build information for joining tables
	// This is required to still correctly handle where clauses
	// and other query modifications which might require the related fields.

	if qs.internals.joinsMap == nil {
		qs.internals.joinsMap = make(map[string]struct{}, len(qs.internals.Joins))
	}

	var (
		lhsAlias = qs.internals.Model.Table
		rhsAlias = defs.TableName()
	)
	if len(aliasList) == 1 {
		rhsAlias = aliasList[0]
	} else if len(aliasList) > 1 {
		lhsAlias = aliasList[len(aliasList)-2]
		rhsAlias = aliasList[len(aliasList)-1]
	}

	var includedFields = []attrs.FieldDefinition{curr.Field}
	if flags&WalkFlagAllFields != 0 {
		includedFields = ForSelectAllFields[attrs.FieldDefinition](
			defs.Fields(),
		)
	}

	var relField = prev.FieldRel.Field()
	if relField == nil {
		currMeta := attrs.GetModelMeta(curr.Model)
		currDefs := currMeta.Definitions()
		relField = currDefs.Primary()
	}

	switch {
	case (relType == attrs.RelManyToOne || relType == attrs.RelOneToMany || (relType == attrs.RelOneToOne && relThrough == nil)):
		var join JoinDef
		if clause, ok := prev.Field.(TargetClauseField); ok {
			var prevMeta = attrs.GetModelMeta(prev.Model)
			var prevDefs = prevMeta.Definitions()

			join = clause.GenerateTargetClause(
				ChangeObjectsType[T, attrs.Definer](qs),
				qs.internals,
				ClauseTarget{ // LHS
					Table: Table{
						Name:  prevDefs.TableName(),
						Alias: lhsAlias,
					},
					Model: prev.Model,
					Field: prev.Field,
				},
				ClauseTarget{ // RHS
					Table: Table{
						Name:  defs.TableName(),
						Alias: rhsAlias,
					},
					Model: curr.Model,
					Field: relField,
				},
			)
		} else {
			join = JoinDef{ // LHS -> RHS
				TypeJoin: calcJoinType(prev.FieldRel, prev.Field),
				Table: Table{
					Name:  defs.TableName(),
					Alias: rhsAlias,
				},
			}

			join.JoinDefCondition = &JoinDefCondition{
				ConditionA: expr.TableColumn{
					TableOrAlias: lhsAlias,
					FieldColumn:  prev.Field,
				},
				Operator: expr.EQ,
				ConditionB: expr.TableColumn{
					TableOrAlias: rhsAlias,
					FieldColumn:  relField,
				},
			}
		}

		info = &FieldInfo[attrs.FieldDefinition]{
			RelType:     relType,
			SourceField: prev.Field,
			Table: Table{
				Name:  defs.TableName(),
				Alias: rhsAlias,
			},
			Model:  curr.Model,
			Fields: includedFields,
			Chain:  prev.Chain(),
		}

		infos = append(infos, info)
		joins = append(joins, join)

	case (relType == attrs.RelManyToMany || relType == attrs.RelOneToOne) && (relThrough != nil):

		var through = relThrough.Model()
		var throughMeta = attrs.GetModelMeta(through)
		var throughDefs = throughMeta.Definitions()
		var throughAlias string = fmt.Sprintf("%s_through", rhsAlias)

		throughSourceField, ok := throughDefs.Field(relThrough.SourceField())
		if !ok {
			return infos, joins, fmt.Errorf(
				"Join: through source field %q not found in %T",
				relThrough.SourceField(), through) //lint:ignore ST1005 Provides information about the source
		}

		throughTargetField, ok := throughDefs.Field(relThrough.TargetField())
		if !ok {
			return infos, joins, fmt.Errorf(
				"Join: through target field %q not found in %T",
				relThrough.TargetField(), through) //lint:ignore ST1005 Provides information about the source
		}

		var join1, join2 JoinDef
		if clause, ok := prev.Field.(TargetClauseThroughField); ok {
			var prevMeta = attrs.GetModelMeta(prev.Model)
			var prevDefs = prevMeta.Definitions()
			join1, join2 = clause.GenerateTargetThroughClause(
				ChangeObjectsType[T, attrs.Definer](qs),
				qs.internals,
				ClauseTarget{ // LHS
					Table: Table{
						Name:  prevDefs.TableName(),
						Alias: lhsAlias,
					},
					Model: prev.Model,
					Field: prev.Field,
				},
				ThroughClauseTarget{ // THROUGH
					Table: Table{
						Name:  throughDefs.TableName(),
						Alias: throughAlias,
					},
					Model: through,
					Left:  throughSourceField,
					Right: throughTargetField,
				},
				ClauseTarget{ // RHS
					Table: Table{
						Name:  defs.TableName(),
						Alias: rhsAlias,
					},
					Model: curr.Model,
					Field: relField,
				},
			)
		} else {
			join1 = JoinDef{ // LHS -> THROUGH
				TypeJoin: TypeJoinLeft,
				Table: Table{
					Name:  throughDefs.TableName(),
					Alias: throughAlias,
				},
				JoinDefCondition: &JoinDefCondition{
					ConditionA: expr.TableColumn{
						TableOrAlias: lhsAlias,
						FieldColumn:  prev.Field,
					},
					Operator: expr.EQ,
					ConditionB: expr.TableColumn{
						TableOrAlias: throughAlias,
						FieldColumn:  throughSourceField,
					},
				},
			}

			join2 = JoinDef{ // THROUGH -> RHS
				TypeJoin: TypeJoinLeft,
				Table: Table{
					Name:  defs.TableName(),
					Alias: rhsAlias,
				},
				JoinDefCondition: &JoinDefCondition{
					ConditionA: expr.TableColumn{
						TableOrAlias: throughAlias,
						FieldColumn:  throughTargetField,
					},
					Operator: expr.EQ,
					ConditionB: expr.TableColumn{
						TableOrAlias: rhsAlias,
						FieldColumn:  relField,
					},
				},
			}
		}

		info = &FieldInfo[attrs.FieldDefinition]{
			RelType:     relType,
			SourceField: prev.Field,
			Table: Table{
				Name:  defs.TableName(),
				Alias: rhsAlias,
			},
			Model:  curr.Model,
			Fields: includedFields,
			Chain:  prev.Chain(),
			Through: &FieldInfo[attrs.FieldDefinition]{
				RelType:     relType,
				SourceField: prev.Field,
				Model:       through,
				Table: Table{
					Name:  throughDefs.TableName(),
					Alias: throughAlias,
				},
				Fields: ForSelectAllFields[attrs.FieldDefinition](throughDefs.Fields()),
			},
		}

		infos = append(infos, info)
		joins = append(joins, join1, join2)

	default:
		return infos, joins, fmt.Errorf("addRelationChainPart: unsupported relation type %s for field %q", relType, curr.Field.Name())
	}

	if flags&WalkFlagAddProxies != 0 {
		var subInfos, subJoins = addProxyChain(
			qs, curr.Chain(),
			qs.internals.proxyMap,
			qs.internals.joinsMap,
			info, aliasList,
		)
		infos = append(infos, subInfos...)
		joins = append(joins, subJoins...)
	}

	return infos, joins, nil
}

func addProxyChain[T attrs.Definer](qs *QuerySet[T], chain []string, proxyM, joinM map[string]struct{}, info *FieldInfo[attrs.FieldDefinition], aliases []string) ([]*FieldInfo[attrs.FieldDefinition], []JoinDef) {
	var (
		chainKey = strings.Join(chain, ".")
		infos    = make([]*FieldInfo[attrs.FieldDefinition], 0, 1)
		joins    = make([]JoinDef, 0, 1)
	)
	if _, ok := proxyM[chainKey]; !ok {
		var subInfos, subJoins = qs.addProxies(
			info, aliases,
		)

		infos = append(infos, subInfos...)
		joins = append(joins, subJoins...)
	}

	return infos, joins
}

func (qs *QuerySet[T]) addProxies(info *FieldInfo[attrs.FieldDefinition], aliasses []string) ([]*FieldInfo[attrs.FieldDefinition], []JoinDef) {
	var proxyChain = ProxyFields(info.Model)
	return qs.addSubProxies(info, proxyChain, aliasses)
}

func (qs *QuerySet[T]) addSubProxies(info *FieldInfo[attrs.FieldDefinition], node *proxyTree, aliasses []string) ([]*FieldInfo[attrs.FieldDefinition], []JoinDef) {

	var (
		sourceMeta      = attrs.GetModelMeta(info.Model)
		sourceDefs      = sourceMeta.Definitions()
		sourceTableName = sourceDefs.TableName()
		infos           = make([]*FieldInfo[attrs.FieldDefinition], 0, node.FieldsLen())
		joins           = make([]JoinDef, 0, node.FieldsLen())
	)

	for head := node.fields.Front(); head != nil; head = head.Next() {

		var chain = append(
			info.Chain,
			head.Value.Name(),
		)

		var (
			rel             = head.Value.Rel()
			targetModel     = rel.Model()
			targetDefs      = targetModel.FieldDefs()
			targetTableName = targetDefs.TableName()
			condA_Alias     = sourceTableName
			condB_Alias     = targetTableName
			subAliasses     = append( // initialize a new slice for sub aliasses
				aliasses,
				qs.AliasGen.GetTableAlias(
					targetTableName, chain,
				),
			)
		)

		if len(subAliasses) == 1 {
			condB_Alias = subAliasses[0]
		} else if len(subAliasses) > 1 {
			condA_Alias = subAliasses[len(subAliasses)-2]
			condB_Alias = subAliasses[len(subAliasses)-1]
		}

		var targetField = getTargetField(
			head.Value,
			targetDefs,
		)

		if clause, ok := head.Value.(TargetClauseField); ok {
			var lhs = ClauseTarget{
				Model: info.Model,
				Table: Table{
					Name:  sourceTableName,
					Alias: condA_Alias,
				},
				Field: head.Value,
			}
			var rhs = ClauseTarget{
				Table: Table{
					Name:  targetTableName,
					Alias: condB_Alias,
				},
				Field: targetField,
			}

			var join = clause.GenerateTargetClause(
				ChangeObjectsType[T, attrs.Definer](qs),
				qs.internals,
				lhs, rhs,
			)

			var info = &FieldInfo[attrs.FieldDefinition]{
				SourceField: head.Value,
				RelType:     rel.Type(),
				Model:       targetModel,
				Table: Table{
					Name:  targetTableName,
					Alias: condB_Alias,
				},
				Fields: ForSelectAllFields[attrs.FieldDefinition](
					targetDefs,
				),
				Chain: chain,
			}

			joins = append(joins, join)
			infos = append(infos, info)

			if proxy, ok := node.proxies.Get(head.Key); ok {
				var subInfos, subJoins = qs.addSubProxies(info, proxy.tree, subAliasses)
				infos = append(infos, subInfos...)
				joins = append(joins, subJoins...)
			}

			continue
		}

		var join = JoinDef{
			TypeJoin: calcJoinType(rel, head.Value),
			Table: Table{
				Name:  targetTableName,
				Alias: condB_Alias,
			},
			JoinDefCondition: &JoinDefCondition{
				ConditionA: expr.TableColumn{
					TableOrAlias: condA_Alias,
					FieldColumn:  head.Value,
				},
				Operator: expr.EQ,
				ConditionB: expr.TableColumn{
					TableOrAlias: condB_Alias,
					FieldColumn:  targetField,
				},
			},
		}

		var info = &FieldInfo[attrs.FieldDefinition]{
			SourceField: head.Value,
			RelType:     rel.Type(),
			Model:       targetModel,
			Table: Table{
				Name:  targetTableName,
				Alias: condB_Alias,
			},
			Fields: ForSelectAllFields[attrs.FieldDefinition](
				targetDefs,
			),
			Chain: chain,
		}

		joins = append(joins, join)
		infos = append(infos, info)
	}

	return infos, joins
}

func (qs *QuerySet[T]) Preload(fields ...any) *QuerySet[T] {
	// Preload is a no-op for QuerySet, as it is used to load related fields
	// in the initial query. The Select method already handles this.
	//
	// If you want to preload specific fields, use the Select method instead.
	qs = qs.clone()
	qs.internals.Preload = &QuerySetPreloads{
		Preloads: make([]*Preload, 0, len(fields)),
		mapping:  make(map[string]*Preload, len(fields)),
	}

	if qs.internals.joinsMap == nil {
		qs.internals.joinsMap = make(map[string]struct{})
	}

	for _, field := range fields {
		var preload Preload
		switch v := field.(type) {
		case string:
			preload = Preload{
				Path:  v,
				Chain: strings.Split(v, "."),
			}
		case Preload:
			if v.Path == "" {
				panic("Preload: empty path in Preload")
			}

			v.Chain = strings.Split(v.Path, ".")
			preload = v

		default:
			panic(fmt.Errorf("Preload: invalid field type %T, can be one of [string, Preload]", v))
		}

		var relatrionChain, err = attrs.WalkRelationChain(
			qs.internals.Model.Object, true, preload.Chain,
		)
		if err != nil {
			panic(fmt.Errorf("Preload: %w", err))
		}

		var preloads = make([]*Preload, 0, len(relatrionChain.Chain))
		var curr = relatrionChain.Root

		var partIdx = 0
		var aliasList = make([]string, 0, len(relatrionChain.Chain))
		var currModelMeta = attrs.GetModelMeta(curr.Model)
		aliasList = append(aliasList, qs.AliasGen.GetTableAlias(
			currModelMeta.Definitions().TableName(), "",
		))

		curr = curr.Next
		if curr == nil {
			panic(fmt.Errorf("Preload: no relation chain found for %q", preload.Path))
		}

		for curr != nil {

			var (
				meta = attrs.GetModelMeta(curr.Model)
				defs = meta.Definitions()
			)

			// create a preload path for each part of the relation chain

			var subChain = relatrionChain.Chain[:partIdx+1]
			var preloadPath = strings.Join(relatrionChain.Chain[:partIdx+1], ".")
			var parentPath = strings.Join(relatrionChain.Chain[:partIdx], ".")

			aliasList = append(aliasList, qs.AliasGen.GetTableAlias(
				defs.TableName(), preloadPath,
			))

			// only add new preload if the preload is not already present in the mapping
			if _, ok := qs.internals.Preload.mapping[preloadPath]; !ok {
				//	if partIdx < len(relatrionChain.Chain)-1 {
				//		panic(fmt.Errorf(
				//			"Preload: all previous preloads for path %q must be included in the QuerySet %v",
				//			preload.Path, subChain,
				//		))
				//	}

				var loadDef = &Preload{
					FieldName:  relatrionChain.Chain[partIdx],
					Path:       preloadPath,
					ParentPath: parentPath,
					Chain:      subChain,
					Rel:        curr.Prev.FieldRel,
					Model:      curr.Model,
					Field:      curr.Prev.Field,
					Primary:    defs.Primary(),
				}

				// Set the QuerySet for the preload
				if partIdx == len(relatrionChain.Chain)-1 {
					loadDef.QuerySet = preload.QuerySet
				}

				qs.internals.Preload.mapping[preloadPath] = loadDef
				preloads = append(
					preloads, loadDef,
				)

				var joins []JoinDef
				if _, joins, err = qs.addRelationChainPart(curr.Prev, curr, aliasList, WalkFlagNone); err != nil {
					panic(fmt.Errorf("Preload: %w", err))
				}

				for _, join := range joins {
					var key = join.JoinDefCondition.String()
					if _, ok := qs.internals.joinsMap[key]; !ok {
						qs.internals.Joins = append(qs.internals.Joins, join)
						qs.internals.joinsMap[key] = struct{}{}
					}
				}

			}

			curr = curr.Next
			partIdx++
		}

		qs.internals.Preload.Preloads = append(
			qs.internals.Preload.Preloads, preloads...,
		)

		slices.SortStableFunc(qs.internals.Preload.Preloads, func(a, b *Preload) int {
			return strings.Compare(a.Path, b.Path)
		})

	}

	return qs
}

// Filter is used to filter the results of a query.
//
// It takes a key and a list of values as arguments and returns a new QuerySet with the filtered results.
//
// The key can be a field name (string), an expr.Expression (expr.Expression) or a map of field names to values.
//
// By default the `__exact` (=) operator is used, each where clause is separated by `AND`.
func (qs *QuerySet[T]) Filter(key interface{}, vals ...interface{}) *QuerySet[T] {
	var nqs = qs.clone()
	nqs.internals.Where = append(qs.internals.Where, expr.Express(key, vals...)...)
	return nqs
}

// Having is used to filter the results of a query after aggregation.
//
// It takes a key and a list of values as arguments and returns a new QuerySet with the filtered results.
//
// The key can be a field name (string), an expr.Expression (expr.Expression) or a map of field names to values.
func (qs *QuerySet[T]) Having(key interface{}, vals ...interface{}) *QuerySet[T] {
	var nqs = qs.clone()
	nqs.internals.Having = append(qs.internals.Having, expr.Express(key, vals...)...)
	return nqs
}

// GroupBy is used to group the results of a query.
//
// It takes a list of field names as arguments and returns a new QuerySet with the grouped results.
func (qs *QuerySet[T]) GroupBy(fields ...any) *QuerySet[T] {
	var nqs = qs.clone()
	nqs.internals.GroupBy, _ = qs.unpackFields(fields...)
	return nqs
}

// OrderBy is used to order the results of a query.
//
// It takes a list of field names as arguments and returns a new QuerySet with the ordered results.
//
// The field names can be prefixed with a minus sign (-) to indicate descending order.
func (qs *QuerySet[T]) OrderBy(fields ...string) *QuerySet[T] {
	var nqs = qs.clone()
	nqs.internals.OrderBy = nqs.compileOrderBy(fields...)
	return nqs
}

// compileOrderBy is used to compile the order by fields into a slice of OrderBy structs.
//
// it processes the field names, checks for descending order (indicated by a leading minus sign),
// and generates the appropriate TableColumn and FieldAlias for each field.
func (qs *QuerySet[T]) compileOrderBy(fields ...string) []OrderBy {
	var orderBy = make([]OrderBy, 0, len(fields))

	if len(fields) == 0 {
		fields = qs.internals.Model.Ordering
	}

	for _, field := range fields {
		var ord = strings.TrimSpace(field)
		var desc = false
		if strings.HasPrefix(ord, "-") {
			desc = true
			ord = strings.TrimPrefix(ord, "-")
		}

		var res, err = qs.WalkField(
			ord, nil, WalkFlagAddJoins,
		)
		if err != nil {
			panic(err)
		}

		if res.Chain == nil && res.Annotation == nil {
			panic(errors.FieldNotFound.Wrapf(
				"WalkFieldResult.TableColumn: no chain or annotation present for %T, cannot create TableColumn",
				res.qs.internals.Model.Object,
			))
		}

		if (res.Chain != nil && len(res.Chain.Fields) > 1) && res.Annotation == nil {
			panic(errors.FieldNotFound.Wrapf(
				"WalkFieldResult.TableColumn: multiple fields in chain %q for %T without annotation present, cannot create TableColumn",
				res.Chain.Fields, res.qs.internals.Model.Object,
			))
		}

		for _, join := range res.Joins {
			qs.internals.AddJoin(join)
		}

		var (
			obj   attrs.Definer
			field attrs.FieldDefinition
		)

		if res.Annotation != nil {
			obj = res.qs.internals.Model.Object
			field = res.Annotation
		} else {
			obj = res.Chain.Final.Model
			field = res.Chain.Final.Field
		}

		var defs = obj.FieldDefs()
		var tableAlias string
		if len(res.Aliases) > 0 {
			tableAlias = res.Aliases[len(res.Aliases)-1]
		} else {
			tableAlias = defs.TableName()
		}

		var alias string
		if vF, ok := field.(AliasField); ok {
			alias = res.qs.AliasGen.GetFieldAlias(
				tableAlias, vF.Alias(),
			)
		}

		if alias != "" {
			tableAlias = ""
			field = nil
		}

		orderBy = append(orderBy, OrderBy{
			Column: expr.TableColumn{
				TableOrAlias: tableAlias,
				FieldColumn:  field,
				FieldAlias:   alias,
			},
			Desc: desc,
		})
	}
	return orderBy
}

// Reverse is used to reverse the order of the results of a query.
//
// It returns a new QuerySet with the reversed order.
func (qs *QuerySet[T]) Reverse() *QuerySet[T] {
	var ordBy = make([]OrderBy, 0, len(qs.internals.OrderBy))
	for _, ord := range qs.internals.OrderBy {
		ordBy = append(ordBy, OrderBy{
			Column: ord.Column,
			Desc:   !ord.Desc,
		})
	}
	var nqs = qs.clone()
	nqs.internals.OrderBy = ordBy
	return nqs
}

// Limit is used to limit the number of results returned by a query.
func (qs *QuerySet[T]) Limit(n int) *QuerySet[T] {
	if n <= 0 {
		return qs
	}
	var nqs = qs.clone()
	nqs.internals.Limit = n
	return nqs
}

// Offset is used to set the offset of the results returned by a query.
func (qs *QuerySet[T]) Offset(n int) *QuerySet[T] {
	if n <= 0 {
		return qs
	}
	var nqs = qs.clone()
	nqs.internals.Offset = n
	return nqs
}

// ForUpdate is used to lock the rows returned by a query for update.
//
// It is used to prevent other transactions from modifying the rows until the current transaction is committed or rolled back.
func (qs *QuerySet[T]) ForUpdate() *QuerySet[T] {
	var nqs = qs.clone()
	nqs.internals.ForUpdate = true
	return nqs
}

// Distinct is used to select distinct rows from the results of a query.
//
// It is used to remove duplicate rows from the results.
func (qs *QuerySet[T]) Distinct() *QuerySet[T] {
	var nqs = qs.clone()
	nqs.internals.Distinct = true
	return nqs
}

// ExplicitSave is used to indicate that the save operation should be explicit.
//
// It is used to prevent the automatic save operation from being performed on the model.
//
// I.E. when using the `Create` method after calling `qs.ExplicitSave()`, it will **not** automatically
// save the model to the database using the model's own `Save` method.
func (qs *QuerySet[T]) ExplicitSave() *QuerySet[T] {
	qs.explicitSave = true
	return qs
}

func (qs *QuerySet[T]) annotate(alias string, expr expr.Expression) {
	// If the has not been added to the annotations, we need to add it
	if qs.internals.annotations == nil {
		qs.internals.annotations = &FieldInfo[attrs.FieldDefinition]{
			Table: Table{
				Name: qs.internals.Model.Table,
			},
			Fields: make([]attrs.FieldDefinition, 0, qs.internals.Annotations.Len()),
		}
		qs.internals.Fields = append(
			qs.internals.Fields, qs.internals.annotations,
		)
	}

	// Add the field to the annotations
	var field = newQueryField[any](alias, expr)
	qs.internals.Annotations.Set(alias, field)
	qs.internals.annotations.Fields = append(
		qs.internals.annotations.Fields, field,
	)
}

// Annotate is used to add annotations to the results of a query.
//
// It takes a string or a map of strings to expr.Expressions as arguments and returns a new QuerySet with the annotations.
//
// If a string is provided, it is used as the alias for the expr.Expression.
//
// If a map is provided, the keys are used as aliases for the expr.Expressions.
func (qs *QuerySet[T]) Annotate(aliasOrAliasMap interface{}, exprs ...expr.Expression) *QuerySet[T] {
	qs = qs.clone()

	switch aliasOrAliasMap := aliasOrAliasMap.(type) {
	case string:
		if len(exprs) == 0 {
			panic("QuerySet: no expr.Expression provided")
		}
		qs.annotate(aliasOrAliasMap, exprs[0])
	case map[string]expr.Expression:
		if len(exprs) > 0 {
			panic("QuerySet: map and expr.Expressions both provided")
		}
		for alias, e := range aliasOrAliasMap {
			qs.annotate(alias, e)
		}
	case map[string]any:
		if len(exprs) > 0 {
			panic("QuerySet: map and expr.Expressions both provided")
		}
		for alias, e := range aliasOrAliasMap {
			if exprs, ok := e.(expr.Expression); ok {
				qs.annotate(alias, exprs)
			} else {
				panic(fmt.Errorf(
					"QuerySet: %q is not an expr.Expression (%T)", alias, e,
				))
			}
		}
	}

	return qs
}

// Scope is used to apply a scope to the QuerySet.
//
// It takes a function that modifies the QuerySet as an argument and returns a QuerySet with the applied scope.
//
// The queryset is modified in place, so the original QuerySet is changed.
func (qs *QuerySet[T]) Scope(scopes ...func(*QuerySet[T], *QuerySetInternals) *QuerySet[T]) *QuerySet[T] {
	var (
		newQs   = qs.clone()
		changed bool
	)
	for _, scopeFunc := range scopes {
		newQs = scopeFunc(newQs, newQs.internals)
		if newQs != nil {
			changed = true
		}
	}
	if changed {
		*qs = *newQs
	}
	return qs
}

func (qs *QuerySet[T]) BuildExpression() expr.Expression {
	qs = qs.clone()
	qs.internals.Limit = 0  // no limit for subqueries
	qs.internals.Offset = 0 // no offset for subqueries
	var subquery = &subqueryExpr[T, *QuerySet[T]]{
		qs: qs.WithContext(makeSubqueryContext(qs.Context())),
	}
	return subquery
}

func (qs *QuerySet[T]) QueryAll(fields ...any) CompiledQuery[[][]interface{}] {
	// Select all fields if no fields are provided
	//
	// Override the pointer to the original QuerySet with the Select("*") QuerySet
	if len(qs.internals.Fields) == 0 && len(fields) == 0 {
		*qs = *qs.Select("*")
	}

	if len(fields) > 0 {
		*qs = *qs.Select(fields...)
	}

	var query = qs.compiler.BuildSelectQuery(
		qs.context,
		ChangeObjectsType[T, attrs.Definer](qs),
		qs.internals,
	)
	qs.latestQuery = query

	return query
}

func (qs *QuerySet[T]) QueryAggregate() CompiledQuery[[][]interface{}] {
	var dereferenced = *qs.internals
	dereferenced.OrderBy = nil     // no order by for aggregates
	dereferenced.Limit = 0         // no limit for aggregates
	dereferenced.Offset = 0        // no offset for aggregates
	dereferenced.ForUpdate = false // no for update for aggregates
	dereferenced.Distinct = false  // no distinct for aggregates
	var query = qs.compiler.BuildSelectQuery(
		qs.context,
		ChangeObjectsType[T, attrs.Definer](qs),
		&dereferenced,
	)
	qs.latestQuery = query
	return query
}

func (qs *QuerySet[T]) QueryCount() CompiledQuery[int64] {
	var q = qs.compiler.BuildCountQuery(
		qs.context,
		ChangeObjectsType[T, attrs.Definer](qs),
		qs.internals,
	)
	qs.latestQuery = q
	return q
}

func (qs *QuerySet[T]) ForEachRow(rowFunc func(qs *QuerySet[T], row *Row[T]) error) *QuerySet[T] {
	qs = qs.clone()
	qs.forEachRow = rowFunc
	return qs
}

// IterAll returns an iterator over all rows in the QuerySet, and the amount of rows to iterate over.
//
// If [ForEachRow] is set, it will be used to process each row inside of the iterator.
func (qs *QuerySet[T]) IterAll() (int, iter.Seq2[*Row[T], error], error) {

	var resultQuery = qs.QueryAll()
	var results, err = resultQuery.Exec()
	if err != nil {
		return 0, nil, err
	}

	var runActors = func(o attrs.Definer) error {
		if o == nil {
			return nil
		}
		_, err = runActor(qs.context, actsAfterQuery, o)
		return err
	}

	rows, err := newRows[T](
		qs.internals.Fields,
		internal.NewObjectFromIface(qs.internals.Model.Object),
		runActors,
	)
	if err != nil {
		return 0, nil, errors.NoRows.WithCause(fmt.Errorf(
			"failed to create rows object for QuerySet.All: %w", err,
		))
	}

	for resultIndex, row := range results {
		var (
			obj        = internal.NewObjectFromIface(qs.internals.Model.Object)
			scannables = getScannableFields(qs.internals.Fields, obj)
		)

		var (
			annotator, _ = obj.(DataModel)
			annotations  = make(map[string]any)
			datastore    ModelDataStore
		)

		if annotator != nil {
			datastore = annotator.DataStore()
		}

		for j, field := range scannables {
			f := field.field
			val := row[j]

			if err := f.Scan(val); err != nil {
				return 0, nil, errors.ValueError.WithCause(errors.Wrapf(
					err, "failed to scan field %q (%T) in %T",
					f.Name(), f, f.Instance(),
				))
			}

			// If it's a virtual field not in the model, store as annotation
			if vf, ok := f.(AliasField); ok {
				var (
					alias = vf.Alias()
					val   = vf.GetValue()
				)
				if alias == "" {
					alias = f.Name()
				}

				// If the value is a byte slice, convert it to a string
				// It is highly unlikely that a byte slice will be used as an annotation,
				// thus we convert it to a string in case the database driver returns the wrong type.
				// This is a workaround for some drivers that return []byte instead of string.
				// This should also be done in [Aggregate], [Values] and [ValuesList].
				if bytes, ok := val.([]byte); ok {
					val = string(bytes)
				}

				annotations[alias] = val

				if datastore != nil {
					datastore.SetValue(alias, val)
				}
			}
		}

		var (
			uniqueValue any
			throughObj  attrs.Definer
		)

		// required in case the root object has a through relation bound to it
		if rows.hasRoot() {
			var rootRow = rows.rootRow(scannables)

			// if the root object has no through relation
			// we can use the unique key of the root object
			// as the unique value for the root object.
			//
			// otherwise, we should use the through object to
			// generate the unique value to not clash with other objects
			if rootRow.through == nil {
				uniqueValue, err = GetUniqueKey(rootRow.field)
				switch {
				case err != nil && errors.Is(err, errors.NoUniqueKey) && rows.hasMultiRelations:
					return 0, nil, errors.Wrapf(
						err, "failed to get unique key for %T, but has multi relations",
						rootRow.object,
					)
				case err != nil && errors.Is(err, errors.NoUniqueKey):
					// if no unique key is found, we can use the result index as a unique value
					// this is only valid for the root object, as it is not a relation
					uniqueValue = resultIndex + 1
				}
			}

			// if the root object has a through relation
			// we should store it in the rows tree for
			// binding it to the root.
			throughObj = rootRow.through

			// If the root object has a through relation,
			// we need to get the unique value from the through object
			// if the through object is not nil, we can get the unique value from it
			//
			// this logic is kept in line in [buildChainParts] to ensure
			// data consistency across the relations.
			if throughObj != nil {
				uniqueValue, err = GetUniqueKey(throughObj)
				if err != nil {
					return 0, nil, errors.Wrapf(
						err, "failed to get unique key from through object %T",
						throughObj,
					)
				}
			}

		}

		// fake unique value for the root object is OK
		if uniqueValue == nil {
			uniqueValue = resultIndex + 1
		}

		// add the root object to the rows tree
		// this has to be done before adding possible duplicate relations
		rows.addRoot(
			uniqueValue, obj, throughObj, annotations,
		)

		for _, possibleDuplicate := range rows.possibleDuplicates {
			var chainParts = buildChainParts(
				scannables[possibleDuplicate.idx],
			)
			rows.addRelationChain(chainParts)
		}
	}

	rowCount, rowIter, err := rows.compile(qs)
	if err != nil {
		return 0, nil, errors.Wrapf(
			err, "failed to compile rows for QuerySet.All",
		)
	}

	return rowCount, func(yield func(*Row[T], error) bool) {
		var next, stop = iter.Pull2(rowIter)
		for {
			var row, err, valid = next()
			if !valid {
				break
			}

			if qs.forEachRow != nil {
				if err := qs.forEachRow(qs, row); err != nil {
					//return nil, errors.Wrapf(
					//	err, "failed to execute forEachRow for QuerySet.All: %s", err,
					//)
					// stop the iteration and return the error
					yield(nil, errors.Wrapf(
						err, "failed to execute forEachRow for QuerySet.All: %s", err,
					))
					stop()
					break
				}
			}

			if !yield(row, err) {
				stop()
				break
			}
		}
	}, nil
}

// All is used to retrieve all rows from the database.
//
// It returns a Query that can be executed to get the results, which is a slice of Row objects.
//
// Each Row object contains the model object and a map of annotations.
//
// If no fields are provided, it selects all fields from the model, see `Select()` for more details.
func (qs *QuerySet[T]) All() (Rows[T], error) {
	if qs.cached != nil && qs.useCache {
		return qs.cached.([]*Row[T]), nil
	}

	var rowIdx = 0
	var rowCount, rowIter, err = qs.IterAll()
	if err != nil {
		return nil, errors.Wrapf(
			err, "failed to compile rows for QuerySet.All: %s", err,
		)
	}

	var root = make([]*Row[T], rowCount)
	for row, err := range rowIter {
		if err != nil {
			return nil, err
		}

		root[rowIdx] = row
		rowIdx++
	}

	return root, nil
}

// Values is used to retrieve a list of dictionaries from the database.
//
// It takes a list of field names as arguments and returns a list of maps.
//
// Each map contains the field names as keys and the field values as values.
// If no fields are provided, it selects all fields from the model, see [Select] for more details.
func (qs *QuerySet[T]) Values(fields ...any) ([]map[string]any, error) {
	if qs.cached != nil && qs.useCache {
		return qs.cached.([]map[string]any), nil
	}

	var resultQuery = qs.QueryAll(fields...)
	var results, err = resultQuery.Exec()
	if err != nil {
		return nil, err
	}

	var list = make([]map[string]any, len(results))
	for i, row := range results {
		var (
			obj    = internal.NewObjectFromIface(qs.internals.Model.Object)
			fields = getScannableFields(qs.internals.Fields, obj)
			values = make(map[string]any, len(row))
		)
		for j, field := range fields {
			var f = field.field
			var val = row[j]

			if err = f.Scan(val); err != nil {
				return nil, errors.ValueError.WithCause(fmt.Errorf(
					"failed to scan field %q in %T: %w",
					f.Name(), row, err,
				))
			}

			// generate dict key for the field
			var key string
			var value = f.GetValue()
			if vf, ok := f.(AliasField); ok {
				key = vf.Alias()

				// If the value is a byte slice, convert it to a string
				// It is highly unlikely that a byte slice will be used as an annotation,
				// thus we convert it to a string in case the database driver returns the wrong type.
				// This is a workaround for some drivers that return []byte instead of string.
				// This should also be done in [All], [Aggregate] and [ValuesList].
				if bytes, ok := value.([]byte); ok {
					value = string(bytes)
				}

			} else if len(field.chainKey) > 0 {
				key = strings.Join(
					append([]string{field.chainKey}, f.Name()),
					".",
				)
			} else {
				key = f.Name()
			}

			if key == "" {
				panic(fmt.Errorf(
					"QuerySet.Values: empty key for field %q in %T",
					f.Name(), row,
				))
			}

			if _, ok := values[key]; ok {
				return nil, errors.ValueError.WithCause(fmt.Errorf(
					"QuerySet.Values: duplicate key %q for field %q in %T",
					key, f.Name(), row,
				))
			}

			values[key] = value
		}

		list[i] = values
	}

	if qs.useCache {
		qs.cached = list
	}

	return list, nil
}

// ValuesList is used to retrieve a list of values from the database.
//
// It takes a list of field names as arguments and returns a ValuesListQuery.
func (qs *QuerySet[T]) ValuesList(fields ...any) ([][]interface{}, error) {
	if qs.cached != nil && qs.useCache {
		return qs.cached.([][]any), nil
	}

	var resultQuery = qs.QueryAll(fields...)
	var results, err = resultQuery.Exec()
	if err != nil {
		return nil, err
	}

	var list = make([][]any, len(results))
	for i, row := range results {
		var obj = internal.NewObjectFromIface(qs.internals.Model.Object)
		var fields = getScannableFields(qs.internals.Fields, obj)
		var values = make([]any, len(fields))
		for j, field := range fields {
			var f = field.field
			var val = row[j]

			if err = f.Scan(val); err != nil {
				return nil, errors.ValueError.WithCause(fmt.Errorf(
					"failed to scan field %q in %T: %w",
					f.Name(), row, err,
				))
			}

			var v = f.GetValue()
			// If the value is a byte slice, convert it to a string
			// It is highly unlikely that a byte slice will be used as an annotation,
			// thus we convert it to a string in case the database driver returns the wrong type.
			// This is a workaround for some drivers that return []byte instead of string.
			// This should also be done in [All], [Aggregate] and [Values].
			if _, ok := f.(AliasField); ok {
				if bytes, ok := v.([]byte); ok {
					v = string(bytes)
				}
			}

			values[j] = v
		}

		list[i] = values
	}

	if qs.useCache {
		qs.cached = list
	}

	return list, nil
}

// Aggregate is used to perform aggregation on the results of a query.
//
// It takes a map of field names to expr.Expressions as arguments and returns a Query that can be executed to get the results.
func (qs *QuerySet[T]) Aggregate(annotations map[string]expr.Expression) (map[string]any, error) {
	if qs.cached != nil && qs.useCache {
		return qs.cached.(map[string]any), nil
	}

	qs.internals.Fields = make([]*FieldInfo[attrs.FieldDefinition], 0, len(annotations))

	for alias, expr := range annotations {
		qs.annotate(alias, expr)
	}

	var query = qs.QueryAggregate()
	var results, err = query.Exec()
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return map[string]any{}, nil
	}

	var (
		row        = results[0]
		out        = make(map[string]any)
		scannables = getScannableFields(
			qs.internals.Fields,
			internal.NewObjectFromIface(qs.internals.Model.Object),
		)
	)

	for i, field := range scannables {
		if vf, ok := field.field.(AliasField); ok {
			if err := vf.Scan(row[i]); err != nil {
				return nil, errors.ValueError.WithCause(errors.Wrapf(
					err, "failed to scan field %q (%T) in %T",
					vf.Name(), vf, qs.internals.Model.Object,
				))
			}

			// If the value is a byte slice, convert it to a string
			// It is highly unlikely that a byte slice will be used as an annotation,
			// thus we convert it to a string in case the database driver returns the wrong type.
			// This is a workaround for some drivers that return []byte instead of string.
			// This should also be done in [All], [ValuesList] and [Values].
			var value = vf.GetValue()
			if bytes, ok := value.([]byte); ok {
				value = string(bytes)
			}

			out[vf.Alias()] = value
		}
	}

	if qs.useCache {
		qs.cached = out
	}

	return out, nil

}

// Get is used to retrieve a single row from the database.
//
// It returns a Query that can be executed to get the result, which is a Row object
// that contains the model object and a map of annotations.
//
// It panics if the queryset has no where clause.
//
// If no rows are found, it returns queries.errors.ErrNoRows.
//
// If multiple rows are found, it returns queries.errors.ErrMultipleRows.
func (qs *QuerySet[T]) Get() (*Row[T], error) {
	if qs.cached != nil && qs.useCache {
		return qs.cached.(*Row[T]), nil
	}

	var nillRow = &Row[T]{}

	// limit to max_get_results
	qs.internals.Limit = MAX_GET_RESULTS

	var results, err = qs.All()
	if err != nil {
		return nillRow, err
	}

	var resCnt = len(results)
	if resCnt == 0 {
		return nillRow, errors.NoRows.WithCause(fmt.Errorf(
			"no rows found for %T", qs.internals.Model.Object,
		))
	}

	if resCnt > 1 {
		var errResCnt string
		if MAX_GET_RESULTS == 0 || resCnt < MAX_GET_RESULTS {
			errResCnt = strconv.Itoa(resCnt)
		} else {
			errResCnt = strconv.Itoa(MAX_GET_RESULTS-1) + "+"
		}

		return nillRow, errors.MultipleRows.WithCause(fmt.Errorf(
			"multiple rows returned for %T: %s rows",
			qs.internals.Model.Object, errResCnt,
		))
	}

	if qs.useCache {
		qs.cached = results[0]
	}

	return results[0], nil
}

// GetOrCreate is used to retrieve a single row from the database or create it if it does not exist.
//
// It returns the definer object and an error if any occurred.
//
// This method executes a transaction to ensure that the object is created only once.
//
// It panics if the queryset has no where clause.
func (qs *QuerySet[T]) GetOrCreate(value T) (T, bool, error) {

	if len(qs.internals.Where) == 0 {
		panic(errors.NoWhereClause.WithCause(fmt.Errorf(
			"QuerySet.GetOrCreate: no where clause for %T", qs.internals.Model.Object,
		)))
	}

	// If the queryset is already in a transaction, that transaction will be used
	// automatically.
	var tx, err = qs.GetOrCreateTransaction()
	if err != nil {
		return *new(T), false, errors.FailedStartTransaction.WithCause(err)
	}
	defer tx.Rollback(qs.context)

	// Check if the object already exists
	qs.useCache = false
	row, err := qs.Get()
	if err != nil {
		if errors.Is(err, errors.NoRows) {
			goto create
		} else {
			return *new(T), false, errors.Wrapf(
				err, "failed to get object %T", qs.internals.Model.Object,
			)
		}
	}

	// Object already exists, return it and commit the transaction
	if row != nil {
		return row.Object, false, tx.Commit(qs.context)
	}

	// Object does not exist, create it
create:
	obj, err := qs.Create(value)
	// obj, err := qs.BulkCreate([]T{value})
	if err != nil {
		return *new(T), false, errors.Wrapf(
			err, "failed to create object %T", qs.internals.Model.Object,
		)
	}

	// Object was created successfully, commit the transaction
	return obj, true, tx.Commit(qs.context)
}

// First is used to retrieve the first row from the database.
//
// It returns a Query that can be executed to get the result, which is a Row object
// that contains the model object and a map of annotations.
func (qs *QuerySet[T]) First() (*Row[T], error) {
	qs.internals.Limit = 1 // limit to 1 row

	var results, err = qs.All()
	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return nil, errors.NoRows
	}

	return results[0], nil

}

// Last is used to retrieve the last row from the database.
//
// It reverses the order of the results and then calls First to get the last row.
//
// It returns a Query that can be executed to get the result, which is a Row object
// that contains the model object and a map of annotations.
func (qs *QuerySet[T]) Last() (*Row[T], error) {
	*qs = *qs.Reverse()
	return qs.First()
}

// Exists is used to check if any rows exist in the database.
//
// It returns a Query that can be executed to get the result,
// which is a boolean indicating if any rows exist.
func (qs *QuerySet[T]) Exists() (bool, error) {

	var dereferenced = *qs.internals
	dereferenced.Limit = 1  // limit to 1 row
	dereferenced.Offset = 0 // no offset for exists
	var resultQuery = qs.compiler.BuildCountQuery(
		qs.context,
		ChangeObjectsType[T, attrs.Definer](qs),
		&dereferenced,
	)
	qs.latestQuery = resultQuery

	var exists, err = resultQuery.Exec()
	if err != nil {
		return false, err
	}
	return exists > 0, nil
}

// Count is used to count the number of rows in the database.
//
// It returns a CountQuery that can be executed to get the result, which is an int64 indicating the number of rows.
func (qs *QuerySet[T]) Count() (int64, error) {
	var q = qs.QueryCount()
	var count, err = q.Exec()
	if err != nil {
		return 0, err
	}
	return count, nil
}

// Create is used to create a new object in the database.
//
// It takes a definer object as an argument and returns a Query that can be executed
// to get the result, which is the created object.
//
// It panics if a non- nullable field is null or if the field is not found in the model.
//
// The model can adhere to django's `models.Saver` interface, in which case the `Save()` method will be called
// unless `ExplicitSave()` was called on the queryset.
//
// If `ExplicitSave()` was called, the `Create()` method will return a query that can be executed to create the object
// without calling the `Save()` method on the model.
func (qs *QuerySet[T]) Create(value T) (T, error) {

	var tx, err = qs.GetOrCreateTransaction()
	if err != nil {
		return *new(T), errors.FailedStartTransaction.WithCause(err)
	}
	defer tx.Rollback(qs.context)

	// Check if the object is a saver
	// If it is, we can use the Save method to save the object
	if saver, ok := any(value).(models.ContextSaver); ok && !qs.explicitSave {
		var err error
		value, err = setup(value)
		if err != nil {
			return *new(T), errors.Wrapf(
				err, "failed to setup object %T", value,
			)
		}

		var actor = Actor(value)
		ctx, err := actor.BeforeCreate(qs.context)
		if err != nil {
			return *new(T), errors.Wrapf(
				err, "failed to run ActsBeforeCreate for %T", value,
			)
		}

		// create a fake context to retain control over the save operation
		// and actor methods
		err = saver.Save(actor.Fake(ctx, actsAfterSave))
		if err != nil {
			return *new(T), errors.SaveFailed.WithCause(errors.Wrapf(
				err, "failed to save object %T", value,
			))
		}

		if _, err = actor.AfterCreate(ctx); err != nil {
			return *new(T), errors.Wrapf(
				err, "failed to run ActsAfterCreate for %T", value,
			)
		}

		return saver.(T), tx.Commit(qs.context)
	}

	result, err := qs.BulkCreate([]T{value})
	if err != nil {
		return *new(T), err
	}

	var support = qs.compiler.SupportsReturning()
	if len(result) == 0 && support != drivers.SupportsReturningNone {
		return *new(T), errors.NoRows
	}

	return result[0], tx.Commit(qs.context)
}

// Update is used to update an object in the database.
//
// It takes a definer object as an argument and returns a CountQuery that can be executed
// to get the result, which is the number of rows affected.
//
// It panics if a non- nullable field is null or if the field is not found in the model.
//
// If the model adheres to django's `models.Saver` interface, no where clause is provided
// and ExplicitSave() was not called, the `Save()` method will be called on the model
func (qs *QuerySet[T]) Update(value T, expressions ...any) (int64, error) {
	var tx, err = qs.GetOrCreateTransaction()
	if err != nil {
		return 0, errors.FailedStartTransaction.WithCause(err)
	}
	defer tx.Rollback(qs.context)

	if len(qs.internals.Where) == 0 && !qs.explicitSave {

		if _, err := setup(value); err != nil {
			return 0, errors.Wrapf(
				err, "failed to setup object %T", value,
			)
		}

		if saver, ok := any(value).(models.ContextSaver); ok {
			var actor = Actor(value)
			ctx, err := actor.BeforeSave(qs.context)
			if err != nil {
				return 0, errors.Wrapf(
					err, "failed to run ActsBeforeUpdate for %T", value,
				)
			}

			// create a fake context to retain control over the save operation
			// and actor methods
			err = saver.Save(actor.Fake(ctx, actsAfterSave))
			if err != nil {
				return 0, errors.SaveFailed.WithCause(errors.Wrapf(
					err, "failed to save object %T", value,
				))
			}

			if _, err = actor.AfterSave(qs.context); err != nil {
				return 0, errors.Wrapf(
					err, "failed to run ActsAfterUpdate for %T", value,
				)
			}

			return 1, tx.Commit(qs.context)
		}
	}

	c, err := qs.BulkUpdate([]T{value}, expressions...)
	if err != nil {
		return 0, errors.NoChanges.WithCause(errors.Wrapf(
			err, "failed to update object %T", qs.internals.Model.Object,
		))
	}

	return c, tx.Commit(qs.context)
}

// BulkCreate is used to create multiple objects in the database.
//
// It takes a list of definer objects as arguments and returns a Query that can be executed
// to get the result, which is a slice of the created objects.
//
// It panics if a non- nullable field is null or if the field is not found in the model.
//
// This function will run the [ActsBeforeCreate] and [ActsAfterCreate] actor methods
// for each object, as well as send the [SignalPreModelSave] and [SignalPostModelSave] signals.
func (qs *QuerySet[T]) BulkCreate(objects []T) ([]T, error) {
	var tx, err = qs.GetOrCreateTransaction()
	if err != nil {
		return nil, errors.FailedStartTransaction.WithCause(err)
	}
	defer tx.Rollback(qs.context)

	var (
		infos   = make([]UpdateInfo, 0, len(objects))
		primary attrs.Field
	)

	for _, object := range objects {

		var err error
		object, err = setup(object)
		if err != nil {
			return nil, errors.Wrapf(
				err, "failed to setup object %T", object,
			)
		}

		if _, err = runActor(qs.context, actsBeforeCreate, object); err != nil {
			return nil, errors.Wrapf(
				err,
				"failed to run ActsBeforeCreate for %T",
				object,
			)
		}

		if err = Validate(qs.context, object); err != nil {
			return nil, errors.Wrapf(
				err,
				"failed to validate object %T before create",
				object,
			)
		}

		var defs = object.FieldDefs()
		var fields = defs.Fields()
		var info = UpdateInfo{
			FieldInfo: FieldInfo[attrs.Field]{
				Model: object,
				Table: Table{
					Name: defs.TableName(),
				},
				Chain: make([]string, 0),
			},
			Values: make([]any, 0, len(fields)),
		}

		for _, field := range fields {
			var atts = field.Attrs()
			var v, ok = atts[attrs.AttrAutoIncrementKey]
			if ok && v.(bool) {
				continue
			}

			if migrator.CanAutoIncrement(field) || !ForDBEdit(field) {
				continue
			}

			var isPrimary = field.IsPrimary()
			if isPrimary && primary == nil {
				primary = field
			}

			var value, err = field.Value()
			if err != nil {
				// panic(fmt.Errorf("failed to get value for field %q: %w", field.Name(), err))
				return nil, errors.ValueError.WithCause(errors.Wrapf(
					err, "failed to get value for field %q in %T",
					field.Name(), object,
				))
			}

			var rVal = reflect.ValueOf(value)
			if value == nil && !field.AllowNull() ||
				rVal.Kind() == reflect.Ptr && rVal.IsNil() && !field.AllowNull() {
				return nil, errors.FieldNull.WithCause(fmt.Errorf(
					"field %q cannot be null in %T",
					field.Name(), object,
				))
			}

			if rVal.Kind() == reflect.Ptr && rVal.IsNil() {
				// Make sure to set the value to an untyped nil
				// the mysql compiler might otherwise throw a fit because
				// it used the raw connection, which has some issues...
				value = nil
			}

			info.Fields = append(info.Fields, field)
			info.Values = append(info.Values, value)
		}

		// Copy all the fields from the model to the info fields
		infos = append(infos, info)
	}

	var support = qs.compiler.SupportsReturning()
	var resultQuery = qs.compiler.BuildCreateQuery(
		qs.context,
		ChangeObjectsType[T, attrs.Definer](qs),
		qs.internals,
		infos,
	)
	qs.latestQuery = resultQuery

	// Set the old values on the new object

	// Execute the create query
	results, err := resultQuery.Exec()
	if err != nil {
		return nil, err
	}

	// Check results & which returning method to use
	switch {
	case support == drivers.SupportsReturningNone:

		if len(results) > 0 {
			//return nil, errors.Wrapf(
			//	errors.ErrLastInsertId,
			//	"expected no results returned after insert, got %d",
			//	len(results),
			//)
			return nil, errors.LastInsertId.WithCause(fmt.Errorf(
				"expected no results returned after insert, got %d",
				len(results),
			))
		}

		for _, row := range objects {

			if _, err = runActor(qs.context, actsAfterCreate, row); err != nil {
				return nil, errors.Wrapf(
					err,
					"failed to run ActsAfterCreate for %T",
					row,
				)
			}

			if err = Validate(qs.context, row); err != nil {
				return nil, errors.Wrapf(
					err,
					"failed to validate object %T after create",
					row,
				)
			}
		}

	case support == drivers.SupportsReturningLastInsertId:

		// No results are returned, we cannot set the primary key
		// so we can return and commit the transaction
		if qs.internals.Model.Primary == nil || !migrator.CanAutoIncrement(qs.internals.Model.Primary) {
			return objects, tx.Commit(qs.context)
		}

		// If no results are returned, we cannot set the primary key
		// warn about this, return the objects and commit the transaction
		// this is in case the model's primary key is not an auto-incrementing field (uuid, etc.)
		if len(results) == 0 {
			logger.Warnf(
				"no results returned after insert, cannot set primary key for %T",
				qs.internals.Model.Object,
			)
			return objects, tx.Commit(qs.context)
		}

		if len(results) != len(objects) {
			return nil, errors.LastInsertId.WithCause(fmt.Errorf(
				"expected %d results returned after insert, got %d",
				len(objects), len(results),
			))
		}

		for i, row := range objects {

			var rowDefs = row.FieldDefs()
			var prim = rowDefs.Primary()
			if prim != nil {
				if len(results[i]) != 1 {
					return nil, errors.LastInsertId.WithCause(fmt.Errorf(
						"expected 1 result returned after insert, got %d (%+v)",
						len(results[i]), results[i],
					))
				}
				var id = results[i][0].(int64)
				if err := prim.SetValue(id, true); err != nil {
					return nil, errors.ValueError.WithCause(fmt.Errorf(
						"failed to set primary key %q in %T: %w: %w",
						prim.Name(), row, err, errors.LastInsertId,
					))
				}
			}

			//	if prim != nil {
			//		row = Setup[T](row)
			//	}

			if _, err = runActor(qs.context, actsAfterCreate, row); err != nil {
				return nil, errors.Wrapf(
					err,
					"failed to run ActsAfterCreate for %T",
					row,
				)
			}

			if err = Validate(qs.context, row); err != nil {
				return nil, errors.Wrapf(
					err,
					"failed to validate object %T after create",
					row,
				)
			}
		}

	case support == drivers.SupportsReturningColumns:

		if len(results) != len(objects) {
			return nil, errors.LastInsertId.WithCause(fmt.Errorf(
				"expected %d results returned after insert, got %d (len(results) != len(objects))",
				len(objects), len(results),
			))
		}

		for i, row := range objects {
			var (
				scannables = getScannableFields([]*FieldInfo[attrs.Field]{&infos[i].FieldInfo}, row)
				resLen     = len(results[i])
				newDefs    = row.FieldDefs()
				prim       = newDefs.Primary()
			)

			// only decrease the result length if the primary key is not set
			//
			// the underlying compiler should ensure that there is no duplicate
			// primary key in the RETURNING clause **IF** there was no primary key
			// included in the query (i.e. primary == nil).
			if prim != nil && primary == nil {
				resLen--
			}

			if len(scannables) != resLen {
				return nil, errors.LastInsertId.WithCause(fmt.Errorf(
					"expected %d results returned after insert, got %d (len(scannables) != resLen)",
					len(scannables), resLen,
				))
			}

			// the underlying compiler should ensure that there is no duplicate
			// primary key in the RETURNING clause **IF** there was no primary key
			// included in the query (i.e. primary == nil).
			var idx = 0
			if prim != nil && primary == nil {
				if err := prim.Scan(results[i][0]); err != nil {
					return nil, errors.ValueError.WithCause(errors.Wrapf(
						err, "failed to scan primary key %q in %T",
						prim.Name(), row,
					))
				}
				idx++
			}

			for j, field := range scannables {
				var f = field.field
				var val = results[i][j+idx]

				if err := f.Scan(val); err != nil {
					return nil, errors.ValueError.WithCause(fmt.Errorf(
						"failed to scan field %q in %T: %w",
						f.Name(), row, err,
					))
				}
			}

			if _, err = runActor(qs.context, actsAfterCreate, row); err != nil {
				return nil, errors.Wrapf(
					err,
					"failed to run ActsAfterCreate for %T",
					row,
				)
			}

			if err = Validate(qs.context, row); err != nil {
				return nil, errors.Wrapf(
					err,
					"failed to validate object %T after create",
					row,
				)
			}
		}
	default:
		return nil, errors.LastInsertId.WithCause(fmt.Errorf(
			"unsupported returning method %q for %T",
			support, qs.internals.Model.Object,
		))
	}

	return objects, tx.Commit(qs.context)
}

// BulkUpdate is used to update multiple objects in the database.
//
// It takes a list of definer objects as arguments and any possible NamedExpressions.
// It does not try to call any save methods on the objects.
//
// It will run the actor methods of [ActsBeforeUpdate] and [ActsAfterUpdate] for each object,
// as well as send the [SignalPreModelSave] and [SignalPostModelSave] signals.
func (qs *QuerySet[T]) BulkUpdate(objects []T, expressions ...any) (int64, error) {

	var tx, err = qs.GetOrCreateTransaction()
	if err != nil {
		return 0, errors.FailedStartTransaction.WithCause(err)
	}
	defer tx.Rollback(qs.context)

	var exprMap = make(map[string]expr.NamedExpression, len(expressions))
	for _, expression := range expressions {
		switch e := expression.(type) {
		case expr.NamedExpression:
			var fieldName = e.FieldName()
			if fieldName == "" {
				panic(fmt.Errorf("expression %q has no field name", e))
			}

			if _, ok := exprMap[fieldName]; ok {
				panic(fmt.Errorf("duplicate field %q in update expression", fieldName))
			}

			exprMap[fieldName] = e

		case map[string]expr.Expression:
			for fieldName, e := range e {
				if _, ok := exprMap[fieldName]; ok {
					panic(fmt.Errorf("duplicate field %q in update expression", fieldName))
				}

				if named, ok := e.(expr.NamedExpression); ok {
					exprMap[fieldName] = named
				} else {
					exprMap[fieldName] = expr.As(fieldName, e)
				}
			}
		}
	}

	var (
		infos = make([]UpdateInfo, 0, len(objects))
		where = slices.Clone(qs.internals.Where)
		joins = slices.Clone(qs.internals.Joins)
	)

	var typ reflect.Type
	for _, obj := range objects {

		var err error
		obj, err = setup(obj)
		if err != nil {
			return 0, errors.Wrapf(
				err, "failed to setup object %T", obj,
			)
		}

		if typ == nil {
			typ = reflect.TypeOf(obj)
		} else if typ != reflect.TypeOf(obj) {
			panic(fmt.Errorf(
				"QuerySet: all objects must be of the same type, got %T and %T",
				typ, reflect.TypeOf(obj),
			))
		}

		if _, err = runActor(qs.context, actsBeforeUpdate, obj); err != nil {
			return 0, errors.Wrapf(
				err, "failed to run ActsBeforeUpdate for %T", obj,
			)
		}

		if err = Validate(qs.context, obj); err != nil {
			return 0, errors.Wrapf(
				err, "failed to validate object %T before update", obj,
			)
		}

		var defs, fields = qs.updateFields(obj)
		var info = UpdateInfo{
			FieldInfo: FieldInfo[attrs.Field]{
				Model: obj,
				Table: Table{
					Name: defs.TableName(),
				},
				Fields: make([]attrs.Field, 0, len(fields)),
			},
			Values: make([]any, 0, len(fields)),
		}

		for _, field := range fields {
			var atts = field.Attrs()
			var v, ok = atts[attrs.AttrAutoIncrementKey]
			if ok && v.(bool) {
				continue
			}

			if migrator.CanAutoIncrement(field) || !ForDBEdit(field) {
				continue
			}

			var fieldName = field.Name()
			if expr, ok := exprMap[fieldName]; ok {
				info.FieldInfo.Fields = append(info.FieldInfo.Fields, &exprField{
					Field: field,
					expr:  expr,
				})
				continue
			}

			var value, err = field.Value()
			if err != nil {
				return 0, errors.ValueError.WithCause(errors.Wrapf(
					err, "failed to get value for field %q in %T",
					field.Name(), obj,
				))
			}

			if value == nil && !field.AllowNull() {
				return 0, errors.FieldNull.WithCause(fmt.Errorf(
					"field %q cannot be null in %T",
					field.Name(), obj,
				))
			}

			info.FieldInfo.Fields = append(info.FieldInfo.Fields, field)
			info.Values = append(info.Values, value)
		}

		if len(where) == 0 {
			var err error
			info.Where, err = GenerateObjectsWhereClause(obj)
			if err != nil {
				return 0, errors.NoWhereClause.WithCause(err)
			}
		} else {
			info.Where = where
		}

		if len(joins) > 0 {
			info.Joins = joins
		}

		infos = append(infos, info)
	}

	var resultQuery = qs.compiler.BuildUpdateQuery(
		qs.context,
		ChangeObjectsType[T, attrs.Definer](qs),
		qs.internals,
		infos,
	)
	qs.latestQuery = resultQuery

	res, err := resultQuery.Exec()
	if err != nil {
		return 0, err
	}

	if len(objects) > 0 && res == 0 {
		return 0, errors.NoChanges.WithCause(fmt.Errorf(
			"no rows updated for %T, expected %d rows",
			qs.internals.Model.Object, len(objects),
		))
	}

	// Must always run the after update actor
	// even if it was not implemented - the signal must be sent
	for _, obj := range objects {
		if _, err = runActor(qs.context, actsAfterUpdate, obj); err != nil {
			return 0, errors.Wrapf(
				err,
				"failed to run ActsAfterUpdate for %T",
				obj,
			)
		}

		if err = Validate(qs.context, obj); err != nil {
			return 0, errors.Wrapf(
				err,
				"failed to validate object %T after update",
				obj,
			)
		}
	}

	return res, tx.Commit(qs.context)
}

// BatchCreate is used to create multiple objects in the database.
//
// It takes a list of definer objects as arguments and returns a Query that can be executed
// to get the result, which is a slice of the created objects.
//
// The query is executed in a transaction, so if any error occurs, the transaction is rolled back.
// You can specify the batch size to limit the number of objects created in a single query.
//
// The batch size is based on the [Limit] method of the queryset, which defaults to 1000.
func (qs *QuerySet[T]) BatchCreate(objects []T) ([]T, error) {

	var tx, err = qs.GetOrCreateTransaction()
	if err != nil {
		return nil, errors.FailedStartTransaction.WithCause(err)
	}
	defer tx.Rollback(qs.context)

	var createdObjects = make([]T, 0, len(objects))
	for batchNum, batch := range qs.batch(objects, qs.internals.Limit) {
		var result, err = qs.BulkCreate(batch)
		if err != nil {
			return nil, errors.Wrapf(
				err, "failed to create batch %d of %d objects", batchNum, len(batch),
			)
		}

		createdObjects = append(
			createdObjects,
			result...,
		)
	}

	return createdObjects, tx.Commit(qs.context)
}

// BatchUpdate is used to update multiple objects in the database.
//
// It takes a list of definer objects as arguments and returns a Query that can be executed
// to get the result, which is a slice of the updated objects.
//
// The query is executed in a transaction, so if any error occurs, the transaction is rolled back.
// You can specify the batch size to limit the number of objects updated in a single query.
//
// The batch size is based on the [Limit] method of the queryset, which defaults to 1000.
func (qs *QuerySet[T]) BatchUpdate(objects []T, exprs ...any) (int64, error) {
	if len(objects) == 0 {
		return 0, nil // No objects to update
	}

	var tx, err = qs.GetOrCreateTransaction()
	if err != nil {
		return 0, errors.FailedStartTransaction.WithCause(err)
	}
	defer tx.Rollback(qs.context)

	var updatedObjects int64 = 0
	for batchNum, batch := range qs.batch(objects, qs.internals.Limit) {
		var count, err = qs.BulkUpdate(batch, exprs...)
		if err != nil {
			return 0, errors.Wrapf(
				err, "failed to update batch %d of %d objects", batchNum, len(batch),
			)
		}

		if count == 0 {
			return 0, errors.NoChanges.WithCause(fmt.Errorf(
				"no rows updated for batch %d of %T",
				batchNum, qs.internals.Model.Object,
			))
		}

		updatedObjects += count
	}

	return updatedObjects, tx.Commit(qs.context)
}

// Delete is used to delete an object from the database.
//
// It returns a CountQuery that can be executed to get the result, which is the number of rows affected.
//
// If any objects are provided, it will generate a where clause based on [GenerateObjectsWhereClause].
// It will also run the [ActsBeforeDelete] and [ActsAfterDelete] actor methods and
// send [SignalPreModelDelete] and [SignalPostModelDelete] signals.
func (qs *QuerySet[T]) Delete(objects ...T) (int64, error) {

	var tx, err = qs.GetOrCreateTransaction()
	if err != nil {
		return 0, errors.FailedStartTransaction.WithCause(err)
	}
	defer tx.Rollback(qs.context)

	if len(objects) > 0 {
		for _, obj := range objects {
			if _, err = runActor(qs.context, actsBeforeDelete, obj); err != nil {
				return 0, errors.Wrapf(
					err, "failed to run ActsBeforeDelete for %T", obj,
				)
			}
		}

		var where, err = GenerateObjectsWhereClause(objects...)
		if err != nil {
			return 0, errors.NoWhereClause.WithCause(errors.Wrapf(
				err, "failed to generate where clause for %T",
				qs.internals.Model.Object,
			))
		}

		qs.internals.Where = append(qs.internals.Where, where...)
	}

	var resultQuery = qs.compiler.BuildDeleteQuery(
		qs.context,
		ChangeObjectsType[T, attrs.Definer](qs),
		qs.internals,
	)
	qs.latestQuery = resultQuery

	res, err := resultQuery.Exec()
	if err != nil {
		return 0, err
	}

	if len(objects) > 0 {
		for _, obj := range objects {
			if _, err = runActor(qs.context, actsAfterDelete, obj); err != nil {
				return 0, errors.Wrapf(
					err, "failed to run ActsAfterDelete for %T", obj,
				)
			}
		}
	}

	return res, tx.Commit(qs.context)
}

func (qs *QuerySet[T]) tryParseExprStatement(sqlStr string, args ...interface{}) (string, []interface{}) {
	var (
		changedQs = ChangeObjectsType[T, attrs.Definer](qs)
		info      = qs.compiler.ExpressionInfo(changedQs, qs.internals)
		stmt      = expr.ParseExprStatement(sqlStr, args)
		resolved  = stmt.Resolve(info)
	)

	sqlStr, args = resolved.SQL()

	if rebinder, ok := qs.compiler.(RebindCompiler); ok {
		sqlStr = rebinder.Rebind(qs.Context(), sqlStr)
	}

	qs.latestQuery = &QueryInformation{
		Stmt:    sqlStr,
		Params:  args,
		Builder: qs.compiler,
	}

	return sqlStr, args
}

// Rows is used to execute a raw SQL query on the compilers' current database.
//
// It returns a [drivers.SQLRows] object that can be used to iterate over the results.
// The same transaction and context as the rest of the QuerySet will be used.
//
// It first tries to resolve and parse the SQL statement, see [expr.ParseExprStatement] for more details.
func (qs *QuerySet[T]) Rows(sqlStr string, args ...interface{}) (drivers.SQLRows, error) {
	sqlStr, args = qs.tryParseExprStatement(sqlStr, args...)
	return qs.compiler.DB().QueryContext(qs.Context(), sqlStr, args...)
}

// Row is used to execute a raw SQL query on the compilers' current database
// and returns a single row of type [drivers.SQLRow].
//
// It first tries to resolve and parse the SQL statement, see [expr.ParseExprStatement] for more details.
func (qs *QuerySet[T]) Row(sqlStr string, args ...interface{}) drivers.SQLRow {
	sqlStr, args = qs.tryParseExprStatement(sqlStr, args...)
	return qs.compiler.DB().QueryRowContext(qs.Context(), sqlStr, args...)
}

// Exec executes the given SQL on the compilers' current database and returns the result.
//
// It uses the same context and transaction as the rest of the QuerySet.
//
// It first tries to resolve and parse the SQL statement, see [expr.ParseExprStatement] for more details.
func (qs *QuerySet[T]) Exec(sqlStr string, args ...interface{}) (sql.Result, error) {
	sqlStr, args = qs.tryParseExprStatement(sqlStr, args...)
	return qs.compiler.DB().ExecContext(qs.Context(), sqlStr, args...)
}

// updateFields is used to get the field definitions and fields to be updated for the given object.
// It returns the field definitions and the fields to be updated based on the current QuerySet selection.
func (qs *QuerySet[T]) updateFields(obj attrs.Definer) (attrs.Definitions, []attrs.Field) {
	var defs = obj.FieldDefs()
	var fields []attrs.Field
	if len(qs.internals.Fields) > 0 {
		fields = make([]attrs.Field, 0, len(qs.internals.Fields))
		for _, info := range qs.internals.Fields {
			for _, field := range info.Fields {
				var f, ok = defs.Field(field.Name())
				if !ok {
					panic(errors.FieldNotFound.WithCause(fmt.Errorf(
						"field %q not found in %T", field.Name(), obj,
					)))
				}
				fields = append(fields, f)
			}
		}
	} else {
		var all = defs.Fields()
		fields = make([]attrs.Field, 0, len(all))
		for _, field := range all {
			var val = field.GetValue()
			var rVal = reflect.ValueOf(val)
			if rVal.IsValid() && rVal.IsZero() {
				continue
			}
			fields = append(fields, field)
		}
	}
	return defs, fields
}

// batch is used to batch a list of objects into smaller chunks.
func (qs *QuerySet[T]) batch(objects []T, size int) iter.Seq2[int, []T] {
	if len(objects) == 0 {
		return func(yield func(int, []T) bool) {}
	}

	if size <= 0 {
		size = MAX_DEFAULT_RESULTS
	}

	return func(yield func(int, []T) bool) {
		var batchNum int
		for i := 0; i < len(objects); i += size {
			var end = min(i+size, len(objects))
			var batch = objects[i:end]

			if len(batch) == 0 {
				continue
			}

			if !yield(batchNum, batch) {
				return
			}

			batchNum++
		}
	}
}

func calcJoinType(rel attrs.Relation, parentField attrs.FieldDefinition) JoinType {
	if rel == nil {
		panic(fmt.Errorf(
			"calcJoinType: relation is nil for field %q in model %T",
			parentField.Name(), parentField.Instance(),
		))
	}
	var relType = rel.Type()
	switch relType {
	case attrs.RelManyToOne:
		if !parentField.AllowNull() {
			return TypeJoinInner
		}
		return TypeJoinLeft

	case attrs.RelOneToOne:
		if rel.Through() != nil {
			return TypeJoinInner
		}
		if !parentField.AllowNull() {
			return TypeJoinInner
		}
		return TypeJoinLeft

	case attrs.RelOneToMany:
		return TypeJoinLeft

	case attrs.RelManyToMany:
		return TypeJoinInner

	default:
		panic(fmt.Errorf("unknown relation type %d for field %q", relType, parentField.Name()))
	}
}

func getDatabaseName(model attrs.Definer, database ...string) string {
	var defaultDb = django.APPVAR_DATABASE
	if len(database) > 1 {
		panic("QuerySet: too many databases provided")
	}

	if model != nil {
		// If the model implements the QuerySetDatabaseDefiner interface,
		// it will use the QuerySetDatabase method to get the default database.
		if m, ok := any(model).(QuerySetDatabaseDefiner); ok && len(database) == 0 {
			defaultDb = m.QuerySetDatabase()
		}
	}

	// Arguments take precedence over the default database
	if len(database) == 1 {
		defaultDb = database[0]
	}

	return defaultDb
}

type scannableField struct {
	idx       int
	object    attrs.Definer
	field     attrs.Field
	srcField  *scannableField
	relType   attrs.RelationType
	isThrough bool          // is this a through model field (many-to-many or one-to-one)
	through   attrs.Definer // the through field if this is a many-to-many or one-to-one relation
	chainPart string        // name of the field in the chain
	chainKey  string        // the chain up to this point, joined by "."
}

func getScannableFields[T attrs.FieldDefinition](fields []*FieldInfo[T], root attrs.Definer) []*scannableField {
	var listSize = 0
	for _, info := range fields {
		listSize += len(info.Fields)
	}

	var (
		scannables    = make([]*scannableField, 0, listSize)
		instances     = make(map[string]attrs.Definer)
		parentFields  = make(map[string]*scannableField) // NEW: store parent scannableFields by chain
		rootScannable *scannableField
		idx           = 0
	)

	for _, info := range fields {
		// handle through objects
		//
		// this has to be before the final fields are added - the logic
		// matches that in [FieldInfo.WriteFields].
		var throughObj attrs.Definer
		if info.Through != nil {
			var newObj = internal.NewObjectFromIface(info.Through.Model)
			var newDefs = newObj.FieldDefs()
			throughObj = newObj

			for _, f := range info.Through.Fields {
				var field, ok = newDefs.Field(f.Name())
				if !ok {
					panic(fmt.Errorf("field %q not found in %T", f.Name(), newObj))
				}

				var throughField = &scannableField{
					isThrough: true,
					idx:       idx,
					object:    newObj,
					field:     field,
					relType:   info.RelType,
				}

				scannables = append(scannables, throughField)
				idx++
			}
		}

		// if isNil(reflect.ValueOf(info.SourceField)) {
		if any(info.SourceField) == any(*(new(T))) {
			defs := root.FieldDefs()
			for _, f := range info.Fields {
				if virt, ok := any(f).(VirtualField); ok && info.Model == nil {
					var attrField, ok = virt.(attrs.Field)
					if !ok {
						panic(fmt.Errorf("virtual field %q does not implement attrs.Field", f.Name()))
					}

					scannables = append(scannables, &scannableField{
						idx:     idx,
						field:   attrField,
						relType: -1,
					})
					idx++
					continue
				}
				field, ok := defs.Field(f.Name())
				if !ok {
					panic(fmt.Errorf("field %q not found in %T", f.Name(), root))
				}

				var sf = &scannableField{
					idx:     idx,
					field:   field,
					object:  root,
					through: throughObj,
					relType: -1,
				}

				if field.IsPrimary() && rootScannable == nil {
					rootScannable = sf
				}

				scannables = append(scannables, sf)
				idx++
			}
			continue
		}

		instances[""] = root
		parentFields[""] = rootScannable

		// Walk chain
		var (
			parentScannable = rootScannable
			parentObj       = root

			parentKey string
		)
		for i, name := range info.Chain {
			key := strings.Join(info.Chain[:i+1], ".")
			parent := instances[parentKey]
			defs := parent.FieldDefs()
			field, ok := defs.Field(name)
			if !ok {
				panic(fmt.Errorf("field %q not found in %T", name, parent))
			}

			var rel = field.Rel()
			var relType = rel.Type()
			if _, exists := instances[key]; !exists {
				var obj attrs.Definer
				if i == len(info.Chain)-1 {
					obj = internal.NewObjectFromIface(info.Model)
				} else {
					obj = internal.NewObjectFromIface(rel.Model())
				}

				// only set fk relations - the rest are added later
				// in the dedupe rows object.
				if relType == attrs.RelManyToOne {
					setRelatedObjects(
						name,
						relType,
						parent,
						[]Relation{&baseRelation{object: obj}},
					)
				}

				instances[key] = obj

				// Make the scannableField node for this relation link to its parent
				newParent := &scannableField{
					relType:   relType,
					chainPart: name,
					chainKey:  key,
					object:    obj,
					field:     obj.FieldDefs().Primary(),
					idx:       -1,                      // Not a leaf
					srcField:  parentFields[parentKey], // link to parent in the chain
				}
				parentFields[key] = newParent
			}

			parentScannable = parentFields[key]
			parentObj = instances[key]
			parentKey = key
		}

		var final = parentObj
		var finalDefs = final.FieldDefs()
		for _, f := range info.Fields {
			field, ok := finalDefs.Field(f.Name())
			if !ok {
				panic(fmt.Errorf("field %q not found in %T", f.Name(), final))
			}

			var cpy = *parentScannable
			cpy.idx = idx
			cpy.object = final
			cpy.field = field
			cpy.through = throughObj
			scannables = append(scannables, &cpy)

			idx++
		}
	}

	return scannables
}
