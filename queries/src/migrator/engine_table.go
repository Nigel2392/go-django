package migrator

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/drivers/dbtype"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
	"github.com/elliotchance/orderedmap/v2"
)

var _ Table = (*ModelTable)(nil)

type Changed[T any] struct {
	Old T `json:"old,omitempty"`
	New T `json:"new,omitempty"`
}

func unchanged[T any](v T) *Changed[T] {
	var t T
	return &Changed[T]{
		Old: t,
		New: v,
	}
}

func changed[T any](old, new T) *Changed[T] {
	return &Changed[T]{
		Old: old,
		New: new,
	}
}

type IndexDefiner interface {
	DatabaseIndexes() []Index
}

type Index struct {
	table      *ModelTable `json:"-"`
	Identifier string      `json:"name"`
	Type       string      `json:"type"`
	Fields     []string    `json:"columns"`
	Unique     bool        `json:"unique,omitempty"`
	Comment    string      `json:"comment,omitempty"`
}

func (i *Index) Name() string {
	if i.Identifier != "" {
		return i.Identifier
	}
	var sb strings.Builder
	sb.WriteString(i.table.TableName())
	sb.WriteString("_idx_")
	for i, col := range i.Fields {
		if i > 0 {
			sb.WriteString("_")
		}
		sb.WriteString(col)
	}
	i.Identifier = sb.String()
	return i.Identifier
}

