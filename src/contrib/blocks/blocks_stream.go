package blocks

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/filesystem"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/Nigel2392/go-django/src/forms/fields"
	"github.com/Nigel2392/go-telepath/telepath"
	"github.com/elliotchance/orderedmap/v2"
	"github.com/google/uuid"
)

var (
	_ Block = (*StreamBlock)(nil)
)

type StreamBlockData struct {
	ID    uuid.UUID   `json:"id"`
	Type  string      `json:"type"`
	Data  interface{} `json:"data"`
	Order int         `json:"-"`
}

type JSONStreamBlockData struct {
	ID   uuid.UUID       `json:"id"`
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

type StreamBlockValue struct {
	Block      *StreamBlock
	BlocksJSON []JSONStreamBlockData
	Blocks     []*StreamBlockData
}

func newStreamBlockValue(block *StreamBlock) *StreamBlockValue {
	return &StreamBlockValue{
		Block:  block,
		Blocks: make([]*StreamBlockData, 0),
	}
}

func (s *StreamBlockValue) addStreamValue(v *StreamBlockData) {
	s.Blocks = append(s.Blocks, v)
}

var _ attrs.Binder = (*StreamBlockValue)(nil)

func (s *StreamBlockValue) BindToModel(model attrs.Definer, field attrs.Field) error {
	block, ok := methodGetBlock(model, field.Name())
	if !ok {
		return errors.ValueError.Wrapf(
			"No Get%sBlock() method found on %T, cannot bind StreamBlockValue", field.Name(), model,
		)
	}
	s.Block = block.(*StreamBlock)
	return nil
}

func (s StreamBlockValue) Value() (driver.Value, error) {
	jsonData, err := json.Marshal(s)
	return jsonData, err
}

func (s *StreamBlockValue) Scan(value interface{}) (err error) {
	var jsons = make([]JSONStreamBlockData, 0)
	switch v := value.(type) {
	case []byte:
		err = json.Unmarshal(v, &jsons)
	case string:
		err = json.Unmarshal([]byte(v), &jsons)
	case nil:
		*s = StreamBlockValue{}
		return nil
	default:
		return fmt.Errorf("cannot scan %T into StreamBlockValue", value)
	}
	if err != nil {
		return errors.Wrap(err, "unmarshal StreamBlockValue")
	}

	s.BlocksJSON = jsons
	return nil
}

func (s *StreamBlockValue) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.Blocks)
}

func (s *StreamBlockValue) UnmarshalJSON(data []byte) error {
	var jsons = make([]JSONStreamBlockData, 0)
	err := json.Unmarshal(data, &jsons)
	if err != nil {
		return errors.Wrap(err, "unmarshal StreamBlockValue")
	}

	s.BlocksJSON = jsons
	return nil
}

type StreamBlock struct {
	*BaseBlock `json:"-"`
	Children   *orderedmap.OrderedMap[string, Block] `json:"-"`
	Min        int                                   `json:"-"`
	Max        int                                   `json:"-"`
}

func NewStreamBlock(opts ...func(*StreamBlock)) *StreamBlock {

	var l = &StreamBlock{
		BaseBlock: NewBaseBlock(),
		Children:  orderedmap.NewOrderedMap[string, Block](),
		Min:       -1,
		Max:       -1,
	}

	l.FormField = fields.CharField()

	for _, opt := range opts {
		opt(l)
	}

	return l
}

func (l *StreamBlock) MinNum() int {
	return l.Min
}

func (l *StreamBlock) MaxNum() int {
	return l.Max
}

func (l *StreamBlock) AddField(name string, block Block) {
	l.Children.Set(name, block)
	block.SetName(name)
}

func (l *StreamBlock) makeError(err error) error {
	return err
}

func (b *StreamBlock) ValueOmittedFromData(ctx context.Context, data url.Values, files map[string][]filesystem.FileHeader, name string) bool {
	return !data.Has(fmt.Sprintf("%s--total", name))
}

