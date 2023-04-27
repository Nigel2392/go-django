package views

import (
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/Nigel2392/go-django/auth"
	"github.com/Nigel2392/go-django/core/httputils/tags"
	"github.com/Nigel2392/go-django/core/models/modelutils"
	"github.com/Nigel2392/go-django/core/views/fields"
	"github.com/Nigel2392/go-django/core/views/interfaces"
	"github.com/Nigel2392/orderedmap"
	"github.com/Nigel2392/router/v3/request"
	"github.com/Nigel2392/router/v3/request/response"
)

type FormData struct {
	Values []string
	Files  []interfaces.File
}

type FormGroup struct {
	Label interfaces.Element
	Input interfaces.Element
}

// A FormView is a view that renders a form and handles the submission of that form.
//
// It will fill in the values of the form based on the model passed in.
//
// The default post method will call the Save method on the model passed in.
type BaseFormView[T interfaces.Saver] struct {
	// The action of the form
	//
	// This will be used in mainly error messages.
	//
	// Example: "create", "view", "update"
	Action string
	// The formfields of the model passed into the form.
	formFields *orderedmap.Map[string, interfaces.FormField]
	// A fallback url for when an error occurs, and the form is submitted.
	BackURL func(r *request.Request) string
	// The required permissions needed to perform an action on the model.
	RequiredPermissions []*auth.Permission
	// Whether the user needs to be authenticated to view the page.
	NeedsAuth bool
	// Function to run on submission, when overridden; the default save method will not be called.
	OnSubmit func(r *request.Request, data T) error
	// The form data for when the form is submitted.
	//
	// This will only get filled on the POST method!
	FormData map[string]FormData
	// MaxMemory is the maximum memory used when parsing the files of a request.
	MaxMemory int64 // defaults to 32MB
	// Override the get and post methods.
	GET  func(r *request.Request) error
	POST func(r *request.Request, data T) error
	// Template to pass the template variables to.
	Template string
	// The instance of the model passed in.
	//
	// This will be automatically filled in with the model passed in.
	//
	// Do not edit this before the POST method is called!
	Instance T
	// The function to get the instance of the model passed in.
	//
	// This will set the instance to the result of the function.
	GetInstance func(r *request.Request) T

	// Function to run after the form is submitted and the view specified has been called.
	//
	// This will only get called on the POST method!
	AfterSubmit func(r *request.Request, data T)

	fields  []string
	tagsFor map[string]tags.TagMap

	// The tag to use to fetch attributes from struct fields.
	//
	// Defaults to "fields".
	FormTag string
}

func (c *BaseFormView[T]) WithFields(fields ...string) error {
	// var allFields bool
	if len(fields) == 1 && fields[0] == "*" {
		// allFields = true
		var v = reflect.ValueOf(*new(T))
		if !v.IsValid() {
			return nil
		}
		var t = v.Type()
		if t.Kind() == reflect.Ptr {
			t = t.Elem()
		}
		if t.Kind() != reflect.Struct {
			return fmt.Errorf("model is not a struct: %v", t.Kind())
		}
		var names = make([]string, 0, t.NumField())
		for i := 0; i < t.NumField(); i++ {
			var field = t.Field(i)
			if field.IsExported() && field.Type.Implements(reflect.TypeOf((*interfaces.FormField)(nil)).Elem()) {
				names = append(names, field.Name)
			}
		}
		return nil
	}
	if len(fields) == 0 {
		return fmt.Errorf("no fields found in model")
	}
	c.fields = fields
	return nil
}

