package resolver_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/Nigel2392/go-django/djester/testdb"
	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/expr"
	"github.com/Nigel2392/go-django/queries/src/models"
	"github.com/Nigel2392/go-django/queries/src/resolver"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/core/attrs"
)

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

func getTestResolver() *resolver.Resolver {
	var compiler = queries.Compiler(django.APPVAR_DATABASE)
	return resolver.New(
		context.Background(),
		&TestModel{},
		compiler,
	)
}

// fixSQL replaces backticks with the compiler's quote character and ? with its placeholder.
func fixSQL(resolver *resolver.Resolver, sqlStr string) string {
	var info = resolver.ExpressionInfo()
	var replacer = strings.NewReplacer(
		"`test_model`", info.QuoteIdentifier("test_model"),
		"`T_test_model`", info.QuoteIdentifier("T_test_model"),
		"`T1_test_model`", info.QuoteIdentifier("T1_test_model"),
		"`other_test_model`", info.QuoteIdentifier("other_test_model"),
		"`T_other_test_model`", info.QuoteIdentifier("T_other_test_model"),
		"`T1_other_test_model`", info.QuoteIdentifier("T1_other_test_model"),
		"`t`", info.QuoteIdentifier("t"),
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

func TestMain(m *testing.M) {

	_, db := testdb.Open()
	var settings = map[string]interface{}{
		django.APPVAR_DATABASE: db,
	}
	django.App(django.Configure(settings))
	attrs.RegisterModel(&TestModel{})
	attrs.RegisterModel(&OtherTestModel{})

	code := m.Run()
	os.Exit(code)
}

func TestFuncUpperSQL(t *testing.T) {
	var info = getTestResolver()
	var sb strings.Builder
	info.ExpressionSQL(&sb, expr.UPPER("Name"))
	if sql := sb.String(); sql != fixSQL(info, "UPPER(`test_model`.`name`)") {
		t.Errorf("Unexpected UPPER SQL: %s", sql)
	}
}

// SQL Generation 2
func TestFuncLowerSQL(t *testing.T) {
	var info = getTestResolver()
	var sb strings.Builder
	info.ExpressionSQL(&sb, expr.LOWER("Name"))
	if sql := sb.String(); sql != fixSQL(info, "LOWER(`test_model`.`name`)") {
		t.Errorf("Unexpected LOWER SQL: %s", sql)
	}
}

// Happy Path 1
func TestFuncLengthResolve(t *testing.T) {
	var info = getTestResolver()
	var sb strings.Builder
	info.ExpressionSQL(&sb, expr.LENGTH("Name"))
	if sql := sb.String(); sql != fixSQL(info, "LENGTH(`test_model`.`name`)") {
		t.Errorf("Unexpected LENGTH SQL: %s", sql)
	}
}

func TestRawExprSQL(t *testing.T) {
	var info = getTestResolver()
	var sb strings.Builder
	info.ExpressionSQL(&sb, expr.Raw("![Age] = ?", 18))
	if !strings.Contains(sb.String(), fixSQL(info, "`test_model`.`age` = ?")) {
		t.Errorf("Unexpected Raw SQL generation: %s", sb.String())
	}
}

// SQL Generation 2
func TestFExprSQL(t *testing.T) {
	info := getTestResolver()
	resolved := info.ResolveExpression(expr.F("![Age] + ?[1] + ![Score]", 3))
	var sb strings.Builder
	args := resolved.SQL(&sb)
	sql := sb.String()
	if !strings.Contains(sql, fixSQL(info, "`test_model`.`age` + ? + `test_model`.`score`")) {
		t.Errorf("Unexpected F SQL generation: %s", sql)
	}
	if len(args) != 1 || args[0] != 3 {
		t.Errorf("Expected arg 3, got %v", args)
	}
}

// SQL Generation 3
func TestModelTableExprSQL(t *testing.T) {
	info := getTestResolver()
	f := expr.F("#[resolver_test.OtherTestModel.Name] + ?[1] + ![Score]", 3)
	resolved := f.Resolve(info.ExpressionInfo())
	var sb strings.Builder
	args := resolved.SQL(&sb)
	sql := sb.String()
	if !strings.Contains(sql, fixSQL(info, "`other_test_model`.`name` + ? + `test_model`.`score`")) {
		t.Errorf("Unexpected F SQL generation: %s", sql)
	}
	if len(args) != 1 || args[0] != 3 {
		t.Errorf("Expected arg 3, got %v", args)
	}
}

// SQL Generation 4
func TestModelTableAliasExprSQL(t *testing.T) {
	info := getTestResolver()
	f := expr.F("#[t1.resolver_test.OtherTestModel.Name] + ?[1] + ![Score]", 3)
	resolved := f.Resolve(info.ExpressionInfo())
	var sb strings.Builder
	args := resolved.SQL(&sb)
	sql := sb.String()
	if !strings.Contains(sql, fixSQL(info, "`t1`.`name` + ? + `test_model`.`score`")) {
		t.Errorf("Unexpected F SQL generation: %s", sql)
	}
	if len(args) != 1 || args[0] != 3 {
		t.Errorf("Expected arg 3, got %v", args)
	}
}

// SQL Generation 5
func TestModelTableOtherExprSQL(t *testing.T) {
	info := getTestResolver()
	f := expr.F("#[resolver_test.OtherTestModel.TestModel.Name] + ?[1] + ![Score]", 3)
	resolved := f.Resolve(info.ExpressionInfo())
	var sb strings.Builder
	args := resolved.SQL(&sb)
	sql := sb.String()
	if !strings.Contains(sql, fixSQL(info, "`T_test_model`.`name` + ? + `test_model`.`score`")) {
		t.Errorf("Unexpected F SQL generation: %s", sql)
	}
	if len(args) != 1 || args[0] != 3 {
		t.Errorf("Expected arg 3, got %v", args)
	}
}

// SQL Generation 6
func TestModelTableOtherBackrefExprSQL(t *testing.T) {
	info := getTestResolver()
	f := expr.F("#[resolver_test.OtherTestModel.TestModel.OtherTestModelSet.Name] #[t1.resolver_test.OtherTestModel.TestModel.OtherTestModelSet.Name] #[t1.resolver_test.OtherTestModel.Name] #[t1.resolver_test.OtherTestModel.Name] #[t1.resolver_test.OtherTestModel.Name] #[t1.resolver_test.OtherTestModel.Name] TABLE(resolver_test.OtherTestModel) + ?[1] + ![Score]", 3)
	resolved := f.Resolve(info.ExpressionInfo())
	var sb strings.Builder
	args := resolved.SQL(&sb)
	sql := sb.String()
	if !strings.Contains(sql, fixSQL(info, "`T1_other_test_model`.`name` `t1`.`name` `t1`.`name` `t1`.`name` `t1`.`name` `t1`.`name` `other_test_model` + ? + `test_model`.`score`")) {
		t.Errorf("Unexpected F SQL generation: %s", sql)
	}
	if len(args) != 1 || args[0] != 3 {
		t.Errorf("Expected arg 3, got %v", args)
	}
}

func TestTableOtherExprSQL(t *testing.T) {
	info := getTestResolver()
	f := expr.F("TABLE(resolver_test.OtherTestModel AS t) + #[t.resolver_test.OtherTestModel.ID] + ?[1] + ![Score]", 3)
	resolved := f.Resolve(info.ExpressionInfo())
	var sb strings.Builder
	args := resolved.SQL(&sb)
	sql := sb.String()
	if expected := fixSQL(info, "`other_test_model` AS `t` + `t`.`id` + ? + `test_model`.`score`"); !strings.Contains(sql, expected) {
		t.Errorf("Unexpected F SQL generation: %s \\!=\\ %s", sql, expected)
	}
	if len(args) != 1 || args[0] != 3 {
		t.Errorf("Expected arg 3, got %v", args)
	}
}

func TestTableOtherBackrefExprSQL(t *testing.T) {
	info := getTestResolver()
	f := expr.F("![OtherTestModelSet.ID] == + ![t.OtherTestModelSet.ID] TABLE(OtherTestModelSet.TestModel aS t) + ![OtherTestModelSet.ID] + ?[1] + ![Score]", 3)
	resolved := f.Resolve(info.ExpressionInfo())
	var sb strings.Builder
	args := resolved.SQL(&sb)
	sql := sb.String()
	if expected := fixSQL(info, "`T_other_test_model`.`id` == + `t`.`id` `test_model` AS `t` + `T_other_test_model`.`id` + ? + `test_model`.`score`"); !strings.Contains(sql, expected) {
		t.Errorf("Unexpected F SQL generation: \n\t%s\n\t!=\n\t%s", sql, expected)
	}
	if len(args) != 1 || args[0] != 3 {
		t.Errorf("Expected arg 3, got %v", args)
	}
}

func TestQuotesExprSQL(t *testing.T) {
	info := getTestResolver()
	f := expr.F("'test_model.score' == ![Score] + ?", 3)
	resolved := f.Resolve(info.ExpressionInfo())
	var sb strings.Builder
	args := resolved.SQL(&sb)
	sql := sb.String()
	if !strings.Contains(sql, fixSQL(info, "`test_model`.`score` == `test_model`.`score` + ?")) {
		t.Errorf("Unexpected F SQL generation: %s", sql)
	}
	if len(args) != 1 || args[0] != 3 {
		t.Errorf("Expected arg 3, got %v", args)
	}
}

// Happy Path 1
func TestFExprMultipleArgs(t *testing.T) {
	info := getTestResolver()
	f := expr.F("![Age] > ? AND ![Score] < ?", 18, 100)
	resolved := f.Resolve(info.ExpressionInfo())
	if resolved == nil {
		t.Fatalf("Failed to resolve FExpr")
	}
}

func TestRawExprNot(t *testing.T) {
	info := getTestResolver()
	q := expr.Q("Age", 18)
	notR := q.Not(true)
	resolved := notR.Resolve(info.ExpressionInfo())
	var sb strings.Builder
	resolved.SQL(&sb)
	if !strings.Contains(sb.String(), fixSQL(info, "NOT (`test_model`.`age` = ?)")) {
		t.Errorf("Unexpected NOT Raw SQL: %s", sb.String())
	}
}

// Unhappy Path 1
func TestFExprResolveInvalidField(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected panic when resolving FExpr with invalid field")
		}
	}()
	info := getTestResolver()
	f := expr.F("![InvalidField] = ?", 1)
	f.Resolve(info.ExpressionInfo())
}

