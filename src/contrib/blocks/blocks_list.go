package blocks

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"github.com/Nigel2392/go-django/src/core/assert"
	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/filesystem"
	"github.com/Nigel2392/go-django/src/forms/fields"
	"github.com/Nigel2392/go-telepath/telepath"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

var _ Block = (*ListBlock)(nil)

type ListBlockValue struct {
	ID    uuid.UUID   `json:"id"`
	Order int         `json:"order"`
	Data  interface{} `json:"data"`
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

func (b *ListBlock) ValueOmittedFromData(ctx context.Context, data url.Values, files map[string][]filesystem.FileHeader, name string) bool {
	var totalKey = fmt.Sprintf("%s--total", name)
	if !data.Has(totalKey) {
		return true
	}

	var totalValue, err = strconv.Atoi(strings.TrimSpace(data.Get(totalKey)))
	if err != nil || totalValue == 0 {
		return true
	}

	var omitted = true
	for i := 0; i < totalValue; i++ {
		var deletedKey = fmt.Sprintf("%s-%d--deleted", name, i)
		if data.Has(deletedKey) {
			var deletedValue = strings.TrimSpace(data.Get(deletedKey))
			if deletedValue == "on" || deletedValue == "true" || deletedValue == "1" {
				omitted = false // Deleted, so not omitted
				break
			}
		}

		var key = fmt.Sprintf("%s-%d", name, i)
		if !b.Child.ValueOmittedFromData(ctx, data, files, key) {
			omitted = false
			break
		}
	}
	return omitted
}

func sortListBlocks(a, b *ListBlockValue) int {
	if a.Order < b.Order {
		return -1
	}
	if a.Order > b.Order {
		return 1
	}
	return 0
}

func (l *ListBlock) ValueFromDataDict(ctx context.Context, d url.Values, files map[string][]filesystem.FileHeader, name string) (interface{}, []error) {
	var totalKey = fmt.Sprintf("%s--total", name)
	if !d.Has(totalKey) {
		return nil, []error{fmt.Errorf("Malformed form data, missing key %s", totalKey)} //lint:ignore ST1005 ignore this lint
	}

	var totalCount = 0
	var deletedCount = 0
	var totalValue = strings.TrimSpace(d.Get(totalKey))
	var total, err = strconv.Atoi(totalValue)
	if err != nil {
		return nil, []error{l.makeError(err)}
	}

	var errs = NewBlockErrors[int]()
	// var ordered = make(map[int]struct{})
	var data = make(ListBlockData, 0, total)
	for i := 0; i < total; i++ {
		var deletedKey = fmt.Sprintf("%s-%d--deleted", name, i)
		if d.Has(deletedKey) {
			var deletedValue = strings.TrimSpace(d.Get(deletedKey))
			if deletedValue == "on" || deletedValue == "true" || deletedValue == "1" {
				deletedCount++
				continue
			}
		}

		var key = fmt.Sprintf("%s-%d", name, i)
		if l.Child.ValueOmittedFromData(ctx, d, files, key) {
			continue
		}

		var (
			idKey    = fmt.Sprintf("%s-id-%d", name, i)
			orderKey = fmt.Sprintf("%s-order-%d", name, i)
			orderStr = d.Get(orderKey)
			idStr    = d.Get(idKey)
			id       uuid.UUID
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

		if orderStr == "" {
			errs.AddError(i, fmt.Errorf("Missing order key: %s", orderKey)) //lint:ignore ST1005 ignore this lint
			continue
		}

		var order int
		if order, err = strconv.Atoi(orderStr); err != nil {
			errs.AddError(i, fmt.Errorf("Invalid order: %s", orderStr)) //lint:ignore ST1005 ignore this lint
			continue
		}

		// if _, ok := ordered[order]; ok {
		// errs.AddError(i, fmt.Errorf("Duplicate order: %d", order)) //lint:ignore ST1005 ignore this lint
		// continue
		// }

		var value, e = l.Child.ValueFromDataDict(ctx, d, files, key)
		if len(e) != 0 {
			errs.AddError(i, e...)
			continue
		}

		data = append(data, &ListBlockValue{
			ID:    id,
			Order: order,
			Data:  value,
		})

		// ordered[order] = struct{}{}

		totalCount++
	}

	slices.SortStableFunc(
		data, sortListBlocks,
	)

	if errs.HasErrors() {
		return data, []error{errs}
	}

	if l.Min != -1 && len(data) < l.Min {
		return data, []error{l.makeError(
			fmt.Errorf("Must have at least %d items (has %d)", l.Min, len(data)), //lint:ignore ST1005 ignore this lint
		)}
	}

	if l.Max != -1 && len(data) > l.Max {
		return data, []error{l.makeError(
			fmt.Errorf("Must have at most %d items (has %d)", l.Max, len(data)), //lint:ignore ST1005 ignore this lint
		)}
	}

	if totalCount+deletedCount != total {
		return data, []error{l.makeError(
			fmt.Errorf("Invalid number of items, expected %d, got %d", total, totalCount+deletedCount), //lint:ignore ST1005 ignore this lint
		)}
	}

	return data, nil
}

func (l *ListBlock) ValueToGo(value interface{}) (interface{}, error) {
	if fields.IsZero(value) {
		return "", nil
	}
	var (
		valueArr ListBlockData
		ok       bool
	)

	if valueArr, ok = value.(ListBlockData); !ok {
		return value, fmt.Errorf("value must be of type ListBlockData, got %T", value)
	}

	var (
		newArr = make(ListBlockData, len(valueArr))
		errs   = NewBlockErrors[int]()
	)
	for i, lbVal := range valueArr {
		var childData, err = l.Child.ValueToGo(lbVal.Data)
		if err != nil {
			errs.AddError(i, err)
			continue
		}

		newArr[i] = &ListBlockValue{
			ID:    lbVal.ID,
			Order: lbVal.Order,
			Data:  childData,
		}
	}

	if errs.HasErrors() {
		return value, errs
	}

	return newArr, nil
}

func (l *ListBlock) GetDefault() interface{} {
	if l.Min > 0 {
		var getDefault = l.Child.GetDefault
		if l.Default != nil {
			getDefault = l.Default
		}

		var data = make(ListBlockData, l.Min)
		for i := 0; i < l.Min; i++ {
			data[i] = &ListBlockValue{
				ID:   uuid.New(),
				Data: getDefault(),
			}
		}

		return data
	}
	return make(ListBlockData, 0)
}

func (l *ListBlock) ValueToForm(value interface{}) interface{} {

	if fields.IsZero(value) {
		value = l.GetDefault()
	}

	var valueArr ListBlockData
	var ok bool
	if valueArr, ok = value.(ListBlockData); !ok {
		return value
	}

	var data = make(ListBlockData, 0, max(len(valueArr), l.Min))
	for i := 0; i < max(len(valueArr), l.Min); i++ {
		var v *ListBlockValue
		if i < len(valueArr) {
			v = valueArr[i]
		} else {
			v = &ListBlockValue{
				ID:   uuid.New(),
				Data: l.Child.GetDefault(),
			}
		}
		var lv = &ListBlockValue{
			ID:    v.ID,
			Order: i,
			Data:  l.Child.ValueToForm(v.Data),
		}
		data = append(data, lv)
	}

	return data
}

func (l *ListBlock) Clean(ctx context.Context, value interface{}) (interface{}, error) {
	if fields.IsZero(value) {
		return nil, nil
	}

	var data = make(ListBlockData, 0)
	for i, lbVal := range value.(ListBlockData) {
		var v, err = l.Child.Clean(ctx, lbVal.Data)
		if err != nil {
			return value, l.makeIndexedError(i, errors.Wrapf(err, "index %d", i))
		}

		data = append(data, &ListBlockValue{
			ID:    lbVal.ID,
			Order: lbVal.Order,
			Data:  v,
		})
	}

	return data, nil
}

func (l *ListBlock) Validate(ctx context.Context, value interface{}) []error {

	for _, validator := range l.Validators {
		if err := validator(ctx, value); err != nil {
			return []error{err}
		}
	}

	if fields.IsZero(value) {
		return nil
	}

	var errors = make([]error, 0)
	for i, v := range value.(ListBlockData) {
		var e = l.Child.Validate(ctx, v.Data)
		if len(e) != 0 {
			errors = append(errors, l.makeIndexedError(i, e...))
		}
	}
	return errors
}

func (l *ListBlock) RenderForm(ctx context.Context, w io.Writer, id, name string, value interface{}, errors []error, tplCtx ctx.Context) error {
	var (
		ctxData  = NewBlockContext(l, tplCtx)
		valueArr ListBlockData
		ok       bool
	)
	ctxData.ID = id
	ctxData.Name = name
	ctxData.Value = value

	if value == nil || value == "" {
		value = l.GetDefault()
	}

	if valueArr, ok = value.(ListBlockData); !ok {
		return fmt.Errorf("value must be a []interface{}")
	}

	var listBlockErrors = NewBlockErrors[int](errors...)
	var bt, err = telepath.PackJSON(ctx, JSContext, l)
	if err != nil {
		return err
	}

	return l.RenderTempl(id, name, valueArr, string(bt), listBlockErrors, ctxData).Render(ctx, w)
}
