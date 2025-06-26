package queries_test

import (
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

	var query = `SELECT ![ID], ![Title], ![User.ID], ![User.Name], ![User.Profile.ID], ![User.Profile.Name], ![User.Profile.Email]
	FROM TABLE(SELF)
	INNER JOIN
		TABLE(User) ON ![User.ID] = ![User]
	INNER JOIN
		TABLE(User.Profile) ON ![User.Profile.ID] = ![User.Profile]
	WHERE 
		![Done] = ?[1] AND 
		EXPR(WhereFilter)
	ORDER BY 
		![ID] DESC,
		![User.ID] ASC
	LIMIT ?[2]
	OFFSET ?[3]`

	rows, err := queries.GetQuerySet(&Todo{}).Rows(
		query,
		expr.PARSER.Expr.Expressions(map[string]expr.Expression{
			"WhereFilter": expr.Q("Title__istartswith", "TestQuerySetRow").And(
				expr.Q("User__in", users),
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
		t.Logf("Row %d: %+v", count, todo)
		count++
	}

	if count != 3 {
		t.Fatalf("Expected 3 rows, got %d", count)
	}

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
