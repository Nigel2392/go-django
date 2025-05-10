package attrs_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/forms/widgets"
)

func init() {
	attrs.RegisterModel(&Image{})
	attrs.RegisterModel(&Profile{})
	attrs.RegisterModel(&User{})
	attrs.RegisterModel(&Todo{})
	attrs.RegisterModel(&ObjectWithMultipleRelations{})
	attrs.RegisterModel(&Category{})
	attrs.RegisterModel(&OneToOneWithThrough{})
	attrs.RegisterModel(&OneToOneWithThrough_Through{})
	attrs.RegisterModel(&OneToOneWithThrough_Target{})
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
	).WithTableName("images")
}

type Profile struct {
	ID    int
	Name  string
	Email string
	Image *Image
}

func (m *Profile) FieldDefs() attrs.Definitions {
	return attrs.Define(m,
		attrs.NewField(m, "ID", &attrs.FieldConfig{
			Primary:  true,
			ReadOnly: true,
		}),
		attrs.NewField(m, "Name", &attrs.FieldConfig{}),
		attrs.NewField(m, "Email", &attrs.FieldConfig{}),
		attrs.NewField(m, "Image", &attrs.FieldConfig{
			RelForeignKey: attrs.Relate(&Image{}, "", nil),
			Column:        "image_id",
		}),
	).WithTableName("profiles")
}

type User struct {
	ID      int
	Name    string
	Profile *Profile
}

func (m *User) FieldDefs() attrs.Definitions {
	return attrs.Define(m,
		attrs.NewField(m, "ID", &attrs.FieldConfig{
			Primary:  true,
			ReadOnly: true,
		}),
		attrs.NewField(m, "Name", &attrs.FieldConfig{}),
		attrs.NewField(m, "Profile", &attrs.FieldConfig{
			RelForeignKey: attrs.Relate(&Profile{}, "", nil),
			Column:        "profile_id",
		}),
	).WithTableName("users")
}

type Todo struct {
	ID          int
	Title       string
	Description string
	Done        bool
	User        *User
}

func (m *Todo) FieldDefs() attrs.Definitions {
	return attrs.Define(m,
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
			RelForeignKey: attrs.Relate(&User{}, "", nil),
			Column:        "obj1_id",
			Attributes: map[string]any{
				attrs.AttrReverseAliasKey: "MultiRelationObj1",
			},
		}),
		attrs.NewField(m, "Obj2", &attrs.FieldConfig{
			RelForeignKey: attrs.Relate(&User{}, "", nil),
			Column:        "obj2_id",
			Attributes: map[string]any{
				attrs.AttrReverseAliasKey: "MultiRelationObj2",
			},
		}),
	).WithTableName("object_with_multiple_relations")
}

type Category struct {
	ID     int
	Name   string
	Parent *Category
}

func (m *Category) FieldDefs() attrs.Definitions {
	return attrs.Define(m,
		attrs.NewField(m, "ID", &attrs.FieldConfig{
			Primary:  true,
			ReadOnly: true,
		}),
		attrs.NewField(m, "Name", &attrs.FieldConfig{}),
		attrs.NewField(m, "Parent", &attrs.FieldConfig{
			Column:        "parent_id",
			RelForeignKey: attrs.Relate(&Category{}, "", nil),
		}),
	).WithTableName("categories")
}

type OneToOneWithThrough struct {
	ID      int64
	Title   string
	Through *OneToOneWithThrough_Target
	User    *User
}

func (t *OneToOneWithThrough) FieldDefs() attrs.Definitions {
	return attrs.Define(t,
		attrs.NewField(t, "ID", &attrs.FieldConfig{
			Column:  "id",
			Primary: true,
		}),
		attrs.NewField(t, "Title", &attrs.FieldConfig{
			Column: "title",
		}),
		attrs.NewField(t, "Through", &attrs.FieldConfig{
			NameOverride: "Target",
			Attributes: map[string]interface{}{
				attrs.AttrReverseAliasKey: "TargetReverse",
			},
			RelOneToOne: attrs.Relate(
				&OneToOneWithThrough_Target{},
				"", &attrs.ThroughModel{
					This:   &OneToOneWithThrough_Through{},
					Source: "SourceModel",
					Target: "TargetModel",
				},
			),
		}),
		attrs.NewField(t, "User", &attrs.FieldConfig{
			Column:        "user_id",
			RelForeignKey: attrs.Relate(&User{}, "", nil),
		}),
	).WithTableName("onetoonewiththrough")
}

