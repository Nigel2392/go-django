package models

import (
	"reflect"
)

// ModelState represents the state of a model instance,
// tracking which fields have been changed since the last reset.
//
// A nil *ModelState is considered a valid state,
// meaning that the model has no state to track, the model is
// considered new and fully 'changed'.
type ModelState struct {
	// model is a pointer back to the model instance
	// which this state belongs to.
	model *Model

	// changed fields, used to track which fields have been changed
	changed map[string]struct{}

	// initial is a map of initial values for the model's fields
	//
	// These are the primitive values of the fields
	// which are retrieved from the fields Value() method,
	// which normally returns the database compatible value
	initial map[string]interface{}
}

// initState initializes the model state for the given model.
func initState(model *Model) *ModelState {
	if model == nil {
		return nil
	}

	var state = &ModelState{
		model: model,
	}

	state.Reset()

	return state
}

// checkState checks if the model's state is changed,
// and if so, it updates the model's state to reflect the changes.
func (m *ModelState) checkState() {
	if m == nil {
		return
	}
	if m.model == nil {
		panic("model state is not properly initialized: model is nil")
	}

	// if the model's state is not changed, we can skip the update
	for head := m.model.internals.defs.ObjectFields.Front(); head != nil; head = head.Next() {
		var (
			field = head.Value
			name  = field.Name()
			value = field.GetValue()
		)

		// if the value is not equal to the initial value,
		// we need to mark the field as changed
		var initialValue, ok = m.initial[name]
		if !ok || !reflect.DeepEqual(value, initialValue) {
			m.changed[name] = struct{}{}
		} else {
			// if the value is equal to the initial value,
			// we need to remove the field from the changed map
			delete(m.changed, name)
		}
	}
}

// change marks a field as changed in the model's state.
func (m *ModelState) change(fieldName string) {
	if m == nil {
		return
	}
	m.changed[fieldName] = struct{}{}
}

// Changed returns true if the model's state has changed,
// meaning that at least one field has been modified
// since the last time the state was checked or reset.
//
// If checkState is true, it will first check the state
// to ensure that the changed fields are up to date.
func (m *ModelState) Changed(checkState bool) bool {
	if m == nil {
		return true
	}

	if checkState {
		m.checkState()
	}

	return len(m.changed) > 0
}

// HasChanged checks if a specific field has been changed
// in the model's state.
func (m *ModelState) HasChanged(fieldName string) bool {
	if m == nil {
		return true
	}

	if m.model == nil {
		panic("model state is not properly initialized: model is nil")
	}

	if _, ok := m.changed[fieldName]; ok {
		return true
	}

	// if the field is not in the changed map,
	// we need to check if it has an initial value
	if initialValue, ok := m.initial[fieldName]; ok {
		var field, ok = m.model.internals.defs.Field(fieldName)
		if !ok {
			return false // field does not exist in the model
		}

		var value = field.GetValue()
		return !reflect.DeepEqual(value, initialValue)
	}

	return false
}

// InitialValue returns the initial value of a field in the model's state.
func (m *ModelState) InitialValue(fieldName string) (interface{}, bool) {
	if m == nil {
		return nil, false
	}

	if m.model == nil {
		panic("model state is not properly initialized: model is nil")
	}

	initialValue, ok := m.initial[fieldName]
	if !ok {
		return nil, false // field does not have an initial value
	}

	return initialValue, true
}

// Reset clears the changed fields and initial values and
// reinitializes the initial values from the model's definitions.
// This is useful when the model is saved or reset.
func (m *ModelState) Reset() {
	if m == nil {
		return
	}

	if m.model == nil {
		panic("model state is not properly initialized: model is nil")
	}

	m.changed = make(map[string]struct{})
	m.initial = make(map[string]interface{})

	if m.model.internals != nil && m.model.internals.defs != nil {
		for head := m.model.internals.defs.ObjectFields.Front(); head != nil; head = head.Next() {
			m.initial[head.Value.Name()] = head.Value.GetValue()
		}
	}
}
