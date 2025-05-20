package attrs_test

import (
	"fmt"
	"reflect"
	"sync"
	"testing"

	"github.com/Nigel2392/go-django/src/core/attrs"
)

var (
	_ attrs.Definer = &throughModel{}
	_ attrs.Definer = &sourceModel{}
	_ attrs.Definer = &targetModel{}
)

type throughModel struct {
	SourceModel *sourceModel
	TargetModel *targetModel
}

func (f *throughModel) FieldDefs() attrs.Definitions {
	return attrs.AutoDefinitions(f)
}

type sourceModel struct {
	ID                int64
	Name              string
	O2OWithThrough    *targetModel
	O2OWithoutThrough *targetModel
	FK                *targetModel
}

func (f *sourceModel) FieldDefs() attrs.Definitions {
	var (
		o2o_through = attrs.Relate(&targetModel{}, "", &attrs.ThroughModel{
			This:   &throughModel{},
			Source: "SourceModel",
			Target: "TargetModel",
		})
		o2o = attrs.Relate(&targetModel{}, "", nil)
		fk  = attrs.Relate(&targetModel{}, "", nil)
	)
	return attrs.Define(f,
		attrs.NewField(f, "ID", &attrs.FieldConfig{
			Primary: true,
		}),
		attrs.NewField(f, "Name", nil),
		attrs.NewField(f, "O2OWithThrough", &attrs.FieldConfig{
			RelOneToOne: o2o_through,
			Attributes: map[string]interface{}{
				attrs.AttrReverseAliasKey: "SourceRev1",
			},
		}),
		attrs.NewField(f, "O2OWithoutThrough", &attrs.FieldConfig{
			RelOneToOne: o2o,
			Attributes: map[string]interface{}{
				attrs.AttrReverseAliasKey: "SourceRev2",
			},
		}),
		attrs.NewField(f, "FK", &attrs.FieldConfig{
			RelForeignKey: fk,
		}),
	)
}

type targetModel struct {
	ID        int64
	Name      string
	Age       int
	SourceRev *sourceModel
	SourceSet []*sourceModel
}

func (f *targetModel) FieldDefs() attrs.Definitions {
	//var fk_rev = attrs.ReverseRelation(
	//	&sourceModel{}, mustGetField((&sourceModel{}).FieldDefs(), "FK"),
	//	&typedRelation{
	//		Relation: attrs.Relate(&targetModel{}, "", nil),
	//		typ:      attrs.RelManyToOne,
	//	},
	//)
	//
	//var o2o_rev = attrs.ReverseRelation(
	//	&sourceModel{}, mustGetField((&sourceModel{}).FieldDefs(), "O2OWithThrough"),
	//	&typedRelation{
	//		Relation: attrs.Relate(&targetModel{}, "O2OWithThrough", &attrs.ThroughModel{
	//			This:   &throughModel{},
	//			Source: "SourceModel",
	//			Target: "TargetModel",
	//		}),
	//		typ: attrs.RelOneToOne,
	//	},
	//)

	return attrs.Define(f,
		attrs.NewField(f, "ID", &attrs.FieldConfig{
			Primary: true,
		}),
		attrs.NewField(f, "Name", nil),
		attrs.NewField(f, "Age", nil),
		//attrs.NewField(f, "SourceSet", &attrs.FieldConfig{
		//	RelForeignKeyReverse: fk_rev,
		//}),
		//attrs.NewField(f, "SourceRev", &attrs.FieldConfig{
		//	RelOneToOne: o2o_rev,
		//}),
	)
}

func mustGetField(defs attrs.Definitions, name string) attrs.Field {
	field, ok := defs.Field(name)
	if !ok {
		panic(fmt.Sprintf("field %q not found", name))
	}
	return field
}

func typeEquals(t1, t2 any) bool {
	switch t1 := t1.(type) {
	case reflect.Type:
		switch t2 := t2.(type) {
		case reflect.Type:
			return t1 == t2
		default:
			return t1 == reflect.TypeOf(t2)
		}
	}

	return reflect.TypeOf(t1) == reflect.TypeOf(t2)
}

func fieldEquals[T attrs.FieldDefinition](f1, f2 T) bool {
	if any(f1) == nil || any(f2) == nil {
		if any(f1) == nil {
			panic("f1 is nil")
		} else {
			panic("f2 is nil")
		}
	}

	return f1.Name() == f2.Name() &&
		typeEquals(f1.Type(), f2.Type()) &&
		typeEquals(f1.Instance(), f2.Instance())
}

func TestFieldEqualsPanic(t *testing.T) {
	var f1 = (*attrs.FieldDef)(nil)
	var f2 = (*attrs.FieldDef)(nil)

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic, got nil")
		}
	}()

	fieldEquals(f1, f2)
}

func TestGetModelMetaConcurrent(t *testing.T) {
	var wg sync.WaitGroup
	var metas = make([]attrs.ModelMeta, 100)
	var models = []attrs.Definer{
		&sourceModel{},
		&targetModel{},
		&throughModel{},
	}

	attrs.RegisterModel(&sourceModel{})
	attrs.RegisterModel(&targetModel{})
	attrs.RegisterModel(&throughModel{})

	wg.Add(100)

	for i := 0; i < 100; i++ {
		go func(i int) {
			defer wg.Done()
			metas[i] = attrs.GetModelMeta(
				models[i%len(models)],
			)
		}(i)
	}
}

