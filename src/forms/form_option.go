package forms

import (
	"maps"
	"mime/multipart"
	"net/http"
	"net/url"
	"sync"

	"github.com/Nigel2392/go-django/src/core/filesystem"
)

type multipartFileHeader struct {
	header *multipart.FileHeader
}

func (f *multipartFileHeader) Name() string {
	return f.header.Filename
}

func (f *multipartFileHeader) Size() int64 {
	return f.header.Size
}

func (f *multipartFileHeader) Open() (multipart.File, error) {
	return f.header.Open()
}

func WithRequestData(method string, r *http.Request) func(Form) {
	if r.Method != method {
		return func(f Form) {
			var (
				data  = make(url.Values)
				files = make(map[string][]filesystem.FileHeader)
			)
			f.WithData(data, files, r)
		}
	}

	var once = new(sync.Once)
	var data = make(url.Values)
	var files = make(map[string][]filesystem.FileHeader)
	var onceFn = func() (url.Values, map[string][]filesystem.FileHeader) {
		once.Do(func() {
			r.ParseForm()

			maps.Copy(data, r.Form)
			if r.MultipartForm != nil && r.MultipartForm.File != nil {
				for k, v := range r.MultipartForm.File {
					var files_ = make([]filesystem.FileHeader, 0, len(v))
					for _, file := range v {
						files_ = append(files_, &multipartFileHeader{file})
					}
					files[k] = files_
				}
			}
		})

		return data, files
	}

	return func(f Form) {
		data, files := onceFn()
		f.WithData(data, files, r)
	}
}

func WithData(data url.Values, files map[string][]filesystem.FileHeader, r *http.Request) func(Form) {
	if files == nil {
		files = make(map[string][]filesystem.FileHeader)
	}

	return func(f Form) {
		f.WithData(data, files, r)
	}
}

func WithFields(fields ...Field) func(Form) {
	return func(f Form) {
		for _, field := range fields {
			f.AddField(field.Name(), field)
		}
	}
}

func WithPrefix(prefix string) func(Form) {
	return func(f Form) {
		f.SetPrefix(prefix)
	}
}

func WithInitial(initial map[string]interface{}) func(Form) {
	return func(f Form) {
		f.SetInitial(initial)
	}
}

func OnValid(funcs ...func(Form)) func(Form) {
	return func(f Form) {
		f.OnValid(funcs...)
	}
}

func OnInvalid(funcs ...func(Form)) func(Form) {
	return func(f Form) {
		f.OnInvalid(funcs...)
	}
}

func OnFinalize(funcs ...func(Form)) func(Form) {
	return func(f Form) {
		f.OnFinalize(funcs...)
	}
}

func Initialize[T Form](f T, initfuncs ...func(Form)) T {

	for _, initfunc := range initfuncs {
		initfunc(f)
	}

	return f
}
