package expr_test

import (
	"context"
	"strings"

	"github.com/Nigel2392/go-django/djester/testdb"
	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/expr"
	"github.com/Nigel2392/go-django/queries/src/models"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/core/attrs"
)

func init() {
	_, db := testdb.Open()
	var settings = map[string]interface{}{
		django.APPVAR_DATABASE: db,
	}
	django.App(django.Configure(settings))
	attrs.RegisterModel(&TestModel{})
	attrs.RegisterModel(&OtherTestModel{})
}

type OtherTestModel struct {
	models.Model `table:"other_test_model"`
	ID           int
	Name         string
	TestModel    *TestModel
}

func (m *OtherTestModel) FieldDefs(ctx context.Context) attrs.Definitions {
	return m.Model.Define(ctx, m,
		attrs.NewField(m, "ID", &attrs.FieldConfig{Primary: true}),
		attrs.NewField(m, "Name", &attrs.FieldConfig{}),
		attrs.NewField(m, "TestModel", &attrs.FieldConfig{
			RelForeignKey: attrs.Relate(&TestModel{}, "", nil),
		}),
	)
}

type TestModel struct {
	models.Model `table:"test_model"`
	ID           int
	Name         string
	Age          int
	Score        int
	CreatedAt    string
	FirstName    string
	LastName     string
	Nickname     string
}

func (m *TestModel) FieldDefs(ctx context.Context) attrs.Definitions {
	return m.Model.Define(ctx, m,
		attrs.NewField(m, "ID", &attrs.FieldConfig{Primary: true}),
		attrs.NewField(m, "Name", &attrs.FieldConfig{}),
		attrs.NewField(m, "Age", &attrs.FieldConfig{}),
		attrs.NewField(m, "Score", &attrs.FieldConfig{}),
		attrs.NewField(m, "CreatedAt", &attrs.FieldConfig{}),
		attrs.NewField(m, "FirstName", &attrs.FieldConfig{}),
		attrs.NewField(m, "LastName", &attrs.FieldConfig{}),
		attrs.NewField(m, "Nickname", &attrs.FieldConfig{}),
	)
}

func getTestInfo() *expr.ExpressionInfo {
	qs := queries.GetQuerySet(&TestModel{})
	var info *expr.ExpressionInfo
	qs.Scope(func(q *queries.QuerySet[*TestModel], internals *queries.QuerySetInternals) *queries.QuerySet[*TestModel] {
		info = q.Compiler().ExpressionInfo(q)
		return q
	})
	return info
}

// fixSQL replaces backticks with the compiler's quote character and ? with its placeholder.
func fixSQL(info *expr.ExpressionInfo, sqlStr string) string {
	var replacer = strings.NewReplacer(
		"`test_model`", info.QuoteIdentifier("test_model"),
		"`T_test_model`", info.QuoteIdentifier("T_test_model"),
		"`T1_test_model`", info.QuoteIdentifier("T1_test_model"),
		"`other_test_model`", info.QuoteIdentifier("other_test_model"),
		"`T_other_test_model`", info.QuoteIdentifier("T_other_test_model"),
		"`T1_other_test_model`", info.QuoteIdentifier("T1_other_test_model"),
		"`t1`", info.QuoteIdentifier("t1"),
		"`id`", info.QuoteIdentifier("id"),
		"`age`", info.QuoteIdentifier("age"),
		"`name`", info.QuoteIdentifier("name"),
		"`score`", info.QuoteIdentifier("score"),
		"`created_at`", info.QuoteIdentifier("created_at"),
		"`first_name`", info.QuoteIdentifier("first_name"),
		"`last_name`", info.QuoteIdentifier("last_name"),
		"`nickname`", info.QuoteIdentifier("nickname"),
		"`alias`", info.QuoteIdentifier("alias"),
		"?", info.Placeholder,
	)
	return replacer.Replace(sqlStr)
}
