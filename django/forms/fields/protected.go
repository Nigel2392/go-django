package fields

import "github.com/Nigel2392/django/core/errs"

var _ Field = (*ProtectedFormField)(nil)

// Translates errors which might have too much information into
// a generic error message.
type ProtectedFormField struct {
	Field
	ErrorMessage func(err error) error
}

func Protect(w Field, errFn func(err error) error) *ProtectedFormField {
	if errFn == nil {
		errFn = func(err error) error {
			return err
		}
	}
	return &ProtectedFormField{
		Field:        w,
		ErrorMessage: errFn,
	}
}

func (pw *ProtectedFormField) ValueToGo(value interface{}) (interface{}, error) {
	var val, err = pw.Field.ValueToGo(value)
	if err != nil {
		return nil, pw.ErrorMessage(err)
	}
	return val, nil
}

func (pw *ProtectedFormField) Clean(value interface{}) (interface{}, error) {
	var val, err = pw.Field.Clean(value)
	if err != nil {
		return nil, pw.ErrorMessage(err)
	}
	return val, nil
}

func (pw *ProtectedFormField) Validate(value interface{}) []error {
	var errors = pw.Field.Validate(value)
	if len(errors) != 0 {
		var merged = pw.ErrorMessage(
			errs.NewMultiError(errors...),
		)
		return []error{pw.ErrorMessage(merged)}
	}
	return nil
}
