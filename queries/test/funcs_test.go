package queries_test

import (
	"strings"
	"testing"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/expr"
	"github.com/Nigel2392/go-django/src/core/attrs"
)

func TestSelectFuncExpr(t *testing.T) {
	var test = &TestStruct{
		Name: "TestWhereFilterVirtualFieldAliassed",
		Text: "TestWhereFilterVirtualFieldAliassed",
	}

	if err := queries.CreateObject(test); err != nil {
		t.Fatalf("Failed to create object: %v", err)
	}

	t.Run("SelectNameSubString", func(t *testing.T) {
		var qs = queries.Objects[attrs.Definer](test).
			Select("ID", expr.SUBSTR(expr.UPPER("Name"), expr.Value(1), expr.Value(9)), "Text").
			Filter("ID", test.ID)
		var rows, err = qs.All()

		if err != nil {
			t.Fatalf("Failed to execute query: %v", err)
		}

		if len(rows) == 0 {
			t.Fatal("expected at least one result")
		}

		t.Logf("SQL: %s", qs.LatestQuery().SQL())
		t.Logf("ARGS: %v", qs.LatestQuery().Args())

		if rows[0].Object.(*TestStruct).ID != test.ID {
			t.Errorf("expected ID = %d, got %d", test.ID, rows[0].Object.(*TestStruct).ID)
		}

		if rows[0].Object.(*TestStruct).Name != strings.ToUpper(test.Name[0:9]) {
			// Note: Name is aliassed to Substr(Name, 0, 9)
			t.Errorf("expected Name = %q, got %q", strings.ToUpper(test.Name[0:9]), rows[0].Object.(*TestStruct).Name)
		}

		if rows[0].Object.(*TestStruct).Text != test.Text {
			t.Errorf("expected Text = %q, got %q", test.Text, rows[0].Object.(*TestStruct).Text)
		}

		t.Logf("Row: %#v", rows[0].Object.(*TestStruct))
	})
}
