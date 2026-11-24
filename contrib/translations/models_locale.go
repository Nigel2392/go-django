package translations

import (
	"context"

	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/attrs/fattrs"
)

type Locale struct {
	LanguageTag LanguageTag
}

func (m *Locale) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make(ctx, m,
		fattrs.Field(m, "LanguageTag", &m.LanguageTag, func() fattrs.PtrFieldConfig[*Locale, LanguageTag] {
			return fattrs.PtrFieldConfig[*Locale, LanguageTag]{
				Column:    "language_id",
				Primary:   true,
				MaxLength: maxLanguageTagLength,
			}
		}),
	)
}
