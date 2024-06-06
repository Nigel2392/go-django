package widgets

import (
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/Nigel2392/django/core/errs"
	"github.com/pkg/errors"
	"golang.org/x/exp/constraints"
)

type NumberType interface {
	constraints.Integer | constraints.Float
}

type NumberWidget[T NumberType] struct {
	*BaseWidget
}

func NewNumberInput[T NumberType](attrs map[string]string) Widget {
	return &NumberWidget[T]{NewBaseWidget(S("number"), "forms/widgets/number.html", attrs)}
}

func (n *NumberWidget[T]) ValueToGo(value interface{}) (interface{}, error) {

	var (
		newT = new(T)
		t    = reflect.TypeOf(newT)
		v    = reflect.ValueOf(newT)
	)

	if t.Kind() == reflect.Pointer {
		t = t.Elem()
		v = v.Elem()
	}

	switch val := value.(type) {
	case string:
		if val == "" {
			return reflect.Zero(t).Interface(), nil
		}

		switch t.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			i, err := strconv.Atoi(val)
			if err != nil {
				return nil, errs.ErrInvalidSyntax
			}
			v.SetInt(int64(i))
			return v.Interface(), nil
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			i, err := strconv.Atoi(val)
			if err != nil {
				return nil, errs.ErrInvalidSyntax
			}
			v.SetUint(uint64(i))
			return v.Interface(), nil
		case reflect.Float32, reflect.Float64:
			f, err := strconv.ParseFloat(val, 64)
			if err != nil {
				return nil, errs.ErrInvalidSyntax
			}
			v.SetFloat(f)
			return v.Interface(), nil
		default:
			return nil, errors.New("invalid type")
		}
	default:
		return val, nil
	}
}

func (n *NumberWidget[T]) ValueToForm(value interface{}) interface{} {
	if value == nil {
		return value
	}

	if s, ok := value.(string); ok {
		return s
	}

	var (
		newT   = new(T)
		new_rT = reflect.TypeOf(newT)
		v      = reflect.ValueOf(value)
	)

	new_rT = new_rT.Elem()

	v = v.Convert(new_rT)

	switch new_rT.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.Itoa(int(v.Int()))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.Itoa(int(v.Uint()))
	case reflect.Float32, reflect.Float64:
		return strconv.FormatFloat(v.Float(), 'f', -1, 64)
	default:
		return value
	}
}

type DateWidgetType string

const (
	DateWidgetTypeDate     DateWidgetType = "date"
	DateWidgetTypeDateTime DateWidgetType = "datetime-local"
)

type DateWidget struct {
	*BaseWidget
	DateType DateWidgetType
}

func NewDateInput(attrs map[string]string, t DateWidgetType) Widget {
	return &DateWidget{NewBaseWidget(S(string(t)), "forms/widgets/date.html", attrs), t}
}

func (d *DateWidget) ValueToGo(value interface{}) (interface{}, error) {
	var (
		v   time.Time
		err error
	)
	switch val := value.(type) {
	case string:
		if val == "" {
			return "", nil
		}

		if d.DateType == DateWidgetTypeDate {
			v, err = time.Parse("2006-01-02", val)
		} else {
			var split = strings.Split(val, ":")
			if len(split) == 2 {
				v, err = time.Parse("2006-01-02T15:04", val)
			} else if len(split) == 3 {
				v, err = time.Parse("2006-01-02T15:04:05", val)
			} else {
				return "", errors.Wrapf(
					errs.ErrInvalidSyntax,
					"invalid date format %q", val,
				)
			}
		}
		if err != nil {
			return "", errors.Wrapf(
				errs.ErrInvalidSyntax,
				"invalid date format %q", val,
			)
		}
		return v, nil
	default:
		return val, nil
	}
}

func (d *DateWidget) ValueToForm(value interface{}) interface{} {
	if value == nil || value == "" {
		return ""
	}

	switch val := value.(type) {
	case time.Time:
		if d.DateType == DateWidgetTypeDate {
			return val.Format("2006-01-02")
		}
		return val.Format("2006-01-02T15:04:05")
	case string:
		var t, err = time.Parse("2006-01-02", val)
		if err != nil {
			return val
		}
		if d.DateType == DateWidgetTypeDate {
			return t.Format("2006-01-02")
		}
		return t.Format("2006-01-02T15:04:05")
	default:
		return value
	}
}
