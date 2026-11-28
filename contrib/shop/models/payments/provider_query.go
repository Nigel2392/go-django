package payments

import (
	"fmt"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/queries/src/expr"
)

type ProviderQuery struct {
	*queries.WrappedQuerySet[*ProviderAccount, *ProviderQuery, *queries.QuerySet[*ProviderAccount]]
}

func ProvidersQuery() *ProviderQuery {
	providerQs := &ProviderQuery{}
	providerQs.WrappedQuerySet = queries.WrapQuerySet[*ProviderAccount](
		queries.GetQuerySet(&ProviderAccount{}),
		providerQs,
	)
	return providerQs
}

func (qs *ProviderQuery) CloneQuerySet(wrapped *queries.WrappedQuerySet[*ProviderAccount, *ProviderQuery, *queries.QuerySet[*ProviderAccount]]) *ProviderQuery {
	return &ProviderQuery{
		WrappedQuerySet: wrapped,
	}
}

// Sync synchronizes registered payment provider backends with the ProviderAccount table.
// Unregistered rows are disabled, existing registered rows keep their Enabled value,
// and missing registered backends are created with Enabled set to true.
func (qs *ProviderQuery) Sync() error {
	qs = qs.Clone()

	tx, err := qs.GetOrCreateTransaction()
	if err != nil {
		return fmt.Errorf("payments.ProviderQuery.Sync/GetOrCreateTransaction: %w", err)
	}

	defer tx.Rollback(qs.Context())

	providerNames := make([]string, 0, ProviderBackendCount())
	for _, provider := range IterProviderBackends(false) {
		providerNames = append(providerNames, provider.InternalName())
	}

	if len(providerNames) == 0 {
		_, err := qs.BulkUpdate(expr.As("Enabled", expr.V(false)))
		if err != nil && !errors.Is(err, errors.NoRows) {
			return err
		}
		return tx.Commit(qs.Context())
	}

	providers := make([]*ProviderAccount, 0, len(providerNames))
	for _, name := range providerNames {
		providers = append(providers, &ProviderAccount{
			InternalName: name,
			Enabled:      true,
		})
	}

	_, err = qs.Filter(expr.Q("InternalName__in", providerNames).Not(true)).BulkUpdate(
		expr.As("Enabled", expr.V(false)),
	)
	if err != nil && !errors.Is(err, errors.NoRows) {
		return err
	}

	foundProviders := make(map[string]struct{})
	rows, err := qs.Filter(expr.Q("InternalName__in", providerNames)).All()
	if err != nil {
		return err
	}

	for _, row := range rows {
		foundProviders[row.Object.InternalName] = struct{}{}
	}

	if len(foundProviders) == len(providers) {
		return tx.Commit(qs.Context())
	}

	createProviders := make([]*ProviderAccount, 0, len(providers)-len(foundProviders))
	for _, provider := range providers {
		if _, found := foundProviders[provider.InternalName]; found {
			continue
		}

		createProviders = append(createProviders, provider)
	}

	_, err = qs.BulkCreate(createProviders)
	if err != nil {
		return err
	}

	return tx.Commit(qs.Context())
}
