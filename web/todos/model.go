package todos

import (
	"context"

	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/forms/widgets"
)

const (
	createTable = `CREATE TABLE IF NOT EXISTS todos (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        title TEXT,
        description TEXT,
        done BOOLEAN
    )`
	listTodos  = `SELECT id, title, description, done FROM todos ORDER BY id DESC LIMIT ? OFFSET ?`
	insertTodo = `INSERT INTO todos (title, description, done) VALUES (?, ?, ?)`
	updateTodo = `UPDATE todos SET title = ?, description = ?, done = ? WHERE id = ?`
	selectTodo = `SELECT id, title, description, done FROM todos WHERE id = ?`
	countTodos = `SELECT COUNT(id) FROM todos`
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
	)
}

// Save is a method that will either insert or update the todo in the database.
//
// If the todo has an ID of 0, it will be inserted into the database; otherwise, it will be updated.
//
// This method should exist on all models that need to be saved to the database.
func (t *Todo) Save(ctx context.Context) error {
	if t.ID == 0 {
		return t.Insert(ctx)
	}
	return t.Update(ctx)
}

// Not Required
func (t *Todo) Insert(ctx context.Context) error {
	var res, err = globalDB.ExecContext(ctx, insertTodo, t.Title, t.Description, t.Done)
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	t.ID = int(id)
	return nil
}

// Not Required
func (t *Todo) Update(ctx context.Context) error {
	_, err := globalDB.ExecContext(ctx, updateTodo, t.Title, t.Description, t.Done, t.ID)
	return err
}