type OneToOneWithThrough_Through struct {
	ID          int64
	SourceModel *OneToOneWithThrough
	TargetModel *OneToOneWithThrough_Target
}

func (t *OneToOneWithThrough_Through) FieldDefs() attrs.Definitions {
	return attrs.Define(t,
		attrs.NewField(t, "ID", &attrs.FieldConfig{
			Column:  "id",
			Primary: true,
		}),
		attrs.NewField(t, "SourceModel", &attrs.FieldConfig{
			Column: "source_id",
			Null:   false,
		}),
		attrs.NewField(t, "TargetModel", &attrs.FieldConfig{
			Column: "target_id",
			Null:   false,
		}),
	).WithTableName("onetoonewiththrough_through")
}

type OneToOneWithThrough_Target struct {
	ID   int64
	Name string
	Age  int
}

func (t *OneToOneWithThrough_Target) FieldDefs() attrs.Definitions {
	return attrs.Define(t,
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
	).WithTableName("onetoonewiththrough_target")
}

func getField(m attrs.Definer, field string) attrs.Field {
	defs := m.FieldDefs()
	f, _ := defs.Field(field)
	return f
}

type walkFieldPathsExpected struct {
	definer       attrs.Definer
	parent        attrs.Definer
	field         attrs.Field
	chain         []string
	relationchain [][]string
	relationType  attrs.RelationType
}

type walkFieldPathsTest struct {
	name     string
	model    attrs.Definer
	column   string
	expected walkFieldPathsExpected
}

