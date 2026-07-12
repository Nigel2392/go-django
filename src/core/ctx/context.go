package ctx

import (
	"errors"
	"maps"
	"net/http"
)

func TemplateDictFunc(values ...interface{}) (map[string]interface{}, error) {
	if len(values)%2 != 0 {
		return nil, errors.New("invalid dict call: must have an even number of arguments")
	}
	dict := make(map[string]interface{}, len(values)/2)
	for i := 0; i < len(values); i += 2 {
		key, ok := values[i].(string)
		if !ok {
			return nil, errors.New("dict keys must be strings")
		}
		dict[key] = values[i+1]
	}
	return dict, nil
}

type Setter interface {
	Set(key string, value any)
}

type Getter interface {
	Get(key string) any
}

type Editor interface {
	EditContext(key string, context Context)
}

type Context interface {
	Setter
	Getter
	Data() map[string]any
	Clone(values ...any) (Context, error)
}

type ContextWithRequest interface {
	Context
	Request() *http.Request
}

func NewContext(m map[string]any) Context {
	if m == nil {
		m = make(map[string]any)
	}
	return &context{data: m}
}

type context struct {
	data map[string]any
}

func (c *context) Set(key string, value any) {
	if v, ok := value.(Editor); ok {
		v.EditContext(key, c)
		return
	}
	c.data[key] = value
}

func (c *context) Get(key string) any {
	return c.data[key]
}

func (c *context) Data() map[string]any {
	return c.data
}

func (c *context) Clone(values ...interface{}) (Context, error) {
	var copy = &context{
		data: maps.Clone(c.data),
	}

	var other, err = TemplateDictFunc(values...)
	maps.Copy(copy.data, other)
	return copy, err
}
