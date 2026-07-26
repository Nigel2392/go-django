package builder

import (
	"errors"
	"io"
	"strings"
)

var _ Builder = (*BaseBuilder)(nil)

type Builder interface {
	io.Writer
	WriteString(string) error
	WriteRune(rune) error
	AddVar(...interface{})
	AddError(...error)
}

type BaseBuilder struct {
	strings.Builder
	Vars   []any
	Errors []error
}

func (b *BaseBuilder) WriteTo(other Builder) (err error) {
	if b.Builder.Len() > 0 {
		err = other.WriteString(b.String())
	}
	if len(b.Vars) > 0 {
		other.AddVar(b.Vars...)
	}
	if len(b.Errors) > 0 {
		other.AddError(b.Errors...)
	}
	return err
}

func (b *BaseBuilder) GetError() error {
	if len(b.Errors) == 0 {
		return nil
	}

	var errs = make([]error, 0, len(b.Errors))
	for _, err := range b.Errors {
		if err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) == 0 {
		return nil
	}

	if len(errs) == 1 {
		return errs[0]
	}

	return errors.Join(errs...)
}

func (b *BaseBuilder) String() string {
	return b.Builder.String()
}

func (b *BaseBuilder) Write(v []byte) (n int, err error) {
	return b.Builder.Write(v)
}

func (b *BaseBuilder) WriteString(s string) (err error) {
	_, err = b.Builder.WriteString(s)
	return err
}

func (b *BaseBuilder) WriteRune(r rune) (err error) {
	_, err = b.Builder.WriteRune(r)
	return err
}

func (b *BaseBuilder) AddVar(v ...interface{}) {
	b.Vars = append(b.Vars, v...)
}

func (b *BaseBuilder) AddError(err ...error) {
	b.Errors = append(b.Errors, err...)
}
