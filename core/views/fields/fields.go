package fields

import (
	"database/sql/driver"
	"fmt"
	"strconv"
	"time"

	"github.com/Nigel2392/go-django/core/httputils/tags"
	"github.com/Nigel2392/go-django/core/views/interfaces"
	"github.com/Nigel2392/router/v3/request"
)

type BoolField bool

func (i *BoolField) Scan(src interface{}) error {
	*i = BoolField(src.(bool))
	return nil
}

func (i BoolField) Value() (driver.Value, error) {
	return bool(i), nil
}

func (i *BoolField) FormValues(v []string) error {
	if len(v) == 0 {
		return nil
	}
	var intie, err = strconv.ParseBool(v[0])
	if err != nil {
		return err
	}
	*i = BoolField(intie)
	return nil
}

func (i BoolField) LabelHTML(_ *request.Request, name string, tags tags.TagMap) interfaces.Element {
	return ElementType(fmt.Sprintf(`<label for="%s" %s>%s</label>`, name, TagMapToElementAttributes(tags, AllTagsLabel...), name))
}

func (i BoolField) InputHTML(_ *request.Request, name string, tags tags.TagMap) interfaces.Element {
	if i {
		return ElementType(fmt.Sprintf(`<input type="checkbox" name="%s" checked %s>`, name, TagMapToElementAttributes(tags, AllTagsInput...)))
	}
	return ElementType(fmt.Sprintf(`<input type="checkbox" name="%s" %s>`, name, TagMapToElementAttributes(tags, AllTagsInput...)))
}

type DateField time.Time

func (i *DateField) Scan(src interface{}) error {
	var t, err = time.Parse("2006-01-02", src.(string))
	if err != nil {
		return err
	}
	*i = DateField(t)
	return nil
}

func (i DateField) Value() (driver.Value, error) {
	return time.Time(i).Format("2006-01-02"), nil
}

func (i *DateField) FormValues(v []string) error {
	if len(v) == 0 {
		return nil
	}
	var intie, err = time.Parse("2006-01-02", v[0])
	if err != nil {
		return err
	}
	*i = DateField(intie)
	return nil
}

func (i DateField) LabelHTML(_ *request.Request, name string, tags tags.TagMap) interfaces.Element {
	return ElementType(fmt.Sprintf(`<label for="%s" %s>%s</label>`, name, TagMapToElementAttributes(tags, AllTagsLabel...), name))
}

func (i DateField) InputHTML(_ *request.Request, name string, tags tags.TagMap) interfaces.Element {
	return ElementType(fmt.Sprintf(`<input type="date" name="%s" value="%s" %s>`, name, time.Time(i).Format("2006-01-02"), TagMapToElementAttributes(tags, AllTagsInput...)))
}

type DateTimeField time.Time

func (i *DateTimeField) Scan(src interface{}) error {
	var t, err = time.Parse("2006-01-02 15:04:05", src.(string))
	if err != nil {
		return err
	}
	*i = DateTimeField(t)
	return nil
}

func (i DateTimeField) Value() (driver.Value, error) {
	return time.Time(i).Format("2006-01-02T15:04"), nil
}

func (i *DateTimeField) FormValues(v []string) error {
	if len(v) == 0 {
		return nil
	}
	var intie, err = time.Parse("2006-01-02T15:04", v[0])
	if err != nil {
		return err
	}
	*i = DateTimeField(intie)
	return nil
}

func (i DateTimeField) LabelHTML(_ *request.Request, name string, tags tags.TagMap) interfaces.Element {
	return ElementType(fmt.Sprintf(`<label for="%s" %s>%s</label>`, name, TagMapToElementAttributes(tags, AllTagsLabel...), name))
}

func (i DateTimeField) InputHTML(_ *request.Request, name string, tags tags.TagMap) interfaces.Element {
	return ElementType(fmt.Sprintf(`<input type="datetime-local" name="%s" value="%s" %s>`, name, time.Time(i).Format("2006-01-02T15:04"), TagMapToElementAttributes(tags, AllTagsInput...)))
}
