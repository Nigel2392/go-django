package pages

import (
	"context"
	"fmt"
	"reflect"

	models "github.com/Nigel2392/go-django/src/contrib/pages/page_models"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
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
		err = UpdateNode(
			queries, ctx, ref,
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

	if err := UpdateNode(q, ctx, ref); err != nil {
		return err
	}

	return p.Save(ctx)
}

func DeletePage(q models.DBQuerier, ctx context.Context, p DeletablePage) (err error) {
	var ref = p.Reference()
	if ref.PK == 0 {
		return fmt.Errorf("page id must not be zero")
	}

	if err = DeleteNode(q, ctx, ref.PK, ref.Path, ref.Depth); err != nil {
		return err
	}

	err = p.Delete(ctx)
	if err != nil {
		return err
	}

	return FixTree(q, ctx)
}
