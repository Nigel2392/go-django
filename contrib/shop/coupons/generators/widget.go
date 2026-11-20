package generators

import (
	"context"
	"io"
	"reflect"
	"time"

	"github.com/Nigel2392/go-django/contrib/shop/coupons"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/go-django/src/forms/fields"
	"github.com/Nigel2392/go-django/src/forms/media"
	"github.com/Nigel2392/go-django/src/forms/widgets"
)

func init() {
	attrs.RegisterFormFieldType(reflect.TypeFor[coupons.CouponCode](), func(opts ...func(fields.Field)) fields.Field {
		return fields.CharField(append(opts, fields.Widget(NewCouponGeneratorWidget(nil)))...)
	})
}

type CouponGeneratorWidget struct {
	*widgets.BaseWidget
}

func NewCouponGeneratorWidget(attrs map[string]string) widgets.Widget {
	return &CouponGeneratorWidget{BaseWidget: widgets.NewBaseWidget("text", "shop/admin/widgets/coupon_generator.tmpl", attrs)}
}

func (b *CouponGeneratorWidget) Media() media.Media {
	m := media.NewMedia()
	m.AddCSS(media.CSS(django.Static("shop/admin/css/widgets/coupon-generator.css")))
	m.AddJS(media.JS(django.Static("shop/admin/js/coupon-generator.js")))
	return m
}

func (b *CouponGeneratorWidget) GetContextData(c context.Context, id string, name string, value interface{}, attrs map[string]string) ctx.Context {
	base := b.BaseWidget.GetContextData(c, id, name, value, attrs)
	time := time.Now()
	base.Set("generator", map[string]any{
		"url": django.Reverse("admin:ext:shop:generate_coupon"),
		"time": map[string]int{
			"year":    time.Year(),
			"month":   int(time.Month()),
			"weekday": int(time.Weekday()),
			"day":     time.Day(),
			"hour":    time.Hour(),
			"minute":  time.Minute(),
		},
		"i18n": map[string]string{
			"Length":  trans.T(c, "Length"),
			"Value":   trans.T(c, "Value"),
			"Refresh": trans.T(c, "Refresh"),

			"CPT_RANDOM":  trans.T(c, "Random String, requires length"),
			"CPT_STATIC":  trans.T(c, "Static String, requires custom input"),
			"CPT_YEAR":    trans.T(c, "Current Year"),
			"CPT_MONTH":   trans.T(c, "Current Month"),
			"CPT_WEEKDAY": trans.T(c, "Current Weekday"),
			"CPT_DAY":     trans.T(c, "Current day of the year"),
			"CPT_HOUR":    trans.T(c, "Current hour"),
			"CPT_MINUTE":  trans.T(c, "Current minute"),
			"CPT_UUID":    trans.T(c, "Random UUID, requires length"),
		},
	})
	return base
}

func (b *CouponGeneratorWidget) Render(ctx context.Context, w io.Writer, id, name string, value interface{}, attrs map[string]string) error {
	return b.RenderWithErrors(ctx, w, id, name, value, nil, attrs, b.GetContextData(ctx, id, name, value, attrs))
}
