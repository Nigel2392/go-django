package fs

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Nigel2392/router/v3"
	"github.com/Nigel2392/router/v3/templates"
)

/*
EXAMPLE:

	var text = `Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.
	Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat.
	Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur.
	Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum.`

	var path, err = WriteToMedia("app/text"+time.Now().Format("20060102150405")+".txt", []byte(text))

*/

type Manager struct {
	// The root path of the static files directory.
	FS_STATIC_ROOT string
	// The root path of the media files directory.
	FS_MEDIA_ROOT string
	// The URL base to serve the static files.
	FS_STATIC_URL string
	// The URL base to serve the media files.
	FS_MEDIA_URL string
	// File system to access the static files.
	fs_STATICFILES fs.FS
	// File system to access the media files.
	fs_MEDIAFILES fs.FS
	// The queue of files to process.
	_QUEUE chan FileQueueItem
	// The size of the file queue.
	FS_FILE_QUEUE_SIZE int

	// A function to create new FileQueueItem.
	NewFileFunc func(path string, r io.Reader) FileQueueItem
	// A function to process the media files which are read from the media directory.
	OnReadFromMedia func(path string, r io.Reader)
	// A function to process the media files which are written to the media directory.
	OnWriteToMedia func(path string, r io.Reader)

	// The registrar for the static files.
	staticRegistrar router.Registrar
	// The registrar for the media files.
	mediaRegistrar router.Registrar
}

func (fm *Manager) Init() {
	fm.fs_STATICFILES = os.DirFS(fm.FS_STATIC_ROOT)
	fm.fs_MEDIAFILES = os.DirFS(fm.FS_MEDIA_ROOT)
	fm._QUEUE = make(chan FileQueueItem, fm.FS_FILE_QUEUE_SIZE)

	go fm.worker()
}

// Write a file to the media files directory.
//
// If there is a collision with the filename, it will append the current time to the filename.
//
// This is done to prevent overwriting files.
//
// We will push the file to a queue and wait for the function to return, and write to the path channel or the error channel
func (fm *Manager) WriteToMedia(path string, r io.Reader) (string, error) {
	if path == "" {
		return "", errors.New("path is empty")
	}
	path = strings.Replace(path, "\\", "/", -1)
	if len(strings.Split(path, "/")) <= 1 {
		return "", errors.New("path is invalid")
	}
	var item FileQueueItem
	if fm.NewFileFunc != nil {
		item = fm.NewFileFunc(path, r)
	} else {
		item = newFile(path, r)
	}
	fm._QUEUE <- item
	defer item.Close()
	select {
	case err := <-item.Err():
		return "", err
	case path := <-item.PathChan():
		return path, nil
	}
}

func (fm *Manager) ReadFromMedia(path string) (io.ReadCloser, error) {
	var mediaFS = fm.fs_MEDIAFILES
	if mediaFS == nil {
		return nil, errors.New("no media file system")
	}
	var nice_path = templates.NicePath(false, path)
	if !strings.HasPrefix(nice_path, templates.NicePath(false, fm.FS_MEDIA_ROOT)) {
		nice_path = templates.NicePath(false, fm.FS_MEDIA_ROOT, path)
	}

	// Check if the file exists
	file, err := mediaFS.Open(nice_path)
	if err != nil {
		return nil, err
	}

	// Hook for when reading from media.
	if fm.OnReadFromMedia != nil {
		fm.OnReadFromMedia(nice_path, file)
	}

	return file, nil
}

func (fm *Manager) worker() {
	var maxGoroutines = 6
	var guard = make(chan struct{}, maxGoroutines)
	defer close(guard)
	for {
		guard <- struct{}{}
		go func() {
			defer func() { <-guard }()
			fm.writeFile(<-fm._QUEUE)
		}()
	}
}

func (fm *Manager) writeFile(fileItem FileQueueItem) {
	var mediaFS = fm.fs_MEDIAFILES
	if mediaFS == nil {
		fileItem.Err() <- errors.New("no media file system")
		return
	}

	var nice_path = templates.NicePath(false, fm.FS_MEDIA_ROOT, fileItem.Path())
	if len(nice_path) > 255 {
		fileItem.Err() <- errors.New("path too long: " + nice_path)
		return
	}

	// Validate path
	var cleaned = strings.ReplaceAll(filepath.Clean(nice_path), "\\", "/")
	if cleaned != nice_path {
		fileItem.Err() <- errors.New("path not clean")
	}

	// Check if path is in media root
	if nice_path[:len(fm.FS_MEDIA_ROOT)] != fm.FS_MEDIA_ROOT {
		fileItem.Err() <- errors.New("path not in media root")
	}

	// Check if file exists
	if _, err := fs.Stat(mediaFS, fileItem.Path()); err == nil || (err != nil && !errors.Is(err, fs.ErrNotExist)) {
		var filename = templates.FilenameFromPath(fileItem.Path())
		var uniqueTime = strconv.FormatInt(time.Now().UnixMicro(), 10)
		filename = uniqueTime + "_" + filename
		fileItem.SetPath(templates.NicePath(false, filepath.Dir(fileItem.Path()), filename))
		fm.writeFile(fileItem)
		return
	}

	// Hook for when the file is written to the media
	if fm.OnWriteToMedia != nil {
		fm.OnWriteToMedia(nice_path, fileItem)
	}

	if err := createFile(nice_path, fileItem); err != nil {
		fileItem.Err() <- err
	} else {
		fileItem.PathChan() <- nice_path
	}
}

func createFile(path string, wr io.Reader) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	var file, err = os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	if _, err := io.Copy(file, wr); err != nil {
		return err
	}
	return nil
}
