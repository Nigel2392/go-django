package blocks

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"

	"github.com/Nigel2392/django/core/assert"
	"github.com/Nigel2392/django/core/ctx"
	"github.com/Nigel2392/django/forms/fields"
	"github.com/Nigel2392/go-telepath/telepath"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

var _ Block = (*ListBlock)(nil)

type ListBlockValue struct {
	ID   uuid.UUID   `json:"id"`
	Data interface{} `json:"data"`
}

type ListBlock struct {
	*BaseBlock
	Child Block
	Min   int
	Max   int
}

func NewListBlock(block Block, minMax ...int) *ListBlock {

	block.SetName("item")

	var l = &ListBlock{
		BaseBlock: NewBaseBlock(),
		Child:     block,
		Min:       -1,
		Max:       -1,
	}

	assert.Lt(minMax, 3, "Too many arguments (min, max)")

	if len(minMax) == 2 {
		l.Min = minMax[0]
		l.Max = minMax[1]
	}
	if len(minMax) == 1 {
		l.Min = minMax[0]
	}
	l.FormField = fields.CharField()
	return l
}

func (l *ListBlock) MinNum() int {
	return l.Min
}

func (l *ListBlock) MaxNum() int {
	return l.Max
}

func (l *ListBlock) makeError(err error) error {
	return err
}

func (l *ListBlock) makeIndexedError(index int, err ...error) error {
	if len(err) == 0 || len(err) >= 1 && err[0] == nil {
		return nil
	}
	var e = NewBlockErrors[int]()
	e.AddError(index, err...)
	return e
}

func (b *ListBlock) ValueOmittedFromData(data url.Values, files map[string][]io.ReadCloser, name string) bool {
	var addedKey = fmt.Sprintf("%s-added", name)
	if !data.Has(addedKey) {
		return true
	}

	var omitted = true
	for i := 0; ; i++ {
		var key = fmt.Sprintf("%s-%d", name, i)
		if data.Has(key) {
			omitted = false
			break
		}
	}
	return omitted
}

func (l *ListBlock) ValueFromDataDict(d url.Values, files map[string][]io.ReadCloser, name string) (interface{}, []error) {
	var data = make([]*ListBlockValue, 0)

	var (
		added    = 0
		addedKey = fmt.Sprintf("%s-added", name)
		addedCnt = 0
	)

	if !d.Has(addedKey) {
		return nil, []error{fmt.Errorf("Malformed form data, missing key %s", addedKey)} //lint:ignore ST1005 ignore this lint
	}

	var addedValue = strings.TrimSpace(d.Get(addedKey))
	var err error
	added, err = strconv.Atoi(addedValue)
	if err != nil {
		return nil, []error{l.makeError(err)}
	}

	var errs = NewBlockErrors[int]()

	for i := 0; ; i++ {
		var key = fmt.Sprintf("%s-%d", name, i)
		if l.Child.ValueOmittedFromData(d, files, key) {
			break
		}

		var (
			idKey = fmt.Sprintf("%s-id-%d", name, i)
			idStr = d.Get(idKey)
			id    uuid.UUID
		)

		if idStr != "" {
			id, err = uuid.Parse(idStr)
			if err != nil {
				errs.AddError(i, fmt.Errorf("Invalid UUID: %s", idStr)) //lint:ignore ST1005 ignore this lint
				continue
			}
		} else {
			id = uuid.New()
		}

		var value, e = l.Child.ValueFromDataDict(d, files, key)
		if len(e) != 0 {
			errs.AddError(i, e...)
			continue
		}

		data = append(data, &ListBlockValue{
			ID:   id,
			Data: value,
		})

		addedCnt++
	}

	if errs.HasErrors() {
		return nil, []error{errs}
	}

	if l.Min != -1 && len(data) < l.Min {
		return nil, []error{l.makeError(
			fmt.Errorf("Must have at least %d items (has %d)", l.Min, len(data)), //lint:ignore ST1005 ignore this lint
		)}
	}

	if l.Max != -1 && len(data) > l.Max {
		return nil, []error{l.makeError(
			fmt.Errorf("Must have at most %d items (has %d)", l.Max, len(data)), //lint:ignore ST1005 ignore this lint
		)}
	}

	if addedCnt != added {
		return nil, []error{l.makeError(
			fmt.Errorf("Invalid number of items, expected %d, got %d", added, addedCnt), //lint:ignore ST1005 ignore this lint
		)}
	}

	return data, nil
}

