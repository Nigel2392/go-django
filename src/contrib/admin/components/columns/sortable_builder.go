package columns

import (
	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/src/core/assert"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/views/list"
)

type SortableColumnBuilder[T attrs.Definer] struct {
	// if this is false, the builder will not add sorting capabilities
	// this might spare you some if else statements.
	CanSort bool

	model           T
	meta            attrs.ModelMeta
	defs            attrs.StaticDefinitions
	isSortFieldFunc func(f attrs.FieldDefinition) bool

	// all valid sortable fields
	_fields_asc map[string]struct{}

	// all valid sortable fields in descending order (prefix with "-" sign)
	// saves on string comparisons / allocations
	_fields_desc map[string]string
}

func NewSortableColumnBuilder[T attrs.Definer](model T, isSortField ...func(f attrs.FieldDefinition) bool) *SortableColumnBuilder[T] {
	var isSortFieldFunc func(f attrs.FieldDefinition) bool
	if len(isSortField) > 1 {
		isSortFieldFunc = isSortField[0]
	}

	var meta = attrs.GetModelMeta(model)
	var b = &SortableColumnBuilder[T]{
		CanSort:         true,
		model:           model,
		meta:            meta,
		defs:            meta.Definitions(),
		isSortFieldFunc: isSortFieldFunc,
		_fields_asc:     make(map[string]struct{}),
		_fields_desc:    make(map[string]string),
	}

	if isSortFieldFunc != nil {
		for _, field := range b.defs.Fields() {
			if !isSortFieldFunc(field) {
				continue
			}

			b._fields_asc[field.Name()] = struct{}{}
			b._fields_desc["-"+field.Name()] = field.Name()
		}
	}

	return b
}

func (b *SortableColumnBuilder[T]) IsSortable(field string) bool {
	if !b.CanSort {
		return false
	}

	if _, ok := b._fields_asc[field]; ok {
		return true
	}
	_, ok := b._fields_desc[field]
	return ok
}

func (b *SortableColumnBuilder[T]) isSortField(f attrs.FieldDefinition) bool {
	if b.isSortFieldFunc == nil {
		return f.Rel() == nil
	}
	return b.isSortFieldFunc(f)
}

func (b *SortableColumnBuilder[T]) AddColumn(column any) list.ListColumn[T] {
	var (
		ok    bool
		fName string
		col   list.ListColumn[T]
		fld   attrs.FieldDefinition
	)

	if col, ok = column.(list.ListColumn[T]); ok {
		fName = col.FieldName()
		fld, _ = b.defs.Field(fName)
		goto checkSortable
	}

	if s, ok := column.(string); ok {
		fName = s
	}

	fld, ok = b.defs.Field(fName)
	if !ok {
		assert.Fail(
			"Field not found in definitions but column type %T is not list.ListColumn",
			column,
		)
	}

	col = list.FieldColumn[T](
		fld.Label, fld.Name(),
	)

checkSortable:
	if !b.CanSort || fld == nil || !b.isSortField(fld) {
		return col
	}

	b._fields_asc[fName] = struct{}{}
	b._fields_desc["-"+fName] = fName

	return &sortableListColumn[T]{
		ListColumn: col,
	}
}

func (b *SortableColumnBuilder[T]) Sort(qs *queries.QuerySet[T], sortOrder []string) *queries.QuerySet[T] {
	if !b.CanSort {
		return qs
	}

	var sortFields = make([]string, 0, len(sortOrder))
	for _, field := range sortOrder {
		if !b.IsSortable(field) {
			continue
		}

		sortFields = append(sortFields, field)
	}

	if len(sortFields) > 0 {
		qs = qs.OrderBy(sortFields...)
	}

	return qs
}
