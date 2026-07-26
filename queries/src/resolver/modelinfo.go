package resolver

import (
	"fmt"

	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/src/core/attrs"
)

// Basic information about the model used in the QuerySet.
// It contains the model's meta information, primary key field, all fields,
// and the table name.
type ModelInfo struct {
	Primary  attrs.FieldDefinition
	Object   attrs.Definer
	Fields   []attrs.FieldDefinition
	Table    string
	Ordering []string
}

func NewModelInfo(model attrs.Definer) (ModelInfo, error) {
	var (
		meta        = attrs.GetModelMeta(model)
		definitions = meta.Definitions()
		primary     = definitions.Primary()
		tableName   = definitions.TableName()
	)

	if tableName == "" {
		return ModelInfo{}, errors.NoTableName.WithCause(fmt.Errorf(
			"model %T has no table name", model,
		))
	}

	var orderBy []string
	if ord, ok := any(model).(OrderByDefiner); ok {
		orderBy = ord.OrderBy()
	}

	// specifically check for nil instead of len() == 0
	// a model might override and return and empty list of strings, like `[]string{}`
	if orderBy == nil && primary != nil {
		orderBy = []string{primary.Name()}
	}

	info := ModelInfo{
		Primary:  primary,
		Object:   model,
		Fields:   definitions.Fields(),
		Table:    tableName,
		Ordering: orderBy,
	}

	return info, nil
}

func (m ModelInfo) Model() attrs.Definer {
	return m.Object
}

func (m ModelInfo) TableName() string {
	return m.Table
}

func (m ModelInfo) PrimaryKey() attrs.FieldDefinition {
	return m.Primary
}

func (m ModelInfo) OrderBy() []string {
	return m.Ordering
}
