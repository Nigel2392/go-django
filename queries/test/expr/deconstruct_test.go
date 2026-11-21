package expr_test

import (
	"context"
	"encoding/json/v2"
	"reflect"
	"strings"
	"testing"

	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/expr"
	"github.com/Nigel2392/go-django/queries/src/expr/builder"
	"github.com/Nigel2392/go-django/queries/src/expr/constraints"
	"github.com/Nigel2392/go-django/src/core/deconstruct"
)

func renderSQL(t *testing.T, info *expr.ExpressionInfo, e expr.Expression) (string, []any) {
	t.Helper()
	resolved := e.Resolve(info)
	var sb builder.BaseBuilder
	resolved.SQL(&sb)
	if err := sb.GetError(); err != nil {
		t.Fatalf("SQL builder error: %v", err)
	}
	return sb.String(), sb.Vars
}

func roundTrip[T any](t *testing.T, v any) T {
	t.Helper()
	data, err := json.Marshal(v, deconstruct.Constructors.JSONOptions())
	if err != nil {
		t.Fatalf("json.Marshal(%T) failed: %v", v, err)
	}
	var out T
	if err := deconstruct.Constructors.Unmarshal(data, &out); err != nil {
		t.Fatalf("Constructors.Unmarshal(%T) failed: %v\nJSON: %s", out, err, string(data))
	}
	return out
}

