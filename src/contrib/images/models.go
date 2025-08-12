package images

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"hash"
	"reflect"
	"time"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/fields/formfields"
	"github.com/Nigel2392/go-django/queries/src/models"
	"github.com/Nigel2392/go-django/src/core/assert"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/filesystem/mediafiles"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/go-django/src/forms/fields"
	django_formfields "github.com/Nigel2392/go-django/src/forms/fields"
)

var (
	_ queries.ActsBeforeCreate = (*Image)(nil)
	_ queries.ContextValidator = (*Image)(nil)

	ErrEmptyPath = fmt.Errorf("empty path")
)

func init() {
	attrs.RegisterFormFieldGetter(&Image{}, func(f attrs.Field, new_field_t_indirected reflect.Type, field_v reflect.Value, opts ...func(fields.Field)) (fields.Field, bool) {
		var rel = f.Rel()
		if rel == nil {
			return nil, false
		}

		opts = append(opts, django_formfields.Widget(NewImageWidget(map[string]string{
			"accept": "image/*",
		})))

		var relType = rel.Type()
		switch relType {
		case attrs.RelManyToOne:
			return &formfields.ForeignKeyFormField{
				BaseRelationField: formfields.BaseRelationField{
					BaseField: django_formfields.NewField(opts...),
					Field:     f,
					Relation:  rel,
				},
			}, true
		case attrs.RelOneToOne:
			return &formfields.OneToOneFormField{
				BaseRelationField: formfields.BaseRelationField{
					BaseField: django_formfields.NewField(opts...),
					Field:     f,
					Relation:  rel,
				},
			}, true
		default:
			assert.Fail("unknown relation type %s for field %s", relType, f.Name())
		}

		return nil, false
	})
}

func newImageHasher() hash.Hash {
	return sha256.New()
}

// readonly:id,created_at
type Image struct {
	models.Model
	ID        uint32        `json:"id"`
	Title     string        `json:"title"`
	Path      string        `json:"path"`
	CreatedAt time.Time     `json:"created_at"`
	FileSize  sql.NullInt32 `json:"file_size"`
	FileHash  string        `json:"file_hash"`

	file mediafiles.StoredObject
}

func (o *Image) String() string {
	return fmt.Sprintf(
		"<Image %v>",
		o.ID,
	)
}

func (o *Image) BeforeCreate(ctx context.Context) error {
	if o.CreatedAt.IsZero() {
		o.CreatedAt = time.Now()
	}
	return nil
}

func (o *Image) Validate(ctx context.Context) error {
	if o.Path == "" {
		return ErrEmptyPath
	}

	if o.FileSize.Int32 < 0 {
		return fmt.Errorf("file size cannot be negative")
	}
	o.FileSize.Valid = true
	//
	//if o.FileHash == "" {
	//	return fmt.Errorf("file hash cannot be empty")
	//}

	return nil
}

func (o *Image) File() (mediafiles.StoredObject, error) {
	if o.file != nil {
		return o.file, nil
	}

	if o.Path == "" {
		return nil, ErrEmptyPath
	}

	var backend = app.MediaBackend()
	var f, err = backend.Open(o.Path)
	if err != nil {
		return nil, err
	}
	o.file = f
	return f, nil
}

func (o *Image) FieldDefs() attrs.Definitions {
	var fields = make([]attrs.Field, 6)
	fields[0] = attrs.NewField(
		o, "ID", &attrs.FieldConfig{
			Null:     true,
			Blank:    true,
			ReadOnly: true,
			Primary:  true,
			Label:    trans.S("ID"),
		},
	)
	fields[1] = attrs.NewField(
		o, "Title", &attrs.FieldConfig{
			Null:  true,
			Blank: true,
			Label: trans.S("Title"),
		},
	)
	fields[2] = attrs.NewField(
		o, "Path", &attrs.FieldConfig{
			Null:  true,
			Blank: true,
			Label: trans.S("Path"),
		},
	)
	fields[3] = attrs.NewField(
		o, "CreatedAt", &attrs.FieldConfig{
			Null:     true,
			Blank:    true,
			ReadOnly: true,
			Label:    trans.S("Created At"),
		},
	)
	fields[4] = attrs.NewField(
		o, "FileSize", &attrs.FieldConfig{
			Label: trans.S("File Size"),
		},
	)
	fields[5] = attrs.NewField(
		o, "FileHash", &attrs.FieldConfig{
			Null:  true,
			Blank: true,
			Label: trans.S("File Hash"),
		},
	)
	return o.Model.Define(o, fields)
}
