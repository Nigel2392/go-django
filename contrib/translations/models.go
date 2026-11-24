package translations

import (
	"context"
	"database/sql/driver"
	"fmt"

	"github.com/Nigel2392/go-django/queries/src/drivers/dbtype"
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/attrs/fattrs"
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

type Locale struct {
	LanguageTag LanguageTag
}

func (m *Locale) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make(ctx, m,
		fattrs.Field(m, "LanguageTag", &m.LanguageTag, func() fattrs.PtrFieldConfig[*Locale, LanguageTag] {
			return fattrs.PtrFieldConfig[*Locale, LanguageTag]{
				Config: attrs.Config{
					Column:    "language_id",
					Primary:   true,
					MaxLength: maxLanguageTagLength,
				},
			}
		}),
	)
}
