package images

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"path"
	"path/filepath"
	"slices"

	autherrors "github.com/Nigel2392/go-django/src/contrib/auth/auth_errors"
	"github.com/Nigel2392/go-django/src/core/except"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/Nigel2392/go-django/src/permissions"
	"github.com/Nigel2392/go-django/src/utils/httputils"
	"github.com/Nigel2392/mux"
	"github.com/justinas/nosurf"
)

const err_open_generic = "Error opening file"

func uploadImage(w http.ResponseWriter, r *http.Request) {
	// Deny any non-POST requests
	if r.Method != http.MethodPost {
		var csrfToken = nosurf.Token(r)
		var jsonResp = map[string]interface{}{
			"status":     "error",
			"message":    "Method not allowed",
			"csrf_token": csrfToken,
		}

		err := json.NewEncoder(w).Encode(jsonResp)
		if err != nil {
			logger.Error("Error encoding response: %s", err)
			httputils.JSONHttpError(
				w, "Error encoding response", http.StatusInternalServerError,
			)
		}

		return
	}

	// Check if user has permission to upload images
	if !permissions.HasPermission(r, "images:upload") {
		autherrors.Fail(
			403, "You do not have permission to upload images",
		)
	}

	// Parse the multipart form data
	var maxBytes = app.MaxByteSize()
	if err := r.ParseMultipartForm(int64(maxBytes)); err != nil {
		logger.Error("Error parsing form: %s", err)
		httputils.JSONHttpError(w, "Error parsing form", http.StatusInternalServerError)
		return
	}

	// Check if file was properly uploaded and able to be parsed
	if r.MultipartForm == nil || r.MultipartForm.File == nil {
		logger.Error("No file found in request")
		httputils.JSONHttpError(w, "No file found in request", http.StatusBadRequest)
		return
	}

	// Retrieve the file header from the request
	var hdrs, ok = r.MultipartForm.File["file"]
	if !ok || len(hdrs) == 0 {
		logger.Error("Error retrieving file from request")
		httputils.JSONHttpError(w, "Error retrieving file from request", http.StatusInternalServerError)
		return
	}

	// Check if the file is too large
	var hdr = hdrs[0]
	if hdr.Size > int64(maxBytes) {
		logger.Error("File is too large: %d / %s", hdr.Size, maxBytes)
		httputils.JSONHttpError(w, "File is too large", http.StatusBadRequest)
		return
	}

	// Check if the file extension is allowed.
	// This can be spoofed.
	var allowedExtensions = app.AllowedFileExts()
	var ext = filepath.Ext(hdr.Filename)
	if len(allowedExtensions) > 0 && !slices.Contains(allowedExtensions, ext) {
		logger.Error("File extension not allowed: %s", ext)
		httputils.JSONHttpError(w, "File extension not allowed", http.StatusBadRequest)
		return
	}

	// Check if the file MIME type is allowed.
	// This can be spoofed.
	var allowedMimeTypes = app.AllowedMimeTypes()
	var mime = hdr.Header.Get("Content-Type")

	if len(allowedMimeTypes) > 0 && !slices.Contains(allowedMimeTypes, mime) {
		logger.Error("MIME type not allowed: %s", mime)
		httputils.JSONHttpError(w, "MIME type not allowed", http.StatusBadRequest)
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

	if filePath, err = backend.Save(filePath, file); err != nil {
		logger.Error("Error saving file: %s", err)
		httputils.JSONHttpError(w, "Error saving file", http.StatusInternalServerError)
		return
	}

	// Respond with the file path
	logger.Debugf("File uploaded successfully: %s", filePath)
	var jsonResp = map[string]interface{}{
		"status":   "success",
		"filePath": filePath,
	}

	err = json.NewEncoder(w).Encode(jsonResp)
	if err != nil {
		logger.Error("Error encoding response: %s", err)
		httputils.JSONHttpError(
			w, "Error encoding response", http.StatusInternalServerError,
		)
	}
}

func viewImage(w http.ResponseWriter, r *http.Request) {
	var backend = app.MediaBackend()
	var filePath = path.Join(mux.Vars(r)["*"]...)
	var fileObj, err = backend.Open(filePath)
	if err != nil {
		logger.Error("Error opening file (%s): %s", filePath, err)
		http.Error(w, err_open_generic, http.StatusInternalServerError)
		return
	}

	file, err := fileObj.Open()
	if err != nil {
		logger.Error("Error opening file object: %s", err)
		http.Error(w, err_open_generic, http.StatusInternalServerError)
		return
	}

	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		logger.Error("Error reading file stat: %s", err)
		http.Error(w, err_open_generic, http.StatusInternalServerError)
		return
	}

	modTime, err := stat.TimeModified()
	if err != nil {
		logger.Error("Error retrieving file modified time: %s", err)
		http.Error(w, err_open_generic, http.StatusInternalServerError)
		return
	}

	var buf = new(bytes.Buffer)
	if _, err := io.Copy(buf, file); err != nil {
		logger.Error("Error reading file content: %s", err)
		except.Fail(
			http.StatusInternalServerError,
			"Error reading file content",
		)
		return
	}

	var reader = bytes.NewReader(buf.Bytes())
	http.ServeContent(w, r, filePath, modTime, reader)
}
