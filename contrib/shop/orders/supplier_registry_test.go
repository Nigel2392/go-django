package orders

import (
	"iter"
	"testing"

	"github.com/Nigel2392/go-django/contrib/shop/products"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/forms/widgets/options"
	"github.com/elliotchance/orderedmap/v2"
	"github.com/stretchr/testify/require"
)

func TestSupplierRegistry(t *testing.T) {
	previous := registry
	registry = &backendRegistry{backends: orderedmap.NewOrderedMap[string, SupplierBackend]()}
	t.Cleanup(func() { registry = previous })

	field, ok := attrs.Define(t.Context(), &products.Product{}).Field("Supplier")
	require.True(t, ok)
	widget, ok := field.FormField().Widget().(*options.OptionsWidget)
	require.True(t, ok, "supplier field should use a select widget")
	require.Empty(t, widget.Choices(t.Context()))

	RegisterSupplier(&Supplier{Name: "zulu", Order: 5})
	RegisterSupplier(&Supplier{Name: "alpha", Order: 5})
	RegisterSupplier(&Supplier{Name: "first", Order: -1})
	replacement := &Supplier{Name: "zulu", Order: 5}
	RegisterSupplier(replacement)

	require.Equal(t, 3, SupplierBackendCount())
	backend, ok := RetrieveSupplier(products.SupplierValue("zulu"))
	require.True(t, ok)
	require.Same(t, replacement, backend)
	_, ok = RetrieveSupplier("unknown")
	require.False(t, ok)

	names := func(backends iter.Seq2[int, SupplierBackend]) []string {
		var names []string
		for index, backend := range backends {
			require.Equal(t, len(names), index)
			names = append(names, backend.InternalName())
		}
		return names
	}
	require.Equal(t, []string{"zulu", "alpha", "first"}, names(IterSupplierBackends(false)))
	require.Equal(t, []string{"first", "alpha", "zulu"}, names(IterSupplierBackends(true)))

	bound := BoundSupplierBackends(t.Context())
	require.Equal(t, "First", bound[0].Label())
	choices := widget.Choices(t.Context())
	require.Len(t, choices, 3)
	for i, option := range choices {
		require.Equal(t, bound[i].Label(), option.Label())
		require.Equal(t, bound[i].InternalName(), option.Value())
	}
}
