package payments

import (
	"context"

	"github.com/Nigel2392/go-django/contrib/shop/internal/prov"
	"github.com/Nigel2392/go-django/contrib/shop/models"
)

type PaymentProvider interface {
	prov.BasePaymentProvider

	CreateTransaction(ctx context.Context, order *models.Order) (*models.Payment, error)
	ParseWebhook(ctx context.Context, rawPayload []byte) (*PaymentSignalPayload, error)
	Cancel(ctx context.Context, payment *models.Payment) error
}

type PaymentSignalPayload struct {
	ProviderName  string
	TransactionID string
	OrderID       string
	Status        string
	Amount        uint64
	RawPayload    []byte
}

func RegisterProvider(p PaymentProvider) {
	prov.Payments.Set(p.ProviderID(), p)
}
