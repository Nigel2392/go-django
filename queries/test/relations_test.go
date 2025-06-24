package queries_test

import (
	"fmt"
	"reflect"
	"testing"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/quest"
	"github.com/Nigel2392/go-django/src/core/attrs"
)

type relationTestExpected struct {
	type_ attrs.RelationType
	final reflect.Type
}

type relationTest struct {
	name       string
	model      attrs.Definer
	fieldDefs  attrs.Definitions
	expectsFwd map[string]relationTestExpected
	expectsRev map[string]relationTestExpected
}

func getType(obj any) reflect.Type {
	return reflect.TypeOf(obj)
}

var tests = []relationTest{
	{
		name:  "ExpectedForwardRelation",
		model: &Category{},
		expectsFwd: map[string]relationTestExpected{
			"Parent": {
				type_: attrs.RelManyToOne,
				final: getType(&Category{}),
			},
		},
		expectsRev: map[string]relationTestExpected{
			"CategorySet": {
				type_: attrs.RelOneToMany,
				final: getType(&Category{}),
			},
		},
	},
	{
		name:  "ExpectedReverseRelation",
		model: &Todo{},
		expectsFwd: map[string]relationTestExpected{
			"User": {
				type_: attrs.RelOneToOne,
				final: getType(&User{}),
			},
		},
		expectsRev: map[string]relationTestExpected{},
	},
	{
		name:  "ExpectedReverseRelation",
		model: &User{},
		expectsRev: map[string]relationTestExpected{
			"Todo": {
				type_: attrs.RelOneToOne,
				final: getType(&Todo{}),
			},
		},
	},
}

func TestRegisterModelRelations(t *testing.T) {

	for _, test := range tests {
		test.fieldDefs = test.model.FieldDefs()
		t.Run(test.name, func(t *testing.T) {
			attrs.RegisterModel(test.model)
			meta := attrs.GetModelMeta(test.model)

			for field, exp := range test.expectsFwd {
				rel, ok := meta.Forward(field)
				if !ok {
					t.Errorf("expected forward relation for field %q", field)
					continue
				}

				_, ok = test.fieldDefs.Field(field)
				if !ok {
					t.Errorf("expected field %q in model %T", field, test.model)
					continue
				}

				if rel.Type() != exp.type_ {
					t.Errorf("expected forward relation type %v for %q, got %v", exp.type_, field, rel.Type())
				}

				if reflect.TypeOf(rel.Model()) != exp.final {
					t.Errorf("expected final model type %v for %q, got %v", exp.final, field, reflect.TypeOf(rel.Model()))
				}
			}

			for field, exp := range test.expectsRev {
				rel, ok := meta.Reverse(field)
				if !ok {
					t.Errorf("expected reverse relation for field %q", field)
					continue
				}

				if rel.Type() != exp.type_ {
					t.Errorf("expected reverse relation type %v for %q, got %v", exp.type_, field, rel.Type())
				}

				_, ok = test.fieldDefs.Field(field)
				if !ok {
					t.Errorf("expected field %q in model %T", field, test.model)
					continue
				}

				if reflect.TypeOf(rel.Model()) != exp.final {
					t.Errorf("expected final model type %v for %q, got %v", exp.final, field, reflect.TypeOf(rel.Model()))
				}
			}

			t.Logf("model %T has %d forward relations and %d reverse relations", test.model, meta.ForwardMap().Len(), meta.ReverseMap().Len())
			for head := meta.ForwardMap().Front(); head != nil; head = head.Next() {
				field := head.Key
				rel := head.Value
				if rel == nil {
					t.Errorf("expected forward relation %q, got nil", field)
					continue
				}
				model := rel.Model()
				f := rel.Field()
				if f == nil {
					t.Errorf("expected forward relation %q, got nil", field)
					continue
				}
				t.Logf("forward relation %q: %T.%s", field, model, f.Name())
			}
			for head := meta.ReverseMap().Front(); head != nil; head = head.Next() {
				field := head.Key
				rel := head.Value
				if rel == nil {
					t.Errorf("expected reverse relation %q, got nil", field)
					continue
				}
				model := rel.Model()
				f := rel.Field()
				if f == nil {
					t.Errorf("expected reverse relation %q, got nil", field)
					continue
				}

				t.Logf("reverse relation %q: %T.%s", field, model, f.Name())
			}
		})
	}
}

func TestReverseRelations(t *testing.T) {
	var user = &User{
		Name: "TestReverseRelations",
	}

	if err := queries.CreateObject(user); err != nil {
		t.Errorf("expected no error, got %v", err)
		return
	}

	var meta = attrs.GetModelMeta(user)
	t.Logf("model %T has %d forward relations and %d reverse relations", user, meta.ForwardMap().Len(), meta.ReverseMap().Len())
	for head := meta.ForwardMap().Front(); head != nil; head = head.Next() {
		field := head.Key
		rel := head.Value
		t.Logf("forward relation %q: %T.%s", field, rel.Model(), rel.Field().Name())
	}
	for head := meta.ReverseMap().Front(); head != nil; head = head.Next() {
		field := head.Key
		rel := head.Value
		t.Logf("reverse relation %q: %T.%s", field, rel.Model(), rel.Field().Name())
	}

	var todo = &Todo{
		Title:       "TestReverseRelations",
		Description: "TestReverseRelations",
		Done:        false,
		User:        user,
	}

	if err := queries.CreateObject(todo); err != nil {
		t.Errorf("expected no error, got %v", err)
		return
	}

	var u = &User{}
	var defs = u.FieldDefs()
	var _, ok = defs.Field("Todo")
	if !ok {
		t.Errorf("expected field Todo, got nil")
		return
	}

	var q = queries.Objects[attrs.Definer](&User{}).
		Select("ID", "Name", "Todo.*").
		Filter("ID", user.ID)
	var dbTodo, err = q.First()
	if err != nil {
		t.Errorf("expected no error, got %v (%s)", err, q.LatestQuery().SQL())
		return
	}

	if dbTodo == nil {
		t.Errorf("expected todo not nil, got nil")
		return
	}

	// fields
	if dbTodo.Object.(*User).ID != user.ID {
		t.Errorf("expected todo ID %d, got %d", user.ID, dbTodo.Object.(*User).ID)
		return
	}

	if dbTodo.Object.(*User).Name != user.Name {
		t.Errorf("expected todo Name %q, got %q", user.Name, dbTodo.Object.(*User).Name)
		return
	}

	// Todo.*
	todoSet, ok := dbTodo.Object.(*User).FieldDefs().Field("Todo")
	if !ok {
		t.Errorf("expected todoSet field, got nil")
		return
	}

	if todoSet == nil {
		t.Errorf("expected todoSet not nil, got nil")
		return
	}

	var val, isOk = todoSet.GetValue().(*Todo)
	if val == nil || !isOk {
		t.Errorf("expected todoSet value not nil, got %v", val)
		return
	}

	if val.ID != todo.ID {
		t.Errorf("expected todoSet ID %d, got %d", todo.ID, val.ID)
		return
	}

	if val.Title != todo.Title {
		t.Errorf("expected todoSet Title %q, got %q", todo.Title, val.Title)
		return
	}

	if val.Description != todo.Description {
		t.Errorf("expected todoSet Description %q, got %q", todo.Description, val.Description)
		return
	}

	if val.Done != todo.Done {
		t.Errorf("expected todoSet Done %v, got %v", todo.Done, val.Done)
		return
	}

	// Todo.User.*
	if val.User == nil {
		t.Errorf("expected todoSet User not nil, got nil")
		return
	}

	if val.User.ID != user.ID {
		t.Errorf("expected todoSet User ID %d, got %d", user.ID, val.User.ID)
		return
	}

	if val.User.Name != "" {
		t.Errorf("expected todoSet User Name %q, got %q", "", val.User.Name)
		return
	}
}

