package price

import "github.com/shopspring/decimal"

var decimal100 = decimal.NewFromInt(100)

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
