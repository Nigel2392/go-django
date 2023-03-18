package forms

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/Nigel2392/go-django/admin/internal/tags"
	coreModels "github.com/Nigel2392/go-django/core/models"
	"github.com/Nigel2392/go-django/core/modelutils"
	"github.com/Nigel2392/router/v3/request"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func getTypes(mdl any, name string) (string, reflect.Type) {
	var v = reflect.ValueOf(mdl)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	var f = v.FieldByName(name)
	switch f.Kind() {
	case reflect.String:
		return "text", f.Type()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		// Check if time.Duration
		if f.Type().Name() == "Duration" {
			return "duration", f.Type()
		}
		return "number", f.Type()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "number", f.Type()
	case reflect.Float32, reflect.Float64:
		return "number", f.Type()
	case reflect.Bool:
		return "checkbox", f.Type()
	case reflect.Slice:
		return "select", f.Type()
	case reflect.Struct:
		switch f.Type().Name() {
		case "Time":
			return "datetime", f.Type()
		}
	}
	return "text", f.Type()
}

func generateFields(mdl any, db *gorm.DB, rq *request.Request) []*FormField {
	var fields = make([]*FormField, 0)

	var t = reflect.TypeOf(mdl)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	for i := 0; i < t.NumField(); i++ {
		var field = t.Field(i)

		var tag = field.Tag.Get("form")

		if tag == "-" {
			continue
		}

		var tagMap = tags.ParseTags(tag)

		var multiple = tagMap.Get("multiple") == "true"
		var fieldName = tagMap.Get("name", field.Name)
		var disabled = tagMap.Exists("disabled")
		var needsAdmin = tagMap.Exists("needs_admin")
		var isAdmin = rq.User != nil && rq.User.IsAdmin()
		var value string

		var options = make([]*FormField, 0)
		var t, reflectedType = getTypes(mdl, field.Name)

		var kind reflect.Kind = reflectedType.Kind()
		if kind == reflect.Ptr {
			kind = reflectedType.Elem().Kind()
		}

		switch kind {
		case reflect.Slice:
			// If it is a slice, there is a good chance it will be a M2M relationship.
			// This means we need to fetch both the already bound items,
			// and the other items which are not bound.
			var sliceType = reflectedType.Elem()
			if sliceType.Kind() == reflect.Ptr {
				sliceType = sliceType.Elem()
			}
			if !modelutils.IsModel(sliceType) {
				var formField = &FormField{
					isAdmin:    isAdmin,
					Name:       fieldName,
					Label:      fieldName,
					Type:       "text",
					Disabled:   disabled,
					NeedsAdmin: needsAdmin,
				}
				// Get the value of the field.
				var fieldValue = reflect.ValueOf(mdl)
				if fieldValue.Kind() == reflect.Ptr {
					fieldValue = fieldValue.Elem()
				}
				fieldValue = fieldValue.FieldByName(field.Name)
				fieldValue = modelutils.DePtr(fieldValue)
				// Check if it implements the Stringer interface.
				if fieldValue.CanInterface() {
					var stringer, ok = fieldValue.Interface().(fmt.Stringer)
					if ok {
						formField.Value = stringer.String()
					}
				}

				addLabels(formField, tagMap)
				fields = append(fields, formField)
				continue
			}
			multiple = true
			var ownItems = reflect.MakeSlice(reflectedType, 0, 0).Interface()
			var err = db.Model(mdl).Association(field.Name).Find(&ownItems)
			if err != nil {
				panic(err)
			}
			var ownItemValue = reflect.ValueOf(ownItems)
			var otherItems = reflect.MakeSlice(reflectedType, 0, 0).Interface()
			var otherVal = modelutils.DePtr(reflect.New(reflectedType))
			var otherInterface = otherVal.Interface()

			var tx = db.Model(otherInterface)

			// Whoops, I wrote some ugle code.
			//
			// Convert a slice of models to a slice of ID's
			//
			// These ID's will then be used to exclude items which are already bound
			// from the 'unselected' list.
			var fieldType, isPtr = getFieldType(sliceType, "ID")
			switch fieldType.Kind() {
			case reflect.Int:
				tx = setIDSlice[int](tx, ownItemValue, "ID", isPtr)
			case reflect.Int8:
				tx = setIDSlice[int8](tx, ownItemValue, "ID", isPtr)
			case reflect.Int16:
				tx = setIDSlice[int16](tx, ownItemValue, "ID", isPtr)
			case reflect.Int32:
				tx = setIDSlice[int32](tx, ownItemValue, "ID", isPtr)
			case reflect.Int64:
				tx = setIDSlice[int64](tx, ownItemValue, "ID", isPtr)
			case reflect.Uint:
				tx = setIDSlice[uint](tx, ownItemValue, "ID", isPtr)
			case reflect.Uint8:
				tx = setIDSlice[uint8](tx, ownItemValue, "ID", isPtr)
			case reflect.Uint16:
				tx = setIDSlice[uint16](tx, ownItemValue, "ID", isPtr)
			case reflect.Uint32:
				tx = setIDSlice[uint32](tx, ownItemValue, "ID", isPtr)
			case reflect.Uint64:
				tx = setIDSlice[uint64](tx, ownItemValue, "ID", isPtr)
			case reflect.String:
				tx = setIDSlice[string](tx, ownItemValue, "ID", isPtr)
			default:
				// Check if it is of type core.DefaultIDField
				if fieldType == reflect.TypeOf(coreModels.UUIDField{}) {
					tx = setIDSlice[coreModels.UUIDField](tx, ownItemValue, "ID", isPtr)
				} else if fieldType == reflect.TypeOf(uuid.UUID{}) {
					tx = setIDSlice[uuid.UUID](tx, ownItemValue, "ID", isPtr)
				} else {
					panic("Unsupported type for M2M field: " + fieldType.String())
				}
			}

			tx.Find(&otherItems)

			var name = modelutils.DePtr(reflect.New(reflectedType)).Type().String()

			// Create the two select fields.
			var selectedField = &FormField{
				isAdmin:    isAdmin,
				Name:       field.Name,
				Label:      "Selected",
				Type:       "select",
				Disabled:   disabled,
				NeedsAdmin: needsAdmin,
				Options:    make([]*FormField, 0),
			}
			var otherField = &FormField{
				isAdmin:    isAdmin,
				Name:       "deselected_" + name,
				Label:      "Other",
				Type:       "select",
				Disabled:   disabled,
				NeedsAdmin: needsAdmin,
				Options:    make([]*FormField, 0),
			}

			// Fill already selected items.
			var ownArray = reflect.ValueOf(ownItems)
			for i := 0; i < ownArray.Len(); i++ {
				var vval = ownArray.Index(i).Interface()
				var option = &FormField{
					isAdmin:    isAdmin,
					Name:       name,
					Label:      valueFromInterface(vval),
					Value:      modelutils.GetID(vval, "ID").String(),
					Checked:    true,
					Selected:   true,
					NeedsAdmin: needsAdmin,
				}
				selectedField.Options = append(selectedField.Options, option)
			}

			// Fill other items.
			var otherArray = reflect.ValueOf(otherItems)
			for i := 0; i < otherArray.Len(); i++ {
				var vval = otherArray.Index(i).Interface()
				var name = modelutils.GetName(vval)
				var option = &FormField{
					isAdmin:    isAdmin,
					Name:       name,
					Label:      modelutils.GetModelDisplay(vval),
					Value:      modelutils.GetID(vval, "ID").String(),
					NeedsAdmin: needsAdmin,
				}
				otherField.Options = append(otherField.Options, option)
			}

			// Create the field group.
			var fieldGroup = &FormField{
				isAdmin: isAdmin,
				Type:    "m2m",
				Name:    field.Name,
				Options: []*FormField{
					selectedField, otherField,
				},
				Model:      reflect.New(reflectedType).Elem().Interface(),
				NeedsAdmin: needsAdmin,
			}

			fields = append(fields, fieldGroup)
			continue

		case reflect.Struct:
			if !modelutils.IsModel(field) || modelutils.IsModelField(field) {
				continue
			}

			// check if time.Time
			var v = reflect.ValueOf(mdl)
			if v.Kind() == reflect.Ptr {
				v = v.Elem()
			}
			var f = v.FieldByName(field.Name)
			var vinter = f.Interface()

			if modelutils.IsActualModel(vinter) {
				var id = modelutils.GetID(vinter, "ID")
				value = fmt.Sprintf("%v", id)
				fieldName = "ID"
				disabled = true
			} else {
				switch vinter := vinter.(type) {
				case time.Time:
					value = vinter.Format("2006-01-02 15:04:05")
				case *time.Time:
					value = vinter.Format("2006-01-02 15:04:05")
				case gorm.Model, coreModels.Model:
					var id = modelutils.GetID(vinter, "ID")
					value = fmt.Sprintf("%v", id)
					fieldName = "ID"
					disabled = true
				default:
					var options []*FormField = make([]*FormField, 0)
					if reflectedType.Kind() == reflect.Ptr {
						reflectedType = reflectedType.Elem()
					}
					var selectedItem = reflect.New(reflectedType).Interface()
					var err = db.Model(mdl).Association(field.Name).Find(&selectedItem)
					if err == nil {
						var id = modelutils.GetID(selectedItem, "ID")
						if id.IsZero() {
							goto noSelected
						}
						options = append(options, &FormField{
							Name:       modelutils.GetName(selectedItem),
							Label:      modelutils.GetModelDisplay(selectedItem),
							Value:      id.String(),
							Selected:   true,
							NeedsAdmin: needsAdmin,
						})
						goto selectOthers
					noSelected:
						options = append(options, &FormField{
							Name:       "none",
							Label:      "None",
							Value:      "0",
							Selected:   true,
							NeedsAdmin: needsAdmin,
						})
					}
				selectOthers:
					var otherItems = reflect.MakeSlice(reflect.SliceOf(reflectedType), 0, 0).Interface()
					var otherVal = modelutils.DePtr(reflect.New(reflectedType))
					var otherInterface = otherVal.Interface()
					db.Model(otherInterface).Not(selectedItem).Find(&otherItems)
					var otherArray = reflect.ValueOf(otherItems)

					for i := 0; i < otherArray.Len(); i++ {
						var vval = otherArray.Index(i).Interface()
						var name = modelutils.GetName(vval)
						var option = &FormField{
							isAdmin:    isAdmin,
							Name:       name,
							Label:      modelutils.GetModelDisplay(vval),
							Value:      modelutils.GetID(vval, "ID").String(),
							NeedsAdmin: needsAdmin,
						}
						options = append(options, option)
					}

					var f = &FormField{
						isAdmin:    isAdmin,
						Name:       field.Name,
						Label:      fieldName,
						Type:       "fk",
						Disabled:   disabled,
						Options:    options,
						NeedsAdmin: needsAdmin,
					}

					var cls = strings.Split(tagMap.Get("class"), " ")
					var lblcls = strings.Split(tagMap.Get("label_class"), " ")
					var divcls = strings.Split(tagMap.Get("div_class"), " ")
					f.Classes = append(cls, "admin-form-input")
					f.LabelClasses = append(lblcls, "admin-form-label")
					f.DivClasses = append(divcls, "admin-form-div")

					fields = append(fields, f)

					continue
				}
			}
		default:
			value = getValue(mdl, field.Name)
		}

		var formFieldTyp = tagMap.Get("type", t)
		if formFieldTyp == "select" {
			var currentValue = getValue(mdl, field.Name)
			options = getOptions(mdl, field, fieldName, currentValue, isAdmin, tagMap)
		}

		var f = &FormField{
			isAdmin:      isAdmin,
			Name:         field.Name,
			Label:        fieldName,
			Type:         formFieldTyp,
			ReadOnly:     tagMap.Get("readonly") == "true" || tagMap.Exists("readonly"),
			Required:     tagMap.Get("required") == "true" || tagMap.Exists("required"),
			Disabled:     disabled,
			Multiple:     multiple,
			Model:        mdl,
			Options:      options,
			Value:        value,
			Autocomplete: tagMap.Get("autocomplete", ""),
			Custom:       tagMap.Get("custom", ""),
			disabledfull: tagMap.Get("disabledfull", "") == "true" || tagMap.Exists("disabledfull"),
			readonlyfull: tagMap.Get("readonlyfull", "") == "true" || tagMap.Exists("readonlyfull"),
			bcrypt:       tagMap.Get("bcrypt", "") == "true" || tagMap.Exists("bcrypt"),
			NeedsAdmin:   needsAdmin,
		}

		addLabels(f, tagMap)

		if f.Type == "checkbox" {
			f.Checked = f.Value == "true"
			f.Value = ""
		}
		fields = append(fields, f)
	}
	return fields
}

