package payments

import (
	"context"

	"github.com/elliotchance/orderedmap/v2"
)

type PaymentProvider interface {
	Name() string
	Label(context.Context) string
}

type PaymentSignalPayload struct {
	ProviderName  string
	TransactionID string
	OrderID       string
	Status        string
	Amount        uint64
	RawPayload    []byte
}

type Payments struct {
	Prodivers []PaymentProvider
	providers *orderedmap.OrderedMap[string, PaymentProvider]
}
