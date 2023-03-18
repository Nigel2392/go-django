package modelutils

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Nigel2392/go-django/core/models"
	"github.com/google/uuid"
)

type FromStringer interface {
	FromString(string) error
}

type BaseField interface {
	fmt.Stringer
	driver.Valuer
	sql.Scanner
	FromStringer
}

type Slice[T any] []T

func (s *Slice[T]) Scan(value interface{}) error {
	v, ok := value.(string)
	if !ok {
		return fmt.Errorf("type assertion to string failed")
	}
	var jsonDecoded, err = decodeJson[Slice[T]](v)
	if err != nil {
		return err
	}
	if s == nil {
		*s = make(Slice[T], 0)
	}
	if jsonDecoded == nil {
		return nil
	}
	*s = *jsonDecoded
	return nil
}

func (s Slice[T]) Value() (driver.Value, error) {
	return encodeJson(s)
}

func (s Slice[T]) String() string {
	var b strings.Builder
	for i, v := range s {
		b.WriteString(fmt.Sprintf("%v", v))
		if i < len(s)-1 {
			b.WriteString(";")
		}
	}
	return b.String()
}

func (s *Slice[T]) FromString(str string) error {
	var sl = strings.Split(str, ";")
	var l = len(sl)
	var r = make(Slice[T], l)
	for i, v := range sl {
		newv, err := Convert[T](v)
		if err != nil {
			return err
		}
		if newv == nil {
			continue
		}
		r[i] = newv.(T)
	}
	*s = r
	return nil

}

type Field[T any] struct {
	Val T
}

func (f Field[T]) NewField(val ...T) Field[T] {
	if len(val) > 0 {
		return Field[T]{val[0]}
	}
	return Field[T]{}
}

func (f Field[T]) String() string {
	return fmt.Sprintf("%v", f.Val)
}

func (f *Field[T]) Scan(value interface{}) error {
	switch v := value.(type) {
	case string:
		return f.FromString(v)
	case []byte:
		return f.FromString(string(v))
	case models.Model[models.DefaultIDField],
		models.Model[uuid.UUID]:
		f.Val = v.(T)
		return nil
	case models.DefaultIDField,
		uuid.UUID:
		f.Val = v.(T)
		return nil
	case int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64,
		float32, float64:
		f.Val = v.(T)
		return nil
	}
	return fmt.Errorf("type assertion to string failed")
}

func (f Field[T]) Value() (driver.Value, error) {
	switch v := any(f.Val).(type) {
	case string, []byte, int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64,
		float32, float64:
		return v, nil
	case driver.Valuer:
		return v.Value()
	}
	return nil, fmt.Errorf("type assertion to string failed")
}

func (f *Field[T]) FromString(str string) error {
	var v, err = Convert[T](str)
	if err != nil {
		return err
	}
	f.Val = v.(T)
	return nil
}

func encodeJson(v any) (string, error) {
	var jsonEncoded, err = json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(jsonEncoded), nil
}

func decodeJson[T any](v string) (*T, error) {
	var jsonDecoded = new(T)
	if err := json.Unmarshal([]byte(v), &jsonDecoded); err != nil {
		return nil, err
	}
	return jsonDecoded, nil
}
