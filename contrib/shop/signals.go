package shop

import (
	"github.com/Nigel2392/go-django/contrib/shop/models"
	"github.com/Nigel2392/go-signals"
)

type PaymentSignalData struct {
	Payment  *models.Payment
	RawBytes []byte
}

type ProductSignalData struct {
	Product *models.Product
	Sku     *models.ProductSku
}

type OrderSignalData struct {
	Order *models.Order
}

type CartSignalData struct {
	Cart *models.Cart
}

type ModelSignalManager[SIGNAL_T any] struct {
	Created signals.Signal[SIGNAL_T]
	Updated signals.Signal[SIGNAL_T]
	Deleted signals.Signal[SIGNAL_T]
}

type PaymentSignalManager struct {
	Started   signals.Signal[*PaymentSignalData]
	Cancelled signals.Signal[*PaymentSignalData]
	Refunded  signals.Signal[*PaymentSignalData]
}

type SignalManager struct {
	Products ModelSignalManager[*ProductSignalData]
	Cart     ModelSignalManager[*CartSignalData]
	Orders   ModelSignalManager[*OrderSignalData]
	Payment  PaymentSignalManager
}

var (
	productPool = signals.NewPool[*ProductSignalData]()
	cartPool    = signals.NewPool[*CartSignalData]()
	orderPool   = signals.NewPool[*OrderSignalData]()
	paymentPool = signals.NewPool[*PaymentSignalData]()
)

func newSignalManager() SignalManager {
	return SignalManager{
		Products: ModelSignalManager[*ProductSignalData]{
			Created: productPool.Get("products:created"),
			Updated: productPool.Get("products:updated"),
			Deleted: productPool.Get("products:deleted"),
		},
		Cart: ModelSignalManager[*CartSignalData]{
			Created: cartPool.Get("cart:created"),
			Updated: cartPool.Get("cart:updated"),
			Deleted: cartPool.Get("cart:deleted"),
		},
		Orders: ModelSignalManager[*OrderSignalData]{
			Created: orderPool.Get("orders:created"),
			Updated: orderPool.Get("orders:updated"),
			Deleted: orderPool.Get("orders:deleted"),
		},
		Payment: PaymentSignalManager{
			Started:   paymentPool.Get("payments:started"),
			Cancelled: paymentPool.Get("payments:cancelled"),
			Refunded:  paymentPool.Get("payments:refunded"),
		},
	}
}
