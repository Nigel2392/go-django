package blocks

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"maps"
	"net/url"

	"github.com/Nigel2392/django/core/ctx"
	"github.com/Nigel2392/django/forms/fields"
	"github.com/Nigel2392/django/forms/widgets"
	"github.com/elliotchance/orderedmap/v2"
	"github.com/pkg/errors"
)

var _ Block = (*BaseBlock)(nil)
var _ Block = (*MultiBlock)(nil)

func CharBlock() Block {
	var base = NewBaseBlock()
	base.Template = "blocks/templates/text.html"
	base.FormField = fields.CharField()
	return base
}

func NumberBlock() Block {
	var base = NewBaseBlock()
	base.Template = "blocks/templates/number.html"
	base.FormField = fields.NumberField[int]()
	return base
}

func TextBlock() Block {
	var base = NewBaseBlock()
	base.Template = "blocks/templates/text.html"
	base.FormField = fields.CharField(
		fields.Widget(widgets.NewTextarea(nil)),
	)
	return base
}

type MultiBlock struct {
	*BaseBlock
	Fields *orderedmap.OrderedMap[string, Block]
	ToGo   func(map[string]interface{}) (interface{}, error)
	ToForm func(interface{}) (map[string]interface{}, error)
}

func NewMultiBlock() *MultiBlock {
	var m = &MultiBlock{
		BaseBlock: NewBaseBlock(),
		Fields:    orderedmap.NewOrderedMap[string, Block](),
	}
	m.FormField = fields.CharField()
	return m
}

func (m *MultiBlock) AddField(name string, block Block) {
	m.Fields.Set(name, block)
	block.SetName(name)
}

func (m *MultiBlock) Name() string {
	return m.Name_
}

func (m *MultiBlock) SetName(name string) {
	m.Name_ = name
}

func (m *MultiBlock) Field() fields.Field {
	if m.FormField == nil {
		var field = fields.CharField()
		field.SetName(m.Name_)
	}
	return m.FormField
}

func (m *MultiBlock) ValueFromDataDict(d url.Values, files map[string][]io.ReadCloser, name string) (interface{}, []error) {
	var data = make(map[string]interface{})
	for head := m.Fields.Front(); head != nil; head = head.Next() {
		var key = head.Key
		var block = head.Value
		var value, errs = block.ValueFromDataDict(
			d, files, fmt.Sprintf("%s-%s", name, key),
		)
		if len(errs) != 0 {
			return nil, errs
		}
		data[key] = value
	}

	return data, nil
}

