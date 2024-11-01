package fields

import (
	"fmt"
	"regexp"

	"github.com/Nigel2392/go-django/src/core/assert"
	"github.com/Nigel2392/go-django/src/core/errs"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/go-django/src/forms/widgets"
)

func getTextFn(v any) func() string {
	var fn func() string
	switch v := v.(type) {
	case string:
		fn = trans.S(v)
	case func() string:
		fn = v
	}
	return fn
}

func Label(label any) func(Field) {
	var fn = getTextFn(label)

	assert.Truthy(fn,
		"FieldLabel: invalid type %T", label,
	)

	return func(f Field) {
		f.SetLabel(fn)
	}
}

func HelpText(helpText any) func(Field) {
	var fn = getTextFn(helpText)

	assert.Truthy(fn,
		"FieldHelpText: invalid type %T", helpText,
	)

	return func(f Field) {
		f.SetHelpText(fn)
	}
}

func Name(name string) func(Field) {
	return func(f Field) {
		f.SetName(name)
	}
}

func Required(b bool) func(Field) {
	return func(f Field) {
		if b {
			f.SetAttrs(map[string]string{"required": ""})
			f.SetRequired(b)
		}
	}
}

func ReadOnly(b bool) func(Field) {
	return func(f Field) {
		f.SetAttrs(map[string]string{"readonly": ""})
		f.SetReadOnly(b)
	}
}

const (
	ErrRegexInvalid = errs.Error("regex does not match")
)

func Regex(regex string) func(Field) {
	var rex = regexp.MustCompile(regex)
	return func(f Field) {
		f.SetValidators(func(value interface{}) error {
			if value == nil {
				return nil
			}
			var v = fmt.Sprintf("%v", value)
			if !rex.MatchString(v) {
				return fmt.Errorf("Invalid value %q (%w \"%s\")", v, ErrRegexInvalid, regex) //lint:ignore ST1005 ignore this lint
			}
			return nil
		})
	}
}

func MinLength(min int) func(Field) {
	return func(f Field) {
		f.SetAttrs(map[string]string{"minlength": fmt.Sprintf("%d", min)})
		f.SetValidators(func(value interface{}) error {
			if value == nil || value == "" {
				if min > 0 {
					return fmt.Errorf("Ensure this value has at least %d characters (%w).", min, errs.ErrLengthMin) //lint:ignore ST1005 ignore this lint
				}
				return nil
			}
			var v = fmt.Sprintf("%v", value)
			if len(v) < min {
				return fmt.Errorf("Ensure this value has at least %d characters (it has %d, %w).", min, len(v), errs.ErrLengthMin) //lint:ignore ST1005 ignore this lint
			}
			return nil
		})
	}
}

func MaxLength(max int) func(Field) {
	return func(f Field) {
		f.SetAttrs(map[string]string{"maxlength": fmt.Sprintf("%d", max)})
		f.SetValidators(func(value interface{}) error {
			if value == nil || value == "" {
				return nil
			}
			var v = fmt.Sprintf("%v", value)
			if len(v) > max {
				return fmt.Errorf("Ensure this value has at most %d characters (it has %d, %w).", max, len(v), errs.ErrLengthMax) //lint:ignore ST1005 ignore this lint
			}
			return nil
		})
	}
}

func Widget(w widgets.Widget) func(Field) {
	return func(f Field) {
		f.SetWidget(w)
	}
}

func Hide(b bool) func(Field) {
	return func(f Field) {
		f.Hide(b)
	}
}

func Validators(validators ...func(interface{}) error) func(Field) {
	return func(f Field) {
		f.SetValidators(validators...)
	}
}
