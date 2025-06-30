package yml_test

import (
	_ "embed"
	"fmt"
	"strings"
	"testing"

	"github.com/Nigel2392/go-django/pkg/yml"
)

//	name: abc
//	number: 123
//	active: false
//	items:
//	  - name: item1
//	    number: 1
//	    active: true
//
//	  - name: item2
//	    number: 2
//	    active: false
//
//	  - name: item3
//	    number: 3
//	    active: true

//go:embed test1.yml
var testFile1 string

//go:embed test2.yml
var testFile2 string

var testFiles = [...]string{
	testFile1,
	testFile2,
}

var testFilePaths = [...]string{
	"test1.yml",
	"test2.yml",
}

type TestObject struct {
	Name   string                              `yaml:"name"`
	Number int                                 `yaml:"number"`
	Active bool                                `yaml:"active"`
	Items  yml.OrderedMap[string, *TestObject] `yaml:"items"`
}

func testObject(t *testing.T, obj *TestObject) {
	if obj.Name != "abc" {
		t.Errorf("unexpected name: %q", obj.Name)
	}

	if obj.Number != 123 {
		t.Errorf("unexpected number: %d", obj.Number)
	}

	if obj.Active != false {
		t.Errorf("unexpected active: %v", obj.Active)
	}

	if obj.Items.Len() != 3 {
		t.Errorf("unexpected number of items: %d", obj.Items.Len())
	}

	var (
		item1, ok1 = obj.Items.Get("item1")
		item2, ok2 = obj.Items.Get("item2")
		item3, ok3 = obj.Items.Get("item3")
	)

	if !ok1 || item1 == nil {
		t.Errorf("item1 not found or nil")
	}

	if item1.Number != 1 || item1.Active != true {
		t.Errorf("unexpected item1: %+v", item1)
	}

	if !ok2 || item2 == nil {
		t.Errorf("item2 not found or nil")
	}

	if item2.Number != 2 || item2.Active != false {
		t.Errorf("unexpected item2: %+v", item2)
	}

	if !ok3 || item3 == nil {
		t.Errorf("item3 not found or nil")
	}

	if item3.Number != 3 || item3.Active != true {
		t.Errorf("unexpected item3: %+v", item3)
	}
}

func TestUnmarshal(t *testing.T) {

	for i, testFile := range testFilePaths {
		t.Run(fmt.Sprintf("UnmarshalFile-%d", i), func(t *testing.T) {
			var obj TestObject
			if err := yml.Unmarshal(testFile, &obj, true); err != nil {
				t.Fatalf("failed to unmarshal file: %v", err)
			}

			testObject(t, &obj)
		})
	}

	for i, testFile := range testFiles {
		t.Run(fmt.Sprintf("UnmarshalReader-%d", i), func(t *testing.T) {
			var obj TestObject
			if err := yml.UnmarshalReader(strings.NewReader(testFile), &obj, true); err != nil {
				t.Fatalf("failed to unmarshal reader: %v", err)
			}

			testObject(t, &obj)
		})

	}

}
