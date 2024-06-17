package pages

import (
	"context"
	"fmt"

	"github.com/Nigel2392/django/contrib/pages/models"
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

	if parent.Path == "" {
		return fmt.Errorf("parent path must not be empty")
	}

	if ref.Path == "" {
		var err = CreateChildNode(
			q, ctx, parent, ref,
		)
		if err != nil {
			return err
		}

		if err = p.Save(ctx); err != nil {
			return err
		}
	} else {
		err = queries.UpdateNode(
			ctx, ref.Title, ref.Path, ref.Depth, ref.Numchild, int64(ref.StatusFlags), ref.PageID, ref.ContentType, ref.ID,
		)
		if err != nil {
			return err
		}

		if err = p.Save(ctx); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func UpdatePage(q models.DBQuerier, ctx context.Context, p SaveablePage) error {
	var ref = p.Reference()
	if ref.Path == "" {
		return fmt.Errorf("page path must not be empty")
	}

	if ref.ID == 0 {
		return fmt.Errorf("page id must not be zero")
	}

	if err := q.UpdateNode(
		ctx,
		ref.Title,
		ref.Path,
		ref.Depth,
		ref.Numchild,
		int64(ref.StatusFlags),
		ref.PageID,
		ref.ContentType,
		ref.ID,
	); err != nil {
		return err
	}

	return p.Save(ctx)
}

func DeletePage(q models.DBQuerier, ctx context.Context, p DeletablePage) error {
	var ref = p.Reference()
	if ref.ID == 0 {
		return fmt.Errorf("page id must not be zero")
	}

	if err := DeleteNode(q, ctx, ref.ID, ref.Path, ref.Depth); err != nil {
		return err
	}

	return p.Delete(ctx)
}
