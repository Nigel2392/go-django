package queries_test

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"

	"github.com/Nigel2392/go-django/queries/internal"
	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/queries/src/expr"
	"github.com/Nigel2392/go-django/queries/src/fields"
	"github.com/Nigel2392/go-django/queries/src/models"
	"github.com/Nigel2392/go-django/queries/src/quest"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/djester/testdb"
	"github.com/Nigel2392/go-django/src/forms/widgets"
	"github.com/jmoiron/sqlx"
)

func selectTodo(db drivers.Database) string {
	const query = `SELECT id, title, description, done, user_id FROM %s%s%s WHERE id = ?`
	var quote = "`"
	switch internal.SqlxDriverName(db) {
	case "mysql", "mariadb", "sqlite3":
		quote = "`"
	case "postgres":
		quote = "\""
	}
	return sqlx.Rebind(sqlx.BindType(internal.SqlxDriverName(db)), fmt.Sprintf(query, quote, "queries-todos", quote))
}

type Image struct {
	ID   int
	Path string
}

func (m *Image) FieldDefs() attrs.Definitions {
	return attrs.Define(m,
		attrs.NewField(m, "ID", &attrs.FieldConfig{
			Primary:  true,
			ReadOnly: true,
		}),
		attrs.NewField(m, "Path", &attrs.FieldConfig{}),
	).WithTableName("queries-images")
}

type Profile struct {
	models.Model
	ID    int
	Name  string
	Email string
	Image *Image
}

func (m *Profile) FieldDefs() attrs.Definitions {
	return m.Model.Define(m,
		attrs.NewField(m, "ID", &attrs.FieldConfig{
			Primary:  true,
			ReadOnly: true,
		}),
		attrs.NewField(m, "Name", &attrs.FieldConfig{}),
		attrs.NewField(m, "Email", &attrs.FieldConfig{}),
		attrs.NewField(m, "Image", &attrs.FieldConfig{
			Null:          true,
			RelForeignKey: attrs.Relate(&Image{}, "", nil),
			Column:        "image_id",
		}),
	).WithTableName("queries-profiles")
}

type User struct {
	models.Model
	ID                 int
	Name               string
	Profile            *Profile
	ModelManyToManySet *queries.RelRevFK[attrs.Definer]
}

func (m *User) FieldDefs() attrs.Definitions {
	return m.Model.Define(m,
		attrs.NewField(m, "ID", &attrs.FieldConfig{
			Primary:  true,
			ReadOnly: true,
		}),
		attrs.NewField(m, "Name", &attrs.FieldConfig{}),
		attrs.NewField(m, "Profile", &attrs.FieldConfig{
			Null:        true,
			RelOneToOne: attrs.Relate(&Profile{}, "", nil),
			Column:      "profile_id",
			Attributes: map[string]interface{}{
				attrs.AttrReverseAliasKey: "User",
			},
		}),
	).WithTableName("queries-users")
}

type Related struct {
	Object        attrs.Definer
	ThroughObject attrs.Definer
}

func (r *Related) Model() attrs.Definer {
	return r.Object
}

func (r *Related) Through() attrs.Definer {
	return r.ThroughObject
}

type Todo struct {
	models.Model `table:"queries-todos"`
	ID           int
	Title        string
	Description  string
	Done         bool
	User         *User
}

func (m *Todo) FieldDefs() attrs.Definitions {
	return m.Model.Define(m,
		attrs.NewField(m, "ID", &attrs.FieldConfig{
			Column:   "id", // can be inferred, but explicitly set for clarity
			Primary:  true,
			ReadOnly: true,
		}),
		attrs.NewField(m, "Title", &attrs.FieldConfig{
			Column: "title", // can be inferred, but explicitly set for clarity
		}),
		attrs.NewField(m, "Description", &attrs.FieldConfig{
			Column: "description", // can be inferred, but explicitly set for clarity
			FormWidget: func(cfg attrs.FieldConfig) widgets.Widget {
				return widgets.NewTextarea(nil)
			},
		}),
		attrs.NewField(m, "Done", &attrs.FieldConfig{}),
		attrs.NewField(m, "User", &attrs.FieldConfig{
			Null:        true,
			Column:      "user_id",
			RelOneToOne: attrs.Relate(&User{}, "", nil),
		}),
	)
}

type ObjectWithMultipleRelations struct {
	ID   int
	Obj1 *User
	Obj2 *User
}

func (m *ObjectWithMultipleRelations) FieldDefs() attrs.Definitions {
	return attrs.Define(m,
		attrs.NewField(m, "ID", &attrs.FieldConfig{
			Primary:  true,
			ReadOnly: true,
		}),
		attrs.NewField(m, "Obj1", &attrs.FieldConfig{
			Null:          true,
			RelForeignKey: attrs.Relate(&User{}, "", nil),
			Column:        "obj1_id",
			Attributes: map[string]any{
				attrs.AttrReverseAliasKey: "MultiRelationObj1",
			},
		}),
		attrs.NewField(m, "Obj2", &attrs.FieldConfig{
			Null:          true,
			RelForeignKey: attrs.Relate(&User{}, "", nil),
			Column:        "obj2_id",
			Attributes: map[string]any{
				attrs.AttrReverseAliasKey: "MultiRelationObj2",
			},
		}),
	).WithTableName("queries-object_with_multiple_relations")
}

type Category struct {
	models.Model
	ID     int
	Name   string
	Parent *Category
}

func (m *Category) FieldDefs() attrs.Definitions {
	return m.Model.Define(m,
		attrs.NewField(m, "ID", &attrs.FieldConfig{
			Primary:  true,
			ReadOnly: true,
		}),
		attrs.NewField(m, "Name", &attrs.FieldConfig{}),
		attrs.NewField(m, "Parent", &attrs.FieldConfig{
			Null:          true,
			Column:        "parent_id",
			RelForeignKey: attrs.Relate(&Category{}, "", nil),
		}),
	).WithTableName("queries-categories")
}

type OneToOneWithThrough struct {
	models.Model
	ID      int64
	Title   string
	Through *queries.RelO2O[*OneToOneWithThrough_Target, *OneToOneWithThrough_Through]
	User    *User
}

func (t *OneToOneWithThrough) FieldDefs() attrs.Definitions {
	return t.Model.Define(t,
		attrs.NewField(t, "ID", &attrs.FieldConfig{
			Column:  "id",
			Primary: true,
		}),
		attrs.NewField(t, "Title", &attrs.FieldConfig{
			Column: "title",
		}),
		//  fields.NewOneToOneField[*queries.RelO2O[*OneToOneWithThrough_Target, *OneToOneWithThrough_Through]](t, &t.Through, "Target", "TargetReverse", "id", attrs.Relate(
		//  	&OneToOneWithThrough_Target{},
		//  	"", &attrs.ThroughModel{
		//  		This:   &OneToOneWithThrough_Through{},
		//  		Source: "SourceModel",
		//  		Target: "TargetModel",
		//  	},
		//  )),
		fields.NewOneToOneField[*queries.RelO2O[*OneToOneWithThrough_Target, *OneToOneWithThrough_Through]](t, "Target", &fields.FieldConfig{
			ScanTo:      &t.Through,
			ReverseName: "TargetReverse",
			Rel: attrs.Relate(
				&OneToOneWithThrough_Target{},
				"", &attrs.ThroughModel{
					This:   &OneToOneWithThrough_Through{},
					Source: "SourceModel",
					Target: "TargetModel",
				},
			),
		}),

		attrs.NewField(t, "User", &attrs.FieldConfig{
			Null:          true,
			Column:        "user_id",
			RelForeignKey: attrs.Relate(&User{}, "", nil),
		}),
	).WithTableName("queries-onetoonewiththrough")
}

type OneToOneWithThrough_Through struct {
	models.Model
	SourceModel *OneToOneWithThrough
	TargetModel *OneToOneWithThrough_Target
}

func (t *OneToOneWithThrough_Through) FieldDefs() attrs.Definitions {
	return t.Model.Define(t,
		attrs.NewField(t, "SourceModel", &attrs.FieldConfig{
			Column: "source_id",
			Null:   false,
		}),
		attrs.NewField(t, "TargetModel", &attrs.FieldConfig{
			Column: "target_id",
			Null:   false,
		}),
	).WithTableName("queries-onetoonewiththrough_through")
}

type OneToOneWithThrough_Target struct {
	models.Model
	ID   int64
	Name string
	Age  int
}

func (t *OneToOneWithThrough_Target) FieldDefs() attrs.Definitions {
	return t.Model.Define(t,
		attrs.NewField(t, "ID", &attrs.FieldConfig{
			Column:  "id",
			Primary: true,
		}),
		attrs.NewField(t, "Name", &attrs.FieldConfig{
			Column: "name",
		}),
		attrs.NewField(t, "Age", &attrs.FieldConfig{
			Column: "age",
		}),
	).WithTableName("queries-onetoonewiththrough_target")
}

type ModelManyToMany struct {
	models.Model
	ID    int64
	Title string
	User  *User
	// Target []*ModelManyToMany_Target
	Target *queries.RelM2M[*ModelManyToMany_Target, *ModelManyToMany_Through]
}

// func (m *ModelManyToMany) String() string {
// 	var sb strings.Builder
// 	sb.WriteString("ModelManyToMany{")
// 	sb.WriteString("ID: ")
// 	fmt.Fprintf(&sb, "%d", m.ID)
// 	sb.WriteString(", Model: ")
// 	fmt.Fprintf(&sb, "%v", m.Model)
// 	sb.WriteString(", User: ")
// 	if m.User != nil {
// 		fmt.Fprintf(&sb, "%d", m.User.ID)
// 	} else {
// 		fmt.Fprintf(&sb, "nil")
// 	}
// 	sb.WriteString("}")
// 	return sb.String()
// }

func (t *ModelManyToMany) FieldDefs() attrs.Definitions {
	return t.Model.Define(t,
		attrs.NewField(t, "ID", &attrs.FieldConfig{
			Column:  "id",
			Primary: true,
		}),
		attrs.NewField(t, "Title", &attrs.FieldConfig{
			Column: "title",
			Attributes: map[string]any{
				attrs.AttrUniqueKey: true,
			},
			MaxLength: 255,
		}),
		//	//)),fields.NewManyToManyField[*queries.RelM2M[*ModelManyToMany_Target, *ModelManyToMany_Through]](t, &t.Target, "Target", "TargetReverse", "id", attrs.Relate(
		//	//)),	&ModelManyToMany_Target{},
		//	//)),	"", &attrs.ThroughModel{
		//	//)),		This:   &ModelManyToMany_Through{},
		//	//)),		Source: "SourceModel",
		//	//)),		Target: "TargetModel",
		//	//)),	},
		//	//)),
		fields.NewManyToManyField[*queries.RelM2M[*ModelManyToMany_Target, *ModelManyToMany_Through]](t, "Target", &fields.FieldConfig{
			ScanTo:      &t.Target,
			ReverseName: "TargetReverse",
			Rel: attrs.Relate(
				&ModelManyToMany_Target{},
				"", &attrs.ThroughModel{
					This:   &ModelManyToMany_Through{},
					Source: "SourceModel",
					Target: "TargetModel",
				},
			),
		}),
		// attrs.NewField(t, "Target", &attrs.FieldConfig{
		// RelManyToMany: attrs.Relate(&ModelManyToMany_Target{}, "", &attrs.ThroughModel{
		// This:   &ModelManyToMany_Through{},
		// Source: "SourceModel",
		// Target: "TargetModel",
		// }),
		// Attributes: map[string]any{
		// attrs.AttrReverseAliasKey: "TargetReverse",
		// },
		// }),
		// fields.ForeignKey[*User]("User", "user_id", &fields.FieldConfig{
		// ReverseName: "ModelManyToManySet",
		// }),
		attrs.NewField(t, "User", &attrs.FieldConfig{
			Null:          true,
			Column:        "user_id",
			RelForeignKey: attrs.Relate(&User{}, "", nil),
		}),
	).WithTableName("queries-model_manytomany")
}

type ModelManyToMany_Through struct {
	models.Model
	SourceModel *ModelManyToMany
	TargetModel *ModelManyToMany_Target
}

// func (m *ModelManyToMany_Through) String() string {
// 	var sb strings.Builder
// 	sb.WriteString("ModelManyToMany_Through{")
// 	sb.WriteString("ID: ")
// 	fmt.Fprintf(&sb, "%d", m.ID)
// 	sb.WriteString(", Model: ")
// 	fmt.Fprintf(&sb, "%v", m.Model)
// 	sb.WriteString(", SourceModel: ")
// 	fmt.Fprintf(&sb, "%T", m.SourceModel)
// 	sb.WriteString(", TargetModel: ")
// 	fmt.Fprintf(&sb, "%T", m.TargetModel)
// 	sb.WriteString("}")
// 	return sb.String()
// }

