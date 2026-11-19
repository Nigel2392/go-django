package adminviews

import (
	"net/http"

	"github.com/Nigel2392/go-django/contrib/admin"
	"github.com/Nigel2392/go-django/contrib/shop/coupons"
	"github.com/Nigel2392/go-django/contrib/shop/coupons/generators"
	"github.com/Nigel2392/go-django/contrib/shop/shared/api"
	"github.com/Nigel2392/go-django/src/core/except"
	"github.com/Nigel2392/go-django/src/permissions"
)

type CouponGenerationRequest struct {
	Parts []generators.CodePart `json:"code_format_parts"`
}

type CouponGenerationResponse struct {
	CouponCode string `json:"coupon_code"`
}

func GenerateCouponCode(w http.ResponseWriter, r *http.Request) {
	if !permissions.HasObjectPermission(r, &coupons.Coupon{}, "coupons:generate") {
		admin.ReLogin(w, r, r.URL.Path)
		return
	}

	request := CouponGenerationRequest{}
	err := api.Decode(r, &request)
	if err != nil {
		api.RespondError(w,
			http.StatusBadRequest,
			"failed to read or decode request body: %v", err,
		)
		return
	}

	code, err := generators.GenerateCouponCode(r.Context(), request.Parts)
	if err != nil {
		api.RespondError(w,
			http.StatusBadRequest,
			"failed to generate a new coupon code: %v", err,
		)
		return
	}

	err = api.Respond(w, CouponGenerationResponse{
		CouponCode: code,
	})
	if err != nil {
		except.Fail(
			http.StatusInternalServerError,
			"failed to encode the code generation response: %v", err,
		)
	}
}
