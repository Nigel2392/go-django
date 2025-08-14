package documents

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"hash"
	"time"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/queries/src/models"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/filesystem/mediafiles"
	"github.com/Nigel2392/go-django/src/core/trans"
)

var (
	_ queries.ActsBeforeCreate = (*Document)(nil)
	_ queries.ContextValidator = (*Document)(nil)
)

func newDocumentHasher() hash.Hash {
	return sha256.New()
}

// readonly:id,created_at
type Document struct {
	models.Model
	ID        uint32        `json:"id"`
	Title     string        `json:"title"`
	Path      string        `json:"path"`
	CreatedAt time.Time     `json:"created_at"`
	FileSize  sql.NullInt32 `json:"file_size"`
	FileHash  string        `json:"file_hash"`

	file mediafiles.StoredObject
}

func (o *Document) String() string {
	return fmt.Sprintf(
		"<Document %v>",
		o.ID,
	)
}

func (o *Document) BeforeCreate(ctx context.Context) error {
	if o.CreatedAt.IsZero() {
		o.CreatedAt = time.Now()
	}
	return nil
}

func (o *Document) Validate(ctx context.Context) error {
	if o.Path == "" {
		return errors.ValueError.Wrapf(
			"Document path cannot be empty",
		)
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

func (o *Document) File() (mediafiles.StoredObject, error) {
	if o.file != nil {
		return o.file, nil
	}

	if o.Path == "" {
		return nil, errors.ValueError.Wrapf(
			"Document path cannot be empty",
		)
	}

	var backend = app.MediaBackend()
	var f, err = backend.Open(o.Path)
	if err != nil {
		return nil, err
	}
	o.file = f
	return f, nil
}

func (o *Document) FieldDefs() attrs.Definitions {
	var fields = make([]attrs.Field, 6)
	fields[0] = attrs.NewField(
		o, "ID", &attrs.FieldConfig{
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
			Label:    trans.S("File Size"),
			ReadOnly: true,
		},
	)
	fields[5] = attrs.NewField(
		o, "FileHash", &attrs.FieldConfig{
			Null:     true,
			Blank:    true,
			ReadOnly: true,
			Label:    trans.S("File Hash"),
		},
	)
	return o.Model.Define(o, fields)
}