func (c *BaseFormView[T]) Serve(r *request.Request) {
	if c.formFields == nil {
		var err = c.WithFields("*")
		if err != nil {
			panic(err)
		}
	}
	var err error
	if auth.LOGIN_URL != "" {
		if !r.User.IsAuthenticated() && c.NeedsAuth {
			r.Data.AddMessage("error", fmt.Sprintf("You need to be logged in to %s this item.", c.Action))
			r.Redirect(auth.LOGIN_URL, 302, r.Request.URL.Path)
			return
		}

		var u, ok = r.User.(*auth.User)
		if !ok {
			// This should never happen, but just in case.
			goto skip_auth
		}

		if len(c.RequiredPermissions) > 0 {
			if !r.User.(*auth.User).HasPerms(c.RequiredPermissions...) || !u.IsAuthenticated() {
				goto notAllowed
			}
		}
		goto skip_auth
	notAllowed:
		r.Data.AddMessage("error", fmt.Sprintf("You are not allowed to %s this item.", c.Action))
		goto errRedirect
	}
skip_auth:

	c.setInstance(r)

	switch r.Method() {
	case http.MethodGet:
		var form = make([]FormGroup, 0, c.formFields.Len())
		c.formFields.ForEach(func(k string, v interfaces.FormField) bool {
			var formGroup = FormGroup{
				Label: v.LabelHTML(r, k, c.tagsFor[k]),
				Input: v.InputHTML(r, k, c.tagsFor[k]),
			}
			form = append(form, formGroup)
			return true
		})
		r.Data.Set("form", form)
		if c.GET != nil {
			if err := c.GET(r); err != nil {
				r.Data.AddMessage("error", err.Error())
				goto errRedirect
			}
		}
		err = c.defaultGet(r)
		if err != nil {
			r.Data.AddMessage("error", err.Error())
			goto errRedirect
		}
		return
	case http.MethodPost:
		if err = c.aquireFormData(r); err != nil {
			r.Data.AddMessage("error", err.Error())
			goto errRedirect
		}
		if err = c.fillModelInstance(r); err != nil {
			r.Data.AddMessage("error", err.Error())
			goto errRedirect
		}
		if c.AfterSubmit != nil {
			defer func() {
				if err != nil {
					return
				}
				c.AfterSubmit(r, c.Instance)
			}()
		}
		fmt.Println("POST", c.FormData)
		if c.POST != nil {
			if err := c.POST(r, c.Instance); err != nil {
				r.Data.AddMessage("error", err.Error())
				goto errRedirect
			}
			return
		}
		err = c.defaultPost(r)
		if err != nil {
			r.Data.AddMessage("error", err.Error())
			goto errRedirect
		}
		return
	}

errRedirect:
	if c.BackURL != nil {
		r.Redirect(c.BackURL(r), 302, r.Request.URL.Path)
		return
	}
	r.Redirect(r.Request.Referer(), 302, r.Request.URL.Path)
	return
}

func (c *BaseFormView[T]) setInstance(r *request.Request) {
	if c.GetInstance != nil {
		c.Instance = c.GetInstance(r)
	} else {
		c.Instance = *new(T)
	}
	if c.formFields == nil {
		c.formFields = orderedmap.New[string, interfaces.FormField]()
	}
	if c.tagsFor == nil {
		c.tagsFor = make(map[string]tags.TagMap)
	}
	if c.FormTag == "" {
		c.FormTag = "fields"
	}
	for _, field := range c.fields {
		var f, tagmap, err = getFieldData(c.Instance, field, c.FormTag)
		if err != nil {
			panic(err)
		}

		c.tagsFor[field] = tagmap

		var formField, ok = f.(interfaces.FormField)
		if !ok {
			if canGuessFormField(f) {
				formField = guessFormField(f, tagmap)
				if formField != nil {
					goto addFormField
				}
			}
			panic(fmt.Errorf("field %s does not implement FormField interface", field))
		} else {
			var valueOf = reflect.ValueOf(formField)
			if valueOf.Kind() == reflect.Ptr {
				if valueOf.IsNil() {
					var newRefl = reflect.New(valueOf.Type().Elem())
					formField = newRefl.Interface().(interfaces.FormField)
				}
			}
		}
	addFormField:
		c.formFields.Set(field, formField)
	}
}

var (
	TAG_DEFAULT_DELIMITER_KEYVALUE = ";"
	TAG_DEFAULT_DELIMITER_KEY      = "="
	TAG_DEFAULT_DELIMITER_VALUE    = ","
)

func getFieldData(model any, field string, tagToFetch string) (interface{}, tags.TagMap, error) {
	var v = reflect.ValueOf(model)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if !v.IsValid() {
		v = reflect.ValueOf(modelutils.GetNewModel(model, false))
		if v.Kind() == reflect.Ptr {
			v = v.Elem()
		}
		if !v.IsValid() {
			return nil, nil, fmt.Errorf("model is not valid: %v", v.Kind())
		}
	}

	if v.Kind() != reflect.Struct {
		return nil, nil, fmt.Errorf("model is not a struct: %v", v.Kind())
	}
	var f = v.FieldByName(field)
	if f.IsValid() && f.CanInterface() {
		var structField, ok = v.Type().FieldByName(field)
		if !ok {
			return nil, nil, fmt.Errorf("field %s is not valid", field)
		}
		var tag = structField.Tag.Get(tagToFetch)
		var tagMap = tags.ParseWithDelimiter(tag, TAG_DEFAULT_DELIMITER_KEYVALUE, TAG_DEFAULT_DELIMITER_KEY, TAG_DEFAULT_DELIMITER_VALUE)

		if f.Kind() == reflect.Ptr {
			if f.IsNil() {
				var newRefl = reflect.New(f.Type().Elem())
				f.Set(newRefl)
			}
		}

		return f.Interface(), tagMap, nil
	}
	return nil, nil, fmt.Errorf("field %s is not valid", field)
}