func TestO2OWithThrough(t *testing.T) {
	var source = &sourceModel{
		ID:   1,
		Name: "source",
	}

	var (
		defs  = source.FieldDefs()
		field = mustGetField(defs, "O2OWithThrough")
		rel   = field.Rel()

		target         = rel.Model()
		targetField    = rel.Field()
		through        = rel.Through()
		throughSource  = through.SourceField()
		throughTarget  = through.TargetField()
		throughDefiner = through.Model()
	)

	// Check relation target
	if !typeEquals(target, &targetModel{}) {
		t.Errorf("expected %T, got %T", &targetModel{}, target)
	}

	if !fieldEquals[attrs.FieldDefinition](targetField, getField(&targetModel{}, "ID")) {
		t.Errorf("expected %q, got %q", "ID", targetField.Name())
	}

	// Check through model
	if !typeEquals(throughDefiner, &throughModel{}) {
		t.Errorf("expected %T, got %T", &throughModel{}, through)
	}

	if throughSource != "SourceModel" {
		t.Errorf("expected %q, got %q", "SourceModel", throughSource)
	}

	if throughTarget != "TargetModel" {
		t.Errorf("expected %q, got %q", "TargetModel", throughTarget)
	}
}

//	func TestO2OWithThroughReverse(t *testing.T) {
//		var source = &targetModel{
//			ID:   1,
//			Name: "source",
//		}
//
//		var (
//			defs  = source.FieldDefs()
//			field = mustGetField(defs, "SourceRev")
//			rel   = field.Rel()
//
//			target      = rel.Model()
//			targetField = rel.Field()
//			through     = rel.Through()
//		)
//
//		// Check relation target
//		if !typeEquals(target, &sourceModel{}) {
//			t.Errorf("expected %T, got %T", &sourceModel{}, target)
//		}
//
//		if !fieldEquals(targetField, mustGetField((&sourceModel{}).FieldDefs(), "O2OWithThrough")) {
//			t.Errorf("expected %q, got %q", "ID", targetField.Name())
//		}
//
//		// Check through model
//		if !typeEquals(through.Model(), &throughModel{}) {
//			t.Errorf("expected %T, got %T", &throughModel{}, through)
//		}
//
//		if through.SourceField() != "TargetModel" {
//			t.Errorf("expected %q, got %q", "TargetModel", through.SourceField())
//		}
//
//		if through.TargetField() != "SourceModel" {
//			t.Errorf("expected %q, got %q", "SourceModel", through.TargetField())
//		}
//	}
func TestO2OWithoutThrough(t *testing.T) {
	var source = &sourceModel{
		ID:   1,
		Name: "source",
	}

	var (
		defs  = source.FieldDefs()
		field = mustGetField(defs, "O2OWithoutThrough")
		rel   = field.Rel()

		target      = rel.Model()
		targetField = rel.Field()
		through     = rel.Through()
	)

	if through != nil {
		t.Errorf("expected nil, got %T", through)
	}

	if !typeEquals(target, &targetModel{}) {
		t.Errorf("expected %T, got %T", &targetModel{}, target)
	}

	if !fieldEquals[attrs.FieldDefinition](targetField, mustGetField((&targetModel{}).FieldDefs(), "ID")) {
		t.Errorf("expected %q, got %q", "ID", targetField.Name())
	}

}

func TestFK(t *testing.T) {
	var source = &sourceModel{
		ID:   1,
		Name: "source",
	}

	var (
		defs  = source.FieldDefs()
		field = mustGetField(defs, "FK")
		rel   = field.Rel()

		target      = rel.Model()
		targetField = rel.Field()
		through     = rel.Through()
	)

	if through != nil {
		t.Errorf("expected nil, got %T", through)
	}

	if !typeEquals(target, &targetModel{}) {
		t.Errorf("expected %T, got %T", &targetModel{}, target)
	}

	if !fieldEquals[attrs.FieldDefinition](targetField, mustGetField((&targetModel{}).FieldDefs(), "ID")) {
		t.Errorf("expected %q, got %q", "ID", targetField.Name())
	}
}

//
//func TestFKReverse(t *testing.T) {
//	var source = &targetModel{
//		ID:   1,
//		Name: "source",
//		Age:  1,
//	}
//
//	var (
//		defs  = source.FieldDefs()
//		field = mustGetField(defs, "SourceSet")
//		rel   = field.Rel()
//
//		target      = rel.Model()
//		targetField = rel.Field()
//		through     = rel.Through()
//	)
//
//	if through != nil {
//		t.Errorf("expected nil, got %T", through)
//	}
//
//	if !typeEquals(target, &sourceModel{}) {
//		t.Errorf("expected %T, got %T", &targetModel{}, target)
//	}
//
//	if !fieldEquals(targetField, mustGetField((&sourceModel{}).FieldDefs(), "FK")) {
//		t.Errorf("expected %q, got %q", "ID", targetField.Name())
//	}
//
//}
//
