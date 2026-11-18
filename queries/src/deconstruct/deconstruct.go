package deconstruct

import "encoding/json/jsontext"

type Deconstructible interface {
	Deconstruct() (path string, args []any, kwargs map[string]any)
}

type Kwargs map[string]any

func (k Kwargs) Get[T any](name string) (val T) {
	val, _ = k.GetOk[T](name)
	return val
}

func (k Kwargs) GetOk[T any](name string) (val T, ok bool) {
	v, ok := k[name]
	if !ok {
		return val, false
	}

	val, ok = v.(T)
	if !ok {
		return val, false
	}

	return val, true
}

type Deconstructed struct {
	Path   string           `json:"path"`
	Args   []jsontext.Value `json:"args"`
	Kwargs Kwargs           `json:"kwargs"`
}
