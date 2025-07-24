package queries

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"reflect"
	"strings"
	"unsafe"

	"github.com/Nigel2392/go-django/queries/internal"
	"github.com/Nigel2392/go-django/queries/src/alias"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/queries/src/expr"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/go-sql-driver/mysql"
	"github.com/jackc/pgx/v5"
)

var (
	_ QueryCompiler = (*genericQueryBuilder)(nil)
	_ QueryCompiler = (*postgresQueryBuilder)(nil)
	_ QueryCompiler = (*mariaDBQueryBuilder)(nil)
	_ QueryCompiler = (*mariaDBQueryBuilder)(nil)

	_ RebindCompiler = (*genericQueryBuilder)(nil)
	_ RebindCompiler = (*postgresQueryBuilder)(nil)
	_ RebindCompiler = (*mariaDBQueryBuilder)(nil)
	_ RebindCompiler = (*mariaDBQueryBuilder)(nil)
)

func init() {
	RegisterCompiler(&drivers.DriverSQLite{}, NewGenericQueryBuilder)
	RegisterCompiler(&drivers.DriverPostgres{}, NewPostgresQueryBuilder)
	RegisterCompiler(&drivers.DriverMariaDB{}, NewMariaDBQueryBuilder)
	RegisterCompiler(&drivers.DriverMySQL{}, NewMySQLQueryBuilder)
}

func newExpressionInfo(g *genericQueryBuilder, qs *QuerySet[attrs.Definer], i *QuerySetInternals, updating bool) *expr.ExpressionInfo {
	var dbName = internal.SqlxDriverName(g.queryInfo.DB)
	var supportsWhereAlias bool
	switch dbName {
	case "mysql", "mariadb":
		supportsWhereAlias = false // MySQL does not support WHERE alias
	case "sqlite3":
		supportsWhereAlias = true
	case "postgres", "pgx":
		supportsWhereAlias = false // Postgres does not support WHERE alias
	default:
		panic(fmt.Errorf("unknown database driver: %s", dbName))
	}

	var exprInfo = &expr.ExpressionInfo{
		Driver: g.driver,
		Model: attrs.NewObject[attrs.Definer](
			qs.Meta().Model(),
		),
		Quote:           g.QuoteString,
		QuoteIdentifier: g.QuoteIdentifier,
		FormatField:     g.FormatColumn,
		Resolver:        qs,
		Placeholder:     generic_PLACEHOLDER,
		Lookups: expr.ExpressionLookupInfo{
			PrepForLikeQuery: g.PrepForLikeQuery,
			FormatLookupCol:  g.FormatLookupCol,
			LogicalOpRHS:     g.LogicalOpRHS(),
			OperatorsRHS:     g.LookupOperatorsRHS(),
			PatternOpsRHS:    g.LookupPatternOperatorsRHS(),
		},
		ForUpdate:          updating,
		Annotations:        i.Annotations,
		SupportsWhereAlias: supportsWhereAlias,
		SupportsAsExpr:     true,
	}

	return exprInfo
}

const generic_PLACEHOLDER = "?"

type genericQueryBuilder struct {
	transaction drivers.Transaction
	queryInfo   *internal.QueryInfo
	support     drivers.SupportsReturningType
	quote       string
	driver      driver.Driver
	self        QueryCompiler // for embedding purposes to link back to the top-most compiler
}

func NewGenericQueryBuilder(db string) QueryCompiler {
	var q, err = internal.GetQueryInfo(db)
	if err != nil {
		panic(err)
	}

	var quote = "`"
	switch internal.SqlxDriverName(q.DB) {
	case "mysql", "mariadb":
		quote = "`"
	case "postgres", "pgx":
		quote = "\""
	case "sqlite3":
		quote = "`"
	}

	return &genericQueryBuilder{
		quote:     quote,
		support:   drivers.SupportsReturning(q.DB),
		driver:    q.DB.Driver(),
		queryInfo: q,
	}
}

func (g *genericQueryBuilder) This() QueryCompiler {
	if g.self == nil {
		return g
	}
	return g.self
}

func (g *genericQueryBuilder) Rebind(ctx context.Context, s string) string {
	if !expr.IsSubqueryContext(ctx) {
		return g.queryInfo.DBX(s)
	}
	return s
}

func (g *genericQueryBuilder) DatabaseName() string {
	return g.queryInfo.DatabaseName
}

func (g *genericQueryBuilder) DB() drivers.DB {
	if g.InTransaction() {
		return g.transaction
	}
	return g.queryInfo.DB
}

func (g *genericQueryBuilder) Quote() (string, string) {
	return g.quote, g.quote
}

func (g *genericQueryBuilder) Placeholder() string {
	return generic_PLACEHOLDER
}

func (g *genericQueryBuilder) QuoteString(s string) string {
	var sb strings.Builder
	sb.Grow(len(s) + 2)
	switch internal.SqlxDriverName(g.queryInfo.DB) {
	case "mysql", "mariadb":
		sb.WriteString("'")
		sb.WriteString(s)
		sb.WriteString("'")
	case "sqlite3":
		sb.WriteString("'")
		sb.WriteString(s)
		sb.WriteString("'")
	case "postgres", "pgx":
		sb.WriteString("'")
		sb.WriteString(s)
		sb.WriteString("'")
	}
	return sb.String()
}

func (g *genericQueryBuilder) QuoteIdentifier(s string) string {
	var sb strings.Builder
	var front, back = g.Quote()
	sb.Grow(len(s) + len(front) + len(back))
	sb.WriteString(front)
	sb.WriteString(s)
	sb.WriteString(back)
	return sb.String()
}

