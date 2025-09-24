package forms

import (
	"context"
	"reflect"

	"github.com/Nigel2392/go-django/src/core/assert"
	"github.com/Nigel2392/go-django/src/core/errs"
	"github.com/Nigel2392/go-django/src/core/filesystem"
	"github.com/Nigel2392/go-django/src/utils/mixins"
	"github.com/elliotchance/orderedmap/v2"
)

type IsValidDefiner interface {
	IsValid() bool
}

type FormWrapper interface {
	Unwrap() []Form
}

type PrevalidatorMixin interface {
	Prevalidate(ctx context.Context, root Form) []error
}

type ValidatorMixin interface {
	Validators() []func(ctx context.Context, root Form) []error
}

type FullCleanMixin interface {
	Widget(name string) (Widget, bool)
	PrefixName(fieldName string) string
	FieldMap() *orderedmap.OrderedMap[string, Field]

	// BindCleanedData might be called multiple times for a single IsValid() call
	BindCleanedData(invalid, defaults, cleaned map[string]interface{})
}

func FullClean(ctx context.Context, f Form) (invalid, defaults, cleaned map[string]any) {
	var rawData, files = f.Data()
	return fullClean(ctx, f, rawData, files)
}

type pointerContextKey struct {
	ptr uintptr
}

type wasCleanedChecker interface {
	WasCleaned() bool
	ErrorList() []error
	BoundErrors() *orderedmap.OrderedMap[string, []error]
}

func checkWasCleaned(f wasCleanedChecker, finalChk func(formObj wasCleanedChecker) (valid, ok bool)) (valid, ok bool) {
	if !f.WasCleaned() {
		return false, false
	}

	var errorList = f.ErrorList()
	if len(errorList) > 0 {
		return false, true
	}

	var boundErrors = f.BoundErrors()
	if boundErrors != nil && boundErrors.Len() > 0 {
		return false, true
	}

	return finalChk(f)
}

func IsValid(ctx context.Context, formObj any) bool {

	var rv = reflect.ValueOf(formObj)
	if rv.Kind() != reflect.Pointer {
		panic("IsValid() only accepts a pointer to a Form, not a value.")
	}

	valid, ok := checkWasCleaned(formObj.(wasCleanedChecker), func(formObj wasCleanedChecker) (valid, ok bool) {
		if isValidDef, ok := formObj.(IsValidDefiner); ok {
			return isValidDef.IsValid(), true
		}
		return true, true
	})
	if ok {
		return valid
	}

	var topKey = pointerContextKey{ptr: rv.Pointer()}
	var _, hasPtr = ctx.Value(topKey).(struct{})
	if unwrapper, ok := formObj.(FormWrapper); ok && !hasPtr {
		valid = true
		for _, form := range unwrapper.Unwrap() {
			if form == nil {
				continue
			}

			// create a unique key for every form based on its pointer address
			// so we don't get stuck in an infinite loop if the same form is included in the unwrap chain
			var wrappedFormKey = pointerContextKey{
				ptr: reflect.ValueOf(form).Pointer(),
			}

			// make sure every form wrapped still gets cleaned and validated
			// by using the & operator on isValid
			valid = valid && IsValid(
				context.WithValue(ctx, wrappedFormKey, struct{}{}),
				form,
			)
		}

		if definer, ok := formObj.(IsValidDefiner); ok && valid {
			return definer.IsValid()
		}

		return valid
	}

	var f = formObj.(Form)
	var rawData, files = f.Data()
	assert.False(
		rawData == nil,
		"You cannot call IsValid() without setting the data first.",
	)

	for mixin := range mixins.Mixins[any](f, true) {
		if prevalidator, ok := mixin.(PrevalidatorMixin); ok {
			var errors = prevalidator.Prevalidate(ctx, f)
			if len(errors) > 0 {
				f.AddFormError(errors...)
			}
		}
	}

	var (
		invalid, defaults, cleaned = fullClean(ctx, f, rawData, files)
		errs                       = f.ErrorList()
		bndErrs                    = f.BoundErrors()
	)

	var hasErrors bool
	if bndErrs == nil || bndErrs.Len() == 0 {
		for _, validator := range f.Validators() {
			var errors = validator(f, cleaned)
			if len(errors) > 0 {
				f.AddFormError(errors...)
			}
		}

		errs = f.ErrorList()
		bndErrs = f.BoundErrors()
		hasErrors = (bndErrs != nil && bndErrs.Len() > 0) || len(errs) > 0
		if hasErrors {
			goto postValidateErrCheck
		}

	loopMixins:
		for mixin := range mixins.Mixins[any](f, true) {
			cleaner, ok := mixin.(ValidatorMixin)
			if !ok {
				continue
			}

			for _, validator := range cleaner.Validators() {
				var errs = validator(ctx, f)
				if len(errs) > 0 {
					hasErrors = true
					f.AddFormError(errs...)
				}
			}

			if hasErrors {
				break loopMixins
			}
		}
	}

postValidateErrCheck:
	errs = f.ErrorList()
	bndErrs = f.BoundErrors()

	if (bndErrs == nil || bndErrs.Len() == 0) && len(errs) == 0 {

		for _, fn := range f.CallbackOnValid() {
			fn(f)
		}
	} else {

		f.BindCleanedData(invalid, defaults, nil)
		for _, fn := range f.CallbackOnInvalid() {
			fn(f)
		}
	}

	for _, fn := range f.CallbackOnFinalize() {
		fn(f)
	}

	if bndErrs != nil && bndErrs.Len() > 0 || len(errs) > 0 {
		f.BindCleanedData(invalid, defaults, nil)
	}

	errs = f.ErrorList()
	bndErrs = f.BoundErrors()
	if (bndErrs == nil || bndErrs.Len() == 0) && len(errs) == 0 {
		if isValidDef, ok := f.(IsValidDefiner); ok {
			return isValidDef.IsValid()
		}
		return true
	}
	return false
}

