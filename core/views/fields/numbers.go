package fields

import (
	"database/sql/driver"
	"fmt"
	"reflect"
	"strconv"

	"github.com/Nigel2392/go-django/core/httputils/tags"
	"github.com/Nigel2392/go-django/core/views/interfaces"
	"github.com/Nigel2392/router/v3/request"
)

func NumberField(n any) interfaces.FormField {
	switch n := n.(type) {
	case int:
		return IntField(n)
	case int8:
		return IntField(n)
	case int16:
		return IntField(n)
	case int32:
		return IntField(n)
	case int64:
		return IntField(n)
	case uint:
		return IntField(n)
	case uint8:
		return IntField(n)
	case uint16:
		return IntField(n)
	case uint32:
		return IntField(n)
	case uint64:
		return IntField(n)
	case float32:
		return FloatField(n)
	case float64:
		return FloatField(n)
	default:
		var valueOf = reflect.ValueOf(n)
		if valueOf.Kind() == reflect.Ptr {
			valueOf = valueOf.Elem()
		}
		switch valueOf.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return IntField(valueOf.Int())
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return IntField(valueOf.Uint())
		case reflect.Float32, reflect.Float64:
			return FloatField(valueOf.Float())
		}
		panic(fmt.Sprintf("NumberField: unsupported type %T", n))
	}
}

type IntField int64

func (i *IntField) Scan(src interface{}) error {
	*i = IntField(src.(int64))
	return nil
}

func (i IntField) Value() (driver.Value, error) {
	return int64(i), nil
}

func (i *IntField) FormValues(v []string) error {
	if len(v) == 0 {
		return nil
	}
	var intie, err = strconv.ParseInt(v[0], 10, 64)
	if err != nil {
		return err
	}
	*i = IntField(intie)
	return nil
}

func (i IntField) LabelHTML(_ *request.Request, name string, tags tags.TagMap) interfaces.Element {
	return ElementType(fmt.Sprintf(`<label for="%s" %s>%s</label>`, name, TagMapToElementAttributes(tags, AllTagsLabel...), name))
}

func (i IntField) InputHTML(_ *request.Request, name string, tags tags.TagMap) interfaces.Element {
	return ElementType(fmt.Sprintf(`<input type="number" name="%s" value="%d" %s>`, name, i, TagMapToElementAttributes(tags, AllTagsInput...)))
}

type FloatField float64

func (i *FloatField) Scan(src interface{}) error {
	*i = FloatField(src.(float64))
	return nil
}

func (i FloatField) Value() (driver.Value, error) {
	return float64(i), nil
}

func (i *FloatField) FormValues(v []string) error {
	if len(v) == 0 {
		return nil
	}
	var intie, err = strconv.ParseFloat(v[0], 64)
	if err != nil {
		return err
	}
	*i = FloatField(intie)
	return nil
}

func (i FloatField) LabelHTML(_ *request.Request, name string, tags tags.TagMap) interfaces.Element {
	return ElementType(fmt.Sprintf(`<label for="%s" %s>%s</label>`, name, TagMapToElementAttributes(tags, AllTagsLabel...), name))
}

func (i FloatField) InputHTML(_ *request.Request, name string, tags tags.TagMap) interfaces.Element {
	return ElementType(fmt.Sprintf(`<input type="number" name="%s" value="%f" %s>`, name, i, TagMapToElementAttributes(tags, AllTagsInput...)))
}