func (t *ModelManyToMany_Through) FieldDefs() attrs.Definitions {
	return t.Model.Define(t,
		attrs.NewField(t, "SourceModel", &attrs.FieldConfig{
			Column: "source_id",
			Null:   false,
		}),
		attrs.NewField(t, "TargetModel", &attrs.FieldConfig{
			Column: "target_id",
			Null:   false,
		}),
	).WithTableName("queries-model_manytomany_through")
}

type ModelManyToMany_Target struct {
	models.Model
	ID   int64
	Name string
	Age  int
}

var _ queries.UniqueTogetherDefiner = (*ModelManyToMany_Target)(nil)

func (t *ModelManyToMany_Target) UniqueTogether() [][]string {
	// This method is used to define unique constraints on the ModelManyToMany_Target model.
	// In this case, we are defining a unique constraint on the Name field.
	return [][]string{
		{"Name"},
	}
}

// func (t *ModelManyToMany_Target) String() string {
// 	return fmt.Sprintf("ModelManyToMany_Target(ID=%d, Model=%v)", t.ID, t.Model)
// }

func (t *ModelManyToMany_Target) FieldDefs() attrs.Definitions {
	return t.Model.Define(t,
		attrs.NewField(t, "ID", &attrs.FieldConfig{
			Column:  "id",
			Primary: true,
		}),
		attrs.NewField(t, "Name", &attrs.FieldConfig{
			Column: "name",
		}),
		attrs.NewField(t, "Age", &attrs.FieldConfig{
			Column: "age",
		}),
	).WithTableName("queries-model_manytomany_target")
}

type TestTransaction struct {
	models.Model
	ID   int64
	Name string
}

func (t *TestTransaction) FieldDefs() attrs.Definitions {
	return t.Model.Define(t,
		attrs.NewField(t, "ID", &attrs.FieldConfig{
			Column:  "id",
			Primary: true,
		}),
		attrs.NewField(t, "Name", &attrs.FieldConfig{
			Column: "name",
		}),
	).WithTableName("queries-test_transaction")
}

func init() {

	// create tables
	var tables = quest.Table[*testing.T](nil,
		&Image{},
		&Profile{},
		&User{},
		&ObjectWithMultipleRelations{},
		&Category{},
		&Todo{},
		&OneToOneWithThrough{},
		&OneToOneWithThrough_Target{},
		&OneToOneWithThrough_Through{},
		&ModelManyToMany{},
		&ModelManyToMany_Target{},
		&ModelManyToMany_Through{},

		&TestStruct{},
		&TestStructNoObject{},
		&Author{},
		&Book{},

		&TestTransaction{},
	)

	// Reset the definitions to ensure all models are registered
	// before reverse fields are fully setup.
	attrs.ResetDefinitions.Send(nil)

	tables.Create()
}

func TestTodoInsert(t *testing.T) {
	var todos = []*Todo{
		{Title: "Test Todo 1", Description: "Description 1", Done: false},
		{Title: "Test Todo 2", Description: "Description 2", Done: true},
		{Title: "Test Todo 3", Description: "Description 3", Done: false},
	}

	var db = django.ConfigGet[drivers.Database](
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

		var selectStmt = selectTodo(db)
		var row = db.QueryRowContext(context.Background(), selectStmt, todo.ID)
		var test = &Todo{User: &User{}}
		if err := row.Scan(&test.ID, &test.Title, &test.Description, &test.Done, &sql.NullInt64{}); err != nil {
			t.Fatalf("Failed to query todo: %v", err)
		}

		if test.ID != todo.ID || test.Title != todo.Title || test.Description != todo.Description || test.Done != todo.Done {
			t.Fatalf("Inserted todo does not match expected values: got %+v, want %+v", test, todo)
		}

		t.Logf("Inserted todo: %+v", todo)
	}
}

func TestTodoUpdate(t *testing.T) {
	var db = django.ConfigGet[drivers.Database](
		django.Global.Settings,
		django.APPVAR_DATABASE,
	)

	var todo = &Todo{
		ID:          1,
		Title:       "Updated Todo",
		Description: "Updated Description",
		Done:        true,
	}

	if updated, err := queries.UpdateObject(todo); err != nil {
		t.Fatalf("Failed to update todo: %v", err)
	} else if updated == 0 {
		t.Fatalf("Expected 1 todo to be updated, got %d", updated)
	}

	var selectStmt = selectTodo(db)
	var row = db.QueryRowContext(context.Background(), selectStmt, todo.ID)
	var test Todo = Todo{User: &User{}}
	if err := row.Scan(&test.ID, &test.Title, &test.Description, &test.Done, &sql.NullInt64{}); err != nil {
		t.Fatalf("Failed to query todo: %v", err)
	}

	if test.ID != todo.ID || test.Title != todo.Title || test.Description != todo.Description || test.Done != todo.Done {
		t.Fatalf("Updated todo does not match expected values: got %+v, want %+v", test, todo)
	}

	t.Logf("Updated todo: %+v", todo)
}

func TestTodoGet(t *testing.T) {
	var db = django.ConfigGet[drivers.Database](
		django.Global.Settings,
		django.APPVAR_DATABASE,
	)

	var todo, err = queries.GetObject[*Todo](&Todo{}, 1)
	if err != nil {
		t.Fatalf("Failed to get todo: %v", err)
	}

	var selectStmt = selectTodo(db)
	var row = db.QueryRowContext(context.Background(), selectStmt, todo.ID)
	var test Todo = Todo{User: &User{}}
	if err := row.Scan(&test.ID, &test.Title, &test.Description, &test.Done, &sql.NullInt64{}); err != nil {
		t.Fatalf("Failed to query todo: %v", err)
	}

	if test.ID != todo.ID || test.Title != todo.Title || test.Description != todo.Description || test.Done != todo.Done {
		t.Fatalf("Got todo does not match expected values: got %+v, want %+v", test, todo)
	}

	t.Logf("Got todo: %+v", todo)
}

func TestTodoList(t *testing.T) {
	var db = django.ConfigGet[drivers.Database](
		django.Global.Settings,
		django.APPVAR_DATABASE,
	)

	var todos, err = queries.ListObjects[*Todo](&Todo{}, 0, 1000, "-ID")
	if err != nil {
		t.Fatalf("Failed to list todos: %v", err)
	}

	var todoCount = len(todos)
	if len(todos) != 3 {
		t.Fatalf("Expected 3 todos, got %d", todoCount)
	}

	for _, todo := range todos {
		var selectStmt = selectTodo(db)
		var row = db.QueryRowContext(context.Background(), selectStmt, todo.ID)
		var test Todo = Todo{User: &User{}}
		if err := row.Scan(&test.ID, &test.Title, &test.Description, &test.Done, &sql.NullInt64{}); err != nil {
			t.Fatalf("Failed to query todo: %v", err)
		}

		if test.ID != todo.ID || test.Title != todo.Title || test.Description != todo.Description || test.Done != todo.Done {
			t.Fatalf("Listed todo does not match expected values: got %+v, want %+v", test, todo)
		}

		t.Logf("Listed todo: %+v", todo)
	}
}

func TestListTodoByIDs(t *testing.T) {
	var db = django.ConfigGet[drivers.Database](
		django.Global.Settings,
		django.APPVAR_DATABASE,
	)

	var ids = []int{1, 2}
	var todos, err = queries.ListObjectsByIDs[*Todo](&Todo{}, 0, 1000, ids)
	if err != nil {
		t.Fatalf("Failed to get todos by IDs: %v", err)
	}

	var todoCount = len(todos)
	if todoCount != len(ids) {
		t.Fatalf("Expected %d todos, got %d", len(ids), todoCount)
	}

	for _, todo := range todos {
		var selectStmt = selectTodo(db)
		var row = db.QueryRowContext(context.Background(), selectStmt, todo.ID)
		var test Todo = Todo{User: &User{}}
		if err := row.Scan(&test.ID, &test.Title, &test.Description, &test.Done, &sql.NullInt64{}); err != nil {
			t.Fatalf("Failed to query todo: %v", err)
		}

		if test.ID != todo.ID || test.Title != todo.Title || test.Description != todo.Description || test.Done != todo.Done {
			t.Fatalf("Got todo does not match expected values: got %+v, want %+v", test, todo)
		}

		t.Logf("Got todo by ID: %+v", todo)
	}
}

