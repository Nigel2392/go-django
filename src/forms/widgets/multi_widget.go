package widgets

import (
	"context"
	"fmt"
	"io"
	"net/url"

	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/src/core/assert"
	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/filesystem"
	"github.com/Nigel2392/go-django/src/forms/media"
	"github.com/elliotchance/orderedmap/v2"
)

var _ Widget = (*MultiWidget)(nil)

type MultiWidget struct {
	BaseWidget
	Widgets *orderedmap.OrderedMap[string, Widget]
}

func NewMultiWidget(attrs map[string]string) *MultiWidget {
	return &MultiWidget{}
}

func (b *MultiWidget) AddWidget(name string, widget Widget) {
	if b.Widgets == nil {
		b.Widgets = orderedmap.NewOrderedMap[string, Widget]()
	}
	b.Widgets.Set(name, widget)
}

func (b *MultiWidget) Media() media.Media {
	var m media.Media = media.NewMedia()
	for head := b.Widgets.Front(); head != nil; head = head.Next() {
		m = m.Merge(head.Value.Media())
	}
	return m
}

func (b *MultiWidget) Validate(ctx context.Context, value interface{}) []error {
	var valMap, ok = value.(map[string]interface{})
	if !ok {
		return []error{errors.ValueError.Wrapf(
			"unexpected value type %T received when validating, multi-widget only supports map[string]interface{} and nil",
			value,
		)}
	}

	var errs = make([]error, 0)
	for head := b.Widgets.Front(); head != nil; head = head.Next() {
		var widgetValue = valMap[head.Key]
		var widgetErrs = head.Value.Validate(ctx, widgetValue)
		errs = append(errs, widgetErrs...)
	}
	return errs
}

func (b *MultiWidget) ValueFromDataDict(ctx context.Context, data url.Values, files map[string][]filesystem.FileHeader, name string) (interface{}, []error) {
	var values = make(map[string]interface{})
	for head := b.Widgets.Front(); head != nil; head = head.Next() {
		var widgetName = fmt.Sprintf("%s__%s", name, head.Key)
		var value, errs = head.Value.ValueFromDataDict(ctx, data, files, widgetName)
		if len(errs) > 0 {
			return values, errs
		}

		values[head.Key] = value
	}
	return values, nil
}

func (b *MultiWidget) ValueOmittedFromData(ctx context.Context, data url.Values, files map[string][]filesystem.FileHeader, name string) bool {
	for head := b.Widgets.Front(); head != nil; head = head.Next() {
		var widgetName = fmt.Sprintf("%s__%s", name, head.Key)
		if !head.Value.ValueOmittedFromData(ctx, data, files, widgetName) {
			return false
		}
	}
	return true
}

func (b *MultiWidget) ValueToForm(value interface{}) interface{} {
	var formData = make(map[string]interface{})
	switch v := value.(type) {
	case map[string]interface{}:
		for head := b.Widgets.Front(); head != nil; head = head.Next() {
			formData[head.Key] = head.Value.ValueToForm(v[head.Key])
		}
	case nil:
		for head := b.Widgets.Front(); head != nil; head = head.Next() {
			formData[head.Key] = head.Value.ValueToForm(nil)
		}
	default:
		assert.Fail(
			"unexpected value type %T, multi-widget only supports map[string]interface{} and nil",
			value,
		)
	}
	return formData
}

func (b *MultiWidget) ValueToGo(value interface{}) (interface{}, error) {
	var result = make(map[string]interface{})
	switch v := value.(type) {
	case map[string]interface{}:
		for head := b.Widgets.Front(); head != nil; head = head.Next() {
			var val, err = head.Value.ValueToGo(v[head.Key])
			if err != nil {
				return nil, err
			}
			result[head.Key] = val
		}
	case nil:
		for head := b.Widgets.Front(); head != nil; head = head.Next() {
			var val, err = head.Value.ValueToGo(nil)
			if err != nil {
				return nil, err
			}
			result[head.Key] = val
		}
	default:
		assert.Fail(
			"unexpected value type %T, multi-widget only supports map[string]interface{} and nil",
			value,
		)
	}
	return result, nil
}

func (b *MultiWidget) GetContextData(widgetCtx context.Context, id string, name string, value interface{}, attrs map[string]string) ctx.Context {
	var context = ctx.NewContext(nil)
	var widgetContext = make(map[string]ctx.Context)
	var valMap, ok = value.(map[string]interface{})
	if !ok && value != nil {
		assert.Fail(
			"unexpected value type %T, multi-widget only supports map[string]interface{} and nil",
			value,
		)
	}

	for head := b.Widgets.Front(); head != nil; head = head.Next() {
		var (
			name = fmt.Sprintf("%s__%s", name, head.Key)
			id   = fmt.Sprintf("%s__%s", id, head.Key)
		)

		widgetContext[head.Key] = head.Value.GetContextData(widgetCtx, id, name, valMap[head.Key], attrs)
	}

	context.Set("widgets", widgetContext)
	return context
}

func (b *MultiWidget) Render(ctx context.Context, w io.Writer, id, name string, value interface{}, attrs map[string]string) error {
	return b.RenderWithErrors(ctx, w, id, name, value, nil, attrs, b.GetContextData(ctx, id, name, value, attrs))
}

func (b *MultiWidget) RenderWithErrors(c context.Context, w io.Writer, id, name string, value interface{}, errors []error, attrs map[string]string, context ctx.Context) error {
	var valMap, ok = value.(map[string]interface{})
	if !ok && value != nil {
		return fmt.Errorf(
			"unexpected value type %T (%v), multi-widget only supports map[string]interface{} and nil",
			value, value,
		)
	}

	widgetContext, ok := context.Get("widgets").(map[string]ctx.Context)
	if !ok {
		return fmt.Errorf("unexpected context type, multi-widget requires widget context map in top-level context")
	}

	w.Write([]byte(`<div class="multi-widget">`))
	for head := b.Widgets.Front(); head != nil; head = head.Next() {
		var widgetContext, ok = widgetContext[head.Key]
		if !ok {
			return fmt.Errorf("missing widget context for %q", head.Key)
		}

		var (
			name = fmt.Sprintf("%s__%s", name, head.Key)
			id   = fmt.Sprintf("%s__%s", id, head.Key)
		)

		w.Write([]byte(`<div class="multi-widget-field">`))
		w.Write([]byte(fmt.Sprintf("<label for=\"%s\">%s</label>", id, head.Value.Field().Label(c))))
		if err := head.Value.RenderWithErrors(c, w, id, name, valMap[head.Key], errors, attrs, widgetContext); err != nil {
			return err
		}
		w.Write([]byte(`</div>`))
	}
	w.Write([]byte(`</div>`))

	return nil
}