func sortStreamBlocks(a, b *StreamBlockData) int {
	if a.Order < b.Order {
		return -1
	}
	if a.Order > b.Order {
		return 1
	}
	return 0
}

func (l *StreamBlock) ValueFromDataDict(ctx context.Context, d url.Values, files map[string][]filesystem.FileHeader, name string) (interface{}, []error) {
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
	var data = newStreamBlockValue(l)
	for i := 0; i < total; i++ {
		var deletedKey = fmt.Sprintf("%s-%d--deleted", name, i)
		if d.Has(deletedKey) {
			var deletedValue = strings.TrimSpace(d.Get(deletedKey))
			if deletedValue == "on" || deletedValue == "true" || deletedValue == "1" {
				deletedCount++
				continue
			}
		}

		var typeKey = fmt.Sprintf("%s-%d--type", name, i)
		if !d.Has(typeKey) {
			errs.AddError(i, fmt.Errorf("Missing type key: %s", typeKey)) //lint:ignore ST1005 ignore this lint
			continue
		}

		var typeValue = strings.TrimSpace(d.Get(typeKey))
		var child, ok = l.Children.Get(typeValue)
		if !ok {
			logger.Warnf("Unknown child block type: %s for StreamBlock", typeValue, name)
			// error is not needed - maybe the block type was removed
			continue
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

		var value, e = child.ValueFromDataDict(ctx, d, files, key)
		if len(e) != 0 {
			errs.AddError(i, e...)
			continue
		}

		data.addStreamValue(&StreamBlockData{
			ID:    id,
			Type:  typeValue,
			Data:  value,
			Order: order,
		})

		// ordered[order] = struct{}{}

		totalCount++
	}

	slices.SortStableFunc(
		data.Blocks, sortStreamBlocks,
	)

	if errs.HasErrors() {
		return data, []error{errs}
	}

	if l.Min != -1 && len(data.Blocks) < l.Min {
		errs.AddNonBlockError(fmt.Errorf("Must have at least %d items (has %d)", l.Min, len(data.Blocks))) //lint:ignore ST1005 ignore this lint
		return data, []error{errs}
	}

	if l.Max != -1 && len(data.Blocks) > l.Max {
		errs.AddNonBlockError(fmt.Errorf("Must have at most %d items (has %d)", l.Max, len(data.Blocks))) //lint:ignore ST1005 ignore this lint
		return data, []error{errs}
	}

	if totalCount+deletedCount != total {
		errs.AddNonBlockError(fmt.Errorf("Invalid number of items, expected %d, got %d", total, totalCount+deletedCount)) //lint:ignore ST1005 ignore this lint
		return data, []error{errs}
	}

	return data, nil
}

func (l *StreamBlock) ValueToGo(value interface{}) (interface{}, error) {
	if fields.IsZero(value) {
		return "", nil
	}
	var (
		valueArr *StreamBlockValue
		ok       bool
	)

	if valueArr, ok = value.(*StreamBlockValue); !ok {
		return value, fmt.Errorf("value must be of type StreamBlockValue, got %T", value)
	}

	var (
		newArr = make([]*StreamBlockData, len(valueArr.Blocks))
		errs   = NewBlockErrors[int]()
	)
	for i, lbVal := range valueArr.Blocks {
		var child, ok = l.Children.Get(lbVal.Type)
		if !ok {
			continue // this really shouldn't happen
		}

		var childData, err = child.ValueToGo(lbVal.Data)
		if err != nil {
			errs.AddError(i, err)
			continue
		}

		newArr[i] = &StreamBlockData{
			ID:    lbVal.ID,
			Type:  lbVal.Type,
			Order: lbVal.Order,
			Data:  childData,
		}
	}

	valueArr.Blocks = newArr
	if errs.HasErrors() {
		return valueArr, errs
	}

	return valueArr, nil
}

//
//	func (l *StreamBlock) GetDefault() interface{} {
//		if l.Min > 0 {
//			var getDefault = l.Child.GetDefault
//			if l.Default != nil {
//				getDefault = l.Default
//			}
//
//			var data = make(StreamBlockData, l.Min)
//			for i := 0; i < l.Min; i++ {
//				data[i] = &StreamBlockValue{
//					ID:   uuid.New(),
//					Data: getDefault(),
//				}
//			}
//
//			return data
//		}
//		return make(StreamBlockData, 0)
//	}

func (l *StreamBlock) ValueToForm(value interface{}) interface{} {

	if fields.IsZero(value) {
		value = l.GetDefault()
	}

	var blockData *StreamBlockValue
	var ok bool
	if blockData, ok = value.(*StreamBlockValue); !ok {
		return value
	}

	var data = make([]*StreamBlockData, 0, len(blockData.Blocks))
	for i := 0; i < len(blockData.Blocks); i++ {
		var v = blockData.Blocks[i]
		var child, ok = l.Children.Get(v.Type)
		if !ok {
			continue // this really shouldn't happen
		}

		var lv = &StreamBlockData{
			ID:    v.ID,
			Order: v.Order,
			Type:  v.Type,
			Data:  child.ValueToForm(v.Data),
		}

		data = append(data, lv)
	}

	blockData.Blocks = data
	return blockData
}

func (l *StreamBlock) Clean(ctx context.Context, value interface{}) (interface{}, error) {
	if fields.IsZero(value) {
		return nil, nil
	}

	var blockData *StreamBlockValue
	var ok bool
	if blockData, ok = value.(*StreamBlockValue); !ok {
		return value, fmt.Errorf("value must be of type StreamBlockValue, got %T", value)
	}

	var errs = NewBlockErrors[int]()
	var data = make([]*StreamBlockData, 0, len(blockData.Blocks))
	for i, lbVal := range blockData.Blocks {
		var child, ok = l.Children.Get(lbVal.Type)
		if !ok {
			continue // this really shouldn't happen
		}

		var v, err = child.Clean(ctx, lbVal.Data)
		if err != nil {
			errs.AddError(i, errors.Wrapf(err, "index %d", i))
			data = append(data, &StreamBlockData{
				ID:    lbVal.ID,
				Type:  lbVal.Type,
				Order: i,
				Data:  lbVal.Data,
			})
			continue
		}

		data = append(data, &StreamBlockData{
			ID:    lbVal.ID,
			Type:  lbVal.Type,
			Order: lbVal.Order,
			Data:  v,
		})
	}

	if errs.HasErrors() {
		return blockData, errs
	}

	blockData.Blocks = data
	return blockData, nil
}

func (l *StreamBlock) Validate(ctx context.Context, value interface{}) []error {

	for _, validator := range l.Validators {
		if err := validator(ctx, value); err != nil {
			return []error{err}
		}
	}

	if fields.IsZero(value) {
		return nil
	}

	var errs = NewBlockErrors[int]()
	for i, v := range value.(*StreamBlockValue).Blocks {
		var child, ok = l.Children.Get(v.Type)
		if !ok {
			continue // this really shouldn't happen
		}

		var e = child.Validate(ctx, v.Data)
		if len(e) != 0 {
			errs.AddError(i, e...)
		}
	}
	if errs.HasErrors() {
		return []error{errs}
	}
	return nil
}

func (l *StreamBlock) RenderForm(ctx context.Context, w io.Writer, id, name string, value interface{}, errors []error, tplCtx ctx.Context) error {
	var (
		ctxData = NewBlockContext(l, tplCtx)
		val     *StreamBlockValue
		ok      bool
	)
	ctxData.ID = id
	ctxData.Name = name
	ctxData.Value = value

	if value == nil || value == "" {
		value = l.GetDefault()
	}

	if val, ok = value.(*StreamBlockValue); !ok {
		return fmt.Errorf("value must be a []interface{}")
	}

	var blockErrs = NewBlockErrors[int](errors...)
	var bt, err = telepath.PackJSON(ctx, JSContext, l)
	if err != nil {
		return err
	}

	return l.RenderTempl(id, name, val, string(bt), blockErrs, ctxData).Render(ctx, w)
}
