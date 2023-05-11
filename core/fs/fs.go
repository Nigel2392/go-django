package fs

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Nigel2392/go-datastructures/stack"
)

/*
EXAMPLE:

	var text = `Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.
	Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat.
	Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur.
	Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum.`

	var path, err = WriteToMedia("app/text"+time.Now().Format("20060102150405")+".txt", []byte(text))

*/

type Filer interface {
	// Return the base path.
	Base() string
	// Initialize the filer.
	Initialize() error
	// Open a file for reading.
	Open(path string) (io.ReadCloser, error)
	Create(pathParts ...string) (file io.WriteCloser, path string, err error)
	Delete(path string) error
}

type queueItem struct {
	// The path to the file.
	path string
	// The channel to send the result to.
	result chan *fileresult
}

type fileresult struct {
	// The writer that was created.
	writer io.WriteCloser
	// The path to the file.
	//
	// This might be different from the path passed in, depending on the file existing or not.
	path string
	// The error that occurred.
	err error
}

type FileFiler struct {
	BasePath string
	queue    *stack.Stack[*queueItem]
	close    chan struct{}
	mu       *sync.Mutex
}

func NewFiler(basePath string) Filer {
	var err error
	basePath, err = filepath.Abs(basePath)
	if err != nil {
		panic(err)
	}

	basePath = filepath.Clean(basePath)

	var m = &FileFiler{BasePath: basePath}
	if err = m.Initialize(); err != nil {
		panic(err)
	}

	go m.run()

	return m
}

func (f *FileFiler) Initialize() error {
	if f.BasePath == "" {
		return errors.New("BasePath is empty")
	}
	var err = os.MkdirAll(f.BasePath, os.ModePerm)
	if err != nil {
		return err
	}
	if f.queue == nil {
		f.queue = &stack.Stack[*queueItem]{}
	}
	if f.close == nil {
		f.close = make(chan struct{})
	}
	if f.mu == nil {
		f.mu = new(sync.Mutex)
	}
	return nil
}

func (f *FileFiler) Base() string {
	return f.BasePath
}

func (f *FileFiler) run() {
	for {
		select {
		case <-f.close:
			return
		case item := <-f.queue.PopWaiter(time.Millisecond * 1): // Sleepy time!
			var err error
			var path = item.path
			var writer io.WriteCloser
		tryAgain:
			// Check if the file exists
			if _, err = os.Stat(path); os.IsNotExist(err) {
				// If not, create it
				writer, err = f.create(path)
				item.result <- &fileresult{
					writer: writer,
					path:   path,
					err:    err,
				}
			} else if err == nil || os.IsExist(err) {
				// If it does, try again with a new name
				var dir, file = filepath.Split(path)
				var now = time.Now()
				var newName = strconv.FormatInt(now.UnixMilli(), 10) + "-" + file
				path = filepath.Join(dir, newName)
				goto tryAgain
			} else {
				// If it's some other error, return it
				item.result <- &fileresult{
					err: err,
				}
			}
		}
	}
}

func (f *FileFiler) Create(pathParts ...string) (file io.WriteCloser, path string, err error) {
	path = filepath.Join(pathParts...)
	if path, err = f.joinPath(path); err != nil {
		return nil, "", err
	}
	var dir = filepath.Dir(path)
	err = os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		return nil, "", err
	}
	var item = &queueItem{
		path:   path,
		result: make(chan *fileresult),
	}
	f.mu.Lock()
	f.queue.Push(item)
	f.mu.Unlock()
	var result = <-item.result
	close(item.result)
	return result.writer, result.path, result.err
}

func (f *FileFiler) create(path string) (file io.WriteCloser, err error) {
	file, err = os.Create(path)
	if err != nil {
		return nil, err
	}

	return file, nil
}

func (f *FileFiler) Open(path string) (file io.ReadCloser, err error) {
	if path, err = f.joinPath(path); err != nil {
		return nil, err
	}
	return os.Open(path)
}

func (f *FileFiler) Delete(path string) (err error) {
	if path, err = f.joinPath(path); err != nil {
		return err
	}
	return os.Remove(path)
}

func (f *FileFiler) joinPath(path string) (string, error) {
	path = filepath.Clean(path)
	if filepath.IsAbs(path) {
		if strings.HasPrefix(
			path,
			f.BasePath,
		) {
			return path, nil
		}
		return "", errors.New("path is not in base path")
	}
	path = filepath.Join(f.BasePath, path)
	path = filepath.ToSlash(path)
	return path, nil
}
