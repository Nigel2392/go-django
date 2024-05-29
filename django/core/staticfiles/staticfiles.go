package staticfiles

import (
	"fmt"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"path/filepath"

	"github.com/Nigel2392/django/core/tpl"
)

var Handler = NewFileHandler()

func AddFS(filesys fs.FS, matches func(path string) bool) {
	Handler.AddFS(filesys, matches)
}

func Collect(fn func(fs.File) error) error {
	return Handler.Collect(fn)
}

func Open(name string) (fs.File, error) {
	return Handler.Open(name)
}

type FileHandler struct {
	fs *tpl.MultiFS
}

func NewFileHandler() *FileHandler {
	return &FileHandler{
		fs: tpl.NewMultiFS(),
	}
}

func (h *FileHandler) AddFS(filesys fs.FS, matches func(path string) bool) {
	h.fs.Add(filesys, matches)
}

func (h *FileHandler) Collect(fn func(fs.File) error) error {
	var files, err = fs.ReadDir(h.fs, ".")
	if err != nil {
		return err
	}

	for _, file := range files {
		var f, err = h.Open(file.Name())
		if err != nil {
			return err
		}
		defer f.Close()
		if err = fn(f); err != nil {
			return err
		}
	}

	return nil
}

func (h *FileHandler) Open(name string) (fs.File, error) {
	return h.fs.Open(name)
}

func (h *FileHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var path = r.URL.Path
	var file, err = h.fs.Open(path)
	if err != nil {
		http.Error(w, err.Error(), 404)
		return
	}

	stat, err := file.Stat()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	switch file := file.(type) {
	case io.ReadSeeker:
		http.ServeContent(w, r, stat.Name(), stat.ModTime(), file)
	case io.ReaderAt:
		var reader = io.NewSectionReader(file, 0, stat.Size())
		http.ServeContent(w, r, stat.Name(), stat.ModTime(), reader)
	default:
		w.Header().Set("Content-Length", fmt.Sprint(stat.Size()))
		w.Header().Set("Content-Type", mime.TypeByExtension(
			filepath.Ext(stat.Name()),
		))
		_, err = io.Copy(w, file)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
	}
}
