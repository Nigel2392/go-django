package migrator

import (
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/Nigel2392/go-django/queries/internal"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/drivers/dbtype"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
)

type Column struct {
	Table        Table              `json:"-"`
	Field        attrs.Field        `json:"-"`
	Name         string             `json:"name"`
	Column       string             `json:"column"`
	UseInDB      bool               `json:"use_in_db,omitempty"`
	MinLength    int64              `json:"min_length,omitempty"`
	MaxLength    int64              `json:"max_length,omitempty"`
	MinValue     float64            `json:"min_value,omitempty"`
	MaxValue     float64            `json:"max_value,omitempty"`
	Precision    int64              `json:"precision,omitempty"`
	Scale        int64              `json:"scale,omitempty"` // for decimal types
	Unique       bool               `json:"unique,omitempty"`
	Nullable     bool               `json:"nullable,omitempty"`
	Primary      bool               `json:"primary,omitempty"`
	Auto         bool               `json:"auto,omitempty"`
	Default      interface{}        `json:"default"`
	ReverseAlias string             `json:"reverse_alias,omitempty"`
	Rel          *MigrationRelation `json:"relation,omitempty"`
}

func (c *Column) String() string {
	var sb strings.Builder
	sb.WriteString("Column{")
	sb.WriteString(fmt.Sprintf("Name: %s, ", c.Name))
	sb.WriteString(fmt.Sprintf("Column: %s, ", c.Column))
	sb.WriteString(fmt.Sprintf("UseInDB: %t, ", c.UseInDB))
	sb.WriteString(fmt.Sprintf("MinLength: %d, ", c.MinLength))
	sb.WriteString(fmt.Sprintf("MaxLength: %d, ", c.MaxLength))
	sb.WriteString(fmt.Sprintf("MinValue: %f, ", c.MinValue))
	sb.WriteString(fmt.Sprintf("MaxValue: %f, ", c.MaxValue))
	sb.WriteString(fmt.Sprintf("Unique: %t, ", c.Unique))
	sb.WriteString(fmt.Sprintf("Nullable: %t, ", c.Nullable))
	sb.WriteString(fmt.Sprintf("Primary: %t", c.Primary))
	sb.WriteString("}")
	return sb.String()
}

