package migrator

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"strings"

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

	attrUseInDB, ok := attrs.GetFromAttributes[bool](atts, AttrUseInDBKey)
	if !ok {
		attrUseInDB = true
	}

	if canMigrator, ok := field.(CanMigrate); ok {
		attrUseInDB = attrUseInDB && canMigrator.CanMigrate()
	}

	if attrs.IsEmbeddedField(field) {
		attrUseInDB = false
	}

	attrMaxLength, _ := attrs.GetFromAttributes[int64](atts, attrs.AttrMaxLengthKey)
	attrMinLength, _ := attrs.GetFromAttributes[int64](atts, attrs.AttrMinLengthKey)
	attrMinValue, _ := attrs.GetFromAttributes[float64](atts, attrs.AttrMinValueKey)
	attrMaxValue, _ := attrs.GetFromAttributes[float64](atts, attrs.AttrMaxValueKey)
	attrAutoIncrement, _ := attrs.GetFromAttributes[bool](atts, attrs.AttrAutoIncrementKey)
	attrUnique, _ := attrs.GetFromAttributes[bool](atts, attrs.AttrUniqueKey)
	attrReverseAlias, _ := attrs.GetFromAttributes[string](atts, attrs.AttrReverseAliasKey)
	attrPrecision, _ := attrs.GetFromAttributes[int64](atts, attrs.AttrPrecisionKey)
	attrScale, _ := attrs.GetFromAttributes[int64](atts, attrs.AttrScaleKey)
	attrOnDelete, _ := attrs.GetFromAttributes[Action](atts, AttrOnDeleteKey)
	attrOnUpdate, _ := attrs.GetFromAttributes[Action](atts, AttrOnUpdateKey)

	var rel *MigrationRelation
	var fRel = field.Rel()
	var isRev bool
	if f, ok := field.(attrs.CanIsReverse); ok && f.IsReverse() {
		isRev = true
	}

	if fRel != nil && !isRev {

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
			var defs = attrs.Define(context.Background(), def)
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

	//	if d, ok := dflt.([]byte); ok {
	//		dflt = string(d)
	//	}

	var nullable = field.AllowNull()
	nullable = nullable || (rel != nil && rel.TargetField != nil && rel.TargetField.AllowNull())
	if drivers.FieldType(field).Kind() == reflect.String && !attrUnique {
		nullable = true // strings are not nullable in the database
		if attrs.IsZero(dflt) {
			dflt = ""
		}
	}

	if field.Name() == "" {
		panic(fmt.Sprintf("field (%T)%v has name %q", field, field, field))
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

func jsonCompare(newDefault, oldDefault interface{}) (bool, error) {

	var (
		aBytes, bBytes []byte
		err            error
	)
	if aBytes, err = json.Marshal(newDefault); err != nil {
		return false, err
	}
	if bBytes, err = json.Marshal(oldDefault); err != nil {
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

	var rv = reflect.ValueOf(c.Default)
	if rv.Type().Kind() == reflect.Interface && rv.IsNil() {
		return false
	}

	if rv.Type().Kind() == reflect.Ptr && rv.IsNil() {
		return false
	}

	if isZero, ok := c.Default.(interface{ IsZero() bool }); ok {
		return !isZero.IsZero()
	}

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
		return !c.Primary && c.Rel == nil || rv.Int() != 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return !c.Primary && c.Rel == nil || rv.Uint() != 0
	case reflect.Float32, reflect.Float64:
		return !c.Primary && c.Rel == nil || rv.Float() != 0.0
	case reflect.Array, reflect.Slice, reflect.Map:
		return rv.Len() > 0
	case reflect.Invalid:
		return false
	}
	return true
}

func (c *Column) ChangeList(other *Column) []string {
	if c == nil && other == nil {
		return []string{}
	}
	if (c == nil) != (other == nil) {
		return []string{"*"}
	}

	var l = make([]string, 0)
	if c.Name != other.Name {
		l = append(l, "Name")
	}
	if c.Column != other.Column {
		l = append(l, "Column")
	}
	if c.MinLength != other.MinLength {
		l = append(l, "MinLength")
	}
	if c.MaxLength != other.MaxLength {
		l = append(l, "MaxLength")
	}
	if c.MinValue != other.MinValue {
		l = append(l, "MinValue")
	}
	if c.MaxValue != other.MaxValue {
		l = append(l, "MaxValue")
	}
	if c.Unique != other.Unique {
		l = append(l, "Unique")
	}
	if c.Nullable != other.Nullable {
		l = append(l, "Nullable")
	}
	if c.Primary != other.Primary {
		l = append(l, "Primary")
	}
	if c.Auto != other.Auto {
		l = append(l, "Auto")
	}
	if c.DBType() != other.DBType() {
		l = append(l, "DBType")
	}

	var _default = c.Default
	if (_default == nil) != (other.Default == nil) {
		l = append(l, "Default")
		// skip json compare check
		goto checkRevAlias
	}

	if _default == nil {
		goto checkRevAlias
	}

	if equal, err := jsonCompare(c.Default, other.Default); err != nil {
		if !EqualDefaultValue(c.Default, other.Default) {
			l = append(l, "Default")
		}
	} else if !equal {
		l = append(l, "Default")
	}

checkRevAlias:
	if c.ReverseAlias != other.ReverseAlias {
		l = append(l, "ReverseAlias")
	}
	if (c.Rel == nil) != (other.Rel == nil) {
		l = append(l, "Rel")
	}
	if c.Rel != nil {
		var other = other.Rel
		if c.Rel.Type != other.Type {
			l = append(l, "Rel:Type")
		}

		if (c.Rel.TargetModel == nil) != (other.TargetModel == nil) {
			l = append(l, "Rel:TargetModel")
			return l
		}

		if c.Rel.TargetModel != nil {
			if !c.Rel.TargetModel.Equals(other.TargetModel) {
				l = append(l, "Rel:TargetModel")
			}
			//if c.Rel.TargetModel.TypeName() != other.TargetModel.TypeName() {
			//	return false
			//}
		}

		if (c.Rel.TargetField == nil) != (other.TargetField == nil) {
			l = append(l, "Rel:TargetField")
			return l
		}

		if c.Rel.TargetField != nil {
			if c.Rel.TargetField.Name() != other.TargetField.Name() {
				l = append(l, "Rel:TargetField:Name")
			}

			if c.Rel.TargetField.ColumnName() != other.TargetField.ColumnName() {
				l = append(l, "Rel:TargetField:ColumnName")
			}

			if c.Rel.TargetField.AllowNull() != other.TargetField.AllowNull() {
				l = append(l, "Rel:TargetField:AllowNull")
			}

			if c.Rel.TargetField.IsPrimary() != other.TargetField.IsPrimary() {
				l = append(l, "Rel:TargetField:IsPrimary")
			}

			var (
				c1, ok1 = c.Rel.TargetField.(interface{ GetDefault() any })
				c2, ok2 = other.TargetField.(interface{ GetDefault() any })
			)

			if ok1 && ok2 {
				if c1.GetDefault() != c2.GetDefault() {
					l = append(l, "Rel:TargetField:GetDefault")
				}
			}

			if c.Rel.TargetField.Type() != other.TargetField.Type() {
				l = append(l, "Rel:TargetField:Type")
			}
		}
	}

	return l
}

func (c *Column) Equals(other *Column) bool {
	return len(c.ChangeList(other)) == 0
}
