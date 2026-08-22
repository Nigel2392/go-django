package forms

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"net/url"
	"reflect"

	"github.com/Nigel2392/go-django/internal/django_reflect"
	dr "github.com/Nigel2392/go-django/internal/django_reflect"
	errsPkg "github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/src/core/assert"
	"github.com/Nigel2392/go-django/src/core/errs"
	"github.com/Nigel2392/go-django/src/core/filesystem"
	"github.com/Nigel2392/go-django/src/utils/mixins"
	"github.com/elliotchance/orderedmap/v2"
)

type IsValidDefiner interface {
	IsValid(ctx context.Context) bool
}

type IsValidChecker[T any] interface {
	CheckIsValid(ctx context.Context, formObj T) bool
}

type FormWrapper[T any] interface {
	Unwrap() []T
}

type PrevalidatorMixin interface {
	Prevalidate(ctx context.Context, root any, data url.Values, files map[string][]filesystem.FileHeader) []error
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
	ErrorDefiner
	WasCleaned() bool
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

func checkUnwrappedForms[T any](ctx context.Context, formObj FormWrapper[T]) bool {

	var valid = true
	for _, form := range formObj.Unwrap() {
		var rv = reflect.ValueOf(form)
		if rv.Kind() != reflect.Pointer {
			panic("IsValid() only accepts a pointer to a Form, not a value.")
		}
		if rv.IsNil() {
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

	if definer, ok := any(formObj).(IsValidDefiner); ok && valid {
		return definer.IsValid(ctx)
	}

	return valid

}

func IsValid[T any](ctx context.Context, formObj T) bool {

	var rv = reflect.ValueOf(formObj)
	if rv.Kind() != reflect.Pointer {
		panic("IsValid() only accepts a pointer to a Form, not a value.")
	}

	if chk, ok := any(formObj).(wasCleanedChecker); ok {
		valid, ok := checkWasCleaned(chk, func(formObj wasCleanedChecker) (valid, ok bool) {
			if isValidDef, ok := formObj.(IsValidDefiner); ok {
				return isValidDef.IsValid(ctx), true
			}
			return true, true
		})
		if ok {
			return valid
		}
	}

	var topKey = pointerContextKey{ptr: rv.Pointer()}
	var _, hasPtr = ctx.Value(topKey).(struct{})
	if unwrapper, ok := any(formObj).(FormWrapper[T]); ok && !hasPtr {
		return checkUnwrappedForms(ctx, unwrapper)
	}
	if unwrapper, ok := any(formObj).(FormWrapper[Form]); ok && !hasPtr {
		return checkUnwrappedForms(ctx, unwrapper)
	}
	if unwrapper, ok := any(formObj).(FormWrapper[any]); ok && !hasPtr {
		return checkUnwrappedForms(ctx, unwrapper)
	}

	f, ok := any(formObj).(Form)
	if !ok {
		fn, err := dr.Method[func(context.Context, any) bool](
			formObj, "CheckIsValid",
		)
		if err != nil {
			panic(fmt.Errorf("Invalid CheckIsValid function on %T: %w", formObj, err))
		}

		return fn(ctx, formObj)
	}

	var rawData, files = f.Data()
	assert.False(
		rawData == nil,
		"You cannot call IsValid() without setting the data first on %T.", f,
	)

	for mixin := range mixins.Mixins[any](f, true) {
		if prevalidator, ok := mixin.(PrevalidatorMixin); ok {
			var errors = prevalidator.Prevalidate(ctx, formObj, rawData, files)
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

	if bndErrs == nil || bndErrs.Len() == 0 {
		cleaner, ok := f.(CleanableForm)
		if ok {
			var c, validationErrs = cleaner.Clean(ctx, cleaned)
			if len(validationErrs) > 0 {
				f.AddFormError(validationErrs...)
			} else {
				cleaned = c
			}
			errs = f.ErrorList()
			bndErrs = f.BoundErrors()
		}
	}

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
		f.OnValid().Send(ctx, f)
	} else {
		f.BindCleanedData(invalid, defaults, nil)
		f.OnInvalid().Send(ctx, f)
	}

	f.OnFinalize().Send(ctx, f)

	if bndErrs != nil && bndErrs.Len() > 0 || len(errs) > 0 {
		f.BindCleanedData(invalid, defaults, nil)
	}

	errs = f.ErrorList()
	bndErrs = f.BoundErrors()
	var isValid = (bndErrs == nil || bndErrs.Len() == 0) && len(errs) == 0
	if isValidDef, ok := f.(IsValidDefiner); ok {
		return isValidDef.IsValid(ctx) && isValid
	}

	return isValid
}

func fullClean(ctx context.Context, f ErrorAdder, rawData map[string][]string, files map[string][]filesystem.FileHeader) (invalid_, defaults_, cleaned_ map[string]any) {

	var (
		base_invalid  = make(map[string]any)
		base_defaults = make(map[string]any)
		base_cleaned  = make(map[string]any)
	)

	var addError = func(mixin any, depth int, field string, errList ...error) {
		if len(errList) == 0 {
			return
		}

		if depth == 0 {
			f.AddError(field, errList...)
		} else {
			f.AddFormError(errList...)
		}

		if mixin == f {
			return
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
				panic(fmt.Errorf("Form %T does not implement FullCleanMixin", mixin))
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

		if initialGetter, ok := mixin.(interface{ InitialData() map[string]interface{} }); ok {
			var data = initialGetter.InitialData()
			maps.Copy(defaults, data)
		}

		if unsafeGetter, ok := mixin.(interface{ CleanedDataUnsafe() map[string]interface{} }); ok {
			var data = unsafeGetter.CleanedDataUnsafe()
			maps.Copy(defaults, data)
			maps.Copy(cleaned, data)
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
			} else {
				v.SetWidget(widget)
			}

			// todo: write tests
			wf, ok := widget.(WidgetFormDefiner)
			if ok {
				formFn, err := dr.Method[func() WidgetFormType](
					wf, "WidgetForm",
					dr.WithContext(ctx),
					dr.WithFuncArgs(fm),
				)
				if err != nil {
					assert.Fail(
						"%T does not have the correct WidgetForm function: %v",
						widget, err,
					)
				}

				form := formFn()
				form.SetPrefix(fm.PrefixName(k))
				form.WithContext(ctx)
				form.WithData(rawData, files, nil)

				if !IsValid(ctx, form) {
					var (
						_errList = form.ErrorList()
						_errBnd  = form.BoundErrors()
						errs     = make([]error, 0, len(_errList)+_errBnd.Len())
					)

					errs = append(errs, _errList...)
					for head := _errBnd.Front(); head != nil; head = head.Next() {
						errs = append(errs, head.Value...)
					}

					addError(mixin, depth, k, errs...)
					invalid[k] = form
					continue
				}

				data = form
				goto saveField
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
				addError(mixin, depth, k, errs.ErrFieldRequired)
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

			/*

				this allows for defining form methods to clean field values.
				i.e. form with field called "email" can implement method:

					CleanField__email(context.Context?, data) (data, error?)

			*/
			//	cleanField, err := dr.Method[func(data any) (any, error)](
			//		fm, fmt.Sprintf("CleanField__%s", head.Key),
			//		dr.WithContext(ctx),
			//	)
			//	if err == nil {
			//		data, err = cleanField(data)
			//		if err != nil {
			//			addError(mixin, depth, k, err)
			//			invalid[k] = initial
			//			continue
			//		}
			//	}

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

		saveField:
			// Check if the field is saveable and call Save() on it.
			// This might be used to save a relation to the database, among other things.
			// TODO: FIX INTERACTION WITH FileField
			if field, saveable := v.(SaveableField); saveable {
				data, err = field.Save(data)
				if err != nil {
					addError(mixin, depth, k, err)
					invalid[k] = data
					continue
				}

			}

			defaults[k] = data
			cleaned[k] = data
		}

		fm.BindCleanedData(invalid, defaults, cleaned)
	}

	return base_invalid, base_defaults, base_cleaned
}

func FormValueFromDataDict[T any](ctx context.Context, form FormFieldDefiner, name string, data url.Values, files map[string][]filesystem.FileHeader) (T, bool, []error) {
	var field, ok = form.Field(name)
	if !ok {
		return *new(T), false, []error{errsPkg.FieldNotFound.Wrapf(
			"field %q not found in form %T", name, form,
		)}
	}

	widget, ok := form.Widget(name)
	if !ok || widget == nil {
		widget = field.Widget()
	}

	if widget == nil {
		return *new(T), false, []error{errsPkg.ValueError.Wrapf(
			"field %q in form %T has no widget", name, form,
		)}
	}

	var namePrefixed = form.PrefixName(name)
	var omitted = widget.ValueOmittedFromData(ctx, data, files, namePrefixed)
	if omitted {
		return *new(T), false, nil
	}

	var value, errs = widget.ValueFromDataDict(ctx, data, files, namePrefixed)
	if len(errs) > 0 {
		return *new(T), true, errs
	}

	value, err := widget.ValueToGo(value)
	if err != nil {
		return *new(T), true, []error{err}
	}

	var _nT T
	var dstT = reflect.TypeOf(_nT)
	var rv = reflect.ValueOf(value)

	if rv.Type() == dstT {
		return rv.Interface().(T), true, nil
	}

	if rv.Type().ConvertibleTo(dstT) {
		return rv.Convert(dstT).Interface().(T), true, nil
	}

	if rv.Type().AssignableTo(dstT) {
		return rv.Interface().(T), true, nil
	}

	return *new(T), true, []error{errsPkg.TypeMismatch.Wrapf(
		"field %q in form %T: value is %T, cannot convert to %T",
		name, form, value, _nT,
	)}
}

func HasErrors(form ErrorDefiner) bool {
	var errs = form.BoundErrors()
	return errs != nil && errs.Len() > 0 || len(form.ErrorList()) > 0
}

func SaveForm(ctx context.Context, form any, args ...any) error {
	saveFn, err := dr.Method[func() error](form, "Save",
		dr.WithContext(ctx),
		dr.WithFuncArgs(args...),
	)
	if err != nil && !errors.Is(err, django_reflect.ErrMethodNotFound) {
		return err
	}

	err = saveFn()
	if a, ok := form.(ErrorAdder); ok {
		if err != nil {
			a.AddFormError(err)
		}
	}

	return err
}