func TestTodoDelete(t *testing.T) {
	var db = django.ConfigGet[drivers.Database](
		django.Global.Settings,
		django.APPVAR_DATABASE,
	)
	var err error
	var todo = &Todo{ID: 1}
	if deleted, err := queries.DeleteObject(todo); err != nil {
		t.Fatalf("Failed to delete todo: %v", err)
	} else if deleted == 0 {
		t.Fatalf("Expected 1 todo to be deleted, got %d", deleted)
	}

	var selectStmt = selectTodo(db)
	var row = db.QueryRowContext(context.Background(), selectStmt, todo.ID)
	var test Todo = Todo{User: &User{}}
	if err = row.Scan(&test.ID, &test.Title, &test.Description, &test.Done, &sql.NullInt64{}); err == nil {
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

func TestQuerySet_Filter(t *testing.T) {
	var query = queries.GetQuerySet[attrs.Definer](&Todo{}).
		Select("ID", "Title", "Description", "Done").
		Filter("Title__icontains", "test").
		Filter("Done", false).
		Filter("User__isnull", true).
		OrderBy("-ID").
		Limit(5)

	if query == nil {
		t.Fatalf("Expected query to be not nil")
	}

	if query.Meta().Model() == nil {
		t.Fatalf("Expected query model to be not nil")
	}

	todos, err := query.All()

	if err != nil {
		t.Fatalf("Failed to filter todos: %v", err)
	}

	if len(todos) != 1 {
		t.Fatalf("Expected 1 todos, got %d: %v", len(todos), query.LatestQuery().Args())
	}

	for _, todo := range todos {
		todo := todo.Object.(*Todo)
		if !strings.Contains(strings.ToLower(todo.Title), "test") {
			t.Fatalf("Expected todo title to contain 'test', got: %s", todo.Title)
		}

		if todo.Done {
			t.Fatalf("Expected todo to be not done, got done: %+v", todo)
		}
	}

	t.Logf("Filtered todos: %+v", todos)
}

func TestQuerySet_First(t *testing.T) {
	todo, err := queries.GetQuerySet[attrs.Definer](&Todo{}).
		Select("ID", "Title", "Description", "Done").
		Filter("Done", true).
		First()

	if err != nil {
		t.Fatalf("Failed to get first todo: %v", err)
	}

	if todo == nil {
		t.Fatalf("Expected a todo, got nil")
	}

	var tdo = todo.Object.(*Todo)
	if !tdo.Done {
		t.Fatalf("Expected todo to be done, got not done: %+v", tdo)
	}

	t.Logf("First todo: %+v", tdo)
}
func TestQuerySet_Where(t *testing.T) {
	todos, err := queries.GetQuerySet[attrs.Definer](&Todo{}).
		Select("ID", "Title", "Description", "Done").
		Filter(
			expr.Expr("Title", "icontains", "test"),
			expr.Q("Done", false),
		).
		All()

	if err != nil {
		t.Fatalf("Failed to get first todo: %v", err)
	}

	if len(todos) != 1 {
		t.Fatalf("Expected 1 todo, got %d", len(todos))
	}

	for _, todo := range todos {
		todo := todo.Object.(*Todo)
		if !strings.Contains(strings.ToLower(todo.Title), "test") {
			t.Fatalf("Expected todo title to contain 'test', got: %s", todo.Title)
		}

		if todo.Done {
			t.Fatalf("Expected todo to be not done, got done: %+v", todo)
		}

		t.Logf("Filtered todos: %+v", todo)
	}
}

func TestQuerySet_Count(t *testing.T) {
	count, err := queries.GetQuerySet[attrs.Definer](&Todo{}).
		Filter(expr.And(
			expr.Expr("Title", "icontains", "test"),
			expr.Q("Done", false),
		)).
		Count()

	if err != nil {
		t.Fatalf("Failed to count todos: %v", err)
	}

	if count != 1 {
		t.Fatalf("Expected 1 todo, got %d", count)
	}

	t.Logf("Counted todos: %d", count)
}

func TestQueryRelated(t *testing.T) {

	var profile = &Profile{
		Name:  "test profile",
		Email: "test@example.com",
	}

	if err := queries.CreateObject(profile); err != nil || profile.ID == 0 {
		t.Fatalf("Failed to insert profile: %v", err)
	}

	var user = &User{
		Name:    "test user",
		Profile: profile,
	}
	if err := queries.CreateObject(user); err != nil || user.ID == 0 {
		t.Fatalf("Failed to insert user: %v", err)
	}

	var todo = &Todo{
		Title:       "New Test Todo",
		Description: "This is a new test todo",
		Done:        false,
		User:        user,
	}

	if err := queries.CreateObject(todo); err != nil {
		t.Fatalf("Failed to insert todo: %v", err)
	}

	var qs = queries.GetQuerySet[attrs.Definer](&Todo{}).
		Select("ID", "Title", "Description", "Done", "User.Name", "User.Profile.*").
		Filter(
			expr.Q("Title__icontains", "new test"),
			expr.Q("Done", false),
			expr.Q("User.Name__icontains", "test"),
		).
		OrderBy("-ID", "-User.Name").
		Limit(5)
	todos, err := qs.All()
	if err != nil {
		t.Fatalf("Failed to filter todos: %v (%s)", err, qs.LatestQuery().SQL())
	}

	if len(todos) != 1 {
		t.Fatalf("Expected 1 todo, got %d", len(todos))
	}

	var dbTodo = todos[0].Object.(*Todo)
	t.Logf("Created todo: %+v, %+v", todo, todo.User)
	t.Logf("Filtered todo: %+v, %+v", dbTodo, dbTodo.User)

	if dbTodo.ID != todo.ID || todo.ID == 0 {
		t.Fatalf("Expected todo ID %d, got %d", todo.ID, dbTodo.ID)
	}

	if dbTodo.Title != todo.Title {
		t.Fatalf("Expected todo title %q, got %q", todo.Title, dbTodo.Title)
	}

	if dbTodo.Description != todo.Description {
		t.Fatalf("Expected todo description %q, got %q", todo.Description, dbTodo.Description)
	}

	if dbTodo.Done != todo.Done {
		t.Fatalf("Expected todo done %v, got %v", todo.Done, dbTodo.Done)
	}

	if dbTodo.User == nil {
		t.Fatalf("Expected todo user to be not nil")
	}

	if dbTodo.User.ID != 0 {
		t.Fatalf("Expected todo user ID to be 0, got %d", dbTodo.User.ID)
	}

	if dbTodo.User.Name != todo.User.Name {
		t.Fatalf("Expected todo user name %q, got %q", todo.User.Name, dbTodo.User.Name)
	}

	if dbTodo.User.Profile == nil {
		t.Fatalf("Expected todo user profile to be not nil")
	}

	if dbTodo.User.Profile.ID != todo.User.Profile.ID {
		t.Fatalf("Expected todo user profile ID %d, got %d", todo.User.Profile.ID, dbTodo.User.Profile.ID)
	}

	if dbTodo.User.Profile.Name != todo.User.Profile.Name {
		t.Fatalf("Expected todo user profile name %q, got %q", todo.User.Profile.Name, dbTodo.User.Profile.Name)
	}

	if dbTodo.User.Profile.Email != todo.User.Profile.Email {
		t.Fatalf("Expected todo user profile email %q, got %q", todo.User.Profile.Email, dbTodo.User.Profile.Email)
	}
}

func TestQueryRelatedMultiple(t *testing.T) {
	var user1 = &User{
		Name: "TestQueryRelatedMultiple 1",
	}
	if err := queries.CreateObject(user1); err != nil || user1.ID == 0 {
		t.Fatalf("Failed to insert user1: %v", err)
	}

	var user2 = &User{
		Name: "TestQueryRelatedMultiple 2",
	}
	if err := queries.CreateObject(user2); err != nil || user2.ID == 0 {
		t.Fatalf("Failed to insert user2: %v", err)
	}

	var obj = &ObjectWithMultipleRelations{
		Obj1: user1,
		Obj2: user2,
	}

	if err := queries.CreateObject(obj); err != nil {
		t.Fatalf("Failed to insert object with multiple relations: %v", err)
	}

	var qs = queries.GetQuerySet[attrs.Definer](&ObjectWithMultipleRelations{}).
		Select("ID", "Obj1.*", "Obj2.*").
		OrderBy("-ID")
	var objs, err = qs.All()
	if err != nil {
		t.Fatalf("Failed to filter objects with multiple relations: %v (%s)", err, qs.LatestQuery().SQL())
	}

	if len(objs) != 1 {
		t.Fatalf("Expected 1 object, got %d", len(objs))
	}

	var dbObj = objs[0].Object.(*ObjectWithMultipleRelations)
	t.Logf("Created object: %+v, %+v, %+v", obj, obj.Obj1, obj.Obj2)

	if dbObj.Obj1.ID == 0 {
		t.Fatalf("Expected Obj1 ID to be not 0")
	}

	if dbObj.Obj2.ID == 0 {
		t.Fatalf("Expected Obj2 ID to be not 0")
	}

	if user1.ID != dbObj.Obj1.ID {
		t.Fatalf("Expected Obj1 ID %d, got %d", user1.ID, dbObj.Obj1.ID)
	}

	if user2.ID != dbObj.Obj2.ID {
		t.Fatalf("Expected Obj2 ID %d, got %d", user2.ID, dbObj.Obj2.ID)
	}

	if user1.Name != dbObj.Obj1.Name {
		t.Fatalf("Expected Obj1 name %q, got %q", user1.Name, dbObj.Obj1.Name)
	}

	if user2.Name != dbObj.Obj2.Name {
		t.Fatalf("Expected Obj2 name %q, got %q", user2.Name, dbObj.Obj2.Name)
	}
}

func TestQuerySetSelectExpressions(t *testing.T) {
	var todo = &Todo{
		Title:       "TestQuerySet_Select_Expressions",
		Description: "This is a test todo",
		Done:        false,
	}

	if err := queries.CreateObject(todo); err != nil {
		t.Fatalf("Failed to insert todo: %v", err)
	}

	if todo.ID == 0 {
		t.Fatalf("Expected ID to be set after insert, got 0")
	}

	var qs = queries.GetQuerySet[attrs.Definer](&Todo{}).
		Select("ID", expr.F(expr.UPPER("Title")), "Description", "Done").
		Filter("Title", "TestQuerySet_Select_Expressions").
		OrderBy("-ID")

	var todos, err = qs.All()
	if err != nil {
		t.Fatalf("Failed to filter todos: %v (%s)", err, qs.LatestQuery().SQL())
	}

	if len(todos) != 1 {
		t.Fatalf("Expected 1 todo, got %d", len(todos))
	}

	var dbTodo = todos[0].Object.(*Todo)
	t.Logf("Created todo: %+v", todo)

	t.Logf("Filtered todo: %+v", dbTodo)

	if dbTodo.ID != todo.ID || todo.ID == 0 {
		t.Fatalf("Expected todo ID %d, got %d", todo.ID, dbTodo.ID)
	}

	if dbTodo.Title != strings.ToUpper(todo.Title) {
		t.Fatalf("Expected todo title %q, got %q", todo.Title, dbTodo.Title)
	}

	if dbTodo.Description != todo.Description {
		t.Fatalf("Expected todo description %q, got %q", todo.Description, dbTodo.Description)
	}

	if dbTodo.Done != todo.Done {
		t.Fatalf("Expected todo done %v, got %v", todo.Done, dbTodo.Done)
	}
}

func TestQuerySetSelectExpressionsWithRelated(t *testing.T) {
	var user = &User{
		Name: "TestQuerySet_Select_ExpressionsWithRelated",
	}
	if err := queries.CreateObject(user); err != nil || user.ID == 0 {
		t.Fatalf("Failed to insert user: %v", err)
	}

	var todo = &Todo{
		Title:       "TestQuerySet_Select_ExpressionsWithRelated",
		Description: "This is a test todo",
		Done:        false,
		User:        user,
	}

	if err := queries.CreateObject(todo); err != nil {
		t.Fatalf("Failed to insert todo: %v", err)
	}

	var qs = queries.GetQuerySet[attrs.Definer](&Todo{}).
		Select(
			"ID",
			expr.F("UPPER(![Title])"),
			"Description",
			"Done",
			"User.ID",
			expr.F("UPPER(![User.Name])"),
		).
		Filter("Title", "TestQuerySet_Select_ExpressionsWithRelated").
		OrderBy("-ID")

	todos, err := qs.All()
	if err != nil {
		t.Fatalf("Failed to filter todos: %v (%s)", err, qs.LatestQuery().SQL())
	}

	if len(todos) != 1 {
		t.Fatalf("Expected 1 todo, got %d", len(todos))
	}

	var dbTodo = todos[0].Object.(*Todo)
	t.Logf("Created todo: %+v, %+v", todo, todo.User)

	t.Logf("Filtered todo: %+v, %+v", dbTodo, dbTodo.User)

	if dbTodo.ID != todo.ID || todo.ID == 0 {
		t.Fatalf("Expected todo ID %d, got %d", todo.ID, dbTodo.ID)
	}

	if dbTodo.Title != strings.ToUpper(todo.Title) {
		t.Fatalf("Expected todo title %q, got %q", todo.Title, dbTodo.Title)
	}

	if dbTodo.Description != todo.Description {
		t.Fatalf("Expected todo description %q, got %q", todo.Description, dbTodo.Description)
	}

	if dbTodo.Done != todo.Done {
		t.Fatalf("Expected todo done %v, got %v", todo.Done, dbTodo.Done)
	}

	if dbTodo.User == nil {
		t.Fatalf("Expected todo user to be not nil")
	}

	if dbTodo.User.ID != todo.User.ID {
		t.Fatalf("Expected todo user ID %d, got %d", todo.User.ID, dbTodo.User.ID)
	}

	if dbTodo.User.Name != strings.ToUpper(todo.User.Name) {
		t.Fatalf("Expected todo user name %q, got %q", todo.User.Name, dbTodo.User.Name)
	}

	if dbTodo.User.Profile != nil {
		t.Fatalf("Expected todo user profile to be nil")
	}
}

func TestQueryRelatedIDOnly(t *testing.T) {
	var user = &User{
		Name: "TestQueryRelatedIDOnly",
	}
	if err := queries.CreateObject(user); err != nil || user.ID == 0 {
		t.Fatalf("Failed to insert user: %v", err)
	}

	var todo = &Todo{
		Title:       "TestQueryRelatedIDOnly",
		Description: "This is a new test todo",
		Done:        false,
		User:        user,
	}

	if err := queries.CreateObject(todo); err != nil {
		t.Fatalf("Failed to insert todo: %v", err)
	}

	todos, err := queries.GetQuerySet[attrs.Definer](&Todo{}).
		Select("ID", "Title", "Description", "Done", "User").
		Filter("Title", "TestQueryRelatedIDOnly").
		OrderBy("-ID", "-User").
		Limit(5).
		All()

	if err != nil {
		t.Fatalf("Failed to filter todos: %v", err)
	}

	if len(todos) != 1 {
		t.Fatalf("Expected 1 todo, got %d", len(todos))
	}

	var dbTodo = todos[0].Object.(*Todo)
	t.Logf("Created todo: %+v, %+v", todo, todo.User)
	t.Logf("Filtered todo: %+v, %+v", dbTodo, dbTodo.User)

	if dbTodo.ID != todo.ID || todo.ID == 0 {
		t.Fatalf("Expected todo ID %d, got %d", todo.ID, dbTodo.ID)
	}

	if dbTodo.Title != todo.Title {
		t.Fatalf("Expected todo title %q, got %q", todo.Title, dbTodo.Title)
	}

	if dbTodo.Description != todo.Description {
		t.Fatalf("Expected todo description %q, got %q", todo.Description, dbTodo.Description)
	}

	if dbTodo.Done != todo.Done {
		t.Fatalf("Expected todo done %v, got %v", todo.Done, dbTodo.Done)
	}

	if dbTodo.User == nil {
		t.Fatalf("Expected todo user to be not nil")
	}

	if dbTodo.User.ID != todo.User.ID {
		t.Fatalf("Expected todo user ID %d, got %d", todo.User.ID, dbTodo.User.ID)
	}

	if dbTodo.User.Name != "" {
		t.Fatalf("Expected todo user name to be empty, got %q", dbTodo.User.Name)
	}
}

func TestQueryValuesList(t *testing.T) {
	var user = &User{
		Name: "TestQueryValuesList",
	}

	if err := queries.CreateObject(user); err != nil || user.ID == 0 {
		t.Fatalf("Failed to insert user: %v", err)
	}

	var todos = []*Todo{
		{Title: "TestQueryValuesList 1", Description: "Description 1", Done: false, User: user},
		{Title: "TestQueryValuesList 2", Description: "Description 2", Done: true, User: user},
		{Title: "TestQueryValuesList 3", Description: "Description 3", Done: false, User: user},
	}

	for _, todo := range todos {
		if err := queries.CreateObject(todo); err != nil {
			t.Fatalf("Failed to insert todo: %v", err)
		}
	}

	var values, err = queries.GetQuerySet[attrs.Definer](&Todo{}).
		Filter("Title__istartswith", "testqueryvalueslist").
		OrderBy("ID", "-User.Name").
		// ValuesList("ID", "Title", "Description", "Done", "User.ID", "User.Name")
		ValuesList("ID", "Title", "Description", "Done", "User", "User.ID", "User.Name", "User.*")
	// ValuesList("ID", "Title", "Description", "Done", "User")

	if err != nil {
		t.Fatalf("Failed to get values list: %v", err)
	}

	if len(values) != 3 {
		t.Fatalf("Expected 3 values, got %d", len(values))
	}

	for _, value := range values {
		var b strings.Builder
		b.WriteString("[")
		for i, v := range value {
			b.WriteString(fmt.Sprintf("(%T) %v", v, v))
			if i < len(value)-1 {
				b.WriteString(", ")
			}
		}
		b.WriteString("]")
		t.Logf("Got todo values: %s", b.String())
	}

	for i, value := range values {
		//if len(value) != 7 {
		//	t.Fatalf("Expected 7 values, got %d", len(value))
		//}

		if value[0] == nil || value[1] == nil || value[2] == nil || value[3] == nil {
			t.Fatalf("Expected all values to be not nil")
		}

		if value[0].(int) != todos[i].ID {
			t.Fatalf("Expected todo ID %d, got %d", todos[i].ID, value[0])
		}

		if value[1].(string) != todos[i].Title {
			t.Fatalf("Expected todo title %q, got %q", todos[i].Title, value[1])
		}

		if value[2].(string) != todos[i].Description {
			t.Fatalf("Expected todo description %q, got %q", todos[i].Description, value[2])
		}

		if value[3].(bool) != todos[i].Done {
			t.Fatalf("Expected todo done %v, got %v", todos[i].Done, value[3])
		}

		//if value[4] == nil {
		//	t.Fatalf("Expected todo user ID to be not nil")
		//}
		//
		//if value[5] == nil {
		//	t.Fatalf("Expected todo user name to be not nil")
		//}
		//
		//if value[4].(int) != todos[i].User.ID {
		//	t.Fatalf("Expected todo user ID %d, got %d", todos[i].User.ID, value[4])
		//}
		//
		//if value[5].(string) != todos[i].User.Name {
		//	t.Fatalf("Expected todo user name %q, got %q", todos[i].User.Name, value[5])
		//}

	}
}

func TestValues(t *testing.T) {
	var user = &User{
		Name: "TestValues",
	}

	if err := queries.CreateObject(user); err != nil || user.ID == 0 {
		t.Fatalf("Failed to insert user: %v", err)
	}

	var todos = []*Todo{
		{Title: "TestValues 1", Description: "Description 1", Done: false, User: user},
		{Title: "TestValues 2", Description: "Description 2", Done: true, User: user},
		{Title: "TestValues 3", Description: "Description 3", Done: false, User: user},
	}

	todos, err := queries.GetQuerySet(&Todo{}).BulkCreate(todos)
	if err != nil {
		t.Fatalf("Failed to insert todos: %v", err)
	}

	values, err := queries.GetQuerySet[attrs.Definer](&Todo{}).
		Filter("Title__icontains", "testvalues").
		OrderBy("ID", "-User.Name").
		Values("ID", "Title", "Description", "Done", "User.ID", "User.Name")
	if err != nil {
		t.Fatalf("Failed to get values: %v", err)
	}
	if len(values) != 3 {
		t.Fatalf("Expected 3 values, got %d", len(values))
	}

	for i, value := range values {
		var todo = todos[i]
		if len(value) != 6 {
			t.Fatalf("Expected 6 values, got %d", len(value))
		}

		t.Logf("Got todo values: %+v", value)

		if value["ID"] != todo.ID {
			t.Fatalf("Expected todo ID %d, got %v", todo.ID, value["ID"])
		}

		if value["Title"] != todo.Title {
			t.Fatalf("Expected todo title %q, got %v", todo.Title, value["Title"])
		}

		if value["Description"] != todo.Description {
			t.Fatalf("Expected todo description %q, got %v", todo.Description, value["Description"])
		}

		if value["Done"] != todo.Done {
			t.Fatalf("Expected todo done %v, got %v", todo.Done, value["Done"])
		}

		if value["User.ID"] != todo.User.ID {
			t.Fatalf("Expected todo user ID %d, got %v", todo.User.ID, value["User.ID"])
		}

		if value["User.Name"] != todo.User.Name {
			t.Fatalf("Expected todo user name %q, got %v", todo.User.Name, value["User.Name"])
		}
	}

}

func TestQueryNestedRelated(t *testing.T) {
	var image = &Image{
		Path: "test/path/to/image.jpg",
	}

	if err := queries.CreateObject(image); err != nil || image.ID == 0 {
		t.Fatalf("Failed to insert image: %v", err)
	}

	var profile = &Profile{
		Name:  "test profile",
		Email: "test@example.com",
		Image: image,
	}

	if err := queries.CreateObject(profile); err != nil || profile.ID == 0 {
		t.Fatalf("Failed to insert profile: %v", err)
	}

	var user = &User{
		Name:    "test user",
		Profile: profile,
	}

	if err := queries.CreateObject(user); err != nil || user.ID == 0 {
		t.Fatalf("Failed to insert user: %v", err)
	}

	var todo = &Todo{
		Title:       "New Test Todo",
		Description: "This is a new test todo",
		Done:        false,
		User:        user,
	}

	if err := queries.CreateObject(todo); err != nil {
		t.Fatalf("Failed to insert todo: %v", err)
	}

	var qs = queries.GetQuerySet[attrs.Definer](&Todo{}).
		Select("*", "User.*", "User.Profile.*", "User.Profile.Image.*").
		Filter(
			expr.Q("Title__icontains", "new test"),
			expr.Q("Done", false),
			expr.Q("User.Name__icontains", "test"),
			expr.Q("User.ID", user.ID),
			expr.Q("User.Profile.Email__icontains", profile.Email),
			expr.Q("User.Profile.ID", profile.ID),
			//&queries.FuncExpr{
			//	Statement: "LOWER(SUBSTR(%s, 0, 2)) LIKE LOWER(?)",
			//	Fields:    []string{"User.Name"},
			//	Params:    []any{"%te%"},
			//},

			&expr.RawExpr{
				Statement: &expr.ExpressionStatement{
					Statement: "![User.ID] = ?",
					Values:    []any{user.ID},
				},
			},
			// queries.Q("User.Profile.Email__icontains", "example"),
		).
		OrderBy("-ID", "-User.Name", "-User.Profile.Email").
		Limit(5)
	todos, err := qs.All()
	if err != nil {
		t.Fatalf("Failed to filter todos: %v (%s)", err, qs.LatestQuery().SQL())
	}

	if len(todos) != 1 {
		t.Fatalf("Expected 1 todo, got %d", len(todos))
	}

	var dbTodo = todos[0].Object.(*Todo)
	t.Logf("Created todo: %+v, %+v, %+v, %+v", todo, todo.User, todo.User.Profile, todo.User.Profile.Image)
	t.Logf("Filtered todo: %+v, %+v, %+v, %+v", dbTodo, dbTodo.User, dbTodo.User.Profile, dbTodo.User.Profile.Image)

	if dbTodo.ID != todo.ID || todo.ID == 0 {
		t.Fatalf("Expected todo ID %d, got %d", todo.ID, dbTodo.ID)
	}

	if dbTodo.Title != todo.Title {
		t.Fatalf("Expected todo title %q, got %q", todo.Title, dbTodo.Title)
	}

	if dbTodo.Description != todo.Description {
		t.Fatalf("Expected todo description %q, got %q", todo.Description, dbTodo.Description)
	}

	if dbTodo.Done != todo.Done {
		t.Fatalf("Expected todo done %v, got %v", todo.Done, dbTodo.Done)
	}

	if dbTodo.User == nil {
		t.Fatalf("Expected todo user to be not nil")
	}

	if dbTodo.User.ID != todo.User.ID {
		t.Fatalf("Expected todo user ID %d, got %d", todo.User.ID, dbTodo.User.ID)
	}

	if dbTodo.User.Profile == nil {
		t.Fatalf("Expected todo user profile to be not nil")
	}

	if dbTodo.User.Profile.ID != todo.User.Profile.ID {
		t.Fatalf("Expected todo user profile ID %d, got %d", todo.User.Profile.ID, dbTodo.User.Profile.ID)
	}

	if dbTodo.User.Profile.Email != todo.User.Profile.Email {
		t.Fatalf("Expected todo user profile email %q, got %q", todo.User.Profile.Email, dbTodo.User.Profile.Email)
	}

	if dbTodo.User.Profile.Name != todo.User.Profile.Name {
		t.Fatalf("Expected todo user profile name %q, got %q", todo.User.Profile.Name, dbTodo.User.Profile.Name)
	}

	if dbTodo.User.Profile.Image == nil {
		t.Fatalf("Expected todo user profile image to be not nil")
	}

	if dbTodo.User.Profile.Image.ID != todo.User.Profile.Image.ID {
		t.Fatalf("Expected todo user profile image ID %d, got %d", todo.User.Profile.Image.ID, dbTodo.User.Profile.Image.ID)
	}
}

func TestQueryUpdate(t *testing.T) {

	var user = &User{
		Name: "TestQueryUpdate",
	}

	if err := queries.CreateObject(user); err != nil || user.ID == 0 {
		t.Fatalf("Failed to insert user: %v", err)
	}

	var todo1 = &Todo{
		Title:       "TestQueryUpdate1",
		Description: "This is a new test todo",
	}

	var todo2 = &Todo{
		Title:       "TestQueryUpdate2",
		Description: "This is a new test todo",
	}

	if err := queries.CreateObject(todo1); err != nil {
		t.Fatalf("Failed to insert todo: %v", err)
	}

	if err := queries.CreateObject(todo2); err != nil {
		t.Fatalf("Failed to insert todo: %v", err)
	}

	var updated, err = queries.GetQuerySet[attrs.Definer](&Todo{}).
		Select("Title", "User").
		Filter("Title__istartswith", "testqueryupdate").
		Filter("Done", false).
		Update(&Todo{
			Title: "Updated Title",
			User:  user,
		})

	if err != nil {
		t.Fatalf("Failed to update todo: %v", err)
	}

	if updated == 0 {
		t.Fatalf("Expected 1 todo to be updated, got %d", updated)
	}

	dbTodo1, err := queries.GetObject[*Todo](&Todo{}, todo1.ID)
	if err != nil {
		t.Fatalf("Failed to get todo: %v", err)
	}

	if dbTodo1.ID != todo1.ID || todo1.ID == 0 {
		t.Fatalf("Expected todo ID %d, got %d", todo1.ID, dbTodo1.ID)
	}

	if dbTodo1.Title != "Updated Title" {
		t.Fatalf("Expected todo title 'Updated Title', got %q", dbTodo1.Title)
	}

	if dbTodo1.Description != todo1.Description {
		t.Fatalf("Expected todo description %q, got %q", todo1.Description, dbTodo1.Description)
	}

	if dbTodo1.Done != todo1.Done {
		t.Fatalf("Expected todo done %v, got %v", todo1.Done, dbTodo1.Done)
	}

	if dbTodo1.User == nil {
		t.Fatalf("Expected todo user to be not nil")
	}

	if dbTodo1.User.ID != user.ID {
		t.Fatalf("Expected todo user ID %d, got %d", user.ID, dbTodo1.User.ID)
	}

	dbTodo2, err := queries.GetObject[*Todo](&Todo{}, todo2.ID)
	if err != nil {
		t.Fatalf("Failed to get todo: %v", err)
	}

	if dbTodo2.ID != todo2.ID || todo2.ID == 0 {
		t.Fatalf("Expected todo ID %d, got %d", todo2.ID, dbTodo2.ID)
	}

	if dbTodo2.Title != "Updated Title" {
		t.Fatalf("Expected todo title %q, got %q", todo2.Title, dbTodo2.Title)
	}

	if dbTodo2.Description != todo2.Description {
		t.Fatalf("Expected todo description %q, got %q", todo2.Description, dbTodo2.Description)
	}

	if dbTodo2.Done != todo2.Done {
		t.Fatalf("Expected todo done %v, got %v", todo2.Done, dbTodo2.Done)
	}

	if dbTodo2.User == nil {
		t.Fatalf("Expected todo user to be not nil")
	}

	if dbTodo2.User.ID != user.ID {
		t.Fatalf("Expected todo user ID %d, got %d", todo2.User.ID, dbTodo2.User.ID)
	}

	t.Logf("Updated todo: %+v", dbTodo1)
	t.Logf("Updated todo user: %+v", dbTodo1.User)

	t.Logf("Updated todo: %+v", dbTodo2)
	t.Logf("Updated todo user: %+v", dbTodo2.User)
}

func TestUpdateWithExpressions(t *testing.T) {
	var todo = &Todo{
		Title:       "TestUpdateWithExpressions",
		Description: "This is a new test todo",
		Done:        false,
	}

	if err := queries.CreateObject(todo); err != nil {
		t.Fatalf("Failed to insert todo: %v", err)
	}

	var updated, err = queries.GetQuerySet[attrs.Definer](&Todo{}).
		Select("Title", "Done").
		Filter("ID", todo.ID).
		ExplicitSave().
		Update(
			&Todo{},
			expr.As("Title", expr.UPPER("Title")),
			expr.As("Done", expr.F("![ID] % ?[1] = ?[2] OR ![ID] % ?[1] = ?[3] OR ?[4]", 2, 0, 1, true)),
			// expr.F("![Title] = UPPER(![Title])"),
			// expr.F("![Done] = (![ID] % ?[1] == ?[2] OR ![ID] % ?[1] == ?[3] OR ?[4])", 2, 0, 1, true),
		)
	if err != nil {
		t.Fatalf("Failed to update todo: %v", err)
	}
	if updated == 0 {
		t.Fatalf("Expected 1 todo to be updated, got %d", updated)
	}

	dbTodo, err := queries.GetObject(&Todo{}, todo.ID)
	if err != nil {
		t.Fatalf("Failed to get todo: %v", err)
	}

	if dbTodo.ID != todo.ID || todo.ID == 0 {
		t.Fatalf("Expected todo ID %d, got %d", todo.ID, dbTodo.ID)
	}

	if dbTodo.Title != strings.ToUpper(todo.Title) {
		t.Fatalf("Expected todo title %q, got %q", strings.ToUpper(todo.Title), dbTodo.Title)
	}

	if dbTodo.Description != todo.Description {
		t.Fatalf("Expected todo description %q, got %q", todo.Description, dbTodo.Description)
	}

	if dbTodo.Done != true {
		t.Fatalf("Expected todo done %v, got %v", true, dbTodo.Done)
	}
}

func TestQueryGet(t *testing.T) {

	var todos = []*Todo{
		{Title: "TestQueryGet1", Description: "Description TestQueryGet", Done: false},
		{Title: "TestQueryGet2", Description: "Description TestQueryGet", Done: true},
		{Title: "TestQueryGet3", Description: "Description TestQueryGet", Done: false},
	}

	for _, todo := range todos {
		if err := queries.CreateObject(todo); err != nil {
			t.Fatalf("Failed to insert todo: %v", err)
		}
	}

	var todo, err = queries.GetQuerySet[attrs.Definer](&Todo{}).
		Select("*").
		Filter("Title", "TestQueryGet1").
		Get()
	if err != nil {
		t.Fatalf("Failed to get todo: %v", err)
	}

	if todo == nil {
		t.Fatalf("Expected a todo, got nil")
	}

	var tdo = todo.Object.(*Todo)

	if tdo.ID != todos[0].ID || todos[0].ID == 0 {
		t.Fatalf("Expected todo ID %d, got %d", todos[0].ID, tdo.ID)
	}

	if tdo.Title != todos[0].Title {
		t.Fatalf("Expected todo title %q, got %q", todos[0].Title, tdo.Title)
	}

	if tdo.Description != todos[0].Description {
		t.Fatalf("Expected todo description %q, got %q", todos[0].Description, tdo.Description)
	}

	if tdo.Done != todos[0].Done {
		t.Fatalf("Expected todo done %v, got %v", todos[0].Done, tdo.Done)
	}

	if tdo.User != nil {
		t.Fatalf("Expected todo user to be nil, got %+v", tdo.User)
	}
}

func TestQueryGetErrNoRows(t *testing.T) {
	var _, err = queries.GetQuerySet[attrs.Definer](&Todo{}).
		Select("*").
		Filter("Title", "NonExistentTitle").
		Get()
	if err == nil {
		t.Fatalf("Expected an error, got nil")
	}

	if !errors.Is(err, errors.NoRows) {
		t.Fatalf("Expected ErrNoRows, got %v", err)
	}

	t.Logf("No todo found as expected: %v", err)
}

func TestQueryGetMultipleRows(t *testing.T) {
	var todos = []*Todo{
		{Title: "TestQueryGetMultipleRows1", Description: "Description TestQueryGetMultipleRows", Done: false},
		{Title: "TestQueryGetMultipleRows2", Description: "Description TestQueryGetMultipleRows", Done: true},
		{Title: "TestQueryGetMultipleRows3", Description: "Description TestQueryGetMultipleRows", Done: false},
	}

	for _, todo := range todos {
		if err := queries.CreateObject(todo); err != nil {
			t.Fatalf("Failed to insert todo: %v", err)
		}
	}

	var _, err = queries.GetQuerySet[attrs.Definer](&Todo{}).
		Select("*").
		Filter("Title__icontains", expr.Value("TestQueryGetMultipleRows")).
		Get()
	if err == nil {
		t.Fatalf("Expected an error, got nil")
	}

	if !errors.Is(err, errors.MultipleRows) {
		t.Fatalf("Expected ErrMultipleRows, got %v", err)
	}

	t.Logf("Multiple todos found as expected: %v", err)
}

func TestQueryCreate(t *testing.T) {

	var db = django.ConfigGet[drivers.Database](
		django.Global.Settings,
		django.APPVAR_DATABASE,
	)

	if _, ok := db.Driver().(*drivers.DriverSQLite); !ok {
		t.Skip("This test is only supported by SQLite driver")
	}

	var user = &User{
		Name: "TestQueryCreate",
	}

	if err := queries.CreateObject(user); err != nil || user.ID == 0 {
		t.Fatalf("Failed to insert user: %v", err)
	}

	var todo = &Todo{
		Title:       "TestQueryCreate",
		Description: "This is a new test todo",
		Done:        false,
		User:        user,
	}

	t.Run("CreateReturningLastInsertID", func(t *testing.T) {
		drivers.Change("sqlite3", func(driver *drivers.Driver) {
			driver.SupportsReturning = drivers.SupportsReturningLastInsertId
		})
		todo.ID = 0 // Ensure ID is reset for creation

		var dbTodo, err = queries.GetQuerySet[attrs.Definer](&Todo{}).ExplicitSave().Create(todo)
		if err != nil {
			t.Fatalf("Failed to create todo: %v", err)
		}

		if dbTodo == nil {
			t.Fatalf("Expected a todo, got nil")
		}

		var tdo = dbTodo.(*Todo)

		if tdo.ID == 0 {
			t.Fatalf("Expected todo ID to be not 0, got %d", tdo.ID)
		}

		if tdo.Title != todo.Title {
			t.Fatalf("Expected todo title %q, got %q", todo.Title, tdo.Title)
		}

		if tdo.Description != todo.Description {
			t.Fatalf("Expected todo description %q, got %q", todo.Description, tdo.Description)
		}

		if tdo.Done != todo.Done {
			t.Fatalf("Expected todo done %v, got %v", todo.Done, tdo.Done)
		}

		if tdo.User == nil {
			t.Fatalf("Expected todo user to be not nil")
		}

		if tdo.User.ID != user.ID {
			t.Fatalf("Expected todo user ID %d, got %d", user.ID, tdo.User.ID)
		}

		t.Logf("Created todo: %+v, %+v", tdo, tdo.User)

		drivers.Change("sqlite3", func(driver *drivers.Driver) {
			driver.SupportsReturning = drivers.SupportsReturningColumns
		})
	})

	t.Run("CreateReturningColumns", func(t *testing.T) {
		drivers.Change("sqlite3", func(driver *drivers.Driver) {
			driver.SupportsReturning = drivers.SupportsReturningColumns
		})

		todo.ID = 0 // Ensure ID is reset for creation

		var dbTodo, err = queries.GetQuerySet[attrs.Definer](&Todo{}).ExplicitSave().Create(todo)
		if err != nil {
			t.Fatalf("Failed to create todo: %v", err)
		}

		if dbTodo == nil {
			t.Fatalf("Expected a todo, got nil")
		}

		var tdo = dbTodo.(*Todo)

		if tdo.ID == 0 {
			t.Fatalf("Expected todo ID to be not 0, got %d", tdo.ID)
		}

		if tdo.Title != todo.Title {
			t.Fatalf("Expected todo title %q, got %q", todo.Title, tdo.Title)
		}

		if tdo.Description != todo.Description {
			t.Fatalf("Expected todo description %q, got %q", todo.Description, tdo.Description)
		}

		if tdo.Done != todo.Done {
			t.Fatalf("Expected todo done %v, got %v", todo.Done, tdo.Done)
		}

		if tdo.User == nil {
			t.Fatalf("Expected todo user to be not nil")
		}

		if tdo.User.ID != user.ID {
			t.Fatalf("Expected todo user ID %d, got %d", user.ID, tdo.User.ID)
		}

		t.Logf("Created todo: %+v, %+v", tdo, tdo.User)
	})

	t.Run("CreateReturningNone", func(t *testing.T) {
		drivers.Change("sqlite3", func(driver *drivers.Driver) {
			driver.SupportsReturning = drivers.SupportsReturningNone
		})

		todo.ID = 0 // Ensure ID is reset for creation

		var dbTodo, err = queries.GetQuerySet[attrs.Definer](&Todo{}).ExplicitSave().Create(todo)
		if err != nil {
			t.Fatalf("Failed to create todo: %v", err)
		}

		if dbTodo == nil {
			t.Fatalf("Expected a todo, got nil")
		}

		var tdo = dbTodo.(*Todo)
		if tdo.ID != 0 {
			t.Fatalf("Expected todo ID to be 0, got %d", tdo.ID)
		}

		if tdo.Title != todo.Title {
			t.Fatalf("Expected todo title %q, got %q", todo.Title, tdo.Title)
		}

		if tdo.Description != todo.Description {
			t.Fatalf("Expected todo description %q, got %q", todo.Description, tdo.Description)
		}

		if tdo.Done != todo.Done {
			t.Fatalf("Expected todo done %v, got %v", todo.Done, tdo.Done)
		}

		if tdo.User == nil {
			t.Fatalf("Expected todo user to be not nil")
		}

		if tdo.User.ID != user.ID {
			t.Fatalf("Expected todo user ID %d, got %d", user.ID, tdo.User.ID)
		}

		t.Logf("Created todo: %+v, %+v", tdo, tdo.User)

		drivers.Change("sqlite3", func(driver *drivers.Driver) {
			driver.SupportsReturning = drivers.SupportsReturningColumns
		})
	})
}

func TestQueryGetOrCreate(t *testing.T) {
	var user = &User{
		Name: "TestQueryGetOrCreate",
	}

	if err := queries.CreateObject(user); err != nil || user.ID == 0 {
		t.Fatalf("Failed to insert user: %v", err)
	}

	var _user = *user

	var todo = &Todo{
		Title:       "TestQueryGetOrCreate",
		Description: "This is a new test todo",
		Done:        false,
		User:        &_user,
	}

	todo.User.Name = ""

	var dbTodo, created, err = queries.GetQuerySet[attrs.Definer](&Todo{}).
		Select("ID", "Title", "Description", "Done", "User").
		Filter("Title", todo.Title).
		GetOrCreate(todo)
	if err != nil {
		t.Fatalf("Failed to get or create todo: %v", err)
	}

	if !created {
		t.Fatalf("Expected todo to be created, got false")
	}

	if dbTodo == nil {
		t.Fatalf("Expected a todo, got nil")
	}

	var tdo = dbTodo.(*Todo)

	if tdo.ID == 0 {
		t.Fatalf("Expected todo ID to be not 0, got %d (dbTodo: %d, %+v)", tdo.ID, dbTodo.(*Todo).ID, dbTodo)
	}

	if tdo.Title != todo.Title {
		t.Fatalf("Expected todo title %q, got %q", todo.Title, tdo.Title)
	}

	if tdo.Description != todo.Description {
		t.Fatalf("Expected todo description %q, got %q", todo.Description, tdo.Description)
	}

	if tdo.Done != todo.Done {
		t.Fatalf("Expected todo done %v, got %v", todo.Done, tdo.Done)
	}

	if tdo.User == nil {
		t.Fatalf("Expected todo user to be not nil")
	}

	if tdo.User.ID != user.ID {
		t.Fatalf("Expected todo user ID %d, got %d", user.ID, tdo.User.ID)
	}

	if tdo.User.Name != "" {
		t.Fatalf("Expected todo user name to be empty, got %q", tdo.User.Name)
	}

	t.Logf("Created or retrieved todo: %+v, %+v", tdo, tdo.User)
}

// error checking is irrelevant for these tests,
// there don't need to actually be any todos in the database
func TestQuerySet_LatestQuery(t *testing.T) {
	// Test All() CompiledQuery[[][]interface{}]
	t.Run("TestAll", func(t *testing.T) {
		var query = queries.GetQuerySet[attrs.Definer](&Todo{}).
			Select("ID", "Title", "Description", "Done").
			Filter("Title__icontains", "test").
			Filter("Done", false)

		query.All()

		var latest = query.LatestQuery()
		if latest == nil {
			t.Fatalf("expected latest query, got nil")
		}

		t.Log(latest.SQL())

		if _, ok := latest.(queries.CompiledQuery[[][]interface{}]); !ok {
			t.Fatalf("expected *QueryObject[[][]interface{}], got %T", latest)
		}
	})

	// Test ValuesList(fields ...any) CompiledQuery[[][]any]
	t.Run("TestValuesList", func(t *testing.T) {
		var query = queries.GetQuerySet[attrs.Definer](&Todo{}).
			Select("ID", "Title", "Description", "Done", "User").
			Filter("Title__icontains", "test").
			Filter("Done", false)

		query.ValuesList("ID", "Title", "Description", "Done", "User")

		var latest = query.LatestQuery()
		if latest == nil {
			t.Fatalf("expected latest query, got nil")
		}

		t.Log(latest.SQL())

		if _, ok := latest.(queries.CompiledQuery[[][]any]); !ok {
			t.Fatalf("expected *QueryObject[[][]any], got %T", latest)
		}
	})

	// Test Aggregate(annotations map[string]expr.Expression) CompiledQuery[[][]interface{}]
	t.Run("TestAggregate", func(t *testing.T) {
		var query = queries.GetQuerySet[attrs.Definer](&Todo{}).
			Select("ID", "Title", "Description", "Done", "User").
			Filter("Title__icontains", "test").
			Filter("Done", false)

		query.Aggregate(map[string]expr.Expression{
			"Total": &expr.RawExpr{
				Statement: &expr.ExpressionStatement{Statement: "COUNT(*)"},
			},
			"MinID": &expr.RawExpr{
				Statement: &expr.ExpressionStatement{Statement: "MIN(id)"},
			},
			"MaxID": &expr.RawExpr{
				Statement: &expr.ExpressionStatement{Statement: "MAX(id)"},
			},
		})

		var latest = query.LatestQuery()
		if latest == nil {
			t.Fatalf("expected latest query, got nil")
		}

		t.Log(latest.SQL())

		if _, ok := latest.(queries.CompiledQuery[[][]interface{}]); !ok {
			t.Fatalf("expected *QueryObject[[][]interface{}], got %T", latest)
		}
	})

	// Test Get() CompiledQuery[[][]interface{}]
	t.Run("TestGet", func(t *testing.T) {
		var query = queries.GetQuerySet[attrs.Definer](&Todo{}).
			Select("ID", "Title", "Description", "Done", "User").
			Filter("Title__icontains", "test").
			Filter("Done", false)

		query.Get()

		var latest = query.LatestQuery()
		if latest == nil {
			t.Fatalf("expected latest query, got nil")
		}

		t.Log(latest.SQL())

		if _, ok := latest.(queries.CompiledQuery[[][]interface{}]); !ok {
			t.Fatalf("expected *QueryObject[[][]interface{}], got %T", latest)
		}
	})

	// Test GetOrCreate(value T) CompiledQuery[[][]interface{}] | CompiledQuery[[]interface{}]
	t.Run("TestGetOrCreate", func(t *testing.T) {
		var query = queries.GetQuerySet[attrs.Definer](&Todo{}).
			Select("ID", "Title", "Description", "Done", "User").
			Filter("Title", "LatestQuery_TestGetOrCreate").
			Filter("Done", false)

		var todo = &Todo{Title: "LatestQuery_TestGetOrCreate"}

		t.Run("TestGetOrCreate_Create", func(t *testing.T) {
			var _, _, err = query.GetOrCreate(todo)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
				return
			}

			var latest = query.LatestQuery()
			if latest == nil {
				t.Fatalf("expected latest query, got nil")
			}

			t.Log(latest.SQL())

			if _, ok := latest.(queries.CompiledQuery[[][]interface{}]); !ok {
				t.Fatalf("expected *QueryObject[[][]interface{}], got %T", latest)
			}
		})

		t.Run("TestGetOrCreate_Get", func(t *testing.T) {
			var _, _, err = query.GetOrCreate(todo)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
				return
			}

			var latest = query.LatestQuery()
			if latest == nil {
				t.Fatalf("expected latest query, got nil")
			}

			t.Log(latest.SQL())

			if _, ok := latest.(queries.CompiledQuery[[][]interface{}]); !ok {
				t.Fatalf("expected *QueryObject[[][]interface{}], got %T", latest)
			}
		})

		if todo.ID == 0 {
			t.Fatalf("expected todo ID to be not 0, got %d", todo.ID)
		}

		queries.DeleteObject(todo)
	})

	// Test First() CompiledQuery[[][]interface{}]
	t.Run("TestFirst", func(t *testing.T) {
		var query = queries.GetQuerySet[attrs.Definer](&Todo{}).
			Select("ID", "Title", "Description", "Done").
			Filter("Title__icontains", "test").
			Filter("Done", false)

		query.First()

		var latest = query.LatestQuery()
		if latest == nil {
			t.Fatalf("expected latest query, got nil")
		}

		t.Log(latest.SQL())

		if _, ok := latest.(queries.CompiledQuery[[][]interface{}]); !ok {
			t.Fatalf("expected *QueryObject[[][]interface{}], got %T", latest)
		}

	})

	// Test Last() CompiledQuery[[][]interface{}]
	t.Run("TestLast", func(t *testing.T) {
		var query = queries.GetQuerySet[attrs.Definer](&Todo{}).
			Select("ID", "Title", "Description", "Done").
			Filter("Title__icontains", "test").
			Filter("Done", false)

		query.Last()

		var latest = query.LatestQuery()
		if latest == nil {
			t.Fatalf("expected latest query, got nil")
		}

		t.Log(latest.SQL())

		if _, ok := latest.(queries.CompiledQuery[[][]interface{}]); !ok {
			t.Fatalf("expected *QueryObject[[][]interface{}], got %T", latest)
		}
	})

	// Test Exists() CompiledQuery[int64]
	t.Run("TestExists", func(t *testing.T) {
		var query = queries.GetQuerySet[attrs.Definer](&Todo{}).
			Select("ID", "Title", "Description", "Done").
			Filter("Title__icontains", "test").
			Filter("Done", false)

		query.Exists()

		var latest = query.LatestQuery()
		if latest == nil {
			t.Fatalf("expected latest query, got nil")
		}

		t.Log(latest.SQL())

		if _, ok := latest.(queries.CompiledQuery[int64]); !ok {
			t.Fatalf("expected *QueryObject[int64], got %T", latest)
		}
	})

	// Test Count() CompiledQuery[int64]
	t.Run("TestCount", func(t *testing.T) {
		var query = queries.GetQuerySet[attrs.Definer](&Todo{}).
			Select("ID", "Title", "Description", "Done").
			Filter("Title__icontains", "test").
			Filter("Done", false)

		query.Count()

		var latest = query.LatestQuery()
		if latest == nil {
			t.Fatalf("expected latest query, got nil")
		}

		t.Log(latest.SQL())

		if _, ok := latest.(queries.CompiledQuery[int64]); !ok {
			t.Fatalf("expected *QueryObject[int64], got %T", latest)
		}
	})

	// Test Create(value T) CompiledQuery[[]interface{}]
	t.Run("TestCreate", func(t *testing.T) {
		var query = queries.GetQuerySet[attrs.Definer](&Todo{}).
			Select("ID", "Title", "Description", "Done").
			Filter("Title__icontains", "test")

		var todo = &Todo{Title: "TestCreate"}
		var _, err = query.ExplicitSave().Create(todo)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
			return
		}

		var latest = query.LatestQuery()
		if latest == nil {
			t.Fatalf("expected latest query, got nil")
		}

		if _, ok := latest.(queries.CompiledQuery[[][]interface{}]); !ok {
			t.Fatalf("expected *QueryObject[[][]interface{}], got %T", latest)
		}

		t.Log(latest.SQL())

		queries.DeleteObject(todo)
	})

	// Test Update(value attrs.Definer, expressions ...expr.NamedExpression) CompiledQuery[int64]
	t.Run("TestUpdate", func(t *testing.T) {
		var query = queries.GetQuerySet[attrs.Definer](&Todo{}).
			Select("Title").
			Filter("Title__icontains", "test").
			Filter("Done", false)

		query.Update(&Todo{}, expr.As("Title", expr.UPPER("Title")))

		var latest = query.LatestQuery()
		if latest == nil {
			t.Fatalf("expected latest query, got nil")
		}

		if _, ok := latest.(queries.CompiledQuery[int64]); !ok {
			t.Fatalf("expected *QueryObject[int64], got %T", latest)
		}

		t.Log(latest.SQL())
	})

	// Test Delete() CompiledQuery[int64]
	t.Run("TestDelete", func(t *testing.T) {
		var query = queries.GetQuerySet[attrs.Definer](&Todo{}).
			Select("ID", "Title", "Description", "Done").
			Filter("Title__icontains", "test").
			Filter("Done", false)

		query.Delete()

		var latest = query.LatestQuery()
		if latest == nil {
			t.Fatalf("expected latest query, got nil")
		}

		if _, ok := latest.(queries.CompiledQuery[int64]); !ok {
			t.Fatalf("expected *QueryObject[[][]interface{}], got %T", latest)
		}

		t.Log(latest.SQL())
	})
}

