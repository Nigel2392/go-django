package revisions

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"time"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/queries/src/expr"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/elliotchance/orderedmap/v2"
)

type RevisionQuerySet struct {
	*queries.WrappedQuerySet[*Revision, *RevisionQuerySet, *queries.QuerySet[*Revision]]
}

func newRevisionQuerySet(qs *queries.QuerySet[*Revision]) *RevisionQuerySet {
	var specific = &RevisionQuerySet{}
	specific.WrappedQuerySet = queries.WrapQuerySet[*Revision](
		qs, specific,
	)
	return specific
}

func NewRevisionQuerySet() *RevisionQuerySet {
	return newRevisionQuerySet(queries.GetQuerySet(&Revision{}))
}

func (qs *RevisionQuerySet) CloneQuerySet(wrapped *queries.WrappedQuerySet[*Revision, *RevisionQuerySet, *queries.QuerySet[*Revision]]) *RevisionQuerySet {
	return &RevisionQuerySet{
		WrappedQuerySet: wrapped,
	}
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

type specificRevision struct {
	ids       []string
	revisions map[string]*queries.Row[*ObjectRevision]
}

type ObjectRevision struct {
	Object attrs.Definer
	*Revision
}

func (qs *RevisionQuerySet) All() (queries.Rows[*ObjectRevision], error) {
	// Use iter method of the base queryset to not have to
	// iterate over all rows multiple times.
	var rowCount, baseRows, err = qs.WrappedQuerySet.Base().IterAll()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get all rows for specific revisions")
	}

	// Create a new map to hold the specific revision content type and ID-list pairs.
	var preloadMap = orderedmap.NewOrderedMap[string, *specificRevision]()
	var rows = make(queries.Rows[*ObjectRevision], rowCount)
	var rowIdx int
	for row, err := range baseRows {
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get row %d for specific revisions", rowIdx)
		}

		var specificRevisionRow = &queries.Row[*ObjectRevision]{
			ObjectFieldDefs: row.ObjectFieldDefs,
			Through:         row.Through,
			Annotations:     row.Annotations,

			// currently we set the object to the node instead of the specific revision,
			// if a specific revision can be found, it will be properly set to the right object
			// later in the prefetch loop.
			Object: &ObjectRevision{
				Revision: row.Object,
				Object:   nil,
			},
		}

		// If the revision has no content type or object ID, we skip it,
		// it cannot be preloaded - log a warning.
		if row.Object.ObjectID == "" || row.Object.ContentType == "" {
			logger.Warnf("Revision with ID %d has no content type or object ID, skipping preload", row.Object.ID)

			// add the row to the results, increase idx
			rows[rowIdx] = specificRevisionRow
			rowIdx++
			continue
		}

		// Cache a reference to the specific revision row in
		// the preload map - this allows us to efficiently
		// set the revision object later in a single query
		// for each content type and id list combination.
		var preload, exists = preloadMap.Get(
			row.Object.ContentType,
		)
		if !exists {
			preload = &specificRevision{
				ids:       make([]string, 0, 1),
				revisions: make(map[string]*queries.Row[*ObjectRevision], 1),
			}
		}

		preload.ids = append(preload.ids, row.Object.ObjectID)
		preload.revisions[row.Object.ObjectID] = specificRevisionRow

		preloadMap.Set(
			row.Object.ContentType,
			preload,
		)

		// add the row to the results, increase idx
		rows[rowIdx] = specificRevisionRow
		rowIdx++
	}

	if preloadMap.Len() == 0 {
		return rows, err
	}

	// prefetch all rows for each content type
	// and set the revision object for each row.
	for head := preloadMap.Front(); head != nil; head = head.Next() {
		var definition = contenttypes.DefinitionForType(head.Key)
		if definition == nil {
			return rows, errors.New(errors.CodeNoRows, fmt.Sprintf(
				"no content type definition found for %s",
				head.Key,
			))
		}

		var model = definition.Object().(attrs.Definer)
		var defs = model.FieldDefs()
		var primaryField = defs.Primary()
		var rows, err = queries.GetQuerySet(model).
			WithContext(qs.Context()).
			Filter(fmt.Sprintf("%s__in", primaryField.Name()), head.Value.ids).
			All()
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get rows for content type %s", head.Key)
		}

		for _, row := range rows {
			var pkStr, _, err = getIdAndContentType(qs.Context(), row.Object)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to get ID for object %T", row.Object)
			}

			var pk = attrs.PrimaryKey(row.Object)
			var revRow, exists = head.Value.revisions[pkStr]
			if !exists {
				return nil, errors.NoRows.Wrapf(
					"no revision found for content type %s with PK %d",
					head.Key, pk,
				)
			}

			revRow.Object.Object = row.Object
		}
	}

	return rows, err
}

func (qs *RevisionQuerySet) Get() (*queries.Row[*ObjectRevision], error) {
	var nillRow = &queries.Row[*ObjectRevision]{}

	// limit to max_get_results
	*qs = *qs.Limit(queries.MAX_GET_RESULTS)

	var results, err = qs.All()
	if err != nil {
		return nillRow, err
	}

	var resCnt = len(results)
	if resCnt == 0 {
		return nillRow, errors.NoRows.WithCause(fmt.Errorf(
			"no rows found for %T", qs.Meta().Model(),
		))
	}

	if resCnt > 1 {
		var errResCnt string
		if queries.MAX_GET_RESULTS == 0 || resCnt < queries.MAX_GET_RESULTS {
			errResCnt = strconv.Itoa(resCnt)
		} else {
			errResCnt = strconv.Itoa(queries.MAX_GET_RESULTS-1) + "+"
		}

		return nillRow, errors.MultipleRows.WithCause(fmt.Errorf(
			"multiple rows returned for %T: %s rows",
			qs.Meta().Model(), errResCnt,
		))
	}

	return results[0], nil
}

// First is used to retrieve the first row from the database.
//
// It returns a Query that can be executed to get the result, which is a Row object
// that contains the model object and a map of annotations.
func (qs *RevisionQuerySet) First() (*queries.Row[*ObjectRevision], error) {
	*qs = *qs.Limit(1)

	var results, err = qs.All()
	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return nil, errors.NoRows
	}

	return results[0], nil
}

// Last is used to retrieve the last row from the database.
//
// It reverses the order of the results and then calls First to get the last row.
//
// It returns a Query that can be executed to get the result, which is a Row object
// that contains the model object and a map of annotations.
func (qs *RevisionQuerySet) Last() (*queries.Row[*ObjectRevision], error) {
	*qs = *qs.Reverse()
	return qs.First()
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
