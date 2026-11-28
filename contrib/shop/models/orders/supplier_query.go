package orders

import (
	"fmt"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/queries/src/expr"
)

type SupplierQuery struct {
	*queries.WrappedQuerySet[*SupplierModel, *SupplierQuery, *queries.QuerySet[*SupplierModel]]
}

func SuppliersQuery() *SupplierQuery {
	supplierQs := &SupplierQuery{}
	supplierQs.WrappedQuerySet = queries.WrapQuerySet[*SupplierModel](
		queries.GetQuerySet(&SupplierModel{}),
		supplierQs,
	)
	return supplierQs
}

func (qs *SupplierQuery) CloneQuerySet(wrapped *queries.WrappedQuerySet[*SupplierModel, *SupplierQuery, *queries.QuerySet[*SupplierModel]]) *SupplierQuery {
	return &SupplierQuery{
		WrappedQuerySet: wrapped,
	}
}

// A helper function to automatically create all suppliers in the database
func (qs *SupplierQuery) Sync() error {
	qs = qs.Clone()

	tx, err := qs.GetOrCreateTransaction()
	if err != nil {
		return fmt.Errorf("suppliers.SupplierQuery.Sync/GetOrCreateTransaction: %w", err)
	}

	defer tx.Rollback(qs.Context())

	var supplierNames = make([]string, 0, SupplierBackendCount())
	for _, supplier := range IterSupplierBackends(false) {
		supplierNames = append(supplierNames, supplier.InternalName())
	}

	if len(supplierNames) == 0 {
		_, err := qs.BulkUpdate(expr.As("Enabled", expr.V(false)))
		if err != nil && !errors.Is(err, errors.NoRows) {
			return err
		}
		return tx.Commit(qs.Context())
	}

	var suppliers = make([]*SupplierModel, 0, len(supplierNames))
	for _, name := range supplierNames {
		suppliers = append(suppliers, &SupplierModel{
			InternalName: name,
			Enabled:      true,
		})
	}

	_, err = qs.Filter(expr.Q("InternalName__in", supplierNames).Not(true)).BulkUpdate(
		expr.As("Enabled", expr.V(false)),
	)
	if err != nil && !errors.Is(err, errors.NoRows) {
		return err
	}

	foundSuppliers := make(map[string]struct{})
	rows, err := qs.Filter(expr.Q("InternalName__in", supplierNames)).All()
	if err != nil {
		return err
	}

	for _, row := range rows {
		foundSuppliers[row.Object.InternalName] = struct{}{}
	}

	if len(foundSuppliers) == len(suppliers) {
		return tx.Commit(qs.Context())
	}

	var createSuppliers = make([]*SupplierModel, 0, len(suppliers)-len(foundSuppliers))
	for _, supplier := range suppliers {
		if _, found := foundSuppliers[supplier.InternalName]; found {
			continue
		}

		createSuppliers = append(createSuppliers, supplier)
	}

	_, err = qs.BulkCreate(createSuppliers)
	if err != nil {
		return err
	}

	return tx.Commit(qs.Context())
}
