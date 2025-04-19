package queries_test

import (
	"database/sql"
	"os"
	"testing"

	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/Nigel2392/go-django/src/forms/widgets"
	"github.com/Nigel2392/go-django/src/queries"
)

const (
	createTable = `CREATE TABLE IF NOT EXISTS todos (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	title TEXT,
	description TEXT,
	done BOOLEAN
	)`
	selectTodo = `SELECT id, title, description, done FROM todos WHERE id = ?`
)

type Todo struct {
	ID          int
	Title       string
	Description string
	Done        bool
}

func (m *Todo) FieldDefs() attrs.Definitions {
	return attrs.Define(m,
		attrs.NewField(m, "ID", &attrs.FieldConfig{
			Primary:  true,
			ReadOnly: true,
			Label:    "ID",
			HelpText: "The unique identifier of the model",
		}),
		attrs.NewField(m, "Title", &attrs.FieldConfig{
			Label:    "Title",
			HelpText: "The title of the todo",
		}),
		attrs.NewField(m, "Description", &attrs.FieldConfig{
			Label:    "Description",
			HelpText: "A description of the todo",
			FormWidget: func(cfg attrs.FieldConfig) widgets.Widget {
				return widgets.NewTextarea(nil)
			},
		}),
		attrs.NewField(m, "Done", &attrs.FieldConfig{
			Label:    "Done",
			HelpText: "Indicates whether the todo is done or not",
		}),
	).WithTableName("todos")
}

func init() {
	// make db globally available
	var db, err = sql.Open("sqlite3", ":memory:")
	if err != nil {
		panic(err)
	}
	var settings = map[string]interface{}{
		django.APPVAR_DATABASE: db,
	}

	_, err = db.Exec(createTable)
	if err != nil {
		panic(err)
	}

	logger.Setup(&logger.Logger{
		Level:       logger.DBG,
		OutputDebug: os.Stdout,
		OutputInfo:  os.Stdout,
		OutputWarn:  os.Stdout,
		OutputError: os.Stdout,
	})

	django.App(django.Configure(settings))
}

func TestTodoInsert(t *testing.T) {
	var todos = []*Todo{
		{Title: "Test Todo 1", Description: "Description 1", Done: false},
		{Title: "Test Todo 2", Description: "Description 2", Done: true},
		{Title: "Test Todo 3", Description: "Description 3", Done: false},
	}

	var db = django.ConfigGet[*sql.DB](
		django.Global.Settings,
		django.APPVAR_DATABASE,
	)

	for _, todo := range todos {
		if err := queries.CreateObject(todo); err != nil {
			t.Fatalf("Failed to insert todo: %v", err)
		}

		if todo.ID == 0 {
			t.Fatalf("Expected ID to be set after insert, got 0")
		}

		var row = db.QueryRow(selectTodo, todo.ID)
		var test Todo
		if err := row.Scan(&test.ID, &test.Title, &test.Description, &test.Done); err != nil {
			t.Fatalf("Failed to query todo: %v", err)
		}

		if test.ID != todo.ID || test.Title != todo.Title || test.Description != todo.Description || test.Done != todo.Done {
			t.Fatalf("Inserted todo does not match expected values: got %+v, want %+v", test, todo)
		}

		t.Logf("Inserted todo: %+v", todo)
	}
}

func TestTodoUpdate(t *testing.T) {
	var db = django.ConfigGet[*sql.DB](
		django.Global.Settings,
		django.APPVAR_DATABASE,
	)

	var todo = &Todo{
		ID:          1,
		Title:       "Updated Todo",
		Description: "Updated Description",
		Done:        true,
	}

	if err := queries.UpdateObject(todo); err != nil {
		t.Fatalf("Failed to update todo: %v", err)
	}

	var row = db.QueryRow(selectTodo, todo.ID)
	var test Todo
	if err := row.Scan(&test.ID, &test.Title, &test.Description, &test.Done); err != nil {
		t.Fatalf("Failed to query todo: %v", err)
	}

	if test.ID != todo.ID || test.Title != todo.Title || test.Description != todo.Description || test.Done != todo.Done {
		t.Fatalf("Updated todo does not match expected values: got %+v, want %+v", test, todo)
	}

	t.Logf("Updated todo: %+v", todo)
}

func TestTodoGet(t *testing.T) {
	var db = django.ConfigGet[*sql.DB](
		django.Global.Settings,
		django.APPVAR_DATABASE,
	)

	var todo = &Todo{ID: 1}
	if err := queries.GetObject(todo); err != nil {
		t.Fatalf("Failed to get todo: %v", err)
	}

	var row = db.QueryRow(selectTodo, todo.ID)
	var test Todo
	if err := row.Scan(&test.ID, &test.Title, &test.Description, &test.Done); err != nil {
		t.Fatalf("Failed to query todo: %v", err)
	}

	if test.ID != todo.ID || test.Title != todo.Title || test.Description != todo.Description || test.Done != todo.Done {
		t.Fatalf("Got todo does not match expected values: got %+v, want %+v", test, todo)
	}

	t.Logf("Got todo: %+v", todo)
}

func TestTodoList(t *testing.T) {
	var db = django.ConfigGet[*sql.DB](
		django.Global.Settings,
		django.APPVAR_DATABASE,
	)

	var todos, err = queries.ListObjects(&Todo{}, 0, 1000, "-id")
	if err != nil {
		t.Fatalf("Failed to list todos: %v", err)
	}

	var todoCount = len(todos)
	if len(todos) != 3 {
		t.Fatalf("Expected 3 todos, got %d", todoCount)
	}

	for _, todo := range todos {
		var row = db.QueryRow(selectTodo, todo.ID)
		var test Todo
		if err := row.Scan(&test.ID, &test.Title, &test.Description, &test.Done); err != nil {
			t.Fatalf("Failed to query todo: %v", err)
		}

		if test.ID != todo.ID || test.Title != todo.Title || test.Description != todo.Description || test.Done != todo.Done {
			t.Fatalf("Listed todo does not match expected values: got %+v, want %+v", test, todo)
		}
	}
}

func TestTodoDelete(t *testing.T) {
	var db = django.ConfigGet[*sql.DB](
		django.Global.Settings,
		django.APPVAR_DATABASE,
	)
	var err error
	var todo = &Todo{ID: 1}
	if err = queries.DeleteObject(todo); err != nil {
		t.Fatalf("Failed to delete todo: %v", err)
	}

	var row = db.QueryRow(selectTodo, todo.ID)
	var test Todo
	if err = row.Scan(&test.ID, &test.Title, &test.Description, &test.Done); err == nil {
		t.Fatalf("Expected error when querying deleted todo, got: %v", test)
	}

	t.Logf("Deleted todo: %+v, (%s)", todo, err)
}

func TestTodoCount(t *testing.T) {
	var count, err = queries.CountObjects(&Todo{})
	if err != nil {
		t.Fatalf("Failed to count todos: %v", err)
	}

	if count != 2 {
		t.Fatalf("Expected 2 todos, got %d", count)
	}

	t.Logf("Counted todos: %d", count)
}