func (g *genericQueryBuilder) PrepForLikeQuery(v any) string {
	// For LIKE queries, we need to escape the percent and underscore characters.
	// This is done by replacing them with their escaped versions.
	switch internal.SqlxDriverName(g.queryInfo.DB) {
	case "mysql", "mariadb":
		return strings.ReplaceAll(
			strings.ReplaceAll(fmt.Sprint(v), "%", "\\%"),
			"_", "\\_",
		)

	case "postgres", "pgx":
		return strings.ReplaceAll(
			strings.ReplaceAll(fmt.Sprint(v), "%", "\\%"),
			"_", "\\_",
		)

	case "sqlite3":
		return strings.ReplaceAll(
			strings.ReplaceAll(fmt.Sprint(v), "%", "\\%"),
			"_", "\\_",
		)

	default:
		panic(fmt.Errorf("unknown database driver: %s", internal.SqlxDriverName(g.queryInfo.DB)))
	}
}

func (g *genericQueryBuilder) FormatLookupCol(lookupName string, inner string) string {
	switch lookupName {
	case "iexact", "icontains", "istartswith", "iendswith":
		switch internal.SqlxDriverName(g.queryInfo.DB) {
		case "mysql", "mariadb":
			return fmt.Sprintf("LOWER(%s)", inner)
		case "postgres", "pgx":
			return fmt.Sprintf("LOWER(%s)", inner)
		case "sqlite3":
			return fmt.Sprintf("LOWER(%s)", inner)
		default:
			panic(fmt.Errorf("unknown database driver: %s", internal.SqlxDriverName(g.queryInfo.DB)))
		}
	default:
		return inner
	}
}

func equalityFormat(op expr.LogicalOp) func(string, []any) (string, []any) {
	return func(rhs string, value []any) (string, []any) {
		if len(value) == 0 {
			return fmt.Sprintf("%s %s", op, rhs), []any{}
		}
		return fmt.Sprintf("%s %s", op, rhs), []any{value[0]}
	}
}

func mathOpFormat(op expr.LogicalOp) func(string, []any) (string, []any) {
	return func(rhs string, value []any) (string, []any) {
		return fmt.Sprintf("%s %s = %s", op, rhs, rhs), []any{value[0], value[0]}
	}
}

var defaultCompilerLogicalOperators = map[expr.LogicalOp]func(rhs string, value []any) (string, []any){
	expr.EQ:  equalityFormat(expr.EQ),  // = %s
	expr.NE:  equalityFormat(expr.NE),  // != %s
	expr.GT:  equalityFormat(expr.GT),  // > %s
	expr.LT:  equalityFormat(expr.LT),  // < %s
	expr.GTE: equalityFormat(expr.GTE), // >= %s
	expr.LTE: equalityFormat(expr.LTE), // <= %s
	//expr.ADD:    mathOpFormat(expr.ADD),    // + %s = %s
	//expr.SUB:    mathOpFormat(expr.SUB),    // - %s = %s
	//expr.MUL:    mathOpFormat(expr.MUL),    // * %s = %s
	//expr.DIV:    mathOpFormat(expr.DIV),    // / %s = %s
	//expr.MOD:    mathOpFormat(expr.MOD),    // % %s = %s
	expr.BITAND: mathOpFormat(expr.BITAND), // & %s = %s
	expr.BITOR:  mathOpFormat(expr.BITOR),  // | %s = %s
	expr.BITXOR: mathOpFormat(expr.BITXOR), // ^ %s = %s
	expr.BITLSH: mathOpFormat(expr.BITLSH), // << %s = %s
	expr.BITRSH: mathOpFormat(expr.BITRSH), // >> %s = %s
	expr.BITNOT: mathOpFormat(expr.BITNOT), // ~ %s = %s
}

func (g *genericQueryBuilder) LogicalOpRHS() map[expr.LogicalOp]func(rhs string, value []any) (string, []any) {
	return defaultCompilerLogicalOperators
}

func (g *genericQueryBuilder) LookupOperatorsRHS() map[string]string {
	switch internal.SqlxDriverName(g.queryInfo.DB) {
	case "mysql", "mariadb":
		return map[string]string{
			"iexact":      "= LOWER(%s)",
			"contains":    "LIKE LOWER(%s)",
			"icontains":   "LIKE %s",
			"startswith":  "LIKE LOWER(%s)",
			"endswith":    "LIKE LOWER(%s)",
			"istartswith": "LIKE %s",
			"iendswith":   "LIKE %s",
		}
	case "postgres", "pgx":
		return map[string]string{
			"iexact":      "= LOWER(%s)",
			"contains":    "LIKE %s",
			"icontains":   "LIKE LOWER(%s)",
			"regex":       "~ %s",
			"startswith":  "LIKE %s",
			"endswith":    "LIKE %s",
			"istartswith": "LIKE LOWER(%s)",
			"iendswith":   "LIKE LOWER(%s)",
		}
	case "sqlite3":
		return map[string]string{
			"iexact":      "LIKE %s ESCAPE '\\'",
			"contains":    "LIKE %s ESCAPE '\\'",
			"icontains":   "LIKE %s ESCAPE '\\'",
			"regex":       "REGEXP %s",
			"iregex":      "REGEXP '(?i)' || %s",
			"startswith":  "LIKE %s ESCAPE '\\'",
			"endswith":    "LIKE %s ESCAPE '\\'",
			"istartswith": "LIKE %s ESCAPE '\\'",
			"iendswith":   "LIKE %s ESCAPE '\\'",
		}
	}
	panic(fmt.Errorf("unknown database driver: %s", internal.SqlxDriverName(g.queryInfo.DB)))
}

