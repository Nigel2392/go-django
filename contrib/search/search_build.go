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

func BuildSearchFields[T attrs.Definer](model T, fields []SearchField) ([]BuiltSearchField, error) {
	if len(fields) > 0 {
		return buildSearchFields(model, fields)
	}

	if s, ok := any(model).(SearchableModel); ok {
		return buildSearchFields(s, s.SearchableFields())
	}

	return nil, nil
}

func buildSearchFields[T attrs.Definer](model T, fields []SearchField) ([]BuiltSearchField, error) {
	var meta = attrs.GetModelMeta(model)
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