func BenchmarkResolver(b *testing.B) {
	resolverStandalone := getTestResolver()
	infoStandalone := resolverStandalone.ExpressionInfo()
	resolverQueries := queries.Objects(&TestModel{})
	infoQueries := resolverQueries.Compiler().ExpressionInfo(resolverQueries)

	b.Run("standalone", func(b *testing.B) {
		b.Run("flat", func(b *testing.B) {
			for b.Loop() {
				_, _, _, err := resolverStandalone.Resolve("ID", infoStandalone)
				if err != nil {
					b.Fatalf("error while running benchmark for resolve: %v", err)
				}
			}
		})
		b.Run("related_depth_1", func(b *testing.B) {
			for b.Loop() {
				_, _, _, err := resolverStandalone.Resolve("OtherTestModelSet.ID", infoStandalone)
				if err != nil {
					b.Fatalf("error while running benchmark for resolve: %v", err)
				}
			}
		})
		b.Run("related_depth_5", func(b *testing.B) {
			for b.Loop() {
				_, _, _, err := resolverStandalone.Resolve("OtherTestModelSet.TestModel.OtherTestModelSet.TestModel.OtherTestModelSet.ID", infoStandalone)
				if err != nil {
					b.Fatalf("error while running benchmark for resolve: %v", err)
				}
			}
		})
	})

	b.Run("queries", func(b *testing.B) {
		b.Run("flat", func(b *testing.B) {
			for b.Loop() {
				_, _, _, err := resolverQueries.Resolve("ID", infoQueries)
				if err != nil {
					b.Fatalf("error while running benchmark for resolve: %v", err)
				}
			}
		})
		b.Run("related_depth_1", func(b *testing.B) {
			for b.Loop() {
				_, _, _, err := resolverQueries.Resolve("OtherTestModelSet.ID", infoQueries)
				if err != nil {
					b.Fatalf("error while running benchmark for resolve: %v", err)
				}
			}
		})
		b.Run("related_depth_5", func(b *testing.B) {
			for b.Loop() {
				_, _, _, err := resolverQueries.Resolve("OtherTestModelSet.TestModel.OtherTestModelSet.TestModel.OtherTestModelSet.ID", infoQueries)
				if err != nil {
					b.Fatalf("error while running benchmark for resolve: %v", err)
				}
			}
		})
	})

}
