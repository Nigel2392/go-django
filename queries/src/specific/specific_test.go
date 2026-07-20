package specific_test

import (
	"context"

	"github.com/Nigel2392/go-django/queries/src/models"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
)

type Author struct {
	ID   int64
	Name string
}

func (a *Author) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make(ctx, a,
		attrs.NewField(a, "ID", &attrs.FieldConfig{
			Primary: true,
		}),
		attrs.NewField(a, "Name", nil),
	).WithTableName("a")
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
		attrs.NewField(b, "SpecificID", nil),
		attrs.NewField(b, "SpecificContentType", nil),
	).WithTableName("c")
}

func (b *GenericComment[T]) BeforeSave(ctx context.Context) error {
	if b.Specific != nil && (b.SpecificID == 0 || b.SpecificContentType == "") {
		b.SpecificID = attrs.PrimaryKey(ctx, any(b.Specific).(attrs.Definer)).(int)
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
	return attrs.Make(ctx, b,
		attrs.NewField(b, "ID", &attrs.FieldConfig{
			Primary: true,
		}),
		attrs.NewField(b, "Author", &attrs.FieldConfig{
			RelForeignKey: attrs.Relate(&Author{}, "", nil),
		}),
	).WithTableName("cwa")
}
