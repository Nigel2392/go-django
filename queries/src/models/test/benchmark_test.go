package models_test

import (
	"context"
	"iter"
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

func (m *BasicJoinedModel) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make(ctx, m,
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

func (m *BasicModel) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make[attrs.Definer, any](ctx, m,
		attrs.NewField(m, "ID"),
		attrs.NewField(m, "Title"),
		attrs.NewField(m, "Description"),
		attrs.NewField(m, "Joined", &attrs.FieldConfig{
			Null:          true,
			Column:        "joined_id",
			RelForeignKey: attrs.Relate(&BasicJoinedModel{}, "", nil),
		}),
		fields.Embed(ctx, "Joined"),
	)
}

type BasicConstructedModel struct {
	ID          int
	Title       string
	Description string
	Joined      *BasicJoinedModel
}

func (m *BasicConstructedModel) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make[attrs.Definer, any](ctx, m,
		attrs.Unbound("ID", &attrs.FieldConfig{}),
		attrs.Unbound("Title", &attrs.FieldConfig{}),
		attrs.Unbound("Description", &attrs.FieldConfig{}),
		attrs.Unbound("Joined", &attrs.FieldConfig{
			Null:          true,
			Column:        "joined_id",
			RelForeignKey: attrs.Relate(&BasicJoinedModel{}, "", nil),
		}),
		fields.Embed(ctx, "Joined"),
	)
}

type ComplexModel struct {
	models.Model
	ID          int
	Title       string
	Description string
	Joined      *BasicJoinedModel
}

func (m *ComplexModel) fields(ctx context.Context, def attrs.Definer) []any {
	return []any{
		attrs.NewField(m, "ID"),
		attrs.NewField(m, "Title"),
		attrs.NewField(m, "Description"),
		attrs.NewField(m, "Joined", &attrs.FieldConfig{
			Null:          true,
			Column:        "joined_id",
			RelForeignKey: attrs.Relate(&BasicJoinedModel{}, "", nil),
		}),
		fields.Embed(ctx, "Joined"),
	}
}

func (m *ComplexModel) FieldDefs(ctx context.Context) attrs.Definitions {
	// using a function here will make sure that the fields dont get allocated
	// each time, we do not want to allocate fields for each call to FieldDefs when
	// the result might be cached.
	return m.Model.Define(ctx, m, m.fields)
}

func BenchmarkCreateObjects(b *testing.B) {
	var ctx = context.Background()
	b.Run("BasicModel", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			var m = attrs.NewObject[*BasicModel](ctx, &BasicModel{})
			_ = m
		}
	})

	b.Run("ComplexModel", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			var m = attrs.NewObject[*ComplexModel](ctx, &ComplexModel{})
			_ = m
		}
	})
}