func TestQuerySetChaining(t *testing.T) {
	var todos = []*Todo{
		{Title: "TestQuerySetChaining1", Description: "Description TestQuerySetChaining", Done: false},
		{Title: "TestQuerySetChaining2", Description: "Description TestQuerySetChaining", Done: true},
		{Title: "TestQuerySetChaining3", Description: "Description TestQuerySetChaining", Done: false},
	}

	for _, todo := range todos {
		if err := queries.CreateObject(todo); err != nil {
			t.Fatalf("Failed to insert todo: %v", err)
		}
	}

	var qs = queries.GetQuerySet[attrs.Definer](&Todo{}).
		Select("ID", "Title", "Description", "Done", "User").
		Filter("Title__icontains", "TestQuerySetChaining").
		Filter("Done", false)

	qs = qs.Filter("ID", expr.Value(todos[0].ID))

	todosList, err := qs.All()
	if err != nil {
		t.Fatalf("Failed to get todos: %v", err)
	}

	if len(todosList) != 1 {
		t.Fatalf("Expected 1 todo, got %d", len(todosList))
	}
}

type testQuerySet_Concurrency struct {
	idx   int
	sql   string
	args  []any
	todos []*queries.Row[attrs.Definer]
	err   error
}

func TestQuerySet_SharedInstance_Concurrency(t *testing.T) {

	var db = django.ConfigGet[drivers.Database](
		django.Global.Settings,
		django.APPVAR_DATABASE,
	)

	if _, ok := db.Driver().(*drivers.DriverSQLite); !ok {
		t.Skip("This test is only supported by SQLite driver")
	}

	queries.QUERYSET_USE_CACHE_DEFAULT = false
	var ctx = drivers.SetLogSQLContext(
		context.Background(), false,
	)
	var baseQS = queries.GetQuerySetWithContext[attrs.Definer](ctx, &Todo{}).
		Select("ID", "Title", "Description", "Done", "User").
		Filter("Done", false).
		Filter("Title__startswith", "ConcurrentTodo")

	const goroutines = 1000
	var todos = make([]*Todo, goroutines)
	for i := range goroutines {
		todo := &Todo{
			Title:       fmt.Sprintf("ConcurrentTodo %d", i),
			Description: "Testing thread safety",
			Done:        i%2 != 0,
		}

		if _, err := queries.GetQuerySetWithContext(ctx, todo).Create(todo); err != nil {
			t.Fatalf("Failed to insert todo: %v", err)
		}

		todos[i] = todo
	}

	var items = make(chan testQuerySet_Concurrency, goroutines)
	for i, todo := range todos {
		idx := i
		todo := todo
		go func(idx int, todo *Todo) {
			defer func() {
				if r := recover(); r != nil {
					items <- testQuerySet_Concurrency{
						err: fmt.Errorf("goroutine %d panicked: %v", i, r),
					}
				}
			}()

			var qs = baseQS.Clone()
			if idx%2 == 0 {
				qs = baseQS.Filter("ID", todo.ID)
			}

			todos, err := qs.All()
			items <- testQuerySet_Concurrency{
				idx:   idx,
				sql:   qs.LatestQuery().SQL(),
				args:  qs.LatestQuery().Args(),
				todos: todos,
				err:   err,
			}
		}(idx, todo)
	}

	var checkTodo = func(todo *Todo) {
		if todo == nil {
			t.Fatalf("Expected a todo, got nil")
		}

		if todo.ID == 0 {
			t.Fatalf("Expected todo ID to be not 0, got %d", todo.ID)
		}

		if todo.Title == "" {
			t.Fatalf("Expected todo title to be not empty")
		}

		if !strings.Contains(strings.ToLower(todo.Title), "concurrenttodo") {
			t.Errorf("Expected todo title to contain 'concurrenttodo', got: %s", todo.Title)
		}

		if todo.Done {
			t.Errorf("Expected todo to be not done, got done: %+v", todo)
		}
	}

	for i := 0; i < goroutines; i++ {
		var item = <-items
		if item.err != nil {
			t.Errorf("Failed to get todos: %v", item.err)
			continue
		}

		if len(item.todos) == 0 {
			t.Errorf("Expected at least 1 todo, got 0")
			continue
		}

		if len(item.todos) > 0 {
			if item.idx%2 == 0 {
				if len(item.todos) != 1 {
					t.Errorf("Expected 1 todo, got %d", len(item.todos))
				}
				checkTodo(item.todos[0].Object.(*Todo))
				continue
			}

			for _, todo := range item.todos {
				var todo = todo.Object.(*Todo)
				checkTodo(todo)
			}
		}
	}
}

