package revisions

import (
	"context"
	"slices"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/src/core/attrs"
)

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

func UpdateRevision(ctx context.Context, rev *Revision) error {
	var _, err = queries.GetQuerySet(&Revision{}).WithContext(ctx).Filter("ID", rev.ID).Update(rev)
	return err
}

func DeleteRevision(ctx context.Context, rev *Revision) error {
	var _, err = queries.GetQuerySet(&Revision{}).WithContext(ctx).Filter("ID", rev.ID).Delete()
	return err
}
