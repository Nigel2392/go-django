package pages

import (
	"context"
	"fmt"
	"reflect"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
	django_models "github.com/Nigel2392/go-django/src/models"
	"github.com/pkg/errors"
)

// SavePage saves a custom page object to the database.
//
// If the parent is nil, the page is presumed to be updating.
//
// This can not be used to create a new root page.
//
// The Save() is called on the custom page object before the reference node is created/updated.
func SavePage(ctx context.Context, parent *PageNode, p SaveablePage) error {
	if parent == nil {
		return UpdatePage(ctx, p)
	}

	var querySet = queries.GetQuerySet(&PageNode{}).WithContext(ctx)
	var transaction, err = querySet.GetOrCreateTransaction()
	if err != nil {
		return errors.Wrap(err, "failed to start transaction")
	}
	defer transaction.Rollback(ctx)

	var ref = p.Reference()
	if ref.ContentType == "" && !reflect.DeepEqual(ref, p) {
		var cType = contenttypes.NewContentType(p)
		ref.ContentType = cType.TypeName()
	}

	if parent.Path == "" {
		return fmt.Errorf("parent path must not be empty")
	}

	var definition = DefinitionForType(ref.ContentType)
	if definition != nil {
		// if definition.MaxNum > 0 {
		// var count, err = CountNodesByType(queries, ctx, ref.ContentType)
		// if err != nil {
		// return err
		// }
		//
		// if count >= int64(definition.MaxNum) {
		// return fmt.Errorf("cannot create more than %d pages of type %s", definition.MaxNum, ref.ContentType)
		// }
		// }

		if len(definition._childPageTypes) > 0 {
			var parentType = parent.ContentType
			var parentDef = DefinitionForType(parentType)
			if _, exists := parentDef._childPageTypes[ref.ContentType]; !exists {
				return fmt.Errorf("parent type %s is not allowed to have children of type %s", parentType, ref.ContentType)
			}

			if len(definition._parentPageTypes) > 0 {
				if _, exists := definition._parentPageTypes[parentType]; !exists {
					return fmt.Errorf("page type %s is not allowed to have parent of type %s", ref.ContentType, parentType)
				}
			}
		}
	}

	var saved bool
	saved, err = django_models.SaveModel(
		ctx, p,
	)
	if err != nil {
		return err
	}
	if !saved {
		return fmt.Errorf("page %T could not be saved", p)
	}

	var (
		srcDefs = p.FieldDefs()
		dstDefs = ref.FieldDefs()
	)

	var refField, _ = dstDefs.Field("PageID")
	var srcVal, _ = srcDefs.Primary().Value()
	if err = refField.Scan(srcVal); err != nil {
		return errors.Wrapf(
			err, "failed to set PageID for %T", p,
		)
	}

	if ref.Path == "" {
		err = CreateChildNode(
			ctx, parent, ref,
		)
	} else {
		err = UpdateNode(
			ctx, ref,
		)
	}
	if err != nil {
		return err
	}

	return transaction.Commit(ctx)
}

// UpdatePage updates a page object in the database.
//
// It calls page.Save() to update the custom page object.
//
// The reference node is updated before the page is saved.
func UpdatePage(ctx context.Context, p SaveablePage) error {
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

	if err := UpdateNode(ctx, ref); err != nil {
		return err
	}

	var saved, err = django_models.SaveModel(ctx, p)
	if err != nil {
		return err
	}
	if !saved {
		return fmt.Errorf("page %T could not be saved", p)
	}
	return nil
}

// DeletePage deletes a page object from the database.
//
// It calls page.Delete() to delete the custom page object after the reference node is deleted.
//
// FixTree is called after the page is deleted to ensure the tree is in a consistent state.
func DeletePage(ctx context.Context, p DeletablePage) (err error) {
	var ref = p.Reference()
	if ref.PK == 0 {
		return fmt.Errorf("page id must not be zero")
	}

	if err = DeleteNode(ctx, ref); err != nil {
		return err
	}

	var deleted bool
	deleted, err = django_models.DeleteModel(ctx, p)
	if err != nil {
		return err
	}

	if !deleted {
		return fmt.Errorf("page %T could not be deleted", p)
	}

	return FixTree(ctx)
}
