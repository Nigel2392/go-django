package signals

import (
	"context"
	"maps"
	"net"
	"net/http"
	"time"

	"github.com/Nigel2392/go-django/contrib/auth/users"
	"github.com/Nigel2392/go-django/contrib/shop/internal/prov"
	"github.com/Nigel2392/go-django/contrib/shop/models"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-signals"
)

var (
	productPool = signals.NewPool[*ProductSignalData]()
	cartPool    = signals.NewPool[*CartSignalData]()
	orderPool   = signals.NewPool[*OrderSignalData]()
	paymentPool = signals.NewPool[*PaymentSignalData]()
)

type BaseSignal struct {
	Context   context.Context
	Timestamp time.Time
	User      users.User
	IPAddr    net.IP
	Meta      map[string]any
}

func NewBaseSignal(ctx context.Context, r *http.Request, user users.User, meta map[string]any) BaseSignal {
	var (
		ipAddr = django.GetIP(r)
		netIp  = net.ParseIP(ipAddr)
		newM   = make(map[string]any)
	)

	maps.Copy(newM, meta)

	newM["request"] = map[string]any{
		"method":     r.Method,
		"user_agent": r.UserAgent(),
		"url":        r.URL,
	}

	return BaseSignal{
		Context:   ctx,
		Timestamp: time.Now(),
		User:      user,
		IPAddr:    netIp,
		Meta:      meta,
	}
}

type PaymentSignalData struct {
	BaseSignal
	Provider prov.BaseProvider
	Last     *models.Payment
	Payment  *models.Payment
	Order    *models.Order
	RawBytes []byte
}

type ProductSignalData struct {
	BaseSignal
	Last    *models.Product
	Product *models.Product
	Skus    []*models.ProductSku
}

type OrderSignalData struct {
	BaseSignal
	Last    *models.Order
	Current *models.Order
	Reason  string
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