func TestDeconstructRoundTripSQL(t *testing.T) {
	info := getTestInfo()

	tests := []struct {
		name string
		expr expr.Expression
	}{
		{
			name: "String",
			expr: expr.String("1 = 1"),
		},
		{
			name: "Field",
			expr: expr.Field("Age"),
		},
		{
			name: "FieldWithAlias",
			expr: expr.Field("alias.Name"),
		},
		{
			name: "ValueString",
			expr: expr.Value("alice"),
		},
		{
			name: "ValueInt",
			expr: expr.Value(42),
		},
		{
			name: "ValueFloat",
			expr: expr.Value(3.14),
		},
		{
			name: "ValueBool",
			expr: expr.Value(true),
		},
		{
			name: "ValueNil",
			expr: expr.Value(nil),
		},
		{
			name: "ValueUnsafeInt",
			expr: expr.Value(99, true),
		},
		{
			name: "ValueUnsafeString",
			expr: expr.Value("raw_lit", true),
		},
		{
			name: "ValuesMixed",
			expr: expr.Values(1, "two", 3.5, true),
		},
		{
			name: "ValuesIntsAndStrings",
			expr: expr.Values(42, "two"),
		},
		{
			name: "AsField",
			expr: expr.As("age_alias", expr.Field("Age")),
		},
		{
			name: "AsComputed",
			expr: expr.As("over_18", expr.Q("Age__gt", 18)),
		},
		{
			name: "QGT",
			expr: expr.Q("Age__gt", 18),
		},
		{
			name: "QInSlice",
			expr: expr.Q("Age__in", []any{1, 2, 3}),
		},
		{
			name: "QNestedExpression",
			expr: expr.Q("Score__gt", expr.Field("Age")),
		},
		{
			name: "QNot",
			expr: expr.Q("Age", 18).Not(true),
		},
		{
			name: "QIContains",
			expr: expr.Q("Name__icontains", "foo"),
		},
		{
			name: "ExprGroupAnd",
			expr: expr.And(expr.Q("Age__gt", 18), expr.Q("Name", "alice")),
		},
		{
			name: "ExprGroupOr",
			expr: expr.Or(expr.Q("Age__lt", 10), expr.Q("Age__gt", 60)),
		},
		{
			name: "ExprGroupNot",
			expr: expr.And(expr.Q("Age", 18)).Not(true),
		},
		{
			name: "ExprGroupNested",
			expr: expr.Or[expr.Expression](
				expr.And(expr.Q("Age__gt", 18), expr.Q("Score__gte", 50)),
				expr.Q("Name", "admin"),
			),
		},
		{
			name: "CaseWithDefault",
			expr: expr.Case(
				expr.When("Age__gt", 18).Then("adult"),
				expr.When("Age__lt", 13).Then("child"),
				expr.Value("teen"),
			),
		},
		{
			name: "CaseNoDefault",
			expr: expr.Case(expr.When(expr.Q("Age__gte", 13), "teen")),
		},
		{
			name: "CastString",
			expr: expr.CastString("Name", 255),
		},
		{
			name: "CastInt",
			expr: expr.CastInt("Age"),
		},
		{
			name: "CastFloat",
			expr: expr.CastFloat("Score", 10, 2),
		},
		{
			name: "CastDecimal",
			expr: expr.Cast(expr.CastTypeDecimal, expr.Field("Age"), 8, 2),
		},
		{
			name: "ChainWithOpAndValue",
			expr: expr.Chain("Age", expr.ADD, expr.Value(5)),
		},
		{
			name: "ChainWithFields",
			expr: expr.Chain(expr.Field("Score"), expr.SUB, expr.Field("Age")),
		},
		{
			name: "ChainWithStringExpr",
			expr: expr.Chain(expr.String("Age")),
		},
		{
			name: "LogicalEQ",
			expr: expr.Logical("Age").EQ(18),
		},
		{
			name: "LogicalChained",
			expr: expr.Logical("Age").ADD(5).GT(20),
		},
		{
			name: "LogicalScope",
			expr: expr.Logical("Age").Scope(expr.EQ, expr.Value(42)),
		},
		{
			name: "RawIndexedValue",
			expr: expr.Raw("![Age] + ?[1]", 5),
		},
		{
			name: "FStatement",
			expr: expr.F("![Age] = ? AND ![Name] = ?", 18, "alice"),
		},
		{
			name: "RawWithExprMap",
			expr: expr.Raw(
				"![Age] = ? AND expr(cond)",
				18,
				expr.PARSER.Expr.Expressions(map[string]expr.Expression{
					"cond": expr.Q("Score__gt", 50),
				}),
			),
		},
		{
			name: "RawWithExprList",
			expr: expr.Raw(
				"![Age] = ? AND expr(0)",
				18,
				expr.PARSER.Expr.Expressions(expr.Q("Score__lt", 10)),
			),
		},
		{
			name: "RawWithNestedExprArg",
			expr: expr.Raw("![Age] = ?[1] AND ?[2]", 18, expr.Field("Score")),
		},
		{
			name: "FuncSUM",
			expr: expr.SUM("Score"),
		},
		{
			name: "FuncCOUNT",
			expr: expr.COUNT("ID"),
		},
		{
			name: "FuncAVG",
			expr: expr.AVG("Age"),
		},
		{
			name: "FuncCOALESCE",
			expr: expr.COALESCE(expr.Field("Nickname"), expr.Value("anon")),
		},
		{
			name: "FuncSUBSTR",
			expr: expr.SUBSTR("Name", 1, 3),
		},
		{
			name: "FuncDateFormat",
			expr: expr.DATE_FORMAT("CreatedAt", "%Y-%m-%d"),
		},
		{
			name: "FuncNOW",
			expr: expr.NOW(),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			wantSQL, wantArgs := renderSQL(t, info, tc.expr)
			reconstructed := roundTrip[expr.Expression](t, tc.expr)
			gotSQL, gotArgs := renderSQL(t, info, reconstructed)

			if gotSQL != wantSQL {
				t.Errorf("SQL mismatch:\nwant: %s\ngot:  %s", wantSQL, gotSQL)
			}
			if !reflect.DeepEqual(gotArgs, wantArgs) {
				t.Errorf("args mismatch:\nwant: %#v\ngot:  %#v", wantArgs, gotArgs)
			}
		})
	}

	t.Run("AlreadyResolvedRoundTrip", func(t *testing.T) {
		resolvedInputs := []expr.Expression{
			expr.Field("Age"),
			expr.Value(42),
			expr.Q("Age__gt", 18),
			expr.Case(expr.When("Age__gt", 18).Then("adult"), expr.Value("minor")),
			expr.CastInt("Age"),
			expr.SUM("Score"),
			expr.Raw("![Age] + ?[1]", 5),
			expr.Logical("Age").ADD(5).GT(20),
		}

		for i, orig := range resolvedInputs {
			resolvedOrig := orig.Resolve(info)
			wantSQL, wantArgs := renderSQL(t, info, resolvedOrig)
			reconstructed := roundTrip[expr.Expression](t, resolvedOrig)
			gotSQL, gotArgs := renderSQL(t, info, reconstructed)

			if gotSQL != wantSQL {
				t.Errorf("case %d SQL mismatch:\nwant: %s\ngot:  %s", i, wantSQL, gotSQL)
			}
			if !reflect.DeepEqual(gotArgs, wantArgs) {
				t.Errorf("case %d args mismatch:\nwant: %#v\ngot:  %#v", i, wantArgs, gotArgs)
			}
		}
	})
}

