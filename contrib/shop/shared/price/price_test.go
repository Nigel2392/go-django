package price

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

// Helper to easily create decimals in tests
func d(val string) decimal.Decimal {
	dec, _ := decimal.NewFromString(val)
	return dec
}

func TestPrice_Calculate(t *testing.T) {
	tests := []struct {
		name             string
		price            Price
		opts             PriceCalculation
		wantTotalInclTax decimal.Decimal
		wantCalcPrice    Totals
	}{
		// ----------------------------------------------------------------
		// 1. STANDARD SCENARIOS (No Discounts)
		// ----------------------------------------------------------------
		{
			name: "Standard purchase without fee or discount",
			price: Price{
				ItemPrice:     d("100"),
				TaxPercentage: d("21"), // 21%
			},
			opts: PriceCalculation{
				ItemCount: 1,
			},
			wantTotalInclTax: d("121"),
			wantCalcPrice: Totals{
				TotalPrice:       d("100"),
				TotalFee:         d("0"),
				TotalTax:         d("21"),
				TotalDiscount:    d("0"),
				LeftOverDiscount: d("0"),
			},
		},
		{
			name: "Multiple items without fee or discount",
			price: Price{
				ItemPrice:     d("50"),
				TaxPercentage: d("21"),
			},
			opts: PriceCalculation{
				ItemCount: 3,
			},
			wantTotalInclTax: d("181.50"), // (150) + 31.50 tax
			wantCalcPrice: Totals{
				TotalPrice:       d("150"),
				TotalFee:         d("0"),
				TotalTax:         d("31.50"),
				TotalDiscount:    d("0"),
				LeftOverDiscount: d("0"),
			},
		},
		{
			name: "Standard purchase with fixed fee",
			price: Price{
				ItemPrice:     d("100"),
				ItemFee:       AmountValue{Amount: d("10"), IsPercentage: false},
				TaxPercentage: d("21"),
			},
			opts: PriceCalculation{
				ItemCount: 1,
			},
			wantTotalInclTax: d("133.10"), // (100 + 10) + 23.10 tax
			wantCalcPrice: Totals{
				TotalPrice:       d("100"),
				TotalFee:         d("10"),
				TotalTax:         d("23.10"),
				TotalDiscount:    d("0"),
				LeftOverDiscount: d("0"),
			},
		},
		{
			name: "Multiple items with fixed fee",
			price: Price{
				ItemPrice:     d("100"),
				ItemFee:       AmountValue{Amount: d("10"), IsPercentage: false},
				TaxPercentage: d("21"),
			},
			opts: PriceCalculation{
				ItemCount: 2, // Fee scales with item count
			},
			wantTotalInclTax: d("266.20"), // (200 + 20) + 46.20 tax
			wantCalcPrice: Totals{
				TotalPrice:       d("200"),
				TotalFee:         d("20"),
				TotalTax:         d("46.20"),
				TotalDiscount:    d("0"),
				LeftOverDiscount: d("0"),
			},
		},
		{
			name: "Standard purchase with percentage fee",
			price: Price{
				ItemPrice:     d("100"),
				ItemFee:       AmountValue{Amount: d("5"), IsPercentage: true},
				TaxPercentage: d("21"),
			},
			opts: PriceCalculation{
				ItemCount: 1,
			},
			wantTotalInclTax: d("127.05"), // (100 + 5) + 22.05 tax
			wantCalcPrice: Totals{
				TotalPrice:       d("100"),
				TotalFee:         d("5"),
				TotalTax:         d("22.05"),
				TotalDiscount:    d("0"),
				LeftOverDiscount: d("0"),
			},
		},
		{
			name: "Multiple items with percentage fee",
			price: Price{
				ItemPrice:     d("100"),
				ItemFee:       AmountValue{Amount: d("5"), IsPercentage: true},
				TaxPercentage: d("21"),
			},
			opts: PriceCalculation{
				ItemCount: 2,
			},
			wantTotalInclTax: d("254.10"), // (200 + 10) + 44.10 tax
			wantCalcPrice: Totals{
				TotalPrice:       d("200"),
				TotalFee:         d("10"),
				TotalTax:         d("44.10"),
				TotalDiscount:    d("0"),
				LeftOverDiscount: d("0"),
			},
		},

		// ----------------------------------------------------------------
		// 2. PERCENTAGE DISCOUNTS
		// ----------------------------------------------------------------
		{
			name: "Percentage discount no fee",
			price: Price{
				ItemPrice:     d("100"),
				TaxPercentage: d("21"),
			},
			opts: PriceCalculation{
				ItemCount: 1,
				Discount:  AmountValue{Amount: d("20"), IsPercentage: true},
			},
			wantTotalInclTax: d("96.80"), // 80 + 16.80 tax
			wantCalcPrice: Totals{
				TotalPrice:       d("80"),
				TotalFee:         d("0"),
				TotalTax:         d("16.80"),
				TotalDiscount:    d("20"),
				LeftOverDiscount: d("0"),
			},
		},
		{
			name: "Percentage discount exactly 100%",
			price: Price{
				ItemPrice:     d("100"),
				TaxPercentage: d("21"),
			},
			opts: PriceCalculation{
				ItemCount: 1,
				Discount:  AmountValue{Amount: d("100"), IsPercentage: true},
			},
			wantTotalInclTax: d("0"),
			wantCalcPrice: Totals{
				TotalPrice:       d("0"),
				TotalFee:         d("0"),
				TotalTax:         d("0"),
				TotalDiscount:    d("100"),
				LeftOverDiscount: d("0"),
			},
		},
		{
			name: "Percentage discount exceeding 100% caps at 100%",
			price: Price{
				ItemPrice:     d("100"),
				TaxPercentage: d("21"),
			},
			opts: PriceCalculation{
				ItemCount: 1,
				Discount:  AmountValue{Amount: d("150"), IsPercentage: true},
			},
			wantTotalInclTax: d("0"),
			wantCalcPrice: Totals{
				TotalPrice:       d("0"),
				TotalFee:         d("0"),
				TotalTax:         d("0"),
				TotalDiscount:    d("100"),
				LeftOverDiscount: d("0"), // Percentage discounts don't carry over
			},
		},
		{
			name: "Percentage discount with fixed fee",
			price: Price{
				ItemPrice:     d("100"),
				ItemFee:       AmountValue{Amount: d("10"), IsPercentage: false},
				TaxPercentage: d("21"),
			},
			opts: PriceCalculation{
				ItemCount: 1,
				Discount:  AmountValue{Amount: d("20"), IsPercentage: true},
			},
			wantTotalInclTax: d("108.90"), // (80 + 10) + 18.90 tax
			wantCalcPrice: Totals{
				TotalPrice:       d("80"),
				TotalFee:         d("10"),
				TotalTax:         d("18.90"),
				TotalDiscount:    d("20"),
				LeftOverDiscount: d("0"),
			},
		},
		{
			name: "100% Percentage discount with fixed fee leaves fee intact",
			price: Price{
				ItemPrice:     d("100"),
				ItemFee:       AmountValue{Amount: d("10"), IsPercentage: false},
				TaxPercentage: d("21"),
			},
			opts: PriceCalculation{
				ItemCount: 1,
				Discount:  AmountValue{Amount: d("100"), IsPercentage: true},
			},
			wantTotalInclTax: d("12.10"), // (0 + 10) + 2.10 tax
			wantCalcPrice: Totals{
				TotalPrice:       d("0"),
				TotalFee:         d("10"),
				TotalTax:         d("2.10"),
				TotalDiscount:    d("100"),
				LeftOverDiscount: d("0"),
			},
		},
		{
			name: "Percentage discount with percentage fee lowers fee proportionally",
			price: Price{
				ItemPrice:     d("100"),
				ItemFee:       AmountValue{Amount: d("5"), IsPercentage: true}, // 5% fee
				TaxPercentage: d("21"),
			},
			opts: PriceCalculation{
				ItemCount: 1,
				Discount:  AmountValue{Amount: d("20"), IsPercentage: true},
			},
			wantTotalInclTax: d("101.64"), // (80 + (5% of 80)) + 21% tax -> (80 + 4) + 17.64
			wantCalcPrice: Totals{
				TotalPrice:       d("80"),
				TotalFee:         d("4"),
				TotalTax:         d("17.64"),
				TotalDiscount:    d("20"),
				LeftOverDiscount: d("0"),
			},
		},

		// ----------------------------------------------------------------
		// 3. FIXED DISCOUNTS (No cascading)
		// ----------------------------------------------------------------
		{
			name: "Fixed discount partial coverage no fee",
			price: Price{
				ItemPrice:     d("100"),
				TaxPercentage: d("21"),
			},
			opts: PriceCalculation{
				ItemCount: 1,
				Discount:  AmountValue{Amount: d("25"), IsPercentage: false},
			},
			wantTotalInclTax: d("90.75"), // 75 + 15.75 tax
			wantCalcPrice: Totals{
				TotalPrice:       d("75"),
				TotalFee:         d("0"),
				TotalTax:         d("15.75"),
				TotalDiscount:    d("25"),
				LeftOverDiscount: d("0"),
			},
		},
		{
			name: "Fixed discount exactly equals product price no fee",
			price: Price{
				ItemPrice:     d("100"),
				TaxPercentage: d("21"),
			},
			opts: PriceCalculation{
				ItemCount: 1,
				Discount:  AmountValue{Amount: d("100"), IsPercentage: false},
			},
			wantTotalInclTax: d("0"),
			wantCalcPrice: Totals{
				TotalPrice:       d("0"),
				TotalFee:         d("0"),
				TotalTax:         d("0"),
				TotalDiscount:    d("100"),
				LeftOverDiscount: d("0"),
			},
		},
		{
			name: "Fixed discount exceeds product price no fee",
			price: Price{
				ItemPrice:     d("100"),
				TaxPercentage: d("21"),
			},
			opts: PriceCalculation{
				ItemCount: 1,
				Discount:  AmountValue{Amount: d("150"), IsPercentage: false},
			},
			wantTotalInclTax: d("0"),
			wantCalcPrice: Totals{
				TotalPrice:       d("0"),
				TotalFee:         d("0"),
				TotalTax:         d("0"),
				TotalDiscount:    d("100"),
				LeftOverDiscount: d("50"),
			},
		},

		// ----------------------------------------------------------------
		// 4. FIXED DISCOUNTS (Cascading over into Fees)
		// ----------------------------------------------------------------
		{
			name: "Fixed discount partially covers product, fee remains untouched",
			price: Price{
				ItemPrice:     d("100"),
				ItemFee:       AmountValue{Amount: d("10"), IsPercentage: false},
				TaxPercentage: d("21"),
			},
			opts: PriceCalculation{
				ItemCount: 1,
				Discount:  AmountValue{Amount: d("25"), IsPercentage: false},
			},
			wantTotalInclTax: d("102.85"), // (75 + 10) + 17.85 tax
			wantCalcPrice: Totals{
				TotalPrice:       d("75"),
				TotalFee:         d("10"),
				TotalTax:         d("17.85"),
				TotalDiscount:    d("25"),
				LeftOverDiscount: d("0"),
			},
		},
		{
			name: "Fixed discount clears product and partially covers fixed fee",
			price: Price{
				ItemPrice:     d("100"),
				ItemFee:       AmountValue{Amount: d("10"), IsPercentage: false},
				TaxPercentage: d("21"),
			},
			opts: PriceCalculation{
				ItemCount: 1,
				Discount:  AmountValue{Amount: d("105"), IsPercentage: false},
			},
			wantTotalInclTax: d("6.05"), // (0 + 5) + 1.05 tax
			wantCalcPrice: Totals{
				TotalPrice:       d("0"),
				TotalFee:         d("5"),
				TotalTax:         d("1.05"),
				TotalDiscount:    d("105"),
				LeftOverDiscount: d("0"),
			},
		},
		{
			name: "Fixed discount exactly clears product and fixed fee",
			price: Price{
				ItemPrice:     d("100"),
				ItemFee:       AmountValue{Amount: d("10"), IsPercentage: false},
				TaxPercentage: d("21"),
			},
			opts: PriceCalculation{
				ItemCount: 1,
				Discount:  AmountValue{Amount: d("110"), IsPercentage: false},
			},
			wantTotalInclTax: d("0"),
			wantCalcPrice: Totals{
				TotalPrice:       d("0"),
				TotalFee:         d("0"),
				TotalTax:         d("0"),
				TotalDiscount:    d("110"),
				LeftOverDiscount: d("0"),
			},
		},
		{
			name: "Fixed discount clears product and fixed fee with leftovers",
			price: Price{
				ItemPrice:     d("100"),
				ItemFee:       AmountValue{Amount: d("10"), IsPercentage: false},
				TaxPercentage: d("21"),
			},
			opts: PriceCalculation{
				ItemCount: 1,
				Discount:  AmountValue{Amount: d("150"), IsPercentage: false},
			},
			wantTotalInclTax: d("0"),
			wantCalcPrice: Totals{
				TotalPrice:       d("0"),
				TotalFee:         d("0"),
				TotalTax:         d("0"),
				TotalDiscount:    d("110"), // Maxed out at product (100) + fee (10)
				LeftOverDiscount: d("40"),
			},
		},
		{
			name: "The 9 / 1 / 10 scenario from prompt",
			price: Price{
				ItemPrice:     d("9"),
				ItemFee:       AmountValue{Amount: d("1"), IsPercentage: false},
				TaxPercentage: d("21"),
			},
			opts: PriceCalculation{
				ItemCount: 1,
				Discount:  AmountValue{Amount: d("10"), IsPercentage: false},
			},
			wantTotalInclTax: d("0"),
			wantCalcPrice: Totals{
				TotalPrice:       d("0"),
				TotalFee:         d("0"),
				TotalTax:         d("0"),
				TotalDiscount:    d("10"),
				LeftOverDiscount: d("0"),
			},
		},

		// ----------------------------------------------------------------
		// 5. COMPLEX / MULTIPLE ITEMS CASCADING
		// ----------------------------------------------------------------
		{
			name: "Multiple items, fixed discount clears all products and partial fee",
			price: Price{
				ItemPrice:     d("50"),
				ItemFee:       AmountValue{Amount: d("10"), IsPercentage: false},
				TaxPercentage: d("20"), // Clean 20% for easy math
			},
			opts: PriceCalculation{
				ItemCount: 2, // Total Price = 100, Total Fee = 20
				Discount:  AmountValue{Amount: d("110"), IsPercentage: false},
			},
			wantTotalInclTax: d("12"), // (0 + 10) + 2.00 tax
			wantCalcPrice: Totals{
				TotalPrice:       d("0"),
				TotalFee:         d("10"),
				TotalTax:         d("2"),
				TotalDiscount:    d("110"),
				LeftOverDiscount: d("0"),
			},
		},
		{
			name: "Percentage fee recalculates on zero product price",
			price: Price{
				ItemPrice:     d("100"),
				ItemFee:       AmountValue{Amount: d("5"), IsPercentage: true}, // 5% fee
				TaxPercentage: d("21"),
			},
			opts: PriceCalculation{
				ItemCount: 1,
				Discount:  AmountValue{Amount: d("150"), IsPercentage: false},
			},
			// Product is 0, so 5% of 0 is 0. Total fee should be 0.
			wantTotalInclTax: d("0"),
			wantCalcPrice: Totals{
				TotalPrice:       d("0"),
				TotalFee:         d("0"),
				TotalTax:         d("0"),
				TotalDiscount:    d("100"), // Maxed out at product
				LeftOverDiscount: d("50"),
			},
		},

		// ----------------------------------------------------------------
		// 6. ZERO VALUES & EDGE CASES
		// ----------------------------------------------------------------
		{
			name: "Zero price item",
			price: Price{
				ItemPrice:     d("0"),
				TaxPercentage: d("21"),
			},
			opts: PriceCalculation{
				ItemCount: 1,
			},
			wantTotalInclTax: d("0"),
			wantCalcPrice: Totals{
				TotalPrice:       d("0"),
				TotalFee:         d("0"),
				TotalTax:         d("0"),
				TotalDiscount:    d("0"),
				LeftOverDiscount: d("0"),
			},
		},
		{
			name: "Zero price item with fixed fee",
			price: Price{
				ItemPrice:     d("0"),
				ItemFee:       AmountValue{Amount: d("10"), IsPercentage: false},
				TaxPercentage: d("21"),
			},
			opts: PriceCalculation{
				ItemCount: 1,
			},
			wantTotalInclTax: d("12.10"),
			wantCalcPrice: Totals{
				TotalPrice:       d("0"),
				TotalFee:         d("10"),
				TotalTax:         d("2.10"),
				TotalDiscount:    d("0"),
				LeftOverDiscount: d("0"),
			},
		},
		{
			name: "Zero price item with percentage fee",
			price: Price{
				ItemPrice:     d("0"),
				ItemFee:       AmountValue{Amount: d("10"), IsPercentage: true},
				TaxPercentage: d("21"),
			},
			opts: PriceCalculation{
				ItemCount: 1,
			},
			wantTotalInclTax: d("0"),
			wantCalcPrice: Totals{
				TotalPrice:       d("0"),
				TotalFee:         d("0"),
				TotalTax:         d("0"),
				TotalDiscount:    d("0"),
				LeftOverDiscount: d("0"),
			},
		},
		{
			name: "Zero items purchased",
			price: Price{
				ItemPrice:     d("100"),
				ItemFee:       AmountValue{Amount: d("10"), IsPercentage: false},
				TaxPercentage: d("21"),
			},
			opts: PriceCalculation{
				ItemCount: 0,
			},
			wantTotalInclTax: d("0"),
			wantCalcPrice: Totals{
				TotalPrice:       d("0"),
				TotalFee:         d("0"),
				TotalTax:         d("0"),
				TotalDiscount:    d("0"),
				LeftOverDiscount: d("0"),
			},
		},
		{
			name: "Negative Tax Percentage (Refund scenario/edge case)",
			price: Price{
				ItemPrice:     d("275"),
				TaxPercentage: d("-21"),
			},
			opts: PriceCalculation{
				ItemCount: 1,
			},
			wantTotalInclTax: d("217.25"),
			wantCalcPrice: Totals{
				TotalPrice:       d("275"),
				TotalFee:         d("0"),
				TotalTax:         d("-57.75"),
				TotalDiscount:    d("0"),
				LeftOverDiscount: d("0"),
			},
		},
		{
			name: "Zero Tax Percentage",
			price: Price{
				ItemPrice:     d("100"),
				TaxPercentage: d("0"),
			},
			opts: PriceCalculation{
				ItemCount: 1,
			},
			wantTotalInclTax: d("100"),
			wantCalcPrice: Totals{
				TotalPrice:       d("100"),
				TotalFee:         d("0"),
				TotalTax:         d("0"),
				TotalDiscount:    d("0"),
				LeftOverDiscount: d("0"),
			},
		},

		// ----------------------------------------------------------------
		// 7. EXTREME PRECISION MATH
		// ----------------------------------------------------------------
		{
			name: "High precision decimal values",
			price: Price{
				ItemPrice:     d("9.99"),
				ItemFee:       AmountValue{Amount: d("1.25"), IsPercentage: false},
				TaxPercentage: d("9.5"), // 9.5% tax
			},
			opts: PriceCalculation{
				ItemCount: 3,
				Discount:  AmountValue{Amount: d("3.33"), IsPercentage: false},
			},
			// Price: 3 * 9.99 = 29.97
			// Price after discount: 29.97 - 3.33 = 26.64
			// Fee: 3 * 1.25 = 3.75
			// Subtotal: 26.64 + 3.75 = 30.39
			// Tax: 30.39 * 0.095 = 2.88705
			// Total: 30.39 + 2.88705 = 33.27705
			wantTotalInclTax: d("33.27705"),
			wantCalcPrice: Totals{
				TotalPrice:       d("26.64"),
				TotalFee:         d("3.75"),
				TotalTax:         d("2.88705"),
				TotalDiscount:    d("3.33"),
				LeftOverDiscount: d("0"),
			},
		},
		{
			name: "Percentage precision test",
			price: Price{
				ItemPrice:     d("100"),
				TaxPercentage: d("21"),
			},
			opts: PriceCalculation{
				ItemCount: 1,
				Discount:  AmountValue{Amount: d("33.3333"), IsPercentage: true},
			},
			// Discount: 100 * 0.333333 = 33.3333
			// Price: 66.6667
			// Tax: 66.6667 * 0.21 = 14.000007
			// Total: 80.666707
			wantTotalInclTax: d("80.666707"),
			wantCalcPrice: Totals{
				TotalPrice:       d("66.6667"),
				TotalFee:         d("0"),
				TotalTax:         d("14.000007"),
				TotalDiscount:    d("33.3333"),
				LeftOverDiscount: d("0"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotTotalInclTax, gotCalcPrice := tt.price.Calculate(tt.opts)

			// Using decimal.Equal instead of assert.Equal for decimals
			// to avoid failing on different underlying representations of mathematically equal values.
			assert.Truef(t, tt.wantTotalInclTax.Equal(gotTotalInclTax),
				"TotalInclTax: want %s, got %s", tt.wantTotalInclTax, gotTotalInclTax)

			assert.Truef(t, tt.wantCalcPrice.TotalPrice.Equal(gotCalcPrice.TotalPrice),
				"TotalPrice: want %s, got %s", tt.wantCalcPrice.TotalPrice, gotCalcPrice.TotalPrice)

			assert.Truef(t, tt.wantCalcPrice.TotalFee.Equal(gotCalcPrice.TotalFee),
				"TotalFee: want %s, got %s", tt.wantCalcPrice.TotalFee, gotCalcPrice.TotalFee)

			assert.Truef(t, tt.wantCalcPrice.TotalTax.Equal(gotCalcPrice.TotalTax),
				"TotalTax: want %s, got %s", tt.wantCalcPrice.TotalTax, gotCalcPrice.TotalTax)

			assert.Truef(t, tt.wantCalcPrice.TotalDiscount.Equal(gotCalcPrice.TotalDiscount),
				"TotalDiscount: want %s, got %s", tt.wantCalcPrice.TotalDiscount, gotCalcPrice.TotalDiscount)

			assert.Truef(t, tt.wantCalcPrice.LeftOverDiscount.Equal(gotCalcPrice.LeftOverDiscount),
				"LeftOverDiscount: want %s, got %s", tt.wantCalcPrice.LeftOverDiscount, gotCalcPrice.LeftOverDiscount)
		})
	}
}
