package queries_test

import (
	"strings"
	"testing"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/expr"
	"github.com/Nigel2392/go-django/src/core/attrs"
)

func TestQuerySetRawExecution(t *testing.T) {

	var profiles = []*Profile{
		{Name: "TestQuerySetRawExecution 1", Email: "example_1@example.com"},
		{Name: "TestQuerySetRawExecution 2", Email: "example_2@example.com"},
	}

	var users = []*User{
		{Name: "TestQuerySetRawExecution 1", Profile: profiles[0]},
		{Name: "TestQuerySetRawExecution 2", Profile: profiles[1]},
	}

	var todos = []*Todo{
		{Title: "TestQuerySetRow 1", Description: "TestQuerySetRow 1 Description", User: users[0], Done: false},
		{Title: "TestQuerySetRow 2", Description: "TestQuerySetRow 2 Description", User: users[0], Done: true},
		{Title: "TestQuerySetRow 3", Description: "TestQuerySetRow 3 Description", User: users[1], Done: false},
		{Title: "TestQuerySetRow 4", Description: "TestQuerySetRow 4 Description", User: users[1], Done: true},
		{Title: "TestQuerySetRow 5", Description: "TestQuerySetRow 5 Description", User: users[1], Done: false},
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

	_, err = queries.GetQuerySet(&Todo{}).BulkCreate(todos)
	if err != nil {
		t.Fatalf("Failed to create *Todo objects: %v", err)
	}

	t.Run("GENERIC_CATCHALL", func(t *testing.T) {
		var todos_m = make(map[int]*Todo)
		for _, todo := range todos {
			todos_m[todo.ID] = todo
		}

		var query = `SELECT ![p.ID], EXPR(UpperTitle), ![User.ID], ![UserName], ![User.Profile.ID], ![User.Profile.Name], ![User.Profile.Email]
	FROM TABLE(SELF) as p
	INNER JOIN
		TABLE(User) ON ![User.ID] = ![p.User]
	INNER JOIN
		TABLE(User.Profile) ON ![User.Profile.ID] = ![User.Profile]
	WHERE 
		![p.Done] = ?[1] AND 
		EXPR(WhereFilter)
	ORDER BY 
		![p.ID] DESC,
		![User.ID] ASC
	LIMIT ?[2]
	OFFSET ?[3]`

		rows, err := queries.GetQuerySet(&Todo{}).Annotate("UserName", expr.UPPER("User.Name")).Rows(
			query,
			expr.PARSER.Expr.Expressions(map[string]expr.Expression{
				"UpperTitle": expr.UPPER("p.Title"),
				"WhereFilter": expr.Q("p.Title__istartswith", "TestQuerySetRow").And(
					expr.Q("User.ID__in", users),
				),
			}),
			false, 1000, 0,
		)
		if err != nil {
			t.Fatalf("Failed to get rows: %v", err)
		}

		var count = 0
		for rows.Next() {
			var todo = &Todo{
				User: &User{
					Profile: &Profile{},
				},
			}

			var err = rows.Scan(
				&todo.ID, &todo.Title,
				&todo.User.ID, &todo.User.Name,
				&todo.User.Profile.ID, &todo.User.Profile.Name, &todo.User.Profile.Email,
			)
			if err != nil {
				t.Fatalf("Failed to scan row: %v", err)
			}

			var checkTodo, ok = todos_m[todo.ID]
			if !ok {
				t.Fatalf("Todo with ID %d not found in created todos", todo.ID)
			}

			if strings.ToUpper(checkTodo.Title) != todo.Title {
				t.Fatalf("Expected Title %q, got %q", strings.ToUpper(checkTodo.Title), todo.Title)
			}

			if checkTodo.User.ID != todo.User.ID {
				t.Fatalf("Expected User ID %d, got %d", checkTodo.User.ID, todo.User.ID)
			}

			if strings.ToUpper(checkTodo.User.Name) != todo.User.Name {
				t.Fatalf("Expected User Name %q, got %q", strings.ToUpper(checkTodo.User.Name), todo.User.Name)
			}

			t.Logf("Row %d: %+v", count, todo)
			count++
		}

		if count != 3 {
			t.Fatalf("Expected 3 rows, got %d", count)
		}
	})

	_, err = queries.GetQuerySet[attrs.Definer](&Todo{}).Filter("ID__in", todos).Delete()
	if err != nil {
		t.Fatalf("Failed to delete objects: %v", err)
	}

	_, err = queries.GetQuerySet[attrs.Definer](&User{}).Filter("ID__in", users).Delete()
	if err != nil {
		t.Fatalf("Failed to delete objects: %v", err)
	}

	_, err = queries.GetQuerySet[attrs.Definer](&Profile{}).Filter("ID__in", profiles).Delete()
	if err != nil {
		t.Fatalf("Failed to delete objects: %v", err)
	}
}

func TestQuerySetRawExec(t *testing.T) {
	var user = &User{Name: "TestQuerySetRawExec User"}
	if err := queries.CreateObject(user); err != nil {
		t.Fatalf("Failed to create *User: %v", err)
	}

	var todo = &Todo{Title: "Raw Exec Title", Description: "Exec test", User: user, Done: false}
	if err := queries.CreateObject(todo); err != nil {
		t.Fatalf("Failed to create *Todo: %v", err)
	}

	// Exec raw query using expression parser
	res, err := queries.GetQuerySet(&Todo{}).Exec(
		`UPDATE TABLE(SELF) SET done = ?[1], title = ?[2] WHERE id = ?[3]`,
		true, "Updated Raw Exec Title", todo.ID,
	)
	if err != nil {
		t.Fatalf("Failed to Exec raw update: %v", err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected != 1 {
		t.Fatalf("Expected 1 row updated, got %d", rowsAffected)
	}

	// Fetch back and verify
	updatedTodoWrap, err := queries.GetQuerySet[attrs.Definer](&Todo{}).Filter("ID", todo.ID).First()
	if err != nil {
		t.Fatalf("Failed to get updated *Todo: %v", err)
	}

	updatedTodo := updatedTodoWrap.Object.(*Todo)
	if !updatedTodo.Done || updatedTodo.Title != "Updated Raw Exec Title" {
		t.Fatalf("Expected Done=true and Title='Updated Raw Exec Title', got Done=%v, Title='%s'", updatedTodo.Done, updatedTodo.Title)
	}

	// Clean up
	queries.GetQuerySet[attrs.Definer](&Todo{}).Filter("ID", todo.ID).Delete()
	queries.GetQuerySet[attrs.Definer](&User{}).Filter("ID", user.ID).Delete()
}

func TestQuerySetRawRow(t *testing.T) {
	var user = &User{Name: "TestQuerySetRawRow User"}
	if err := queries.CreateObject(user); err != nil {
		t.Fatalf("Failed to create *User: %v", err)
	}

	var todos = []*Todo{
		{Title: "Raw Row 1", Description: "Row test 1", User: user, Done: false},
		{Title: "Raw Row 2", Description: "Row test 2", User: user, Done: true},
	}
	_, err := queries.GetQuerySet(&Todo{}).BulkCreate(todos)
	if err != nil {
		t.Fatalf("Failed to bulk create *Todo objects: %v", err)
	}

	// Raw Row to get count
	var count int
	err = queries.GetQuerySet(&Todo{}).Row(
		`SELECT COUNT(*) FROM TABLE(SELF) WHERE title LIKE ?[1] AND user_id = ?[2]`,
		"Raw Row%", user.ID,
	).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to Row count: %v", err)
	}

	if count != 2 {
		t.Fatalf("Expected count 2, got %d", count)
	}

	// Raw Row to get specific column of a row
	var fetchedTitle string
	err = queries.GetQuerySet(&Todo{}).Row(
		`SELECT title FROM TABLE(SELF) WHERE id = ?[1]`,
		todos[1].ID,
	).Scan(&fetchedTitle)
	if err != nil {
		t.Fatalf("Failed to Row title: %v", err)
	}

	if fetchedTitle != "Raw Row 2" {
		t.Fatalf("Expected fetched title 'Raw Row 2', got '%s'", fetchedTitle)
	}

	// Clean up
	queries.GetQuerySet[attrs.Definer](&Todo{}).Filter("User", user).Delete()
	queries.GetQuerySet[attrs.Definer](&User{}).Filter("ID", user.ID).Delete()
}
