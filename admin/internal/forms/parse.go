package forms

import (
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"reflect"
	"strconv"
	"time"

	"github.com/Nigel2392/go-django/auth"
	"github.com/Nigel2392/go-django/core"
	"github.com/Nigel2392/go-django/core/fs"
	"github.com/Nigel2392/go-django/core/models/modelutils"
	"github.com/Nigel2392/router/v3/request"
	"gorm.io/gorm"
)

// 'submit' the form.
// This processes the form fields and might update some model fields.
func (f *Form) submit(kv map[string][]string, mgr *fs.Manager, db *gorm.DB, rq *request.Request) (any, error) {
	var t = reflect.TypeOf(f.Model)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	var reflectValue = reflect.ValueOf(f.Model)
	if reflectValue.Kind() == reflect.Ptr {
		reflectValue = reflectValue.Elem()
	}

	var newModel any
	var id = modelutils.GetID(f.Model, "ID")
	if !id.IsZero() {
		newModel = reflect.New(t).Interface()
		if err := db.First(newModel, id).Error; err != nil {
			return nil, errors.New("could not find model: " + err.Error())
		}
	}

	for i := 0; i < t.NumField(); i++ {
		// Setup getting the field
		var field = t.Field(i)
		var modelField = reflectValue.Field(i)
		var adminTag = field.Tag.Get("admin")
		if adminTag == "-" {
			continue
		}
		var formField = f.getField(field.Name)
		if formField == nil {
			continue
		}

		if formField.Required && len(kv[field.Name]) == 0 {
			//lint:ignore ST1005 We want to use the field name here.
			return nil, fmt.Errorf("Field %s is required", field.Name)
		}

		// Validate if the field is readonly, disabled or hidden.
		if (formField.ReadOnly || formField.Disabled || (formField.NeedsAdmin && !rq.User.IsAdmin())) || formField.Type == "hidden" {
			continue
		}

		// Check if the field is empty.
		// If it is not, and readonlyfull or disabledfull is set, skip the field.
		if !isNoneField(f.Model, field.Name) {
			if formField.readonlyfull || formField.disabledfull {
				continue
			}
		}

		// Get the value from the map
		var value = kv[field.Name]

		// Check if we need to hash the value
		//
		// This is only done if the field is not empty.
		//
		// If the field is empty, we don't want to hash the empty string.
		//
		// If the fieldvalue is the same as the model's previous value, we don't want to hash it.
		if formField.bcrypt && newModel == nil ||
			formField.bcrypt && !fieldValueEqual(newModel, field.Name, value[0]) {
			if len(value) == 0 {
				continue
			}
			var hash, err = auth.BcryptHash(value[0])
			if err != nil {
				return nil, errors.New("failed to hash field: " + err.Error())
			}
			value[0] = string(hash)
		}

		// Set the value according to the field's type.
		//
		// This will manage setting M2M/FK relations, checkboxes, selects, etc.
		switch formField.Type {
		case "checkbox":
			var checked = len(value) > 0
			// Check if the struct field implements .FieldValidator
			var err = validateModelField(modelField, checked)
			if err != nil {
				return nil, errors.New("failed to validate field: " + err.Error())
			}
			modelField.SetBool(checked)
		case "select":
			var v any
			// If the select isnt a model, check if it implements FromStringer
			//
			// If it does, call FromString.
			if !modelutils.IsModel(modelField.Type()) {
				var fromStringer, ok = modelField.Interface().(core.FromStringer)
				if ok {
					if err := fromStringer.FromString(value[0]); err != nil {
						return nil, errors.New("failed to parse field, are you sure it is formatted correctly?")
					}
					v = fromStringer
				} else {
					// Otherwise, parse the value.
					//
					// This will try to parse the value to the correct type.
					var err error
					v, err = modelutils.ParseValue(value[0], modelField.Type())
					if err != nil {
						return nil, errors.New("failed to parse field: " + err.Error())
					}
				}
			} else {
				// If the select is a model, find the model and set it.
				var selected any
				if formField.Multiple {
					selected = value
				} else {
					selected = value[0]
				}
				v = reflect.New(field.Type.Elem()).Interface()
				if err := db.First(v, selected).Error; err != nil {
					return nil, errors.New("failed to find selected value: " + err.Error())

				}
			}

			// Validate the field
			var err = validateModelField(modelField, v)
			if err != nil {
				return nil, errors.New("failed to validate field: " + err.Error())
			}

			// Set the field
			correctPtrSet(modelField, v)
		case "m2m":
			// If the field is a m2m, the field is a model.
			//
			// We need to find the selected models and set them.
			var selected = value
			if len(selected) == 1 && selected[0] == "-" && newModel != nil {
				// Clear the relation
				if err := db.Model(f.Model).Association(field.Name).Clear(); err != nil {
					return nil, errors.New("failed to clear relation: " + err.Error())
				}
				continue
			}

			// Create a slice of the model type
			var selectedValues = reflect.New(reflect.SliceOf(field.Type.Elem())).Interface()
			if err := db.Where("id IN (?)", selected).Find(selectedValues).Error; err != nil {
				return nil, errors.New("failed to find selected values: " + err.Error())
			}
			// If the model is new, we cannot replace the relation just yet.
			//
			// The relations will be saved later in the Save() function.
			if newModel != nil {
				var tx = db.Model(f.Model)

				var id = modelutils.GetID(newModel, "ID")
				tx = tx.Where(id)

				var err = tx.Association(field.Name).Replace(selectedValues)
				if err != nil {
					return nil, errors.New("failed to replace relation: " + err.Error())
				}
			}

			// Validate the field
			var err = validateModelField(modelField, selectedValues)
			if err != nil {
				return nil, errors.New("failed to validate field: " + err.Error())
			}

			// Set the field
			correctPtrSet(modelField, selectedValues)
		case "fk":
			// If the field is a fk, the field is a model.
			var selected = value[0]

			// Check if the supplied ID is valid
			// If it is not, skip the field.
			if modelutils.ID(selected).IsZero() {
				if formField.Required {
					return nil, errors.New("failed to parse field: " + field.Name + " is required")
				}
				continue
			}

			// Find the model
			var selectedValue = reflect.New(field.Type.Elem()).Interface()
			if err := db.First(selectedValue, selected).Error; err != nil {
				return nil, errors.New("failed to find selected value: " + err.Error())
			}

			// Validate the field
			var err = validateModelField(modelField, selectedValue)
			if err != nil {
				return nil, errors.New("failed to validate field: " + err.Error())
			}

			// Set the field
			correctPtrSet(reflectValue.Field(i), selectedValue)
		case "file":
			// If the field is a file, the field is a struct of type fields.File.
			// We need to get the file from the request and save it.
			var file, header, err = rq.Request.FormFile("form_" + field.Name)
			if err != nil {
				if err == http.ErrMissingFile {
					if formField.Required {
						return nil, errors.New("failed to parse field: " + field.Name + " is required")
					}
					continue
				}
				return nil, errors.New("failed to parse field: " + err.Error())
			}

			// Create a new file
			defer file.Close()
			fileField, err := fs.NewFile(mgr, filepath.Join("admin/"+header.Filename), file)
			if err != nil {
				return nil, errors.New("failed to parse field: " + err.Error())
			}

			// Validate the field
			err = validateModelField(modelField, fileField)
			if err != nil {
				return nil, errors.New("failed to validate field: " + err.Error())
			}

			// Set the field
			correctPtrSet(modelField, fileField)
		case "text", "textarea", "password", "email", "url", "tel", "search", "color", "range":
			// If the field is a text, textarea, password, email, url, tel, search, color, range or file, the field is a string.
			//
			// We need to parse the value to the correct type.
			if len(value) == 0 {
				value = []string{""}
			}
			var v any = value[0]

			// Check if the struct field implements .FromString(string) error
			if modelField.Kind() != reflect.Ptr {
				modelField = reflect.New(modelField.Type())
			}

			// If the field implements FromStringer, call FromString.
			if value[0] != "" {
				var fromStringer, ok = modelField.Interface().(core.FromStringer)
				if ok {
					if err := fromStringer.FromString(value[0]); err != nil {
						return nil, fmt.Errorf("failed to parse field %s, are you sure it is formatted correctly?", field.Name)
					}
					v = fromStringer
				}
			} else {
				v = reflect.New(modelField.Type().Elem()).Interface()
			}

			// Validate the field
			var err = validateModelField(modelField, v)
			if err != nil {
				return nil, errors.New("failed to validate field: " + err.Error())
			}

			// Set the field
			correctPtrSet(reflectValue.Field(i), v)
		case "number":
			// If the field is a number, we need to parse the value to the correct type.
			var (
				numberValue any
				err         error
			)
			numberValue, err = strconv.ParseFloat(value[0], 64)
			if err != nil {
				return nil, errors.New("failed to parse number: " + err.Error())
			}

			// Validate the field
			err = validateModelField(modelField, numberValue)
			if err != nil {
				return nil, errors.New("failed to validate field: " + err.Error())
			}

			// Set the field
			correctPtrSet(reflectValue.Field(i), numberValue)
		case "date":
			// Parse time related fields.
			var dateValue, err = time.Parse("2006-01-02", value[0])
			if err != nil {
				return nil, errors.New("failed to parse date: " + err.Error())
			}

			// Validate the field
			err = validateModelField(modelField, dateValue)
			if err != nil {
				return nil, errors.New("failed to validate field: " + err.Error())
			}

			// Set the field
			correctPtrSet(reflectValue.Field(i), dateValue)
		case "datetime":
			// Parse time related fields.
			var dateValue, err = time.Parse("2006-01-02 15:04:05", value[0])
			if err != nil {
				return nil, errors.New("failed to parse datetime: " + err.Error())
			}

			// Validate the field
			err = validateModelField(modelField, dateValue)
			if err != nil {
				return nil, errors.New("failed to validate field: " + err.Error())
			}

			// Set the field
			correctPtrSet(reflectValue.Field(i), dateValue)
		case "duration":
			// Parse time related fields.
			var durationValue, err = time.ParseDuration(value[0])
			if err != nil {
				return nil, errors.New("failed to parse duration: " + err.Error())
			}

			// Validate the field
			err = validateModelField(modelField, durationValue)
			if err != nil {
				return nil, errors.New("failed to validate field: " + err.Error())
			}

			// Set the field
			correctPtrSet(reflectValue.Field(i), durationValue)
		default:
			return nil, fmt.Errorf("unknown form field type %s for field %s", formField.Type, formField.Type)
		}
	}

	// Return pointer to the model
	// The interface does not have a pointer, so we need to get the pointer to the value
	return f.Model, nil
}
