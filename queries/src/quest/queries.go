package quest

import (
	"testing"

	"github.com/Nigel2392/go-django/queries/internal"
	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/src/core/attrs"
)

func CreateObjects[T attrs.Definer](t *testing.T, objects ...T) (created []T, delete func(alreadyDeleted int) error) {
	var err error
	if len(objects) == 0 {
		t.Fatalf("No objects provided for creation")
		return nil, nil
	}

	created, err = queries.GetQuerySet[T](objects[0]).BulkCreate(
		objects,
	)
	if err != nil {
		t.Fatalf("Failed to create objects: %v", err)
		return nil, nil
	}

	if len(created) != len(objects) {
		t.Fatalf("Expected %d objects to be created, got %d", len(objects), len(created))
		return nil, nil
	}

	return created, func(alreadyDeleted int) error {
		var newObj = internal.NewDefiner[T]()
		var deleted, err = queries.GetQuerySet(newObj).Delete(
			created...,
		)

		if err != nil {
			t.Fatalf("Failed to delete objects: %v", err)
			return err
		}

		if int(deleted) != len(created)-alreadyDeleted {
			t.Fatalf("Expected %d objects to be deleted, got %d", len(created), deleted)
			return nil
		}

		return nil
	}
}
