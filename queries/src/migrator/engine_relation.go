package migrator

import (
	"encoding/json"

	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
	"github.com/pkg/errors"
)

type (
	// Action is the action to take when the target model is deleted or updated.
	Action int
)

func (a Action) String() string {
	var action, ok = actions_map[a]
	if !ok {
		return "<ACTION_UNKNOWN>"
	}
	return action
}

const (
	// OnDeleteCascade is the action to cascade the delete to the target model.
	CASCADE Action = iota
	// OnDeleteRestrict is the action to restrict the delete of the target model.
	RESTRICT
	// OnDeleteSetNull is the action to set the field to null when the target model is deleted.
	SET_NULL

	// not yet supported:
	//	// OnDeleteSetDefault is the action to set the field to the default value when the target model is deleted.
	//	SET_DEFAULT
)

var actions_map = map[Action]string{
	SET_NULL: "SET NULL",
	CASCADE:  "CASCADE",
	RESTRICT: "RESTRICT",
	// SET_DEFAULT: "SET DEFAULT",
}

type migrationRelationSerialized struct {
	Type        attrs.RelationType        `json:"type"`                // The type of the relation
	TargetModel *MigrationModel           `json:"model"`               // The target model of the relation
	TargetField string                    `json:"field,omitempty"`     // The field in the target model
	Through     *MigrationRelationThrough `json:"through,omitempty"`   // The through model of the relation
	OnDelete    Action                    `json:"on_delete,omitempty"` // The on delete action of the relation
	OnUpdate    Action                    `json:"on_update,omitempty"` // The on update action of the relation
}

type MigrationRelationThrough struct {
	Model       *MigrationModel `json:"model"`        // The through model of the relation
	SourceField string          `json:"source_field"` // The source field of the relation
	TargetField string          `json:"target_field"` // The target field of the relation
}

type MigrationModel struct {
	CType        *contenttypes.BaseContentType[attrs.Definer] `json:"content_type"`   // The content type of the model
	LazyModelKey string                                       `json:"lazy_model_key"` // The lazy model key of the model, in case it is a lazy relation.

	def *contenttypes.ContentTypeDefinition `json:"-"`
}

func (m *MigrationModel) Equals(other *MigrationModel) bool {
	if m == nil || other == nil {
		return m == other
	}
	if m.LazyModelKey != "" && other.LazyModelKey != "" {
		return m.LazyModelKey == other.LazyModelKey
	}
	if m.CType != nil && other.CType != nil {
		return m.CType.TypeName() == other.CType.TypeName()
	}
	return false
}

func (m *MigrationModel) Object() attrs.Definer {
	if m.CType == nil {
		if m.LazyModelKey == "" {
			panic("MigrationModel has no ContentType or LazyModelKey set")
		}
		m.def = contenttypes.LoadModel(m.LazyModelKey)
		return m.def.ContentType().New().(attrs.Definer)
	}
	return m.CType.New()
}

func (m *MigrationModel) ContentType() contenttypes.ContentType {
	if m.def != nil {
		return m.def.ContentType()
	}

	if m.LazyModelKey != "" {
		m.def = contenttypes.LoadModel(m.LazyModelKey)
		return m.def.ContentType()
	}

	if m.CType == nil {
		panic("MigrationModel has no ContentType or LazyModelKey set")
	}
	return contenttypes.ChangeBaseType[attrs.Definer, any](m.CType)
}

type MigrationRelation struct {
	Type        attrs.RelationType        `json:"type"`                // The type of the relation
	TargetModel *MigrationModel           `json:"model"`               // The target model of the relation
	TargetField attrs.FieldDefinition     `json:"field,omitempty"`     // The field in the target model
	Through     *MigrationRelationThrough `json:"through,omitempty"`   // The through model of the relation
	OnDelete    Action                    `json:"on_delete,omitempty"` // The on delete action of the relation
	OnUpdate    Action                    `json:"on_update,omitempty"` // The on update action of the relation
}

func (m *MigrationRelation) Model() attrs.Definer {
	if m.TargetModel == nil {
		return nil
	}
	return m.TargetModel.Object()
}

func (m *MigrationRelation) IsLazy() bool {
	if m.TargetModel == nil {
		return false
	}
	return m.TargetModel.LazyModelKey != ""
}

func (m *MigrationRelation) Field() attrs.FieldDefinition {
	return m.TargetField
}

func (m *MigrationRelation) MarshalJSON() ([]byte, error) {
	var targetField string
	if m.TargetField != nil {
		targetField = m.TargetField.Name()
	}
	var rel = migrationRelationSerialized{
		Type:        m.Type,
		Through:     m.Through,
		TargetModel: m.TargetModel,
		TargetField: targetField,
		OnDelete:    m.OnDelete,
		OnUpdate:    m.OnUpdate,
	}
	return json.Marshal(rel)
}

func (m *MigrationRelation) UnmarshalJSON(data []byte) error {
	var rel migrationRelationSerialized
	if err := json.Unmarshal(data, &rel); err != nil {
		return errors.Wrap(
			err, "error unmarshalling relation",
		)
	}

	var obj = rel.TargetModel.Object()
	if obj == nil {
		return nil
	}

	if rel.TargetField != "" {
		var defs = obj.FieldDefs()
		var field, ok = defs.Field(rel.TargetField)
		if !ok {
			return nil
		}
		m.TargetField = field
	}

	m.Type = rel.Type
	m.TargetModel = rel.TargetModel
	m.Through = rel.Through
	m.OnDelete = rel.OnDelete
	m.OnUpdate = rel.OnUpdate

	return nil
}
