package deconstruct

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
)

type Reconstructor func(args []jsontext.Value, kwargs Kwargs, opts ...json.Options) (any, error)

type ConstructorRegistry struct {
	constructors map[string]Reconstructor
	unmarshalers []*json.Unmarshalers
}

var Constructors = ConstructorRegistry{
	constructors: make(map[string]Reconstructor),
}

func (c *ConstructorRegistry) Register(path string, fn Reconstructor) {
	c.constructors[path] = fn
}

func (c *ConstructorRegistry) unmarshalAny(dec *jsontext.Decoder) (any, error) {
	if dec.PeekKind() == 'n' {
		_, err := dec.ReadToken()
		return nil, err
	}

	var p Deconstructed
	// Inherits dec's full parent json.Options (caller unmarshalers, duplicate key rules, etc.)
	if err := json.UnmarshalDecode(dec, &p); err != nil {
		return nil, err
	}

	fn, ok := c.constructors[p.Path]
	if !ok {
		return nil, fmt.Errorf("unregistered deconstruct path: %q", p.Path)
	}

	// Pass the parent decoder's exact options down to the reconstructor
	return fn(p.Args, p.Kwargs, dec.Options())
}

func (c *ConstructorRegistry) RegisterDestructibleType[T any]() {
	c.unmarshalers = append(c.unmarshalers, json.UnmarshalFromFunc(func(dec *jsontext.Decoder, dst *T) error {
		val, err := c.unmarshalAny(dec)
		if err != nil {
			return err
		}
		if val == nil {
			var zero T
			*dst = zero
			return nil
		}
		typed, ok := val.(T)
		if !ok {
			var zero T
			return fmt.Errorf("reconstructed %T cannot be assigned to %T", val, zero)
		}
		*dst = typed
		return nil
	}))
}

func (c *ConstructorRegistry) Unmarshal(data []byte, dst any, opts ...json.Options) (err error) {
	return json.Unmarshal(data, dst, append([]json.Options{c.JSONOptions()}, opts...)...)
}

func (c *ConstructorRegistry) JSONOptions() json.Options {
	allUnmarshalers := make([]*json.Unmarshalers, 0, 1+len(c.unmarshalers))
	allUnmarshalers = append(allUnmarshalers, json.UnmarshalFromFunc(c.unmarshalValue))
	allUnmarshalers = append(allUnmarshalers, c.unmarshalers...)
	return json.JoinOptions(
		json.WithMarshalers(json.MarshalToFunc(c.marshalValue)),
		json.WithUnmarshalers(json.JoinUnmarshalers(allUnmarshalers...)),
	)
}

func (c *ConstructorRegistry) marshalValue(enc *jsontext.Encoder, d Deconstructible) error {
	path, args, kwargs := d.Deconstruct()

	rawArgs := make([]jsontext.Value, len(args))
	for i, a := range args {
		b, err := json.Marshal(a, enc.Options())
		if err != nil {
			return err
		}
		rawArgs[i] = jsontext.Value(b)
	}

	return json.MarshalEncode(enc, Deconstructed{
		Path:   path,
		Args:   rawArgs,
		Kwargs: kwargs,
	})
}

func (c *ConstructorRegistry) unmarshalValue(dec *jsontext.Decoder, dst *Deconstructible) error {
	if dec.PeekKind() == 'n' {
		_, err := dec.ReadToken()
		*dst = nil
		return err
	}

	var p Deconstructed
	if err := json.UnmarshalDecode(dec, &p); err != nil {
		return err
	}
	fn, ok := c.constructors[p.Path]
	if !ok {
		return fmt.Errorf("unregistered deconstruct path: %q", p.Path)
	}
	val, err := fn(p.Args, p.Kwargs, dec.Options())
	if err != nil {
		return err
	}
	if val == nil {
		*dst = nil
		return nil
	}
	d, ok := val.(Deconstructible)
	if !ok {
		return fmt.Errorf("reconstructor for %q returned non-Deconstructible %T", p.Path, val)
	}
	*dst = d
	return nil
}
