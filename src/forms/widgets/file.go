package widgets

import (
	"io"
	"net/url"

	"github.com/Nigel2392/go-django/src/core/filesystem"
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
	Validators []func(filename string, file io.ReadSeeker) error
}

func NewFileInput(attrs map[string]string, validators ...func(filename string, file io.ReadSeeker) error) Widget {
	var base = NewBaseWidget(S("file"), "forms/widgets/file.html", attrs)
	var widget = &FileWidget{base, validators}
	return widget
}

func (f *FileWidget) ValueOmittedFromData(data url.Values, files map[string][]filesystem.FileHeader, name string) bool {
	var _, ok = files[name]
	return !ok
}

func (f *FileWidget) ValueFromDataDict(data url.Values, files map[string][]filesystem.FileHeader, name string) (interface{}, []error) {
	var fileList, ok = files[name]
	if !ok {
		return nil, nil
	}

	var file = fileList[0]
	var fileObj, err = file.Open()
	if err != nil {
		return nil, []error{err}
	}

	for _, validator := range f.Validators {
		if err := validator(file.Name(), fileObj); err != nil {
			return nil, []error{err}
		}
		if _, err := fileObj.Seek(0, io.SeekStart); err != nil {
			return nil, []error{err}
		}
	}

	return fileObj, nil
}

func (f *FileWidget) ValueToForm(value interface{}) interface{} {
	return nil
}
