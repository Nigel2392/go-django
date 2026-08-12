package generic

import (
	"context"
	"fmt"
	"io"
	"maps"
	"net/http"
	"slices"
	"strconv"

	"github.com/Nigel2392/go-django/contrib/admin/components"
	"github.com/Nigel2392/go-django/contrib/filters"
	"github.com/Nigel2392/go-django/internal/forms"
	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/forms/fields"
	"github.com/Nigel2392/go-django/src/forms/widgets"
	"github.com/Nigel2392/go-django/src/forms/widgets/options"
	"github.com/a-h/templ"
)

type filterSpecOpts struct {
	formFieldOpts []func(forms.Field)
	dflt          string
	useApply      bool
}

func OptFilterSpecFormField(opts ...func(forms.Field)) func(*filterSpecOpts) {
	return func(fso *filterSpecOpts) {
		fso.formFieldOpts = append(fso.formFieldOpts, opts...)
	}
}

func OptFilterSpecApplyAmount(b bool) func(*filterSpecOpts) {
	return func(fso *filterSpecOpts) {
		fso.useApply = false
	}
}

func OptFilterSpecDefaultAmount(i int) func(*filterSpecOpts) {
	if i == 0 {
		i = 10
	}
	return func(fso *filterSpecOpts) {
		fso.dflt = strconv.Itoa(i)
	}
}

type amountFilterSpec[T attrs.Definer] struct {
	amounts []int
	opts    filterSpecOpts
	name    string
}

func (b amountFilterSpec[T]) Name() string {
	return b.name
}

func (b amountFilterSpec[T]) getCurrent(r *http.Request) string {
	var current = r.URL.Query().Get(b.name)
	if current == "" {
		current = b.opts.dflt
	}
	return current
}

func (b amountFilterSpec[T]) getOptions(r *http.Request, amounts []int) []widgets.Option {
	if len(amounts) == 0 {
		amounts = AMOUNT_OPTIONS
	}

	var current = b.getCurrent(r)
	var opts = make([]widgets.Option, 0, len(amounts))
	for _, amount := range amounts {
		s := strconv.Itoa(amount)
		opt := widgets.NewOption(s, s, s)

		if s == current {
			opt = &widgets.WrappedOption{
				Option:   opt,
				Selected: true,
			}
		}

		opts = append(opts, opt)
	}
	return opts
}

func (b amountFilterSpec[T]) Field(r *http.Request) fields.Field {
	wOpts := b.getOptions(r, b.amounts)
	fldOpts := slices.Clone(b.opts.formFieldOpts)
	fldOpts = append(fldOpts, fields.Widget(options.NewSelectInput(
		nil, func() []widgets.Option { return wOpts },
	)))
	return fields.CharField(
		fldOpts...,
	)
}

func (b amountFilterSpec[T]) Filter(req *http.Request, value interface{}, object *queries.QuerySet[T]) (*queries.QuerySet[T], error) {
	if !b.opts.useApply {
		return object, nil
	}

	if fields.IsZero(value) {
		return object, nil
	}

	v, ok := value.(string)
	if !ok {
		return nil, errors.TypeMismatch.Wrapf(
			"%T is not of type string", value,
		)
	}

	if v == "" {
		return object, nil
	}

	l, err := strconv.Atoi(v)
	if err != nil {
		return nil, err
	}

	return object.Limit(l), nil
}

var AMOUNT_OPTIONS = []int{
	25,
	50,
	75,
	100,
}

func AmountHeaderAction(r *http.Request, currentAmount int, reverseURL string, amounts []int) components.ShowableComponent {
	if len(amounts) == 0 {
		amounts = AMOUNT_OPTIONS
	}
	return components.NewShowableComponent(r, nil, templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		var q = maps.Clone(r.URL.Query())
		q.Del("page")
		q.Del("amount")

		w.Write([]byte(`<form method="GET" action="`))
		w.Write([]byte(django.Reverse(reverseURL)))
		w.Write([]byte("?"))
		w.Write([]byte(q.Encode()))
		w.Write([]byte(`" class="amount-form form-field">`))

		for k, v := range q {
			if len(v) > 0 {
				w.Write([]byte(`<input type="hidden" name="`))
				w.Write([]byte(k))
				w.Write([]byte(`" value="`))
				w.Write([]byte(v[0]))
				w.Write([]byte(`">`))
			}
		}

		w.Write([]byte(`<select name="amount" onchange="this.form.submit()">`))
		for _, v := range amounts {
			var selectedText string
			if v == currentAmount {
				selectedText = ` selected`
			}

			fmt.Fprintf(w, `<option value="%d"%s>%d</option>`, v, selectedText, v)
		}
		w.Write([]byte(`</select>`))
		w.Write([]byte(`</form>`))
		return nil
	}))
}

func FilterSpecAmount[T attrs.Definer](name string, amounts []int, opts ...func(*filterSpecOpts)) filters.FilterSpec[T] {

	if name == "" {
		name = "amount"
	}

	var cnf = new(filterSpecOpts)
	for _, opt := range opts {
		opt(cnf)
	}

	if cnf.dflt == "" {
		cnf.dflt = "10"
	}

	return &amountFilterSpec[T]{
		name:    name,
		amounts: amounts,
		opts:    *cnf,
	}
}