func TestDeconstructPostgresTypePreservation(t *testing.T) {
	pgInfo := *getTestInfo()
	pgInfo.Driver = &drivers.DriverPostgres{}
	pgInfo.Placeholder = "?"

	cases := []struct {
		name      string
		expr      expr.Expression
		substring string
	}{
		{
			name:      "ValueInt",
			expr:      expr.Value(42),
			substring: "::INT",
		},
		{
			name:      "ValueFloat",
			expr:      expr.Value(3.5),
			substring: "::FLOAT",
		},
		{
			name:      "ValuesMixed",
			expr:      expr.Values(1, "a", 2.5, true),
			substring: "::INT, ?::TEXT, ?::FLOAT, ?::BOOLEAN",
		},
		{
			name:      "CastString",
			expr:      expr.CastString("Name", 255),
			substring: "VARCHAR(255)",
		},
		{
			name:      "CastFloat",
			expr:      expr.CastFloat("Score", 10, 2),
			substring: "NUMERIC(10,2)",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			wantSQL, wantArgs := renderSQL(t, &pgInfo, tc.expr)
			reconstructed := roundTrip[expr.Expression](t, tc.expr)
			gotSQL, gotArgs := renderSQL(t, &pgInfo, reconstructed)

			if gotSQL != wantSQL {
				t.Errorf("Postgres SQL mismatch:\nwant: %s\ngot:  %s", wantSQL, gotSQL)
			}
			if !strings.Contains(gotSQL, tc.substring) {
				t.Errorf("expected Postgres SQL %q to contain %q", gotSQL, tc.substring)
			}
			if !reflect.DeepEqual(gotArgs, wantArgs) {
				t.Errorf("Postgres args mismatch:\nwant: %#v\ngot:  %#v", wantArgs, gotArgs)
			}
		})
	}
}