func (g *genericQueryBuilder) LookupPatternOperatorsRHS() map[string]string {
	switch internal.SqlxDriverName(g.queryInfo.DB) {
	case "mysql", "mariadb":
		return map[string]string{
			"contains":    "LIKE CONCAT('%%', %s, '%%')",
			"icontains":   "LIKE LOWER(CONCAT('%%', %s, '%%'))",
			"startswith":  "LIKE CONCAT(%s, '%%')",
			"istartswith": "LIKE LOWER(CONCAT(%s, '%%'))",
			"endswith":    "LIKE CONCAT('%%', %s)",
			"iendswith":   "LIKE LOWER(CONCAT('%%', %s))",
		}
	case "postgres", "pgx":
		return map[string]string{
			"contains":    "LIKE '%%' || %s || '%%'",
			"icontains":   "LIKE '%%' || LOWER(%s) || '%%'",
			"startswith":  "LIKE %s || '%%'",
			"istartswith": "LIKE LOWER(%s) || '%%'",
			"endswith":    "LIKE '%%' || %s",
			"iendswith":   "LIKE '%%' || LOWER(%s)",
		}
	case "sqlite3":
		return map[string]string{
			"contains":    "LIKE '%%' || %s || '%%' ESCAPE '\\'",
			"icontains":   "LIKE '%%' || LOWER(%s) || '%%' ESCAPE '\\'",
			"startswith":  "LIKE %s || '%%' ESCAPE '\\'",
			"istartswith": "LIKE LOWER(%s) || '%%' ESCAPE '\\'",
			"endswith":    "LIKE '%%' || %s ESCAPE '\\'",
			"iendswith":   "LIKE '%%' || LOWER(%s) ESCAPE '\\'",
		}
	}
	panic(fmt.Errorf("unknown database driver: %s", internal.SqlxDriverName(g.queryInfo.DB)))
}

func (g *genericQueryBuilder) ExpressionInfo(
	qs *GenericQuerySet,
	internals *QuerySetInternals,
) *expr.ExpressionInfo {
	return newExpressionInfo(g, qs, internals, false)
}

func (g *genericQueryBuilder) FormatColumn(aliasGen *alias.Generator, col *expr.TableColumn) (string, []any) {
	var (
		sb   = new(strings.Builder)
		args = make([]any, 0, 1)
	)

	var err = col.Validate()
	if err != nil {
		panic(fmt.Errorf("cannot format column: %w", err))
	}

	if col.TableOrAlias != "" {
		sb.WriteString(g.quote)

		if aliasGen.Prefix != "" {
			sb.WriteString(aliasGen.Prefix)
			sb.WriteString("_")
		}

		sb.WriteString(col.TableOrAlias)
		sb.WriteString(g.quote)
		sb.WriteString(".")
	}

	var aliasWritten bool
	switch {
	case col.FieldColumn != nil:
		var colName = col.FieldColumn.ColumnName()
		if colName == "" {
			panic(fmt.Errorf("cannot format column with empty column name: %+v (%s)", col, col.FieldColumn.Name()))
		}
		sb.WriteString(g.quote)
		sb.WriteString(colName)
		sb.WriteString(g.quote)

	case col.RawSQL != "":
		sb.WriteString(col.RawSQL)
		args = append(args, col.Values...)
		//if col.Value != nil {
		//	args = append(args, col.Value)
		//}

	case len(col.Values) > 0:
		var flattened bool = false
		var values = make([]any, 0, len(col.Values))
		for _, v := range col.Values {
			if val, ok := v.([]interface{}); ok {
				values = append(values, val...)
				flattened = true
			} else {
				values = append(values, v)
			}
		}

		if len(values) > 1 || flattened {
			sb.WriteString("(")
		}

		for i, v := range values {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(generic_PLACEHOLDER)
			args = append(args, v)
		}

		if len(values) > 1 || flattened {
			sb.WriteString(")")
		}

		//sb.WriteString(generic_PLACEHOLDER)
		//args = append(args, col.Value)

	case col.FieldAlias != "":
		aliasWritten = true
		sb.WriteString(g.quote)
		sb.WriteString(col.FieldAlias)
		sb.WriteString(g.quote)

	default:
		panic(fmt.Errorf("cannot format column, no field, value or raw SQL provided: %+v", col))
	}

	if col.FieldAlias != "" && !aliasWritten {
		sb.WriteString(" AS ")
		sb.WriteString(g.quote)
		sb.WriteString(col.FieldAlias)
		sb.WriteString(g.quote)
	}

	// Values are not used in the column definition.
	// We don't append them here.
	if col.ForUpdate {
		sb.WriteString(" = ")
		sb.WriteString(generic_PLACEHOLDER)
	}

	return sb.String(), args
}

func (g *genericQueryBuilder) Transaction() drivers.Transaction {
	if g.InTransaction() {
		return g.transaction
	}
	return nil
}

func (g *genericQueryBuilder) StartTransaction(ctx context.Context) (drivers.Transaction, error) {
	if g.InTransaction() {
		return g.transaction, nil
	}

	var tx, err = g.queryInfo.DB.Begin(ctx)
	if err != nil {
		return nil, errors.FailedStartTransaction.WithCause(err)
	}

	// logger.Debugf("Starting transaction for %s", g.DatabaseName())

	return g.WithTransaction(tx)
}

func (g *genericQueryBuilder) WithTransaction(t drivers.Transaction) (drivers.Transaction, error) {
	if g.InTransaction() {
		return nil, errors.TransactionStarted
	}

	if t == nil {
		return nil, errors.TransactionNil
	}

	g.transaction = &wrappedTransaction{t, g}
	return g.transaction, nil
}

func (g *genericQueryBuilder) CommitTransaction(ctx context.Context) error {
	if !g.InTransaction() {
		return errors.NoTransaction
	}
	return g.transaction.Commit(ctx)
}

func (g *genericQueryBuilder) RollbackTransaction(ctx context.Context) error {
	if !g.InTransaction() {
		return errors.NoTransaction
	}
	return g.transaction.Rollback(ctx)
}

func (g *genericQueryBuilder) InTransaction() bool {
	return g.transaction != nil && !g.transaction.Finished()
}

func (g *genericQueryBuilder) SupportsReturning() drivers.SupportsReturningType {
	return g.support
}