func (l *ListBlock) ValueToGo(value interface{}) (interface{}, error) {
	if fields.IsZero(value) {
		return "", nil
	}
	var (
		valueArr []*ListBlockValue
		ok       bool
	)

	if valueArr, ok = value.([]*ListBlockValue); !ok {
		return nil, fmt.Errorf("value must be of type []*ListBlockValue], got %T", value)
	}

	var (
		newArr = make([]*ListBlockValue, len(valueArr))
		errs   = NewBlockErrors[int]()
	)
	for i, lbVal := range valueArr {
		var childData, err = l.Child.ValueToGo(lbVal.Data)
		if err != nil {
			errs.AddError(i, err)
			continue
		}

		newArr[i] = &ListBlockValue{
			ID:   lbVal.ID,
			Data: childData,
		}
	}

	if errs.HasErrors() {
		return nil, errs
	}

	return newArr, nil
}

func (l *ListBlock) GetDefault() interface{} {
	if l.Min > 0 {
		var data = make([]*ListBlockValue, l.Min)
		for i := 0; i < l.Min; i++ {
			data[i] = &ListBlockValue{
				ID:   uuid.New(),
				Data: l.Child.GetDefault(),
			}
		}
		return data
	}
	return make([]*ListBlockValue, 0)
}

func (l *ListBlock) ValueToForm(value interface{}) interface{} {

	if fields.IsZero(value) {
		value = l.GetDefault()
	}

	var valueArr []*ListBlockValue
	var ok bool
	if valueArr, ok = value.([]*ListBlockValue); !ok {
		return ""
	}

	var data = make([]*ListBlockValue, 0, len(valueArr))
	for _, v := range valueArr {
		data = append(data, &ListBlockValue{
			ID:   v.ID,
			Data: l.Child.ValueToForm(v.Data),
		})
	}

	return data
}

func (l *ListBlock) Clean(value interface{}) (interface{}, error) {
	if fields.IsZero(value) {
		return nil, nil
	}

	var data = make([]*ListBlockValue, 0)
	for i, lbVal := range value.([]*ListBlockValue) {
		var v, err = l.Child.Clean(lbVal.Data)
		if err != nil {
			return nil, l.makeIndexedError(i, errors.Wrapf(err, "index %d", i))
		}

		data = append(data, &ListBlockValue{
			ID:   lbVal.ID,
			Data: v,
		})
	}

	return data, nil
}

func (l *ListBlock) Validate(value interface{}) []error {

	for _, validator := range l.Validators {
		if err := validator(value); err != nil {
			return []error{err}
		}
	}

	if fields.IsZero(value) {
		return nil
	}

	var errors = make([]error, 0)
	for i, v := range value.([]*ListBlockValue) {
		var e = l.Child.Validate(v.Data)
		if len(e) != 0 {
			errors = append(errors, l.makeIndexedError(i, e...))
		}
	}
	return errors
}

func (l *ListBlock) RenderForm(w io.Writer, id, name string, value interface{}, errors []error, tplCtx ctx.Context) error {
	var (
		ctxData  = NewBlockContext(l, tplCtx)
		valueArr []*ListBlockValue
		ok       bool
	)
	ctxData.ID = id
	ctxData.Name = name
	ctxData.Value = value

	if value == nil || value == "" {
		value = l.GetDefault()
	}

	if valueArr, ok = value.([]*ListBlockValue); !ok {
		return fmt.Errorf("value must be a []interface{}")
	}

	var listBlockErrors = NewBlockErrors[int](errors...)

	var blockArgs = map[string]interface{}{
		"id":         id,
		"name":       name,
		"block":      l,
		"childBlock": l.Child,
	}
	var bt, err = telepath.PackJSON(JSContext, blockArgs)
	if err != nil {
		return err
	}

	return l.RenderTempl(w, id, name, valueArr, string(bt), listBlockErrors, ctxData).Render(context.Background(), w)
}

func (m *ListBlock) Adapter() telepath.Adapter {
	return &telepath.ObjectAdapter[*ListBlock]{
		JSConstructor: "django.blocks.ListBlock",
		GetJSArgs: func(obj *ListBlock) []interface{} {
			return []interface{}{map[string]interface{}{
				"name":     obj.Name(),
				"label":    obj.Label(),
				"helpText": obj.HelpText(),
				"required": obj.Field().Required(),
			}}
		},
	}
}