func addLabels(f *FormField, tagMap tags.TagMap) {
	var cls = strings.Split(tagMap.Get("class"), " ")
	var lblcls = strings.Split(tagMap.Get("label_class"), " ")
	var divcls = strings.Split(tagMap.Get("div_class"), " ")
	f.Classes = append(cls, "admin-form-input")
	f.LabelClasses = append(lblcls, "admin-form-label")
	f.DivClasses = append(divcls, "admin-form-div")
}

func getOptions(mdl any, field reflect.StructField, fieldName string, value string, isAdmin bool, tagMap tags.TagMap) []*FormField {
	var needsAdmin = tagMap.Get("needs_admin") == "true" || tagMap.Exists("needs_admin")
	var optionsCallable = fmt.Sprintf("Get%sOptions", field.Name)
	var optionsFunc = reflect.ValueOf(mdl).MethodByName(optionsCallable)
	var opts = make([]*FormField, 0)
	if optionsFunc.IsValid() {
		var results = optionsFunc.Call([]reflect.Value{})
		if len(results) == 1 {
			var optionsInterface = results[0].Interface()
			var optionsSlice = reflect.ValueOf(optionsInterface)
			for i := 0; i < optionsSlice.Len(); i++ {
				var option = optionsSlice.Index(i).Interface()
				strValue := valueFromInterface(option)
				var selected = value == strValue
				var optionField = &FormField{
					isAdmin:    isAdmin,
					Name:       fieldName,
					Label:      strValue,
					Value:      strValue,
					Selected:   selected,
					NeedsAdmin: needsAdmin,
				}
				opts = append(opts, optionField)
			}
		}
	} else {
		var options = tagMap.Get("options")
		var optionsSlice = strings.Split(options, ",")
		for _, option := range optionsSlice {
			var optionField = &FormField{
				isAdmin: isAdmin,
				Name:    fieldName,
				Label:   option,
				Value:   option,
			}
			opts = append(opts, optionField)
		}
	}
	return opts
}

func getFieldType(field reflect.Type, IDName string) (reflect.Type, bool) {
	if field.Kind() == reflect.Ptr {
		field = field.Elem()
	}

	var newF = reflect.New(field)
	if newF.Kind() == reflect.Ptr {
		newF = newF.Elem()
	}

	var newField = newF.FieldByName(IDName)
	if newField.IsValid() {
		var newFieldType = newField.Type()
		if newFieldType.Kind() == reflect.Ptr {
			newFieldType = newFieldType.Elem()
			return newFieldType, true
		}
		return newFieldType, false
	}
	return field, false
}

func setIDSlice[T any](db *gorm.DB, v reflect.Value, fieldName string, isPtr bool) *gorm.DB {
	if isPtr {
		return db.Not(sliceUp[*T](v, fieldName))
	} else {
		return db.Not(sliceUp[T](v, fieldName))
	}
}
