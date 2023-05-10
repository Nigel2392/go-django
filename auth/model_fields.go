package auth

import (
	"database/sql/driver"
	"errors"
	"fmt"
	"net/mail"

	"github.com/Nigel2392/go-django/core/views/fields"
	"github.com/Nigel2392/go-django/core/views/interfaces"
	"github.com/Nigel2392/router/v3/request"
	"github.com/Nigel2392/tags"
)

type EmailField fields.StringField

func (i *EmailField) Scan(src interface{}) error {
	return (*fields.StringField)(i).Scan(src)
}

func (i EmailField) Value() (driver.Value, error) {
	return (fields.StringField)(i).Value()
}

func (i *EmailField) FormValues(v []string) error {
	return (*fields.StringField)(i).FormValues(v)
}

func (i EmailField) LabelHTML(_ *request.Request, name string, display_text string, tags tags.TagMap) interfaces.Element {
	return fields.ElementType(fmt.Sprintf(`<label for="%s" %s>%s</label>`, name, fields.TagMapToElementAttributes(tags, fields.AllTagsLabel...), display_text))
}

func (i EmailField) InputHTML(_ *request.Request, name string, tags tags.TagMap) interfaces.Element {
	return fields.ElementType(fmt.Sprintf(`<input type="email" name="%s" id="%s" value="%s" %s>`, name, name, i, fields.TagMapToElementAttributes(tags, fields.AllTagsInput...)))
}

func (i EmailField) Validate() error {
	if string(i) == "" {
		return nil
	}
	var _, err = mail.ParseAddress(string(i))
	return errIf(err, "Invalid email address")
}

func errIf(err error, msg ...string) error {
	if err != nil {
		if len(msg) > 0 {
			return errors.New(msg[0])
		} else {
			return err
		}
	}
	return nil
}

type PasswordField string

func (i *PasswordField) Scan(src interface{}) error {
	switch src.(type) {
	case []byte:
		*i = PasswordField(string(src.([]byte)))
	case string:
		*i = PasswordField(src.(string))
	}
	return nil
}

func (i PasswordField) Value() (driver.Value, error) {
	return string(i), nil
}

func (i *PasswordField) FormValues(v []string) error {
	if len(v) == 0 {
		return nil
	}
	var value = v[0]

	if string(*i) == value && value != "" {
		return nil
	}

	if !IS_HASHED(value) {
		var err error
		value, err = HASHER(value)
		if err != nil {
			return err
		}

		// Place it here to only hash it when it is not already hashed
		// *i = PasswordField(value)
	}

	// Place it here to override the password with a custom hash if it matches the format.
	*i = PasswordField(value)
	return nil
}

func (i PasswordField) LabelHTML(_ *request.Request, name string, display_text string, tags tags.TagMap) interfaces.Element {
	return fields.ElementType(fmt.Sprintf(`<label for="%s" %s>%s</label>`, name, fields.TagMapToElementAttributes(tags, fields.AllTagsLabel...), display_text))
}

func (i PasswordField) InputHTML(_ *request.Request, name string, tags tags.TagMap) interfaces.Element {
	return fields.ElementType(fmt.Sprintf(`<input type="password" name="%s" id="%s" value="%s" %s>`, name, name, i, fields.TagMapToElementAttributes(tags, fields.AllTagsInput...)))
}
