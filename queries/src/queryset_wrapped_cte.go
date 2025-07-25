package queries

import (
	"context"
	"strings"

	"github.com/Nigel2392/go-django/queries/src/expr"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/elliotchance/orderedmap/v2"
)

type JoinOption[T attrs.Definer] func(*CTEQuerySet[T], *JoinDef)

func JoinOptionTargetField[T attrs.Definer](source, target string) JoinOption[T] {
	return func(cte *CTEQuerySet[T], join *JoinDef) {
		var base = cte.Base()
		var res, err = base.WalkField(
			target, OptFlags(WalkFlagAddJoins),
		)
		if err != nil {
			panic(err)
		}

		for _, join := range res.Joins {
			base.internals.AddJoin(join)
		}

		var lhsField attrs.FieldDefinition = res.Annotation
		if res.Chain != nil {
			lhsField = res.Chain.Final.Field
		}

		var (
			lhsAlias = res.Aliases[len(res.Aliases)-1]
			rhsAlias = join.Table.Name
		)

		join.JoinDefCondition = &JoinDefCondition{
			ConditionA: expr.TableColumn{
				TableOrAlias: lhsAlias,
				FieldColumn:  lhsField,
			},
			ConditionB: expr.TableColumn{
				TableOrAlias: rhsAlias,
				FieldColumn:  base.internals.Model.Primary,
			},
			Operator: expr.EQ,
		}
	}
}

func JoinOptionCondition[T attrs.Definer](condition *JoinDefCondition) JoinOption[T] {
	return func(cte *CTEQuerySet[T], join *JoinDef) {
		if condition == nil {
			panic("JoinOptionCondition cannot be nil")
		}
		join.JoinDefCondition = condition
	}
}

func JoinOptionJoinType[T attrs.Definer](joinType expr.JoinType) JoinOption[T] {
	return func(cte *CTEQuerySet[T], join *JoinDef) {
		join.TypeJoin = joinType
	}
}

type CTEName = string

type CTE[T attrs.Definer] struct {
	Name      CTEName
	QuerySets []*QuerySet[T]
}

type cteQuerySetInternals[T attrs.Definer] struct {
	ctes *orderedmap.OrderedMap[CTEName, *CTE[T]]
}

type CTEQueryCompiler struct {
	QueryCompiler
}

// BuildSelectQuery builds a select query with the given parameters.
func (c *CTEQueryCompiler) BuildSelectQuery(
	ctx context.Context,
	qs *CTEQuerySet[attrs.Definer],
	internals *QuerySetInternals,
) CompiledQuery[[][]interface{}] {

	if qs.internals.ctes == nil {
		return c.QueryCompiler.BuildSelectQuery(ctx, qs.Base(), internals)
	}

	var (
		exprInfo = c.QueryCompiler.ExpressionInfo(qs.Base(), internals)
		sb       = &strings.Builder{}
		args     = make([]any, 0, 10)
	)

	for head := qs.internals.ctes.Front(); head != nil; head = head.Next() {
		var cte = head.Value
		if len(cte.QuerySets) == 0 {
			continue // Skip empty CTEs
		}

		sb.WriteString("WITH ")
		sb.WriteString(exprInfo.QuoteIdentifier(cte.Name))
		sb.WriteString(" AS (")

		for i, qs := range cte.QuerySets {
			if i > 0 {
				sb.WriteString(" UNION ALL ")
			}

			qs.context = expr.MakeSubqueryContext(ctx)
			var query = qs.QueryAll()

			sb.WriteString(query.SQL())
			args = append(args, query.Args()...)
		}

		sb.WriteString(") ")
	}

	var query = c.QueryCompiler.BuildSelectQuery(ctx, qs.Base(), internals)
	sb.WriteString(query.SQL())
	args = append(args, query.Args()...)

	return &QueryObject[[][]interface{}]{
		QueryInfo: &QueryInformation{
			Stmt:    sb.String(),
			Params:  args,
			Object:  qs.Base().internals.Model.Object,
			Builder: c.QueryCompiler,
		},
		Execute: query.(*QueryObject[[][]any]).Execute,
	}
}

type CTEQuerySet[T attrs.Definer] struct {
	*WrappedQuerySet[T, *CTEQuerySet[T], *QuerySet[T]]
	compiler  *CTEQueryCompiler
	internals cteQuerySetInternals[T]
}

func NewCTEQuerySet[T attrs.Definer](base *QuerySet[T]) *CTEQuerySet[T] {
	var qs = &CTEQuerySet[T]{}
	qs.WrappedQuerySet = WrapQuerySet(base, qs)
	qs.compiler = &CTEQueryCompiler{
		QueryCompiler: base.compiler,
	}
	return qs
}

func (qs *CTEQuerySet[T]) CloneQuerySet(wrapped *WrappedQuerySet[T, *CTEQuerySet[T], *QuerySet[T]]) *CTEQuerySet[T] {
	return &CTEQuerySet[T]{
		WrappedQuerySet: wrapped,
	}
}

func (c *CTEQuerySet[T]) Join(name CTEName, options ...JoinOption[T]) *CTEQuerySet[T] {
	var base = c.Base()

	joinDef := JoinDef{
		Table: Table{
			Name: name,
		},
	}

	for _, opt := range options {
		opt(c, &joinDef)
	}

	if joinDef.TypeJoin == "" {
		joinDef.TypeJoin = expr.TypeJoinInner
	}

	if joinDef.JoinDefCondition == nil {
		// Fall back to default join condition
		// This assumes that the CTE has a primary key that matches the base model's primary key.
		// This is a common pattern in CTEs, but may not always be the case.
		joinDef.JoinDefCondition = &JoinDefCondition{
			ConditionA: expr.TableColumn{
				TableOrAlias: base.internals.Model.Table,
				FieldColumn:  base.internals.Model.Primary,
			},
			ConditionB: expr.TableColumn{
				TableOrAlias: name,
				FieldColumn:  base.internals.Model.Primary,
			},
			Operator: expr.EQ,
		}
	}

	base.internals.AddJoin(joinDef)

	return c
}
