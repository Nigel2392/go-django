package blocks

import (
	"context"
	"fmt"
	"io"
	"maps"
	"net/url"

	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/filesystem"
	"github.com/Nigel2392/go-django/src/forms/fields"
	"github.com/Nigel2392/go-telepath/telepath"
	"github.com/elliotchance/orderedmap/v2"
)

var _ Block = (*StructBlock)(nil)

type StructBlock struct {
	*BaseBlock
	Fields *orderedmap.OrderedMap[string, Block]
	ToGo   func(map[string]interface{}) (interface{}, error)
	ToForm func(interface{}) (map[string]interface{}, error)
}

func NewStructBlock() *StructBlock {
	var m = &StructBlock{
		BaseBlock: NewBaseBlock(),
		Fields:    orderedmap.NewOrderedMap[string, Block](),
	}
	m.FormField = fields.CharField()
	return m
}

func (m *StructBlock) AddField(name string, block Block) {
	m.Fields.Set(name, block)
	block.SetName(name)
}

func (m *StructBlock) Name() string {
	return m.Name_
}

func (m *StructBlock) SetName(name string) {
	m.Name_ = name
}

func (m *StructBlock) Field() fields.Field {
	if m.FormField == nil {
		var field = fields.CharField()
		field.SetName(m.Name_)
	}
	return m.FormField
}

func (m *StructBlock) ValueOmittedFromData(ctx context.Context, data url.Values, files map[string][]filesystem.FileHeader, name string) bool {
	var omitted = true
	for head := m.Fields.Front(); head != nil; head = head.Next() {
		var key = fmt.Sprintf("%s-%s", name, head.Key)
		if !head.Value.ValueOmittedFromData(ctx, data, files, key) {
			omitted = false
			break
		}
	}
	return omitted
}

func (m *StructBlock) ValueFromDataDict(ctx context.Context, d url.Values, files map[string][]filesystem.FileHeader, name string) (interface{}, []error) {
	var data = make(map[string]interface{})
	var errors = NewBlockErrors[string]()
	for head := m.Fields.Front(); head != nil; head = head.Next() {
		var key = head.Key
		var block = head.Value
		var value, e = block.ValueFromDataDict(
			ctx, d, files, fmt.Sprintf("%s-%s", name, key),
		)
		if len(e) != 0 {
			errors.AddError(head.Key, e...)
			continue
		}
		data[key] = value
	}

	if errors.HasErrors() {
		return data, []error{errors}
	}

	return data, nil
}

func (m *StructBlock) ValueToGo(value interface{}) (interface{}, error) {
	if fields.IsZero(value) {
		return nil, nil
	}

	var (
		data     = make(map[string]interface{})
		valueMap map[string]interface{}
		ok       bool
	)

	if valueMap, ok = value.(map[string]interface{}); !ok {
		return nil, fmt.Errorf("value must be a map[string]interface{}")
	}
	var errors = NewBlockErrors[string]()
loop:
	for head := m.Fields.Front(); head != nil; head = head.Next() {
		var v, err = head.Value.ValueToGo(valueMap[head.Key])
		if err != nil {
			errors.AddError(head.Key, err)
			continue loop
		}

		data[head.Key] = v
	}

	if errors.HasErrors() {
		return nil, errors
	}

	if m.ToGo != nil {
		var v, err = m.ToGo(data)
		if err != nil {
			return nil, err
		}
		return v, nil
	}

	return data, nil
}

func (m *StructBlock) ValueToForm(value interface{}) interface{} {
	var data = make(map[string]interface{})

	if m.ToForm != nil {
		var v, _ = m.ToForm(value)
		maps.Copy(data, v)
	}

	if value == nil {
		return data
	}

	var valueMap map[string]interface{}
	var ok bool
	if valueMap, ok = value.(map[string]interface{}); !ok {
		return data
	}

	for head := m.Fields.Front(); head != nil; head = head.Next() {
		data[head.Key] = head.Value.ValueToForm(valueMap[head.Key])
	}

	return data
}

