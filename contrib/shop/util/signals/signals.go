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
	"github.com/Nigel2392/mux/middleware/authentication"
)

// var (
// productPool = signals.NewPool[]()
// cartPool    = signals.NewPool[*CartSignalData]()
// orderPool   = signals.NewPool[*OrderSignalData]()
// paymentPool = signals.NewPool[*PaymentSignalData]()
// )

type BaseSignal struct {
	Context   context.Context `json:"-"`
	Timestamp time.Time       `json:"timestamp"`
	User      users.User      `json:"user"`
	IPAddr    net.IP          `json:"ipaddr"`
	Meta      map[string]any  `json:"meta"`
}

type Signal[T any] = signals.Signal[T]

func NewBaseSignal(ctx context.Context, r *http.Request, user users.User, meta map[string]any) BaseSignal {
	var (
		ipAddr = django.GetIP(r)
		netIp  = net.ParseIP(ipAddr)
		newM   = make(map[string]any)
	)

	if user == nil && r != nil {
		u, ok := authentication.Retrieve(r).(users.User)
		if ok {
			user = u
		}
	}

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

type Changed[TYP any] struct {
	Old TYP
	New TYP
}

type ProductSignalData struct {
	BaseSignal
	Product Changed[*models.Product]
	Skus    Changed[[]*models.ProductSku]
}

type OrderSignalData struct {
	BaseSignal
	Last       *models.Order
	Current    *models.Order
	OrderItems []*models.OrderItem
	Reason     string
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
			Created: signals.Get[*ProductSignalData]("products:created"),
			Updated: signals.Get[*ProductSignalData]("products:updated"),
			Deleted: signals.Get[*ProductSignalData]("products:deleted"),
		},
		Cart: ModelManager[*CartSignalData]{
			Created: signals.Get[*CartSignalData]("cart:created"),
			Updated: signals.Get[*CartSignalData]("cart:updated"),
			Deleted: signals.Get[*CartSignalData]("cart:deleted"),
		},
		Orders: ModelManager[*OrderSignalData]{
			Created: signals.Get[*OrderSignalData]("orders:created"),
			Updated: signals.Get[*OrderSignalData]("orders:updated"),
			Deleted: signals.Get[*OrderSignalData]("orders:deleted"),
		},
		Payment: PaymentManager{
			Started:   signals.Get[*PaymentSignalData]("payments:started"),
			Cancelled: signals.Get[*PaymentSignalData]("payments:cancelled"),
			Refunded:  signals.Get[*PaymentSignalData]("payments:refunded"),
		},
	}
}
