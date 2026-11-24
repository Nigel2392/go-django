package translations

import (
	"context"

	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/attrs/fattrs"
)

type TranslatableMixin struct {
	TranslationKey drivers.ULID
	Locale         *Locale
}

func (m *TranslatableMixin) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make(ctx, m,
		fattrs.Field(m, "TranslationKey", &m.TranslationKey, func() fattrs.PtrFieldConfig[*TranslatableMixin, drivers.ULID] {
			return fattrs.PtrFieldConfig[*TranslatableMixin, drivers.ULID]{
				Column: "translation_key",
				Default: func() drivers.ULID {
					return drivers.NewULID()
				},
			}
		}),
		fattrs.Field(m, "Locale", &m.Locale, func() fattrs.PtrFieldConfig[*TranslatableMixin, *Locale] {
			return fattrs.PtrFieldConfig[*TranslatableMixin, *Locale]{
				Column:    "locale_id",
				MaxLength: maxLanguageTagLength,
			}
		}),
	)
}