func (m *MultiBlock) ValueToGo(value interface{}) (interface{}, error) {
	if value == nil {
		return "", nil
	}
	var (
		data     = make(map[string]interface{})
		valueMap map[string]interface{}
		ok       bool
	)

	if valueMap, ok = value.(map[string]interface{}); !ok {
		return nil, fmt.Errorf("value must be a map[string]interface{}")
	}

	for head := m.Fields.Front(); head != nil; head = head.Next() {
		var v, err = head.Value.ValueToGo(valueMap[head.Key])
		if err != nil {
			return nil, errors.Wrapf(err, "field %s", head.Key)
		}

		data[head.Key] = v
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

func (m *MultiBlock) ValueToForm(value interface{}) interface{} {
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

func (m *MultiBlock) Clean(value interface{}) (interface{}, error) {
	var data = make(map[string]interface{})
	for head := m.Fields.Front(); head != nil; head = head.Next() {
		var v, err = head.Value.Clean(data[head.Key])
		if err != nil {
			return nil, err
		}

		data[head.Key] = v
	}

	return data, nil
}

func (m *MultiBlock) GetDefault() interface{} {
	var data = make(map[string]interface{})
	for head := m.Fields.Front(); head != nil; head = head.Next() {
		data[head.Key] = head.Value.GetDefault()
	}
	return data
}

func (m *MultiBlock) RenderForm(id, name string, value interface{}, context ctx.Context) (template.HTML, error) {
	var (
		ctxData  = NewBlockContext(m, context)
		valueMap map[string]interface{}
		ok       bool
	)
	ctxData.ID = id
	ctxData.Name = name
	ctxData.Value = value

	if value == nil {
		value = m.GetDefault()
	}

	if valueMap, ok = value.(map[string]interface{}); !ok {
		return "", fmt.Errorf("value must be a map[string]interface{}")
	}

	var b = new(bytes.Buffer)
	for head := m.Fields.Front(); head != nil; head = head.Next() {

		var (
			id  = fmt.Sprintf("%s-%s", id, head.Key)
			key = fmt.Sprintf("%s-%s", name, head.Key)
		)

		var v, err = head.Value.RenderForm(
			id, key,
			valueMap[head.Key],
			ctxData,
		)
		if err != nil {
			return "", err
		}

		b.WriteString(string(v))
	}

	return template.HTML(b.String()), nil
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
	if len(minMax) > 2 {
		panic("Too many arguments (min, max)")
	}
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

func (l *ListBlock) ValueFromDataDict(d url.Values, files map[string][]io.ReadCloser, name string) (interface{}, []error) {
	var data = make([]interface{}, 0)
	for i := 0; ; i++ {
		var key = fmt.Sprintf("%s-%d", name, i)
		if !d.Has(key) {
			break
		}

		var value, errs = l.Child.ValueFromDataDict(d, files, key)
		if len(errs) != 0 {
			return nil, errs
		}

		data = append(data, value)
	}

	return data, nil
}

func (l *ListBlock) ValueToGo(value interface{}) (interface{}, error) {
	if value == nil {
		return "", nil
	}
	var (
		data     = make([]interface{}, 0)
		valueArr []interface{}
		ok       bool
	)

	if valueArr, ok = value.([]interface{}); !ok {
		return nil, fmt.Errorf("value must be a []interface{}")
	}

	for i, v := range valueArr {
		var v, err = l.Child.ValueToGo(v)
		if err != nil {
			return nil, errors.Wrapf(err, "index %d", i)
		}

		data = append(data, v)
	}

	return data, nil
}

func (l *ListBlock) ValueToForm(value interface{}) interface{} {

	if value == nil {
		return ""
	}

	var valueArr []interface{}
	var ok bool
	if valueArr, ok = value.([]interface{}); !ok {
		return ""
	}

	var data = make([]interface{}, 0, len(valueArr))
	for _, v := range valueArr {
		data = append(data, l.Child.ValueToForm(v))
	}

	return data
}

func (l *ListBlock) Clean(value interface{}) (interface{}, error) {
	var data = make([]interface{}, 0)
	for i, v := range value.([]interface{}) {
		var v, err = l.Child.Clean(v)
		if err != nil {
			return nil, errors.Wrapf(err, "index %d", i)
		}

		data = append(data, v)
	}

	return data, nil
}

func (l *ListBlock) renderDefaults(id, name string, ctxData *BlockContext) (template.HTML, error) {

	fmt.Println("renderDefaults", id, name, ctxData, l.MinNum())
	var b = new(bytes.Buffer)
	for i := 0; i < l.MinNum(); i++ {
		var (
			id  = fmt.Sprintf("%s-%d", id, i)
			key = fmt.Sprintf("%s-%d", name, i)
		)

		var v, err = l.Child.RenderForm(
			id, key,
			nil,
			ctxData,
		)
		if err != nil {
			fmt.Println("error", err)
			return "", err
		}

		fmt.Println("v", v)

		b.WriteString(string(v))
	}

	return template.HTML(b.String()), nil
}

func (l *ListBlock) RenderForm(id, name string, value interface{}, context ctx.Context) (template.HTML, error) {
	var (
		ctxData  = NewBlockContext(l, context)
		valueArr []interface{}
		ok       bool
	)
	ctxData.ID = id
	ctxData.Name = name
	ctxData.Value = value

	if value == nil || value == "" {
		return l.renderDefaults(id, name, ctxData)
	}

	if valueArr, ok = value.([]interface{}); !ok {
		return "", fmt.Errorf("value must be a []interface{}")
	}

	var b = new(bytes.Buffer)
	for i, v := range valueArr {

		var (
			id  = fmt.Sprintf("%s-%d", name, i)
			key = fmt.Sprintf("%s-%d", name, i)
		)

		var v, err = l.Child.RenderForm(
			id, key,
			v,
			ctxData,
		)
		if err != nil {
			return "", err
		}

		b.WriteString(string(v))
	}

	return template.HTML(b.String()), nil
}
