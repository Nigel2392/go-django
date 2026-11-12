package specific_test

import (
	"context"
	"os"
	"testing"

	"github.com/Nigel2392/go-django/djester"
	"github.com/Nigel2392/go-django/djester/objects"
	"github.com/Nigel2392/go-django/djester/quest"
	"github.com/Nigel2392/go-django/djester/testdb"
	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/fields"
	"github.com/Nigel2392/go-django/queries/src/specific"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/attrs/embedder"
	"github.com/Nigel2392/go-django/src/core/attrs/fattrs"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
	"github.com/Nigel2392/go-django/src/core/logger"
)

type Author struct {
	ID   int
	Name string
}

func (a *Author) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make(ctx, a,
		attrs.NewField(a, "ID", &attrs.FieldConfig{
			Primary: true,
		}),
		attrs.NewField(a, "Name", nil),
	).WithTableName("djstr_specific_a")
}

type GenericComment[T any] struct {
	// models.Model
	specific.Specific[*GenericComment[T]]
	ID   drivers.ULID
	Text string
}

func (b *GenericComment[T]) BeforeSave(ctx context.Context) error {
	if b.ID.IsZero() {
		b.ID = drivers.NewULID()
	}
	return nil
}

func (b *GenericComment[T]) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make[*GenericComment[T], any](ctx, b,
		attrs.NewField(b, "ID", &attrs.FieldConfig{
			Primary: true,
		}),
		attrs.NewField(b, "Text", nil),
		b.Specific.Fields,
	).WithTableName("djstr_specific_c")
}

type CommentWithAuthor struct {
	*GenericComment[CommentWithAuthor]
	Author *Author
}

func (b *CommentWithAuthor) FieldDefs(ctx context.Context) attrs.Definitions {
	if b.GenericComment == nil {
		b.GenericComment = &GenericComment[CommentWithAuthor]{}
	}

	return attrs.Make(ctx, b,
		attrs.NewField(b, "ID", &attrs.FieldConfig{
			Primary: true,
		}),
		attrs.NewField(b, "Author", &attrs.FieldConfig{
			Column:        "author_id",
			RelForeignKey: attrs.Relate(&Author{}, "", nil),
		}),
	).WithTableName("djstr_specific_cwa")
}

func defaultSpecificOpts() specific.SpecificQuerySetOptions[*GenericComment[CommentWithAuthor], attrs.Definer] {
	return specific.SpecificQuerySetOptions[*GenericComment[CommentWithAuthor], attrs.Definer]{
		GetSpecificQuerySet: func(targetContentType *contenttypes.ContentTypeDefinition, target attrs.Definer) queries.BaseReadQuerySet[attrs.Definer, *queries.QuerySet[attrs.Definer]] {
			return queries.GetQuerySet(target)
		},
	}
}

