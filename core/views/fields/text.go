package fields

import (
	"database/sql/driver"
	"fmt"

	"github.com/Nigel2392/go-django/core/views/interfaces"
	"github.com/Nigel2392/router/v3/request"
	"github.com/Nigel2392/tags"
)

type StringField string

func (i *StringField) Scan(src interface{}) error {
	switch src.(type) {
	case []byte:
		*i = StringField(string(src.([]byte)))
	case string:
		*i = StringField(src.(string))
	}
	return nil
}

func (i StringField) Value() (driver.Value, error) {
	return string(i), nil
}

func (i *StringField) FormValues(v []string) error {
	if len(v) == 0 {
		return nil
	}
	*i = StringField(v[0])
	return nil
}

func (i StringField) LabelHTML(_ *request.Request, name string, display_text string, tags tags.TagMap) interfaces.Element {
	return ElementType(fmt.Sprintf(`<label for="%s" %s>%s</label>`, name, TagMapToElementAttributes(tags, AllTagsLabel...), display_text))
}

func (i StringField) InputHTML(_ *request.Request, name string, tags tags.TagMap) interfaces.Element {
	return ElementType(fmt.Sprintf(`<input type="text" name="%s" id="%s" value="%s" %s>`, name, name, i, TagMapToElementAttributes(tags, AllTagsInput...)))
}

type TextField string

func (i *TextField) Scan(src interface{}) error {
	switch src.(type) {
	case []byte:
		*i = TextField(src.([]byte))
	case string:
		*i = TextField(src.(string))
	}
	return nil
}

func (i TextField) Value() (driver.Value, error) {
	return string(i), nil
}

func (i *TextField) FormValues(v []string) error {
	if len(v) == 0 {
		return nil
	}
	*i = TextField(v[0])
	return nil
}

func (i TextField) LabelHTML(_ *request.Request, name string, display_text string, tags tags.TagMap) interfaces.Element {
	return ElementType(fmt.Sprintf(`<label for="%s" %s>%s</label>`, name, TagMapToElementAttributes(tags, AllTagsLabel...), display_text))
}

func (i TextField) InputHTML(_ *request.Request, name string, tags tags.TagMap) interfaces.Element {
	return ElementType(fmt.Sprintf(`<textarea name="%s" id="%s" %s>%s</textarea>`, name, name, TagMapToElementAttributes(tags, AllTagsInput...), i))
}