func TestRecursiveAliasConflict(t *testing.T) {
	// Create deep hierarchy
	root := &Category{Name: "Root"}
	if err := queries.CreateObject(root); err != nil {
		t.Fatalf("Failed to insert root: %v", err)
	}

	child := &Category{Name: "Child", Parent: root}
	if err := queries.CreateObject(child); err != nil {
		t.Fatalf("Failed to insert child: %v", err)
	}

	grandchild := &Category{Name: "Grandchild", Parent: child}
	if err := queries.CreateObject(grandchild); err != nil {
		t.Fatalf("Failed to insert grandchild: %v", err)
	}

	// Select deeply nested field that should require distinct aliases
	qs := queries.GetQuerySet[attrs.Definer](&Category{}).
		Select("*", "Parent.ID", "Parent.Parent.*").
		Filter("Parent.Parent.Name", "Root")

	obj, err := qs.All()
	if err != nil {
		t.Fatalf("Failed to execute query: %v (%s)", err, qs.LatestQuery().SQL())
	}

	if len(obj) != 1 {
		t.Fatalf("Expected 1 object, got %d", len(obj))
	}

	var dbObj = obj[0].Object.(*Category)
	t.Logf("Created object: %+v", dbObj)

	if dbObj.Parent == nil {
		t.Fatalf("Expected Parent to be not nil")
	}

	if dbObj.Parent.Parent == nil {
		t.Fatalf("Expected Parent.Parent to be not nil")
	}

	if dbObj.Parent.Parent.ID != root.ID {
		t.Fatalf("Expected Parent.Parent.ID to be %d, got %d", root.ID, dbObj.Parent.Parent.ID)
	}

	if dbObj.Parent.Parent.Name != "Root" {
		t.Fatalf("Expected Parent.Parent.Name to be 'Root', got %q", dbObj.Parent.Parent.Name)
	}

	if dbObj.Parent.ID != child.ID {
		t.Fatalf("Expected Parent.ID to be %d, got %d", child.ID, dbObj.Parent.ID)
	}

	if dbObj.Parent.Name != "" {
		t.Fatalf("Expected Parent.Name to be '', got %q", dbObj.Parent.Name)
	}

	if dbObj.ID != grandchild.ID {
		t.Fatalf("Expected ID to be %d, got %d", grandchild.ID, dbObj.ID)
	}

	if dbObj.Name != "Grandchild" {
		t.Fatalf("Expected Name to be 'Grandchild', got %q", dbObj.Name)
	}
}