func (g *genericQueryBuilder) BuildSelectQuery(
	ctx context.Context,
	qs *GenericQuerySet,
	internals *QuerySetInternals,
) CompiledQuery[[][]interface{}] {
	var (
		query = new(strings.Builder)
		args  []any
		inf   = newExpressionInfo(g, qs, internals, false)
	)

	query.WriteString("SELECT ")

	if internals.Distinct {
		query.WriteString("DISTINCT ")
	}

	for i, info := range internals.Fields {
		if i > 0 {
			query.WriteString(", ")
		}

		args = append(
			args, info.WriteFields(
				query, inf)...)
	}

	query.WriteString(" FROM ")
	g.writeTableName(query, qs.AliasGen, internals)

	// First we must resolve all where, having clauses & group by clauses.
	// These might add joins to the queryset, so this
	// must be done before we write the joins to the query.
	var sb2 = new(strings.Builder)
	var args2 = make([]any, 0, 8)
	args2 = append(args2, g.writeWhereClause(sb2, inf, internals.Where)...)
	args2 = append(args2, g.writeGroupBy(sb2, inf, internals.GroupBy)...)
	args2 = append(args2, g.writeHaving(sb2, inf, internals.Having)...)

	// Write the joins to the query.
	args = append(args, g.writeJoins(query, inf, internals.Joins)...)

	// Actually write the where and group by clauses to the query.
	query.WriteString(sb2.String())
	args = append(args, args2...)

	for _, union := range internals.Unions {
		union.context = expr.MakeSubqueryContext(ctx)
		var queryObj = g.BuildSelectQuery(
			union.context, union, union.internals,
		)

		query.WriteString(" UNION ")

		if union.internals.Distinct {
			query.WriteString("DISTINCT ")
		}

		query.WriteString(queryObj.SQL())
		args = append(args, queryObj.Args()...)
	}

	if !expr.IsSubqueryContext(ctx) {
		g.writeOrderBy(query, qs.AliasGen, internals.OrderBy)
	}

	args = append(args, g.writeLimitOffset(query, internals.Limit, internals.Offset)...)

	if internals.ForUpdate {
		query.WriteString(" FOR UPDATE")
	}

	return &QueryObject[[][]interface{}]{
		QueryInformation: QueryInformation{
			Stmt:    g.Rebind(ctx, query.String()),
			Object:  inf.Model,
			Params:  args,
			Builder: g,
		},
		Execute: func(sql string, args ...any) ([][]interface{}, error) {

			rows, err := g.DB().QueryContext(ctx, sql, args...)
			if err != nil {
				return nil, errors.Wrap(err, "failed to execute query")
			}

			defer rows.Close()

			if err := rows.Err(); err != nil {
				return nil, errors.Wrap(err, "failed to iterate rows")
			}

			var results = make([][]interface{}, 0, 8)
			var amountCols = 0
			for _, info := range internals.Fields {
				if info.Through != nil {
					amountCols += len(info.Through.Fields)
				}
				amountCols += len(info.Fields)
			}

			for rows.Next() {
				var row = make([]interface{}, amountCols)
				for i := range row {
					row[i] = new(interface{})
				}
				err = rows.Scan(row...)
				if err != nil {
					return nil, errors.Wrap(err, "failed to scan row")
				}

				var result = make([]interface{}, amountCols)
				for i, iface := range row {
					var field = iface.(*interface{})
					result[i] = *field
				}

				results = append(results, result)
			}

			return results, rows.Err()
		},
	}
}

func (g *genericQueryBuilder) BuildCountQuery(
	ctx context.Context,
	qs *GenericQuerySet,
	internals *QuerySetInternals,
) CompiledQuery[int64] {
	var inf = newExpressionInfo(g, qs, internals, false)
	var query = new(strings.Builder)
	var args = make([]any, 0)
	query.WriteString("SELECT COUNT(*) FROM ")
	g.writeTableName(query, qs.AliasGen, internals)

	// First we must resolve all where clauses & group by clauses.
	// These might add joins to the queryset, so this
	// must be done before we write the joins to the query.
	var sb2 = new(strings.Builder)
	var args2 = make([]any, 0, 8)
	args2 = append(args2, g.writeWhereClause(sb2, inf, internals.Where)...)
	args2 = append(args2, g.writeGroupBy(sb2, inf, internals.GroupBy)...)
	args2 = append(args2, g.writeHaving(sb2, inf, internals.Having)...)

	// Write the joins to the query.
	args = append(args, g.writeJoins(query, inf, internals.Joins)...)

	// Actually write the where and group by clauses to the query.
	query.WriteString(sb2.String())
	args = append(args, args2...)

	for _, union := range internals.Unions {
		union.context = expr.MakeSubqueryContext(ctx)
		var queryObj = g.BuildSelectQuery(
			union.context, union, union.internals,
		)

		query.WriteString(" UNION ")

		if union.internals.Distinct {
			query.WriteString("DISTINCT ")
		}

		query.WriteString(queryObj.SQL())
		args = append(args, queryObj.Args()...)
	}

	args = append(args, g.writeLimitOffset(query, internals.Limit, internals.Offset)...)

	return &QueryObject[int64]{
		QueryInformation: QueryInformation{
			Builder: g,
			Stmt:    g.Rebind(ctx, query.String()),
			Object:  inf.Model,
			Params:  args,
		},

		Execute: func(query string, args ...any) (int64, error) {
			var count int64
			var row = g.DB().QueryRowContext(ctx, query, args...)
			if err := row.Scan(&count); err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return 0, nil
				}
				return 0, errors.Wrap(err, "failed to scan row")
			}
			return count, row.Err()
		},
	}
}

