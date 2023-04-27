package modelutils

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
)

type Slice[T any] []T

func (s *Slice[T]) Scan(value interface{}) error {
	var v string
	switch value := value.(type) {
	case string:
		v = value
	case []byte:
		v = string(value)
	default:
		return nil
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
		r[i] = newv
	}
	*s = r
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