func TestDeconstructMetadataAndInterfaces(t *testing.T) {
	info := getTestInfo()

	t.Run("NamedExpressionFieldNames", func(t *testing.T) {
		fieldRT := roundTrip[expr.NamedExpression](t, expr.Field("Age"))
		if fieldRT.FieldName() != "Age" {
			t.Errorf("Field.FieldName() = %q, want %q", fieldRT.FieldName(), "Age")
		}

		asOrig := expr.As("age_alias", expr.Field("Age"))
		asRT := roundTrip[expr.AnnotationExpression](t, asOrig)
		if asRT.FieldName() != "age_alias" {
			t.Errorf("As.FieldName() = %q, want %q", asRT.FieldName(), "age_alias")
		}
		if !asRT.Annotate() {
			t.Errorf("As.Annotate() = false, want true")
		}

		castRT := roundTrip[expr.NamedExpression](t, expr.CastString("Name", 255))
		if castRT.FieldName() != "Name" {
			t.Errorf("CastString.FieldName() = %q, want %q", castRT.FieldName(), "Name")
		}

		chainRT := roundTrip[expr.NamedExpression](t, expr.Chain("Age", expr.ADD, expr.Value(1)))
		if chainRT.FieldName() != "Age" {
			t.Errorf("Chain.FieldName() = %q, want %q", chainRT.FieldName(), "Age")
		}

		logicalRT := roundTrip[expr.LogicalExpression](t, expr.Logical("Age").EQ(18))
		if named, ok := logicalRT.(expr.NamedExpression); !ok || named.FieldName() != "Age" {
			t.Errorf("Logical.FieldName() failed, got %T", logicalRT)
		}

		fOrig := expr.F("![Age] = ?", 18)
		fRT := roundTrip[expr.NamedExpression](t, fOrig)
		if fRT.FieldName() != "Age" {
			t.Errorf("F.FieldName() = %q, want %q", fRT.FieldName(), "Age")
		}

		sumRT := roundTrip[expr.LogicalNamedExpressionFunc](t, expr.SUM("Score"))
		if sumRT.FieldName() != "Score" {
			t.Errorf("SUM.FieldName() = %q, want %q", sumRT.FieldName(), "Score")
		}

		outerRT := roundTrip[expr.NamedExpression](t, expr.OuterRef("Age"))
		if outerRT.FieldName() != "Age" {
			t.Errorf("OuterRef.FieldName() = %q, want %q", outerRT.FieldName(), "Age")
		}
	})

	t.Run("ClauseExpressionState", func(t *testing.T) {
		node := expr.Q("Age", 18).Not(true)
		nodeRT := roundTrip[*expr.ExprNode](t, node)
		if !nodeRT.IsNot() {
			t.Errorf("ExprNode.IsNot() = false, want true")
		}

		group := expr.Or(expr.Q("Age", 18), expr.Q("Score", 10)).Not(true)
		groupRT := roundTrip[expr.ClauseExpression](t, group)
		if !groupRT.IsNot() {
			t.Errorf("ExprGroup.IsNot() = false, want true")
		}
		if eg, ok := groupRT.(*expr.ExprGroup); !ok || eg.Operator() != expr.OpOr {
			t.Errorf("ExprGroup.Operator() failed, got %#v", groupRT)
		}
	})

	t.Run("ConcreteAndInterfaceTargets", func(t *testing.T) {
		strRT := roundTrip[expr.String](t, expr.String("custom_sql"))
		if strRT != "custom_sql" {
			t.Errorf("String = %q, want %q", strRT, "custom_sql")
		}

		opRT := roundTrip[expr.LogicalOp](t, expr.GTE)
		if opRT != expr.GTE {
			t.Errorf("LogicalOp = %q, want %q", opRT, expr.GTE)
		}

		whenRT := roundTrip[expr.WhenExpression](t, expr.When("Age__gt", 18).Then("adult"))
		caseWithWhen := expr.Case(whenRT, expr.Value("minor"))
		wantSQL, wantArgs := renderSQL(t, info, expr.Case(expr.When("Age__gt", 18).Then("adult"), expr.Value("minor")))
		gotSQL, gotArgs := renderSQL(t, info, caseWithWhen)
		if gotSQL != wantSQL || !reflect.DeepEqual(gotArgs, wantArgs) {
			t.Errorf("WhenExpression round-trip SQL/args mismatch: got (%q, %#v), want (%q, %#v)", gotSQL, gotArgs, wantSQL, wantArgs)
		}

		groupRT := roundTrip[*expr.ExprGroup](t, expr.And(expr.Q("Age", 18)))
		if groupRT.Operator() != expr.OpAnd {
			t.Errorf("*ExprGroup.Operator() = %q, want %q", groupRT.Operator(), expr.OpAnd)
		}

		caseRT := roundTrip[*expr.CaseExpression](t, expr.Case(expr.When("Age__gt", 18).Then(1), expr.Value(0)))
		gotSQL, gotArgs = renderSQL(t, info, caseRT)
		wantSQL, wantArgs = renderSQL(t, info, expr.Case(expr.When("Age__gt", 18).Then(1), expr.Value(0)))
		if gotSQL != wantSQL || !reflect.DeepEqual(gotArgs, wantArgs) {
			t.Errorf("*CaseExpression round-trip mismatch")
		}

		rawOrig := expr.Raw("![Age] = ?", 18).(*expr.RawNamedExpression)
		rawRT := roundTrip[*expr.RawNamedExpression](t, rawOrig)
		gotSQL, gotArgs = renderSQL(t, info, rawRT)
		wantSQL, wantArgs = renderSQL(t, info, rawOrig)
		if gotSQL != wantSQL || !reflect.DeepEqual(gotArgs, wantArgs) {
			t.Errorf("*RawNamedExpression round-trip mismatch")
		}

		nilStmtRaw := &expr.RawNamedExpression{Field: "Age"}
		nilStmtRawRT := roundTrip[*expr.RawNamedExpression](t, nilStmtRaw)
		if nilStmtRawRT.FieldName() != "Age" || nilStmtRawRT.Statement != nil {
			t.Errorf("nil Statement RawNamedExpression mismatch: %+v", nilStmtRawRT)
		}

		stmtOrig := expr.ParseExprStatement("![Age] = ?[1]", []any{18})
		stmtRT := roundTrip[*expr.ExpressionStatement](t, stmtOrig)
		resolvedOrig := stmtOrig.Resolve(info)
		resolvedRT := stmtRT.Resolve(info)
		wantStmtSQL, wantStmtArgs := resolvedOrig.SQL()
		gotStmtSQL, gotStmtArgs := resolvedRT.SQL()
		if gotStmtSQL != wantStmtSQL || !reflect.DeepEqual(gotStmtArgs, wantStmtArgs) {
			t.Errorf("*ExpressionStatement mismatch: got (%q, %#v), want (%q, %#v)", gotStmtSQL, gotStmtArgs, wantStmtSQL, wantStmtArgs)
		}

		// Extract inner *expr.Function from expr.SUBSTR("Name", 1, 3) envelope and round-trip directly.
		substrBytes, err := json.Marshal(expr.SUBSTR("Name", 1, 3), deconstruct.Constructors.JSONOptions())
		if err != nil {
			t.Fatalf("marshal SUBSTR failed: %v", err)
		}
		var envelope deconstruct.Deconstructed
		if err := json.Unmarshal(substrBytes, &envelope); err != nil {
			t.Fatalf("unmarshal envelope failed: %v", err)
		}
		var fn1 *expr.Function
		if err := deconstruct.Constructors.Unmarshal(envelope.Args[0], &fn1); err != nil {
			t.Fatalf("unmarshal *expr.Function failed: %v", err)
		}
		fn2 := roundTrip[*expr.Function](t, fn1)
		wantFnSQL, wantFnArgs := renderSQL(t, info, fn1)
		gotFnSQL, gotFnArgs := renderSQL(t, info, fn2)
		if gotFnSQL != wantFnSQL || !reflect.DeepEqual(gotFnArgs, wantFnArgs) {
			t.Errorf("*Function round-trip mismatch: got (%q, %#v), want (%q, %#v)", gotFnSQL, gotFnArgs, wantFnSQL, wantFnArgs)
		}
	})

	t.Run("CollectionsAndWrapperStruct", func(t *testing.T) {
		sliceOrig := []expr.Expression{expr.Q("Age__gt", 18), expr.Value("ok")}
		sliceRT := roundTrip[[]expr.Expression](t, sliceOrig)
		if len(sliceRT) != len(sliceOrig) {
			t.Fatalf("slice len = %d, want %d", len(sliceRT), len(sliceOrig))
		}
		for i := range sliceOrig {
			wSQL, wArgs := renderSQL(t, info, sliceOrig[i])
			gSQL, gArgs := renderSQL(t, info, sliceRT[i])
			if gSQL != wSQL || !reflect.DeepEqual(gArgs, wArgs) {
				t.Errorf("slice[%d] mismatch", i)
			}
		}

		mapOrig := map[string]expr.Expression{
			"age":  expr.Field("Age"),
			"cond": expr.Q("Score__gte", 50),
		}
		mapRT := roundTrip[map[string]expr.Expression](t, mapOrig)
		for k, v := range mapOrig {
			wSQL, wArgs := renderSQL(t, info, v)
			gSQL, gArgs := renderSQL(t, info, mapRT[k])
			if gSQL != wSQL || !reflect.DeepEqual(gArgs, wArgs) {
				t.Errorf("map[%q] mismatch", k)
			}
		}

		type wrapper struct {
			Select expr.Expression
			Where  expr.ClauseExpression
		}
		wrapOrig := wrapper{
			Select: expr.As("max_score", expr.MAX("Score")),
			Where:  expr.And(expr.Q("Age__gt", 18), expr.Q("Name", "alice")),
		}
		wrapRT := roundTrip[wrapper](t, wrapOrig)
		wSQL, wArgs := renderSQL(t, info, wrapOrig.Select)
		gSQL, gArgs := renderSQL(t, info, wrapRT.Select)
		if gSQL != wSQL || !reflect.DeepEqual(gArgs, wArgs) {
			t.Errorf("wrapper.Select mismatch")
		}
		wSQL, wArgs = renderSQL(t, info, wrapOrig.Where)
		gSQL, gArgs = renderSQL(t, info, wrapRT.Where)
		if gSQL != wSQL || !reflect.DeepEqual(gArgs, wArgs) {
			t.Errorf("wrapper.Where mismatch")
		}
	})
}

