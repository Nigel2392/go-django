package resolver

import (
	"context"
	"fmt"
	"io"
	"maps"
	"strings"

	"github.com/Nigel2392/go-django/queries/src/alias"
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/queries/src/expr"
	"github.com/Nigel2392/go-django/queries/src/expr/builder"
	"github.com/Nigel2392/go-django/src/core/attrs"
)

var _ expr.FieldResolver = (*Resolver)(nil)

type Resolver struct {
	Model ModelInfo

	aliassedExprs map[string]expr.Expression
	context       context.Context
	aliasGen      *alias.Generator
	compiler      ExpressionCompiler
	exprInfo      *expr.ExpressionInfo
}

func New(ctx context.Context, model attrs.Definer, compiler ExpressionCompiler) *Resolver {
	var (
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
	if ord, ok := model.(OrderByDefiner); ok {
		orderBy = ord.OrderBy()
	}

	if len(orderBy) == 0 && primary != nil {
		orderBy = []string{primary.Name()}
	}

	var info = ModelInfo{
		Primary:  primary,
		Object:   model,
		Fields:   definitions.Fields(),
		Table:    tableName,
		Ordering: orderBy,
	}

	resolver := new(Resolver)
	resolver.Model = info
	resolver.context = ctx
	resolver.compiler = compiler
	resolver.exprInfo = compiler.ExpressionInfo(resolver)
	resolver.aliassedExprs = make(map[string]expr.Expression)
	return resolver
}

func (rs *Resolver) Meta() expr.ModelMeta {
	return rs.Model
}

func (rs *Resolver) Context() context.Context {
	return rs.context
}

func (rs *Resolver) Clone() *Resolver {
	if rs.aliasGen == nil {
		rs.aliasGen = alias.NewGenerator()
	}

	var inf = *rs.ExpressionInfo()
	return &Resolver{
		Model:         rs.Model,
		aliassedExprs: maps.Clone(rs.aliassedExprs),
		aliasGen:      rs.aliasGen.Clone(),
		context:       rs.context,
		compiler:      rs.compiler,
		exprInfo:      &inf,
	}
}

func (rs *Resolver) ExpressionInfo() *expr.ExpressionInfo {
	if rs.exprInfo == nil {
		rs.exprInfo = rs.compiler.ExpressionInfo(rs)
	}
	return rs.exprInfo
}

func (rs *Resolver) ResolverInfoForModel(model attrs.Definer) *expr.ExpressionInfo {
	other := New(rs.context, model, rs.compiler)
	other.aliasGen = rs.Alias().Clone()
	return other.compiler.ExpressionInfo(other)
}

func (rs *Resolver) Alias() *alias.Generator {
	if rs.aliasGen == nil {
		rs.aliasGen = alias.NewGenerator()
	}
	return rs.aliasGen
}

// Resolve resolves a fieldName together with the expr.ExpressionInfo to an information object
// that can be used to suitably write to the specified database backend.
//
// It returns the final model in the fieldName chain
// The final selected field in the fieldName chain
// The previously mentioned information object
// An error if it occurs during [attrs.WalkRelationChain]
func (rs *Resolver) Resolve(fieldName string, inf *expr.ExpressionInfo) (attrs.Definer, attrs.FieldDefinition, *expr.TableColumn, error) {

	if rs.aliasGen == nil {
		rs.aliasGen = alias.NewGenerator()
	}

	fieldPath := strings.Split(fieldName, ".")
	chain, err := attrs.WalkRelationChain(
		rs.Model.Object, false, fieldPath,
	)
	if err != nil {
		return nil, nil, nil, err
	}

	var (
		table   string
		partIdx int
		curr    = chain.Root
	)
	for curr != nil {
		var (
			meta = attrs.GetModelMeta(curr.Model)
			defs = meta.Definitions()
		)

		table = rs.aliasGen.GetTableAlias(
			defs.TableName(), chain.Chain[:partIdx],
		)

		curr = curr.Next
		partIdx++
	}

	var col = &expr.TableColumn{
		TableOrAlias: table,
		FieldColumn:  chain.Final.Field,
	}

	return chain.Final.Model, chain.Final.Field, col, nil
}

// ResolveExpression resolves an expression with the internally stored [expr.ExpressionInfo].
func (rs *Resolver) ResolveExpression(e expr.Expression) expr.Expression {
	if rs.exprInfo == nil {
		rs.exprInfo = rs.compiler.ExpressionInfo(rs)
	}

	return e.Resolve(rs.exprInfo)
}

// ExpressionSQL resolves the expression and writes its' SQL output to `w`.
//
// The only returned errors are those from `w.Write`.
func (rs *Resolver) ExpressionSQL(w io.Writer, e expr.Expression) error {
	if rs.exprInfo == nil {
		rs.exprInfo = rs.compiler.ExpressionInfo(rs)
	}

	sb, ok := w.(*builder.BaseBuilder)
	if !ok {
		sb = new(builder.BaseBuilder)
	}

	e = e.Resolve(rs.exprInfo)
	e.SQL(sb)

	if !ok {
		_, err := w.Write([]byte(sb.String()))
		return err
	}

	return nil
}
