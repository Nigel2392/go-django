package mhttp

import (
	"bytes"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"path/filepath"
	"slices"
	"time"

	"github.com/Nigel2392/go-django/src/core/filesystem/mediafiles"
	"github.com/Nigel2392/go-django/src/core/logger"
)

// Serve creates a new mediafiles endpoint
func Serve(engine string) http.Handler {
	backend, ok := mediafiles.RetrieveBackend(engine)
	if !ok {
		panic(fmt.Sprintf("Invalid backend %q not found in %v", engine, slices.Collect(maps.Keys(mediafiles.MapRegistry()))))
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fileObj, err := backend.Open(r.URL.Path)
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

		if err != nil {
			modTime = time.Now()
		}

		var buf = make([]byte, stat.Size())
		_, err = file.Read(buf)
		if err != nil {
			logger.Error("Error reading file: %s", err)
			http.Error(w, "Error reading file", http.StatusInternalServerError)
			return
		}

		http.ServeContent(
			w, r, filepath.Base(r.URL.Path), modTime, bytes.NewReader(buf),
		)
	})
}
