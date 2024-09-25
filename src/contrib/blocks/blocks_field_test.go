package blocks_test

import (
	"errors"
	"fmt"
	"net/mail"
	"reflect"
	"testing"
	"time"

	"github.com/Nigel2392/go-django/src/contrib/blocks"
	"github.com/Nigel2392/go-django/src/core/errs"
)

func NewBlock[T1, T2 blocks.Block](name string, init func(...func(T1)) T2, opts ...func(T1)) T2 {
	var b = init(
		append(
			opts,
			func(t T1) {
				t.SetName(name)
			},
		)...,
	)
	return b
}

type FieldBlockTestFunc struct {
	Name    string
	Execute func(b blocks.Block, t *testing.T)
}

func (f *FieldBlockTestFunc) TestName() string {
	return f.Name
}

type FieldBlockTest struct {
	Name  string
	Block blocks.Block
	Tests []FieldBlockTestFunc
}

//	var requiredValidator = func(v interface{}) error {
//		if fields.IsZero(v) {
//			return errs.ErrFieldRequired
//		}
//		return nil
//	}

func TestFieldBlock(t *testing.T) {
	var testValueToGo = func(b blocks.Block, value interface{}, expected interface{}, t *testing.T) {
		var (
			got, err = b.ValueToGo(value)
		)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		var rtyp1 = reflect.TypeOf(got)
		var rtyp2 = reflect.TypeOf(expected)
		var rVal1 = reflect.ValueOf(got)
		var rVal2 = reflect.ValueOf(expected)

		if rtyp1.Kind() == reflect.Ptr {
			rtyp1 = rtyp1.Elem()
			rVal1 = rVal1.Elem()
		}
		if rtyp2.Kind() == reflect.Ptr {
			rtyp2 = rtyp2.Elem()
			rVal2 = rVal2.Elem()
		}

		if rtyp1 != rtyp2 {
			t.Errorf("Expected %v, got %v", rtyp2, rtyp1)
		}

		if rVal1.Interface() != rVal2.Interface() {
			t.Errorf("Expected %v, got %v", rVal2.Interface(), rVal1.Interface())
		}
	}

	var testValueToGoError = func(b blocks.Block, value interface{}, expectedError error, t *testing.T) {
		var _, err = b.ValueToGo(value)
		if err == nil {
			t.Errorf("Expected error, got nil")
		}

		if expectedError != nil && !errors.Is(err, expectedError) {
			t.Errorf("Expected %v, got %v", err, err)
		}
	}

	var testValueToForm = func(b blocks.Block, value interface{}, expected interface{}, t *testing.T) {
		var got = b.ValueToForm(value)
		if got != expected {
			t.Errorf("Expected %v, got %v", expected, got)
		}
	}

	var tests = []FieldBlockTest{
		{
			Name: "CharBlock",
			Block: NewBlock("test", blocks.CharBlock, func(b *blocks.FieldBlock) {
				b.Default = func() interface{} { return "Custom Default" }
			}),
			Tests: []FieldBlockTestFunc{
				{
					Name: "GetDefault",
					Execute: func(b blocks.Block, t *testing.T) {
						var got = b.GetDefault()
						if got != "Custom Default" {
							t.Errorf("Expected %v, got %v", "Custom Default", got)
						}
					},
				},
				{
					Name: "ValueToGo",
					Execute: func(b blocks.Block, t *testing.T) {
						testValueToGo(b, "test", "test", t)
					},
				},
				{
					Name: "ValueToForm",
					Execute: func(b blocks.Block, t *testing.T) {
						testValueToForm(b, "test", "test", t)
					},
				},
			},
		},
		{
			Name:  "NumberBlock",
			Block: NewBlock("test", blocks.NumberBlock),
			Tests: []FieldBlockTestFunc{
				{
					Name: "ValueToGo",
					Execute: func(b blocks.Block, t *testing.T) {
						testValueToGo(b, "1", 1, t)
					},
				},
				{
					Name: "ValueToForm",
					Execute: func(b blocks.Block, t *testing.T) {
						testValueToForm(b, 1, "1", t)
					},
				},
				{
					Name: "ValueToGoError",
					Execute: func(b blocks.Block, t *testing.T) {
						testValueToGoError(b, "1abc", errs.ErrInvalidSyntax, t)
					},
				},
			},
		},
		//{
		//	Block:        NewBlock("test", blocks.TextBlock),
		//	FormValue:    nil,
		//	FormExpected: "",
		//},
		{
			Name:  "EmailBlock",
			Block: NewBlock("test", blocks.EmailBlock),
			Tests: []FieldBlockTestFunc{
				{
					Name: "ValueToGo",
					Execute: func(b blocks.Block, t *testing.T) {
						var emailAddr = "test@example.com"
						var netAddr, err = mail.ParseAddress(emailAddr)
						if err != nil {
							t.Errorf("Expected no error, got %v", err)
						}
						testValueToGo(b, emailAddr, netAddr, t)
					},
				},
				{
					Name: "ValueToGoError",
					Execute: func(b blocks.Block, t *testing.T) {
						testValueToGoError(b, "test", nil, t)
					},
				},
				{
					Name: "ValueToForm",
					Execute: func(b blocks.Block, t *testing.T) {
						var emailAddr = "test@example.com"
						var netAddr, err = mail.ParseAddress(emailAddr)
						if err != nil {
							t.Errorf("Expected no error, got %v", err)
						}

						testValueToForm(b, netAddr, emailAddr, t)
					},
				},
			},
		},
		{
			Name:  "DateBlock",
			Block: NewBlock("test", blocks.DateBlock),
			Tests: []FieldBlockTestFunc{
				{
					Name: "ValueToGo",
					Execute: func(b blocks.Block, t *testing.T) {
						var date = "2021-01-01"
						var tDate, _ = time.Parse("2006-01-02", date)
						testValueToGo(b, "2021-01-01", tDate, t)
					},
				},
				{
					Name: "ValueToGoError",
					Execute: func(b blocks.Block, t *testing.T) {
						testValueToGoError(b, "2021-01-01abc", errs.ErrInvalidSyntax, t)
					},
				},
				{
					Name: "ValueToForm",
					Execute: func(b blocks.Block, t *testing.T) {
						var date = "2021-01-01"
						var tDate, _ = time.Parse("2006-01-02", date)
						testValueToForm(b, tDate, "2021-01-01", t)
					},
				},
			},
		},
		{
			Name:  "DateTimeBlock",
			Block: NewBlock("test", blocks.DateTimeBlock),
			Tests: []FieldBlockTestFunc{
				{
					Name: "ValueToGo",
					Execute: func(b blocks.Block, t *testing.T) {
						var date = "2021-01-01T10:00:23"
						var tDate, err = time.Parse("2006-01-02T15:04:05", date)
						if err != nil {
							t.Errorf("Expected no error, got %v", err)
						}
						testValueToGo(b, date, tDate, t)
					},
				},
				{
					Name: "ValueToGoError",
					Execute: func(b blocks.Block, t *testing.T) {
						testValueToGoError(b, "2021-01-01 10abc", errs.ErrInvalidSyntax, t)
					},
				},
				{
					Name: "ValueToForm",
					Execute: func(b blocks.Block, t *testing.T) {
						var date = "2021-01-01T10:00:23"
						var tDate, err = time.Parse("2006-01-02T15:04:05", date)
						if err != nil {
							t.Errorf("Expected no error, got %v", err)
						}
						testValueToForm(b, tDate, date, t)
					},
				},
			},
		},
	}

	for _, test := range tests {
		var testName = fmt.Sprintf(
			"Test%s",
			test.Name,
		)
		t.Run(testName, func(t *testing.T) {

			for _, tst := range test.Tests {
				t.Run(tst.TestName(), func(t *testing.T) {
					tst.Execute(test.Block, t)
				})
			}
		})
	}
}
