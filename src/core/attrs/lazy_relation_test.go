package attrs_test

import (
	"reflect"
	"testing"

	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/apps"
	"github.com/Nigel2392/go-django/src/core/attrs"
)

type LazyRelationTest struct {
	ID      int64
	Title   string
	Targets []LazyRelationTestTarget
}

func (l *LazyRelationTest) FieldDefs() attrs.Definitions {
	return attrs.Define(l,
		attrs.Unbound("ID", &attrs.FieldConfig{
			Primary: true,
			Column:  "id",
		}),
		attrs.Unbound("Title", &attrs.FieldConfig{
			Column: "title",
		}),
		attrs.Unbound("Targets", &attrs.FieldConfig{
			Attributes: map[string]any{
				attrs.AttrReverseAliasKey: "TargetsReversed",
			},
			RelForeignKeyReverse: attrs.RelatedDeferred(
				attrs.RelOneToMany,
				"attrs_test.LazyRelationTestTarget",
				"", nil,
			),
		}),
	)
}

type LazyRelationTestTarget struct {
	ID   int64
	Name string
}

func (l *LazyRelationTestTarget) FieldDefs() attrs.Definitions {
	return attrs.Define(l,
		attrs.Unbound("ID", &attrs.FieldConfig{
			Primary: true,
			Column:  "id",
		}),
		attrs.Unbound("Name", &attrs.FieldConfig{
			Column: "name",
		}),
	)
}

func TestDeferredRelation(t *testing.T) {
	var old = django.Global
	defer func() {
		django.Global = old
	}()
	django.Global = nil

	var app = django.App(
		django.AppSettings(django.Config(
			map[string]interface{}{},
		)),
		django.Flag(
			django.FlagSkipChecks,
			django.FlagSkipCmds,
			django.FlagSkipDepsCheck,
		),
		django.Apps(&apps.AppConfig{
			AppName: "attrs_test",
			ModelObjects: []attrs.Definer{
				&LazyRelationTest{},
				&LazyRelationTestTarget{},
			},
		}),
	)

	if err := app.Initialize(); err != nil {
		t.Fatalf("Failed to initialize app: %s", err)
	}

	var test = &LazyRelationTest{}
	var defs = test.FieldDefs()
	var field, _ = defs.Field("Targets")
	if field == nil {
		t.Fatal("Expected field 'Targets' to be defined")
	}

	var rel = field.Rel()
	if rel.Type() != attrs.RelOneToMany {
		t.Fatalf("Expected relation type to be OneToMany, got %s", rel.Type())
	}

	var targetModel = rel.Model()
	if targetModel == nil {
		t.Fatal("Expected relation to have a target model")
	}

	if reflect.TypeOf(targetModel) != reflect.TypeOf(&LazyRelationTestTarget{}) {
		t.Fatalf("Expected target model to be of type LazyRelationTestTarget, got %T", targetModel)
	}

	var meta = attrs.GetModelMeta(targetModel)
	var reverseRel, ok = meta.Reverse("TargetsReversed")
	if !ok {
		t.Fatalf("Expected reverse relation 'TargetsReversed' to be defined: %v", meta.ReverseMap().Keys())
	}

	if reverseRel.Type() != attrs.RelManyToOne {
		t.Fatalf("Expected reverse relation type to be ManyToOne, got %s", reverseRel.Type())
	}

	var targetField = reverseRel.Field()
	if targetField == nil {
		t.Fatal("Expected reverse relation to have a field")
	}

	if targetField.Name() != "Targets" {
		t.Fatalf("Expected reverse relation field name to be 'Targets', got %s", targetField.Name())
	}
}