func (g *genericQueryBuilder) BuildCreateQuery(
	ctx context.Context,
	qs *GenericQuerySet,
	internals *QuerySetInternals,
	objects []UpdateInfo,
	// e.g. for 2 rows of 3 fields: [[1, 2, 4], [2, 3, 5]] -> [1, 2, 4, 2, 3, 5]
) CompiledQuery[[][]interface{}] {
	var (
		model   = attrs.NewObject[attrs.Definer](qs.Meta().Model())
		query   = new(strings.Builder)
		support = drivers.SupportsReturning(
			g.queryInfo.DB,
		)
	)

	if len(objects) == 0 {
		return ErrorQueryObject[[][]interface{}](model, g.This(), nil)
	}

	var object = objects[0]

	query.WriteString("INSERT INTO ")
	query.WriteString(g.quote)
	query.WriteString(internals.Model.Table)
	query.WriteString(g.quote)
	query.WriteString(" (")

	var primaryIncluded bool
	for i, field := range object.Fields {
		if i > 0 {
			query.WriteString(", ")
		}

		if field.IsPrimary() {
			primaryIncluded = true
		}

		query.WriteString(g.quote)
		query.WriteString(field.ColumnName())
		query.WriteString(g.quote)
	}

	query.WriteString(") VALUES ")

	var written bool
	var values = make([]any, 0, len(objects)*len(object.Fields))
	for _, obj := range objects {
		if written {
			query.WriteString(", ")
		}

		query.WriteString("(")
		for i := range obj.Fields {
			if i > 0 {
				query.WriteString(", ")
			}

			query.WriteString(generic_PLACEHOLDER)
		}
		query.WriteString(")")
		values = append(values, obj.Values...)
		written = true
	}

	switch {
	case support == drivers.SupportsReturningLastInsertId:

		if internals.Model.Primary != nil {
			query.WriteString(" RETURNING ")
			query.WriteString(g.quote)
			query.WriteString(
				internals.Model.Primary.ColumnName(),
			)
			query.WriteString(g.quote)
		}

	case support == drivers.SupportsReturningColumns:
		query.WriteString(" RETURNING ")

		var written = false
		if internals.Model.Primary != nil && !primaryIncluded {
			query.WriteString(g.quote)
			query.WriteString(
				internals.Model.Primary.ColumnName(),
			)
			query.WriteString(g.quote)
			written = true
		}

		for _, field := range object.Fields {
			if written {
				query.WriteString(", ")
			}
			query.WriteString(g.quote)
			query.WriteString(field.ColumnName())
			query.WriteString(g.quote)
			written = true
		}
	case support == drivers.SupportsReturningNone:
		// do nothing

	default:
		panic(fmt.Errorf("returning not supported: %s", support))
	}

	var fieldLen = 0
	if len(objects) > 0 {
		fieldLen = len(objects[0].Fields)
	}
	if internals.Model.Primary != nil && !primaryIncluded {
		fieldLen++
	}

	return &QueryObject[[][]interface{}]{
		QueryInformation: QueryInformation{
			Builder: g,
			Stmt:    g.Rebind(ctx, query.String()),
			Object:  model,
			Params:  values,
		},
		Execute: func(query string, args ...any) ([][]interface{}, error) {
			var err error
			switch support {
			case drivers.SupportsReturningLastInsertId:

				if internals.Model.Primary == nil {
					return nil, nil
				}

				var rows, err = g.DB().QueryContext(ctx, query, args...)
				if err != nil {
					return nil, errors.Wrap(err, "failed to execute query")
				}
				defer rows.Close()

				var result = make([][]interface{}, 0, len(objects))
				for rows.Next() {

					var id = new(interface{})
					err = rows.Scan(id)
					if err != nil {
						return nil, errors.Wrap(err, "failed to scan row")
					}

					if err := rows.Err(); err != nil {
						return nil, errors.Wrap(err, "failed to iterate rows")
					}

					result = append(result, []interface{}{*id})
				}
				return result, rows.Err()

			case drivers.SupportsReturningColumns:

				var results = make([][]interface{}, 0, len(objects))
				var rows, err = g.DB().QueryContext(ctx, query, args...)
				if err != nil {
					return nil, errors.Wrap(err, "failed to execute query")
				}
				defer rows.Close()

				for rows.Next() {
					var result = make([]interface{}, fieldLen)
					for i := range result {
						result[i] = new(interface{})
					}
					err = rows.Scan(result...)
					if err != nil {
						return nil, errors.Wrap(err, "failed to scan row")
					}

					if err := rows.Err(); err != nil {
						return nil, errors.Wrap(err, "failed to iterate rows")
					}

					for i, iface := range result {
						var field = iface.(*interface{})
						result[i] = *field
					}

					results = append(results, result)
				}

				return results, rows.Err()

			case drivers.SupportsReturningNone:
				_, err = g.DB().ExecContext(ctx, query, args...)
				if err != nil {
					return nil, errors.Wrap(err, "failed to execute query")
				}
				return nil, nil

			default:
				panic(fmt.Errorf("returning not supported: %s", support))
			}
		},
	}
}

func (g *genericQueryBuilder) BuildUpdateQuery(
	ctx context.Context,
	qs *GenericQuerySet,
	internals *QuerySetInternals,
	objects []UpdateInfo, // multiple objects can be updated at once
) CompiledQuery[int64] {
	var (
		inf = newExpressionInfo(
			g,
			qs,
			internals,
			true,
		)
		written bool
		args    = make([]any, 0)
		query   = new(strings.Builder)
	)

	for _, info := range objects {
		// Set ForUpdate to true to ensure
		// correct column formatting when writing fields.
		inf.ForUpdate = true

		if written {
			query.WriteString("; ")
		}

		written = true
		query.WriteString("UPDATE ")
		query.WriteString(g.quote)
		query.WriteString(internals.Model.Table)
		query.WriteString(g.quote)
		query.WriteString(" SET ")

		var fieldWritten bool
		var valuesIdx int
		for _, f := range info.Fields {
			if fieldWritten {
				query.WriteString(", ")
			}

			var a, isSQL, ok = info.WriteField(
				query, inf, f, true,
			)

			fieldWritten = ok || fieldWritten
			if !ok {
				continue
			}

			if isSQL {
				args = append(args, a...)
			} else {
				args = append(args, info.Values[valuesIdx])
				valuesIdx++
			}
		}

		// Set ForUpdate to false to avoid
		// incorrect column formatting in joins and where clauses.
		inf.ForUpdate = false

		args = append(
			args,
			g.writeJoins(query, inf, info.Joins)...,
		)

		args = append(
			args,
			g.writeWhereClause(query, inf, info.Where)...,
		)
	}

	return &QueryObject[int64]{
		QueryInformation: QueryInformation{
			Builder: g,
			Stmt:    g.Rebind(ctx, query.String()),
			Object:  inf.Model,
			Params:  args,
		},
		Execute: func(sql string, args ...any) (int64, error) {
			result, err := g.DB().ExecContext(ctx, sql, args...)
			if err != nil {
				return 0, err
			}
			return result.RowsAffected()
		},
	}
}