func BenchmarkFieldDefs(b *testing.B) {

	var ctx = attrs.ContextWithFlags(context.Background(), attrs.CtxFlagNone, true)
	b.Run("BasicModel", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			var m = &BasicModel{
				ID:          1,
				Title:       "Test",
				Description: "Test description",
			}

			var defs = m.FieldDefs(ctx)
			_ = defs
		}
	})

	b.Run("BasicConstructedModel", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			var m = &BasicConstructedModel{
				ID:          1,
				Title:       "Test",
				Description: "Test description",
			}

			var defs = m.FieldDefs(ctx)
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

			var defs = m.FieldDefs(ctx)
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

		var defs = m.FieldDefs(ctx)

		for i := 0; i < b.N; i++ {
			defs = m.FieldDefs(ctx)
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

			var defs = m.FieldDefs(ctx)
			_ = defs
		}
	})

	b.Run("ComplexModelWithSetup", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			var m = models.Setup(ctx, &ComplexModel{
				ID:          1,
				Title:       "Test",
				Description: "Test description",
			})

			var defs = m.FieldDefs(ctx)
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

			var defs = m.FieldDefs(ctx)
			_ = defs
		}
	})

	b.Run("ComplexModelWithJoinedAndSetup", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			var m = models.Setup(ctx, &ComplexModel{
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
			var defs = m.FieldDefs(ctx)
			_ = defs
		}
	})

	b.Run("ComplexModelWithJoinedAndSetupCached", func(b *testing.B) {
		var m = models.Setup(ctx, &ComplexModel{
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

		var defs = m.FieldDefs(ctx)
		for i := 0; i < b.N; i++ {
			defs = m.FieldDefs(ctx)
			_ = defs
		}
	})

}

/*
benchmark iterator performance itself

the iter.Pull function is so fucking slow... never use.
*/
type Field struct {
	Name string
}

var SAMPLE = []Field{
	{"A"}, {"B"}, {"C"}, {"D"}, {"E"}, {"F"}, {"G"}, {"H"}, {"I"}, {"J"}, {"K"}, {"L"}, {"M"}, {"N"}, {"O"}, {"P"}, {"Q"}, {"R"}, {"S"}, {"T"}, {"U"}, {"V"}, {"W"}, {"X"}, {"Y"}, {"Z"},
	{"A"}, {"B"}, {"C"}, {"D"}, {"E"}, {"F"}, {"G"}, {"H"}, {"I"}, {"J"}, {"K"}, {"L"}, {"M"}, {"N"}, {"O"}, {"P"}, {"Q"}, {"R"}, {"S"}, {"T"}, {"U"}, {"V"}, {"W"}, {"X"}, {"Y"}, {"Z"},
	{"A"}, {"B"}, {"C"}, {"D"}, {"E"}, {"F"}, {"G"}, {"H"}, {"I"}, {"J"}, {"K"}, {"L"}, {"M"}, {"N"}, {"O"}, {"P"}, {"Q"}, {"R"}, {"S"}, {"T"}, {"U"}, {"V"}, {"W"}, {"X"}, {"Y"}, {"Z"},
	{"A"}, {"B"}, {"C"}, {"D"}, {"E"}, {"F"}, {"G"}, {"H"}, {"I"}, {"J"}, {"K"}, {"L"}, {"M"}, {"N"}, {"O"}, {"P"}, {"Q"}, {"R"}, {"S"}, {"T"}, {"U"}, {"V"}, {"W"}, {"X"}, {"Y"}, {"Z"},
	{"A"}, {"B"}, {"C"}, {"D"}, {"E"}, {"F"}, {"G"}, {"H"}, {"I"}, {"J"}, {"K"}, {"L"}, {"M"}, {"N"}, {"O"}, {"P"}, {"Q"}, {"R"}, {"S"}, {"T"}, {"U"}, {"V"}, {"W"}, {"X"}, {"Y"}, {"Z"},
	{"A"}, {"B"}, {"C"}, {"D"}, {"E"}, {"F"}, {"G"}, {"H"}, {"I"}, {"J"}, {"K"}, {"L"}, {"M"}, {"N"}, {"O"}, {"P"}, {"Q"}, {"R"}, {"S"}, {"T"}, {"U"}, {"V"}, {"W"}, {"X"}, {"Y"}, {"Z"},
	{"A"}, {"B"}, {"C"}, {"D"}, {"E"}, {"F"}, {"G"}, {"H"}, {"I"}, {"J"}, {"K"}, {"L"}, {"M"}, {"N"}, {"O"}, {"P"}, {"Q"}, {"R"}, {"S"}, {"T"}, {"U"}, {"V"}, {"W"}, {"X"}, {"Y"}, {"Z"},
	{"A"}, {"B"}, {"C"}, {"D"}, {"E"}, {"F"}, {"G"}, {"H"}, {"I"}, {"J"}, {"K"}, {"L"}, {"M"}, {"N"}, {"O"}, {"P"}, {"Q"}, {"R"}, {"S"}, {"T"}, {"U"}, {"V"}, {"W"}, {"X"}, {"Y"}, {"Z"},
	{"A"}, {"B"}, {"C"}, {"D"}, {"E"}, {"F"}, {"G"}, {"H"}, {"I"}, {"J"}, {"K"}, {"L"}, {"M"}, {"N"}, {"O"}, {"P"}, {"Q"}, {"R"}, {"S"}, {"T"}, {"U"}, {"V"}, {"W"}, {"X"}, {"Y"}, {"Z"},
	{"A"}, {"B"}, {"C"}, {"D"}, {"E"}, {"F"}, {"G"}, {"H"}, {"I"}, {"J"}, {"K"}, {"L"}, {"M"}, {"N"}, {"O"}, {"P"}, {"Q"}, {"R"}, {"S"}, {"T"}, {"U"}, {"V"}, {"W"}, {"X"}, {"Y"}, {"Z"},
}

func BenchmarkFlatAppend(b *testing.B) {
	b.ResetTimer()
	for b.Loop() {
		out := make([]Field, 0, len(SAMPLE))
		for _, f := range SAMPLE {
			out = append(out, f)
		}
		_ = out
	}
}

func BenchmarkFlatIterator(b *testing.B) {
	var iterator = func(yield func(Field) bool) {
		for _, f := range SAMPLE {
			if !yield(f) {
				return
			}
		}
	}

	b.ResetTimer()

	for b.Loop() {
		out := make([]Field, 0, len(SAMPLE))
		for f := range iterator {
			out = append(out, f)
		}

		_ = out
	}
}

func BenchmarkNestedIteratorForLoop(b *testing.B) {

	var iterator = func(yield func(Field) bool) {
		for _, f := range SAMPLE {
			if !yield(f) {
				return
			}
		}
	}

	var nestedIterator = func(iterator iter.Seq[Field]) iter.Seq[Field] {
		return func(yield func(Field) bool) {
			for f := range iterator {
				if !yield(f) {
					return
				}
			}
		}
	}

	b.ResetTimer()

	for b.Loop() {
		out := make([]Field, 0, len(SAMPLE))
		for f := range nestedIterator(iterator) {
			out = append(out, f)
		}

		_ = out
	}
}

func BenchmarkNestedIteratorPull(b *testing.B) {
	var iterator = func(yield func(Field) bool) {
		for _, f := range SAMPLE {
			if !yield(f) {
				return
			}
		}
	}

	var nestedIterator = func(iterator iter.Seq[Field]) iter.Seq[Field] {
		return func(yield func(Field) bool) {

			var next, stop = iter.Pull(iterator)

			for {
				var next, valid = next()
				if !valid {
					stop()
					return
				}

				if !yield(next) {
					stop()
					return
				}
			}
		}
	}

	b.ResetTimer()

	for b.Loop() {
		out := make([]Field, 0, len(SAMPLE))
		for f := range nestedIterator(iterator) {
			out = append(out, f)
		}

		_ = out
	}
}
