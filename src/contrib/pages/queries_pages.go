package pages

import (
	"context"
	"fmt"

	django_models "github.com/Nigel2392/go-django/src/models"
	"github.com/pkg/errors"
)

// DeletePage deletes a page object from the database.
//
// It calls page.Delete() to delete the custom page object after the reference node is deleted.
//
// FixTree is called after the page is deleted to ensure the tree is in a consistent state.
func DeletePage(ctx context.Context, p DeletablePage) error {
	var ref = p.Reference()
	if ref.PK == 0 {
		return fmt.Errorf("page id must not be zero")
	}

	var qs = NewPageQuerySet().WithContext(ctx)
	var tx, err = qs.GetOrCreateTransaction()
	if err != nil {
		return errors.Wrap(err, "failed to start transaction")
	}
	defer tx.Rollback(qs.Context())

	if err = qs.DeleteNode(ref); err != nil {
		return err
	}

	var deleted bool
	deleted, err = django_models.DeleteModel(qs.Context(), p)
	if err != nil {
		return err
	}

	if !deleted {
		return fmt.Errorf("page %T could not be deleted", p)
	}

	if err = FixTree(qs.Context()); err != nil {
		return errors.Wrap(err, "failed to fix tree after page deletion")
	}

	return tx.Commit(qs.Context())
}
