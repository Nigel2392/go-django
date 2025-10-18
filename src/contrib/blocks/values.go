package blocks

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"github.com/Nigel2392/go-django/queries/src/drivers/dbtype"
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/src/core/attrs"
)

type BoundValue[T any] struct {
	Block    Block           `json:"-"`
	V        T               `json:"-"`
	_rawData json.RawMessage `json:"-"`
}

var _ attrs.Binder = (*BoundValue[any])(nil)

func (l *BoundValue[T]) BindToModel(model attrs.Definer, field attrs.Field) error {
	if l == nil {
		return nil
	}

	if l.Block != nil {
		return l.loadData()
	}

	// fmt.Printf("Binding %T to model %T field %s\n%s\n", l, model, field.Name(), debug.Stack())
	var b, err = methodGetBlock(model, field)
	if err != nil {
		panic(fmt.Sprintf("blocks: failed to get block for field %s on %T: %v", field.Name(), model, err))
	}
	l.Block = b
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
	if l.Block == nil {
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
	if l.Block == nil {
		return nil
	}
	l.loadData()
	return nil
}

func (l *BoundValue[T]) LoadData(raw json.RawMessage) error {
	if len(raw) == 0 {
		return nil
	}
	data, err := l.Block.ValueFromDB(raw)
	if err != nil {
		return err
	}
	if data == nil {
		return nil
	}
	l.V = data.(*BoundValue[T]).V
	return nil
}

type ListBlockValue = BoundValue[[]*ListBlockData]
type StreamBlockValue = BoundValue[[]*StreamBlockData]
type StructBlockValue = BoundValue[map[string]interface{}]

func newListBlockValue(block Block, data []*ListBlockData) *ListBlockValue {
	return &ListBlockValue{
		Block: block,
		V:     data,
	}
}

func newStreamBlockValue(block Block, data []*StreamBlockData) *StreamBlockValue {
	return &StreamBlockValue{
		Block: block,
		V:     data,
	}
}

func newStructBlockValue(block Block, data map[string]interface{}) *StructBlockValue {
	return &StructBlockValue{
		Block: block,
		V:     data,
	}
}
