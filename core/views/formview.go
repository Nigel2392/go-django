package views

import (
	"fmt"
	"html/template"
	"io"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/Nigel2392/go-django/core/httputils"
	"github.com/Nigel2392/go-django/core/models/modelutils"
	"github.com/Nigel2392/go-django/core/views/fields"
	"github.com/Nigel2392/go-django/core/views/interfaces"
	"github.com/Nigel2392/orderedmap"
	"github.com/Nigel2392/router/v3/request"
	"github.com/Nigel2392/router/v3/request/response"
	"github.com/Nigel2392/tags"
)

type FormData struct {
	Values []string
	Files  []interfaces.File
}

type FormGroup struct {
	Label interfaces.Element
	Input interfaces.Element
}

type FormField struct {
	Field interfaces.FormField
	Tags  tags.TagMap
	value any
	Label string
}

// A FormView is a view that renders a form and handles the submission of that form.
//
// It will fill in the values of the form based on the model passed in.
//
// The default post method will call the Save method on the model passed in.
//
// Default tag for the formfields is "fields",
// this can be changed by setting the FormTag field.
type BaseFormView[T interfaces.Saver] struct {
	// The action of the form
	//
	// This will be used in mainly error messages.
	//
	// Example: "create", "view", "update"
	Action string
	// The formfields of the model passed into the form.
	formFields *orderedmap.Map[string, *FormField]
	// A fallback url for when an error occurs, and the form is submitted.
	BackURL func(r *request.Request) string
	// The success url for when the form is submitted.
	PostRedirect func(r *request.Request, data T) string
	// Extra function to run for authentication.
	//
	// This will be run before the form is rendered.
	ExtraAuth func(r *request.Request) error
	// The required permissions needed to perform an action on the model.
	NeedsAuth bool
	// Needs admin permissions to perform an action on the model.
	NeedsAdmin bool
	// Whether a superuser can perform the action.
	SuperUserCanPerform bool
	// Function to run before rendering a template.
	BeforeRender func(r *request.Request, data T, fieldMap *orderedmap.Map[string, *FormField])
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
	// Function to get a template.
	GetTemplate func(string) (*template.Template, string, error)
	// The instance of the model passed in.
	//
	// This will be automatically filled in with the model passed in.
	//
	// Do not edit this before the POST method is called!
	Instance T
	// The function to get the instance of the model passed in.
	//
	// This will set the instance to the result of the function.
	GetInstance func(r *request.Request) (T, error)

	// Function to run after the form is submitted and the view specified has been called.
	//
	// This will only get called on the POST method!
	AfterSubmit func(r *request.Request, data T)

	isNew bool

	fields []string

	// The tag to use to fetch attributes from struct fields.
	//
	// Defaults to "fields".
	FormTag string

	// Javascript to include in the template.
	//
	// You must include the <script> tags.
	Scripts map[string]template.HTML
}

