package pages_test

import (
	"errors"
	"fmt"
	"testing"

	_ "unsafe"

	"github.com/Nigel2392/django/contrib/pages"
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
	{"001002003", 4, "", pages.ErrTooManyAncestors},
}

//go:linkname buildPathPart github.com/Nigel2392/django/contrib/pages.buildPathPart
func buildPathPart(numPreviousAncestors int) string

//go:linkname ancestorPath github.com/Nigel2392/django/contrib/pages.ancestorPath
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
					if !errors.Is(r.(error), pages.ErrTooLittleAncestors) {
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
					if !errors.Is(r.(error), pages.ErrTooManyAncestors) {
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
		t.Run(fmt.Sprintf("AncestorPath(%s, %d)", test.path, test.numAncestors), func(t *testing.T) {
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
}
