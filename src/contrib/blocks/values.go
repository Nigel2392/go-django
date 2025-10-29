package blocks

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"io"

	"github.com/Nigel2392/go-django/queries/src/drivers/dbtype"
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/internal/django_reflect"
)

var _ BoundBlockValue = (*BoundValue[interface{}])(nil)
var _ attrs.Binder = (*BoundValue[interface{}])(nil)

type BoundBlockValue interface {
	Block() Block
	Data() interface{}
}

type RenderableValue interface {
	Render(c context.Context, w io.Writer, ctxt ctx.Context) error
}

type BoundValue[T any] struct {
	BlockObject Block           `json:"-"`
	V           T               `json:"-"`
	_rawData    json.RawMessage `json:"-"`
}

func (l *BoundValue[T]) GoString() string {
	return fmt.Sprintf("BoundValue[%T]{V: %+v}", *new(T), l.V)
}

func (l *BoundValue[T]) String() string {
	return fmt.Sprintf("%v", l.V)
}

func (l *BoundValue[T]) Block() Block {
	return l.BlockObject
}

func (l *BoundValue[T]) Data() interface{} {
	return l.V
}

func (l *BoundValue[T]) BindToModel(model attrs.Definer, field attrs.Field) error {
	if l == nil {
		return nil
	}

	if l.BlockObject != nil {
		return l.loadData()
	}

	// fmt.Printf("Binding %T to model %T field %s\n%s\n", l, model, field.Name(), debug.Stack())
	var b, err = methodGetBlock(model, field)
	if err != nil {
		panic(fmt.Sprintf("blocks: failed to get block for field %s on %T: %v", field.Name(), model, err))
	}
	l.BlockObject = b
	return l.loadData()
}

func (l *BoundValue[T]) loadData() error {
	if l._rawData != nil {
		var data = l._rawData
		l._rawData = nil
		return l.LoadData(data)
	}
	return nil
}

func (l *BoundValue[T]) DBType() dbtype.Type {
	return dbtype.JSON
}

func (l *BoundValue[T]) MarshalJSON() ([]byte, error) {
	return json.Marshal(l.V)
}

func (l *BoundValue[T]) UnmarshalJSON(data []byte) error {
	l._rawData = json.RawMessage(data)
	if l.BlockObject == nil {
		return nil
	}
	return l.loadData()
}

func (l BoundValue[T]) Value() (driver.Value, error) {
	jsonData, err := json.Marshal(l.V)
	return string(jsonData), err
}

func (l *BoundValue[T]) Scan(value interface{}) (err error) {
	switch v := value.(type) {
	case []byte:
		l._rawData = json.RawMessage(v)
	case string:
		l._rawData = json.RawMessage(v)
	case nil:
		*l = BoundValue[T]{}
		return nil
	default:
		return errors.TypeMismatch.Wrapf(
			"cannot scan %T into BoundValue[%T]", value, *new(T),
		)
	}
	if l.BlockObject == nil {
		return nil
	}
	l.loadData()
	return nil
}

func (l *BoundValue[T]) LoadData(raw json.RawMessage) error {
	if len(raw) == 0 {
		return nil
	}
	data, err := l.BlockObject.ValueFromDB(raw)
	if err != nil {
		return err
	}
	if data == nil {
		return nil
	}
	l.V = data.(BoundBlockValue).Data().(T)
	return nil
}

func (l *BoundValue[T]) Render(c context.Context, w io.Writer, ctxt ctx.Context) error {
	if l == nil || django_reflect.IsZero(l.V) {
		return nil
	}
	return l.BlockObject.Render(c, w, l, ctxt)
}

type (
	FieldBlockValue  = BoundValue[interface{}]
	ListBlockValue   = BoundValue[[]*ListBlockData]
	StreamBlockValue = BoundValue[[]*StreamBlockData]
	StructBlockValue = BoundValue[map[string]interface{}]
)

func NewBlockValue[T any](block Block, data T) *BoundValue[T] {
	return &BoundValue[T]{
		BlockObject: block,
		V:           data,
	}
}

func newFieldBlockValue(block Block, data interface{}) *FieldBlockValue {
	return NewBlockValue[interface{}](block, data)
}

func newListBlockValue(block Block, data []*ListBlockData) *ListBlockValue {
	return NewBlockValue[[]*ListBlockData](block, data)
}

func newStreamBlockValue(block Block, data []*StreamBlockData) *StreamBlockValue {
	return NewBlockValue[[]*StreamBlockData](block, data)
}

func newStructBlockValue(block Block, data map[string]interface{}) *StructBlockValue {
	return &StructBlockValue{
		BlockObject: block,
		V:           data,
	}
}
