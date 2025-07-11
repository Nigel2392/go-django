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
}
