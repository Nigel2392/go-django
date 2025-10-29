package blocks

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"

	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/filesystem"
	"github.com/Nigel2392/go-django/src/core/filesystem/tpl"
	"github.com/Nigel2392/go-django/src/forms/fields"
	"github.com/Nigel2392/go-telepath/telepath"
	"github.com/elliotchance/orderedmap/v2"
)

var _ Block = (*StructBlock)(nil)

type StructBlock struct {
	*BaseBlock
	Fields *orderedmap.OrderedMap[string, Block]
}

func NewStructBlock(opts ...func(*StructBlock)) *StructBlock {
	var m = &StructBlock{
		BaseBlock: NewBaseBlock(),
		Fields:    orderedmap.NewOrderedMap[string, Block](),
	}
	m.FormField = fields.CharField()
	for _, opt := range opts {
		opt(m)
	}
	return m
}

func (s *StructBlock) ValueFromDB(value json.RawMessage) (interface{}, error) {
	var dataMap = make(map[string]json.RawMessage)
	if len(value) == 0 {
		return nil, nil
	}

	if err := json.Unmarshal(value, &dataMap); err != nil {
		return nil, err
	}

	var data = newStructBlockValue(s, make(map[string]interface{}, len(dataMap)))
	var errors = NewBlockErrors[string]()
	for head := s.Fields.Front(); head != nil; head = head.Next() {
		var v, err = head.Value.ValueFromDB(dataMap[head.Key])
		if err != nil {
			errors.AddError(head.Key, err)
			continue
		}
		data.V[head.Key] = v
	}

	if errors.HasErrors() {
		return data, errors
	}

	return data, nil
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
	var errors = NewBlockErrors[string]()
	var data = newStructBlockValue(m, make(map[string]interface{}, m.Fields.Len()))
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
		data.V[key] = value
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

	valueMap, ok := value.(*StructBlockValue)
	if !ok {
		return value, fmt.Errorf("value must be a *StructBlockValue")
	}
	var errors = NewBlockErrors[string]()
	var data = newStructBlockValue(m, make(map[string]interface{}))
loop:
	for head := m.Fields.Front(); head != nil; head = head.Next() {
		var v, err = head.Value.ValueToGo(valueMap.V[head.Key])
		if err != nil {
			errors.AddError(head.Key, err)
			continue loop
		}

		data.V[head.Key] = v
	}

	if errors.HasErrors() {
		return value, errors
	}

	return data, nil
}

func (m *StructBlock) ValueToForm(value interface{}) interface{} {
	if value == nil {
		return value
	}

	valueMap, ok := value.(*StructBlockValue)
	if !ok {
		return value
	}

	var data = newStructBlockValue(m, make(map[string]interface{}))
	for head := m.Fields.Front(); head != nil; head = head.Next() {
		data.V[head.Key] = head.Value.ValueToForm(valueMap.V[head.Key])
	}

	return data
}

func (m *StructBlock) Clean(ctx context.Context, value interface{}) (interface{}, error) {
	if fields.IsZero(value) {
		return value, nil
	}

	var errs = NewBlockErrors[string]()
	var data = newStructBlockValue(m, make(map[string]interface{}))
	var valueMap = value.(*StructBlockValue)
	for head := m.Fields.Front(); head != nil; head = head.Next() {
		var v, err = head.Value.Clean(ctx, valueMap.V[head.Key])
		if err != nil {
			errs.AddError(head.Key, err)
			continue
		}

		data.V[head.Key] = v
	}

	if errs.HasErrors() {
		return data, errs
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

	valueMap, ok := value.(*StructBlockValue)
	if !ok {
		return []error{fmt.Errorf("value must be a *StructBlockValue")}
	}

	var errors = NewBlockErrors[string]()
	for head := m.Fields.Front(); head != nil; head = head.Next() {
		var e = head.Value.Validate(ctx, valueMap.V[head.Key])
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
	if m.Default != nil {
		return m.Default()
	}

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
	var bt, err = telepath.PackJSON(ctx, JSContext, m)
	if err != nil {
		return err
	}

	return m.RenderTempl(
		id, name, valueMap, string(bt), errs, ctxData,
	).Render(ctx, w)
}

func (l *StructBlock) ValueAtPath(bound BoundBlockValue, parts []string) (interface{}, error) {
	if len(parts) == 0 {
		return bound.Data(), nil
	}

	var val, ok = bound.(*StructBlockValue)
	if !ok {
		return nil, errors.TypeMismatch.Wrapf(
			"value must be a *StructBlockValue, got %T", bound.Data(),
		)
	}

	child, ok := l.Fields.Get(parts[0])
	if !ok {
		return nil, errors.FieldNotFound.Wrapf(
			"no such field: %q", parts[0],
		)
	}

	res, err := child.ValueAtPath(
		val.V[parts[0]].(BoundBlockValue),
		parts[1:],
	)
	if err != nil {
		err = errors.Wrapf(
			err, "[StructBlock] field %q", parts[0],
		)
	}
	return res, err
}

func (b *StructBlock) Render(ctx context.Context, w io.Writer, value interface{}, context ctx.Context) error {
	var blockCtx = NewBlockContext(b, context)
	if b.Template != "" {
		blockCtx.Value = value
		return tpl.FRender(w, blockCtx, b.Template)

	}

	var v, ok = value.(*StructBlockValue)
	if !ok {
		return fmt.Errorf("value must be a *StructBlockValue, got %T", value)
	}

	w.Write([]byte("<dl>"))
	for head := b.Fields.Front(); head != nil; head = head.Next() {
		w.Write([]byte("<dt>"))
		w.Write([]byte(head.Value.Label(ctx)))
		w.Write([]byte("</dt><dd>"))

		var val = v.V[head.Key]
		if err := head.Value.Render(ctx, w, val, context); err != nil {
			return err
		}

		w.Write([]byte("</dd>"))
	}
	w.Write([]byte("</dl>"))
	return nil
}
