package filesystem_test

import (
	"embed"
	"errors"
	"io/fs"
	"path/filepath"
	"slices"
	"testing"

	"github.com/Nigel2392/django/core/filesystem"
)

//go:embed test/fs1/**
var FS1 embed.FS

//go:embed test/fs2/**
var FS2 embed.FS

type testFile struct {
	Name     string
	Expected string
	Error    error
}

type testMatchFS struct {
	name      string
	fs        embed.FS
	matchFunc func(filepath string) bool
	files     []testFile
}

func newTestMatchFS(name string, fs embed.FS, match func(filepath string) bool, files ...testFile) *testMatchFS {
	return &testMatchFS{
		name:      name,
		fs:        fs,
		files:     files,
		matchFunc: match,
	}
}

func TestMatchFS(t *testing.T) {

	var matchFsFunc1 = filesystem.MatchAnd(
		filesystem.MatchPrefix("test/fs1"),
		filesystem.MatchOr(
			filesystem.MatchExt(".txt"),
			filesystem.MatchExt(".text"),
			filesystem.MatchExt(".md"),
		),
	)
	var allFilesFS1 = []testFile{
		{
			Name:     "test/fs1/test1.txt",
			Expected: "fs1-test1",
		},
		{
			Name:     "test/fs1/test2.text",
			Expected: "fs1-test2",
		},
		{
			Name:     "test/fs1/test3.md",
			Expected: "# fs1-test3",
		},
		{
			Name:  "test/fs1/test4.markdown",
			Error: fs.ErrNotExist,
		},
		{
			Name:  "test/fs2/test1.txt",
			Error: fs.ErrNotExist,
		},
		{
			Name:  "test/fs2/test2.text",
			Error: fs.ErrNotExist,
		},
		{
			Name:  "test/fs2/test3.md",
			Error: fs.ErrNotExist,
		},
	}

	var matchFsFunc2 = filesystem.MatchAnd(
		filesystem.MatchPrefix("test/fs2"),
		filesystem.MatchOr(
			filesystem.MatchExt(".txt"),
			filesystem.MatchExt(".md"),
			filesystem.MatchExt(".markdown"),
		),
	)
	var allFilesFS2 = []testFile{
		{
			Name:     "test/fs2/test1.txt",
			Expected: "fs2-test1",
		},
		{
			Name:  "test/fs2/test2.text",
			Error: fs.ErrNotExist,
		},
		{
			Name:     "test/fs2/test3.md",
			Expected: "# fs2-test3",
		},
		{
			Name:     "test/fs2/test4.markdown",
			Expected: "# fs2-test4",
		},
		{
			Name:  "test/fs1/test1.txt",
			Error: fs.ErrNotExist,
		},
		{
			Name:  "test/fs1/test2.text",
			Error: fs.ErrNotExist,
		},
		{
			Name:  "test/fs1/test3.md",
			Error: fs.ErrNotExist,
		},
	}

	var tests = []*testMatchFS{
		newTestMatchFS("FS1", FS1, matchFsFunc1, allFilesFS1...),
		newTestMatchFS("FS2", FS2, matchFsFunc2, allFilesFS2...),
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			slices.SortFunc(tt.files, func(i, j testFile) int {
				if i.Error != nil && j.Error == nil {
					return 1
				} else if i.Error == nil && j.Error != nil {
					return -1
				}
				return 0
			})

			for _, file := range tt.files {

				var fileSys = filesystem.NewMatchFS(tt.fs, tt.matchFunc)

				var name = filepath.Base(file.Name)
				if file.Error != nil {
					name = "(error)-" + name
				}

				t.Run(name, func(t *testing.T) {
					data, err := fs.ReadFile(fileSys, file.Name)

					if file.Error != nil {
						if err == nil {
							t.Errorf("expected error, got nil")
						}

						if !errors.Is(err, file.Error) {
							t.Errorf("expected error %v, got %v", file.Error, err)
						}
						return
					} else if err != nil {
						t.Errorf("unexpected error: %v", err)
						return
					}

					if string(data) != file.Expected {
						t.Errorf("expected %q, got %q", file.Expected, string(data))
					}
				})
			}
		})
	}
}
