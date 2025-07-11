package queries_test

import (
	"testing"

	queries "github.com/Nigel2392/go-django/queries/src"
)

func TestUniqueInsert(t *testing.T) {
	uniqueSource, err := queries.GetQuerySet(&UniqueSource{}).
		Create(&UniqueSource{
			Name: "Unique Source 1",
		})
	if err != nil {
		t.Fatalf("Failed to create unique source: %v", err)
	}

	_, err = queries.GetQuerySet(&UniqueSource{}).
		Create(&UniqueSource{
			Name: uniqueSource.Name,
		})
	if err == nil {
		t.Fatalf("Expected error when creating duplicate unique source, got none")
	} else {
		t.Logf("Expected (and received) error when creating duplicate unique source: %v", err)
	}

	_, err = queries.GetQuerySet(&UniqueSource{}).Delete()
	if err != nil {
		t.Fatalf("Failed to delete unique source: %v", err)
	}
}

func TestUniqueUpdate(t *testing.T) {
	uniqueSources, err := queries.GetQuerySet(&UniqueSource{}).
		BulkCreate([]*UniqueSource{
			{Name: "Unique Source 1"},
			{Name: "Unique Source 2"},
		})

	if err != nil {
		t.Fatalf("Failed to create unique sources: %v", err)
	}

	uniqueSource1 := uniqueSources[0]
	uniqueSource2 := uniqueSources[1]

	updated, err := queries.GetQuerySet(&UniqueSource{}).
		Update(&UniqueSource{
			ID:   uniqueSource2.ID,
			Name: uniqueSource1.Name,
		})
	if err == nil {
		t.Fatalf("Expected error when updating unique source to duplicate name, got none")
	} else {
		t.Logf("Expected (and received) error when updating unique source to duplicate name: %v", err)
	}

	if updated != 0 {
		t.Fatalf("Expected no rows to be updated, got %d", updated)
	}

	_, err = queries.GetQuerySet(&UniqueSource{}).Delete()
	if err != nil {
		t.Fatalf("Failed to delete unique source: %v", err)
	}
}
