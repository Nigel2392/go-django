package queries_test

import (
	"testing"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/quest"
	"github.com/Nigel2392/go-django/src/core/attrs"
)

type UniqueModel struct {
	Email     string
	FirstName string
	LastName  string
}

func (u *UniqueModel) UniqueTogether() [][]string {
	return [][]string{
		{"Email"},
		{"FirstName", "LastName"},
	}
}

func (u *UniqueModel) FieldDefs() attrs.Definitions {
	return attrs.Define(u, attrs.AutoFieldList(u))
}

func TestUniqueModel(t *testing.T) {
	var tables = quest.Table(t,
		&UniqueModel{},
	)

	tables.Create()
	defer tables.Drop()

	var objects = []*UniqueModel{
		{Email: "john@example.com", FirstName: "John", LastName: "Doe"},
		{Email: "joe@example.com", FirstName: "Joe", LastName: "Doe"},
		{Email: "jane@example.com", FirstName: "Jane", LastName: "Doe"},
	}

	var err error
	_, err = queries.GetQuerySet(&UniqueModel{}).BulkCreate(objects)
	if err != nil {
		t.Fatalf("Failed to create objects: %v", err)
	}

	updated, err := queries.GetQuerySet(&UniqueModel{}).
		Select("FirstName").
		Update(&UniqueModel{
			Email:     "john@example.com",
			FirstName: "Updated",
		})
	if err != nil {
		t.Fatalf("Failed to update objects: %v", err)
	}

	if updated != 1 {
		t.Fatalf("Expected 1 object to be updated, got %d", updated)
	}

	updatedObject, err := queries.GetQuerySet(&UniqueModel{}).
		Filter("Email", "john@example.com").
		First()
	if err != nil {
		t.Fatalf("Failed to retrieve updated object: %v", err)
	}

	if updatedObject == nil {
		t.Fatal("Expected to retrieve an updated object, but got nil")
	}

	if updatedObject.Object.Email != "john@example.com" {
		t.Fatalf("Expected Email to be 'john@example.com', got '%s'", updatedObject.Object.Email)
	}

	if updatedObject.Object.FirstName != "Updated" {
		t.Fatalf("Expected FirstName to be 'Updated', got '%s'", updatedObject.Object.FirstName)
	}
}
