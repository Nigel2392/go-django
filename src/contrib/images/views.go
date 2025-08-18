package images

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"path"
	"path/filepath"
	"strings"

	queries "github.com/Nigel2392/go-django/queries/src"
	autherrors "github.com/Nigel2392/go-django/src/contrib/auth/auth_errors"
	"github.com/Nigel2392/go-django/src/core/filesystem/mediafiles"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/Nigel2392/go-django/src/forms/widgets"
	"github.com/Nigel2392/go-django/src/permissions"
	"github.com/Nigel2392/mux"

	_ "unsafe"
)

var _ widgets.BaseWidget

func (a *AppConfig) serveImageFnView(fn func(*AppConfig, http.ResponseWriter, *http.Request) (*Image, error)) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		// Deny any non-GET requests
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		if a.Options.CheckServePermissions && !permissions.HasPermission(r, "images.model:serve") {
			autherrors.Fail(
				403, "You do not have permission to view images",
			)
			return
		}

		var image, err = fn(a, w, r)
		if err != nil {
			logger.Error("Error retrieving image: %s", err)
			http.Error(w, "Error retrieving image", http.StatusInternalServerError)
			return
		}

		var backend = a.MediaBackend()
		fileObj, err := backend.Open(image.Path)
		if err != nil {
			logger.Error("Error opening file object: %s", err)
			http.Error(w, "Error opening file object", http.StatusInternalServerError)
			return
		}

		file, err := fileObj.Open()
		if err != nil {
			logger.Error("Error opening file: %s", err)
			http.Error(w, "Error opening file", http.StatusInternalServerError)
			return
		}
		defer file.Close()

		stat, err := file.Stat()
		if err != nil {
			logger.Error("Error retrieving file info: %s", err)
			http.Error(w, "Error retrieving file info", http.StatusInternalServerError)
			return
		}

		modTime, err := stat.TimeModified()
		if err != nil && !errors.Is(err, mediafiles.ErrNotImplemented) {
			logger.Error("Error retrieving file modified time: %s", err)
			http.Error(w, "Error retrieving file modified time", http.StatusInternalServerError)
			return
		}

		var buf = make([]byte, stat.Size())
		_, err = file.Read(buf)
		if err != nil {
			logger.Error("Error reading file: %s", err)
			http.Error(w, "Error reading file", http.StatusInternalServerError)
			return
		}

		http.ServeContent(
			w, r, image.Title, modTime, bytes.NewReader(buf),
		)
	}
}

func (a *AppConfig) serveImageByIDView(w http.ResponseWriter, r *http.Request) {
	var fn = func(a *AppConfig, w http.ResponseWriter, r *http.Request) (*Image, error) {
		var vars = mux.Vars(r)
		var id = vars.Get("id")
		var imgRow, err = queries.GetQuerySet(&Image{}).
			WithContext(r.Context()).
			Filter("ID", id).
			Get()
		if err != nil {
			return nil, fmt.Errorf("error retrieving image: %w", err)
		}
		return imgRow.Object, nil
	}

	var view = a.serveImageFnView(fn)
	view(w, r)
}

func (a *AppConfig) serveImageByPathView(w http.ResponseWriter, r *http.Request) {
	var fn = func(a *AppConfig, w http.ResponseWriter, r *http.Request) (*Image, error) {
		var vars = mux.Vars(r)
		var pathParts = vars.GetAll("*")
		var path = path.Join(pathParts...)
		path = filepath.Clean(path)
		path = filepath.ToSlash(path)
		path = strings.TrimPrefix(path, "/")
		path = strings.TrimSuffix(path, "/")

		imgRow, err := queries.GetQuerySet[*Image](&Image{}).
			WithContext(r.Context()).
			Filter("Path", path).
			Get()
		if err != nil {
			return nil, fmt.Errorf("error retrieving image by path: %w", err)
		}
		return imgRow.Object, nil
	}

	var view = a.serveImageFnView(fn)
	view(w, r)
}