func TestReverseRelationsNested(t *testing.T) {
	var user = &User{
		Name: "TestReverseRelationsNested",
	}

	if err := queries.CreateObject(user); err != nil {
		t.Errorf("expected no error, got %v", err)
		return
	}

	var todo = &Todo{
		Title:       "TestReverseRelationsNested",
		Description: "TestReverseRelationsNested",
		Done:        false,
		User:        user,
	}

	if err := queries.CreateObject(todo); err != nil {
		t.Errorf("expected no error, got %v", err)
		return
	}

	var u = &User{}
	var defs = u.FieldDefs()
	var _, ok = defs.Field("Todo")
	if !ok {
		t.Errorf("expected field Todo, got nil")
		return
	}

	var q = queries.Objects[attrs.Definer](&User{}).
		Select("ID", "Name", "Todo.*", "Todo.User.*", "Todo.User.Todo.*", "Todo.User.Todo.User.*").
		Filter("ID", user.ID).
		Filter("Todo.ID", todo.ID).
		Filter("Todo.User.ID", user.ID).
		Filter("Todo.User.Todo.ID", todo.ID).
		Filter("Todo.User.Todo.User.ID", user.ID)

	var dbTodo, err = q.First()
	if err != nil {
		t.Errorf("expected no error, got %v (%s)", err, q.LatestQuery().SQL())
		return
	}

	if dbTodo == nil {
		t.Errorf("expected todo not nil, got nil")
		return
	}

	// fields
	if dbTodo.Object.(*User).ID != user.ID {
		t.Errorf("expected todo ID %d, got %d", user.ID, dbTodo.Object.(*User).ID)
		return
	}

	if dbTodo.Object.(*User).Name != user.Name {
		t.Errorf("expected todo Name %q, got %q", user.Name, dbTodo.Object.(*User).Name)
		return
	}

	// Todo.*
	todoSet, ok := dbTodo.Object.(*User).FieldDefs().Field("Todo")
	if !ok {
		t.Errorf("expected todoSet field, got nil")
		return
	}

	if todoSet == nil {
		t.Errorf("expected todoSet not nil, got nil")
		return
	}

	var val, isOk = todoSet.GetValue().(*Todo)
	if val == nil || !isOk {
		t.Errorf("expected todoSet value not nil, got %v", val)
		return
	}

	if val.ID != todo.ID {
		t.Errorf("expected todoSet ID %d, got %d", todo.ID, val.ID)
		return
	}

	if val.Title != todo.Title {
		t.Errorf("expected todoSet Title %q, got %q", todo.Title, val.Title)
		return
	}

	if val.Description != todo.Description {
		t.Errorf("expected todoSet Description %q, got %q", todo.Description, val.Description)
		return
	}

	if val.Done != todo.Done {
		t.Errorf("expected todoSet Done %v, got %v", todo.Done, val.Done)
		return
	}

	// Todo.User.*
	if val.User == nil {
		t.Errorf("expected todoSet User not nil, got nil")
		return
	}

	if val.User.ID != user.ID {
		t.Errorf("expected todoSet User ID %d, got %d", user.ID, val.User.ID)
		return
	}

	if val.User.Name != user.Name {
		t.Errorf("expected todoSet User Name %q, got %q", user.Name, val.User.Name)
		return
	}

	// Todo.User.Todo.*
	todoSet, ok = val.User.FieldDefs().Field("Todo")
	if !ok {
		t.Errorf("expected user.todoSet field, got nil")
		return
	}

	if todoSet == nil {
		t.Errorf("expected user.todoSet not nil, got nil")
		return
	}

	val, isOk = todoSet.GetValue().(*Todo)
	if val == nil || !isOk {
		t.Errorf("expected user.todoSet value not nil, got %v", val)
		return
	}

	if val.ID != todo.ID {
		t.Errorf("expected user.todoSet ID %d, got %d", todo.ID, val.ID)
		return
	}

	if val.Title != todo.Title {
		t.Errorf("expected user.todoSet Title %q, got %q", todo.Title, val.Title)
		return
	}

	if val.Description != todo.Description {
		t.Errorf("expected user.todoSet Description %q, got %q", todo.Description, val.Description)
		return
	}

	if val.Done != todo.Done {
		t.Errorf("expected user.todoSet Done %v, got %v", todo.Done, val.Done)
		return
	}

	// Todo.User.Todo.User.*
	if val.User == nil {
		t.Errorf("expected user.todoSet User not nil, got nil")
		return
	}

	if val.User.ID != user.ID {
		t.Errorf("expected user.todoSet User ID %d, got %d", user.ID, val.User.ID)
		return
	}

	if val.User.Name != user.Name {
		t.Errorf("expected user.todoSet User Name %q, got %q", user.Name, val.User.Name)
		return
	}

	todoSet, ok = val.User.FieldDefs().Field("Todo")
	if !ok {
		t.Errorf("expected user.todoSet field, got nil")
		return
	}

	if todoSet == nil {
		t.Errorf("expected user.todoSet not nil, got nil")
		return
	}
}
func TestOneToOneWithThrough(t *testing.T) {
	// Create the target
	target := &OneToOneWithThrough_Target{
		Name: "Target Name",
		Age:  42,
	}
	if err := queries.CreateObject(target); err != nil {
		t.Fatalf("failed to create target: %v", err)
	}

	// Create the main object
	main := &OneToOneWithThrough{
		Title: "Main Title",
	}
	if err := queries.CreateObject(main); err != nil {
		t.Fatalf("failed to create main: %v", err)
	}

	// Create the through relation manually
	through := &OneToOneWithThrough_Through{
		SourceModel: main,
		TargetModel: target,
	}
	if err := queries.CreateObject(through); err != nil {
		t.Fatalf("failed to create through: %v", err)
	}

	// Query and include the through-relation
	var q = queries.Objects[attrs.Definer](&OneToOneWithThrough{}).
		Select("ID", "Title", "Target.*").
		Filter("ID", main.ID)

	result, err := q.First()
	if err != nil {
		t.Fatalf("query failed: %v (%s)", err, q.LatestQuery().SQL())
	}
	if result == nil {
		t.Fatalf("expected result, got nil")
	}

	obj := result.Object.(*OneToOneWithThrough)
	if obj.Title != main.Title {
		t.Errorf("expected title %q, got %q", main.Title, obj.Title)
	}

	if obj.Through.Object == nil {
		t.Fatalf("expected Through field not nil")
	}

	var targetVal = obj.Through.Object
	if targetVal.ID != target.ID {
		t.Errorf("expected target ID %d, got %d", target.ID, targetVal.ID)
	}
	if targetVal.Name != target.Name {
		t.Errorf("expected target Name %q, got %q", target.Name, targetVal.Name)
	}
	if targetVal.Age != target.Age {
		t.Errorf("expected target Age %d, got %d", target.Age, targetVal.Age)
	}

	t.Logf("OneToOneWithThrough test passed (object): %+v", obj.Through.Object)
	t.Logf("OneToOneWithThrough test passed (through): %+v", obj.Through.ThroughObject)
}

func TestOneToOneWithThroughReverse(t *testing.T) {
	target := &OneToOneWithThrough_Target{
		Name: "ReverseTarget",
		Age:  30,
	}
	if err := queries.CreateObject(target); err != nil {
		t.Fatalf("failed to create target: %v", err)
	}

	main := &OneToOneWithThrough{
		Title: "ReverseMain",
	}
	if err := queries.CreateObject(main); err != nil {
		t.Fatalf("failed to create main: %v", err)
	}

	through := &OneToOneWithThrough_Through{
		SourceModel: main,
		TargetModel: target,
	}
	if err := queries.CreateObject(through); err != nil {
		t.Fatalf("failed to create through: %v", err)
	}

	// Now test reverse relation (Target → Main)
	result, err := queries.Objects[attrs.Definer](&OneToOneWithThrough_Target{}).
		Select("ID", "Name", "TargetReverse.*"). // TargetReverse is the reverse field name
		Filter("ID", target.ID).
		First()
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}

	obj := result.Object.(*OneToOneWithThrough_Target)
	if obj.Name != target.Name {
		t.Errorf("expected name %q, got %q", target.Name, obj.Name)
	}

	reverseVal, ok := obj.FieldDefs().Field("TargetReverse")
	if !ok || reverseVal == nil {
		t.Fatalf("expected reverse field, got nil")
	}

	sourceRel := reverseVal.GetValue().(queries.Relation)
	if sourceRel == nil {
		t.Fatalf("expected source, got nil")
	}

	source := sourceRel.Model().(*OneToOneWithThrough)
	if source.ID != main.ID {
		t.Errorf("expected reverse ID %d, got %d", main.ID, source.ID)
	}
	if source.Title != main.Title {
		t.Errorf("expected reverse title %q, got %q", main.Title, source.Title)
	}
}

