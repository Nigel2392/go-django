package staticfiles

import (
	"fmt"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"path/filepath"

	"github.com/Nigel2392/django/core/tpl"
	"github.com/pkg/errors"
)

var EntryHandler = NewFileHandler()

func AddFS(filesys fs.FS, matches func(path string) bool) {
	EntryHandler.AddFS(filesys, matches)
}

func Collect(fn func(path string, f fs.File) error) error {
	return EntryHandler.Collect(fn)
}

func Open(name string) (fs.File, error) {
	return EntryHandler.Open(name)
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

func (h *FileHandler) Collect(fn func(path string, f fs.File) error) error {
	var filesystems = make([]fs.FS, 0)
	for _, fs := range h.fs.FS() {
		switch fs := fs.(type) {
		case *tpl.MatchFS:
			filesystems = append(filesystems, fs.FS())
		case *tpl.MultiFS:
			filesystems = append(filesystems, fs.FS()...)
		default:
			filesystems = append(filesystems, fs)
		}
	}
	var err error
	for _, fSys := range filesystems {
		err = fs.WalkDir(fSys, ".", func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return errors.Wrapf(err, "failed to walk directory '%s'", path)
			}
			if !d.IsDir() {
				var f fs.File
				f, err = fSys.Open(path)
				if err != nil {
					return errors.Wrapf(err, "failed to open file '%s'", path)
				}
				defer f.Close()

				return fn(path, f)
			}
			return nil
		})
		if err != nil {
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
		var name = stat.Name()
		var _, hasCtype = w.Header()["Content-Type"]
		var ctype = mime.TypeByExtension(filepath.Ext(name))
		if ctype != "" && !hasCtype {
			w.Header().Set("Content-Type", ctype)
		}
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