func (i Index) String() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Index{Name: %s, Type: %s, Unique: %t, Columns: [", i.Name(), i.Type, i.Unique))
	for _, col := range i.Fields {
		sb.WriteString(fmt.Sprintf("%s, ", col))
	}
	sb.WriteString("], Comment: ")
	if i.Comment != "" {
		sb.WriteString(fmt.Sprintf("%q", i.Comment))
	} else {
		sb.WriteString("''")
	}
	sb.WriteString("}")
	return sb.String()
}

func (i Index) Columns() []Column {
	var cols = make([]Column, 0, len(i.Fields))
	for _, col := range i.Fields {
		var tableCol, ok = i.table.Fields.Get(col)
		if !ok {
			panic(fmt.Sprintf("column %s not found in table %s", col, i.table.TableName()))
		}

		cols = append(cols, tableCol)
	}

	return cols
}

type ModelTable struct {
	Object attrs.Definer
	Table  string
	Desc   string
	Fields *orderedmap.OrderedMap[string, Column]
	Index  []Index
}

func (t *ModelTable) String() string {
	var sb strings.Builder
	sb.WriteString("ModelTable{\n")
	sb.WriteString(fmt.Sprintf("  Table: %s,\n", t.TableName()))
	sb.WriteString(fmt.Sprintf("  Model: %s,\n", t.ModelName()))
	sb.WriteString(fmt.Sprintf("  Comment: %s,\n", t.Comment()))
	sb.WriteString("  Fields: [\n")
	for head := t.Fields.Front(); head != nil; head = head.Next() {
		sb.WriteString(fmt.Sprintf("    %s,\n", head.Value.String()))
	}
	sb.WriteString("  ],\n")
	sb.WriteString("  Indexes: [\n")
	for _, idx := range t.Indexes() {
		sb.WriteString(fmt.Sprintf("    %s,\n", idx.String()))
	}
	sb.WriteString("  ],\n")
	sb.WriteString("}\n")
	return sb.String()
}

func NewModelTable(obj attrs.Definer) *ModelTable {

	var (
		newObjV = reflect.New(reflect.TypeOf(obj).Elem())
		object  = newObjV.Interface().(attrs.Definer)
		defs    = object.FieldDefs()
		fields  = defs.Fields()
	)

	var t = &ModelTable{
		Table:  defs.TableName(),
		Object: object,
		Fields: orderedmap.NewOrderedMap[string, Column](),
	}

	// Move primary fields to the front of the list
	slices.SortStableFunc(fields, func(a, b attrs.Field) int {
		if a.IsPrimary() && !b.IsPrimary() {
			return -1
		}
		if !a.IsPrimary() && b.IsPrimary() {
			return 1
		}
		return 0
	})

	for _, field := range fields {
		if field.ColumnName() == "" {
			continue
		}

		var col = NewTableColumn(t, field)
		t.Fields.Set(field.Name(), col)
	}

	if idxDef, ok := obj.(IndexDefiner); ok {
		indexes := idxDef.DatabaseIndexes()
		t.Index = make([]Index, 0, len(indexes))
		for _, idx := range indexes {
			t.Index = append(t.Index, Index{
				table:      t,
				Identifier: idx.Identifier,
				Type:       idx.Type,
				Fields:     idx.Fields,
				Unique:     idx.Unique,
				Comment:    idx.Comment,
			})
		}
	}

	return t
}

type serializableTableColumn struct {
	Column
	GOType  string          `json:"go_type"`
	DBType  string          `json:"db_type"`
	Default json.RawMessage `json:"default"`
}

type serializableModelTable struct {
	Table   string                                       `json:"table"`
	Model   *contenttypes.BaseContentType[attrs.Definer] `json:"model"`
	Fields  []serializableTableColumn                    `json:"fields"`
	Indexes []Index                                      `json:"indexes"`
	Comment string                                       `json:"comment"`
}

func (t *ModelTable) MarshalJSON() ([]byte, error) {
	var s = serializableModelTable{
		Table:   t.TableName(),
		Model:   contenttypes.NewContentType(t.Object),
		Indexes: t.Indexes(),
		Comment: t.Comment(),
		Fields:  make([]serializableTableColumn, 0, t.Fields.Len()),
	}

	for head := t.Fields.Front(); head != nil; head = head.Next() {

		var bytes, err = json.Marshal(head.Value.Default)
		if err != nil {
			return nil, fmt.Errorf("could not marshal default value for column %q: %w", head.Value.Name, err)
		}

		s.Fields = append(s.Fields, serializableTableColumn{
			Column:  head.Value,
			DBType:  head.Value.DBType().String(),
			GOType:  drivers.StringForType(head.Value.FieldType()),
			Default: bytes,
		})
	}

	return json.Marshal(s)
}

var nullLiteral = []byte("null")

func (t *ModelTable) UnmarshalJSON(data []byte) error {
	var s serializableModelTable
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	t.Table = s.Table
	t.Desc = s.Comment
	t.Object = s.Model.New()
	t.Fields = orderedmap.NewOrderedMap[string, Column]()
	t.Index = make([]Index, 0, len(s.Indexes))
	for _, idx := range s.Indexes {
		idx.table = t
		t.Index = append(t.Index, idx)
	}

	var defs = t.Object.FieldDefs()
	for _, col := range s.Fields {
		col.Table = t
		var f, ok = defs.Field(col.Name)
		if ok {
			col.Field = f
		}

		var isNullLiteral = bytes.Equal(col.Default, nullLiteral)
		if !isNullLiteral { // || (isNullLiteral && !col.Nullable && !col.Auto && !col.Primary) {
			// It is fine to unmarshal it into an incompatible type here,
			// i.e. drivers.DateTime into time.Time.
			// This is only used to properly restore migration defaults in SQL.

			var goType, ok = drivers.TypeFromString(col.GOType)
			if !ok {

				var dbType, ok = dbtype.NewFromString(col.DBType)
				if !ok {
					return fmt.Errorf("unknown db type %q for column %q", col.DBType, col.Name)
				}

				goType = drivers.DBToDefaultGoType(dbType)
			}

			var scanToVal = reflect.New(goType)
			var scanTo = scanToVal.Interface()
			if err := json.Unmarshal(col.Default, scanTo); err != nil {
				return fmt.Errorf("could not unmarshal default value for column %q: %w (%s)", col.Name, err, col.Default)
			}

			col.Column.Default = scanToVal.Elem().Interface()
		}

		t.Fields.Set(col.Name, col.Column)
	}

	return nil
}

func (t *ModelTable) ModelName() string {
	var rt = reflect.TypeOf(t.Object)
	if rt.Kind() == reflect.Ptr {
		rt = rt.Elem()
	}
	return rt.Name()
}

func (t *ModelTable) TableName() string {
	if t.Table != "" {
		return t.Table
	}

	var defs = t.Object.FieldDefs()
	return defs.TableName()
}

func (t *ModelTable) Model() attrs.Definer {
	return t.Object
}

func (t *ModelTable) Columns() []*Column {
	if t.Fields == nil {
		return nil
	}
	var cols = make([]*Column, 0, t.Fields.Len())
	for head := t.Fields.Front(); head != nil; head = head.Next() {
		cols = append(cols, &head.Value)
	}
	return cols
}

func (t *ModelTable) Comment() string {
	return t.Desc
}

func (t *ModelTable) Indexes() []Index {
	return t.Index
}

func (t *ModelTable) Diff(other *ModelTable) (added, removed []Column, diffs []Changed[Column]) {
	if t == nil && other == nil {
		return nil, nil, nil
	}
	if t == nil || other == nil {
		return nil, nil, nil
	}

	for head := other.Fields.Front(); head != nil; head = head.Next() {
		var col = head.Value
		var _, ok = t.Fields.Get(col.Name)
		if !ok {
			removed = append(removed, col)
			continue
		}
	}

	for head := t.Fields.Front(); head != nil; head = head.Next() {
		var col = head.Value
		var otherCol, ok = other.Fields.Get(col.Name)
		if !ok {
			added = append(added, col)
			continue
		}

		if !col.Equals(&otherCol) {
			diffs = append(diffs, Changed[Column]{
				Old: otherCol,
				New: col,
			})
		}
	}

	return added, removed, diffs
}