func TestOneToOneWithThroughReverseIntoForward(t *testing.T) {
	target := &OneToOneWithThrough_Target{
		Name: "ReverseTarget",
		Age:  30,
	}
	if err := queries.CreateObject(target); err != nil {
		t.Fatalf("failed to create target: %v", err)
	}

	user := &User{
		Name: "TestOneToOneWithThroughReverseIntoForward",
	}
	if err := queries.CreateObject(user); err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	main := &OneToOneWithThrough{
		Title: "ReverseMain",
		User:  user,
	}
	if err := queries.CreateObject(main); err != nil {
		t.Fatalf("failed to create main: %v", err)
	}

	through := &OneToOneWithThrough_Through{
		SourceModel: main,
		TargetModel: target,
	}
	if err := queries.CreateObject(through); err != nil {
		t.Fatalf("failed to create through: %v", err)
	}

	// Now test reverse relation (Target → Main)
	result, err := queries.Objects[attrs.Definer](&OneToOneWithThrough_Target{}).
		Select("ID", "Name", "TargetReverse.*", "TargetReverse.User.*"). // TargetReverse is the reverse field name
		Filter("ID", target.ID).
		First()
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}

	obj := result.Object.(*OneToOneWithThrough_Target)
	if obj.Name != target.Name {
		t.Errorf("expected name %q, got %q", target.Name, obj.Name)
	}

	reverseVal, ok := obj.FieldDefs().Field("TargetReverse")
	if !ok || reverseVal == nil {
		t.Fatalf("expected reverse field, got nil")
	}

	sourceRel := reverseVal.GetValue().(queries.Relation)
	if sourceRel == nil {
		t.Fatalf("expected source, got nil")
	}

	source := sourceRel.Model().(*OneToOneWithThrough)
	if source.ID != main.ID {
		t.Errorf("expected reverse ID %d, got %d", main.ID, source.ID)
	}
	if source.Title != main.Title {
		t.Errorf("expected reverse title %q, got %q", main.Title, source.Title)
	}
	if source.User == nil {
		t.Fatalf("expected source user, got nil")
	}

	if source.User.ID != user.ID {
		t.Errorf("expected source user ID %d, got %d", user.ID, source.User.ID)
	}

	if source.User.Name != user.Name {
		t.Errorf("expected source user name %q, got %q", user.Name, source.User.Name)
	}
}

func TestOneToOneWithThroughNested(t *testing.T) {
	target := &OneToOneWithThrough_Target{
		Name: "NestedTarget",
		Age:  25,
	}
	if err := queries.CreateObject(target); err != nil {
		t.Fatalf("create target: %v", err)
	}

	main := &OneToOneWithThrough{
		Title: "NestedMain",
	}
	if err := queries.CreateObject(main); err != nil {
		t.Fatalf("create main: %v", err)
	}

	through := &OneToOneWithThrough_Through{
		SourceModel: main,
		TargetModel: target,
	}
	if err := queries.CreateObject(through); err != nil {
		t.Fatalf("create through: %v", err)
	}

	// Nested: Target → Reverse → Target
	result, err := queries.Objects[attrs.Definer](&OneToOneWithThrough_Target{}).
		Select("ID", "Name", "TargetReverse.*", "TargetReverse.Target.*").
		Filter("ID", target.ID).
		First()

	if err != nil {
		t.Fatalf("nested query failed: %v", err)
	}

	obj := result.Object.(*OneToOneWithThrough_Target)

	reverse, ok := obj.FieldDefs().Field("TargetReverse")
	if !ok || reverse == nil {
		t.Fatalf("expected Reverse relation")
	}
	mainObjRel := reverse.GetValue().(queries.Relation)
	if mainObjRel == nil {
		t.Fatalf("expected main from reverse")
	}

	mainObj := mainObjRel.Model().(*OneToOneWithThrough)
	relatedTarget := mainObj.Through
	if relatedTarget.Object == nil || relatedTarget.Object.ID != target.ID {
		t.Errorf("expected reloaded target ID %d, got %v", target.ID, relatedTarget)
	}
}

func TestOneToOneWithThroughDoubleNested(t *testing.T) {
	target := &OneToOneWithThrough_Target{
		Name: "DoubleNestedTarget",
		Age:  25,
	}
	if err := queries.CreateObject(target); err != nil {
		t.Fatalf("create target: %v", err)
	}

	main := &OneToOneWithThrough{
		Title: "DoubleNestedMain",
	}
	if err := queries.CreateObject(main); err != nil {
		t.Fatalf("create main: %v", err)
	}

	through := &OneToOneWithThrough_Through{
		SourceModel: main,
		TargetModel: target,
	}
	if err := queries.CreateObject(through); err != nil {
		t.Fatalf("create through: %v", err)
	}

	// Nested: Target → Reverse → Target
	result, err := queries.Objects[attrs.Definer](&OneToOneWithThrough_Target{}).
		Select("ID", "Name",
			"TargetReverse.*",
			"TargetReverse.Target.*",
			"TargetReverse.Target.TargetReverse.*",
			"TargetReverse.Target.TargetReverse.Target.*",
		).
		Filter("ID", target.ID).
		First()

	if err != nil {
		t.Fatalf("nested query failed: %v", err)
	}

	obj := result.Object.(*OneToOneWithThrough_Target)

	var objDefs = obj.FieldDefs()
	reverse, ok := objDefs.Field("TargetReverse")
	if !ok || reverse == nil {
		t.Fatalf("expected Reverse relation")
	}
	mainObjRel, ok := reverse.GetValue().(queries.Relation)
	if mainObjRel == nil || !ok {
		t.Fatalf("expected main from reverse")
	}

	mainObj := mainObjRel.Model().(*OneToOneWithThrough)
	relatedTarget := mainObj.Through
	if relatedTarget.Object == nil || relatedTarget.Object.ID != target.ID {
		t.Errorf("expected reloaded target ID %d, got %v", target.ID, relatedTarget)
	}

	relatedReverse, ok := relatedTarget.Object.FieldDefs().Field("TargetReverse")
	if !ok || relatedReverse == nil {
		t.Fatalf("expected Reverse relation")
		return
	}

	relatedMainRel := relatedReverse.GetValue().(queries.Relation)
	if relatedMainRel == nil {
		t.Fatalf("expected main from reverse")
		return
	}

	relatedMain := relatedMainRel.Model().(*OneToOneWithThrough)
	relatedRelatedTarget := relatedMain.Through
	if relatedRelatedTarget.Object == nil || relatedRelatedTarget.Object.ID != target.ID {
		t.Errorf("expected reloaded target ID %d, got %v", target.ID, relatedRelatedTarget)
		return
	}
}

type ManyToManyTest struct {
	Name string
	Test func(t *testing.T, profiles []*Profile, userIDs []*User, m2m_sources []*ModelManyToMany, m2m_targets []*ModelManyToMany_Target, m2m_throughs []*ModelManyToMany_Through) (int, int, int, int, int)
}