var walkFieldsTests2 = []walkFieldPathsTest{
	{
		name:   "TestTodoID",
		model:  &Todo{},
		column: "ID",
		expected: walkFieldPathsExpected{
			definer: &Todo{},
			field:   getField(&Todo{}, "ID"),
			chain:   []string{"ID"},
		},
	},
	{
		name:   "TestTodoUser",
		model:  &Todo{},
		column: "User",
		expected: walkFieldPathsExpected{
			definer:       &Todo{},
			field:         getField(&Todo{}, "User"),
			chain:         []string{"User"},
			relationchain: [][]string{{"User.Todo", "ID.User"}},
			relationType:  attrs.RelOneToOne,
		},
	},
	{
		name:   "TestTodoUserWithID",
		model:  &Todo{},
		column: "User.Name",
		expected: walkFieldPathsExpected{
			definer:       &User{},
			parent:        &Todo{},
			field:         getField(&User{}, "Name"),
			chain:         []string{"User", "Name"},
			relationchain: [][]string{{"User.Todo", "ID.User"}},
			relationType:  attrs.RelOneToOne,
		},
	},
	{
		name:   "TestUserWithTodoID",
		model:  &User{},
		column: "Todo.Title",
		expected: walkFieldPathsExpected{
			definer:       &Todo{},
			parent:        &User{},
			field:         getField(&Todo{}, "Title"),
			chain:         []string{"Todo", "Title"},
			relationchain: [][]string{{"ID.User", "User.Todo"}},
			relationType:  attrs.RelOneToOne,
		},
	},
	{
		name:   "TestObjectWithMultipleRelationsID1",
		model:  &ObjectWithMultipleRelations{},
		column: "Obj1.Name",
		expected: walkFieldPathsExpected{
			definer:       &User{},
			parent:        &ObjectWithMultipleRelations{},
			field:         getField(&User{}, "Name"),
			chain:         []string{"Obj1", "Name"},
			relationchain: [][]string{{"Obj1.ObjectWithMultipleRelations", "ID.User"}},
			relationType:  attrs.RelManyToOne,
		},
	},
	{
		name:   "TestObjectWithMultipleRelationsID2",
		model:  &ObjectWithMultipleRelations{},
		column: "Obj2.Name",
		expected: walkFieldPathsExpected{
			definer:       &User{},
			parent:        &ObjectWithMultipleRelations{},
			field:         getField(&User{}, "Name"),
			chain:         []string{"Obj2", "Name"},
			relationchain: [][]string{{"Obj2.ObjectWithMultipleRelations", "ID.User"}},
			relationType:  attrs.RelManyToOne,
		},
	},
	{
		name:   "TestNestedCategoriesParent",
		model:  &Category{},
		column: "Parent.Parent",
		expected: walkFieldPathsExpected{
			definer:       &Category{},
			parent:        &Category{},
			field:         getField(&Category{}, "Parent"),
			chain:         []string{"Parent", "Parent"},
			relationchain: [][]string{{"Parent.Category", "ID.Category"}, {"Parent.Category", "ID.Category"}},
			relationType:  attrs.RelManyToOne,
		},
	},
	{
		name:   "TestNestedCategoriesName",
		model:  &Category{},
		column: "Parent.Parent.Name",
		expected: walkFieldPathsExpected{
			definer:       &Category{},
			parent:        &Category{},
			field:         getField(&Category{}, "Name"),
			chain:         []string{"Parent", "Parent", "Name"},
			relationchain: [][]string{{"Parent.Category", "ID.Category"}, {"Parent.Category", "ID.Category"}},
			relationType:  attrs.RelManyToOne,
		},
	},
	{
		name:   "TestOneToOneWithThrough",
		model:  &OneToOneWithThrough{},
		column: "Target.Name",
		expected: walkFieldPathsExpected{
			definer:       &OneToOneWithThrough_Target{},
			parent:        &OneToOneWithThrough{},
			field:         getField(&OneToOneWithThrough_Target{}, "Name"),
			chain:         []string{"Target", "Name"},
			relationchain: [][]string{{"Target.OneToOneWithThrough", "SourceModel.OneToOneWithThrough_Through.TargetModel", "ID.OneToOneWithThrough_Target"}},
			relationType:  attrs.RelOneToOne,
		},
	},
	{
		name:   "TestOneToOneWithThroughTarget",
		model:  &OneToOneWithThrough_Target{},
		column: "TargetReverse.Title",
		expected: walkFieldPathsExpected{
			definer:       &OneToOneWithThrough{},
			parent:        &OneToOneWithThrough_Target{},
			field:         getField(&OneToOneWithThrough{}, "Title"),
			chain:         []string{"TargetReverse", "Title"},
			relationchain: [][]string{{"TargetReverse.OneToOneWithThrough_Target", "TargetModel.OneToOneWithThrough_Through.SourceModel", "ID.OneToOneWithThrough"}},
			relationType:  attrs.RelOneToOne,
		},
	},
	{
		name:   "TestOneToOneWithThroughNested",
		model:  &OneToOneWithThrough{},
		column: "Target.TargetReverse.Target.Name",
		expected: walkFieldPathsExpected{
			definer:      &OneToOneWithThrough_Target{},
			parent:       &OneToOneWithThrough{},
			field:        getField(&OneToOneWithThrough_Target{}, "Name"),
			chain:        []string{"Target", "TargetReverse", "Target", "Name"},
			relationType: attrs.RelOneToOne,
			relationchain: [][]string{
				{"Target.OneToOneWithThrough", "SourceModel.OneToOneWithThrough_Through.TargetModel", "ID.OneToOneWithThrough_Target"},
				{"TargetReverse.OneToOneWithThrough_Target", "TargetModel.OneToOneWithThrough_Through.SourceModel", "ID.OneToOneWithThrough"},
				{"Target.OneToOneWithThrough", "SourceModel.OneToOneWithThrough_Through.TargetModel", "ID.OneToOneWithThrough_Target"},
			},
		},
	},
	{
		name:   "TestOneToOneWithThroughTargetNested",
		model:  &OneToOneWithThrough_Target{},
		column: "TargetReverse.Target.TargetReverse.Title",
		expected: walkFieldPathsExpected{
			definer:      &OneToOneWithThrough{},
			parent:       &OneToOneWithThrough_Target{},
			field:        getField(&OneToOneWithThrough{}, "Title"),
			chain:        []string{"TargetReverse", "Target", "TargetReverse", "Title"},
			relationType: attrs.RelOneToOne,
			relationchain: [][]string{
				{"TargetReverse.OneToOneWithThrough_Target", "TargetModel.OneToOneWithThrough_Through.SourceModel", "ID.OneToOneWithThrough"},
				{"Target.OneToOneWithThrough", "SourceModel.OneToOneWithThrough_Through.TargetModel", "ID.OneToOneWithThrough_Target"},
				{"TargetReverse.OneToOneWithThrough_Target", "TargetModel.OneToOneWithThrough_Through.SourceModel", "ID.OneToOneWithThrough"},
			},
		},
	},
}