func (g *genericQueryBuilder) BuildDeleteQuery(
	ctx context.Context,
	qs *GenericQuerySet,
	internals *QuerySetInternals,
) CompiledQuery[int64] {
	var inf = newExpressionInfo(g, qs, internals, false)
	var query = new(strings.Builder)
	var args = make([]any, 0)
	query.WriteString("DELETE FROM ")
	query.WriteString(g.quote)
	query.WriteString(internals.Model.Table)
	query.WriteString(g.quote)

	args = append(
		args,
		g.writeJoins(query, inf, internals.Joins)...,
	)

	args = append(
		args,
		g.writeWhereClause(query, inf, internals.Where)...,
	)

	args = append(
		args,
		g.writeGroupBy(query, inf, internals.GroupBy)...,
	)

	return &QueryObject[int64]{
		QueryInformation: QueryInformation{
			Builder: g,
			Stmt:    g.Rebind(ctx, query.String()),
			Object:  inf.Model,
			Params:  args,
		},
		Execute: func(sql string, args ...any) (int64, error) {
			result, err := g.DB().ExecContext(ctx, sql, args...)
			if err != nil {
				return 0, err
			}
			return result.RowsAffected()
		},
	}
}

func (g *genericQueryBuilder) writeTableName(sb *strings.Builder, aliasGen *alias.Generator, internals *QuerySetInternals) {
	sb.WriteString(g.quote)
	sb.WriteString(internals.Model.Table)
	sb.WriteString(g.quote)

	if aliasGen.Prefix != "" {
		sb.WriteString(" AS ")
		sb.WriteString(g.quote)
		sb.WriteString(aliasGen.Prefix)
		sb.WriteString("_")
		sb.WriteString(internals.Model.Table)
		sb.WriteString(g.quote)
	}
}

func (g *genericQueryBuilder) writeJoins(sb *strings.Builder, inf *expr.ExpressionInfo, joins []JoinDef) []any {
	var args = make([]any, 0)
	var aliasGen = inf.Resolver.Alias()
	for _, join := range joins {
		sb.WriteString(" ")
		sb.WriteString(string(join.TypeJoin))
		sb.WriteString(" ")
		sb.WriteString(g.quote)
		sb.WriteString(join.Table.Name)
		sb.WriteString(g.quote)

		if join.Table.Alias != "" {
			sb.WriteString(" AS ")
			sb.WriteString(g.quote)

			if aliasGen.Prefix != "" {
				sb.WriteString(aliasGen.Prefix)
				sb.WriteString("_")
			}

			sb.WriteString(join.Table.Alias)
			sb.WriteString(g.quote)
		}

		sb.WriteString(" ON ")
		var condition = join.JoinDefCondition
		for condition != nil {

			var col, argsCol = g.FormatColumn(aliasGen, &condition.ConditionA)
			sb.WriteString(col)
			args = append(args, argsCol...)

			sb.WriteString(" ")
			sb.WriteString(string(condition.Operator))
			sb.WriteString(" ")

			col, argsCol = g.FormatColumn(aliasGen, &condition.ConditionB)
			sb.WriteString(col)
			args = append(args, argsCol...)

			if condition.Next != nil {
				sb.WriteString(" AND ")
			}

			condition = condition.Next
		}
	}

	return args
}

func (g *genericQueryBuilder) writeWhereClause(sb *strings.Builder, inf *expr.ExpressionInfo, where []expr.ClauseExpression) []any {

	var args = make([]any, 0)
	if len(where) > 0 {
		sb.WriteString(" WHERE ")
		args = append(
			args, buildWhereClause(sb, inf, where)...,
		)
	}
	return args
}

func (g *genericQueryBuilder) writeGroupBy(sb *strings.Builder, inf *expr.ExpressionInfo, groupBy []*FieldInfo[attrs.FieldDefinition]) []any {

	var infCpy = *inf
	infCpy.SupportsAsExpr = false // Disable AS expressions in WHERE clauses

	var args = make([]any, 0)
	if len(groupBy) > 0 {
		sb.WriteString(" GROUP BY ")
		for i, info := range groupBy {
			if i > 0 {
				sb.WriteString(", ")
			}

			args = append(
				args, info.WriteFields(sb, &infCpy)...,
			)
		}
	}
	return args
}

func (g *genericQueryBuilder) writeHaving(sb *strings.Builder, inf *expr.ExpressionInfo, having []expr.ClauseExpression) []any {
	var args = make([]any, 0)
	if len(having) > 0 {
		sb.WriteString(" HAVING ")
		args = append(
			args, buildWhereClause(sb, inf, having)...,
		)
	}
	return args
}

func (g *genericQueryBuilder) writeOrderBy(sb *strings.Builder, aliasGen *alias.Generator, orderBy []expr.OrderBy) {
	if len(orderBy) > 0 {
		sb.WriteString(" ORDER BY ")

		for i, field := range orderBy {
			if i > 0 {
				sb.WriteString(", ")
			}

			if field.Column.TableOrAlias != "" && field.Column.FieldColumn != nil && field.Column.FieldAlias != "" {
				panic(fmt.Errorf(
					"cannot use table/alias, field column and field alias together in order by: %v",
					field.Column,
				))
			}

			var sql, _ = g.FormatColumn(aliasGen, &field.Column)
			sb.WriteString(sql)

			if field.Desc {
				sb.WriteString(" DESC")
			} else {
				sb.WriteString(" ASC")
			}
		}
	}
}

