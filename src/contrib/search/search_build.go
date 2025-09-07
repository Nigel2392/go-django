package search

import (
	"strings"

	"github.com/Nigel2392/go-django/src/core/attrs"
)

type BuiltSearchField struct {
	ModelField  attrs.FieldDefinition
	ModelMeta   attrs.ModelMeta
	SearchField SearchField
}

func BuildSearchFields[T SearchableModel](model T) ([]BuiltSearchField, error) {
	var meta = attrs.GetModelMeta(model)
	var fields = model.SearchableFields()
	var built = make([]BuiltSearchField, 0, len(fields))
	for _, field := range fields {
		var fieldPath = strings.Split(field.Field(), ".")
		var res, err = attrs.WalkMetaFields(model, fieldPath)
		if err != nil {
			return nil, err
		}

		built = append(built, BuiltSearchField{
			ModelField:  res.Last().Field,
			ModelMeta:   meta,
			SearchField: field,
		})
	}

	return built, nil
}
