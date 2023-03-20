package fs

import "io"

// FileQueueItem is the interface for a file queue item.
//
// This is used to write files to the media file system.
//
// You can override the Manager.NewFileFunc to use your own implementation.
//
// Some implementation details are written along with the interface.
type FileQueueItem interface {
	// Path returns the path of the file.
	Path() string

	// SetPath sets the path of the file.
	SetPath(path string)

	// Read from the underlying reader.
	Read(p []byte) (n int, err error)

	// Err returns the error channel.
	//
	// This channel is used to send errors to the caller.
	//
	// We will wait on this channel until the file is written.
	//
	// An error will be returned, from this channel.
	// This means:
	//
	// - The caller should not read from this channel.
	//   - We will read from the channel and return the error to the caller.
	// - The channels should be closed by the caller.
	//   - We will call the Close method.
	Err() chan error

	// PathChan returns the path channel.
	//
	// This channel is used to send the path of the file to the caller after it has been written.
	//
	// We will wait on this channel until the file is written.
	//
	// A path will be returned, from this channel.
	// This means:
	//
	// - The caller should not read from this channel.
	//   - We will read from the channel and return the path to the caller.
	// - The channels should be closed by the caller.
	//   - We will call the Close method.
	PathChan() chan string

	// Close closes the channels.
	// This is called automatically by the caller.
	//
	// You should not call this method.
	Close()
}

type fileQueueItem struct {
	path     string
	r        io.Reader
	err      chan error
	pathChan chan string
}

func newFile(path string, r io.Reader) FileQueueItem {
	return &fileQueueItem{
		path:     path,
		r:        r,
		err:      make(chan error, 1),
		pathChan: make(chan string, 1),
	}
}

func (f *fileQueueItem) Path() string {
	return f.path
}

func (f *fileQueueItem) SetPath(path string) {
	f.path = path
}

func (f *fileQueueItem) Read(p []byte) (n int, err error) {
	return f.r.Read(p)
}

func (f *fileQueueItem) Err() chan error {
	return f.err
}

func (f *fileQueueItem) PathChan() chan string {
	return f.pathChan
}

func (f *fileQueueItem) Close() {
	close(f.err)
	close(f.pathChan)
}
