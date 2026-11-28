package orders

import (
	"context"
	"fmt"
	"html/template"
	"strings"
	"time"

	"github.com/Nigel2392/go-django/contrib/shop/core/util/paystate"
	"github.com/Nigel2392/go-django/contrib/shop/models/payments"
	"github.com/Nigel2392/go-django/src/core/trans"
	"golang.org/x/text/currency"
)

func paymentHistoryHTML(ctx context.Context, payment *payments.Payment) template.HTML {
	var html strings.Builder
	escape := template.HTMLEscapeString

	summary := fmt.Sprintf("%s %s · %s", payment.Amount, payment.Currency, paystate.PAYSTATE.StringOf(ctx, payment.State))
	if !payment.IsFreeCheckout() {
		summary = payment.Provider + " · " + summary
	}

	fmt.Fprintf(&html,
		"<p>%s</p>",
		escape(summary),
	)

	if payment.Currency != "" {
		fmt.Fprintf(&html,
			"<p>%s · %s</p>",
			escape(trans.T(ctx, "Subtotal before payment fee: %s %s", payment.Amount.Sub(payment.Fee), payment.Currency)),
			escape(trans.T(ctx, "Customer payment fee: %s %s", payment.Fee, payment.Currency)),
		)
	}
	if payment.RefundedAmount.IsPositive() && payment.Currency != "" {
		fmt.Fprintf(&html,
			"<p>%s</p>",
			escape(trans.T(ctx, "Refunded: %s %s", payment.RefundedAmount, payment.Currency)),
		)
	}
	if fee := payment.ProviderFee.Data; fee != nil {
		fmt.Fprintf(&html,
			"<p>%s</p>",
			escape(trans.T(ctx, "Provider processing fee: %s %s", fee.Amount, fee.Currency)),
		)
	}

	if len(payment.History.Data) > 0 {
		html.WriteString("<ol>")
		for _, event := range payment.History.Data {
			var label string
			switch event.Kind {
			case payments.EventNoPaymentRequired:
				label = trans.T(ctx, "Order placed without payment")
			case payments.EventCheckoutOpened:
				label = trans.T(ctx, "Provider checkout opened")
			case payments.EventStatusUpdated:
				label = trans.T(ctx, "Provider status updated")
			case payments.EventProviderFeeUpdated:
				label = trans.T(ctx, "Provider processing fee reported")
			case payments.EventRefunded:
				label = trans.T(ctx, "Payment refunded")
			case payments.EventCancelled:
				label = trans.T(ctx, "Payment cancelled")
			default:
				label = string(event.Kind)
			}

			at := event.At.UTC()
			fmt.Fprintf(&html,
				`<li><time datetime="%s">%s</time> · %s · %s`,
				at.Format(time.RFC3339Nano),
				at.Format("2006-01-02 15:04:05 UTC"),
				escape(label),
				escape(paystate.PAYSTATE.StringOf(ctx, event.State)),
			)

			if event.Amount != nil && event.Amount.Currency != (currency.Unit{}) {
				fmt.Fprintf(&html,
					" · %s",
					escape(fmt.Sprintf("%s %s", event.Amount.Amount, event.Amount.Currency)),
				)
			}
			if event.RequestID != "" {
				fmt.Fprintf(&html, " · <code>%s</code>", escape(event.RequestID))
			}
			if event.ProviderRef != "" {
				fmt.Fprintf(&html, " · <code>%s</code>", escape(event.ProviderRef))
			}
			if fee := event.ProviderFee; fee != nil {
				fmt.Fprintf(&html,
					" · %s",
					escape(trans.T(ctx, "Provider processing fee: %s %s", fee.Amount, fee.Currency)),
				)
			}

			html.WriteString("</li>")
		}

		html.WriteString("</ol>")
	} else {
		fmt.Fprintf(
			&html, "<p>%s</p>",
			escape(trans.T(ctx, "No lifecycle events recorded for this payment.")),
		)
	}

	return template.HTML(html.String())
}
