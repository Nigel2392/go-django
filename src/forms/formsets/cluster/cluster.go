package cluster

import (
	"context"
	"fmt"
	"iter"
	"net/http"
	"net/url"

	"github.com/Nigel2392/go-django/internal/forms"
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
	Parent() Clusterable
	Bind(belongsTo Clusterable) forms.Minimum
}

type ClusterForm[FORM Clusterable] interface {
	Clusterable
	Forms() []FORM
	Save() (map[string]any, error)
}

type _THIS = forms.Minimum

type clusterForm[THIS forms.Minimum, FORM Clusterable, K comparable] struct {
	_THIS
	data      forms.FormData
	parent    Clusterable
	order     SAVE_ORDERING
	forms     *orderedmap.OrderedMap[K, FORM]
	formsFunc func(THIS) iter.Seq2[K, FORM]
}

func New[THIS forms.Minimum, FORM Clusterable, K comparable](form THIS, opts ...func(*clusterForm[THIS, FORM, K])) ClusterForm[FORM] {
	var c = &clusterForm[THIS, FORM, K]{
		_THIS: form,
	}

	if s, ok := c._THIS.(SaveOrderable); ok {
		c.order = s.SaveOrder()
	}

	//	fn, err := django_reflect.Method[func() []FORM](c._THIS, "Forms", django_reflect.WithFuncArgs(
	//		c.Context(), django_reflect.Arg[ClusterForm[FORM]](c),
	//	))
	//	if err != nil && !errors.Is(err, django_reflect.ErrMethodNotFound) {
	//		panic(err)
	//	}
	//
	//	if err == nil {
	//
	//	}

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

func (b *clusterForm[T, O, K]) Parent() Clusterable {
	return b.parent
}

func (b *clusterForm[T, O, K]) Forms() []O {
	var forms = b.getForms()
	var l = make([]O, 0, forms.Len())
	for head := forms.Front(); head != nil; head = head.Next() {
		l = append(l, head.Value)
	}
	return l
}

func (b *clusterForm[THIS, FORM, K]) SetForm(key K, form FORM) {
	if b.forms == nil {
		b.forms = orderedmap.NewOrderedMap[K, FORM]()
	}

	form = form.Bind(b).(FORM)
	form.SetPrefix(b.PrefixName(fmt.Sprint(key)))
	form.WithContext(b._THIS.Context())
	b.forms.Set(key, form)
}

func (b *clusterForm[T, O, K]) getForms() *orderedmap.OrderedMap[K, O] {
	if b.forms != nil {
		return b.forms
	}

	b.forms = orderedmap.NewOrderedMap[K, O]()
	for k, v := range b.formsFunc(b._THIS.(T)) {
		b.SetForm(k, v)
	}

	return b.forms
}

func (b *clusterForm[THIS, FORM, K]) getOrderedForms() (pre, post []FORM) {
	formMap := b.getForms()
	post = make([]FORM, 0, formMap.Len())
	pre = make([]FORM, 0, formMap.Len())

	for _, form := range formMap.Iterator() {
		if form.SaveOrder() == SAVE_ORDERING_POST {
			post = append(post, form)
		} else {
			pre = append(pre, form)
		}
	}

	return pre, post
}

func (b *clusterForm[T, O, K]) Prefix() string {
	return b._THIS.Prefix()
}

func (b *clusterForm[T, O, K]) HasChanged() bool {
	if b._THIS.HasChanged() {
		return true
	}

	for _, f := range b.getForms().Iterator() {
		if f.HasChanged() {
			return true
		}
	}

	return false
}

func (b *clusterForm[T, O, K]) SetPrefix(prefix string) {
	if b == nil {
		panic("BaseFormSet.SetPrefix: BaseFormSet is nil")
	}

	b._THIS.SetPrefix(prefix)

	for k, v := range b.getForms().Iterator() {
		v.SetPrefix(b._THIS.PrefixName(fmt.Sprint(k)))
	}
}

func (fs *clusterForm[T, O, K]) Context() context.Context {
	return fs._THIS.Context()
}

func (fs *clusterForm[T, O, K]) WithContext(ctx context.Context) {
	fs._THIS.WithContext(ctx)

	for _, form := range fs.getForms().Iterator() {
		form.WithContext(ctx)
	}
}

func (fs *clusterForm[THIS, FORM, K]) Unwrap() []any {
	pre, post := fs.getOrderedForms()
	out := make([]any, 0, len(pre)+len(post)+1)
	for _, f := range pre {
		out = append(out, f)
	}

	out = append(out, fs)

	for _, f := range post {
		out = append(out, f)
	}

	return out
}

func (f *clusterForm[T, O, K]) WithData(data url.Values, files map[string][]filesystem.FileHeader, r *http.Request) {
	f.data.Request = r
	f.data.Values = data
	f.data.Files = files

	f._THIS.WithData(data, files, r)

	for _, f := range f.getForms().Iterator() {
		f.WithData(data, files, r)
	}
}

func (f *clusterForm[T, O, K]) Data() (url.Values, map[string][]filesystem.FileHeader) {
	return f.data.Values, f.data.Files
}

func (a *clusterForm[T, O, K]) SaveFunc(fn func(context.Context, T) (map[string]interface{}, error)) (c map[string]interface{}, err error) {
	if a.forms == nil || a.forms.Len() == 0 {
		return a._THIS.CleanedData(), forms.SaveForm(a.Context(), a._THIS, SAVE_ORDERING_NONE)
	}

	pre, post := a.getOrderedForms()
	for _, form := range pre {
		if err = forms.SaveForm(a.Context(), form, SAVE_ORDERING_PRE); err != nil {
			return nil, err
		}
	}

	data, err := fn(a.Context(), a._THIS.(T))
	if err != nil {
		return data, err
	}

	for _, form := range post {
		if err = forms.SaveForm(a.Context(), form, SAVE_ORDERING_POST); err != nil {
			return nil, err
		}
	}

	return data, nil
}

func (a *clusterForm[T, O, K]) Save() (map[string]interface{}, error) {
	return a.SaveFunc(func(ctx context.Context, t T) (map[string]interface{}, error) {
		cleaned := a._THIS.CleanedData()
		err := forms.SaveForm(ctx, t, SAVE_ORDERING_NONE)
		if err != nil {
			return cleaned, err
		}
		return a._THIS.CleanedData(), err
	})
}
