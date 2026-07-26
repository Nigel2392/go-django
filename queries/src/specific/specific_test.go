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
	"github.com/Nigel2392/go-django/queries/src/models"
	"github.com/Nigel2392/go-django/queries/src/specific"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/core/attrs"
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
	models.Model
	ID                  int64
	Text                string
	Specific            *T
	SpecificID          int
	SpecificContentType string
}

func (b *GenericComment[T]) FieldDefs(ctx context.Context) attrs.Definitions {
	return b.Model.Define(ctx, b,
		attrs.NewField(b, "ID", &attrs.FieldConfig{
			Primary: true,
		}),
		attrs.NewField(b, "Text", nil),
		attrs.NewField(b, "SpecificID", nil),
		attrs.NewField(b, "SpecificContentType", nil),
	).WithTableName("djstr_specific_c")
}

func (b *GenericComment[T]) BeforeSave(ctx context.Context) error {
	if b.Specific != nil && (b.SpecificID == 0 || b.SpecificContentType == "") {
		var pk = attrs.PrimaryKey(ctx, any(b.Specific).(attrs.Definer))
		switch v := pk.(type) {
		case int:
			b.SpecificID = v
		case int64:
			b.SpecificID = int(v)
		case uint64:
			b.SpecificID = int(v)
		case int32:
			b.SpecificID = int(v)
		case uint32:
			b.SpecificID = int(v)
		}
		b.SpecificContentType = contenttypes.NewContentType(b.Specific).TypeName()
	}
	return nil
}

type CommentWithAuthor struct {
	GenericComment[CommentWithAuthor]
	ID     int
	Author *Author
}

func (b *CommentWithAuthor) FieldDefs(ctx context.Context) attrs.Definitions {
	return b.Model.Define(ctx, b,
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
		GetSpecificPreloadData: func(obj *GenericComment[CommentWithAuthor]) (id any, contentType string, ok bool) {
			if obj.SpecificID != 0 && obj.SpecificContentType != "" {
				return obj.SpecificID, obj.SpecificContentType, true
			}
			return nil, "", false
		},
		GetSpecificTargetID: func(target attrs.Definer) (id any, ok bool) {
			var pk = attrs.PrimaryKey(context.Background(), target)
			switch v := pk.(type) {
			case uint64:
				return int(v), true
			}
			return pk, true
		},
	}
}

func TestSpecificQuerySet(t *testing.T) {
	var _, db = testdb.Open()
	_ = django.App(django.Configure(map[string]any{
		django.APPVAR_DATABASE: db,
	}))

	var tables = quest.Table[*testing.T](nil,
		&Author{},
		&GenericComment[CommentWithAuthor]{},
		&CommentWithAuthor{},
	)

	attrs.ResetDefinitions.Send(nil)
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
		{Text: "Comment 1", Specific: cwas[0]},
		{Text: "Comment 2", Specific: cwas[1]},
		{Text: "Comment 3", Specific: cwas[2]},
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

			for i, row := range results {
				var specificObj = row.Object.Specific.(*CommentWithAuthor)
				if specificObj.ID != cwas[i].ID || specificObj.ID == 0 {
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
				t.Fatalf("Expected specific object to be present for ID %d => %d", row.Object.Original.ID, row.Object.Original.SpecificID)
			}

			t.Logf("ResultRow Original: %+v", row.Object.Original)
			t.Logf("ResultRow Specific: %+v", row.Object.Specific)

			var specificObj = row.Object.Specific.(*CommentWithAuthor)
			if specificObj.ID != cwas[1].ID || specificObj.ID == 0 {
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
				t.Fatalf("Expected specific object to be present for ID %d => %d", row.Object.Original.ID, row.Object.Original.SpecificID)
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
				t.Fatalf("Expected specific object to be present for ID %d => %d", first.Object.Original.ID, first.Object.Original.SpecificID)
			}

			last, err := sqs.OrderBy("ID").Last()
			if err != nil {
				t.Fatalf("Last() failed: %v", err)
			}
			if last.Object.Original.ID != gcs[2].ID {
				t.Fatalf("Expected Last() ID %d, got %d", gcs[2].ID, last.Object.Original.ID)
			}

			if last.Object.Specific == nil {
				t.Fatalf("Expected specific object to be present for ID %d => %d", last.Object.Original.ID, last.Object.Original.SpecificID)
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
				return queries.GetQuerySetWithContext(ctx, target).Preload("Author")
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
					t.Fatalf("Expected specific object to be present for ID %d => %d", row.Object.Original.ID, row.Object.Original.SpecificID)
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
