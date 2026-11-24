package djangoapps

import (
	"fmt"
	"reflect"

	"github.com/Nigel2392/versatile/scan"
)

type BaseSettings interface {
	Set(key string, value interface{})
	Get(key string) (any, bool)
}

var CONFIG = config{}

type config struct {
	ApplicationSettings BaseSettings
}

func NewConfig(settings BaseSettings) *config {
	return &config{ApplicationSettings: settings}
}

func (c *config) Get[T any](key string, default_ ...T) T {
	var value, _ = c.GetOK(key, default_...)
	return value
}

func (c *config) GetOK[T any](key string, default_ ...T) (T, bool) {
	if c.ApplicationSettings == nil && len(default_) == 0 {
		return *(new(T)), false
	}

	if len(default_) > 1 {
		panic("too many arguments: len(default_) > 1")
	}

	var defaultValue T
	if len(default_) > 0 {
		defaultValue = default_[0]
	}

	if c.ApplicationSettings == nil {
		return defaultValue, false
	}

	var value, ok = c.ApplicationSettings.Get(key)
	if !ok && len(default_) > 0 {
		return defaultValue, false
	} else if !ok || value == nil {
		return *(new(T)), false
	}

	var dst T
	wasSet, err := scan.Scan(&dst, value, scan.SF_DEFAULT)
	if err != nil {
		panic(fmt.Errorf("error during scanning of value %T into %T: %w", value, &dst, err))
	}

	if !wasSet {
		panic("could not scan into destination value")
	}

	rvd := reflect.ValueOf(dst)
	switch rvd.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Func,
		reflect.Slice, reflect.Chan:
		if rvd.IsNil() {
			dst = defaultValue
		}

	case reflect.Bool:

	default:
		if rvd.IsZero() {
			dst = defaultValue
		}
	}

	return dst, true
}
