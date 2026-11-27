package price

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/shopspring/decimal"
	"golang.org/x/text/currency"
)

var decimal100 = decimal.NewFromInt(100)

// Money pairs a decimal amount with a developer-controlled ISO currency unit.
type Money struct {
	Amount   decimal.Decimal
	Currency currency.Unit
}

type moneyJSON struct {
	Amount   decimal.Decimal `json:"amount"`
	Currency string          `json:"currency"`
}

func (m Money) MarshalJSON() ([]byte, error) {
	return json.Marshal(moneyJSON{
		Amount:   m.Amount,
		Currency: formatCurrencyUnit(m.Currency),
	})
}

func (m *Money) UnmarshalJSON(data []byte) error {
	var raw moneyJSON
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	unit, err := parseCurrencyUnit(raw.Currency)
	if err != nil {
		return err
	}
	m.Amount = raw.Amount
	m.Currency = unit
	return nil
}

// Quote adds a customer-facing payment fee to m and rounds both subtotal and fee to the currency's scale.
func (m Money) Quote(fixedFee, percentageFee decimal.Decimal) (Quote, error) {
	return QuoteFee(m, fixedFee, percentageFee)
}

// Quote represents a currency-rounded subtotal, payment fee, and total in a developer-controlled currency unit.
type Quote struct {
	Subtotal decimal.Decimal
	Fee      decimal.Decimal
	Total    decimal.Decimal
	Currency currency.Unit
}

type quoteJSON struct {
	Subtotal decimal.Decimal `json:"subtotal"`
	Fee      decimal.Decimal `json:"fee"`
	Total    decimal.Decimal `json:"total"`
	Currency string          `json:"currency"`
}

func (q Quote) MarshalJSON() ([]byte, error) {
	return json.Marshal(quoteJSON{
		Subtotal: q.Subtotal,
		Fee:      q.Fee,
		Total:    q.Total,
		Currency: formatCurrencyUnit(q.Currency),
	})
}

func (q *Quote) UnmarshalJSON(data []byte) error {
	var raw quoteJSON
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	unit, err := parseCurrencyUnit(raw.Currency)
	if err != nil {
		return err
	}
	q.Subtotal = raw.Subtotal
	q.Fee = raw.Fee
	q.Total = raw.Total
	q.Currency = unit
	return nil
}

// QuoteFee rounds subtotal.Amount to subtotal.Currency's standard ISO scale and adds
// fixedFee + base * percentageFee / 100 (rounded to scale) when the rounded base is positive.
func QuoteFee(subtotal Money, fixedFee, percentageFee decimal.Decimal) (Quote, error) {
	if subtotal.Currency == (currency.Unit{}) {
		return Quote{}, errors.New("currency unit is required")
	}
	if subtotal.Amount.IsNegative() || fixedFee.IsNegative() || percentageFee.IsNegative() {
		return Quote{}, errors.New("amount and fees must be nonnegative")
	}
	scale, _ := currency.Standard.Rounding(subtotal.Currency)
	base := subtotal.Amount.Round(int32(scale))
	fee := decimal.Zero
	if base.IsPositive() {
		fee = fixedFee.Add(base.Mul(percentageFee).Shift(-2)).Round(int32(scale))
	}
	return Quote{
		Subtotal: base,
		Fee:      fee,
		Total:    base.Add(fee),
		Currency: subtotal.Currency,
	}, nil
}

func formatCurrencyUnit(unit currency.Unit) string {
	if unit == (currency.Unit{}) {
		return ""
	}
	return unit.String()
}

func parseCurrencyUnit(code string) (currency.Unit, error) {
	if code == "" {
		return currency.Unit{}, nil
	}
	unit, err := currency.ParseISO(code)
	if err != nil || unit.String() != code {
		return currency.Unit{}, fmt.Errorf("invalid ISO currency %q", code)
	}
	return unit, nil
}

type AmountValue struct {
	Amount       decimal.Decimal
	IsPercentage bool
}