var manyToManyTests = []ManyToManyTest{
	{
		Name: "TestManyToOne_Reverse",
		Test: func(t *testing.T, profiles []*Profile, users []*User, m2m_sources []*ModelManyToMany, m2m_targets []*ModelManyToMany_Target, m2m_throughs []*ModelManyToMany_Through) (int, int, int, int, int) {
			var rows, err = queries.Objects[*User](&User{}).
				Select("*", "ModelManyToManySet.*", "ModelManyToManySet.User.*", "ModelManyToManySet.User.Profile.*").
				Filter("ID__in", users[0].ID, users[1].ID).
				OrderBy("ID", "ModelManyToManySet.ID", "ModelManyToManySet.User.ID").
				All()
			if err != nil {
				t.Fatalf("Failed to get objects: %v", err)
			}

			var user1 = rows[0].Object
			t.Logf("User 1: %+v", user1)
			t.Logf("ModelManyToManySet 1: %+v %p %T", user1.ModelManyToManySet, user1.ModelManyToManySet, user1.ModelManyToManySet)
			t.Logf("User 1 ManyToManySet: %+v %+v", user1.ModelManyToManySet.Parent, user1.ModelManyToManySet.AsList())

			var set = user1.ModelManyToManySet.AsList()
			if len(set) != 2 {
				t.Errorf("Expected 2 items in set, got %d", len(set))
				for _, obj := range set {
					t.Logf("obj: %+v", obj)
				}
				t.FailNow()
			}

			var (
				target1 = set[0].(*ModelManyToMany)
				target2 = set[1].(*ModelManyToMany)
			)

			if target1.ID != m2m_sources[0].ID {
				t.Fatalf("Expected ModelManyToMany1.ID to be %d, got %d", m2m_sources[0].ID, target1.ID)
			}

			if target2.ID != m2m_sources[1].ID {
				t.Fatalf("Expected ModelManyToMany2.ID to be %d, got %d", m2m_sources[1].ID, target2.ID)
			}

			if target1.Title != "TestManyToMany1" {
				t.Fatalf("Expected ModelManyToMany1.Title to be %q, got %q", "TestManyToMany1", target1.Title)
			}

			if target2.Title != "TestManyToMany2" {
				t.Fatalf("Expected ModelManyToMany2.Title to be %q, got %q", "TestManyToMany2", target2.Title)
			}

			if target1.User.ID != user1.ID && target1.User.ID != users[0].ID {
				t.Fatalf("Expected ModelManyToMany1.User.ID to be %d, got %d", users[0].ID, target1.User.ID)
			}

			if target2.User.ID != user1.ID && target2.User.ID != users[0].ID {
				t.Fatalf("Expected ModelManyToMany2.User.ID to be %d, got %d", users[0].ID, target2.User.ID)
			}

			if target1.User.Name != "TestManyToManyUser1" {
				t.Fatalf("Expected ModelManyToMany1.User.Name to be %q, got %q", "TestManyToManyUser1", target1.User.Name)
			}

			if target2.User.Name != "TestManyToManyUser1" {
				t.Fatalf("Expected ModelManyToMany2.User.Name to be %q, got %q", "TestManyToManyUser1", target2.User.Name)
			}

			var user2 = rows[1].Object
			var m2mSet, ok = user2.DataStore().GetValue("ModelManyToManySet")
			if !ok {
				t.Fatalf("Expected ModelManyToManySet to be set: %+v\n\t%s", user2.Model, rows[1].QuerySet.LatestQuery().SQL())
			}

			set = m2mSet.([]attrs.Definer)
			if len(set) != 1 {
				t.Errorf("Expected 1 items in set, got %d", len(set))
				for _, obj := range set {
					t.Logf("obj: %+v", obj)
				}
				t.FailNow()
			}

			var target3 = set[0].(*ModelManyToMany)

			if target3.ID != m2m_sources[2].ID {
				t.Fatalf("Expected ModelManyToMany3.ID to be %d, got %d", m2m_sources[2].ID, target3.ID)
				return 0, 0, 0, 0, 0
			}

			if target3.Title != "TestManyToMany3" {
				t.Fatalf("Expected ModelManyToMany3.Title to be %q, got %q", "TestManyToMany3", target3.Title)
			}

			if target3.User.ID != user2.ID && target3.User.ID != users[1].ID {
				t.Fatalf("Expected ModelManyToMany3.User.ID to be %d, got %d", users[1].ID, target3.User.ID)
			}

			if target3.User.Name != "TestManyToManyUser2" {
				t.Fatalf("Expected ModelManyToMany3.User.Name to be %q, got %q", "TestManyToManyUser2", target3.User.Name)
			}

			t.Logf("User 2: %+v", user2)
			t.Logf("ModelManyToManySet 2:  %+v %p %T", m2mSet, m2mSet, m2mSet)

			t.Logf("________________________________________________________")
			return 0, 0, 0, 0, 0
		},
	},
	{
		Name: "Test_Reverse_ManyToOne_Reverse",
		Test: func(t *testing.T, profiles []*Profile, users []*User, m2m_sources []*ModelManyToMany, m2m_targets []*ModelManyToMany_Target, m2m_throughs []*ModelManyToMany_Through) (int, int, int, int, int) {
			var rows, err = queries.Objects[*Profile](&Profile{}).
				Select("*", "User.*", "User.ModelManyToManySet.*", "User.ModelManyToManySet.User.*").
				Filter("ID__in", profiles[0].ID, profiles[1].ID).
				OrderBy("ID", "User.ID", "User.ModelManyToManySet.ID", "User.ModelManyToManySet.User.ID").
				All()
			if err != nil {
				t.Fatalf("Failed to get objects: %v", err)
			}

			var (
				profile1  = rows[0].Object
				user1, ok = profile1.DataStore().GetValue("User")
			)
			if !ok {
				t.Fatalf("Expected User to be set: %+v\n\t%s", profile1.Model, rows[0].QuerySet.LatestQuery().SQL())
			}

			t.Logf("Profile 1: %+v", profile1)
			t.Logf("User 1:  %+v %p %T", user1, user1, user1)

			m2mSet, ok := user1.(*User).DataStore().GetValue("ModelManyToManySet")
			if !ok {
				t.Fatalf("Expected ModelManyToManySet to be set: %+v\n\t%s", user1.(*User).Model, rows[0].QuerySet.LatestQuery().SQL())
			}

			var set = m2mSet.([]attrs.Definer)
			if len(set) != 2 {
				t.Errorf("Expected 2 items in set, got %d", len(set))
				for _, obj := range set {
					t.Logf("obj: %+v", obj)
				}
				t.FailNow()
			}

			var (
				target1 = set[0].(*ModelManyToMany)
				target2 = set[1].(*ModelManyToMany)
			)

			if target1.ID != m2m_sources[0].ID {
				t.Fatalf("Expected ModelManyToMany1.ID to be %d, got %d", m2m_sources[0].ID, target1.ID)
			}

			if target2.ID != m2m_sources[1].ID {
				t.Fatalf("Expected ModelManyToMany2.ID to be %d, got %d", m2m_sources[1].ID, target2.ID)
			}

			if target1.Title != "TestManyToMany1" {
				t.Fatalf("Expected ModelManyToMany1.Title to be %q, got %q", "TestManyToMany1", target1.Title)
			}

			if target2.Title != "TestManyToMany2" {
				t.Fatalf("Expected ModelManyToMany2.Title to be %q, got %q", "TestManyToMany2", target2.Title)
			}

			if target1.User.ID != user1.(*User).ID && target1.User.ID != users[0].ID {
				t.Fatalf("Expected ModelManyToMany1.User.ID to be %d, got %d", users[0].ID, target1.User.ID)
			}

			if target2.User.ID != user1.(*User).ID && target2.User.ID != users[0].ID {
				t.Fatalf("Expected ModelManyToMany2.User.ID to be %d, got %d", users[0].ID, target2.User.ID)
			}

			if target1.User.Name != "TestManyToManyUser1" {
				t.Fatalf("Expected ModelManyToMany1.User.Name to be %q, got %q", "TestManyToManyUser1", target1.User.Name)
			}

			if target2.User.Name != "TestManyToManyUser1" {
				t.Fatalf("Expected ModelManyToMany2.User.Name to be %q, got %q", "TestManyToManyUser1", target2.User.Name)
			}

			var profile2 = rows[1].Object

			user2, ok := profile2.DataStore().GetValue("User")
			if !ok {
				t.Fatalf("Expected User to be set: %+v\n\t%s", profile2.Model, rows[1].QuerySet.LatestQuery().SQL())
			}

			t.Logf("Profile 2: %+v", profile2)
			t.Logf("User 2:  %+v %p %T", user2, user2, user2)

			m2mSet, ok = user2.(*User).DataStore().GetValue("ModelManyToManySet")
			if !ok {
				t.Fatalf("Expected ModelManyToManySet to be set: %+v\n\t%s", profile2.Model, rows[1].QuerySet.LatestQuery().SQL())
			}

			set = m2mSet.([]attrs.Definer)
			if len(set) != 1 {
				t.Errorf("Expected 1 items in set, got %d", len(set))
				for _, obj := range set {
					t.Logf("obj: %+v", obj)
				}
				t.FailNow()
			}

			var target3 = set[0].(*ModelManyToMany)

			if target3.ID != m2m_sources[2].ID {
				t.Fatalf("Expected ModelManyToMany3.ID to be %d, got %d", m2m_sources[2].ID, target3.ID)
				return 0, 0, 0, 0, 0
			}

			if target3.Title != "TestManyToMany3" {
				t.Fatalf("Expected ModelManyToMany3.Title to be %q, got %q", "TestManyToMany3", target3.Title)
			}

			if target3.User.ID != user2.(*User).ID && target3.User.ID != users[1].ID {
				t.Fatalf("Expected ModelManyToMany3.User.ID to be %d, got %d", users[1].ID, target3.User.ID)
			}

			if target3.User.Name != "TestManyToManyUser2" {
				t.Fatalf("Expected ModelManyToMany3.User.Name to be %q, got %q", "TestManyToManyUser2", target3.User.Name)
			}

			t.Logf("User 2: %+v", user2)
			t.Logf("ModelManyToManySet 2:  %+v %p %T", m2mSet, m2mSet, m2mSet)

			t.Logf("________________________________________________________")
			return 0, 0, 0, 0, 0
		},
	},
	{
		Name: "TestManyToMany_Forward",
		Test: func(t *testing.T, profiles []*Profile, users []*User, m2m_sources []*ModelManyToMany, m2m_targets []*ModelManyToMany_Target, m2m_throughs []*ModelManyToMany_Through) (int, int, int, int, int) {
			var rows, err = queries.Objects(&ModelManyToMany{}).
				Select("*", "Target.*").
				Filter("ID__in", m2m_sources[0].ID, m2m_sources[1].ID, m2m_sources[2].ID).
				OrderBy("ID", "Target").
				All()
			if err != nil {
				t.Fatalf("Failed to get objects: %v", err)
			}

			if len(rows) != 3 {
				t.Fatalf("Expected 3 rows, got %d", len(rows))
			}

			var hasTarget = func(row *queries.Row[*ModelManyToMany]) bool {
				var t, ok = row.Object.DataStore().GetValue("Target")
				return ok && t != nil
			}

			var (
				row1 = rows[0]
				row2 = rows[1]
				row3 = rows[2]
			)

			t.Logf("Row 1: %+v", row1.Object)
			t.Logf("Row 2: %+v", row2.Object)
			t.Logf("Row 3: %+v", row3.Object)
			t.Logf("________________________________________________________")

			if !hasTarget(row1) {
				t.Fatalf("Expected row1 to have a target, got nil")
			}

			if !hasTarget(row2) {
				t.Fatalf("Expected row2 to have a target, got nil")
			}

			if !hasTarget(row3) {
				t.Fatalf("Expected row3 to have a target, got nil")
			}

			var (
				target1, _ = row1.Object.DataStore().GetValue("Target")
				target2, _ = row2.Object.DataStore().GetValue("Target")
				target3, _ = row3.Object.DataStore().GetValue("Target")
			)

			t.Logf("Target 1: %+v", target1)
			t.Logf("Target 2: %+v", target2)
			t.Logf("Target 3: %+v", target3)
			t.Logf("________________________________________________________")

			var (
				t1Set = target1.([]queries.Relation)
				t2Set = target2.([]queries.Relation)
				t3Set = target3.([]queries.Relation)
			)

			if len(t1Set) != 3 {
				t.Fatalf("Expected 3 items in target1, got %d", len(t1Set))
			}

			if len(t2Set) != 3 {
				t.Fatalf("Expected 3 items in target2, got %d", len(t2Set))
			}

			if len(t3Set) != 3 {
				t.Fatalf("Expected 3 items in target3, got %d", len(t3Set))
			}
			return 0, 0, 0, 0, 0
		},
	},
	{
		Name: "TestManyToMany_Forward_1",
		Test: func(t *testing.T, profiles []*Profile, users []*User, m2m_sources []*ModelManyToMany, m2m_targets []*ModelManyToMany_Target, m2m_throughs []*ModelManyToMany_Through) (int, int, int, int, int) {
			var rows, err = queries.GetQuerySet(&ModelManyToMany{}).
				Select("*", "Target.*", "Target.TargetReverse.*").
				Filter("ID__in", m2m_sources[0].ID, m2m_sources[1].ID, m2m_sources[2].ID).
				OrderBy("ID", "Target.ID", "Target.TargetReverse.ID").
				All()
			if err != nil {
				t.Fatalf("Failed to get objects: %v", err)
			}

			if len(rows) != 3 {
				t.Fatalf("Expected 3 rows, got %d", len(rows))
			}

			var hasTarget = func(row *queries.Row[*ModelManyToMany]) bool {
				var t, ok = row.Object.DataStore().GetValue("Target")
				return ok && t != nil
			}

			var (
				row1 = rows[0]
				row2 = rows[1]
				row3 = rows[2]
			)

			t.Logf("Row 1: %+v", row1.Object)
			t.Logf("Row 2: %+v", row2.Object)
			t.Logf("Row 3: %+v", row3.Object)
			t.Logf("________________________________________________________")

			if !hasTarget(row1) {
				t.Fatalf("Expected row1 to have a target, got nil")
			}

			if !hasTarget(row2) {
				t.Fatalf("Expected row2 to have a target, got nil")
			}

			if !hasTarget(row3) {
				t.Fatalf("Expected row3 to have a target, got nil")
			}

			var (
				target1 = row1.Object.Target
				target2 = row2.Object.Target
				target3 = row3.Object.Target
			)

			t.Logf("Target 1: %+v", target1)
			t.Logf("Target 2: %+v", target2)
			t.Logf("Target 3: %+v", target3)
			t.Logf("________________________________________________________")

			if target1.Len() != 3 {
				t.Fatalf("Expected 3 items in target1, got %d", target1.Len())
			}

			if target2.Len() != 3 {
				t.Fatalf("Expected 3 items in target2, got %d", target2.Len())
			}

			if target3.Len() != 3 {
				t.Fatalf("Expected 3 items in target3, got %d", target3.Len())
			}

			var checkRow = func(t *testing.T, row *queries.Row[*ModelManyToMany], actual *queries.RelM2M[*ModelManyToMany_Target, *ModelManyToMany_Through], expected []*ModelManyToMany_Target, expectedReverse map[int64][]*ModelManyToMany) {
				var idx = 0
				if actual.Parent == nil || actual.Parent.Object == nil {
					t.Fatalf("Expected actual.Parent.Object to be set: %+v\n\t%s", row.Object.Model, row.QuerySet.LatestQuery().SQL())
				}

				if len(actual.AsList()) != len(expected) {
					t.Fatalf("Expected %d items in actual.AsList(), got %d: %+v\n\t%s", len(expected), len(actual.AsList()), row.Object.Model, row.QuerySet.LatestQuery().SQL())
				}

				for i, item := range actual.AsList() {
					target := item.Model().(*ModelManyToMany_Target)

					if target.ID != expected[idx].ID {
						t.Fatalf("Expected target[%d].ID to be %d, got %d", i, expected[idx].ID, target.ID)
					}

					if target.Name != expected[idx].Name {
						t.Fatalf("Expected target[%d].Name to be %q, got %q", i, expected[idx].Name, target.Name)
					}

					rev, ok := target.DataStore().GetValue("TargetReverse")
					if !ok {
						t.Fatalf("Expected Target.TargetReverse to be set: %+v\n\t%s", target.Model, row.QuerySet.LatestQuery().SQL())
					}

					revList, ok := rev.([]queries.Relation)
					if !ok {
						t.Fatalf("Expected Target.TargetReverse to be a list: %+v\n\t%s", target.Model, row.QuerySet.LatestQuery().SQL())
					}

					expectedRev := expectedReverse[target.ID]
					if len(revList) != len(expectedRev) {
						for j, revTarget := range revList {
							t.Logf("revTarget %d: %+v", j, revTarget)
						}
						t.Errorf(
							"Expected Target.TargetReverse %q to be a list of length %d, got %d: %+v\n\t%s",
							target.Name,
							len(expectedRev), len(revList),
							target.Model, row.QuerySet.LatestQuery().SQL(),
						)
						t.FailNow()
					}

					for j, revItem := range revList {
						revThrough := revItem.Through().(*ModelManyToMany_Through)
						revTarget := revItem.Model().(*ModelManyToMany)

						if revThrough.SourceModel.ID != revTarget.ID {
							t.Fatalf("Expected revThrough.SourceModel.ID to be %d, got %d", row.Object.ID, revThrough.SourceModel.ID)
						}

						if revTarget.ID != expectedRev[j].ID {
							t.Fatalf("Expected revTarget[%d].ID to be %d, got %d", j, expectedRev[j].ID, revTarget.ID)
						}
						if revTarget.Title != expectedRev[j].Title {
							t.Fatalf("Expected revTarget[%d].Title to be %q, got %q", j, expectedRev[j].Title, revTarget.Title)
						}

						t.Run(fmt.Sprintf("TestModelsThroughModelSetOnTarget-%d-%d", revTarget.ID, target.ID), func(t *testing.T) {
							if revTarget.ThroughModel.(*ModelManyToMany_Through).TargetModel.ID != target.ID {
								t.Fatalf("Expected revTarget[%d].ThroughModel.TargetModel.ID to be %d, got %d", j, target.ID, revTarget.ThroughModel.(*ModelManyToMany_Through).TargetModel.ID)
							}
							if revTarget.ThroughModel.(*ModelManyToMany_Through).SourceModel.ID != revTarget.ID {
								t.Fatalf("Expected revTarget[%d].ThroughModel.SourceModel.ID to be %d, got %d", j, row.Object.ID, revTarget.ThroughModel.(*ModelManyToMany_Through).SourceModel.ID)
							}
						})
					}
					idx++
				}
			}

			var (
				targets_check_0 = []*ModelManyToMany_Target{m2m_targets[0], m2m_targets[1], m2m_targets[2]}
				targets_check_1 = []*ModelManyToMany_Target{m2m_targets[1], m2m_targets[2], m2m_targets[3]}
				targets_check_2 = []*ModelManyToMany_Target{m2m_targets[1], m2m_targets[2], m2m_targets[3]}

				expectedReverseMap = map[int64][]*ModelManyToMany{
					m2m_targets[0].ID: {m2m_sources[0]},
					m2m_targets[1].ID: {m2m_sources[0], m2m_sources[1], m2m_sources[2]},
					m2m_targets[2].ID: {m2m_sources[0], m2m_sources[1], m2m_sources[2]},
					m2m_targets[3].ID: {m2m_sources[1], m2m_sources[2]},
				}
			)

			checkRow(t, rows[0], target1, targets_check_0, expectedReverseMap)
			checkRow(t, rows[1], target2, targets_check_1, expectedReverseMap)
			checkRow(t, rows[2], target3, targets_check_2, expectedReverseMap)
			return 0, 0, 0, 0, 0
		},
	},
	{
		Name: "TestManyToMany_RelManyToManyQuerySet",
		Test: func(t *testing.T, profiles []*Profile, users []*User, m2m_sources []*ModelManyToMany, m2m_targets []*ModelManyToMany_Target, m2m_throughs []*ModelManyToMany_Through) (int, int, int, int, int) {
			var row, err = queries.GetQuerySet(&ModelManyToMany{}).
				Select("*", "Target.Name", "Target.Age", "Target.TargetReverse.Title").
				Filter("ID", m2m_sources[0].ID).
				OrderBy("ID", "Target.ID", "Target.TargetReverse.ID").
				First()
			if err != nil {
				t.Fatalf("Failed to get row: %v", err)
			}

			created, err := row.Object.Target.Objects().AddTarget(m2m_targets[3])
			if err != nil {
				t.Fatalf("Failed to add Target object: %v", err)
			}

			if created {
				t.Fatalf("Expected Target object to already exist, but it was created")
			}

			created, err = row.Object.Target.Objects().AddTarget(&ModelManyToMany_Target{
				Name: "TestManyToMany_Target__AddTarget",
				Age:  35,
			})
			if err != nil {
				t.Fatalf("Failed to add Target object: %v", err)
			}

			if !created {
				t.Fatalf("Expected Target object to be created, but it already exists")
			}

			a, err := row.Object.Target.Objects().All()
			if err != nil {
				t.Fatalf("Failed to get Target objects for row 0: %v", err)
			}

			if len(a) != 5 {
				t.Fatalf("Expected 5 Target objects, got %d", len(a))
			}

			var targetObjects = make([]*ModelManyToMany_Target, 0, len(a))
			for _, item := range a {
				var through = item.Through.(*ModelManyToMany_Through)
				t.Logf("___________________________________________")
				t.Logf("Row [1] Target item from queryset: %+v", item.Object)
				t.Logf("Row [2] Target item from queryset: %+v", through.SourceModel)
				t.Logf("Row [3] Target item from queryset: %+v", through.TargetModel)

				targetObjects = append(targetObjects, item.Object)
			}

			removed, err := row.Object.Target.Objects().Filter("ID__in", targetObjects[0].ID, targetObjects[1].ID).ClearTargets()
			// removed, err := row.Object.Target.Relations.RemoveTargets(targetObjects)
			if err != nil {
				t.Fatalf("Failed to remove targets: %v", err)
			}

			if int(removed) != 2 {
				t.Fatalf("Expected to remove %d targets, got %d", 2, removed)
			}

			a, err = row.Object.Target.Objects().All()
			if err != nil {
				t.Fatalf("Failed to get Target objects for row 0: %v", err)
			}

			t.Logf("QuerySet Arguments: %+v", row.QuerySet.LatestQuery().Args())

			for _, item := range a {
				var through = item.Through.(*ModelManyToMany_Through)
				t.Logf("___________________________________________")
				t.Logf("Row [1] Target item from queryset: %+v", item.Object)
				t.Logf("Row [2] Target item from queryset: %+v", through.SourceModel)
				t.Logf("Row [3] Target item from queryset: %+v", through.TargetModel)
			}
			return 0, 0, 0, 2, 0
		},
	},
	{
		Name: "TestManyToMany_RelManyToManyQuerySet_AddTarget",
		Test: func(t *testing.T, profiles []*Profile, users []*User, m2m_sources []*ModelManyToMany, m2m_targets []*ModelManyToMany_Target, m2m_throughs []*ModelManyToMany_Through) (int, int, int, int, int) {
			var obj = &ModelManyToMany{
				Title: "TestManyToMany_AddTarget",
				User:  &User{ID: int(users[0].ID)},
			}
			if err := queries.CreateObject(obj); err != nil {
				t.Fatalf("Failed to create object: %v", err)
			}

			t.Logf("Created new ModelManyToMany object: %+v", obj)
			t.Logf("Adding targets to new ModelManyToMany object %v", obj.Target)

			var target = &ModelManyToMany_Target{
				Name: "TestManyToMany_Target_AddTarget_1",
				Age:  40,
			}
			var created, err = obj.Target.Objects().AddTarget(target)
			if err != nil {
				t.Fatalf("Failed to add Target object: %v", err)
			}
			if !created {
				t.Fatalf("Expected Target object to be created, but it already exists")
			}

			if len(obj.Target.AsList()) != 1 {
				t.Fatalf("Expected 1 Target object, got %d", len(obj.Target.AsList()))
			}

			a, err := obj.Target.Objects().All()
			if err != nil {
				t.Fatalf("Failed to get Target objects for row: %v", err)
			}

			if len(a) != 1 {
				t.Fatalf("Expected 1 Target object, got %d", len(a))
			}

			if len(obj.Target.AsList()) != 1 {
				t.Fatalf("Expected 1 Target object in ModelManyToMany.Target, got %d", len(obj.Target.AsList()))
			}

			for _, item := range a {
				var through = item.Through.(*ModelManyToMany_Through)
				t.Logf("___________________________________________")
				t.Logf("Row [1] Target item from queryset: %+v", item.Object)
				t.Logf("Row [2] Target item from queryset: %+v", through.SourceModel)
				t.Logf("Row [3] Target item from queryset: %+v", through.TargetModel)
			}

			if _, err := queries.GetQuerySet(&ModelManyToMany{}).Filter("ID", obj.ID).Delete(); err != nil {
				t.Fatalf("Failed to delete ModelManyToMany object: %v", err)
			}

			if _, err := queries.GetQuerySet(&ModelManyToMany_Target{}).Filter("ID", target.ID).Delete(); err != nil {
				t.Fatalf("Failed to delete ModelManyToMany_Target object: %v", err)
			}

			if _, err := queries.GetQuerySet(&ModelManyToMany_Through{}).Filter("SourceModel", obj.ID).Filter("TargetModel", target.ID).Delete(); err != nil {
				t.Fatalf("Failed to delete ModelManyToMany_Through object: %v", err)
			}

			return 0, 0, 0, 0, 0
		},
	},
	{
		Name: "TestManyToMany_RelOneToManyQuerySet",
		Test: func(t *testing.T, profiles []*Profile, users []*User, m2m_sources []*ModelManyToMany, m2m_targets []*ModelManyToMany_Target, m2m_throughs []*ModelManyToMany_Through) (int, int, int, int, int) {
			var row, err = queries.Objects(&User{}).
				Select("ID", "Name", "ModelManyToManySet.Title").
				Filter("ID__in", users[0].ID).
				OrderBy("ID", "ModelManyToManySet.ID").
				Get()
			if err != nil {
				t.Fatalf("Failed to get objects: %v", err)
			}

			var user = row.Object
			t.Logf("User 1: %+v", user)
			t.Logf("User 1 ManyToManySet: %+v %+v", user.ModelManyToManySet.Parent, user.ModelManyToManySet.AsList())

			for _, obj := range user.ModelManyToManySet.AsList() {
				t.Logf("obj: %+v", obj)
			}

			if len(user.ModelManyToManySet.AsList()) != 2 {
				t.Fatalf("Expected 2 items in ManyToManySet, got %d", len(user.ModelManyToManySet.AsList()))
			}

			chk, err := queries.GetQuerySet(&ModelManyToMany{}).
				Select("*").Filter("User", user.ID).
				All()
			if err != nil {
				t.Fatalf("Failed to get ManyToManySet relations: %v", err)
			}

			if len(chk) != 2 {
				t.Errorf("Expected 2 items in set, got %d", len(chk))
				for _, obj := range chk {
					t.Logf("obj: %+v", obj)
				}
				t.FailNow()
			}

			var objs = user.ModelManyToManySet.AsList()
			if len(chk) != len(objs) {
				t.Errorf("Expected %d items in set, got %d", len(chk), len(objs))
				for _, obj := range objs {
					t.Logf("obj: %+v", obj)
				}
				t.FailNow()
			}

			a, err := user.ModelManyToManySet.Objects().OrderBy("ID").All()
			if err != nil {
				t.Fatalf("Failed to get ManyToManySet relations: %v", err)
			}

			if len(chk) != len(a) {
				t.Errorf("Expected %d items in set, got %d", len(chk), len(a))
				for _, obj := range a {
					t.Logf("obj: %+v", obj)
				}
				t.FailNow()
			}

			for _, item := range a {
				t.Logf("___________________________________________")
				t.Logf("Row [1] Target item from queryset: %+v", item.Object)

				if item.Object.(*ModelManyToMany).User.ID != user.ID {
					t.Fatalf("Expected ManyToManySet item User.Name to be %v, got %v", user.Name, item.Object.(*ModelManyToMany).User.Name)
				}
			}
			return 0, 0, 0, 0, 0
		},
	},
}

