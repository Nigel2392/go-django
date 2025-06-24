package queries_test

import (
	"context"
	"os"
	"testing"

	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/Nigel2392/go-django/src/models"
	"github.com/pkg/errors"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/query_errors"
)

func init() {
	var db, err = drivers.Open(context.Background(), "sqlite3", "file:queries_model_hooks_test?mode=memory&cache=shared")
	if err != nil {
		panic(err)
	}
	var settings = map[string]interface{}{
		django.APPVAR_DATABASE: db,
	}

	_, err = db.ExecContext(context.Background(), createTableSQLite)
	if err != nil {
		panic(err)
	}

	attrs.RegisterModel(&User{})

	logger.Setup(&logger.Logger{
		Level:       logger.DBG,
		WrapPrefix:  logger.ColoredLogWrapper,
		OutputDebug: os.Stdout,
		OutputInfo:  os.Stdout,
		OutputWarn:  os.Stdout,
		OutputError: os.Stdout,
	})

	django.App(django.Configure(settings))

}

var createTableSQLite = `CREATE TABLE IF NOT EXISTS user (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	name TEXT NOT NULL,
	email TEXT NOT NULL,
	age INTEGER NOT NULL,
	is_active BOOLEAN NOT NULL DEFAULT 1,
	first_name TEXT NOT NULL,
	last_name TEXT NOT NULL
);`

type User struct {
	ID        int64  `attrs:"primary"`
	Name      string `attrs:"max_length=255"`
	Email     string `attrs:"max_length=255"`
	Age       int32  `attrs:"min_value=0;max_value=120"`
	IsActive  bool   `attrs:"default=true"`
	FirstName string `attrs:"label=First Name"`
	LastName  string `attrs:"label=Last Name"`
}

func (m *User) FieldDefs() attrs.Definitions {
	return attrs.AutoDefinitions(m)
}

func TestModelsSave(t *testing.T) {
	var saved, err = models.SaveModel(context.Background(), &User{
		Name:      "John Doe",
		Email:     "test@example.com",
		Age:       30,
		IsActive:  true,
		FirstName: "John",
		LastName:  "Doe",
	})
	if err != nil {
		t.Fatalf("failed to save model: %v", err)
	}

	if !saved {
		t.Fatalf("model not saved")
	}

	fromDb, err := queries.Objects[*User](&User{}).
		Filter("Name", "John Doe").
		Filter("Email", "test@example.com").
		Filter("Age", 30).
		Get()
	if err != nil {
		t.Fatalf("failed to get model: %v", err)
	}

	if fromDb == nil {
		t.Fatalf("model not found in db")
	}

	var user = fromDb.Object
	if user.Name != "John Doe" {
		t.Fatalf("expected name to be 'John Doe', got '%s'", user.Name)
	}

	if user.Email != "test@example.com" {
		t.Fatalf("expected email to be 'test@example.com', got '%s'", user.Email)
	}

	if user.Age != 30 {
		t.Fatalf("expected age to be 30, got %d", user.Age)
	}

	if user.IsActive != true {
		t.Fatalf("expected is_active to be true, got %v", user.IsActive)
	}

	if user.FirstName != "John" {
		t.Fatalf("expected first_name to be 'John', got '%s'", user.FirstName)
	}

	if user.LastName != "Doe" {
		t.Fatalf("expected last_name to be 'Doe', got '%s'", user.LastName)
	}

	_, err = queries.Objects[*User](&User{}).
		Filter("Name", "John Doe").
		Filter("Email", "test@example.com").
		Delete()
	if err != nil {
		t.Fatalf("failed to delete model: %v", err)
	}
}

func TestModelsDelete(t *testing.T) {
	var user = &User{
		Name:      "John Doe",
		Email:     "test@example.com",
		Age:       30,
		IsActive:  true,
		FirstName: "John",
		LastName:  "Doe",
	}

	var saved, err = models.SaveModel(context.Background(), user)
	if err != nil {
		t.Fatalf("failed to save model: %v", err)
	}

	if !saved {
		t.Fatalf("model not saved")
	}

	if user.ID == 0 {
		t.Fatalf("model ID is 0")
	}

	obj, err := queries.Objects[*User](&User{}).
		Filter("ID", user.ID).
		Get()
	if err != nil {
		t.Fatalf("failed to get model: %v", err)
	}
	if *obj.Object != *user {
		t.Fatalf("model not found in db")
	}

	deleted, err := models.DeleteModel(context.Background(), user)
	if err != nil {
		t.Fatalf("failed to delete model: %v", err)
	}

	if !deleted {
		t.Fatalf("model not deleted")
	}

	_, err = queries.Objects[*User](&User{}).
		Filter("ID", user.ID).
		Get()
	if !errors.Is(err, query_errors.ErrNoRows) {
		t.Fatalf("expected no rows error, got: %v", err)
	}
}

// TestContentTypesDefinitions tests the content types definitions for the User model.
//
// These content type definitions are not defined anywhere by the developer - they are
// automatically registered in queries_init.go
func TestContentTypesDefinitions(t *testing.T) {
	var users = []*User{
		{Name: "John Doe", Email: "test@example.com1", Age: 30},
		{Name: "Jane Doe", Email: "test@example.com2", Age: 25},
		{Name: "Jim Doe", Email: "test@example.com3", Age: 35},
		{Name: "Jack Doe", Email: "test@example.com4", Age: 40},
	}

	for _, user := range users {
		var saved, err = models.SaveModel(context.Background(), user)
		if err != nil {
			t.Fatalf("failed to save model: %v", err)
		}

		if !saved {
			t.Fatalf("model not saved")
		}
	}

	var cTypeDef = contenttypes.DefinitionForObject(&User{})
	if cTypeDef == nil {
		t.Fatalf("failed to get content type definition")
	}

	t.Run("TestInstance", func(t *testing.T) {
		var objectByID, err = cTypeDef.Instance(users[2].ID)
		if err != nil {
			t.Fatalf("failed to get object by ID: %v", err)
		}

		if objectByID == nil {
			t.Fatalf("object by ID is nil")
		}

		if *(objectByID.(*User)) != *users[2] {
			t.Fatalf("object by ID does not match: expected %+v, got %+v", *users[2], *(objectByID.(*User)))
		}
	})

	t.Run("TestInstancesByIDs", func(t *testing.T) {
		// Test InstancesByIDs method
		var ids = []any{users[1].ID, users[2].ID}
		objectsByID, err := cTypeDef.InstancesByIDs(ids)
		if err != nil {
			t.Fatalf("failed to get objects by IDs: %v", err)
		}

		if len(objectsByID) != len(ids) {
			t.Fatalf("expected %d objects, got %d", len(ids), len(objectsByID))
		}

		for i := range ids {
			if *(objectsByID[i].(*User)) != *users[i+1] {
				t.Fatalf("object by ID does not match")
			}
		}
	})

	t.Run("TestInstances", func(t *testing.T) {
		// Test Instances method
		instances, err := cTypeDef.Instances(3, 0)
		if err != nil {
			t.Fatalf("failed to get instances: %v", err)
		}

		if len(instances) != 3 {
			t.Fatalf("expected 3 instances, got %d", len(instances))
		}

		for i := range instances {
			if *(instances[i].(*User)) != *users[i] {
				t.Fatalf("instance does not match")
			}
		}
	})
}
