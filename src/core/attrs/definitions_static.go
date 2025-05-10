package attrs

import (
	"maps"
	"reflect"

	"github.com/Nigel2392/go-django/src/core/contenttypes"
	"github.com/Nigel2392/go-django/src/forms/fields"
	"github.com/elliotchance/orderedmap/v2"
)

var (
	_ StaticDefinitions = (*staticDefinition)(nil)
	_ FieldDefinition   = (*staticField)(nil)
)

type staticDefinition struct {
	object       Definer
	defs         Definitions
	fields       *orderedmap.OrderedMap[string, FieldDefinition]
	PrimaryField string
	Table        string
}

func newStaticDefinitions(d Definer) *staticDefinition {

	var sDef = &staticDefinition{
		object: d,
	}

	var primaryField string
	var defs = d.FieldDefs()
	var m = orderedmap.NewOrderedMap[string, FieldDefinition]()
	for _, f := range defs.Fields() {

		if f.IsPrimary() && primaryField == "" {
			primaryField = f.Name()
		}

		m.Set(f.Name(), f)
	}

	var meta = GetModelMeta(d)
	var reverseRelations = meta.ReverseMap()
	for head := reverseRelations.Front(); head != nil; head = head.Next() {
		var key = head.Key
		var value = head.Value
		var typ = value.Type()

		var (
			from           = value.From()
			fromModelField = from.Field()
		)

		var confCopy = &FieldConfig{
			NameOverride: key,
			Null:         fromModelField.AllowNull(),
			Blank:        fromModelField.AllowBlank(),
			ReadOnly:     !fromModelField.AllowEdit(),
			Primary:      fromModelField.IsPrimary(),
			Label:        fromModelField.Label(),
			HelpText:     fromModelField.HelpText(),
			Column:       fromModelField.ColumnName(),
			Attributes:   fromModelField.Attrs(),
			// Validators: ,
			// FormField:
			// WidgetAttrs:
		}

		switch typ {
		case RelOneToOne: // OneToOne
			confCopy.RelOneToOne = value
		case RelManyToOne: // ForeignKey, ForeignKey
			confCopy.RelOneToOne = value
		case RelOneToMany: // ForeignKeyReverse, ForeignKeyReverse
			confCopy.RelForeignKeyReverse = value
		case RelManyToMany: // ManyToMany
			confCopy.RelManyToMany = value
		default:
			panic("unknown relation type")
		}

		var field = &staticField{
			typ:         fromModelField.Type(),
			name:        key,
			definitions: sDef,
			conf:        confCopy,
		}

		m.Set(key, field)
	}

	sDef.defs = d.FieldDefs()
	sDef.Table = sDef.defs.TableName()
	sDef.PrimaryField = primaryField
	sDef.fields = m

	return sDef
}

func (d *staticDefinition) TableName() string {
	if d.Table != "" {
		return d.Table
	}
	return d.defs.TableName()
}

func (d *staticDefinition) Field(name string) (f FieldDefinition, ok bool) {
	f, ok = d.fields.Get(name)
	return
}

func (d *staticDefinition) Fields() []FieldDefinition {
	var m = make([]FieldDefinition, d.fields.Len())
	var i = 0
	for head := d.fields.Front(); head != nil; head = head.Next() {
		m[i] = head.Value
		i++
	}
	return m
}

func (d *staticDefinition) Primary() FieldDefinition {
	if d.PrimaryField == "" {
		return nil
	}
	f, _ := d.fields.Get(d.PrimaryField)
	return f
}

func (d *staticDefinition) Instance() Definer {
	var objectT = reflect.TypeOf(d.object)
	if objectT.Kind() == reflect.Ptr {
		objectT = objectT.Elem()
	}
	return reflect.New(objectT).Interface().(Definer)
}

func (d *staticDefinition) Len() int {
	return d.fields.Len()
}

type staticField struct {
	typ         reflect.Type
	name        string
	definitions *staticDefinition
	conf        *FieldConfig
	formField   fields.Field
}

func (s *staticField) Tag(name string) string {
	return ""
}

func (s *staticField) Instance() Definer {
	return s.definitions.Instance()
}

func (s *staticField) Name() string {
	if s.conf.NameOverride != "" {
		return s.conf.NameOverride
	}
	return s.name
}

func (s *staticField) ColumnName() string {
	return s.conf.Column
}

func (s *staticField) Type() reflect.Type {
	return s.typ
}

func (s *staticField) Attrs() map[string]any {
	var attrs = make(map[string]interface{})
	attrs[AttrNameKey] = s.Name()
	attrs[AttrMaxLengthKey] = s.conf.MaxLength
	attrs[AttrMinLengthKey] = s.conf.MinLength
	attrs[AttrMinValueKey] = s.conf.MinValue
	attrs[AttrMaxValueKey] = s.conf.MaxValue
	attrs[AttrAllowNullKey] = s.AllowNull()
	attrs[AttrAllowBlankKey] = s.AllowBlank()
	attrs[AttrAllowEditKey] = s.AllowEdit()
	attrs[AttrIsPrimaryKey] = s.IsPrimary()
	maps.Copy(attrs, s.conf.Attributes)
	return attrs

}

func (s *staticField) Rel() Relation {
	switch {
	case s.conf.RelForeignKey != nil:
		return s.conf.RelForeignKey
	case s.conf.RelManyToMany != nil:
		return s.conf.RelManyToMany
	case s.conf.RelOneToOne != nil:
		return s.conf.RelOneToOne
	case s.conf.RelForeignKeyReverse != nil:
		return s.conf.RelForeignKeyReverse
	}
	return nil
}

func (s *staticField) Label() string {
	return s.conf.Label
}

func (s *staticField) HelpText() string {
	return s.conf.HelpText
}

func (s *staticField) IsPrimary() bool {
	return s.conf.Primary
}

func (s *staticField) AllowNull() bool {
	return s.conf.Null
}

func (s *staticField) AllowBlank() bool {
	return s.conf.Blank
}

func (s *staticField) AllowEdit() bool {
	return s.conf.ReadOnly
}

func (s *staticField) FormField() fields.Field {
	if s.formField != nil {
		return s.formField
	}

	var opts = make([]func(fields.Field), 0)
	var rel = s.Rel()
	if rel != nil {
		var cTypeDef = contenttypes.DefinitionForObject(rel)
		if cTypeDef != nil {
			opts = append(opts, fields.Label(
				cTypeDef.Label(),
			))
		}
	} else {
		opts = append(opts, fields.Label(s.Label))
	}

	opts = append(opts, fields.HelpText(s.HelpText))

	if s.conf.ReadOnly {
		opts = append(opts, fields.ReadOnly(true))
	}

	if !s.AllowBlank() {
		opts = append(opts, fields.Required(true))
	}

	if s.conf.FormField != nil {
		return s.conf.FormField(opts...)
	}

	return nil
}

func (s *staticField) Validate() error {
	if s.conf.Validators == nil {
		return nil
	}
	for _, v := range s.conf.Validators {
		if err := v(s); err != nil {
			return err
		}
	}
	return nil
}
