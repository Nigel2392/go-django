package widgets

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/url"
	"path/filepath"

	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/filesystem"
	"github.com/Nigel2392/go-django/src/core/filesystem/mediafiles"
	"github.com/Nigel2392/go-django/src/core/filesystem/tpl"
)

//
//type FileWidgetFile struct {
//	Name    string
//	Bytes   []byte
//	Pos     int
//	CloseFn func() error
//}
//
//func (f *FileWidgetFile) Read(p []byte) (n int, err error) {
//	if f.Pos >= len(f.Bytes) {
//		return 0, io.EOF
//	}
//	n = copy(p, f.Bytes[f.Pos:])
//	f.Pos += n
//	return n, nil
//}
//
//func (f *FileWidgetFile) Seek(offset int64, whence int) (int64, error) {
//	switch whence {
//	case io.SeekStart:
//		f.Pos = int(offset)
//	case io.SeekCurrent:
//		f.Pos += int(offset)
//	case io.SeekEnd:
//		f.Pos = len(f.Bytes) + int(offset)
//	}
//	return int64(f.Pos), nil
//}
//
//func (f *FileWidgetFile) Close() error {
//	if f.CloseFn != nil {
//		return f.CloseFn()
//	}
//	return nil
//}

type FileWidget struct {
	*BaseWidget
	Validators []func(filename string, file io.Reader) error
}

type FileObject struct {
	Name string
	File *bytes.Buffer
}

func NewFileInput(attrs map[string]string, validators ...func(filename string, file io.Reader) error) Widget {
	var base = NewBaseWidget("file", "forms/widgets/file.html", attrs)
	var widget = &FileWidget{base, validators}
	return widget
}

func (f *FileWidget) ValueOmittedFromData(ctx context.Context, data url.Values, files map[string][]filesystem.FileHeader, name string) bool {
	var (
		_, ok1 = files[name]
		_, ok2 = data[fmt.Sprintf("%s_path", name)]
	)

	return !ok1 && !ok2
}

func (f *FileWidget) ValueFromDataDict(ctx context.Context, data url.Values, files map[string][]filesystem.FileHeader, name string) (interface{}, []error) {
	var clearVal, ok = data[fmt.Sprintf("%s_clear", name)]
	if ok && len(clearVal) > 0 && (clearVal[0] == "on" || clearVal[0] == "true" || clearVal[0] == "1") {
		return nil, nil
	}

	fileList, hasFile := files[name]
	pathVal, ok := data[fmt.Sprintf("%s_path", name)]
	if ok && !hasFile && len(pathVal) > 0 && pathVal[0] != "" {
		return &FileObject{
			Name: pathVal[0],
			File: nil,
		}, nil
	}

	if !hasFile {
		return nil, nil
	}

	var fileHeader = fileList[0]
	var file, err = fileHeader.Open()
	if err != nil {
		return nil, []error{err}
	}
	defer file.Close()

	var buf = new(bytes.Buffer)
	if _, err := io.Copy(buf, file); err != nil {
		return nil, []error{err}
	}

	var fileName = filepath.Clean(fileHeader.Name())
	for _, validator := range f.Validators {
		if err := validator(fileName, buf); err != nil {
			return nil, []error{err}
		}
	}

	var fileObj = &FileObject{
		Name: fileName,
		File: buf,
	}

	return fileObj, nil
}

func (f *FileWidget) ValueToForm(value interface{}) interface{} {
	if value == nil {
		return nil
	}

	switch v := value.(type) {
	case *FileObject:
		return v.Name
	case mediafiles.StoredObject:
		return v.Path()
	case string:
		return v
	default:
		return nil
	}
}

func (f *FileWidget) GetContextData(c context.Context, id, name string, value interface{}, attrs map[string]string) ctx.Context {
	var widgetCtx = f.BaseWidget.GetContextData(c, id, name, value, attrs)
	var data = widgetCtx.Data()
	var widgetAttrs = data["attrs"].(map[string]string)
	var _, required = widgetAttrs["required"]
	if required {
		data["required"] = true
	}
	data["file_value"] = data["value"]
	data["context"] = c
	delete(data, "value")
	return widgetCtx
}

func (f *FileWidget) RenderWithErrors(ctx context.Context, w io.Writer, id string, name string, value interface{}, errors []error, attrs map[string]string) error {
	var widgetCtx = f.GetContextData(ctx, id, name, value, attrs)
	if errors != nil {
		widgetCtx.Set("errors", errors)
	}
	return tpl.FRender(w, widgetCtx, f.TemplateName)
}

func (f *FileWidget) Render(ctx context.Context, w io.Writer, id string, name string, value interface{}, attrs map[string]string) error {
	return f.RenderWithErrors(ctx, w, id, name, value, nil, attrs)
}
