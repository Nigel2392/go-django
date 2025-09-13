package revisions

import (
	"context"
	"fmt"
	"slices"
	"time"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/queries/src/expr"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
	"github.com/elliotchance/orderedmap/v2"
)

type RevisionQuerySet struct {
	*queries.WrappedQuerySet[*Revision, *RevisionQuerySet, *queries.QuerySet[*Revision]]
}

func newRevisionQuerySet(qs *queries.QuerySet[*Revision]) *RevisionQuerySet {
	var s = &RevisionQuerySet{}
	s.WrappedQuerySet = queries.WrapQuerySet[*Revision](
		qs, s,
	)
	return s
}

func NewRevisionQuerySet() *RevisionQuerySet {
	return newRevisionQuerySet(queries.GetQuerySet(&Revision{}))
}

func (qs *RevisionQuerySet) CloneQuerySet(wrapped *queries.WrappedQuerySet[*Revision, *RevisionQuerySet, *queries.QuerySet[*Revision]]) *RevisionQuerySet {
	return &RevisionQuerySet{
		WrappedQuerySet: wrapped,
	}
}

func (w *RevisionQuerySet) Base() *queries.QuerySet[*Revision] {
	return w.WrappedQuerySet.Base()
}

func (qs *RevisionQuerySet) ForObjects(objs ...attrs.Definer) *RevisionQuerySet {
	var objMapping = orderedmap.NewOrderedMap[string, []string]()
	for _, obj := range objs {
		var objKey, cTypeName, err = getIdAndContentType(qs.Context(), obj)
		if err != nil {
			panic(fmt.Errorf("ForObjects: %w", err))
		}

		var ids, exists = objMapping.Get(cTypeName)
		if !exists {
			ids = make([]string, 0, 1)
		}

		ids = append(ids, objKey)
		objMapping.Set(cTypeName, ids)
	}

	var filters = make([]expr.Expression, 0, objMapping.Len())
	for head := objMapping.Front(); head != nil; head = head.Next() {
		var cTypeExpr expr.ClauseExpression = expr.Q("ContentType", head.Key)
		cTypeExpr = cTypeExpr.And(
			expr.Q("ObjectID__in", head.Value),
		)
		filters = append(filters, cTypeExpr)
	}

	if len(filters) == 0 {
		return qs
	}

	return qs.Filter(expr.Or(filters...))
}

func (qs *RevisionQuerySet) Types(types ...any) *RevisionQuerySet {
	if len(types) == 0 {
		return qs
	}

	var typeNames = make([]string, len(types))
	for i, t := range types {
		switch v := t.(type) {
		case string:
			typeNames[i] = contenttypes.ReverseAlias(v)
		case attrs.Definer:
			typeNames[i] = contenttypes.NewContentType(v).TypeName()
		case contenttypes.ContentType:
			typeNames[i] = v.TypeName()
		default:
			panic(fmt.Errorf(
				"invalid type %T for ForTypes, expected string, attrs.Definer or contenttypes.ContentType", t,
			))
		}
	}

	if len(typeNames) == 1 {
		return qs.Filter("ContentType", typeNames[0])
	}

	return qs.Filter("ContentType__in", typeNames)
}

func ListRevisions(ctx context.Context, limit, offset int) ([]*Revision, error) {
	var rows, err = queries.GetQuerySet(&Revision{}).
		WithContext(ctx).
		Limit(limit).
		Offset(offset).
		OrderBy("-CreatedAt").
		All()
	if err != nil {
		return nil, err
	}
	return slices.Collect(rows.Objects()), nil
}

func GetRevisionByID(ctx context.Context, id int64) (*Revision, error) {
	var row, err = queries.GetQuerySet(&Revision{}).Filter("ID", id).Get()
	if err != nil {
		return nil, err
	}
	return row.Object, nil
}

func LatestRevision(ctx context.Context, obj attrs.Definer, getRevInfo ...QueryInfoFunc) (*Revision, error) {
	var res, err = GetRevisionsByObject(ctx, obj, 1, 0, getRevInfo...)
	if err != nil {
		return nil, err
	}
	if len(res) == 0 {
		return nil, errors.NoRows
	}
	return res[0], nil
}

func GetRevisionsByObject(ctx context.Context, obj attrs.Definer, limit int, offset int, getRevInfo ...QueryInfoFunc) ([]*Revision, error) {
	var objKey, cTypeName, err = getIdAndContentType(ctx, obj, getRevInfo...)
	if err != nil {
		return nil, errors.Wrap(err, "GetRevisionsByObject")
	}

	rowCount, rowsIter, err := queries.GetQuerySet(&Revision{}).
		WithContext(ctx).
		Filter("ObjectID", objKey).
		Filter("ContentType", cTypeName).
		Limit(limit).
		Offset(offset).
		OrderBy("-CreatedAt").
		IterAll()
	if err != nil {
		return nil, errors.Wrap(
			err, "GetRevisionsByObject",
		)
	}
	var idx = 0
	var revisions = make([]*Revision, 0, rowCount)
	for row, err := range rowsIter {
		if err != nil {
			return nil, errors.Wrapf(
				err, "GetRevisionsByObject: row %d", idx,
			)
		}
		revisions = append(revisions, row.Object)
		idx++
	}
	return revisions, nil
}

func DeleteRevisionsByObject(ctx context.Context, obj attrs.Definer, getRevInfo ...QueryInfoFunc) (int64, error) {
	var objKey, cType, err = getIdAndContentType(ctx, obj, getRevInfo...)
	if err != nil {
		return 0, errors.Wrap(err, "DeleteRevisionsByObject")
	}

	return queries.GetQuerySet(&Revision{}).
		WithContext(ctx).
		Filter("ObjectID", objKey).
		Filter("ContentType", cType).
		Delete()
}

func CreateRevision(ctx context.Context, forObj attrs.Definer, getRevInfo ...QueryInfoFunc) (*Revision, error) {
	var revision *Revision
	switch obj := forObj.(type) {
	case *Revision:
		revision = obj
	default:
		var rev, err = NewRevision(forObj, getRevInfo...)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create revision")
		}
		revision = rev
	}
	return queries.GetQuerySet(&Revision{}).WithContext(ctx).Create(revision)
}

func CreateDatedRevision(ctx context.Context, forObj attrs.Definer, at time.Time, getRevInfo ...QueryInfoFunc) (*Revision, error) {
	var revision *Revision
	switch obj := forObj.(type) {
	case *Revision:
		revision = obj
	default:
		var rev, err = NewRevision(forObj, getRevInfo...)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create revision")
		}
		revision = rev
	}
	revision.CreatedAt = at
	return queries.GetQuerySet(&Revision{}).
		WithContext(ctx).
		Create(revision)
}

func UpdateRevision(ctx context.Context, rev *Revision) error {
	var _, err = queries.GetQuerySet(&Revision{}).WithContext(ctx).Filter("ID", rev.ID).Update(rev)
	return err
}

func DeleteRevision(ctx context.Context, rev *Revision) error {
	var _, err = queries.GetQuerySet(&Revision{}).WithContext(ctx).Filter("ID", rev.ID).Delete()
	return err
}
