package expr

import (
	"context"
	"fmt"
	"strings"
)

type subqueryContextKey struct{}

type parentQueryContextKey struct{}

func IsSubqueryContext(ctx context.Context) bool {
	var v, ok = ctx.Value(subqueryContextKey{}).(bool)
	if !ok {
		return false
	}
	return v
}

func MakeSubqueryContext(ctx context.Context) context.Context {
	if IsSubqueryContext(ctx) {
		return ctx
	}
	return context.WithValue(ctx, subqueryContextKey{}, true)
}

func ParentFromSubqueryContext(ctx context.Context) (*ExpressionInfo, bool) {
	var v, ok = ctx.Value(parentQueryContextKey{}).(*ExpressionInfo)
	return v, ok
}

func AddParentSubqueryContext(ctx context.Context, inf *ExpressionInfo) context.Context {
	return context.WithValue(ctx, parentQueryContextKey{}, inf)
}

type outerRef struct {
	fieldName string
	field     *ResolvedField
	used      bool
}

func OuterRef(fld string) NamedExpression {
	return &outerRef{fieldName: fld}
}

func (e *outerRef) FieldName() string {
	return e.fieldName
}

func (e *outerRef) SQL(sb *strings.Builder) []any {
	sb.WriteString(e.field.SQLText)
	return e.field.SQLArgs
}

func (e *outerRef) Clone() Expression {
	return &outerRef{fieldName: e.fieldName, field: e.field, used: e.used}
}

func (e *outerRef) Resolve(inf *ExpressionInfo) Expression {
	if inf.Model == nil || e.used {
		return e
	}

	var nE = e.Clone().(*outerRef)
	nE.used = true

	// A field can never be lowercase, so if the first part is lowercase,
	// we assume it's an alias and the rest is the field name.
	var alias string
	var firstDot = strings.Index(nE.fieldName, ".")
	if firstDot != -1 && strings.ToLower(nE.fieldName[:firstDot]) == nE.fieldName[:firstDot] {
		alias = nE.fieldName[:firstDot]
		nE.fieldName = nE.fieldName[firstDot+1:]
	}

	var outer, ok = ParentFromSubqueryContext(inf.Resolver.Context())
	if !ok {
		panic(fmt.Errorf("failed to resolve outer reference %s: no parent subquery context found", nE.fieldName))
	}

	var _, field, col, err = outer.Resolver.Resolve(nE.fieldName, outer)
	if err != nil {
		panic(fmt.Errorf("failed to resolve field %s: %w", field, err))
	}

	if alias != "" {
		col.TableOrAlias = alias
	}

	var sql, args = outer.FormatField(outer.Resolver.Alias(), col)
	nE.field = newResolvedField(nE.fieldName, sql, field, args)

	return nE
}
