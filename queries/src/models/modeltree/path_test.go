package modeltree_test

import (
	"errors"
	"fmt"
	"slices"
	"testing"

	_ "unsafe"

	"github.com/Nigel2392/go-django/queries/src/models/modeltree"
)

type buildPathTest struct {
	numPreviousAncestors int
	expectedPathPart     string
}

type ancestorPathTest struct {
	path         string
	numAncestors int
	expectedPath string
	expectedErr  error
}

type buildPathPartTest struct {
	pathPart         string
	expectedPathPart string
	expectedErr      error
}

type buildPathTestFromPath struct {
	path             []int64
	expectedPathPart string
	expectedErr      error
}

var buildPathTests = []buildPathTest{
	{0, "001"},
	{1, "002"},
	{9, "010"},
	{10, "011"},
	{99, "100"},
	{100, "101"},
	{998, "999"},
}

var ancestorPathTests = []ancestorPathTest{
	{"001002003", 0, "001002003", nil},
	{"001002003", 1, "001002", nil},
	{"001002003", 2, "001", nil},
	{"001002003", 3, "", nil},
	{"001002003", 4, "", modeltree.ErrTooManyAncestors},
}

var buildPathPartTests = []buildPathPartTest{
	{"001", "002", nil},
	{"002", "003", nil},
	{"099", "100", nil},
	{"100", "101", nil},
	{"998", "999", nil},
	{"999", "", modeltree.ErrTooManyAncestors},
	{"001002", "003", nil},
}

var buildPathTestsFromPath = []buildPathTestFromPath{
	{[]int64{}, "", modeltree.ErrTooLittleAncestors},
	{[]int64{0}, "001", nil}, // safe to use 0 as the first ancestor
	{[]int64{1}, "001", nil},
	{[]int64{1, 2, 3}, "001002003", nil},
	{[]int64{1, 2, 3, 4}, "001002003004", nil},
	{[]int64{1, 2, 3, 4, 5}, "001002003004005", nil},
	{[]int64{1, 2, 3, 4, 5, 6}, "001002003004005006", nil},
	{[]int64{998}, "998", nil},
	{[]int64{999}, "999", nil},
	{[]int64{1000}, "", modeltree.ErrInvalidPathLength},
}

//go:linkname buildPathPart github.com/Nigel2392/go-django/queries/src/models/modeltree.BuildPathPart
func buildPathPart(numPreviousAncestors int) string

//go:linkname ancestorPath github.com/Nigel2392/go-django/queries/src/models/modeltree.ParentPath
func ancestorPath(path string, numAncestors int) (string, error)

func TestPath(t *testing.T) {
	for _, test := range buildPathTests {
		t.Run(fmt.Sprintf("BuildPathPart(%d)", test.numPreviousAncestors), func(t *testing.T) {
			actualPathPart := buildPathPart(test.numPreviousAncestors)
			if actualPathPart != test.expectedPathPart {
				t.Errorf("Expected %s, got %s", test.expectedPathPart, actualPathPart)
			}
		})
	}

	t.Run("BuildPathPartPanic", func(t *testing.T) {
		t.Run("NegativeNumPreviousAncestors", func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("Expected a panic")
				} else {
					if !errors.Is(r.(error), modeltree.ErrTooLittleAncestors) {
						t.Errorf("Expected ErrTooLittleAncestors, got %v", r)
					} else {
						t.Logf("Recovered from panic: %v", r)
					}
				}
			}()
			buildPathPart(-1)
		})

		t.Run("NumPreviousAncestorsTooLarge", func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("Expected a panic")
				} else {
					if !errors.Is(r.(error), modeltree.ErrTooManyAncestors) {
						t.Errorf("Expected ErrTooManyAncestors, got %v", r)
					} else {
						t.Logf("Recovered from panic: %v", r)
					}
				}
			}()
			buildPathPart(999)
		})
	})

	for _, test := range ancestorPathTests {
		t.Run(fmt.Sprintf("AncestorPath(%s-%d)", test.path, test.numAncestors), func(t *testing.T) {
			var actualPath string
			var err error
			actualPath, err = ancestorPath(test.path, test.numAncestors)
			if test.expectedErr != nil {
				if !errors.Is(err, test.expectedErr) {
					t.Errorf("Expected error %v, got %v", test.expectedErr, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
				if actualPath != test.expectedPath {
					t.Errorf("Expected %s, got %s", test.expectedPath, actualPath)
				}
			}
		})
	}

	for _, test := range buildPathPartTests {
		t.Run(fmt.Sprintf("BuildPathPartFromString(%s)", test.pathPart), func(t *testing.T) {
			actualPathPart, err := modeltree.BuildNextPathPartFromFullPath(test.pathPart)
			if test.expectedErr != nil {
				if !errors.Is(err, test.expectedErr) {
					t.Errorf("Expected error %v, got %v", test.expectedErr, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
				if actualPathPart != test.expectedPathPart {
					t.Errorf("Expected %s, got %s", test.expectedPathPart, actualPathPart)
				}
			}
		})
	}

	for _, test := range buildPathTestsFromPath {
		t.Run(fmt.Sprintf("BuildPathPartFromPath(%v)", test.path), func(t *testing.T) {
			actualPathPart, err := modeltree.BuildPath(test.path)
			if test.expectedErr != nil {
				if !errors.Is(err, test.expectedErr) {
					t.Errorf("Expected error %v, got %v", test.expectedErr, err)
				}
				return
			}

			if err != nil {
				t.Errorf("Expected no error, got %v", err)
			}

			if actualPathPart != test.expectedPathPart {
				t.Errorf("Expected %s, got %s", test.expectedPathPart, actualPathPart)
			}

			parsed, err := modeltree.ParsePath(actualPathPart)
			if err != nil {
				t.Errorf("Failed to parse path %s: %v", actualPathPart, err)
			}

			var path = slices.Clone(test.path)
			if len(path) == 1 && path[0] == 0 {
				path[0]++ // adjust for the special case where the first ancestor is 0
			}

			if !slices.Equal(parsed, path) {
				t.Errorf("Expected %v, got %v", path, parsed)
			}
		})
	}
}