func (g *genericQueryBuilder) writeLimitOffset(sb *strings.Builder, limit int, offset int) []any {
	var args = make([]any, 0)
	if limit > 0 {
		sb.WriteString(" LIMIT ?")
		args = append(args, limit)
	}

	if offset > 0 {
		sb.WriteString(" OFFSET ?")
		args = append(args, offset)
	}
	return args
}

type postgresQueryBuilder struct {
	*genericQueryBuilder
}

func NewPostgresQueryBuilder(db string) QueryCompiler {
	var inner = NewGenericQueryBuilder(db)
	var pgxCompiler = &postgresQueryBuilder{
		genericQueryBuilder: inner.(*genericQueryBuilder),
	}

	return pgxCompiler
}

// getPostgresType returns the Postgres type for a given Go type and field.
func getPostgresType(rTyp reflect.Type, field attrs.FieldDefinition) string {
	switch rTyp.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return "BIGINT"
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "BIGINT"
	case reflect.Float32, reflect.Float64:
		return "DOUBLE PRECISION"
	case reflect.String, reflect.Slice, reflect.Array:
		return "TEXT"
	case reflect.Bool:
		return "BOOLEAN"
	}

	if rTyp.Implements(reflect.TypeOf((*attrs.Definer)(nil)).Elem()) {
		var newObj = attrs.NewObject[attrs.Definer](rTyp)
		var defs = newObj.FieldDefs()
		var primary = defs.Primary()
		if primary == nil {
			panic(fmt.Errorf(
				"cannot use object without primary key field in update: %T", newObj,
			))
		}
		return getPostgresType(primary.Type(), primary)
	}

	if field == nil {
		panic(fmt.Errorf(
			"cannot determine Postgres type for field type: %s (%s)",
			rTyp.Name(), rTyp.Kind(),
		))
	}

	var fieldType = field.Type()
	return getPostgresType(fieldType, nil)
}

// Postgres requires a special update statement
// to handle the case where multiple rows are updated at once.
func (g *postgresQueryBuilder) BuildUpdateQuery(
	ctx context.Context,
	qs *GenericQuerySet,
	internals *QuerySetInternals,
	objects []UpdateInfo,
) CompiledQuery[int64] {
	if len(objects) == 0 {
		return ErrorQueryObject[int64](qs.Meta().Model(), g, nil)
	}

	var (
		batch = &pgx.Batch{}
		stmts = make([]string, 0, len(objects))
		args  = make([]any, 0, len(objects))
	)

	for _, obj := range objects {

		var query = g.genericQueryBuilder.BuildUpdateQuery(
			ctx, qs, internals, []UpdateInfo{obj},
		)

		var (
			sql = query.SQL()
			arg = query.Args()
		)

		stmts = append(stmts, sql)
		args = append(args, arg)
		batch.Queue(sql, arg...)
	}

	var conner, ok = g.queryInfo.DB.(interface {
		SendBatch(context.Context, *pgx.Batch) pgx.BatchResults
	})
	if !ok {
		panic(fmt.Errorf(
			"cannot execute batch update, DB does not implement `SendBatch(context.Context, *pgx.Batch) pgx.BatchResults`: %T",
			g.DB(),
		))
	}

	return &QueryObject[int64]{
		QueryInformation: QueryInformation{
			Builder: g,
			Stmt:    strings.Join(stmts, "; "),
			Params:  args,
			Object:  qs.Meta().Model(),
		},
		Execute: func(query string, args ...any) (int64, error) {
			var br = conner.SendBatch(ctx, batch)
			defer br.Close()
			var res, err = br.Exec()
			if err != nil {
				return 0, errors.Wrap(err, "failed to execute batch update")
			}
			return res.RowsAffected(), nil
		},
	}
}

type mariaDBQueryBuilder struct {
	*genericQueryBuilder
}

func NewMariaDBQueryBuilder(db string) QueryCompiler {
	var inner = NewGenericQueryBuilder(db)
	return &mariaDBQueryBuilder{
		genericQueryBuilder: inner.(*genericQueryBuilder),
	}
}

func (g *mariaDBQueryBuilder) BuildUpdateQuery(
	ctx context.Context,
	qs *GenericQuerySet,
	internals *QuerySetInternals,
	objects []UpdateInfo,
) CompiledQuery[int64] {
	if len(objects) == 0 {
		return ErrorQueryObject[int64](qs.Meta().Model(), g, nil)
	}

	var query = g.genericQueryBuilder.BuildUpdateQuery(
		ctx, qs, internals, objects,
	)

	return &QueryObject[int64]{
		QueryInformation: QueryInformation{
			Builder: g,
			Stmt:    query.SQL(),
			Params:  query.Args(),
			Object:  qs.Meta().Model(),
		},
		Execute: func(query string, args ...any) (int64, error) {
			var res, err = getDriverResult(g.DB().ExecContext(ctx, query, args...))
			if err != nil {
				return 0, fmt.Errorf(
					"failed to execute query {argLen: %d}: %w",
					len(args), err,
				)
			}

			var mysqlResult, ok = res.(mysql.Result)
			if !ok {
				return res.RowsAffected()
			}

			var affectedRows int64 = 0
			for _, cnt := range mysqlResult.AllRowsAffected() {
				affectedRows += cnt
			}

			return affectedRows, nil
		},
	}
}

var availableForLastInsertId = map[reflect.Kind]struct{}{
	reflect.Int:    {},
	reflect.Int8:   {},
	reflect.Int16:  {},
	reflect.Int32:  {},
	reflect.Int64:  {},
	reflect.Uint:   {},
	reflect.Uint8:  {},
	reflect.Uint16: {},
	reflect.Uint32: {},
	reflect.Uint64: {},
}