func NewTableColumn(table Table, field attrs.Field) Column {

	var atts = field.Attrs()

	attrUseInDB, ok := internal.GetFromAttrs[bool](atts, AttrUseInDBKey)
	if !ok {
		attrUseInDB = true
	}

	if canMigrator, ok := field.(CanMigrate); ok {
		attrUseInDB = canMigrator.CanMigrate()
	}

	if attrs.IsEmbeddedField(field) {
		attrUseInDB = false
	}

	attrMaxLength, _ := internal.GetFromAttrs[int64](atts, attrs.AttrMaxLengthKey)
	attrMinLength, _ := internal.GetFromAttrs[int64](atts, attrs.AttrMinLengthKey)
	attrMinValue, _ := internal.GetFromAttrs[float64](atts, attrs.AttrMinValueKey)
	attrMaxValue, _ := internal.GetFromAttrs[float64](atts, attrs.AttrMaxValueKey)
	attrAutoIncrement, _ := internal.GetFromAttrs[bool](atts, attrs.AttrAutoIncrementKey)
	attrUnique, _ := internal.GetFromAttrs[bool](atts, attrs.AttrUniqueKey)
	attrReverseAlias, _ := internal.GetFromAttrs[string](atts, attrs.AttrReverseAliasKey)
	attrPrecision, _ := internal.GetFromAttrs[int64](atts, attrs.AttrPrecisionKey)
	attrScale, _ := internal.GetFromAttrs[int64](atts, attrs.AttrScaleKey)
	attrOnDelete, _ := internal.GetFromAttrs[Action](atts, AttrOnDeleteKey)
	attrOnUpdate, _ := internal.GetFromAttrs[Action](atts, AttrOnUpdateKey)

	var rel *MigrationRelation
	var fRel = field.Rel()
	if fRel != nil {
		var model *MigrationModel

		if typ, ok := fRel.(attrs.LazyRelation); ok {
			var modelKey = typ.ModelKey()
			if modelKey != "" {
				model = &MigrationModel{
					LazyModelKey: typ.ModelKey(),
				}
			}
		}

		if model == nil {
			model = &MigrationModel{
				CType: contenttypes.NewContentType(
					fRel.Model(),
				),
			}
		}

		var relType = fRel.Type()
		rel = &MigrationRelation{
			Type:        relType,
			TargetModel: model,
			TargetField: fRel.Field(),
			OnDelete:    attrOnDelete,
			OnUpdate:    attrOnUpdate,
		}

		var through = fRel.Through()
		if relType == attrs.RelManyToMany || relType == attrs.RelOneToMany || (relType == attrs.RelOneToOne && through != nil) {
			// many-to-many, one-to-many or one-to-one with through table do not directly
			// store the field in the table, so we set attrUseInDB to false.
			attrUseInDB = false
		}

		if through != nil {
			var model *MigrationModel
			if typ, ok := through.(attrs.LazyThrough); ok {
				var modelKey = typ.ModelKey()
				if modelKey != "" {
					model = &MigrationModel{
						LazyModelKey: typ.ModelKey(),
					}
				}
			}

			if model == nil {
				model = &MigrationModel{
					CType: contenttypes.NewContentType(
						through.Model(),
					),
				}
			}

			rel.Through = &MigrationRelationThrough{
				Model:       model,
				SourceField: through.SourceField(),
				TargetField: through.TargetField(),
			}
		}
	}

	var dflt = field.GetDefault()
	if def, ok := dflt.(attrs.Definer); ok {
		if !attrs.IsZero(dflt) {
			var defs = def.FieldDefs()
			var prim = defs.Primary()
			if prim != nil {
				dflt = prim.GetDefault()
			} else {
				dflt = nil // no primary field, no default
			}
		} else {
			dflt = nil // zero value, no default
		}
	}
	var nullable = field.AllowNull()
	nullable = nullable || (rel != nil && rel.TargetField != nil && rel.TargetField.AllowNull())
	if drivers.FieldType(field).Kind() == reflect.String && !attrUnique {
		nullable = true // strings are not nullable in the database
		if attrs.IsZero(dflt) {
			dflt = ""
		}
	}

	var col = Column{
		Table:        table,
		Field:        field,
		Name:         field.Name(),
		Column:       field.ColumnName(),
		UseInDB:      attrUseInDB,
		MinLength:    attrMinLength,
		MaxLength:    attrMaxLength,
		MinValue:     attrMinValue,
		MaxValue:     attrMaxValue,
		Unique:       attrUnique,
		Precision:    attrPrecision,
		Scale:        attrScale,
		Auto:         attrAutoIncrement || CanAutoIncrement(field),
		Primary:      field.IsPrimary(),
		Nullable:     nullable,
		Default:      dflt,
		ReverseAlias: attrReverseAlias,
		Rel:          rel,
	}

	return col
}

func (c *Column) DBType() dbtype.Type {
	var fieldType = c.FieldType()
	var fieldVal = reflect.New(fieldType).Elem()
	var dbType dbtype.Type
	if dbTypeDefiner, ok := fieldVal.Interface().(CanColumnDBType); ok {
		return dbTypeDefiner.DBType(c)
	} else if dbTypeDefiner, ok := c.Field.(CanColumnDBType); ok {
		return dbTypeDefiner.DBType(c)
	}

	dbType, ok := drivers.DBType(c.Field)
	if !ok && c.UseInDB {
		panic(fmt.Sprintf(
			"no database type registered for field %s of type %s",
			c.Field.Name(), fieldType.String(),
		))
	}

	return dbType
}

func CanAutoIncrement(field attrs.FieldDefinition) bool {
	return field.IsPrimary() && !field.AllowNull() && slices.Contains(
		[]reflect.Kind{
			reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		},
		field.Type().Kind(),
	)
}

func (c *Column) FieldType() reflect.Type {
	if c.Field == nil {
		return nil
	}
	return drivers.FieldType(c.Field)
}