func TestDeconstructOuterRef(t *testing.T) {
	parentInfo := getTestInfo()
	ctx := expr.AddParentSubqueryContext(context.Background(), parentInfo)
	subInfo := getTestInfo()
	subInfo.Resolver = &mockFieldResolver{
		ctx:           ctx,
		FieldResolver: parentInfo.Resolver,
	}

	for _, fld := range []string{"Age", "alias.Name"} {
		t.Run(fld, func(t *testing.T) {
			orig := expr.OuterRef(fld)
			wantSQL, wantArgs := renderSQL(t, subInfo, orig)

			rt := roundTrip[expr.NamedExpression](t, orig)
			if rt.FieldName() != fld {
				t.Errorf("OuterRef.FieldName() = %q, want %q", rt.FieldName(), fld)
			}
			gotSQL, gotArgs := renderSQL(t, subInfo, rt)
			if gotSQL != wantSQL {
				t.Errorf("OuterRef SQL mismatch: got %q, want %q", gotSQL, wantSQL)
			}
			if !reflect.DeepEqual(gotArgs, wantArgs) {
				t.Errorf("OuterRef args mismatch: got %#v, want %#v", gotArgs, wantArgs)
			}
		})
	}
}

func TestDeconstructConstraints(t *testing.T) {
	tests := []struct {
		name       string
		constraint constraints.Constraint
	}{
		{
			name:       "ForeignKeyConstraint",
			constraint: constraints.ForeignKeyConstraint("fk_user", "invalid user", "Name", constraints.A_CASCADE, constraints.A_SET_NULL),
		},
		{
			name:       "CheckConstraint_Simple",
			constraint: constraints.CheckConstraint("chk_age", "too young", expr.Q("Age__gte", expr.V(18, true))),
		},
		{
			name: "CheckConstraint_Complex",
			constraint: constraints.CheckConstraint(
				"chk_complex",
				"invalid range",
				expr.And[expr.Expression](
					expr.Q("Age__gte", expr.V(18, true)),
					expr.Or(
						expr.Q("Score__gt", expr.V(0, true)),
						expr.Q("Nickname__isnull", false),
					),
				),
			),
		},
		{
			name:       "CheckConstraint_Raw",
			constraint: constraints.CheckConstraint("chk_len", "name too short", expr.Raw("LENGTH(![Name]) >= ?", expr.V(3, true))),
		},
		{
			name:       "UniqueConstraint",
			constraint: constraints.UniqueConstraint("uq_name_age", "duplicate", []string{"Name", "Age"}),
		},
		{
			name:       "UniqueConstraint_SingleAndVariadic",
			constraint: constraints.UniqueConstraint("uq_single", "dup", "FirstName", "LastName"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			origDecon, ok := tc.constraint.(deconstruct.Deconstructible)
			if !ok {
				t.Fatalf("%T does not implement deconstruct.Deconstructible", tc.constraint)
			}
			wantPath, wantArgs, wantKwargs := origDecon.Deconstruct()

			rtExpr := roundTrip[expr.Expression](t, tc.constraint)
			rtDecon := roundTrip[deconstruct.Deconstructible](t, tc.constraint)
			rtConstraint := roundTrip[constraints.Constraint](t, tc.constraint)

			if rtConstraint.ID() != tc.constraint.ID() {
				t.Errorf("ID() = %q, want %q", rtConstraint.ID(), tc.constraint.ID())
			}
			if rtConstraint.Type() != tc.constraint.Type() {
				t.Errorf("Type() = %v, want %v", rtConstraint.Type(), tc.constraint.Type())
			}
			if rtConstraint.ErrorMessage() != tc.constraint.ErrorMessage() {
				t.Errorf("ErrorMessage() = %q, want %q", rtConstraint.ErrorMessage(), tc.constraint.ErrorMessage())
			}

			gotExprDecon, ok := rtExpr.(deconstruct.Deconstructible)
			if !ok {
				t.Fatalf("%T does not implement deconstruct.Deconstructible", rtExpr)
			}
			gotConsDecon, ok := rtConstraint.(deconstruct.Deconstructible)
			if !ok {
				t.Fatalf("%T does not implement deconstruct.Deconstructible", rtConstraint)
			}
			for _, rt := range []deconstruct.Deconstructible{gotExprDecon, rtDecon, gotConsDecon} {
				gotPath, gotArgs, gotKwargs := rt.Deconstruct()
				if gotPath != wantPath {
					t.Errorf("path = %q, want %q", gotPath, wantPath)
				}
				if len(gotArgs) != len(wantArgs) {
					t.Errorf("args len = %d, want %d", len(gotArgs), len(wantArgs))
				}
				if !reflect.DeepEqual(gotKwargs, wantKwargs) {
					t.Errorf("kwargs = %#v, want %#v", gotKwargs, wantKwargs)
				}
				b1, err := json.Marshal(origDecon, deconstruct.Constructors.JSONOptions(), json.Deterministic(true))
				if err != nil {
					t.Fatalf("marshal orig failed: %v", err)
				}
				b2, err := json.Marshal(rt, deconstruct.Constructors.JSONOptions(), json.Deterministic(true))
				if err != nil {
					t.Fatalf("marshal rt failed: %v", err)
				}
				if string(b1) != string(b2) {
					t.Errorf("re-marshaled JSON mismatch:\nwant: %s\ngot:  %s", string(b1), string(b2))
				}
			}

			// Verify Clone() produces an equivalent independent constraint
			cloned := tc.constraint.Clone().(constraints.Constraint)
			bOrig, _ := json.Marshal(tc.constraint, deconstruct.Constructors.JSONOptions(), json.Deterministic(true))
			bClone, _ := json.Marshal(cloned, deconstruct.Constructors.JSONOptions(), json.Deterministic(true))
			if string(bOrig) != string(bClone) {
				t.Errorf("Clone() JSON mismatch:\nwant: %s\ngot:  %s", string(bOrig), string(bClone))
			}
		})
	}
}

