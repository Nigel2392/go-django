package images

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	queries "github.com/Nigel2392/go-django/queries/src"
	autherrors "github.com/Nigel2392/go-django/src/contrib/auth/auth_errors"
	"github.com/Nigel2392/go-django/src/core/filesystem/mediafiles"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/Nigel2392/go-django/src/forms/widgets"
	"github.com/Nigel2392/go-django/src/permissions"
	"github.com/Nigel2392/mux"
	"github.com/justinas/nosurf"

	_ "unsafe"
)

var _ widgets.BaseWidget

//go:linkname httpError github.com/Nigel2392/go-django/src/contrib/editor/features/images.httpError
func httpError(w http.ResponseWriter, message string, code int)

func methodNotAllowed(w http.ResponseWriter, extra map[string]interface{}) {
	var jsonResp = map[string]interface{}{
		"status":  "error",
		"message": "Method not allowed",
	}

	if extra != nil {
		maps.Copy(jsonResp, extra)
	}

	err := json.NewEncoder(w).Encode(jsonResp)
	if err != nil {
		logger.Error("Error encoding response: %s", err)
		httpError(
			w, "Error encoding response", http.StatusInternalServerError,
		)
	}
}

func (a *AppConfig) serveImageFnView(fn func(*AppConfig, http.ResponseWriter, *http.Request) (*Image, error)) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		// Deny any non-GET requests
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		if !permissions.HasPermission(r, "images.model:serve") {
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
		var idStr = r.FormValue("id")
		if idStr == "" {
			return nil, errors.New("no image ID provided")
		}

		var imgRow, err = queries.GetQuerySet[*Image](&Image{}).
			WithContext(r.Context()).
			Filter("ID", idStr).
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

func (a *AppConfig) serveImageUpload(w http.ResponseWriter, r *http.Request) {
	// Deny any non-POST requests
	if r.Method != http.MethodPost {
		var csrfToken = nosurf.Token(r)
		methodNotAllowed(w, map[string]interface{}{
			"csrfToken": csrfToken,
		})
		return
	}

	// Check if user has permission to upload images
	if !permissions.HasPermission(r, "images.model:upload") {
		autherrors.Fail(
			403, "You do not have permission to upload images",
		)
		return
	}

	// Parse the multipart form data
	var maxBytes = app.MaxByteSize()
	if err := r.ParseMultipartForm(int64(maxBytes)); err != nil {
		logger.Error("Error parsing form: %s", err)
		httpError(w, "Error parsing form", http.StatusInternalServerError)
		return
	}

	// Check if file was properly uploaded and able to be parsed
	if r.MultipartForm == nil || r.MultipartForm.File == nil {
		logger.Error("No file found in request")
		httpError(w, "No file found in request", http.StatusBadRequest)
		return
	}

	// Retrieve the file header from the request
	var hdrs, ok = r.MultipartForm.File["file"]
	if !ok || len(hdrs) == 0 {
		logger.Error("Error retrieving file from request")
		httpError(w, "Error retrieving file from request", http.StatusInternalServerError)
		return
	}

	// Check if the file is too large
	var hdr = hdrs[0]
	if hdr.Size > int64(maxBytes) {
		logger.Error("File is too large: %d / %s", hdr.Size, maxBytes)
		httpError(w, "File is too large", http.StatusBadRequest)
		return
	}

	// Check if the file extension is allowed.
	// This can be spoofed.
	var allowedExtensions = app.AllowedFileExts()
	var ext = filepath.Ext(hdr.Filename)
	if len(allowedExtensions) > 0 && !slices.Contains(allowedExtensions, ext) {
		logger.Error("File extension not allowed: %s", ext)
		httpError(w, "File extension not allowed", http.StatusBadRequest)
		return
	}

	// Check if the file MIME type is allowed.
	// This can be spoofed.
	var allowedMimeTypes = app.AllowedMimeTypes()
	var mime = hdr.Header.Get("Content-Type")

	if len(allowedMimeTypes) > 0 && !slices.Contains(allowedMimeTypes, mime) {
		logger.Error("MIME type not allowed: %s", mime)
		httpError(w, "MIME type not allowed", http.StatusBadRequest)
		return
	}

	// Open the file and read its contents
	var file, err = hdr.Open()
	defer file.Close()

	// Save the file to the media backend
	var backend = app.MediaBackend()
	var mediaDir = app.MediaDir()
	var filePath = filepath.Join(
		mediaDir,
		hdr.Filename,
	)

	var hasher = newImageHasher()
	var teeRd = io.TeeReader(file, hasher)

	if filePath, err = backend.Save(filePath, teeRd); err != nil {
		logger.Error("Error saving file: %s", err)
		httpError(w, "Error saving file", http.StatusInternalServerError)
		return
	}

	var hashStr = fmt.Sprintf(
		"%x", hasher.Sum(nil),
	)
	var image = &Image{
		Title: hdr.Filename,
		Path:  filePath,
		FileSize: sql.NullInt32{
			Int32: int32(hdr.Size),
			Valid: true,
		},
		FileHash: hashStr,
	}

	err = image.Save(r.Context())
	if err != nil {
		logger.Error("Error saving image: %s", err)
		httpError(w, "Error saving image", http.StatusInternalServerError)
		return
	}

	logger.Infof(
		"Image %q with ID \"%d\" (%s) saved successfully",
		image.Title, image.ID, image.Path,
	)

	// Respond with the file path
	var jsonResp = map[string]interface{}{
		"status":   "success",
		"filePath": filePath,
		"id":       image.ID,
	}

	err = json.NewEncoder(w).Encode(jsonResp)
	if err != nil {
		logger.Error("Error encoding response: %s", err)
		httpError(
			w, "Error encoding response", http.StatusInternalServerError,
		)
	}
}

func (a *AppConfig) serveImageList(w http.ResponseWriter, r *http.Request) {
	// Deny any non-GET requests
	if r.Method != http.MethodGet {
		methodNotAllowed(w, nil)
		return
	}

	if !permissions.HasPermission(r, "images.model:view") {
		autherrors.Fail(
			403, "You do not have permission to view images",
		)
		return
	}

	//var queries = NewQueryset(app.DB)
	//var images, err = queries.SelectBasic(
	//	r.Context(), 10, 0,
	//)
	var rowCount, rowIter, err = queries.GetQuerySet[*Image](&Image{}).
		WithContext(r.Context()).
		Limit(10).
		Offset(0).
		IterAll()
	if err != nil {
		logger.Error("Error retrieving images: %s", err)
		httpError(w, "Error retrieving images", http.StatusInternalServerError)
		return
	}

	var images = make([]*Image, rowCount)
	var i int
	for row, err := range rowIter {
		if err != nil {
			logger.Error("Error iterating over images: %s", err)
			httpError(w, "Error iterating over images", http.StatusInternalServerError)
			return
		}

		images[i] = row.Object
		i++
	}

	var jsonResp = map[string]interface{}{
		"status": "success",
		"images": images,
	}

	err = json.NewEncoder(w).Encode(jsonResp)
	if err != nil {
		logger.Error("Error encoding response: %s", err)
		httpError(
			w, "Error encoding response", http.StatusInternalServerError,
		)
	}
}

func (a *AppConfig) serveImageDeletion(w http.ResponseWriter, r *http.Request) {
	// Deny any non-POST requests
	if r.Method != http.MethodPost {
		methodNotAllowed(w, nil)
		return
	}

	if !permissions.HasPermission(r, "images.model:delete") {
		autherrors.Fail(
			403, "You do not have permission to delete images",
		)
		return
	}

	var idStr = r.FormValue("id")
	if idStr == "" {
		httpError(w, "No image ID provided", http.StatusBadRequest)
		return
	}

	var id, err = strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		httpError(w, "Invalid image ID", http.StatusBadRequest)
		return
	}

	_, err = queries.GetQuerySet[*Image](&Image{}).
		WithContext(r.Context()).
		Filter("ID", id).
		Delete()
	if err != nil {
		logger.Error("Error deleting image: %s", err)
		httpError(w, "Error deleting image", http.StatusInternalServerError)
		return
	}

	var jsonResp = map[string]interface{}{
		"status": "success",
		"id":     id,
		"message": fmt.Sprintf(
			"Image with ID \"%d\" deleted successfully", id,
		),
	}

	logger.Infof(
		"Image with ID \"%d\" deleted successfully", id,
	)

	err = json.NewEncoder(w).Encode(jsonResp)
	if err != nil {
		logger.Error("Error encoding response: %s", err)
		httpError(
			w, "Error encoding response", http.StatusInternalServerError,
		)
	}
}