func canGuessFormField(f interface{}) bool {
	switch f.(type) {
	case string, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		return true
	case []string:
		return true
	case bool:
		return true
	case time.Time:
		return true
	}
	var valueOf = reflect.ValueOf(f)
	var kind = valueOf.Kind()
	switch kind {
	case reflect.String:
		return true
	case reflect.Slice:
		if valueOf.Type().Elem().Kind() == reflect.String {
			return true
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return true
	case reflect.Float32, reflect.Float64:
		return true
	case reflect.Bool:
		return true
	case reflect.Struct:
		if valueOf.Type().String() == "time.Time" {
			return true
		}
	}
	return false
}

func guessFormField(f interface{}, t tags.TagMap) interfaces.FormField {
	var valueOf = reflect.ValueOf(f)
	var kind = valueOf.Kind()
	switch kind {
	case reflect.String:
		var textarea = t.GetSingle("type") == "textarea"
		if textarea {
			return fields.TextField(valueOf.String())
		}
		return fields.StringField(valueOf.String())
	case reflect.Slice:
		if valueOf.Type().Elem().Kind() == reflect.String {
			return fields.SelectField(valueOf.Interface().([]string))
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return fields.IntField(valueOf.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return fields.IntField(int64(valueOf.Uint()))
	case reflect.Float32, reflect.Float64:
		return fields.FloatField(valueOf.Float())
	case reflect.Bool:
		return fields.BoolField(f.(bool))
	case reflect.Struct:
		var f, ok = f.(time.Time)
		if !ok {
			return nil
		}
		if f.IsZero() {
			f = time.Date(time.Now().Year()-1, time.January, 1, 0, 0, 0, 0, time.UTC)
		}
		return fields.DateTimeField(f)
	}
	return nil
}

type typ int

const (
	typeInvalid typ = iota
	typeString
	typeInt
	typeFloat
	typeUint
	typeBool
	typeDateTime
	typeSelect
)

func (t typ) new() interface{} {
	switch t {
	case typeString:
		return ""
	case typeInt:
		return int64(0)
	case typeFloat:
		return float64(0.00)
	case typeUint:
		return uint64(0)
	case typeBool:
		return false
	case typeDateTime:
		return time.Date(time.Now().Year()-1, time.January, 1, 0, 0, 0, 0, time.UTC)
	case typeSelect:
		return []string{}
	}
	return nil
}

func convertToGuessedField(guessed typ, f []string) interface{} {
	if len(f) == 0 {
		return guessed.new()
	}

	switch guessed {
	case typeString:
		return f[0]
	case typeInt:
		var i, err = strconv.ParseInt(f[0], 10, 64)
		if err != nil {
			return int64(0)
		}
		return i
	case typeFloat:
		var f, err = strconv.ParseFloat(f[0], 64)
		if err != nil {
			return float64(0.00)
		}
		return f
	case typeUint:
		var u, err = strconv.ParseUint(f[0], 10, 64)
		if err != nil {
			return uint64(0)
		}
		return u
	case typeBool:
		switch strings.ToLower(f[0]) {
		case "on", "true", "1":
			return true
		case "off", "false", "0":
			return false
		default:
			return false
		}
	case typeDateTime:
		//"2006-01-02T15:04:05"
		var t, err = time.Parse("2006-01-02T15:04", f[0])
		if err != nil {
			return time.Date(time.Now().Year()-1, time.January, 1, 0, 0, 0, 0, time.UTC)
		}
		return t
	case typeSelect:
		return f
	case typeInvalid:
		return nil
	}
	return nil
}

func guessReflectedTyp(f reflect.Value) typ {
	switch f.Kind() {
	case reflect.String:
		return typeString
	case reflect.Slice:
		if f.Type().Elem().Kind() == reflect.String {
			return typeSelect
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return typeInt
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return typeUint
	case reflect.Float32, reflect.Float64:
		return typeFloat
	case reflect.Bool:
		return typeBool
	case reflect.Struct:
		if f.Type() == reflect.TypeOf(time.Time{}) {
			return typeDateTime
		}
	}
	return typeInvalid
}

func setGuessedField(m reflect.Value, value []string) error {
	var v = convertToGuessedField(guessReflectedTyp(m), value)
	if v == nil {
		return fmt.Errorf("field %s is not convertible to %s", m.Type().Name(), reflect.TypeOf(v).Name())
	}
	if !m.CanSet() {
		return fmt.Errorf("field %s is not settable", m.Type().Name())
	}
	if m.Type().ConvertibleTo(reflect.TypeOf(v)) {
		m.Set(reflect.ValueOf(v))
		return nil
	}
	return fmt.Errorf("field %s is not convertible to %s", m.Type().Name(), reflect.TypeOf(v).Name())
}

func (c *BaseFormView[T]) defaultGet(r *request.Request) error {
	var err error
	err = response.Render(r, c.Template)
	if err != nil {
		return err
	}
	return nil
}

func (c *BaseFormView[T]) defaultPost(r *request.Request) error {
	var err error
	if err = c.Save(r); err != nil {
		return err
	}

	if c.BackURL != nil {
		r.Redirect(c.BackURL(r), 302, r.Request.URL.Path)
	} else {
		r.Redirect(r.Request.Referer(), 302, r.Request.URL.Path)
	}

	return nil
}

func (c *BaseFormView[T]) aquireFormData(r *request.Request) error {
	var data = make(map[string]FormData)
	var err = r.Request.ParseMultipartForm(c.maxMem())
	if err != nil {
		return err
	}
	for k, v := range r.Request.Form {
		if _, ok := c.formFields.GetOK(k); !ok {
			continue
		}
		data[k] = FormData{Values: v}
	}
	for k, v := range r.Request.MultipartForm.File {
		if _, ok := c.formFields.GetOK(k); !ok {
			continue
		}
		var files = make([]interfaces.File, len(v))
		for i, f := range v {
			files[i] = fields.FormFile{Filename: f.Filename, OpenFunc: func() (io.ReadSeekCloser, error) {
				return f.Open()
			}}
		}
		data[k] = FormData{Files: files}
	}
	c.FormData = data
	return nil
}

func (c *BaseFormView[T]) maxMem() int64 {
	if c.MaxMemory <= 0 {
		return 32 << 20
	}
	return c.MaxMemory
}

func (c *BaseFormView[T]) fillModelInstance(r *request.Request) error {
	var typeOf = reflect.TypeOf(c.Instance)
	if typeOf.Kind() == reflect.Ptr {
		typeOf = typeOf.Elem()
	}

	var valueOf = reflect.ValueOf(c.Instance)
	if valueOf.Kind() == reflect.Ptr {
		valueOf = valueOf.Elem()
	}

	var (
		err error
	)

	c.formFields.ForEach(func(fieldName string, _ interfaces.FormField) bool {
		var fieldVal = valueOf.FieldByName(fieldName)
		if !fieldVal.IsValid() {
			return true
		}
		if fieldVal.Kind() == reflect.Ptr {
			fieldVal = fieldVal.Elem()
		}

		var instanceField = reflect.New(fieldVal.Type()).Interface()
		var checked bool
	fillField:
		switch v := instanceField.(type) {
		case interfaces.Field:
			var formField, ok = c.FormData[fieldName]
			if !ok {
				return true
			}
			v.FormValues(formField.Values)
			instanceField = v
		case interfaces.FileField:
			var formField, ok = c.FormData[fieldName]
			if !ok {
				return true
			}
			v.FormFiles(formField.Files)
			instanceField = v
		default:
			if canGuessFormField(fieldVal.Interface()) {
				var formField, ok = c.FormData[fieldName]
				if !ok {
					return true
				}
				err = setGuessedField(fieldVal, formField.Values)
				if err != nil {
					return false
				}

				if v, ok := instanceField.(interfaces.Validator); ok {
					if err = v.Validate(); err != nil {
						return true
					}
				}

				return true
			} else if reflect.TypeOf(instanceField).Kind() == reflect.Ptr && !checked {
				instanceField = reflect.ValueOf(instanceField).Elem().Interface()
				checked = true
				goto fillField
			}
		}
		if v, ok := instanceField.(interfaces.Validator); ok {
			if err = v.Validate(); err != nil {
				return true
			}
		}
		var field = reflect.ValueOf(instanceField)
		if field.Kind() == reflect.Ptr {
			field = field.Elem()
		}
		if field.Type().ConvertibleTo(fieldVal.Type()) {
			fieldVal.Set(field.Convert(fieldVal.Type()))
		} else {
			err = fmt.Errorf("field %s type %s is not convertible to %s", fieldName, field.Type(), fieldVal.Type())
			return false
		}
		return true
	})
	return err
}

func (c *BaseFormView[T]) Save(r *request.Request) error {
	if c.OnSubmit != nil {
		return c.OnSubmit(r, c.Instance)
	}
	return c.Instance.Save()
}
