package ctx

import "net/http"

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
