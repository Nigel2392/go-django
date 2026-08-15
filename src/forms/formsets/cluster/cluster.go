package cluster

import (
	"context"
	"fmt"
	"iter"
	"net/http"
	"net/url"

	"github.com/Nigel2392/go-django/internal/django_reflect"
	"github.com/Nigel2392/go-django/internal/forms"
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/src/core/filesystem"
	"github.com/elliotchance/orderedmap/v2"
)

var (
	_ ClusterForm[Clusterable] = (*clusterForm[Clusterable, Clusterable, int])(nil)
)

type SAVE_ORDERING int

const (
	SAVE_ORDERING_PRE  SAVE_ORDERING = -1
	SAVE_ORDERING_NONE SAVE_ORDERING = 0
	SAVE_ORDERING_POST SAVE_ORDERING = 1
)

type SaveOrderable interface {
	SaveOrder() SAVE_ORDERING
}

type Clusterable interface {
	forms.Minimum
	SaveOrderable
	Bind(belongsTo Clusterable) forms.Minimum
}

type ClusterForm[FORM Clusterable] interface {
	Clusterable
	Forms() []FORM
	Save() (map[string]any, error)
}

type form = forms.Minimum

type clusterForm[THIS forms.Minimum, FORM Clusterable, K comparable] struct {
	data   forms.FormData
	parent Clusterable
	order  SAVE_ORDERING
	form
	forms     *orderedmap.OrderedMap[K, FORM]
	formsFunc func(THIS) iter.Seq2[K, FORM]
}

func NewClusterForm[THIS forms.Minimum, FORM Clusterable, K comparable](form THIS, opts ...func(*clusterForm[THIS, FORM, K])) ClusterForm[FORM] {
	var c = &clusterForm[THIS, FORM, K]{
		form: form,
	}

	if s, ok := c.form.(SaveOrderable); ok {
		c.order = s.SaveOrder()
	}

	//fn, err := django_reflect.Method[func() []FORM](c.form, "Forms", django_reflect.WithContext(
	//	c.Context(),
	//))
	//if err != nil && !errors.Is(err, django_reflect.ErrMethodNotFound) {
	//	panic(err)
	//}
	//
	//if err == nil {
	//	c.forms
	//}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

func (b *clusterForm[T, O, K]) Bind(belongsTo Clusterable) forms.Minimum {
	b.parent = belongsTo
	return b
}

func (b *clusterForm[T, O, K]) SaveOrder() SAVE_ORDERING {
	return b.order
}

func (b *clusterForm[T, O, K]) Forms() []O {
	var forms = b.getForms()
	var l = make([]O, 0, forms.Len())
	for head := forms.Front(); head != nil; head = head.Next() {
		l = append(l, head.Value)
	}
	return l
}

func (b *clusterForm[T, O, K]) getForms() *orderedmap.OrderedMap[K, O] {
	if b.forms != nil {
		return b.forms
	}
	b.forms = orderedmap.NewOrderedMap[K, O]()
	for k, v := range b.formsFunc(b.form.(T)) {
		v = v.Bind(b).(O)
		v.SetPrefix(fmt.Sprint(k))
		v.WithContext(b.form.Context())

		b.forms.Set(k, v)
	}
	return b.forms
}

func (b *clusterForm[T, O, K]) Prefix() string {
	return b.form.Prefix()
}

func (b *clusterForm[T, O, K]) SetPrefix(prefix string) {
	if b == nil {
		panic("BaseFormSet.SetPrefix: BaseFormSet is nil")
	}

	b.form.SetPrefix(prefix)

	for k, v := range b.getForms().Iterator() {
		v.SetPrefix(b.form.PrefixName(fmt.Sprint(k)))
	}
}

func (fs *clusterForm[T, O, K]) Context() context.Context {
	return fs.form.Context()
}

func (fs *clusterForm[T, O, K]) WithContext(ctx context.Context) {
	fs.form.WithContext(ctx)

	for _, form := range fs.getForms().Iterator() {
		form.WithContext(ctx)
	}
}

func (f *clusterForm[T, O, K]) WithData(data url.Values, files map[string][]filesystem.FileHeader, r *http.Request) {
	f.data.Request = r
	f.data.Values = data
	f.data.Files = files
}

func (f *clusterForm[T, O, K]) Data() (url.Values, map[string][]filesystem.FileHeader) {
	return f.data.Values, f.data.Files
}

func (a *clusterForm[T, O, K]) SaveFunc(fn func(context.Context, T) (map[string]interface{}, error)) (c map[string]interface{}, err error) {
	if a.forms == nil || a.forms.Len() == 0 {
		return a.form.CleanedData(), a.saveForm(a.form, SAVE_ORDERING_NONE)
	}

	var (
		preSaveForms  = make([]O, 0)
		postSaveForms = make([]O, 0)
	)

	for _, form := range a.getForms().Iterator() {
		if form.SaveOrder() == SAVE_ORDERING_POST {
			postSaveForms = append(postSaveForms, form)
		} else {
			preSaveForms = append(preSaveForms, form)
		}
	}

	for _, form := range preSaveForms {
		if err = a.saveForm(form, SAVE_ORDERING_PRE); err != nil {
			return nil, err
		}
	}

	data, err := fn(a.form.Context(), a.form.(T))
	if err != nil {
		return data, err
	}

	for _, form := range postSaveForms {
		if err = a.saveForm(form, SAVE_ORDERING_POST); err != nil {
			return nil, err
		}
	}

	return data, nil
}

func (a *clusterForm[T, O, K]) Save() (map[string]interface{}, error) {
	return a.SaveFunc(func(ctx context.Context, t T) (map[string]interface{}, error) {
		cleaned := a.form.CleanedData()
		err := a.saveForm(t, SAVE_ORDERING_NONE)
		if err != nil {
			return cleaned, err
		}
		return a.form.CleanedData(), err
	})
}

func (a *clusterForm[T, O, K]) saveForm(form any, _ SAVE_ORDERING) error {
	saveFn, err := django_reflect.Method[func() error](form, "Save")
	if err != nil {
		if errors.Is(err, django_reflect.ErrMethodNotFound) {
			return nil
		}
		return err
	}

	err = saveFn()
	if err != nil {
		a.form.AddFormError(err)
	}
	return err
}
