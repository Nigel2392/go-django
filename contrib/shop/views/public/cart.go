package public

import "github.com/Nigel2392/go-django/contrib/shop/models"

type CartResponse struct {
	Cart  *models.Cart       `json:"cart,omitempty"`
	Items []*models.CartItem `json:"items,omitempty"`
}