func jsonCompare(a, b interface{}) (bool, error) {

	var (
		aBytes, bBytes []byte
		err            error
	)
	if aBytes, err = json.Marshal(a); err != nil {
		return false, err
	}
	if bBytes, err = json.Marshal(b); err != nil {
		return false, err
	}
	var (
		aFace = new(interface{})
		bFace = new(interface{})
	)
	if err = json.Unmarshal(aBytes, aFace); err != nil {
		return false, err
	}
	if err = json.Unmarshal(bBytes, bFace); err != nil {
		return false, err
	}
	if reflect.DeepEqual(aFace, bFace) {
		return true, nil
	}
	return false, nil
}

// even zero values are considered valid defaults
func (c *Column) HasDefault() bool {
	if c == nil {
		return false
	}
	if c.Default == nil {
		return false
	}

	if isZero, ok := c.Default.(interface{ IsZero() bool }); ok {
		return !isZero.IsZero()
	}

	var rv = reflect.ValueOf(c.Default)
	if rv.Kind() == reflect.Ptr {
		if !rv.IsValid() || rv.IsNil() {
			return false
		}
		rv = rv.Elem()
	}
	switch rv.Kind() {
	case reflect.String:
		return rv.String() != ""
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return rv.Int() != 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return rv.Uint() != 0
	case reflect.Float32, reflect.Float64:
		return rv.Float() != 0.0
	case reflect.Array, reflect.Slice, reflect.Map:
		return rv.Len() > 0
	}
	return true
}

func (c *Column) Equals(other *Column) bool {
	if c == nil && other == nil {
		return true
	}
	if (c == nil) != (other == nil) {
		return false
	}
	if c.Name != other.Name {
		return false
	}
	if c.Column != other.Column {
		return false
	}
	if c.MinLength != other.MinLength {
		return false
	}
	if c.MaxLength != other.MaxLength {
		return false
	}
	if c.MinValue != other.MinValue {
		return false
	}
	if c.MaxValue != other.MaxValue {
		return false
	}
	if c.Unique != other.Unique {
		return false
	}
	if c.Nullable != other.Nullable {
		return false
	}
	if c.Primary != other.Primary {
		return false
	}
	if c.Auto != other.Auto {
		return false
	}
	if c.DBType() != other.DBType() {
		return false
	}

	if equal, err := jsonCompare(c.Default, other.Default); err != nil {
		if !EqualDefaultValue(c.Default, other.Default) {
			return false
		}
	} else if !equal {
		return false
	}

	//if !EqualDefaultValue(c.Default, other.Default) {
	//	return false
	//}

	if c.ReverseAlias != other.ReverseAlias {
		return false
	}
	if (c.Rel == nil) != (other.Rel == nil) {
		return false
	}
	if c.Rel != nil {

		var other = other.Rel
		if c.Rel.Type != other.Type {
			return false
		}
		if (c.Rel.TargetModel == nil) != (other.TargetModel == nil) {
			return false
		}

		if c.Rel.TargetModel != nil {
			if !c.Rel.TargetModel.Equals(other.TargetModel) {
				return false
			}
			//if c.Rel.TargetModel.TypeName() != other.TargetModel.TypeName() {
			//	return false
			//}
		}

		if (c.Rel.TargetField == nil) != (other.TargetField == nil) {
			return false
		}

		if c.Rel.TargetField != nil {
			if c.Rel.TargetField.Name() != other.TargetField.Name() {
				return false
			}

			if c.Rel.TargetField.ColumnName() != other.TargetField.ColumnName() {
				return false
			}

			if c.Rel.TargetField.AllowNull() != other.TargetField.AllowNull() {
				return false
			}

			if c.Rel.TargetField.IsPrimary() != other.TargetField.IsPrimary() {
				return false
			}

			var (
				c1, ok1 = c.Rel.TargetField.(interface{ GetDefault() any })
				c2, ok2 = other.TargetField.(interface{ GetDefault() any })
			)

			if ok1 && ok2 {
				if c1.GetDefault() != c2.GetDefault() {
					return false
				}
			}

			if c.Rel.TargetField.Type() != other.TargetField.Type() {
				return false
			}
		}
	}
	return true
}
