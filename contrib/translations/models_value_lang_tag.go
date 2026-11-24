package translations

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"github.com/Nigel2392/go-django/queries/src/drivers/dbtype"
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"golang.org/x/text/language"
)

type LanguageTag language.Tag

const maxLanguageTagLength = 255

func (LanguageTag) DBType() dbtype.Type {
	return dbtype.String
}

func (l LanguageTag) String() string {
	return language.Tag(l).String()
}

func (l LanguageTag) MarshalJSON() ([]byte, error) {
	return json.Marshal(l.String())
}

func (l *LanguageTag) UnmarshalJSON(src []byte) error {
	var s string
	if err := json.Unmarshal(src, &s); err != nil {
		return err
	}

	tag, err := language.Parse(s)
	if err != nil {
		return err
	}

	*l = (LanguageTag)(tag)
	return nil
}

func (l LanguageTag) Value() (driver.Value, error) {
	tag := l.String()
	if len(tag) > maxLanguageTagLength {
		return nil, fmt.Errorf("language tag %q exceeds %d characters", tag, maxLanguageTagLength)
	}
	return tag, nil
}

func (l *LanguageTag) Scan(src any) error {
	var value string
	switch s := src.(type) {
	case nil:
		return nil
	case string:
		value = s
	case []byte:
		value = string(s)
	default:
		return errors.TypeMismatch.Wrapf("%T is not of type string or []byte", src)
	}
	tag, err := language.Parse(value)
	if err != nil {
		return fmt.Errorf("invalid language tag %q: %w", value, err)
	}
	*l = LanguageTag(tag)
	return nil
}
