package chooser_test

import (
	"errors"
	"testing"

	"github.com/Nigel2392/go-django/src/core/contenttypes"
	"github.com/Nigel2392/go-django/src/forms/widgets/chooser"
)

func init() {
	var def = &MockContentTypeDefinition{}
	contenttypes.Register(&contenttypes.ContentTypeDefinition{
		ContentObject:     def,
		GetInstance:       def.GetInstance,
		GetInstances:      def.GetInstances,
		GetInstancesByIDs: def.GetInstancesByIDs,
	})
}

// Mock implementation of ContentTypeDefinition for testing purposes.
type MockContentTypeDefinition struct {
	contenttypes.ContentTypeDefinition
}

func (m *MockContentTypeDefinition) GetInstances(amount, offset uint) ([]interface{}, error) {
	return []interface{}{"instance1", "instance2"}, nil
}

func (m *MockContentTypeDefinition) GetInstancesByIDs(ids []interface{}) ([]interface{}, error) {
	if len(ids) == 2 && ids[0] == "id1" && ids[1] == "id2" {
		return []interface{}{"instance1", "instance2"}, nil
	}
	return nil, errors.New("some model instances not found")
}

func (m *MockContentTypeDefinition) GetInstance(id interface{}) (interface{}, error) {
	if id == "valid_id" {
		return "valid_instance", nil
	}
	return nil, errors.New("model instance not found")
}

// Test for BaseChooserWidget creation
func TestBaseChooserWidget(t *testing.T) {

	opts := chooser.BaseChooserOptions{
		TargetObject: &MockContentTypeDefinition{},
	}
	attrs := map[string]string{"class": "chooser"}

	widget := chooser.BaseChooserWidget(opts, attrs)
	if widget == nil {
		t.Fatal("BaseChooserWidget returned nil")
	}

	if widget.Opts.TargetObject != opts.TargetObject {
		t.Errorf("Expected TargetObject %v, got %v", opts.TargetObject, widget.Opts.TargetObject)
	}
}

// Test for BaseChooser QuerySet method
func TestBaseChooserQuerySet(t *testing.T) {

	chooserWidget := chooser.BaseChooserWidget(chooser.BaseChooserOptions{
		TargetObject: &MockContentTypeDefinition{},
	}, nil)

	results, err := chooserWidget.QuerySet()
	if err != nil {
		t.Fatalf("QuerySet returned error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("Expected 2 instances, got %d", len(results))
	}
}

// Test for BaseChooser Validate method with a valid ID
func TestBaseChooserValidateWithValidID(t *testing.T) {

	chooserWidget := chooser.BaseChooserWidget(chooser.BaseChooserOptions{
		TargetObject: &MockContentTypeDefinition{},
	}, nil)

	errors := chooserWidget.Validate("valid_id")
	if len(errors) != 0 {
		t.Fatalf("Expected no errors, got %d", len(errors))
	}
}

// Test for BaseChooser Validate method with an invalid ID
func TestBaseChooserValidateWithInvalidID(t *testing.T) {

	chooserWidget := chooser.BaseChooserWidget(chooser.BaseChooserOptions{
		TargetObject: &MockContentTypeDefinition{},
	}, nil)

	errors := chooserWidget.Validate("invalid_id")
	if len(errors) == 0 {
		t.Fatal("Expected errors, got none")
	}
}

// Test for BaseChooser Validate method with an array of valid IDs
func TestBaseChooserValidateWithValidIDArray(t *testing.T) {

	chooserWidget := chooser.BaseChooserWidget(chooser.BaseChooserOptions{
		TargetObject: &MockContentTypeDefinition{},
	}, nil)

	errors := chooserWidget.Validate([]interface{}{"id1", "id2"})
	if len(errors) != 0 {
		t.Fatalf("Expected no errors, got %d", len(errors))
	}
}

// Test for BaseChooser Validate method with an array of invalid IDs
func TestBaseChooserValidateWithInvalidIDArray(t *testing.T) {

	chooserWidget := chooser.BaseChooserWidget(chooser.BaseChooserOptions{
		TargetObject: &MockContentTypeDefinition{},
	}, nil)

	errors := chooserWidget.Validate([]interface{}{"invalid_id1", "invalid_id2"})
	if len(errors) == 0 {
		t.Fatal("Expected errors, got none")
	}
}