func TestSpecificQuerySet(t *testing.T) {
	var _, db = testdb.Open()
	_ = django.App(django.Configure(map[string]any{
		django.APPVAR_DATABASE: db,
	}))

	var tables = quest.Table[*testing.T](t,
		&Author{},
		&GenericComment[CommentWithAuthor]{},
		&CommentWithAuthor{},
	)

	attrs.ResetDefinitions.Send(t.Context(), nil)
	tables.Create()
	t.Cleanup(func() {
		tables.Drop()
	})

	var author = &Author{Name: "John Doe"}
	var cwas = []*CommentWithAuthor{
		{Author: author},
		{Author: author},
		{Author: author},
	}
	var gcs = []*GenericComment[CommentWithAuthor]{
		{Text: "Comment 1", Instance: cwas[0]},
		{Text: "Comment 2", Instance: cwas[1]},
		{Text: "Comment 3", Instance: cwas[2]},
	}

	logger.Setup(&logger.Logger{
		Level:       logger.DBG,
		WrapPrefix:  logger.ColoredLogWrapper,
		OutputDebug: os.Stdout,
		OutputInfo:  os.Stdout,
		OutputWarn:  os.Stdout,
		OutputError: os.Stdout,
	})

	authors, delAuthors := quest.CreateObjects(t, []*Author{author})
	author = authors[0]
	cwas, delCwas := quest.CreateObjects(t, cwas)
	gcs, delGcs := quest.CreateObjects(t, gcs)

	defer delAuthors(0)
	defer delCwas(0)
	defer delGcs(0)

	var testAll = objects.QuerySetTest{
		Execute: func(_ *djester.Tester, t *testing.T, ctx context.Context) {
			var qs = queries.GetQuerySetWithContext(ctx, &GenericComment[CommentWithAuthor]{})
			var sqs = specific.GetSpecificQuerySet(qs, defaultSpecificOpts()).WithContext(ctx)

			results, err := sqs.All()
			if err != nil {
				t.Fatalf("Error querying specific queryset: %v", err)
			}

			if len(results) != len(cwas) {
				t.Fatalf("Expected %d results, got %d", len(cwas), len(results))
			}

			t.Logf("row.Object: %T: %#v", results[0].Object, results[0].Object)

			for i, row := range results {
				var specificObj = row.Object.Specific.(*CommentWithAuthor)
				if specificObj.ID != cwas[i].ID || specificObj.ID.IsZero() {
					t.Fatalf("Expected specific object ID to be %d, got %d", cwas[i].ID, specificObj.ID)
				}
			}
		},
	}

	var testFilter = objects.QuerySetTest{
		Execute: func(_ *djester.Tester, t *testing.T, ctx context.Context) {
			var qs = queries.GetQuerySetWithContext(ctx, &GenericComment[CommentWithAuthor]{})
			var sqs = specific.GetSpecificQuerySet(qs, defaultSpecificOpts()).WithContext(ctx)

			results, err := sqs.Filter("Text__icontains", "Comment 2").All()
			if err != nil {
				t.Fatalf("Error querying specific queryset: %v", err)
			}

			if len(results) != 1 {
				t.Fatalf("Expected 1 result, got %d", len(results))
			}

			var row = results[0]
			if row.Object.Original.ID != gcs[1].ID {
				t.Fatalf("Expected generic comment ID %d, got %d", gcs[1].ID, row.Object.Original.ID)
			}

			if row.Object.Specific == nil {
				t.Fatalf("Expected specific object to be present for ID %d => %s", row.Object.Original.ID, row.Object.Original.SpecificID)
			}

			t.Logf("ResultRow Original: %+v", row.Object.Original)
			t.Logf("ResultRow Instance: %+v", row.Object.Specific)

			var specificObj = row.Object.Specific.(*CommentWithAuthor)
			if specificObj.ID != cwas[1].ID || specificObj.ID.IsZero() {
				t.Fatalf("Expected specific object ID %d, got %d", cwas[1].ID, specificObj.ID)
			}
		},
	}

	var testGet = objects.QuerySetTest{
		Execute: func(_ *djester.Tester, t *testing.T, ctx context.Context) {
			var qs = queries.GetQuerySetWithContext(ctx, &GenericComment[CommentWithAuthor]{})
			var sqs = specific.GetSpecificQuerySet(qs, defaultSpecificOpts()).WithContext(ctx)

			row, err := sqs.Filter("ID", gcs[2].ID).Get()
			if err != nil {
				t.Fatalf("Error querying specific queryset: %v", err)
			}

			if row.Object.Original.ID != gcs[2].ID {
				t.Fatalf("Expected generic comment ID %d, got %d", gcs[2].ID, row.Object.Original.ID)
			}

			if row.Object.Specific == nil {
				t.Fatalf("Expected specific object to be present for ID %d => %s", row.Object.Original.ID, row.Object.Original.SpecificID)
			}

			var specificObj = row.Object.Specific.(*CommentWithAuthor)
			if specificObj.ID != cwas[2].ID {
				t.Fatalf("Expected specific object ID %d, got %d", cwas[2].ID, specificObj.ID)
			}
		},
	}

	var testFirstLast = objects.QuerySetTest{
		Execute: func(_ *djester.Tester, t *testing.T, ctx context.Context) {
			var qs = queries.GetQuerySetWithContext(ctx, &GenericComment[CommentWithAuthor]{})
			var sqs = specific.GetSpecificQuerySet(qs, defaultSpecificOpts()).WithContext(ctx)

			first, err := sqs.OrderBy("ID").First()
			if err != nil {
				t.Fatalf("First() failed: %v", err)
			}
			if first.Object.Original.ID != gcs[0].ID {
				t.Fatalf("Expected First() ID %d, got %d", gcs[0].ID, first.Object.Original.ID)
			}

			if first.Object.Specific == nil {
				t.Fatalf("Expected specific object to be present for ID %d => %s", first.Object.Original.ID, first.Object.Original.SpecificID)
			}

			last, err := sqs.OrderBy("ID").Last()
			if err != nil {
				t.Fatalf("Last() failed: %v", err)
			}
			if last.Object.Original.ID != gcs[2].ID {
				t.Fatalf("Expected Last() ID %d, got %d", gcs[2].ID, last.Object.Original.ID)
			}

			if last.Object.Specific == nil {
				t.Fatalf("Expected specific object to be present for ID %d => %s", last.Object.Original.ID, last.Object.Original.SpecificID)
			}

		},
	}

	var testIterAll = objects.QuerySetTest{
		Execute: func(_ *djester.Tester, t *testing.T, ctx context.Context) {
			var qs = queries.GetQuerySetWithContext(ctx, &GenericComment[CommentWithAuthor]{})
			var sqs = specific.GetSpecificQuerySet(qs, defaultSpecificOpts()).WithContext(ctx)

			count, iter, err := sqs.IterAll()
			if err != nil {
				t.Fatalf("IterAll failed: %v", err)
			}

			if count != len(cwas) {
				t.Fatalf("Expected %d count, got %d", len(cwas), count)
			}

			var iterated = 0
			for row, err := range iter {
				if err != nil {
					t.Fatalf("Iteration error: %v", err)
				}
				if row.Object.Original == nil {
					t.Fatalf("Expected Original to be set during IterAll")
				}
				if row.Object.Specific != nil {
					t.Fatalf("Specific should not be populated during IterAll")
				}
				iterated++
			}

			if iterated != len(cwas) {
				t.Fatalf("Iterated %d times, expected %d", iterated, len(cwas))
			}
		},
	}

	var testPreload = objects.QuerySetTest{
		Execute: func(_ *djester.Tester, t *testing.T, ctx context.Context) {
			var qs = queries.GetQuerySetWithContext(ctx, &GenericComment[CommentWithAuthor]{})

			var opts = defaultSpecificOpts()
			opts.GetSpecificQuerySet = func(targetContentType *contenttypes.ContentTypeDefinition, target attrs.Definer) queries.BaseReadQuerySet[attrs.Definer, *queries.QuerySet[attrs.Definer]] {
				// We pass ctx directly to ensure we use the testdb's active context
				return queries.GetQuerySetWithContext(ctx, target).SelectRelated("Author")
			}

			var sqs = specific.GetSpecificQuerySet(qs, opts).WithContext(ctx)
			results, err := sqs.All()
			if err != nil {
				t.Fatalf("Error querying specific queryset: %v", err)
			}

			if len(results) != 3 {
				t.Fatalf("Expected 3 results, got %d", len(results))
			}

			for _, row := range results {

				if row.Object.Specific == nil {
					t.Fatalf("Expected specific object to be present for ID %d => %s", row.Object.Original.ID, row.Object.Original.SpecificID)
				}

				var specificObj = row.Object.Specific.(*CommentWithAuthor)
				if specificObj.Author == nil {
					t.Fatalf("Expected Author to be preloaded, got nil")
				}
				if specificObj.Author.ID != author.ID {
					t.Fatalf("Expected preloaded Author ID %d, got %d", author.ID, specificObj.Author.ID)
				}
			}
		},
	}

	t.Helper()

	t.Run("TestSpecificQuerySet_All", func(t *testing.T) { testAll.Test(nil, t) })
	t.Run("TestSpecificQuerySet_Filter", func(t *testing.T) { testFilter.Test(nil, t) })
	t.Run("TestSpecificQuerySet_Get", func(t *testing.T) { testGet.Test(nil, t) })
	t.Run("TestSpecificQuerySet_FirstAndLast", func(t *testing.T) { testFirstLast.Test(nil, t) })
	t.Run("TestSpecificQuerySet_IterAll", func(t *testing.T) { testIterAll.Test(nil, t) })
	t.Run("TestSpecificQuerySet_PreloadSpecific", func(t *testing.T) { testPreload.Test(nil, t) })
}