func TestQueryGroupBy(t *testing.T) {
	var todos = []*Todo{
		{Title: "GroupBy1", Description: "Description GroupBy", Done: false},
		{Title: "GroupBy2", Description: "Description GroupBy", Done: true},
		{Title: "GroupBy3", Description: "Description GroupBy", Done: false},
	}

	for _, todo := range todos {
		if err := queries.CreateObject(todo); err != nil {
			t.Fatalf("Failed to insert todo: %v", err)
		}
	}

	allTodos, err := queries.GetQuerySet(&Todo{}).Limit(10000).All()
	if err != nil {
		t.Fatalf("Failed to get all todos: %v", err)
		return
	}

	var allTodosTitleMap = make(map[string]struct{})
	for _, todo := range allTodos {

		short := strings.ToLower(todo.Object.Title)
		short = short[:2] // Get first 2 characters
		if _, exists := allTodosTitleMap[short]; exists {
			continue
		}

		allTodosTitleMap[short] = struct{}{}
	}

	var expression = expr.SUBSTR(expr.LOWER("Title"), 1, 2)
	var qs = queries.GetQuerySet(&Todo{}).
		Annotate("shortTitle", expression).
		GroupBy(expression)
	// GroupBy(expr.FuncSubstr("![Title]", 1, 2))

	grouped, err := qs.All()
	if err != nil {
		t.Fatalf("Failed to group todos: %v", err)
		return
	}

	if len(grouped) == 0 {
		t.Fatalf("Expected at least 1 group, got 0")
		return
	}

	t.Logf("Grouped todos: %d groups found, expecting %d unique titles", len(grouped), len(allTodosTitleMap))
	if len(grouped) != len(allTodosTitleMap) {
		t.Fatalf("Expected %d groups, got %d", len(allTodosTitleMap), len(grouped))
		return
	}

	for _, group := range grouped {

		var shortTitle = group.Annotations["shortTitle"]
		if shortTitle == nil {
			t.Fatalf("Expected shortTitle annotation to be not nil")
			return
		}

		if _, exists := allTodosTitleMap[shortTitle.(string)]; !exists {
			t.Fatalf("Expected shortTitle %q to be in allTodosTitleMap", shortTitle)
			return
		}

		t.Logf("Grouped todo: %+v", group.Object)
	}
}

