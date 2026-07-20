package quest

import (
	"context"
	"reflect"

	"github.com/Nigel2392/go-django/djester"
	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/src/core/attrs"
)

func CreateObjects[T attrs.Definer](t djester.BaseTB, objects ...T) (created []T, delete func(alreadyDeleted int) error) {
	var err error
	if len(objects) == 0 {
		t.Fatalf("No objects provided for creation")
		return nil, nil
	}

	var objType = reflect.TypeOf(objects[0])
	created, err = queries.
		GetQuerySet(objects[0]).
		WithContext(drivers.SetLogSQLContext(context.Background(), false)).
		BulkCreate(
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
		var newObj = attrs.NewObject[T](context.Background(), objType)
		var deleted, err = queries.
			GetQuerySet(newObj).
			WithContext(drivers.SetLogSQLContext(context.Background(), false)).
			Delete(
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
