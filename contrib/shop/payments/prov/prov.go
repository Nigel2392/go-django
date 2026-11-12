package prov

import (
	"context"
	"database/sql/driver"

	"github.com/Nigel2392/go-django/queries/src/drivers/dbtype"
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/src/components"
	"github.com/elliotchance/orderedmap/v2"
)

func init() {
	dbtype.Add(BaseProvider{}, dbtype.String)
}

type BasePaymentProvider interface {
	Base() BaseProvider
	ProviderID() string
	TextLabel(context.Context) string
	TextHelp(context.Context) string
	WidgetIcon() components.Component
}

type BaseProvider struct {
	ID       string                       `json:",inline"`
	Icon     components.Component         `json:"-"`
	Label    func(context.Context) string `json:"-"`
	Helptext func(context.Context) string `json:"-"`
}

func (b BaseProvider) Base() BaseProvider {
	return b
}

func (b BaseProvider) ProviderID() string {
	return b.ID
}

func (b BaseProvider) TextLabel(ctx context.Context) string {
	if b.Label == nil {
		return b.ID
	}
	return b.Label(ctx)
}

func (b BaseProvider) TextHelp(ctx context.Context) string {
	if b.Helptext == nil {
		return ""
	}
	return b.Helptext(ctx)
}

func (b BaseProvider) WidgetIcon() components.Component {
	return b.Icon
}

func (b BaseProvider) Value() (driver.Value, error) {
	return b.ID, nil
}

func (b *BaseProvider) Scan(v any) error {
	var providerName string
	switch d := v.(type) {
	case string:
		providerName = d
	case []byte:
		providerName = string(d)
	case nil:
		return nil
	default:
		return errors.TypeMismatch.Wrapf(
			"%T is not of type string or []byte", d,
		)
	}

	if providerName == "" {
		return nil
	}

	p, ok := Payments.Get(providerName)
	if !ok {
		// unsure if erroring here is smart. we'll find out! :^)
		return errors.NoResults.Wrapf(
			"provider %q not found in registry", providerName,
		)
	}

	*b = p.Base()

	return nil
}

var Payments = orderedmap.NewOrderedMap[string, BasePaymentProvider]()
