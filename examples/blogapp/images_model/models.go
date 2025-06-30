package images

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"hash"
	"time"

	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/filesystem/mediafiles"
	"github.com/Nigel2392/go-django/src/models"
)

var (
	_ models.ContextSaver   = (*Image)(nil)
	_ models.ContextDeleter = (*Image)(nil)

	ErrEmptyPath = fmt.Errorf("empty path")
)

func newImageHasher() hash.Hash {
	return sha256.New()
}

// readonly:id,created_at
type Image struct {
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

func (o *Image) Save(ctx context.Context) error {
	var queries = NewQueryset(app.DB)
	if o.ID == 0 {
		return queries.InsertImage(ctx, o)
	}
	return queries.UpdateImage(ctx, o)
}

func (o *Image) Update(ctx context.Context) error {
	var q = NewQueryset(app.DB)
	return q.UpdateImage(ctx, o)
}

func (o *Image) Delete(ctx context.Context) error {
	var q = NewQueryset(app.DB)
	return q.DeleteImage(ctx, o)
}

func (o *Image) FieldDefs() attrs.Definitions {
	var fields = make([]attrs.Field, 6)
	fields[0] = attrs.NewField(
		o, "ID", &attrs.FieldConfig{
			Null:     true,
			Blank:    true,
			ReadOnly: true,
			Label:    "ID",
			Primary:  true,
		},
	)
	fields[1] = attrs.NewField(
		o, "Title", &attrs.FieldConfig{
			Null:  true,
			Blank: true,
			Label: "Title",
		},
	)
	fields[2] = attrs.NewField(
		o, "Path", &attrs.FieldConfig{
			Null:  true,
			Blank: true,
			Label: "Path",
		},
	)
	fields[3] = attrs.NewField(
		o, "CreatedAt", &attrs.FieldConfig{
			Null:     true,
			Blank:    true,
			ReadOnly: true,
			Label:    "Created At",
		},
	)
	fields[4] = attrs.NewField(
		o, "FileSize", &attrs.FieldConfig{
			Label: "File Size",
		},
	)
	fields[5] = attrs.NewField(
		o, "FileHash", &attrs.FieldConfig{
			Null:  true,
			Blank: true,
			Label: "File Hash",
		},
	)
	return attrs.Define(o, fields...)
}
