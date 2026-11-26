package orders

import (
	"context"
	"fmt"
	"html/template"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/Nigel2392/go-django/contrib/shop/payments"
	"github.com/Nigel2392/go-django/contrib/shop/util/paystate"
	"github.com/Nigel2392/go-django/src/core/trans"
)

func paymentHistoryHTML(ctx context.Context, payment *payments.Payment) template.HTML {
	var html strings.Builder
	escape := template.HTMLEscapeString

	summary := fmt.Sprintf(
		"%s · %s %s · %s",
		payment.Provider,
		payment.Amount,
		payment.Currency,
		paystate.PAYSTATE.StringOf(ctx, payment.State),
	)

	fmt.Fprintf(&html,
		"<p>%s</p>",
		escape(summary),
	)

	recordedOperations := make(map[string]bool)
	if len(payment.History.Data) > 0 {
		html.WriteString("<ol>")
		for _, event := range payment.History.Data {
			var label string
			switch event.Kind {
			case payments.EventPrepared:
				label = trans.T(ctx, "Payment prepared")
			case payments.EventCheckoutOpened:
				label = trans.T(ctx, "Provider checkout opened")
			case payments.EventCheckoutFailed:
				label = trans.T(ctx, "Provider checkout failed to open")
			case payments.EventStatusUpdated:
				label = trans.T(ctx, "Provider status updated")
			case payments.EventOperation:
				label = paymentOperationLabel(ctx, event.Operation, event.Phase, event.Amount)
				recordedOperations[event.RequestID] = true
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

			if event.RequestID != "" {
				fmt.Fprintf(&html, " · <code>%s</code>", escape(event.RequestID))
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

	if len(payment.Operations.Data) > len(recordedOperations) {
		fmt.Fprintf(&html,
			"<p>%s</p><ul>",
			escape(trans.T(ctx, "Earlier operations without timestamps")),
		)

		for _, id := range slices.Sorted(maps.Keys(payment.Operations.Data)) {
			if recordedOperations[id] {
				continue
			}

			record := payment.Operations.Data[id]
			label := paymentOperationLabel(
				ctx,
				record.Kind,
				record.Phase,
				&record.Amount,
			)

			fmt.Fprintf(&html,
				"<li>%s · <code>%s</code></li>",
				escape(label),
				escape(id),
			)
		}
		html.WriteString("</ul>")
	}

	return template.HTML(html.String())
}

func paymentOperationLabel(ctx context.Context, kind payments.OperationKind, phase payments.OperationState, amount *payments.Money) string {
	var operation string
	switch kind {
	case payments.OperationCapture:
		operation = trans.T(ctx, "Capture")
	case payments.OperationRefund:
		operation = trans.T(ctx, "Refund")
	case payments.OperationVoid:
		operation = trans.T(ctx, "Void")
	case payments.OperationCancel:
		operation = trans.T(ctx, "Cancel")
	default:
		operation = string(kind)
	}

	var outcome string
	switch phase {
	case payments.OperationRequested:
		outcome = trans.T(ctx, "Requested")
	case payments.OperationPending:
		outcome = trans.T(ctx, "Pending")
	case payments.OperationUncertain:
		outcome = trans.T(ctx, "Outcome unknown")
	case payments.OperationSucceeded:
		outcome = trans.T(ctx, "Succeeded")
	case payments.OperationFailed:
		outcome = trans.T(ctx, "Failed")
	default:
		outcome = string(phase)
	}

	if amount != nil && amount.Currency != "" {
		return fmt.Sprintf(
			"%s · %s · %s %s",
			operation,
			outcome,
			amount.Amount,
			amount.Currency,
		)
	}

	return fmt.Sprintf(
		"%s · %s",
		operation,
		outcome,
	)
}