func TestAggregateCount(t *testing.T) {
	var agg, err = queries.GetQuerySet[*Todo](&Todo{}).
		Aggregate(map[string]expr.Expression{
			"Count": expr.COUNT("ID"),
		})

	if err != nil {
		t.Fatalf("failed to aggregate: %v", err)
		return
	}

	if agg["Count"] == nil {
		t.Fatalf("expected Count to be not nil")
		return
	}

	if agg["Count"].(int64) == 0 {
		t.Fatalf("expected Count to be not 0")
		return
	}
}

func TestBulkCreate(t *testing.T) {

	var todos = []*Todo{
		{Title: "BulkCreate1", Description: "Description BulkCreate", Done: false},
		{Title: "BulkCreate2", Description: "Description BulkCreate", Done: true},
		{Title: "BulkCreate3", Description: "Description BulkCreate", Done: false},
	}

	var dbTodos, err = queries.GetQuerySet(&Todo{}).BulkCreate(todos)
	if err != nil {
		t.Fatalf("Failed to bulk create todos: %v", err)
		return
	}

	if len(dbTodos) != len(todos) {
		t.Fatalf("Expected %d todos, got %d", len(todos), len(dbTodos))
		return
	}

	var copies = []*Todo{
		{Title: "BulkCreate1", Description: "Description BulkCreate", Done: false},
		{Title: "BulkCreate2", Description: "Description BulkCreate", Done: true},
		{Title: "BulkCreate3", Description: "Description BulkCreate", Done: false},
	}

	for i, dbTodo := range dbTodos {
		if dbTodo.ID == 0 {
			t.Fatalf("Expected todo ID to be not 0, got %d", dbTodo.ID)
		}

		if dbTodo.ID != todos[i].ID {
			t.Fatalf("Expected todo ID %d, got %d", todos[i].ID, dbTodo.ID)
		}

		if dbTodo.Title != copies[i].Title {
			t.Fatalf("Expected todo title %q, got %q", copies[i].Title, dbTodo.Title)
		}

		if dbTodo.Description != copies[i].Description {
			t.Fatalf("Expected todo description %q, got %q", copies[i].Description, dbTodo.Description)
		}

		if dbTodo.Done != copies[i].Done {
			t.Fatalf("Expected todo done %v, got %v", copies[i].Done, dbTodo.Done)
		}

		t.Logf("Created todo: %+v", dbTodo)
	}
}

func TestUpdateWithCast(t *testing.T) {
	var todo = &Todo{
		Title:       "9",
		Description: "4",
	}

	if err := queries.CreateObject(todo); err != nil {
		t.Fatalf("Failed to create object: %v", err)
	}

	var updatedCnt, err = queries.Objects(todo).
		Select("Title").
		Filter("ID", todo.ID).
		Update(
			todo,
			expr.Chain(
				expr.Field("Title"),
				expr.EQ,
				expr.CastInt("Title"),
				expr.ADD,
				expr.CastInt("Description"),
			),
		)
	if err != nil {
		t.Fatalf("Failed to execute query: %v", err)
	}

	if updatedCnt != 1 {
		t.Fatalf("Expected 1 row to be updated, got %d", updatedCnt)
	}

	row, err := queries.Objects(todo).
		Select("ID", "Title", "Description").
		Filter("ID", todo.ID).
		Get()
	if err != nil {
		t.Fatalf("Failed to execute query: %v", err)
	}

	if row.Object.Title != "13" {
		t.Errorf("Expected Title = '13', got '%s'", row.Object.Title)
	}

	_, err = queries.Objects(todo).
		Filter("ID", todo.ID).
		Delete()
	if err != nil {
		t.Fatalf("Failed to delete object: %v", err)
	}
}

func TestBulkUpdate(t *testing.T) {
	var todos, err = queries.GetQuerySet(&Todo{}).BulkCreate([]*Todo{
		{Title: "BulkUpdate1", Description: "Description BulkUpdate", Done: false},
		{Title: "BulkUpdate2", Description: "Description BulkUpdate", Done: true},
		{Title: "BulkUpdate3", Description: "Description BulkUpdate", Done: false},
		{Title: "BulkUpdate4", Description: "Description BulkUpdate", Done: true},
	})
	if err != nil {
		t.Fatalf("Failed to bulk create todos: %v", err)
		return
	}

	todos[0].Title = "BulkUpdate1 Updated"
	todos[1].Title = "BulkUpdate2 Updated"
	todos[2].Title = "BulkUpdate3 Updated"

	var toUpdate = []*Todo{
		todos[0],
		todos[1],
		todos[2],
	}

	t.Logf("Todos to update 1: %s, %+v", todos[0].FieldDefs().Get("Title"), todos[0])
	t.Logf("Todos to update 2: %s, %+v", todos[1].FieldDefs().Get("Title"), todos[1])
	t.Logf("Todos to update 3: %s, %+v", todos[2].FieldDefs().Get("Title"), todos[2])
	t.Logf("Todos to update 4: %s, %+v", todos[3].FieldDefs().Get("Title"), todos[3])

	_, err = queries.GetQuerySet(&Todo{}).BulkUpdate(
		toUpdate,
		expr.As("Description", expr.UPPER("Description")),
	)

	if err != nil {
		t.Fatalf("Failed to bulk update todos: %v", err)
		return
	}

	dbTodos, err := queries.GetQuerySet(&Todo{}).Filter(
		"ID__in", todos[0].ID, todos[1], todos[2:4],
	).All()
	if err != nil {
		t.Fatalf("Failed to get todos: %v", err)
		return
	}

	if len(dbTodos) != 4 {
		t.Fatalf("Expected 4 todos, got %d", len(dbTodos))
		return
	}

	var (
		todo1 = dbTodos[0].Object
		todo2 = dbTodos[1].Object
		todo3 = dbTodos[2].Object
		todo4 = dbTodos[3].Object
	)

	t.Logf("Updated todos 1: %+v", todo1)
	t.Logf("Updated todos 2: %+v", todo2)
	t.Logf("Updated todos 3: %+v", todo3)
	t.Logf("Updated todos 4: %+v", todo4)

	var todoEquals = func(todo *Todo, dbTodo *Todo) {
		if todo.ID != dbTodo.ID {
			t.Fatalf("Expected todo ID %d, got %d", todo.ID, dbTodo.ID)
		}

		if todo.Title != dbTodo.Title {
			t.Fatalf("Expected todo title %q, got %q", todo.Title, dbTodo.Title)
		}

		if strings.ToUpper(todo.Description) != dbTodo.Description {
			t.Fatalf("Expected todo description %q, got %q", strings.ToUpper(todo.Description), dbTodo.Description)
		}

		if todo.Done != dbTodo.Done {
			t.Fatalf("Expected todo done %v, got %v", todo.Done, dbTodo.Done)
		}
	}

	todoEquals(todos[0], todo1)
	todoEquals(todos[1], todo2)
	todoEquals(todos[2], todo3)
}

func TestRunInTransaction(t *testing.T) {
	var user = &User{
		Name: "TestTransaction",
	}

	if err := queries.CreateObject(user); err != nil || user.ID == 0 {
		t.Fatalf("Failed to insert user: %v", err)
	}

	var todo = &Todo{
		Title:       "TestTransaction",
		Description: "This is a new test todo",
		Done:        false,
		User:        user,
	}

	var ctx = context.Background()
	var err = queries.RunInTransaction(ctx, func(ctx context.Context, NewQuerySet queries.ObjectsFunc[*Todo]) (bool, error) {
		var qs = NewQuerySet(&Todo{})

		if !qs.Compiler().InTransaction() {
			return false, fmt.Errorf("expected to be in transaction, but it is not")
		}

		dbTodo, err := qs.Create(todo)
		if err != nil {
			return false, fmt.Errorf("failed to create todo: %w", err)
		}
		if dbTodo == nil {
			return false, fmt.Errorf("expected a todo, got nil")
		}

		var tdo = dbTodo
		if tdo.ID == 0 {
			return false, fmt.Errorf("expected todo ID to be not 0, got %d", tdo.ID)
		}

		if tdo.Title != todo.Title {
			return false, fmt.Errorf("expected todo title %q, got %q", todo.Title, tdo.Title)
		}

		if tdo.Description != todo.Description {
			return false, fmt.Errorf("expected todo description %q, got %q", todo.Description, tdo.Description)
		}

		if tdo.Done != todo.Done {
			return false, fmt.Errorf("expected todo done %v, got %v", todo.Done, tdo.Done)
		}

		return true, nil
	})
	if err != nil {
		t.Fatalf("Failed to run transaction: %v", err)
	}
}

