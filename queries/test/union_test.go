package queries_test

import (
	"reflect"
	"testing"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/expr"
	"github.com/Nigel2392/go-django/src/core/attrs"
)

func TestQuerySetUnion(t *testing.T) {

	var todos1 = []*Todo{
		{Title: "Union1", Description: "Description Union", Done: false},
		{Title: "Union2", Description: "Description Union", Done: true},
	}

	for _, todo := range todos1 {
		if err := queries.CreateObject(todo); err != nil {
			t.Fatalf("Failed to insert todo1: %v", err)
		}
	}

	var todos2 = []*Todo{
		{Title: "Union3", Description: "Description Union", Done: false},
		{Title: "Union4", Description: "Description Union", Done: true},
	}

	for _, todo := range todos2 {
		if err := queries.CreateObject(todo); err != nil {
			t.Fatalf("Failed to insert todo2: %v", err)
		}
	}

	t.Run("UnionAll", func(t *testing.T) {
		var qs1 = queries.GetQuerySet(&Todo{}).
			Select("ID", "Title", "Description", "Done").
			Filter("Done", false)

		var qs2 = queries.GetQuerySet[attrs.Definer](&Todo{}).
			Select("ID", "Title", "Description", "Done").
			Filter("Done", true)

		unioned, err := qs1.Union(qs2).
			Filter("Title__icontains", "Union").
			Filter("User.ID__isnull", true).
			All()
		if err != nil {
			t.Fatalf("Failed to union todos: %v", err)
			return
		}

		if len(unioned) != len(todos1)+len(todos2) {
			t.Fatalf("Expected %d todos, got %d", len(todos1)+len(todos2), len(unioned))
			return
		}

		for _, todo := range append(todos1, todos2...) {
			var found bool
			for _, uTodo := range unioned {
				if uTodo.Object.ID == todo.ID {
					found = true
					break
				}
			}

			if !found {
				t.Fatalf("Expected todo with ID %d to be in unioned results", todo.ID)
			}
		}
	})

	var profiles = []*Profile{
		{Name: "TestQuerySetUnion 1", Email: "example_1@example.com"},
		{Name: "TestQuerySetUnion 2", Email: "example_2@example.com"},
	}

	var users = []*User{
		{Name: "TestQuerySetUnion 1", Profile: profiles[0]},
		{Name: "TestQuerySetUnion 2", Profile: profiles[1]},
	}

	var err error
	_, err = queries.GetQuerySet(&Profile{}).BulkCreate(profiles)
	if err != nil {
		t.Fatalf("Failed to create *Profile objects: %v", err)
	}

	_, err = queries.GetQuerySet(&User{}).BulkCreate(users)
	if err != nil {
		t.Fatalf("Failed to create *User objects: %v", err)
	}

	t.Run("UnionOtherQuerySet", func(t *testing.T) {
		var qs1 = queries.GetQuerySet(&Todo{}).
			Select("ID", "Title").
			Filter("User.ID__isnull", true).
			Filter("Done", false)

		var qs2 = queries.GetQuerySet[attrs.Definer](&Todo{}).
			Select("ID", "Title").
			Filter("User.ID__isnull", true).
			Filter("Done", true)

		var qs3 = queries.GetQuerySet[attrs.Definer](&User{}).
			Select("ID").
			Annotate("Title", expr.LOWER("Name"))

		unioned, err := qs1.
			Union(qs2).
			Union(qs3).
			Filter("Title__icontains", "union").
			All()
		if err != nil {
			t.Fatalf("Failed to union todos: %v", err)
			return
		}

		if len(unioned) != len(todos1)+len(todos2)+len(users) {
			t.Errorf("Expected %d todos, got %d", len(todos1)+len(todos2)+len(users), len(unioned))
			for _, row := range unioned {
				t.Logf("Unioned row: %+v", row.Object)
			}
			t.FailNow()
			return
		}

		for _, todo := range append(todos1, todos2...) {
			var found bool
			for _, uTodo := range unioned {
				if uTodo.Object.ID == todo.ID {
					found = true
					break
				}
			}

			if !found {
				t.Fatalf("Expected todo with ID %d to be in unioned results", todo.ID)
			}
		}
	})

	t.Run("SIMPLE", func(t *testing.T) {
		qs1 := queries.GetQuerySet[attrs.Definer](&User{}).Limit(5000).Annotate("type", expr.Value("user")).Select(
			"ID",           // 1
			"Name",         // 2
			expr.Value(""), // 3
			"type",         // 4
		)

		qs2 := queries.GetQuerySet(&Todo{}).Limit(5000).Annotate("type", expr.Value("todo")).Select(
			"ID",          // 1
			"Title",       // 2
			"Description", // 3
			"type",        // 4
		)

		rows, err := qs2.Union(qs1).OrderBy("type", "ID").All()
		if err != nil {
			t.Fatal("Error for queryset union")
		}

		var typeMap = make(map[string]reflect.Type)
		typeMap["user"] = reflect.TypeOf(&User{})
		typeMap["todo"] = reflect.TypeOf(&Todo{})

		count1, err := qs1.Count()
		if err != nil {
			t.Fatalf("could not retrieve count for queryset 1: %v", err)
		}
		count2, err := qs2.Count()
		if err != nil {
			t.Fatalf("could not retrieve count for queryset 2: %v", err)
		}

		if len(rows) != int(count1+count2) {
			t.Fatalf("missing rows in union query, should be: %d, result: %d", int(count1+count2), len(rows))
		}

		for _, row := range rows {
			t.Logf(
				"queried union row: (annotations=%v)\n\tID: %v\n\tTitle: %q\n\tDesc: %q",
				row.Annotations,
				row.Object.ID,
				row.Object.Title,
				row.Object.Description,
			)
		}
	})

	t.Run("SCAN", func(t *testing.T) {
		qs1 := queries.GetQuerySet[attrs.Definer](&User{}).Limit(5000).Annotate("type", expr.Value("user")).Select(
			"ID",           // 1
			"Name",         // 2
			expr.Value(""), // 3
			"type",         // 4
		)

		qs2 := queries.GetQuerySet(&Todo{}).Limit(5000).Annotate("type", expr.Value("todo")).Select(
			"ID",          // 1
			"Title",       // 2
			"Description", // 3
			"type",        // 4
		)

		rows := qs2.Union(qs1).OrderBy("type").QueryAll()
		if err := rows.Err(); err != nil {
			t.Fatalf("Error for queryset union: %v", err)
		}

		var typeMap = make(map[string]reflect.Type)
		typeMap["user"] = reflect.TypeOf(&User{})
		typeMap["todo"] = reflect.TypeOf(&Todo{})

		count1, err := qs1.Count()
		if err != nil {
			t.Fatalf("could not retrieve count for queryset 1: %v", err)
		}
		count2, err := qs2.Count()
		if err != nil {
			t.Fatalf("could not retrieve count for queryset 2: %v", err)
		}

		type rowObject struct {
			id    int
			title string
			desc  string
			typ   string
		}

		var rowList = make([]rowObject, 0)
		for rows.Next() {
			var row rowObject

			if err := rows.Scan(&row.id, &row.title, &row.desc, &row.typ); err != nil {
				t.Fatalf("error while scanning row: %v", err)
			}

			rowList = append(rowList, row)
		}

		if len(rowList) != int(count1+count2) {
			t.Fatalf("missing rows in union query, should be: %d, result: %d", int(count1+count2), len(rowList))
		}

		for _, row := range rowList {
			t.Logf(
				"queried union row: %q\n\tID: %v\n\tTitle: %q\n\tDesc: %q",
				row.typ,
				row.id,
				row.title,
				row.desc,
			)
		}
	})

}
