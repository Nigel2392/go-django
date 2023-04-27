package forms

import (
	"errors"
	"reflect"
	"strings"

	"github.com/Nigel2392/go-django/core/fs"
	"github.com/Nigel2392/go-django/core/models"
	"github.com/Nigel2392/go-django/core/models/modelutils"
	"github.com/Nigel2392/router/v3/request"
	"gorm.io/gorm"
)

// Create a new form for a model.
type Form struct {
	CSRFToken     *request.CSRFToken
	Method, URL   string
	Fields        []*FormField
	Model         any
	UpdatedFields []string
	Disabled      bool
	JS            map[string]bool
}

// Instantiate a new form.
func NewForm(method, url string, rq *request.Request, db *gorm.DB, mdl any) *Form {
	var form = &Form{
		Method:    method,
		URL:       url,
		CSRFToken: rq.Data.CSRFToken,
	}

	form.Model = mdl
	form.Fields, form.JS = generateFields(mdl, db, rq)

	return form
}

// Save a form to the database.
// Validate if the new value is different from the old value.
func (f *Form) Save(db *gorm.DB) (bool, error) {
	var id = modelutils.GetID(f.Model, "ID")
	if id.IsZero() {
		var err = db.FirstOrCreate(f.Model, f.Model).Error
		if err != nil {
			return false, err
		}
		return true, nil
	}

	var t = reflect.TypeOf(f.Model)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	var reflectValue = reflect.ValueOf(f.Model)
	if reflectValue.Kind() == reflect.Ptr {
		reflectValue = reflectValue.Elem()
	}

	var newModel = reflect.New(t).Interface()
	v, err := modelutils.GetField(f.Model, "ID", false)
	if err != nil {
		return false, err
	}

	switch v.(type) {
	case uint, uint8, uint16, uint32, uint64:
		err = db.First(newModel, id.Uint()).Error
	case int, int8, int16, int32, int64:
		err = db.First(newModel, id.Int()).Error
	case string:
		err = db.First(newModel, id.String()).Error
	case models.Model:
		err = db.First(newModel, id.UUID()).Error
	}
	if err != nil {
		return false, err
	}

	for i := 0; i < t.NumField(); i++ {
		var field = t.Field(i)
		var adminTag = field.Tag.Get("admin")
		if adminTag == "-" {
			continue
		}
		var formField = f.getField(field.Name)
		if formField == nil {
			continue
		}

		var newReflectValue = reflect.ValueOf(newModel)
		if newReflectValue.Kind() == reflect.Ptr {
			newReflectValue = newReflectValue.Elem()
		}

		var newField = newReflectValue.FieldByName(field.Name)
		var oldField = reflectValue.FieldByName(field.Name)

		// Validate if the new value is different from the old value.
		if newField.Kind() == reflect.Ptr {
			if newField.IsNil() {
				if !oldField.IsNil() {
					f.UpdatedFields = append(f.UpdatedFields, field.Name)
					newField.Set(oldField)
				}
			} else {
				if oldField.IsNil() {
					f.UpdatedFields = append(f.UpdatedFields, field.Name)
					newField.Set(oldField)
				} else {
					if newField.Elem().Interface() != oldField.Elem().Interface() {
						f.UpdatedFields = append(f.UpdatedFields, field.Name)
						newField.Set(oldField)
					}
				}
			}
		} else {
			var fieldType = field.Type
			if fieldType.Kind() == reflect.Ptr {
				fieldType = fieldType.Elem()
			}
			var isSlice = fieldType.Kind() == reflect.Slice
			if isSlice {
				fieldType = fieldType.Elem()
			}

			if modelutils.IsModel(fieldType) {
				var changed bool
				if isSlice {
					changed = !slicesEqual(newField, oldField)
				} else {
					var comparable = newField.Comparable() && oldField.Comparable()
					var canInterface = newField.CanInterface() && oldField.CanInterface()
					changed = comparable && canInterface && newField.Equal(oldField)
				}
				if changed {
					f.UpdatedFields = append(f.UpdatedFields, field.Name)
					newField.Set(oldField)
				}
				continue
			}

			if !reflect.DeepEqual(newField.Interface(), oldField.Interface()) ||
				newField.Comparable() && oldField.Comparable() &&
					newField.Interface() != oldField.Interface() {
				f.UpdatedFields = append(f.UpdatedFields, field.Name)
				newField.Set(oldField)
			}
		}
	}
	return false, db.Save(f.Model).Error
}

// Process and save the form if valid.
func (f *Form) Process(r *request.Request, mgr *fs.Manager, db *gorm.DB) (any, bool, error) {
	if f.Disabled {
		return nil, false, errors.New("form is disabled")
	}

	var kv map[string][]string = r.Form()
	for k, v := range kv {
		delete(kv, k)
		if len(v) == 0 {
			v = []string{""}
		}
		kv[strings.TrimPrefix(k, "form_")] = v
	}
	//	for k, v := range r.Request.MultipartForm.File {
	//		r.Request.MultipartForm.File[strings.TrimPrefix(k, "form_")] = v
	//		delete(r.Request.MultipartForm.File, k)
	//	}
	//	for k, v := range r.Request.MultipartForm.Value {
	//		r.Request.MultipartForm.Value[strings.TrimPrefix(k, "form_")] = v
	//		delete(r.Request.MultipartForm.Value, k)
	//	}
	var m, err = f.submit(kv, mgr, db, r)
	if err != nil {
		return nil, false, errors.New("failed to submit form: " + err.Error())
	}

	created, err := f.Save(db)
	if err != nil {
		return nil, false, errors.New("failed to save form: " + err.Error())
	}

	m = modelutils.NewPtr(f.Model).Interface()
	err = db.First(m, modelutils.GetID(f.Model, "ID")).Error
	if err != nil {
		return nil, false, errors.New("failed to get model: " + err.Error())
	}

	return m, created, nil
}

// Formfield struct used to render and validate the form.
type FormField struct {
	Name, Label, Type, Value                string
	Autocomplete, Custom                    string
	ReadOnly, Checked, Disabled, NeedsAdmin bool
	Required, Multiple, Selected            bool
	readonlyfull, disabledfull, bcrypt      bool
	Classes, LabelClasses, DivClasses       []string
	Options                                 []*FormField
	Model                                   interface{}
	isAdmin                                 bool
}

// Disable the form, and all its fields.
func (f *Form) Disable() {
	f.Disabled = true
	for _, field := range f.Fields {
		field.Disable()
	}
}

// Disable an individual field.
func (f *FormField) Disable() {
	f.Disabled = true
	for _, option := range f.Options {
		option.Disabled = true
	}
}