func TestCaseExpression(t *testing.T) {
	var todos = []*Todo{
		{Title: "CaseExpression1", Description: "Description CaseExpression", Done: false},
		{Title: "CaseExpression2", Description: "Description CaseExpression", Done: true},
		{Title: "CaseExpression3", Description: "Description CaseExpression", Done: false},
	}

	for _, todo := range todos {
		if err := queries.CreateObject(todo); err != nil {
			t.Fatalf("Failed to insert todo: %v", err)
		}
	}

	var qs = queries.GetQuerySet(&Todo{}).
		Select("ID", "Title", "Done").
		Filter("Description", "Description CaseExpression").
		Annotate("Status",
			expr.Case().
				When("Done", true).Then("Completed").
				When(expr.Logical("Done").EQ(false), "Pending").
				Default("Unknown"),
		).
		Annotate("TitleIndex",
			expr.Case(
				expr.When(
					expr.Q("Title__icontains", "NOT FOUND").
						Or(expr.Q("Title", "CaseExpression1")),
				).Then("ONE"),
				expr.When("Title", "CaseExpression2").Then("TWO"),
				expr.When("Title__iendswith", "3").Then("THREE"),
				"Unknown",
			),
		)

	var results, err = qs.All()
	if err != nil {
		t.Fatalf("Failed to get todos with case expression: %v", err)
	}

	if len(results) != len(todos) {
		t.Fatalf("Expected %d todos, got %d", len(todos), len(results))
	}

	for i, result := range results {
		var todo = result.Object

		switch todo.Done {
		case true:
			if result.Annotations["Status"] != "Completed" {
				t.Errorf("Expected Status to be 'Completed', got %s", result.Annotations["Status"])
			}
		case false:
			if result.Annotations["Status"] != "Pending" {
				t.Errorf("Expected Status to be 'Pending', got %s", result.Annotations["Status"])
			}
		}

		switch todo.Title {
		case "CaseExpression1":
			if result.Annotations["TitleIndex"] != "ONE" {
				t.Errorf("Expected TitleIndex to be 'ONE', got %s", result.Annotations["TitleIndex"])
			}
		case "CaseExpression2":
			if result.Annotations["TitleIndex"] != "TWO" {
				t.Errorf("Expected TitleIndex to be 'TWO', got %s", result.Annotations["TitleIndex"])
			}
		case "CaseExpression3":
			if result.Annotations["TitleIndex"] != "THREE" {
				t.Errorf("Expected TitleIndex to be 'THREE', got %s", result.Annotations["TitleIndex"])
			}
		default:
			if result.Annotations["TitleIndex"] != "Unknown" {
				t.Errorf("Expected TitleIndex to be 'Unknown', got %s", result.Annotations["TitleIndex"])
			}
		}

		t.Logf("Todo %d: %+v, Status: %s", i+1, todo, result.Annotations["Status"])
	}
}

func TestGetOrCreateInTransaction(t *testing.T) {
	var todo = &Todo{
		Title:       "TestGetOrCreateInTransaction",
		Description: "This is a new test todo",
		Done:        false,
	}

	var err = queries.RunInTransaction(context.Background(), func(ctx context.Context, NewQuerySet queries.ObjectsFunc[*Todo]) (bool, error) {
		var todo, created, err = NewQuerySet(&Todo{}).
			Select("ID", "Title", "Description", "Done", "User").
			Filter("Title", todo.Title).
			GetOrCreate(todo)
		if err != nil {
			return false, fmt.Errorf("failed to get or create todo: %w", err)
		}

		if !created {
			return false, fmt.Errorf("expected todo to be created, but it already exists")
		}

		if todo.ID == 0 {
			return false, fmt.Errorf("expected todo ID to be not 0, got %d", todo.ID)
		}

		if todo.Title != "TestGetOrCreateInTransaction" {
			return false, fmt.Errorf("expected todo title %q, got %q", "TestGetOrCreateInTransaction", todo.Title)
		}

		if todo.Description != "This is a new test todo" {
			return false, fmt.Errorf("expected todo description %q, got %q", "This is a new test todo", todo.Description)
		}

		if todo.Done != false {
			return false, fmt.Errorf("expected todo done %v, got %v", false, todo.Done)
		}

		t.Logf("Created or retrieved todo: %+v", todo)
		return true, nil
	})
	if err != nil {
		t.Fatalf("Failed to run transaction: %v", err)
	}
}

func TestLogicalExpression(t *testing.T) {
	//var todo = &Todo{
	//	Title:       "TestLogicalExpression",
	//	Description: "This is a new test todo",
	//	Done:        false,
	//}

	var todos, err = queries.GetQuerySet(&Todo{}).
		Select("ID", "Title", "Description", "Done", "User").
		Filter(expr.Logical("Done", expr.EQ, expr.Value(true))).All()
	if err != nil {
		t.Fatalf("Failed to get todos: %v", err)
	}

	if len(todos) == 0 {
		t.Fatalf("Expected at least 1 todo, got 0")
	}

	t.Logf("Todos with logical expression: %d todos found", len(todos))
}

func TestTransactionRollbackAfterInsert(t *testing.T) {

	var obj = &TestTransaction{
		Name: "TestTransactionRollbackAfterInsert",
	}

	//if db_tag == "mysql_local" || db_tag == "mysql" {
	//	// currently these do not support transactions on create
	//	t.Skip("local MySQL server does not support transactions")
	//}

	var ctx = context.Background()
	var err = queries.RunInTransaction(ctx, func(ctx context.Context, NewQuerySet queries.ObjectsFunc[*TestTransaction]) (bool, error) {
		var qs = NewQuerySet(&TestTransaction{})

		if !qs.Compiler().InTransaction() {
			return false, fmt.Errorf("expected query set to be in transaction")
		}

		dbTodo, err := qs.Create(obj)
		if err != nil {
			return false, fmt.Errorf("failed to create todo: %w", err)
		}
		if dbTodo == nil {
			return false, fmt.Errorf("expected a todo, got nil")
		}

		// Rollback the transaction
		return false, nil
	})
	if err != nil {
		t.Fatalf("Failed to run transaction: %v", err)
	}

	t.Logf("Transaction rolled back, no object should be created (local obj: %+v %T)", obj, obj)

	dbObj, err := queries.GetQuerySetWithContext(ctx, &TestTransaction{}).Get()
	if err == nil || !errors.Is(err, errors.NoRows) {
		t.Fatalf("Expected error, got nil: %+v", dbObj.Object)
		return
	}
}

type TestRowsAffected struct {
	ID   int64
	Name string
}

func (t *TestRowsAffected) FieldDefs() attrs.Definitions {
	return attrs.Define(t,
		attrs.Unbound("ID", &attrs.FieldConfig{
			Primary: true,
		}),
		attrs.Unbound("Name"),
	)
}

func TestWorkingRowsAffected(t *testing.T) {
	if !(testdb.ENGINE == "mysql" || testdb.ENGINE == "mysql_local" || testdb.ENGINE == "mariadb") {
		t.Skipf("Skipping test for %s database", testdb.ENGINE)
		return
	}

	var tables = quest.Table(t, &TestRowsAffected{})
	tables.Create()
	defer tables.Drop()

	var objects = []*TestRowsAffected{
		{Name: "Test1"},
		{Name: "Test2"},
		{Name: "Test3"},
	}

	var created, delete = quest.CreateObjects(t, objects...)
	defer func() {
		if err := delete(0); err != nil {
			t.Fatalf("Failed to delete objects: %v", err)
		}
	}()

	var rowsAffected, err = queries.GetQuerySet(&TestRowsAffected{}).
		Select("Name").
		Filter("Name__in", "Test1", "Test2", "Test3").
		Update(&TestRowsAffected{}, expr.As("Name", expr.UPPER("Name")))
	if err != nil {
		t.Fatalf("Failed to update objects: %v", err)
	}

	if rowsAffected != int64(len(created)) || rowsAffected != 3 {
		t.Fatalf("Expected %d rows affected, got %d", len(objects), rowsAffected)
	}

	for _, obj := range created {
		t.Logf("Updated object: %+v", obj)
	}
}

func TestQuerySetRows(t *testing.T) {
	var tables = quest.Table(t, &TestRowsAffected{})
	tables.Create()
	defer tables.Drop()

	var objects = []*TestRowsAffected{
		{Name: "TestRows1"},
		{Name: "TestRows2"},
		{Name: "TestRows3"},
	}

	for _, obj := range objects {
		if err := queries.CreateObject(obj); err != nil {
			t.Fatalf("Failed to insert object: %v", err)
		}
	}

	var qs = queries.GetQuerySet[attrs.Definer](&TestRowsAffected{})
	var rows, err = qs.Rows("SELECT ![ID], ![Name] FROM TABLE(SELF) WHERE ? = ? AND EXPR(NameFilter) AND EXPR(0) AND 1 = ?[3] AND ?[1] - ?[2] = 0",
		expr.PARSER.Expr.Expressions(
			map[string]expr.Expression{
				"NameFilter": expr.Q("Name__istartswith", "testrows"),
			},
			expr.String("1 = 1"),
		),
		expr.Value(2), expr.Value(2), expr.Value(1),
	)
	if err != nil {
		t.Fatalf("Failed to get rows: %v (%s)", err, qs.LatestQuery().SQL())
	}
	defer rows.Close()

	var count int
	for rows.Next() {
		var row TestRowsAffected
		if err := rows.Scan(&row.ID, &row.Name); err != nil {
			t.Fatalf("Failed to scan row: %v", err)
		}
		count++
		t.Logf("Row %d: %+v", count, row)
	}

	if count != len(objects) {
		t.Fatalf("Expected %d rows, got %d", len(objects), count)
	}

	_, err = queries.GetQuerySet[attrs.Definer](&TestRowsAffected{}).Delete()
	if err != nil {
		t.Fatalf("Failed to delete objects: %v", err)
	}
}

func TestQuerySetRow(t *testing.T) {
	var tables = quest.Table(t, &TestRowsAffected{})
	tables.Create()
	defer tables.Drop()

	var obj = &TestRowsAffected{
		Name: "TestQuerySetRow",
	}

	if err := queries.CreateObject(obj); err != nil {
		t.Fatalf("Failed to insert object: %v", err)
	}

	var rows = queries.GetQuerySet[attrs.Definer](&TestRowsAffected{}).
		Row("SELECT ![ID], ![Name] FROM TABLE(SELF) WHERE ![Name] = ?", "TestQuerySetRow")
	if err := rows.Err(); err != nil {
		t.Fatalf("Failed to get rows: %v", err)
	}

	var row TestRowsAffected
	if err := rows.Scan(&row.ID, &row.Name); err != nil {
		t.Fatalf("Failed to scan row: %v", err)
	}
	t.Logf("Row %+v", row)

	var _, err = queries.GetQuerySet[attrs.Definer](&TestRowsAffected{}).Delete()
	if err != nil {
		t.Fatalf("Failed to delete objects: %v", err)
	}
}

//
//	func TestFakeModel(t *testing.T) {
//
//		var m = models.FakeModel(&models.FakeModelObject{
//			Data:      make(models.MapDataStore),
//			TableName: "images",
//			FieldFuncs: []models.NewFieldFunc{
//				func(mds queries.ModelDataStore, d1 attrs.Definer, d2 attrs.Definitions) attrs.Field {
//					var (
//						iface    = 0
//						ifacePtr = &iface
//					)
//					mds.SetValue("ID", ifacePtr)
//					return fields.NewDataModelField[int](
//						d1, ifacePtr, "ID",
//					)
//				},
//				func(mds queries.ModelDataStore, d1 attrs.Definer, d2 attrs.Definitions) attrs.Field {
//					var (
//						iface    = ""
//						ifacePtr = &iface
//					)
//					mds.SetValue("Path", ifacePtr)
//					return fields.NewDataModelField[string](
//						d1, ifacePtr, "Path",
//					)
//				},
//			},
//		})
//
//		var newObj = m.InitNew()
//		var defs = newObj.FieldDefs()
//		defs.Set("Path", "/path/to/fake/image.jpg")
//		var err = queries.CreateObject(newObj)
//		if err != nil {
//			t.Fatalf("Failed to create fake model: %v", err)
//		}
//
//		dbModel, err := queries.GetQuerySet(&Image{}).
//			Select("ID", "Path").
//			Filter("Path", "/path/to/fake/image.jpg").
//			Get()
//		if err != nil {
//			t.Fatalf("Failed to get fake model: %v", err)
//		}
//
//		if dbModel.Object.ID != defs.Get("ID").(int) {
//			t.Fatalf("Expected ID %d, got %d", defs.Get("ID").(int), dbModel.Object.ID)
//		}
//	}
//