type mysqlQueryBuilder struct {
	*mariaDBQueryBuilder
}

func NewMySQLQueryBuilder(db string) QueryCompiler {
	var inner = NewMariaDBQueryBuilder(db)
	return &mysqlQueryBuilder{
		mariaDBQueryBuilder: inner.(*mariaDBQueryBuilder),
	}
}

// mysql does not properly support returning last insert id
// when multiple rows are inserted, so we need to use a different approach.
// This is a workaround to ensure that we can still return the last inserted ID
// when using MySQL, by using a separate query to get the last inserted ID.
func (g *mysqlQueryBuilder) BuildCreateQuery(
	ctx context.Context,
	qs *GenericQuerySet,
	internals *QuerySetInternals,
	objects []UpdateInfo,
) CompiledQuery[[][]interface{}] {

	if len(objects) == 0 {
		return ErrorQueryObject[[][]interface{}](qs.Meta().Model(), g, nil)
	}

	var (
		values = make([]any, 0, len(objects)*len(objects[0].Fields))
		stmt   = make([]string, 0, len(objects))
	)
	for _, object := range objects {

		if len(object.Fields) != len(object.Values) {
			return ErrorQueryObject[[][]interface{}](
				qs.Meta().Model(), g, errors.TypeMismatch.WithCause(fmt.Errorf(
					"cannot build create query, number of fields (%d) does not match number of values (%d)",
					len(object.Fields), len(object.Values),
				)),
			)
		}

		var query = new(strings.Builder)

		query.WriteString("INSERT INTO ")
		query.WriteString(g.quote)
		query.WriteString(internals.Model.Table)
		query.WriteString(g.quote)
		query.WriteString(" (")

		for i, field := range object.Fields {
			if i > 0 {
				query.WriteString(", ")
			}

			query.WriteString(g.quote)
			query.WriteString(field.ColumnName())
			query.WriteString(g.quote)
		}

		query.WriteString(") VALUES (")
		for i := range object.Fields {
			if i > 0 {
				query.WriteString(", ")
			}

			query.WriteString(generic_PLACEHOLDER)
		}
		query.WriteString(")")
		values = append(values, object.Values...)

		stmt = append(stmt, query.String())
	}

	return &QueryObject[[][]interface{}]{
		QueryInformation: QueryInformation{
			Builder: g,
			Stmt:    g.Rebind(ctx, strings.Join(stmt, "; ")),
			Params:  values,
			Object:  attrs.NewObject[attrs.Definer](qs.Meta().Model()),
		},
		Execute: func(query string, args ...any) ([][]interface{}, error) {
			if internals.Model.Primary != nil {
				return g.querySupportsLastInsertId(ctx, internals, objects, query, args)
			}

			// No need to return last insert ID if there is no primary key.
			var _, err = g.queryInfo.DB.ExecContext(ctx, query, args...)
			if err != nil {
				return nil, fmt.Errorf(
					"failed to execute query: %w", err,
				)
			}

			return [][]interface{}{}, nil
		},
	}
}

func (g *mysqlQueryBuilder) querySupportsLastInsertId(
	ctx context.Context,
	internals *QuerySetInternals,
	objects []UpdateInfo,
	query string,
	args []any,
) ([][]interface{}, error) {
	res, err := getDriverResult(g.DB().ExecContext(ctx, query, args...))
	if err != nil {
		return nil, fmt.Errorf(
			"failed to execute query {argLen: %d}: %w",
			len(args), err,
		)
	}

	var allIds = res.(mysql.Result).AllLastInsertIds()
	var result = make([][]interface{}, len(allIds))
	if len(allIds) != len(objects) {
		var idList string
		if len(allIds) > 0 && len(allIds) < MAX_GET_RESULTS {
			idList = fmt.Sprintf(" (%v)", allIds)
		}

		//return nil, fmt.Errorf(
		//	"expected %d last insert ids, got %d%s: %w",
		//	len(objects), len(allIds), idList, errors.ErrLastInsertId)
		return nil, errors.LastInsertId.WithCause(fmt.Errorf(
			"expected %d last insert ids, got %d%s",
			len(objects), len(allIds), idList,
		))
	}

	if _, ok := availableForLastInsertId[internals.Model.Primary.Type().Kind()]; ok {
		for i, id := range allIds {
			result[i] = []interface{}{id}
		}
	}
	return result, nil
}

var (
	_sqlDriverResultType = reflect.TypeOf((*driver.Result)(nil)).Elem()
)

func getDriverResult(res driver.Result, err error) (driver.Result, error) {
	if err != nil {
		return res, err
	}

	var rVal = reflect.ValueOf(&res)
	var rStruct = rVal.Elem().Elem() // ptr -> iface -> struct
	if rStruct.Kind() != reflect.Ptr {
		var rType = rStruct.Type()
		var rNew = reflect.New(rType)
		rNew.Elem().Set(rStruct)
		rStruct = rNew.Elem()
	}
	for i := 0; i < rStruct.NumField(); i++ {
		if rStruct.Type().Field(i).Type.Implements(_sqlDriverResultType) {
			var field = rStruct.Field(i)
			var val = reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem().Interface()
			return val.(driver.Result), nil
		}
	}

	panic(fmt.Errorf(
		"failed to find driver.Result field in driver.Result type: %s",
		rStruct.Type().Name(),
	))
}

// -----------------------------------------------------------------------------
// Helpers
// -----------------------------------------------------------------------------

func buildWhereClause(b *strings.Builder, inf *expr.ExpressionInfo, exprs []expr.ClauseExpression) []any {
	var args = make([]any, 0)
	for i, e := range exprs {
		e := e.Resolve(inf)
		var a = e.SQL(b)
		if i < len(exprs)-1 {
			b.WriteString(" AND ")
		}
		args = append(args, a...)
	}
	return args
}
