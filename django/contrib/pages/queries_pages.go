package pages

import (
	"context"
	"fmt"
	"reflect"

	"github.com/Nigel2392/django/contrib/pages/models"
	"github.com/Nigel2392/django/core/contenttypes"
)

func SavePage(q models.DBQuerier, ctx context.Context, parent *models.PageNode, p SaveablePage) error {
	if parent == nil {
		return UpdatePage(q, ctx, p)
	}

	var tx, err = q.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer tx.Rollback()

	var (
		queries = q.WithTx(tx)
		ref     = p.Reference()
	)

	if ref.ContentType == "" && !reflect.DeepEqual(ref, p) {
		var cType = contenttypes.NewContentType(p)
		ref.ContentType = cType.TypeName()
	}

	if parent.Path == "" {
		return fmt.Errorf("parent path must not be empty")
	}

	if err = p.Save(ctx); err != nil {
		return err
	}

	if ref.Path == "" {
		err = CreateChildNode(
			q, ctx, parent, ref,
		)
	} else {
		err = queries.UpdateNode(
			ctx, ref.Title, ref.Path, ref.Depth, ref.Numchild, ref.UrlPath, int64(ref.StatusFlags), ref.PageID, ref.ContentType, ref.PK,
		)
	}
	if err != nil {
		return err
	}

	return tx.Commit()
}

func UpdatePage(q models.DBQuerier, ctx context.Context, p SaveablePage) error {
	var ref = p.Reference()
	if ref.Path == "" {
		return fmt.Errorf("page path must not be empty")
	}

	if ref.PK == 0 {
		return fmt.Errorf("page id must not be zero")
	}

	if ref.ContentType == "" && !reflect.DeepEqual(ref, p) {
		var cType = contenttypes.NewContentType(p)
		ref.ContentType = cType.TypeName()
	}

	if err := q.UpdateNode(
		ctx,
		ref.Title,
		ref.Path,
		ref.Depth,
		ref.Numchild,
		ref.UrlPath,
		int64(ref.StatusFlags),
		ref.PageID,
		ref.ContentType,
		ref.PK,
	); err != nil {
		return err
	}

	return p.Save(ctx)
}

func DeletePage(q models.DBQuerier, ctx context.Context, p DeletablePage) error {
	var ref = p.Reference()
	if ref.PK == 0 {
		return fmt.Errorf("page id must not be zero")
	}

	if err := DeleteNode(q, ctx, ref.PK, ref.Path, ref.Depth); err != nil {
		return err
	}

	return p.Delete(ctx)
}
