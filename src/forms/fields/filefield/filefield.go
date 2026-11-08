package filefield

import (
	"fmt"
	"io"
	"path/filepath"
	"reflect"

	"github.com/Nigel2392/go-django/src/core/assert"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/errs"
	"github.com/Nigel2392/go-django/src/core/filesystem/mediafiles"
	"github.com/Nigel2392/go-django/src/forms/fields"
	"github.com/Nigel2392/go-django/src/forms/widgets"
	"github.com/Nigel2392/go-django/src/forms/widgets/filewidget"
)

func init() {

	attrs.RegisterFormFieldGetter(
		mediafiles.SimpleStoredObject{},
		func(f attrs.Field, new_field_t_indirected reflect.Type, field_v reflect.Value, opts ...func(fields.Field)) (fields.Field, bool) {
			var atts = f.Attrs()
			var uploadTo, ok = atts[attrs.AttrUploadToKey]
			if !ok {
				uploadTo = ""
			}

			var uploadToFn func(fileObject *widgets.FileObject) string
			switch uploadTo := uploadTo.(type) {
			case string:
				uploadToFn = func(fileObject *widgets.FileObject) string {
					if uploadTo == "" {
						return fileObject.Name
					}
					return filepath.Join(uploadTo, fileObject.Name)
				}
			case func() string:
				uploadToFn = func(fileObject *widgets.FileObject) string { return filepath.Join(uploadTo(), fileObject.Name) }
			case func(fileObject *widgets.FileObject) string:
				uploadToFn = uploadTo
			default:
				assert.Fail(
					"invalid uploadTo type %T for field %s",
					uploadTo, f.Name(),
				)
			}

			var formfield = FileField("", opts...)
			formfield.UploadTo = uploadToFn
			return formfield, true
		},
	)
}

type FileStorageField struct {
	*fields.BaseField
	IsImageField   bool
	DisallowClear  bool
	StorageBackend mediafiles.Backend
	StorageEngine  string
	Extensions     []string // Allowed file extensions
	UploadTo       func(fileObject *widgets.FileObject) string
	Validators     []func(filename string, file io.Reader) error
}

func FileFieldWithExtensions(allowedExtensions []string) func(fields.Field) {
	return func(f fields.Field) {
		var fld, ok = f.(*FileStorageField)
		if !ok {
			panic(fmt.Sprintf(
				"FileFieldWithExtensions requires a FileStorageField, got %T",
				f,
			))
		}

		fld.Extensions = allowedExtensions
	}
}

func FileField(engine string, opts ...func(fields.Field)) *FileStorageField {
	var f = &FileStorageField{
		BaseField:     fields.NewField(opts...),
		StorageEngine: engine,
	}
	return f
}

func ImageFileField(engine string, opts ...func(fields.Field)) *FileStorageField {
	var f = &FileStorageField{
		IsImageField:  true,
		BaseField:     fields.NewField(opts...),
		StorageEngine: engine,
	}
	return f
}

func (i *FileStorageField) ValueToGo(value interface{}) (interface{}, error) {
	return i.Widget().ValueToGo(value)
}

func (i *FileStorageField) ValueToForm(value interface{}) interface{} {
	return i.Widget().ValueToForm(value)
}

func (f *FileStorageField) Save(value interface{}) (interface{}, error) {
	if value == nil {
		return nil, nil
	}

	var file, ok = value.(*widgets.FileObject)
	if !ok {
		return nil, errs.Wrap(
			errs.ErrInvalidType,
			fmt.Sprintf("Expected *widgets.FileObject, got %T", value),
		)
	}

	var storage = f.StorageBackend
	if storage == nil {
		storage, ok = mediafiles.RetrieveBackend(f.StorageEngine)
		assert.True(ok, "Storage engine %q not found", f.StorageEngine)
	}

	if file.File == nil {
		return storage.Open(file.Name)
	}

	if f.UploadTo == nil {
		f.UploadTo = func(fileObject *widgets.FileObject) string {
			if f.IsImageField {
				return filepath.Join("images", fileObject.Name)
			}
			return fileObject.Name
		}
	}

	var uploadPath = f.UploadTo(file)
	var path, err = storage.Save(uploadPath, file.File)
	if err != nil {
		return nil, err
	}

	return mediafiles.Open(path)
}

func (f *FileStorageField) Widget() widgets.Widget {
	if f.FormWidget != nil {
		return f.FormWidget
	}

	var w *filewidget.FileWidget
	if f.IsImageField {
		w = filewidget.NewImageFileInput(f.Attributes, f.Extensions, f.Validators...)
	} else {
		w = filewidget.NewFileInput(f.Attributes, f.Extensions, f.Validators...)
	}
	w.DisallowClear = f.DisallowClear
	return w
}
