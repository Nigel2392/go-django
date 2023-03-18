package fs

import (
	"bytes"
	"errors"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Nigel2392/go-django/core/httputils"

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
	FS_STATIC_ROOT     string
	FS_MEDIA_ROOT      string
	FS_STATIC_URL      string
	FS_MEDIA_URL       string
	fs_STATICFILES     fs.FS
	fs_MEDIAFILES      fs.FS
	_QUEUE             chan *FileQueueItem
	FS_FILE_QUEUE_SIZE int

	OnReadFromMedia func(path string, buf *bytes.Buffer)
	OnWriteToMedia  func(path string, b []byte)

	staticRegistrar router.Registrar
	mediaRegistrar  router.Registrar
}

func (fm *Manager) Init() {
	fm.fs_STATICFILES = os.DirFS(fm.FS_STATIC_ROOT)
	fm.fs_MEDIAFILES = os.DirFS(fm.FS_MEDIA_ROOT)
	fm._QUEUE = make(chan *FileQueueItem, fm.FS_FILE_QUEUE_SIZE)

	go fm.worker()
}

func (fm *Manager) AsStaticURL(path string) string {
	return templates.NicePath(false, fm.FS_STATIC_URL, path)
}

func (fm *Manager) AsMediaURL(path string) string {
	return templates.NicePath(false, fm.FS_MEDIA_URL, path)
}

type FileQueueItem struct {
	path     string
	data     []byte
	err      chan error
	pathChan chan string
}

func newFile(path string, data []byte) *FileQueueItem {
	return &FileQueueItem{
		path:     path,
		data:     data,
		err:      make(chan error, 1),
		pathChan: make(chan string, 1),
	}
}

func (fm *Manager) WriteToMedia(path string, data []byte) (string, error) {
	var item = newFile(path, data)
	fm._QUEUE <- item
	defer close(item.err)
	defer close(item.pathChan)
	select {
	case err := <-item.err:
		return "", err
	case path := <-item.pathChan:
		return path, nil
	}
}

func (fm *Manager) ReadFromMedia(path string) (*bytes.Buffer, error) {
	var mediaFS = fm.fs_MEDIAFILES
	if mediaFS == nil {
		return nil, errors.New("no media file system")
	}
	var nice_path = templates.NicePath(false, fm.FS_MEDIA_ROOT, path)

	// Check if the file exists
	file, err := mediaFS.Open(nice_path)
	if err != nil {
		return nil, err
	}

	// Read the file
	var buffer = new(bytes.Buffer)
	_, err = buffer.ReadFrom(file)
	if err != nil {
		return nil, err
	}

	// Hook for when reading from media.
	if fm.OnReadFromMedia != nil {
		fm.OnReadFromMedia(nice_path, buffer)
	}

	return buffer, nil
}

func (fm *Manager) worker() {
	var maxGoroutines = 10
	var guard = make(chan struct{}, maxGoroutines)
	for {
		guard <- struct{}{}
		go func() {
			defer func() { <-guard }()
			fm.writeFile(<-fm._QUEUE)
		}()
	}
}

func (fm *Manager) writeFile(fileItem *FileQueueItem) {
	var mediaFS = fm.fs_MEDIAFILES
	if mediaFS == nil {
		fileItem.err <- errors.New("no media file system")
		return
	}
	var nice_path = templates.NicePath(false, fm.FS_MEDIA_ROOT, fileItem.path)

	if len(nice_path) > 255 {
		fileItem.err <- errors.New("path too long: " + nice_path)
		return
	}

	// Validate path
	var cleaned = strings.ReplaceAll(filepath.Clean(nice_path), "\\", "/")
	if cleaned != nice_path {
		fileItem.err <- errors.New("path not clean")
	}

	// Check if path is in media root
	if nice_path[:len(fm.FS_MEDIA_ROOT)] != fm.FS_MEDIA_ROOT {
		fileItem.err <- errors.New("path not in media root")
	}

	// Check if file exists
	if _, err := fs.Stat(mediaFS, fileItem.path); !errors.Is(err, fs.ErrNotExist) {
		var filename = templates.FilenameFromPath(fileItem.path)
		fileItem.path = filepath.Dir(fileItem.path)
		var uniqueTime = strconv.FormatInt(time.Now().UnixMicro(), 10)
		filename = uniqueTime + "_" + filename
		fileItem.path = templates.NicePath(false, fileItem.path, filename)
		fm.writeFile(fileItem)
		return
	}

	// Hook for when the file is written to the media
	if fm.OnWriteToMedia != nil {
		fm.OnWriteToMedia(nice_path, fileItem.data)
	}

	if err := createFile(nice_path, fileItem.data); err != nil {
		fileItem.err <- err
	} else {
		fileItem.pathChan <- nice_path
	}
}

// Return the default registrars for the routes.
// If the registrars are already set, they will be returned.
func (fm *Manager) Registrars() (router.Registrar, router.Registrar) {
	var StaticRegistrar router.Registrar
	var MediaRegistrar router.Registrar

	if fm.staticRegistrar != nil || fm.mediaRegistrar != nil {
		return fm.staticRegistrar, fm.mediaRegistrar
	}

	// Register staticfiles
	if fm.fs_STATICFILES != nil {
		var static_url = templates.NicePath(false, "/", fm.FS_STATIC_URL, "/<<any>>")
		StaticRegistrar = router.Group(
			static_url, "static",
		)
		var r = StaticRegistrar.(*router.Route)
		r.Method = router.GET
		r.HandlerFunc = router.FromHTTPHandler(http.StripPrefix(
			httputils.WrapSlash(fm.FS_STATIC_URL),
			http.FileServer(
				http.FS(fm.fs_STATICFILES),
			),
		)).ServeHTTP
	}
	// Register mediafiles
	if fm.fs_MEDIAFILES != nil {
		var media_url = templates.NicePath(false, "/", fm.FS_MEDIA_URL, "/<<any>>")
		MediaRegistrar = router.Group(
			media_url, "media",
		)
		var r = MediaRegistrar.(*router.Route)
		r.Method = router.GET
		r.HandlerFunc = router.FromHTTPHandler(http.StripPrefix(
			httputils.WrapSlash(fm.FS_MEDIA_URL),
			http.FileServer(
				http.FS(fm.fs_MEDIAFILES),
			),
		)).ServeHTTP
	}

	fm.staticRegistrar = StaticRegistrar
	fm.mediaRegistrar = MediaRegistrar

	return StaticRegistrar, MediaRegistrar
}

func createFile(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	var file, err = os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.Write(data)
	return err
}