func (m *StructBlock) Clean(ctx context.Context, value interface{}) (interface{}, error) {
	if fields.IsZero(value) {
		return nil, nil
	}

	var data = make(map[string]interface{})
	var errs = NewBlockErrors[string]()
	var valueMap = value.(map[string]interface{})
	for head := m.Fields.Front(); head != nil; head = head.Next() {
		var v, err = head.Value.Clean(ctx, valueMap[head.Key])
		if err != nil {
			errs.AddError(head.Key, err)
			continue
		}

		data[head.Key] = v
	}

	if errs.HasErrors() {
		return nil, errs
	}

	return data, nil
}

func (m *StructBlock) Validate(ctx context.Context, value interface{}) []error {

	for _, validator := range m.Validators {
		if err := validator(ctx, value); err != nil {
			return []error{err}
		}
	}

	if fields.IsZero(value) {
		return nil
	}

	var valueMap map[string]interface{}
	var ok bool
	if valueMap, ok = value.(map[string]interface{}); !ok {
		return []error{fmt.Errorf("value must be a map[string]interface{}")}
	}

	var errors = NewBlockErrors[string]()
	for head := m.Fields.Front(); head != nil; head = head.Next() {
		var e = head.Value.Validate(ctx, valueMap[head.Key])
		if len(e) != 0 {
			errors.AddError(head.Key, e...)
		}
	}

	if errors.HasErrors() {
		return []error{errors}
	}

	return nil
}

func (m *StructBlock) GetDefault() interface{} {
	var data = make(map[string]interface{})
	for head := m.Fields.Front(); head != nil; head = head.Next() {
		data[head.Key] = head.Value.GetDefault()
	}
	return data
}

func (m *StructBlock) RenderForm(ctx context.Context, w io.Writer, id, name string, value interface{}, errors []error, tplCtx ctx.Context) error {
	var (
		ctxData  = NewBlockContext(m, tplCtx)
		valueMap map[string]interface{}
		ok       bool
	)

	if value == nil {
		value = m.GetDefault()
	}

	ctxData.ID = id
	ctxData.Name = name
	ctxData.Value = value

	if valueMap, ok = value.(map[string]interface{}); !ok {
		return fmt.Errorf("value must be a map[string]interface{}")
	}

	var errs = NewBlockErrors[string](errors...)

	var blockArgs = map[string]interface{}{
		"id":    id,
		"name":  name,
		"block": m,
		"childBlockDefs": (func() any {
			var blocks = make([]any, 0)
			for head := m.Fields.Front(); head != nil; head = head.Next() {
				var value = head.Value.ValueToForm(
					valueMap[head.Key],
				)

				blocks = append(blocks, map[string]interface{}{
					"block": head.Value,
					"value": value,
					"errors": (func() []error {
						var errors = errs.Get(head.Key)
						if errors != nil {
							return errors
						}
						return nil
					})(),
				})
			}
			return blocks
		}()),
	}

	var bt, err = telepath.PackJSON(JSContext, blockArgs)
	if err != nil {
		return err
	}

	return m.RenderTempl(
		ctx, id, name, valueMap, string(bt), errs, ctxData,
	).Render(context.Background(), w)
}

func (m *StructBlock) Adapter() telepath.Adapter {
	var ctx = context.Background()
	return &telepath.ObjectAdapter[*StructBlock]{
		JSConstructor: "django.blocks.StructBlock",
		GetJSArgs: func(obj *StructBlock) []interface{} {

			var fields = make(map[string]interface{})
			for head := obj.Fields.Front(); head != nil; head = head.Next() {
				fields[head.Key] = head.Value
			}

			return []interface{}{map[string]interface{}{
				"name":     obj.Name(),
				"label":    obj.Label(ctx),
				"helpText": obj.HelpText(ctx),
				"required": obj.Field().Required(),
				"fields":   fields,
			}}
		},
	}
}