func TestManyToMany(t *testing.T) {

	// FORWARD
	//	TestManyToMany1 -> [TestManyToMany_Target1, TestManyToMany_Target2, TestManyToMany_Target3]
	//	TestManyToMany2 -> [TestManyToMany_Target2, TestManyToMany_Target3, TestManyToMany_Target4]
	//	TestManyToMany3 -> [TestManyToMany_Target2, TestManyToMany_Target3, TestManyToMany_Target4]

	// REVERSE
	//	TestManyToMany_Target1 -> [TestManyToMany1]
	//	TestManyToMany_Target2 -> [TestManyToMany1, TestManyToMany2, TestManyToMany3]
	//	TestManyToMany_Target3 -> [TestManyToMany1, TestManyToMany2, TestManyToMany3]
	//	TestManyToMany_Target4 -> [TestManyToMany2, TestManyToMany3]

	// LIST FORWARD
	//	m2m_sources[0] -> [m2m_targets[0], m2m_targets[1], m2m_targets[2]]
	//	m2m_sources[1] -> [m2m_targets[1], m2m_targets[2], m2m_targets[3]]
	//	m2m_sources[2] -> [m2m_targets[1], m2m_targets[2], m2m_targets[3]]

	// LIST REVERSE
	// 	m2m_targets[0] -> [m2m_sources[0]]
	// 	m2m_targets[1] -> [m2m_sources[1], m2m_sources[2], m2m_sources[3]]
	// 	m2m_targets[2] -> [m2m_sources[1], m2m_sources[2], m2m_sources[3]]
	// 	m2m_targets[3] -> [m2m_sources[2], m2m_sources[3]]

	// var deletions = make([]func() error, 0, len(manyToManyTests)*5)

	for _, test := range manyToManyTests {
		t.Run(test.Name, func(t *testing.T) {
			var profiles, profile_delete = quest.CreateObjects[*Profile](t,
				&Profile{
					Name: "TestManyToManyProfile1",
				},
				&Profile{
					Name: "TestManyToManyProfile2",
				},
			)

			var users, user_delete = quest.CreateObjects[*User](t,
				&User{
					Name:    "TestManyToManyUser1",
					Profile: profiles[0],
				},
				&User{
					Name:    "TestManyToManyUser2",
					Profile: profiles[1],
				},
			)

			var m2m_sources, m2m_source_delete = quest.CreateObjects[*ModelManyToMany](t,
				&ModelManyToMany{
					Title: "TestManyToMany1",
					User:  &User{ID: int(users[0].ID)},
				},
				&ModelManyToMany{
					Title: "TestManyToMany2",
					User:  &User{ID: int(users[0].ID)},
				},
				&ModelManyToMany{
					Title: "TestManyToMany3",
					User:  &User{ID: int(users[1].ID)},
				},
			)

			var m2m_targets, m2m_target_delete = quest.CreateObjects[*ModelManyToMany_Target](t,
				&ModelManyToMany_Target{
					Name: "TestManyToMany_Target1",
					Age:  25,
				},
				&ModelManyToMany_Target{
					Name: "TestManyToMany_Target2",
					Age:  25,
				},
				&ModelManyToMany_Target{
					Name: "TestManyToMany_Target3",
					Age:  25,
				},
				&ModelManyToMany_Target{
					Name: "TestManyToMany_Target4",
					Age:  30,
				},
			)

			var m2m_throughs, m2m_through_delete = quest.CreateObjects[*ModelManyToMany_Through](t,
				&ModelManyToMany_Through{
					SourceModel: &ModelManyToMany{
						ID: m2m_sources[0].ID,
					},
					TargetModel: &ModelManyToMany_Target{
						ID: m2m_targets[0].ID,
					},
				},
				&ModelManyToMany_Through{
					SourceModel: &ModelManyToMany{
						ID: m2m_sources[0].ID,
					},
					TargetModel: &ModelManyToMany_Target{
						ID: m2m_targets[1].ID,
					},
				},
				&ModelManyToMany_Through{
					SourceModel: &ModelManyToMany{
						ID: m2m_sources[0].ID,
					},
					TargetModel: &ModelManyToMany_Target{
						ID: m2m_targets[2].ID,
					},
				},
				&ModelManyToMany_Through{
					SourceModel: &ModelManyToMany{
						ID: m2m_sources[1].ID,
					},
					TargetModel: &ModelManyToMany_Target{
						ID: m2m_targets[1].ID,
					},
				},
				&ModelManyToMany_Through{
					SourceModel: &ModelManyToMany{
						ID: m2m_sources[1].ID,
					},
					TargetModel: &ModelManyToMany_Target{
						ID: m2m_targets[2].ID,
					},
				},
				&ModelManyToMany_Through{
					SourceModel: &ModelManyToMany{
						ID: m2m_sources[1].ID,
					},
					TargetModel: &ModelManyToMany_Target{
						ID: m2m_targets[3].ID,
					},
				},
				&ModelManyToMany_Through{
					SourceModel: &ModelManyToMany{
						ID: m2m_sources[2].ID,
					},
					TargetModel: &ModelManyToMany_Target{
						ID: m2m_targets[1].ID,
					},
				},
				&ModelManyToMany_Through{
					SourceModel: &ModelManyToMany{
						ID: m2m_sources[2].ID,
					},
					TargetModel: &ModelManyToMany_Target{
						ID: m2m_targets[2].ID,
					},
				},
				&ModelManyToMany_Through{
					SourceModel: &ModelManyToMany{
						ID: m2m_sources[2].ID,
					},
					TargetModel: &ModelManyToMany_Target{
						ID: m2m_targets[3].ID,
					},
				})

			var p, u, msrc, mthru, mtgt = test.Test(t, profiles, users, m2m_sources, m2m_targets, m2m_throughs)

			t.Log("_________________________________________________________")
			t.Logf("Test %s completed successfully", test.Name)

			m2m_source_delete(msrc)   // model_manytomany
			m2m_target_delete(mtgt)   // model_manytomany_target
			m2m_through_delete(mthru) // model_manytomany_through
			user_delete(u)            // users
			profile_delete(p)         // profiles
		})
	}
}

