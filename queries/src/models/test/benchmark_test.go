package models_test

import (
	"testing"

	"github.com/Nigel2392/go-django/queries/src/fields"
	"github.com/Nigel2392/go-django/queries/src/models"
	"github.com/Nigel2392/go-django/src/core/attrs"
)

func init() {
	attrs.RegisterModel(&BasicJoinedModel{})
	attrs.RegisterModel(&BasicModel{})
	attrs.RegisterModel(&ComplexModel{})
}

type BasicJoinedModel struct {
	ID        int
	Age       int
	FirstName string
	LastName  string
}

func (m *BasicJoinedModel) FieldDefs() attrs.Definitions {
	return attrs.Define(m,
		attrs.NewField(m, "ID"),
		attrs.NewField(m, "Age"),
		attrs.NewField(m, "FirstName"),
		attrs.NewField(m, "LastName"),
	)
}

type BasicModel struct {
	ID          int
	Title       string
	Description string
	Joined      *BasicJoinedModel
}

func (m *BasicModel) FieldDefs() attrs.Definitions {
	return attrs.Define[attrs.Definer, any](m,
		attrs.NewField(m, "ID"),
		attrs.NewField(m, "Title"),
		attrs.NewField(m, "Description"),
		attrs.NewField(m, "Joined", &attrs.FieldConfig{
			Null:          true,
			Column:        "joined_id",
			RelForeignKey: attrs.Relate(&BasicJoinedModel{}, "", nil),
		}),
		fields.Embed("Joined"),
	)
}

type ComplexModel struct {
	models.Model
	ID          int
	Title       string
	Description string
	Joined      *BasicJoinedModel
}

func (m *ComplexModel) FieldDefs() attrs.Definitions {
	// using a function here will make sure that the fields dont get allocated
	// each time, we do not want to allocate fields for each call to FieldDefs when
	// the result might be cached.
	return m.Model.Define(m, func(def attrs.Definer) []any {
		return []any{
			attrs.NewField(m, "ID"),
			attrs.NewField(m, "Title"),
			attrs.NewField(m, "Description"),
			attrs.NewField(m, "Joined", &attrs.FieldConfig{
				Null:          true,
				Column:        "joined_id",
				RelForeignKey: attrs.Relate(&BasicJoinedModel{}, "", nil),
			}),
			fields.Embed("Joined"),
		}
	})
}

func BenchmarkCreateObjects(b *testing.B) {
	b.Run("BasicModel", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			var m = attrs.NewObject[*BasicModel](&BasicModel{})
			_ = m
		}
	})

	b.Run("ComplexModel", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			var m = attrs.NewObject[*ComplexModel](&ComplexModel{})
			_ = m
		}
	})
}

func BenchmarkFieldDefs(b *testing.B) {

	b.Run("BasicModel", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			var m = &BasicModel{
				ID:          1,
				Title:       "Test",
				Description: "Test description",
			}

			var defs = m.FieldDefs()
			_ = defs
		}
	})

	b.Run("BasicModelWithJoined", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			var m = &BasicModel{
				ID:          1,
				Title:       "Test",
				Description: "Test description",
				Joined: &BasicJoinedModel{
					ID:        1,
					Age:       30,
					FirstName: "John",
					LastName:  "Doe",
				},
			}

			var defs = m.FieldDefs()
			_ = defs
		}
	})

	b.Run("BasicModelFieldDefsCantCache", func(b *testing.B) {
		var m = &BasicModel{
			ID:          1,
			Title:       "Test",
			Description: "Test description",
			Joined: &BasicJoinedModel{
				ID:        1,
				Age:       30,
				FirstName: "John",
				LastName:  "Doe",
			},
		}

		var defs = m.FieldDefs()

		for i := 0; i < b.N; i++ {
			defs = m.FieldDefs()
			_ = defs
		}
	})

	b.Run("ComplexModel", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			var m = &ComplexModel{
				ID:          1,
				Title:       "Test",
				Description: "Test description",
			}

			var defs = m.FieldDefs()
			_ = defs
		}
	})

	b.Run("ComplexModelWithSetup", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			var m = models.Setup(&ComplexModel{
				ID:          1,
				Title:       "Test",
				Description: "Test description",
			})

			var defs = m.FieldDefs()
			_ = defs
		}
	})

	b.Run("ComplexModelWithJoined", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			var m = &ComplexModel{
				ID:          1,
				Title:       "Test",
				Description: "Test description",
				Joined: &BasicJoinedModel{
					ID:        1,
					Age:       30,
					FirstName: "John",
					LastName:  "Doe",
				},
			}

			var defs = m.FieldDefs()
			_ = defs
		}
	})

	b.Run("ComplexModelWithJoinedAndSetup", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			var m = models.Setup(&ComplexModel{
				ID:          1,
				Title:       "Test",
				Description: "Test description",
				Joined: &BasicJoinedModel{
					ID:        1,
					Age:       30,
					FirstName: "John",
					LastName:  "Doe",
				},
			})
			var defs = m.FieldDefs()
			_ = defs
		}
	})

	b.Run("ComplexModelWithJoinedAndSetupCached", func(b *testing.B) {
		var m = models.Setup(&ComplexModel{
			ID:          1,
			Title:       "Test",
			Description: "Test description",
			Joined: &BasicJoinedModel{
				ID:        1,
				Age:       30,
				FirstName: "John",
				LastName:  "Doe",
			},
		})

		var defs = m.FieldDefs()
		for i := 0; i < b.N; i++ {
			defs = m.FieldDefs()
			_ = defs
		}
	})

}
