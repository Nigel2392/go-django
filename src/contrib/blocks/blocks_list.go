package blocks

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/src/core/assert"
	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/filesystem"
	"github.com/Nigel2392/go-django/src/core/filesystem/tpl"
	"github.com/Nigel2392/go-django/src/forms/fields"
	"github.com/Nigel2392/go-telepath/telepath"
	"github.com/google/uuid"
)

var _ Block = (*ListBlock)(nil)

type ListBlockData struct {
	ID    uuid.UUID   `json:"id"`
	Order int         `json:"order"`
	Data  interface{} `json:"data"`
}

type JSONListBlockData struct { // used only for deserialization
	ID    uuid.UUID       `json:"id"`
	Order int             `json:"order"`
	Data  json.RawMessage `json:"data"`
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

func (s *ListBlock) ValueFromDB(value json.RawMessage) (interface{}, error) {
	var dataList = make([]JSONListBlockData, 0)
	if len(value) == 0 {
		return nil, nil
	}

	if err := json.Unmarshal(value, &dataList); err != nil {
		return nil, err
	}

	var data = newListBlockValue(s, make([]*ListBlockData, len(dataList)))
	var errors = NewBlockErrors[int]()
	for i, item := range dataList {
		var v, err = s.Child.ValueFromDB(item.Data)
		data.V[i] = &ListBlockData{
			ID:    item.ID,
			Order: i,
			Data:  v,
		}
		if err != nil {
			errors.AddError(i, err)
			continue
		}
	}

	if errors.HasErrors() {
		return data, errors
	}

	return data, nil
}

func (b *ListBlock) ValueOmittedFromData(ctx context.Context, data url.Values, files map[string][]filesystem.FileHeader, name string) bool {
	return !data.Has(fmt.Sprintf("%s--total", name))
}

func sortListBlocks(a, b *ListBlockData) int {
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
	var data = newListBlockValue(l, make([]*ListBlockData, 0, total))
	for i := 0; i < total; i++ {
		var deletedKey = fmt.Sprintf("%s-%d--deleted", name, i)
		if d.Has(deletedKey) {
			var deletedValue = strings.TrimSpace(d.Get(deletedKey))
			if deletedValue == "on" || deletedValue == "true" || deletedValue == "1" {
				deletedCount++
				continue
			}
		}

		var (
			key      = fmt.Sprintf("%s-%d", name, i)
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

		data.V = append(data.V, &ListBlockData{
			ID:    id,
			Order: order,
			Data:  value,
		})

		// ordered[order] = struct{}{}

		totalCount++
	}

	slices.SortStableFunc(
		data.V, sortListBlocks,
	)

	if totalCount+deletedCount != total {
		errs.AddNonBlockError(fmt.Errorf("Invalid number of items, expected %d, got %d", total, totalCount+deletedCount)) //lint:ignore ST1005 ignore this lint
	}

	if errs.HasErrors() {
		return data, []error{errs}
	}

	return data, nil
}

func (l *ListBlock) ValueToGo(value interface{}) (interface{}, error) {
	if fields.IsZero(value) {
		return "", nil
	}
	var (
		valueArr *ListBlockValue
		ok       bool
	)

	if valueArr, ok = value.(*ListBlockValue); !ok {
		return value, fmt.Errorf("value must be of type ListBlockData, got %T", value)
	}

	var (
		newArr = newListBlockValue(l, make([]*ListBlockData, len(valueArr.V)))
		errs   = NewBlockErrors[int]()
	)
	for i, lbVal := range valueArr.V {
		var childData, err = l.Child.ValueToGo(lbVal.Data)
		if err != nil {
			errs.AddError(i, err)
			continue
		}

		newArr.V[i] = &ListBlockData{
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

		var data = newListBlockValue(l, make([]*ListBlockData, l.Min))
		for i := 0; i < l.Min; i++ {
			data.V[i] = &ListBlockData{
				ID:   uuid.New(),
				Data: getDefault(),
			}
		}

		return data
	}
	return newListBlockValue(l, make([]*ListBlockData, 0))
}

func (l *ListBlock) ValueToForm(value interface{}) interface{} {
	if fields.IsZero(value) {
		value = l.GetDefault()
	}

	var valueArr *ListBlockValue
	var ok bool
	if valueArr, ok = value.(*ListBlockValue); !ok {
		return value
	}

	var data = newListBlockValue(l, make([]*ListBlockData, 0, max(len(valueArr.V), l.Min)))
	for i := 0; i < max(len(valueArr.V), l.Min); i++ {
		var v *ListBlockData
		if i < len(valueArr.V) {
			v = valueArr.V[i]
		} else {
			v = &ListBlockData{
				ID:   uuid.New(),
				Data: l.Child.GetDefault(),
			}
		}
		var lv = &ListBlockData{
			ID:    v.ID,
			Order: i,
			Data:  l.Child.ValueToForm(v.Data),
		}
		data.V = append(data.V, lv)
	}

	return data
}

func (l *ListBlock) Clean(ctx context.Context, value interface{}) (interface{}, error) {
	if fields.IsZero(value) {
		return nil, nil
	}

	var errs = NewBlockErrors[int]()
	var data = newListBlockValue(l, make([]*ListBlockData, 0))
	for i, lbVal := range value.(*ListBlockValue).V {
		var v, err = l.Child.Clean(ctx, lbVal.Data)
		if err != nil {
			errs.AddError(i, errors.Wrapf(err, "index %d", i))
			data.V = append(data.V, &ListBlockData{
				ID:    lbVal.ID,
				Order: lbVal.Order,
				Data:  lbVal.Data,
			})
			continue
		}

		data.V = append(data.V, &ListBlockData{
			ID:    lbVal.ID,
			Order: lbVal.Order,
			Data:  v,
		})
	}

	return data, nil
}

func (l *ListBlock) Validate(ctx context.Context, value interface{}) []error {

	var data, ok = value.(*ListBlockValue)
	if !ok {
		return []error{fmt.Errorf("value must be a *ListBlockValue")}
	}

	var errs = NewBlockErrors[int]()
	if l.Min != -1 && len(data.V) < l.Min {
		errs.AddNonBlockError(fmt.Errorf("Must have at least %d items (has %d)", l.Min, len(data.V))) //lint:ignore ST1005 ignore this lint
		return []error{errs}
	}

	if l.Max != -1 && len(data.V) > l.Max {
		errs.AddNonBlockError(fmt.Errorf("Must have at most %d items (has %d)", l.Max, len(data.V))) //lint:ignore ST1005 ignore this lint
		return []error{errs}
	}

	for _, validator := range l.Validators {
		if err := validator(ctx, value); err != nil {
			return []error{err}
		}
	}

	if data == nil {
		return nil
	}

	for i, v := range data.V {
		var e = l.Child.Validate(ctx, v.Data)
		if len(e) != 0 {
			errs.AddError(i, e...)
		}
	}

	if errs.HasErrors() {
		return []error{errs}
	}

	return nil
}

func (l *ListBlock) RenderForm(ctx context.Context, w io.Writer, id, name string, value interface{}, errors []error, tplCtx ctx.Context) error {
	var (
		ctxData  = NewBlockContext(l, tplCtx)
		valueArr *ListBlockValue
		ok       bool
	)
	ctxData.ID = id
	ctxData.Name = name
	ctxData.Value = value

	if value == nil || value == "" {
		value = l.GetDefault()
	}

	if valueArr, ok = value.(*ListBlockValue); !ok {
		return fmt.Errorf("value must be a []interface{}")
	}

	var listBlockErrors = NewBlockErrors[int](errors...)
	var bt, err = telepath.PackJSON(ctx, JSContext, l)
	if err != nil {
		return err
	}

	return l.RenderTempl(id, name, valueArr, string(bt), listBlockErrors, ctxData).Render(ctx, w)
}

func (l *ListBlock) ValueAtPath(bound BoundBlockValue, parts []string) (interface{}, error) {
	if len(parts) == 0 {
		return bound.Data(), nil
	}

	var val, ok = bound.(*ListBlockValue)
	if !ok {
		return nil, errors.TypeMismatch.Wrapf(
			"[ListBlock] value must be a *ListBlockValue, got %T", bound.Data(),
		)
	}

	index, err := strconv.Atoi(parts[0])
	if err != nil {
		return nil, fmt.Errorf("[ListBlock] invalid index: %s", parts[0])
	}

	if index < 0 || index >= len(val.V) {
		return nil, fmt.Errorf("[ListBlock] index out of range: %d", index)
	}

	res, err := l.Child.ValueAtPath(
		val.V[index].Data.(BoundBlockValue),
		parts[1:],
	)
	if err != nil {
		err = errors.Wrapf(
			err, "[ListBlock] index %d", index,
		)
	}
	return res, err
}

func (b *ListBlock) Render(ctx context.Context, w io.Writer, value interface{}, context ctx.Context) error {
	var blockCtx = NewBlockContext(b, context)
	if b.Template != "" {
		blockCtx.Value = value
		return tpl.FRender(w, blockCtx, b.Template)

	}

	var v, ok = value.(*ListBlockValue)
	if !ok {
		return fmt.Errorf("value must be a *ListBlockValue")
	}
	for _, item := range v.V {
		if err := b.Child.Render(ctx, w, item.Data, context); err != nil {
			return err
		}
	}
	return nil
}