func TestDeconstructLegacyValueKwargs(t *testing.T) {
	info := getTestInfo()

	tests := []struct {
		name string
		raw  string
		want expr.Expression
	}{
		{
			name: "UnsafeInt",
			raw:  `{"path":"queries.expr.Value","args":[],"kwargs":{"value":123,"unsafe":true}}`,
			want: expr.Value(123, true),
		},
		{
			name: "SafeString",
			raw:  `{"path":"queries.expr.Value","args":[],"kwargs":{"value":"hello"}}`,
			want: expr.Value("hello"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var got expr.Expression
			if err := deconstruct.Constructors.Unmarshal([]byte(tc.raw), &got); err != nil {
				t.Fatalf("Unmarshal failed: %v", err)
			}
			gotSQL, gotArgs := renderSQL(t, info, got)
			wantSQL, wantArgs := renderSQL(t, info, tc.want)
			if gotSQL != wantSQL || !reflect.DeepEqual(gotArgs, wantArgs) {
				t.Errorf("legacy Value mismatch: got (%q, %#v), want (%q, %#v)", gotSQL, gotArgs, wantSQL, wantArgs)
			}
		})
	}
}

func TestDeconstructErrors(t *testing.T) {
	t.Run("UnregisteredPath", func(t *testing.T) {
		var e expr.Expression
		err := deconstruct.Constructors.Unmarshal(
			[]byte(`{"path":"queries.expr.DoesNotExist","args":[],"kwargs":null}`),
			&e,
		)
		if err == nil {
			t.Fatal("expected error for unregistered path")
		}

		// Nested unregistered path inside Values
		err = deconstruct.Constructors.Unmarshal(
			[]byte(`{"path":"queries.expr.Values","args":[{"path":"queries.expr.DoesNotExist","args":[],"kwargs":null}],"kwargs":null}`),
			&e,
		)
		if err == nil {
			t.Fatal("expected error for nested unregistered path inside Values")
		}
	})

	t.Run("InvalidArityAndKwargs", func(t *testing.T) {
		payloads := []struct {
			name string
			json string
		}{
			{"LogicalOp_0Args", `{"path":"queries.expr.LogicalOp","args":[],"kwargs":null}`},
			{"String_0Args", `{"path":"queries.expr.String","args":[],"kwargs":null}`},
			{"Field_0Args", `{"path":"queries.expr.Field","args":[],"kwargs":null}`},
			{"Value_2Args", `{"path":"queries.expr.Value","args":[1,2],"kwargs":null}`},
			{"As_1Arg", `{"path":"queries.expr.As","args":["a"],"kwargs":null}`},
			{"When_1Arg", `{"path":"queries.expr.When","args":[null],"kwargs":null}`},
			{"Cast_1Arg", `{"path":"queries.expr.Cast","args":[1],"kwargs":null}`},
			{"Chain_0Args", `{"path":"queries.expr.Chain","args":[],"kwargs":null}`},
			{"ExprNode_1Arg", `{"path":"queries.expr.ExprNode","args":[null],"kwargs":null}`},
			{"Logical_0Args", `{"path":"queries.expr.Logical","args":[],"kwargs":null}`},
			{"Raw_0Args", `{"path":"queries.expr.Raw","args":[],"kwargs":null}`},
			{"Statement_0Args", `{"path":"queries.expr.Statement","args":[],"kwargs":null}`},
			{"ParserArg_1Arg", `{"path":"queries.expr.ParserArg","args":["expr"],"kwargs":null}`},
			{"ExpressionData_1Arg", `{"path":"queries.expr.ExpressionData","args":[[]],"kwargs":null}`},
			{"OuterRef_0Args", `{"path":"queries.expr.OuterRef","args":[],"kwargs":null}`},
			{"ForeignKey_0Args", `{"path":"queries.expr.constraints.ForeignKey","args":[],"kwargs":null}`},
			{"Check_0Args", `{"path":"queries.expr.constraints.Check","args":[],"kwargs":null}`},
			{"Function_MissingFunc", `{"path":"queries.expr.Function","args":[],"kwargs":{}}`},
			{"Function_EmptyFunc", `{"path":"queries.expr.Function","args":[],"kwargs":{"func":""}}`},
			{"Function_InvalidParams", `{"path":"queries.expr.Function","args":[],"kwargs":{"func":"SUM","params":"not_a_slice"}}`},
		}

		for _, tc := range payloads {
			t.Run(tc.name, func(t *testing.T) {
				var d deconstruct.Deconstructible
				if err := deconstruct.Constructors.Unmarshal([]byte(tc.json), &d); err == nil {
					t.Errorf("%s: expected error, got nil (result: %#v)", tc.name, d)
				}
			})
		}
	})

	t.Run("TargetTypeMismatch", func(t *testing.T) {
		strJSON, err := json.Marshal(expr.String("x"), deconstruct.Constructors.JSONOptions())
		if err != nil {
			t.Fatalf("marshal String failed: %v", err)
		}

		var clause expr.ClauseExpression
		if err := deconstruct.Constructors.Unmarshal(strJSON, &clause); err == nil {
			t.Error("expected error unmarshaling String into ClauseExpression")
		}

		var node *expr.ExprNode
		if err := deconstruct.Constructors.Unmarshal(strJSON, &node); err == nil {
			t.Error("expected error unmarshaling String into *ExprNode")
		}
	})
}
