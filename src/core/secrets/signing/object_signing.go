package signing

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
)

type Encoder interface {
	Encode(v any) error
}

type Decoder interface {
	Decode(v any) error
}

var (
	DefaultEncoder = JSONEncoder
	DefaultDecoder = JSONDecoder
)

func SignObject(ctx context.Context, signer Signer, obj interface{}) (string, error) {
	return SignObjectEncoder(ctx, signer, obj, DefaultEncoder)
}

func UnsignObject(ctx context.Context, signer Signer, signed string, obj interface{}) error {
	return UnsignObjectDecoder(ctx, signer, signed, obj, DefaultDecoder)
}

func SignObjectEncoder(ctx context.Context, signer Signer, obj interface{}, encoder func(w io.Writer) Encoder) (string, error) {
	var buf bytes.Buffer
	e := encoder(&buf)
	if err := e.Encode(obj); err != nil {
		return "", err
	}
	return signer.Sign(ctx, buf.Bytes())
}

func UnsignObjectDecoder(ctx context.Context, signer Signer, signed string, obj interface{}, decoder func(r io.Reader) Decoder) error {
	var data, err = signer.Unsign(ctx, signed)
	if err != nil {
		return err
	}
	var d = decoder(bytes.NewReader(data))
	return d.Decode(obj)
}

func JSONEncoder(w io.Writer) Encoder {
	return json.NewEncoder(w)
}

func JSONDecoder(r io.Reader) Decoder {
	return json.NewDecoder(r)
}
