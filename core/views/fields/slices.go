package fields

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"unsafe"

	"github.com/Nigel2392/go-django/core/modelutils/converters"
	"github.com/Nigel2392/go-django/core/views/interfaces"
	"github.com/Nigel2392/router/v3/request"
	"github.com/Nigel2392/tags"
)

// GenericOptionsGetter must be implemented.
type SliceField[T interfaces.Option] struct {
	items   []interfaces.Option
	Convert func([]string) ([]T, error)
}

func (i SliceField[T]) Selected() (value T) {
	if len(i.items) == 0 {
		return value
	}
	var item = i.items[0]
	if item == nil {
		return value
	}
	return item.(T)
}

func (i *SliceField[T]) Scan(src interface{}) error {
	var str = src.(string)
	return json.Unmarshal([]byte(str), i)
}

func (i SliceField[T]) Value() (driver.Value, error) {
	var b, err = json.Marshal(i)
	return string(b), err
}

type iface struct {
	_type uintptr
	data  unsafe.Pointer
}

func (i *SliceField[T]) FormValues(v []string) error {
	i.items = make([]interfaces.Option, len(v))
	if len(v) == 0 {
		return nil
	}
	if i.Convert != nil {
		var items, err = i.Convert(v)
		if err != nil {
			return err
		}
		i.items = make([]interfaces.Option, len(items))
		for index, item := range items {
			i.items[index] = item
		}
		return nil
	}

	for index, v := range v {
		var val, err = converters.Convert[T](v)
		if err != nil {
			return err
		}
		i.items[index] = val
	}
	return nil
}

// GenericOptionsGetter must be implemented.
func (i *SliceField[T]) Initial(r *request.Request, model any, fieldName string) {
	var getOptionsFuncName = fmt.Sprintf("Get%sOptions", fieldName)
	var valueOf = reflect.ValueOf(model)
	var method = valueOf.MethodByName(getOptionsFuncName)
	if !method.IsValid() {
		panic(fmt.Sprintf("Method %s does not exist on model %s", getOptionsFuncName, valueOf.Type().Name()))
	}

	switch method := method.Interface().(type) {
	case func() []interfaces.Option:
		i.items = method()
	case func(r *request.Request) []interfaces.Option:
		i.items = method(r)
	case func(r *request.Request, model any) []interfaces.Option:
		i.items = method(r, model)
	case func(r *request.Request, model any, fieldName string) []interfaces.Option:
		i.items = method(r, model, fieldName)
	default:
		panic(fmt.Sprintf("Method %s on model %s does not have the correct signature for SliceField[T]", getOptionsFuncName, valueOf.Type().Name()))
	}
}

func (i SliceField[T]) LabelHTML(_ *request.Request, name string, display_text string, tags tags.TagMap) interfaces.Element {
	return ElementType(fmt.Sprintf(`<label for="%s" %s>%s</label>`, name, TagMapToElementAttributes(tags, AllTagsLabel...), display_text))
}

func (i SliceField[T]) InputHTML(_ *request.Request, name string, tags tags.TagMap) interfaces.Element {
	var b strings.Builder
	b.WriteString(fmt.Sprintf(`<select name="%s" id="%s" %s>`, name, name, TagMapToElementAttributes(tags, AllTagsInput...)))
	for _, v := range i.items {
		b.WriteString(fmt.Sprintf(`<option value="%s"`, v.OptionValue()))
		if v.OptionSelected() {
			b.WriteString(` selected`)
		}
		b.WriteString(fmt.Sprintf(`>%s</option>`, v.OptionLabel()))

	}
	b.WriteString(`</select>`)
	return ElementType(b.String())
}

// OptionsGetter must be implemented.
type SelectField []string

func (i *SelectField) Scan(src interface{}) error {
	var str = src.(string)
	return json.Unmarshal([]byte(str), i)
}

func (i SelectField) Value() (driver.Value, error) {
	var b, err = json.Marshal(i)
	return string(b), err
}

func (i *SelectField) Initial(r *request.Request, model any, fieldName string) {
	var getOptionsFuncName = fmt.Sprintf("Get%sOptions", fieldName)
	var valueOf = reflect.ValueOf(model)
	var method = valueOf.MethodByName(getOptionsFuncName)
	if !method.IsValid() {
		panic(fmt.Sprintf("Method %s does not exist on model %s", getOptionsFuncName, valueOf.Type().Name()))
	}
	switch method := method.Interface().(type) {
	case func() []string:
		*i = method()
	case func(r *request.Request) []string:
		*i = method(r)
	case func(r *request.Request, model any) []string:
		*i = method(r, model)
	case func(r *request.Request, model any, fieldName string) []string:
		*i = method(r, model, fieldName)
	default:
		panic(fmt.Sprintf("Method %s on model %s does not have the correct signature for SelectField", getOptionsFuncName, valueOf.Type().Name()))
	}
}

