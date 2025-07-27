package preload_test

import (
	"context"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/fields"
	"github.com/Nigel2392/go-django/queries/src/models"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/Nigel2392/go-django/src/core/trans"
)

var (
	_ queries.ActsBeforeSave   = (*PreloadAuthor)(nil)
	_ queries.ActsBeforeCreate = (*PreloadAuthor)(nil)
)

type PreloadBook struct {
	models.Model `table:"preload_books" json:"-"`
	ID           uint64                                        `json:"id"`
	Title        string                                        `json:"title"`
	Authors      *queries.RelM2M[attrs.Definer, attrs.Definer] `json:"-"`
}

func (pb *PreloadBook) FieldDefs() attrs.Definitions {
	return pb.Model.Define(pb,
		attrs.Unbound("ID", &attrs.FieldConfig{
			ReadOnly: true,
			Primary:  true,
			Column:   "id",
		}),
		attrs.Unbound("Title", &attrs.FieldConfig{
			Label:     "Book Title",
			HelpText:  trans.S("Title of the book. This is the title that will be displayed in the UI."),
			MaxLength: 100,
		}),
	)
}

type PreloadAuthorBook struct {
	models.Model `table:"preload_author_books" json:"-"`
	Author       *PreloadAuthor `json:"author_id"`
	Book         *PreloadBook   `json:"book_id"`
}

func (pab *PreloadAuthorBook) UniqueTogether() [][]string {
	return [][]string{
		{"Author", "Book"},
	}
}

func (pab *PreloadAuthorBook) SourceField() string {
	return "Author"
}

func (pab *PreloadAuthorBook) TargetField() string {
	return "Book"
}

func (pab *PreloadAuthorBook) FieldDefs() attrs.Definitions {
	return pab.Model.Define(pab,
		attrs.Unbound("Author", &attrs.FieldConfig{
			ReadOnly: true,
			Column:   "author_id",
		}),
		attrs.Unbound("Book", &attrs.FieldConfig{
			ReadOnly: true,
			Column:   "book_id",
		}),
	)
}

type PreloadAuthorProfile struct {
	models.Model `table:"preload_author_profiles" json:"-"`
	ID           uint64         `json:"id" attrs:"primary;readonly"`
	Email        *drivers.Email `json:"email" attrs:"null;blank;max_length:254"`
	FirstName    string         `json:"first_name" attrs:"max_length:50"`
	LastName     string         `json:"last_name" attrs:"max_length:50"`
	Author       *PreloadAuthor `json:"author" attrs:"-"`
}

func (pap *PreloadAuthorProfile) BeforeSave(ctx context.Context) error {
	logger.Warnf("BeforeSave called for PreloadAuthorProfile", pap.ID, pap.Email, pap.Author)
	return nil
}

func (pap *PreloadAuthorProfile) FieldDefs() attrs.Definitions {
	return pap.Model.Define(pap, attrs.AutoFieldList(
		pap, "*",
		fields.NewOneToOneField[*PreloadAuthor](pap,
			"Author", &fields.FieldConfig{
				AllowEdit:   true,
				ColumnName:  "author_id",
				ScanTo:      &pap.Author,
				ReverseName: "Profile",
				Rel: attrs.Relate(
					&PreloadAuthor{},
					"", nil,
				),
			},
		),
	))
}

type PreloadAuthor struct {
	models.Model `table:"preload_authors" json:"-"`
	ID           uint64                                            `json:"id"`
	Name         string                                            `json:"name"`
	CreatedAt    drivers.DateTime                                  `json:"created_at" attrs:"readonly"`
	UpdatedAt    drivers.DateTime                                  `json:"updated_at" attrs:"readonly"`
	Books        *queries.RelM2M[*PreloadBook, *PreloadAuthorBook] `json:"-"`
	Profile      *PreloadAuthorProfile                             `json:"profile,omitempty" attrs:"-"`
}

func (pa *PreloadAuthor) BeforeCreate(ctx context.Context) error {
	// Ensure that the CreatedAt field is set before creating.
	if pa.CreatedAt.IsZero() {
		pa.CreatedAt = drivers.CurrentDateTime()
	}
	// Set UpdatedAt to the current time as well.
	pa.UpdatedAt = drivers.CurrentDateTime()
	return nil
}

func (pa *PreloadAuthor) BeforeSave(ctx context.Context) error {
	// Ensure that the CreatedAt and UpdatedAt fields are set before saving.
	if pa.CreatedAt.IsZero() {
		pa.CreatedAt = drivers.CurrentDateTime()
	}
	pa.UpdatedAt = drivers.CurrentDateTime()
	return nil
}

func (pa *PreloadAuthor) FieldDefs() attrs.Definitions {
	return pa.Model.Define(pa,
		attrs.Unbound("ID", &attrs.FieldConfig{
			ReadOnly: true,
			Primary:  true,
			Column:   "id",
		}),
		attrs.Unbound("Name", &attrs.FieldConfig{
			Label:     "Author Name",
			HelpText:  trans.S("Name of the author. This is the name that will be displayed in the UI."),
			MaxLength: 100,
		}),
		attrs.Unbound("CreatedAt", &attrs.FieldConfig{
			ReadOnly: true,
			Column:   "created_at",
		}),
		attrs.Unbound("UpdatedAt", &attrs.FieldConfig{
			ReadOnly: true,
			Column:   "updated_at",
		}),
		fields.NewManyToManyField[*queries.RelM2M[*PreloadBook, *PreloadAuthorBook]](pa, "Books", &fields.FieldConfig{
			ScanTo:      &pa.Books,
			ReverseName: "Authors",
			Rel: attrs.Relate(
				&PreloadBook{},
				"", &attrs.ThroughModel{
					This:   &PreloadAuthorBook{},
					Source: (&PreloadAuthorBook{}).SourceField(),
					Target: (&PreloadAuthorBook{}).TargetField(),
				},
			),
		}),
	)
}