func fullClean(ctx context.Context, f ErrorAdder, rawData map[string][]string, files map[string][]filesystem.FileHeader) (invalid_, defaults_, cleaned_ map[string]any) {

	var (
		base_invalid  = make(map[string]any)
		base_defaults = make(map[string]any)
		base_cleaned  = make(map[string]any)
	)

	var addError = func(mixin any, depth int, field string, errList ...error) {
		if depth == 0 {
			f.AddError(field, errList...)
		} else {
			f.AddFormError(errList...)
		}

		if add, ok := mixin.(ErrorAdder); ok {
			add.AddError(field, errList...)
			return
		}
	}

	for mixin, depth := range mixins.Mixins[any](f, false) {
		fm, ok := mixin.(FullCleanMixin)
		if !ok {
			if depth == 0 {
				panic("Form does not implement FullCleanMixin")
			}
			continue
		}

		var (
			invalid  map[string]any
			defaults map[string]any
			cleaned  map[string]any
			err      error
		)

		if depth == 0 {
			invalid = base_invalid
			defaults = base_defaults
			cleaned = base_cleaned
		} else {
			invalid = make(map[string]any)
			defaults = make(map[string]any)
			cleaned = make(map[string]any)
		}

		for head := fm.FieldMap().Front(); head != nil; head = head.Next() {
			var (
				k       = head.Key
				v       = head.Value
				errors  []error
				initial interface{}
				data    interface{}
			)

			if v.ReadOnly() {
				continue
			}

			var widget, ok = fm.Widget(k)
			if !ok {
				widget = v.Widget()
			}

			if !widget.ValueOmittedFromData(ctx, rawData, files, fm.PrefixName(k)) {
				initial, errors = widget.ValueFromDataDict(ctx, rawData, files, fm.PrefixName(k))
			}

			if len(errors) > 0 {
				addError(mixin, depth, k, errors...)
				invalid[k] = initial
				continue
			}

			if v.Required() && v.IsEmpty(initial) {
				addError(mixin, depth, k, errs.NewValidationError(k, errs.ErrFieldRequired))
				invalid[k] = initial
				continue
			}

			data, err = v.ValueToGo(initial)
			if err != nil {
				addError(mixin, depth, k, err)
				invalid[k] = initial
				continue
			}

			// Set the initial value again in case the value was modified by ValueToGo.
			// This is important so we add the right value to the invalid defaults.
			initial = data

			data, err = v.Clean(ctx, initial)
			if err != nil {
				addError(mixin, depth, k, err)
				invalid[k] = initial
				continue
			}

			errors = v.Validate(ctx, data)
			if len(errors) > 0 {
				var errList = make([]error, 0, len(errors))
				for _, err := range errors {
					switch e := err.(type) {
					case interface{ Unwrap() []error }:
						errList = append(errList, e.Unwrap()...)
					default:
						errList = append(errList, err)
					}
				}

				addError(mixin, depth, k, errList...)
				invalid[k] = data
				continue
			}

			defaults[k] = data
			cleaned[k] = data
		}

		fm.BindCleanedData(invalid, defaults, cleaned)
	}

	return base_invalid, base_defaults, base_cleaned
}
