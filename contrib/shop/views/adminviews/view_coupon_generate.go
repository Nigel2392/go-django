package adminviews

import (
	"net/http"

	"github.com/Nigel2392/go-django/contrib/shop/coupons/generators"
)

type CouponGenerationRequest struct {
	Parts []generators.CodePart `json:"code_format_parts"`
}

func GenerateCouponCode(w http.ResponseWriter, r *http.Request) {

}