type Price struct {
	ItemPrice     decimal.Decimal `json:"item_price"`     // currency
	ItemFee       AmountValue     `json:"item_fee"`       // currency
	TaxPercentage decimal.Decimal `json:"tax_percentage"` // percentage
}

type Totals struct {
	TotalPrice       decimal.Decimal `json:"total_price"`        // currency
	TotalFee         decimal.Decimal `json:"total_fee"`          // currency
	TotalTax         decimal.Decimal `json:"total_tax"`          // currency
	TotalDiscount    decimal.Decimal `json:"total_discount"`     // currency
	LeftOverDiscount decimal.Decimal `json:"left_over_discount"` // currency
}

// Payable returns the amount after discounts, including product fees and tax.
func (t Totals) Payable() decimal.Decimal {
	return t.TotalPrice.Add(t.TotalFee).Add(t.TotalTax)
}

type PriceCalculation struct {
	ItemCount int
	Discount  AmountValue
}

func (p Price) Calculate(opts PriceCalculation) (totalPriceInclTax decimal.Decimal, calculatedPrice Totals) {
	var (
		itemCount        = decimal.NewFromInt(int64(opts.ItemCount))
		totalPrice       = p.ItemPrice.Mul(itemCount)
		totalFee         = decimal.Zero
		totalDiscount    = decimal.Zero
		leftOverDiscount = decimal.Zero
	)

	if !opts.Discount.Amount.IsZero() {
		if opts.Discount.IsPercentage {
			// Cap percentage at 100% to prevent negative prices
			pct := opts.Discount.Amount
			if pct.GreaterThan(decimal100) {
				pct = decimal100
			}

			discountPrice := totalPrice.Mul(pct).Div(decimal100)
			totalPrice = totalPrice.Sub(discountPrice)
			totalDiscount = totalDiscount.Add(discountPrice)

		} else { // Fixed discount
			if opts.Discount.Amount.GreaterThanOrEqual(totalPrice) {
				// Korting is groter of gelijk aan de productprijs
				totalDiscount = totalDiscount.Add(totalPrice)
				leftOverDiscount = opts.Discount.Amount.Sub(totalPrice)
				totalPrice = decimal.Zero
			} else {
				// Korting is kleiner dan de productprijs
				totalPrice = totalPrice.Sub(opts.Discount.Amount)
				totalDiscount = totalDiscount.Add(opts.Discount.Amount)
			}
		}
	}

	// Bereken de fee over de prijs NA korting
	if !p.ItemFee.Amount.IsZero() {
		if p.ItemFee.IsPercentage {
			totalFee = totalPrice.Mul(p.ItemFee.Amount).Div(decimal100)
		} else {
			totalFee = p.ItemFee.Amount.Mul(itemCount)
		}
	}

	// Als er nog vaste korting over is (omdat het product al gratis was), pas dit toe op de fee
	if !opts.Discount.IsPercentage && leftOverDiscount.GreaterThan(decimal.Zero) {
		if leftOverDiscount.GreaterThanOrEqual(totalFee) {
			totalDiscount = totalDiscount.Add(totalFee)
			leftOverDiscount = leftOverDiscount.Sub(totalFee)
			totalFee = decimal.Zero
		} else {
			totalFee = totalFee.Sub(leftOverDiscount)
			totalDiscount = totalDiscount.Add(leftOverDiscount)
			leftOverDiscount = decimal.Zero
		}
	}

	// Bereken belasting over het daadwerkelijk te betalen bedrag (allPrice)
	var (
		allPrice = totalPrice.Add(totalFee)
		totalTax = allPrice.Mul(p.TaxPercentage).Div(decimal100)
	)

	// Vul de output struct
	totalPriceInclTax = allPrice.Add(totalTax)

	calculatedPrice.TotalPrice = totalPrice
	calculatedPrice.TotalFee = totalFee
	calculatedPrice.TotalTax = totalTax
	calculatedPrice.TotalDiscount = totalDiscount
	calculatedPrice.LeftOverDiscount = leftOverDiscount

	return totalPriceInclTax, calculatedPrice
}