type CommentReaction struct {
	ID      int64
	Comment *GenericComment[CommentWithAuthor]
	Emoji   string
}

func (m *CommentReaction) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make(ctx, m,
		fattrs.Field(m, "ID", &m.ID, func() fattrs.PtrFieldConfig[*CommentReaction, int64] {
			return fattrs.PtrFieldConfig[*CommentReaction, int64]{
				Config: attrs.FieldConfig{
					Primary:  true,
					ReadOnly: true,
				},
			}
		}),
		fattrs.Field(m, "Comment", &m.Comment, func() fattrs.PtrFieldConfig[*CommentReaction, *GenericComment[CommentWithAuthor]] {
			return fattrs.PtrFieldConfig[*CommentReaction, *GenericComment[CommentWithAuthor]]{
				Config: attrs.FieldConfig{
					Column:        "comment_id",
					RelForeignKey: attrs.Relate(&GenericComment[CommentWithAuthor]{}, "", nil),
				},
			}
		}),
		fattrs.Field(m, "Emoji", &m.Emoji, func() fattrs.PtrFieldConfig[*CommentReaction, string] {
			return fattrs.PtrFieldConfig[*CommentReaction, string]{}
		}),
	).WithTableName("djstr_specific_reaction")
}

type CommentWithFattrs struct {
	*GenericComment[CommentWithFattrs]
	ID    int64
	Extra string
}