func TestPluckRows(t *testing.T) {

	var profiles, profiles_delete = quest.CreateObjects[*Profile](t,
		&Profile{
			Name: "TestPluckRowsProfile1",
		},
		&Profile{
			Name: "TestPluckRowsProfile2",
		},
	)

	var users, users_delete = quest.CreateObjects[*User](t,
		&User{
			Name:    "TestPluckRowsUser1",
			Profile: &Profile{ID: int(profiles[0].ID)},
		},
		&User{
			Name:    "TestPluckRowsUser2",
			Profile: &Profile{ID: int(profiles[1].ID)},
		},
	)

	var m2m_sources, m2m_source_delete = quest.CreateObjects[*ModelManyToMany](t,
		&ModelManyToMany{
			Title: "TestPluckRows1",
			User:  &User{ID: int(users[0].ID)},
		},
		&ModelManyToMany{
			Title: "TestPluckRows2",
			User:  &User{ID: int(users[1].ID)},
		},
	)

	var m2m_targets, m2m_target_delete = quest.CreateObjects[*ModelManyToMany_Target](t,
		&ModelManyToMany_Target{
			Name: "TestPluckRows_Target1",
			Age:  25,
		},
		&ModelManyToMany_Target{
			Name: "TestPluckRows_Target2",
			Age:  30,
		},
	)

	var _, m2m_through_delete = quest.CreateObjects[*ModelManyToMany_Through](t,
		&ModelManyToMany_Through{
			SourceModel: &ModelManyToMany{
				ID: m2m_sources[0].ID,
			},
			TargetModel: &ModelManyToMany_Target{
				ID: m2m_targets[0].ID,
			},
		},
		&ModelManyToMany_Through{
			SourceModel: &ModelManyToMany{
				ID: m2m_sources[0].ID,
			},
			TargetModel: &ModelManyToMany_Target{
				ID: m2m_targets[1].ID,
			},
		},
		&ModelManyToMany_Through{
			SourceModel: &ModelManyToMany{
				ID: m2m_sources[1].ID,
			},
			TargetModel: &ModelManyToMany_Target{
				ID: m2m_targets[0].ID,
			},
		},
		&ModelManyToMany_Through{
			SourceModel: &ModelManyToMany{
				ID: m2m_sources[1].ID,
			},
			TargetModel: &ModelManyToMany_Target{
				ID: m2m_targets[1].ID,
			},
		},
	)

	defer func() {
		m2m_source_delete(0)  // model_manytomany
		m2m_target_delete(0)  // model_manytomany_target
		m2m_through_delete(0) // model_manytomany_through
		users_delete(0)       // users
		profiles_delete(0)    // profiles
	}()

	var rows, err = queries.GetQuerySet(&ModelManyToMany{}).
		Select("*", "User.*", "User.Profile.*").
		OrderBy("User.Profile.ID").
		All()

	if err != nil {
		t.Fatalf("Failed to get rows: %v", err)
	}

	if len(rows) == 0 {
		t.Fatalf("Expected at least 1 row, got 0")
	}

	var values = make([]int, 0, len(rows))
	for idx, value := range queries.PluckRowValues[int](rows, "User.Profile.ID") {
		t.Logf("Row %d: %v\n", idx, value)
		values = append(values, value)
	}

	for idx, row := range rows {
		var profileID = row.Object.User.Profile.ID
		if profileID != values[idx] {
			t.Errorf("Expected Profile ID %d, got %d", profileID, values[idx])
		}
	}
}