func (i *SelectField) FormValues(v []string) error {
	if len(v) == 0 {
		return nil
	}
	*i = SelectField(v)
	return nil
}

func (i SelectField) LabelHTML(_ *request.Request, name string, display_text string, tags tags.TagMap) interfaces.Element {
	return ElementType(fmt.Sprintf(`<label for="%s" %s>%s</label>`, name, TagMapToElementAttributes(tags, AllTagsLabel...), display_text))
}

func (i SelectField) InputHTML(_ *request.Request, name string, tags tags.TagMap) interfaces.Element {
	var b strings.Builder
	b.WriteString(fmt.Sprintf(`<select name="%s" id="%s" %s>`, name, name, TagMapToElementAttributes(tags, AllTagsInput...)))
	for _, v := range i {
		b.WriteString(fmt.Sprintf(`<option value="%s">%s</option>`, v, v))
	}
	b.WriteString(`</select>`)
	return ElementType(b.String())
}

type MultipleSelectField []string

func (i *MultipleSelectField) Scan(src interface{}) error {
	var str = src.(string)
	return json.Unmarshal([]byte(str), i)
}

func (i MultipleSelectField) Value() (driver.Value, error) {
	var b, err = json.Marshal(i)
	return string(b), err
}

func (i *MultipleSelectField) Initial(r *request.Request, model any, fieldName string) {
	var getOptionsFuncName = fmt.Sprintf("Get%sOptions", fieldName)
	var valueOf = reflect.ValueOf(model)
	var method = valueOf.MethodByName(getOptionsFuncName)
	if !method.IsValid() {
		panic(fmt.Sprintf("Method %s does not exist on model %s", getOptionsFuncName, valueOf.Type().Name()))
	}
	switch method := method.Interface().(type) {
	case func() []string:
		*i = method()
	case func(r *request.Request) []string:
		*i = method(r)
	case func(r *request.Request, model any) []string:
		*i = method(r, model)
	case func(r *request.Request, model any, fieldName string) []string:
		*i = method(r, model, fieldName)
	default:
		panic(fmt.Sprintf("Method %s on model %s does not have the correct signature for SelectField", getOptionsFuncName, valueOf.Type().Name()))
	}
}

func (i *MultipleSelectField) FormValues(v []string) error {
	if len(v) == 0 {
		return nil
	}
	*i = MultipleSelectField(v)
	return nil
}

func (i MultipleSelectField) LabelHTML(_ *request.Request, name string, display_text string, tags tags.TagMap) interfaces.Element {
	return ElementType(fmt.Sprintf(`<label for="%s" %s>%s</label>`, name, TagMapToElementAttributes(tags, AllTagsLabel...), display_text))
}

func (i MultipleSelectField) InputHTML(_ *request.Request, name string, tags tags.TagMap) interfaces.Element {
	var b strings.Builder
	b.WriteString(fmt.Sprintf(`<select name="%s" id="%s" multiple %s>`, name, name, TagMapToElementAttributes(tags, AllTagsInput...)))
	for _, v := range i {
		b.WriteString(fmt.Sprintf(`<option value="%s">%s</option>`, v, v))
	}
	b.WriteString(`</select>`)
	return ElementType(b.String())
}

type CheckBoxField []string

func (i *CheckBoxField) Scan(src interface{}) error {
	var str = src.(string)
	return json.Unmarshal([]byte(str), i)
}

func (i CheckBoxField) Value() (driver.Value, error) {
	var b, err = json.Marshal(i)
	return string(b), err
}

func (i *CheckBoxField) FormValues(v []string) error {
	if len(v) == 0 {
		return nil
	}
	*i = CheckBoxField(v)
	return nil
}

func (i CheckBoxField) LabelHTML(_ *request.Request, name string, display string, tags tags.TagMap) interfaces.Element {
	return ElementType(fmt.Sprintf(`<label for="%s" %s>%s</label>`, name, TagMapToElementAttributes(tags, AllTagsLabel...), display))
}

func (i CheckBoxField) InputHTML(_ *request.Request, name string, tags tags.TagMap) interfaces.Element {
	var b strings.Builder
	for _, v := range i {
		b.WriteString(fmt.Sprintf(`<input type="checkbox" name="%s" id="%s" value="%s" %s>`, name, name, v, TagMapToElementAttributes(tags, AllTagsLabel...)))
	}
	return ElementType(b.String())
}