func (m *CommentWithFattrs) FieldDefs(ctx context.Context) attrs.Definitions {
	if m.GenericComment == nil {
		m.GenericComment = &GenericComment[CommentWithFattrs]{}
	}
	return attrs.Make[*CommentWithFattrs, any](ctx, m,
		fields.Embed(ctx, &m.GenericComment, fields.EmbedOptions{
			EmbedFields: []any{"ID", "Text"},
		}),
		fattrs.Field(m, "Extra", &m.Extra, func() fattrs.PtrFieldConfig[*CommentWithFattrs, string] {
			return fattrs.PtrFieldConfig[*CommentWithFattrs, string]{}
		}),
	).WithTableName("djstr_specific_cwf")
}

func TestSpecificForeignKeyAndFattrsRegistration(t *testing.T) {
	t.Run("ForeignKeyToSpecificBaseModel", func(t *testing.T) {
		attrs.RegisterModel(&GenericComment[CommentWithAuthor]{})
		attrs.RegisterModel(&CommentWithAuthor{})
		attrs.RegisterModel(&CommentReaction{})

		reaction := attrs.NewObject[*CommentReaction](t.Context(), &CommentReaction{})
		if reaction == nil {
			t.Fatal("expected non-nil CommentReaction")
		}
		if reaction.Comment != nil && reaction.Comment.Instance != nil {
			t.Fatalf("expected reaction.Comment.Instance to be nil, got %T (%+v)", reaction.Comment.Instance, reaction.Comment.Instance)
		}

		cwa := attrs.NewObject[*CommentWithAuthor](t.Context(), &CommentWithAuthor{})
		cwa.ID = drivers.NewULID()
		reaction.Comment = cwa.GenericComment
		if err := embedder.PreSave.Bind(t.Context(), reaction); err != nil {
			t.Fatalf("PreSave.Bind failed on reaction: %v", err)
		}
		if reaction.Comment.Instance != cwa {
			t.Fatalf("expected reaction.Comment.Instance to remain cwa (%p), got %T (%+v)", cwa, reaction.Comment.Instance, reaction.Comment.Instance)
		}
	})

	t.Run("SpecificSubtypeWithFattrs", func(t *testing.T) {
		attrs.RegisterModel(&GenericComment[CommentWithFattrs]{})
		attrs.RegisterModel(&CommentWithFattrs{})

		cwf := attrs.NewObject[*CommentWithFattrs](t.Context(), &CommentWithFattrs{})
		if cwf == nil {
			t.Fatal("expected non-nil CommentWithFattrs")
		}
		if cwf.GenericComment == nil {
			t.Fatal("expected embedded GenericComment to be initialized")
		}
		if cwf.GenericComment.Instance != cwf {
			t.Fatalf("expected GenericComment.Instance to be %p, got %v", cwf, cwf.GenericComment.Instance)
		}
		defs := attrs.Define(t.Context(), cwf)
		if defs.Primary() == nil {
			t.Fatal("expected embedded ID field to be primary")
		}
	})
}