func TestPluckManyToManyRows(t *testing.T) {
	var profiles, profiles_delete = quest.CreateObjects[*Profile](t,
		&Profile{
			Name: "TestPluckRowsProfile1",
		},
		&Profile{
			Name: "TestPluckRowsProfile2",
		},
	)

	var users, users_delete = quest.CreateObjects[*User](t,
		&User{
			Name:    "TestPluckRowsUser1",
			Profile: &Profile{ID: int(profiles[0].ID)},
		},
		&User{
			Name:    "TestPluckRowsUser2",
			Profile: &Profile{ID: int(profiles[1].ID)},
		},
	)

	var m2m_sources, m2m_source_delete = quest.CreateObjects[*ModelManyToMany](t,
		&ModelManyToMany{
			Title: "TestPluckRows1",
			User:  &User{ID: int(users[0].ID)},
		},
		&ModelManyToMany{
			Title: "TestPluckRows2",
			User:  &User{ID: int(users[1].ID)},
		},
	)

	var m2m_targets, m2m_target_delete = quest.CreateObjects[*ModelManyToMany_Target](t,
		&ModelManyToMany_Target{
			Name: "TestPluckRows_Target1",
			Age:  25,
		},
		&ModelManyToMany_Target{
			Name: "TestPluckRows_Target2",
			Age:  30,
		},
	)

	var _, m2m_through_delete = quest.CreateObjects[*ModelManyToMany_Through](t,
		&ModelManyToMany_Through{
			SourceModel: &ModelManyToMany{
				ID: m2m_sources[0].ID,
			},
			TargetModel: &ModelManyToMany_Target{
				ID: m2m_targets[0].ID,
			},
		},
		&ModelManyToMany_Through{
			SourceModel: &ModelManyToMany{
				ID: m2m_sources[0].ID,
			},
			TargetModel: &ModelManyToMany_Target{
				ID: m2m_targets[1].ID,
			},
		},
		&ModelManyToMany_Through{
			SourceModel: &ModelManyToMany{
				ID: m2m_sources[1].ID,
			},
			TargetModel: &ModelManyToMany_Target{
				ID: m2m_targets[0].ID,
			},
		},
		&ModelManyToMany_Through{
			SourceModel: &ModelManyToMany{
				ID: m2m_sources[1].ID,
			},
			TargetModel: &ModelManyToMany_Target{
				ID: m2m_targets[1].ID,
			},
		},
	)

	defer func() {
		m2m_source_delete(0)  // model_manytomany
		m2m_target_delete(0)  // model_manytomany_target
		m2m_through_delete(0) // model_manytomany_through
		users_delete(0)       // users
		profiles_delete(0)    // profiles
	}()

	var rows, err = queries.GetQuerySet(&ModelManyToMany{}).
		Select("*", "Target.*", "Target.TargetReverse.*").
		OrderBy("ID", "Target.ID", "Target.TargetReverse.ID").
		All()

	if err != nil {
		t.Fatalf("Failed to get rows: %v", err)
	}

	if len(rows) == 0 {
		t.Fatalf("Expected at least 1 row, got 0")
	}

	var values = make([]int64, 0, len(rows))
	for idx, value := range queries.PluckRowValues[int64](rows, "Target.TargetReverse.ID") {
		t.Logf("Row %d: %v\n", idx, value)
		values = append(values, value)
	}

	t.Logf("Values: %v", values)

	var idx = 0
	for _, row := range rows {
		for _, target := range row.Object.Target.AsList() {
			var rev, ok = target.Object.DataStore().GetValue("TargetReverse")
			if !ok {
				t.Fatalf("Expected Target.TargetReverse to be set: %+v\n\t%s", target.Object.Model, row.QuerySet.LatestQuery().SQL())
			}

			revList, ok := rev.(*queries.RelM2M[attrs.Definer, attrs.Definer])
			if !ok {
				t.Fatalf("Expected Target.TargetReverse to be a list: %+v %T\n\t%s", target.Object.Model, rev, row.QuerySet.LatestQuery().SQL())
			}

			for _, revItem := range revList.AsList() {

				revTarget := revItem.Model().(*ModelManyToMany)
				if revTarget.ID != values[idx] {
					t.Fatalf("Expected Target.TargetReverse[%d].ID to be %d, got %d", idx, values[idx], revTarget.ID)
				}

				idx++
			}
		}
	}
}
