package blocks

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"net/url"
	"strconv"
	"strings"

	"github.com/Nigel2392/django/core/assert"
	"github.com/Nigel2392/django/core/ctx"
	"github.com/Nigel2392/django/forms/fields"
	"github.com/pkg/errors"
)

var _ Block = (*ListBlock)(nil)

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
	if len(err) == 0 {
		return nil
	}
	var e = NewBlockErrors[int]()
	e.AddError(index, err...)
	return e
}

func (b *ListBlock) ValueOmittedFromData(data url.Values, files map[string][]io.ReadCloser, name string) bool {
	var addedKey = fmt.Sprintf("%sAdded", name)
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
	var data = make([]interface{}, 0)

	var (
		added    = 0
		addedKey = fmt.Sprintf("%sAdded", name)
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

		var value, e = l.Child.ValueFromDataDict(d, files, key)
		if len(e) != 0 {
			errs.AddError(i, e...)
			continue
		}

		data = append(data, value)
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
			return nil, l.makeIndexedError(i, errors.Wrapf(err, "index %d", i))
		}

		data = append(data, v)
	}

	return data, nil
}

func (l *ListBlock) GetDefault() interface{} {
	var data = make([]interface{}, l.Min)
	for i := 0; i < l.Min; i++ {
		data[i] = l.Child.GetDefault()
	}
	return data
}

func (l *ListBlock) ValueToForm(value interface{}) interface{} {

	if fields.IsZero(value) {
		value = l.GetDefault()
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
	if fields.IsZero(value) {
		return nil, nil
	}

	var data = make([]interface{}, 0)
	for i, v := range value.([]interface{}) {
		var v, err = l.Child.Clean(v)
		if err != nil {
			return nil, l.makeIndexedError(i, errors.Wrapf(err, "index %d", i))
		}

		data = append(data, v)
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
	for i, v := range value.([]interface{}) {
		var e = l.Child.Validate(v)
		if len(e) != 0 {
			errors = append(errors, l.makeIndexedError(i, e...))
		}
	}
	return errors
}

func (l *ListBlock) RenderForm(id, name string, value interface{}, errors []error, context ctx.Context) (template.HTML, error) {
	var (
		ctxData  = NewBlockContext(l, context)
		valueArr []interface{}
		ok       bool
	)
	ctxData.ID = id
	ctxData.Name = name
	ctxData.Value = value

	if value == nil || value == "" {
		value = l.GetDefault()
	}

	if valueArr, ok = value.([]interface{}); !ok {
		return "", fmt.Errorf("value must be a []interface{}")
	}

	var listBlockErrors = NewBlockErrors[int](errors...)
	var b = new(bytes.Buffer)

	b.WriteString(
		fmt.Sprintf(`<input type="hidden" name="%sAdded" value="%d">`, name, len(valueArr)),
	)

	for i, v := range valueArr {

		var (
			id  = fmt.Sprintf("%s-%d", name, i)
			key = fmt.Sprintf("%s-%d", name, i)
		)

		var v, err = l.Child.RenderForm(
			id, key,
			v,
			listBlockErrors.Get(i),
			ctxData,
		)
		if err != nil {
			return "", err
		}

		b.WriteString(string(v))
	}

	return template.HTML(b.String()), nil
}
