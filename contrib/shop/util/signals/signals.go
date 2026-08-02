package signals

import (
	"context"

	"github.com/Nigel2392/go-django/contrib/shop/models"
	"github.com/Nigel2392/go-django/contrib/shop/payments"
	"github.com/Nigel2392/go-signals"
)

var (
	productPool = signals.NewPool[*ProductSignalData]()
	cartPool    = signals.NewPool[*CartSignalData]()
	orderPool   = signals.NewPool[*OrderSignalData]()
	paymentPool = signals.NewPool[*PaymentSignalData]()
)

type BaseSignal struct {
	Context context.Context
}

type PaymentSignalData struct {
	BaseSignal
	Payment  *payments.Payment
	RawBytes []byte
}

type ProductSignalData struct {
	BaseSignal
	Product *models.Product
	Sku     *models.ProductSku
}

type OrderSignalData struct {
	BaseSignal
	Order *models.Order
}

type CartSignalData struct {
	BaseSignal
	Cart *models.Cart
}

type ModelManager[SIGNAL_T any] struct {
	Created signals.Signal[SIGNAL_T]
	Updated signals.Signal[SIGNAL_T]
	Deleted signals.Signal[SIGNAL_T]
}

type PaymentManager struct {
	Started   signals.Signal[*PaymentSignalData]
	Cancelled signals.Signal[*PaymentSignalData]
	Refunded  signals.Signal[*PaymentSignalData]
}

type Manager struct {
	Products ModelManager[*ProductSignalData]
	Cart     ModelManager[*CartSignalData]
	Orders   ModelManager[*OrderSignalData]
	Payment  PaymentManager
}

func NewManager() Manager {
	return Manager{
		Products: ModelManager[*ProductSignalData]{
			Created: productPool.Get("products:created"),
			Updated: productPool.Get("products:updated"),
			Deleted: productPool.Get("products:deleted"),
		},
		Cart: ModelManager[*CartSignalData]{
			Created: cartPool.Get("cart:created"),
			Updated: cartPool.Get("cart:updated"),
			Deleted: cartPool.Get("cart:deleted"),
		},
		Orders: ModelManager[*OrderSignalData]{
			Created: orderPool.Get("orders:created"),
			Updated: orderPool.Get("orders:updated"),
			Deleted: orderPool.Get("orders:deleted"),
		},
		Payment: PaymentManager{
			Started:   paymentPool.Get("payments:started"),
			Cancelled: paymentPool.Get("payments:cancelled"),
			Refunded:  paymentPool.Get("payments:refunded"),
		},
	}
}