// The fields to use with the form
//
// If the field cannot be found on the struct, we will check the instance's struct methods.
//
// This method must implement one of the following signatures:
//
//	func() interface{}
//	func() (interface{}, tags.TagMap)
//	func() (interface{}, tags.TagMap, error)
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
	var err error
	if c.formFields == nil {
		err = c.WithFields("*")
		if err != nil {
			panic(err)
		}
	}

	if c.NeedsAuth {
		if r.User == nil {
			r.Data.AddMessage("error", "You need to be logged in to perform this action.")
			goto errRedirect
		}
		if c.SuperUserCanPerform && !r.User.IsAdmin() {
			r.Data.AddMessage("error", "You need to be a superuser to perform this action.")
			goto errRedirect
		}
		if c.ExtraAuth != nil {
			if err := c.ExtraAuth(r); err != nil {
				r.Data.AddMessage("error", err.Error())
				goto errRedirect
			}
		}
	}

	err = c.setInstance(r)
	if err != nil {
		r.Data.AddMessage("error", err.Error())
		goto errRedirect
	}

	switch r.Method() {
	case http.MethodGet:
		if c.BeforeRender != nil {
			c.BeforeRender(r, c.Instance, c.formFields)
		}
		var form = make([]FormGroup, 0, c.formFields.Len())
		var err error
		c.formFields.ForEach(func(k string, v *FormField) bool {
			var formGroup = FormGroup{
				Label: v.Field.LabelHTML(r, k, v.Label, v.Tags),
				Input: v.Field.InputHTML(r, k, v.Tags),
			}
			form = append(form, formGroup)
			return true
		})

		var scripts = make([]template.HTML, 0, len(c.Scripts))
		for _, v := range c.Scripts {
			if len(v) == 0 {
				continue
			}
			scripts = append(scripts, v)
		}
		if len(scripts) > 0 {
			r.Data.Set("scripts", scripts)
		}

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

func (c *BaseFormView[T]) setInstance(r *request.Request) error {
	if c.GetInstance != nil {
		var err error
		c.Instance, err = c.GetInstance(r)
		if err != nil {
			return err
		}
	} else {
		c.Instance = *new(T)
	}

	if c.formFields == nil {
		c.formFields = orderedmap.New[string, *FormField]()
	}
	if c.FormTag == "" {
		c.FormTag = "fields"
	}
	for _, field := range c.fields {
		var f, tagmap, err = getFieldData(c.Instance, field, c.FormTag)
		if err != nil {
			panic(err)
		}

		var permissions, ok = tagmap.GetOK("permissions")
		if ok {
			if !r.User.HasPermissions(permissions...) {
				continue
			}
		}

		formField, ok := f.(interfaces.FormField)
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
		if c.isNew {
			if tagmap.Exists("omit_on_create") {
				continue
			}
		} else {
			if tagmap.Exists("omit_on_update") {
				continue
			}
		}

		scripter, ok := formField.(interfaces.Scripter)
		if ok {
			scriptName, script := scripter.Script()
			c.Scripts[scriptName] = script
		}

		// Fill up the initial value of a formfield if it implements the Initializer interface.
		var initializer interfaces.Initializer
		var reflectedField = reflect.ValueOf(formField)
		if reflectedField.Kind() == reflect.Ptr {
			reflectedField = reflectedField.Elem()
		}
		var typeOfInitializer = reflect.TypeOf((*interfaces.Initializer)(nil)).Elem()
		var newOf = reflect.New(reflectedField.Type())
		newOf.Elem().Set(reflectedField)
		if newOf.Type().Implements(typeOfInitializer) {
			initializer = newOf.Interface().(interfaces.Initializer)
		} else {
			if newOf.Elem().Type().Implements(typeOfInitializer) {
				initializer = newOf.Elem().Interface().(interfaces.Initializer)
			}
		}

		if initializer != nil {
			initializer.Initial(r, c.Instance, field)
			formField = initializer.(interfaces.FormField)
		}

		var label string
		var labelMethod = fmt.Sprintf("Get%sLabel", strings.Title(field))
		var labelMethodValue = reflect.ValueOf(c.Instance).MethodByName(labelMethod)
		if labelMethodValue.IsValid() {
			var labelMethodResult = labelMethodValue.Call([]reflect.Value{})
			label = labelMethodResult[0].String()
		}
		if label == "" || !ok {
			label = httputils.FormatLabel(field)
		}
		var _formField = &FormField{
			Field: formField,
			Tags:  tagmap,
			Label: label,
			value: f,
		}
		c.formFields.Set(field, _formField)
	}
	return nil
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
		v = reflect.ValueOf(modelutils.NewOf(model, false))
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
			return nil, nil, fmt.Errorf("field %s is not valid or does not exist", field)
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
	} else {
		var method = v.MethodByName(field)
		if method.IsValid() && method.CanInterface() {
			switch method.Interface().(type) {
			case func() interface{}:
				return method.Interface().(func() interface{})(), nil, nil
			case func() (interface{}, tags.TagMap):
				var res, tagmap = method.Interface().(func() (interface{}, tags.TagMap))()
				return res, tagmap, nil
			case func() (interface{}, tags.TagMap, error):
				return method.Interface().(func() (interface{}, tags.TagMap, error))()
			}
		}
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
	if c.GetTemplate != nil {
		var tmpl *template.Template
		var name string
		tmpl, name, err = c.GetTemplate(c.Template)
		if err != nil {
			return err
		}
		err = response.Template(r, tmpl, name)
	} else {
		err = response.Render(r, c.Template)
	}
	return err
}

func (c *BaseFormView[T]) defaultPost(r *request.Request) error {
	var err error
	if err = c.Save(r); err != nil {
		return err
	}

	if c.PostRedirect != nil {
		r.Redirect(c.PostRedirect(r, c.Instance), http.StatusFound)
	}

	return nil
}

// aquireFormData will parse the request and aquire the form data
//
// the data will be stored in the FormData map
//
// this will be later used to populate the instance's fields
//
// BEWARE: if a field is not present in the form, it will not be set, unless it is a boolean.
//
// This is for compatibility with html forms, where unchecked checkboxes are not sent.
//
// This is also to not override any values which were previously set on the instance, for updating purposes.
func (c *BaseFormView[T]) aquireFormData(r *request.Request) error {
	var err = r.Request.ParseMultipartForm(c.maxMem())
	if err != nil {
		return err
	}
	c.FormData = make(map[string]FormData)
	c.formFields.ForEach(func(k string, v *FormField) bool {
		if value, ok := r.Request.Form[k]; ok {
			c.FormData[k] = FormData{Values: value}
		} else if formFiles, ok := r.Request.MultipartForm.File[k]; ok {
			var files = make([]interfaces.File, len(formFiles))
			for i, f := range formFiles {
				files[i] = fields.FormFile{Filename: f.Filename, OpenFunc: func() (io.ReadSeekCloser, error) {
					return f.Open()
				}}
			}
			c.FormData[k] = FormData{Files: files}
		} else {
			if reflect.TypeOf(v.value).Kind() == reflect.Bool {
				c.FormData[k] = FormData{}
			}
		}
		return true
	})
	return nil
}

func (c *BaseFormView[T]) maxMem() int64 {
	if c.MaxMemory <= 0 {
		return 32 << 20
	}
	return c.MaxMemory
}

// fillModelInstance will fill the instance's fields with the data aquired from the form
//
// if a field is not present in the form, it will not be set, unless it is a boolean.
//
// The fields must either consist of primitives,
// time values or implement one of the following interfaces:
//
//   - interfaces.Field
//   - interfaces.FileField
func (c *BaseFormView[T]) fillModelInstance(r *request.Request) error {
	var typeOf = reflect.TypeOf(c.Instance)
	if typeOf.Kind() == reflect.Ptr {
		typeOf = typeOf.Elem()
	}

	var valueOf = reflect.ValueOf(c.Instance)
	if valueOf.Kind() == reflect.Ptr {
		valueOf = valueOf.Elem()
	}

	// The instance is nil.
	if !valueOf.IsValid() {
		valueOf = reflect.New(typeOf).Elem()
		if !valueOf.IsValid() {
			return fmt.Errorf("instance is not valid")
		}
		if reflect.TypeOf(c.Instance).Kind() == reflect.Ptr {
			c.Instance = valueOf.Addr().Interface().(T)
		} else {
			c.Instance = valueOf.Interface().(T)
		}
	}

	var err error
	c.formFields.ForEach(func(fieldName string, f *FormField) bool {
		var structField = valueOf.FieldByName(fieldName)
		if !structField.IsValid() {
			return true
		}
		if structField.Kind() == reflect.Ptr {
			structField = structField.Elem()
		}

		var rNewField = reflect.New(structField.Type())
		rNewField.Elem().Set(structField)
		var newField = rNewField.Interface()
		var checked bool
	fillField:
		switch v := newField.(type) {
		case interfaces.Field:
			var formField, ok = c.FormData[fieldName]
			if !ok {
				return true
			}
			v.FormValues(formField.Values)
			newField = v
		case interfaces.FileField:
			var formField, ok = c.FormData[fieldName]
			if !ok {
				return true
			}
			v.FormFiles(formField.Files)
			newField = v
		default:
			if canGuessFormField(structField.Interface()) {
				var formField, ok = c.FormData[fieldName]
				if !ok {
					return true
				}
				err = setGuessedField(structField, formField.Values)
				if err != nil {
					return false
				}

				if err = fieldValid(newField, f.Tags); err != nil {
					return false
				}

				return true
			} else if reflect.TypeOf(newField).Kind() == reflect.Ptr && !checked {
				newField = reflect.ValueOf(newField).Elem().Interface()
				checked = true
				goto fillField
			}
		}

		if err = fieldValid(newField, f.Tags); err != nil {
			return false
		}

		var field = reflect.ValueOf(newField)
		if field.Kind() == reflect.Ptr {
			field = field.Elem()
		}

		if !field.Type().ConvertibleTo(structField.Type()) {
			err = fmt.Errorf("field %s type %s is not convertible to %s", fieldName, field.Type(), structField.Type())
			return false
		}

		structField.Set(field)
		return true
	})

	return err
}

// fieldValid will check if the field is valid
func fieldValid(field interface{}, tags tags.TagMap) error {
	if v, ok := field.(interfaces.Validator); ok {
		if err := v.Validate(); err != nil {
			return err
		}
	}
	if v, ok := field.(interfaces.ValidatorTagged); ok {
		if err := v.ValidateWithTags(tags); err != nil {
			return err
		}
	}
	return nil
}

func (c *BaseFormView[T]) Save(r *request.Request) error {
	if c.OnSubmit != nil {
		return c.OnSubmit(r, c.Instance)
	}
	return c.Instance.Save(c.isNew)
}