func TestWalkFieldPaths(t *testing.T) {
	for _, test := range walkFieldsTests2 {

		attrs.RegisterModel(test.model)

		t.Run(test.name, func(t *testing.T) {

			//defer func() {
			//	if r := recover(); r != nil {
			//		t.Fatalf("expected no panic, got %v", r)
			//	}
			//}()

			var modelMeta = attrs.GetModelMeta(test.model)
			if modelMeta == nil {
				t.Errorf("expected modelMeta not nil, got nil")
				return
			}

			var relationsFwd = modelMeta.ForwardMap()
			var relationsRev = modelMeta.ReverseMap()

			for head := relationsFwd.Front(); head != nil; head = head.Next() {
				var key = head.Key
				var value = head.Value
				t.Logf("forward relation %s -> %T.%s", key, value.Model(), value.Field().Name())
			}

			for head := relationsRev.Front(); head != nil; head = head.Next() {
				var key = head.Key
				var value = head.Value
				t.Logf("reverse relation %s -> %T.%s", key, value.Model(), value.Field().Name())
			}

			for _, field := range modelMeta.Definitions().Fields() {
				t.Logf("field %s -> %T.%s", field.Name(), field.Instance(), field.Name())
			}

			var meta, err = attrs.WalkMetaFields(test.model, test.column)
			if err != nil {
				t.Errorf("expected no error, got %v %v", err, attrs.FieldNames(test.model, nil))
				return
			}

			if meta == nil {
				t.Errorf("expected meta not nil, got nil")
				return
			}

			if meta.Last() == nil || meta.Last().Object == nil {
				t.Errorf("expected meta.Last not nil, got nil")
				return
			}

			if reflect.TypeOf(meta.Last().Object) != reflect.TypeOf(test.expected.definer) {
				t.Errorf("expected meta.Last.Object %T, got %T (%T)", test.expected.definer, meta.Last().Object, meta.First().Object)
			}

			if meta.Last().Parent() != nil {
				if reflect.TypeOf(meta.Last().Parent().Object) != reflect.TypeOf(test.expected.parent) {
					t.Errorf("expected meta.Last.Parent.Object %T, got %T", test.expected.parent, meta.Last().Parent().Object)
				}
			}

			if test.expected.parent == nil && meta.Last().Parent() != nil {
				t.Errorf("expected meta.Last.Parent nil, got %T", meta.Last().Parent().Object)
			}

			if !fieldEquals[attrs.FieldDefinition](meta.Last().Field, test.expected.field) {
				t.Errorf("expected meta.Last.Field %T.%s, got %T.%s", test.expected.field.Instance(), test.expected.field.Name(), meta.Last().Field.Instance(), meta.Last().Field.Name())
			}

			if meta.Last().String() != strings.Join(test.expected.chain, ".") {
				t.Errorf("expected meta.Last.String() %s, got %s", strings.Join(test.expected.chain, "."), meta.Last().String())
			}

		metaLoop:
			for i := range meta {
				var current = meta[i]

				if test.expected.relationchain == nil {
					if current.Relation != nil {
						t.Errorf("expected meta[%d].Relation nil, got %T", i, current.Relation)
					}
					continue metaLoop
				}

				var rel = current.Relation
				switch {
				case i >= len(test.expected.relationchain) && rel != nil:
					t.Errorf("expected meta[%d].Relation nil, got %T", i, rel)
					continue metaLoop
				case i >= len(test.expected.relationchain) && rel == nil:
					continue metaLoop
				}

				if len(test.expected.relationchain[i]) == 0 {
					continue metaLoop
				}

				var (
					from        = rel.From()
					targetModel = rel.Model()
					targetField = rel.Field()
				)

				if from == nil {
					t.Errorf("expected meta[%d].Relation.From() not nil, got nil, %T -> %T.%s", i, from, targetModel, targetField.Name())
					continue metaLoop
				}

				var (
					fromModel = from.Model()
					fromField = from.Field()
					through   = rel.Through()
				)

				if targetModel == nil {
					t.Errorf("expected meta[%d].Relation.Model() not nil, got nil", i)
					continue metaLoop
				}

				if targetField == nil {
					t.Errorf("expected meta[%d].Relation.Field() not nil, got nil", i)
					continue metaLoop
				}

				if fromModel == nil || fromField == nil {
					t.Errorf("expected meta[%d].Relation.From() model/field not nil, got %T/%T", i, fromModel, fromField)
					continue metaLoop
				}

				if len(test.expected.relationchain[i]) == 3 {
					if through == nil {
						t.Errorf("expected meta[%d].Relation.Through() not nil, got nil", i)
						continue metaLoop
					}

					var (
						sourceFieldStr  string
						throughModelStr string
						targetFieldStr  string
					)

					var parts = strings.Split(test.expected.relationchain[i][1], ".")
					if len(parts) != 3 {
						t.Error("Malformed test.expected.relationchain, expected 3 parts for through relation")
						continue metaLoop
					}

					sourceFieldStr = parts[0]
					throughModelStr = parts[1]
					targetFieldStr = parts[2]

					if sourceFieldStr != through.SourceField() {
						t.Errorf("expected meta[%d].Relation.Through.SourceField() %s, got %s", i, sourceFieldStr, through.SourceField())
					}

					var rtyp = reflect.TypeOf(through.Model())
					if rtyp.Kind() == reflect.Ptr {
						rtyp = rtyp.Elem()
					}

					if throughModelStr != rtyp.Name() {
						t.Errorf("expected meta[%d].Relation.Through.Model() %q, got %q", i, throughModelStr, rtyp.Name())
					}

					if targetFieldStr != through.TargetField() {
						t.Errorf("expected meta[%d].Relation.Through.TargetField() %q, got %q", i, targetFieldStr, through.TargetField())
					}

					t.Logf(
						"through relation %T.%s -> %T.%s, %T.%s -> %T.%s",
						fromModel, fromModel.FieldDefs().Primary().Name(), through.Model(), sourceFieldStr,
						through.Model(), targetFieldStr, targetModel, targetModel.FieldDefs().Primary().Name(),
					)
				} else {
					t.Logf("relation %T.%s -> %T.%s", fromModel, fromField.Name(), targetModel, targetField.Name())
				}

				if rel.Type() != test.expected.relationType {
					t.Errorf("expected meta[%d].Relation.Type() %d, got %d", i, test.expected.relationType, rel.Type())
				}

				//var checkPart = func(t *testing.T, i, j int, part string) {
				//	var expectedField, expectedModel = nameParts(part)
				//
				//	var (
				//		rtyp  = reflect.TypeOf(model)
				//	)
				//
				//	if rtyp.Kind() == reflect.Ptr {
				//		rtyp = rtyp.Elem()
				//	}
				//
				//	var modelName = rtyp.Name()
				//	if modelName == "" {
				//		t.Errorf("expected meta[%d].Relation.Chain()[%d] model name empty for %s (%T)", i, j, part, model)
				//		return
				//	}
				//
				//	if modelName != expectedModel {
				//		t.Errorf("expected meta[%d].Relation.Chain()[%d] model %s, got %s", i, j, expectedModel, modelName)
				//		return
				//	}
				//
				//	if field.Name() != expectedField {
				//		t.Errorf("expected meta[%d].Relation.Chain()[%d] field %s, got %s", i, j, expectedField, field.Name())
				//		return
				//	}
				//}

				//var chain = rel.Chain()
				//for j, part := range test.expected.relationchain[i] {
				//	var expectedField, expectedModel = nameParts(part)
				//
				//	var (
				//		model = chain.Model()
				//		field = chain.Field()
				//		rtyp  = reflect.TypeOf(model)
				//	)
				//
				//	if rtyp.Kind() == reflect.Ptr {
				//		rtyp = rtyp.Elem()
				//	}
				//
				//	var modelName = rtyp.Name()
				//	if modelName == "" {
				//		t.Errorf("expected meta[%d].Relation.Chain()[%d] model name empty for %s (%T)", i, j, part, model)
				//		break
				//	}
				//
				//	if modelName != expectedModel {
				//		t.Errorf("expected meta[%d].Relation.Chain()[%d] model %s, got %s", i, j, expectedModel, modelName)
				//		break
				//	}
				//
				//	if field.Name() != expectedField {
				//		t.Errorf("expected meta[%d].Relation.Chain()[%d] field %s, got %s", i, j, expectedField, field.Name())
				//		break
				//	}
				//
				//	chain = chain.To()
				//
				//	if chain == nil && j < len(test.expected.relationchain[i])-1 {
				//		t.Errorf("expected meta[%d].Relation.Chain()[%d] child relation not nil, got nil %v", i, j, test.expected.relationchain[i])
				//		break
				//	}
				//
				//	if chain != nil && j >= len(test.expected.relationchain[i])-1 {
				//		t.Errorf("expected meta[%d].Relation.Chain()[%d] child relation nil, got %T", i, j, model)
				//		break
				//	}
				//}

				//var j = 0
				//for chain != nil {
				//	var rTyp = reflect.TypeOf(chain.Model())
				//	var _, modelName = nameParts(rTyp.Name())
				//
				//	chainNames = append(chainNames, fmt.Sprintf(
				//		"%s.%s",
				//		chain.Field().Name(),
				//		modelName,
				//	))
				//	if j >= len(test.expected.relationchain[i]) {
				//		t.Errorf("expected meta[%d].Relation.Chain()[%d] length %d, got %d %v", i, j, len(test.expected.relationchain[i]), len(chainNames), chainNames)
				//		break
				//	}
				//
				//	var expectedNameParts = test.expected.relationchain[i][j]
				//	var expectedField, expectedModel = nameParts(expectedNameParts)
				//
				//	if modelName != expectedModel {
				//		t.Errorf("expected meta[%d].Relation.Chain()[%d] model %s, got %s %v", i, j, test.expected.relationchain[i][j], modelName, chainNames)
				//	}
				//
				//	if chain.Field().Name() != expectedField {
				//		t.Errorf("expected meta[%d].Relation.Chain()[%d] field %s, got %s %v", i, j, test.expected.relationchain[i][j], chain.Field().Name(), chainNames)
				//	}
				//
				//	j++
				//	chain = chain.To()
				//}
			}

			//var sb = strings.Builder{}
			//for _, current := range meta {
			//	if current.Field != nil {
			//		sb.WriteString(current.Field.Name())
			//	}
			//	if current.Child() != nil {
			//		sb.WriteString(".")
			//	}
			//}
			//
			//t.Logf("meta string = %s", sb.String())
			//t.Logf("meta.Last.String() = %s", meta.Last().String())
		})
	}
}
