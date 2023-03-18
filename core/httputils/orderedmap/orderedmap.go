package orderedmap

import (
	"database/sql/driver"
	"errors"

	"github.com/Nigel2392/go-django/core/httputils"
	"golang.org/x/exp/slices"
)

// An ordered map
type Map[T comparable, T2 any] struct {
	order  []T
	values map[T]T2
}

// Create a new ordered map
func New[T comparable, T2 any](i ...int) *Map[T, T2] {
	var j = 0
	if len(i) > 0 {
		j = i[0]
	}
	return &Map[T, T2]{
		values: make(map[T]T2, j),
		order:  make([]T, 0, j),
	}
}

// Get a value from the map
func (u *Map[T, T2]) Get(key T) T2 {
	return u.values[key]
}

// Get a value with OK from the map
func (u *Map[T, T2]) GetOK(key T) (T2, bool) {
	v, ok := u.values[key]
	return v, ok
}

// Set a value in the map
func (u *Map[T, T2]) Set(key T, value T2, external ...bool) {
	if u.values == nil {
		u.values = make(map[T]T2)
		u.order = make([]T, 0)
	}
	u.values[key] = value
	u.order = append(u.order, key)
}

// Delete a value from the map
func (u *Map[T, T2]) Delete(key T) {
	delete(u.values, key)
	for i, v := range u.order {
		if v == key {
			if i == len(u.order)-1 {
				u.order = u.order[:i]
				return
			} else if i == 0 {
				u.order = u.order[1:]
				return
			} else {
				u.order = append(u.order[:i], u.order[i+1:]...)
				return
			}
		}
	}
}

// Loop through the value map in order
func (u *Map[T, T2]) InOrder(reverse ...bool) []T2 {
	var ret = make([]T2, 0)
	if len(reverse) > 0 && reverse[0] {
		for i := len(u.order) - 1; i >= 0; i-- {
			ret = append(ret, u.values[u.order[i]])
		}
		return ret
	}
	for _, v := range u.order {
		ret = append(ret, u.values[v])
	}
	return ret
}

// Order the keys in the map
func (u *Map[T, T2]) Sort(f func(i, j T) bool) []T {
	var newOrder = make([]T, len(u.order))
	copy(newOrder, u.order)
	slices.SortFunc(newOrder, f)
	return newOrder
}

// Order the map by keys by a slice of keys
func (u *Map[T, T2]) SortBySlice(v ...T) []T2 {
	var copy = u.Copy()
	var newOrder = make([]T2, len(u.order))
	for i, v := range v {
		if i >= len(u.order) {
			break
		}
		newOrder[i] = copy.Get(v)
		copy.Delete(v)
	}
	var i int
	copy.ForEach(func(k T, v T2) {
		newOrder = append(newOrder, v)
		i++
	})
	return newOrder

}

// Loop through all key, value pairs in order
func (u *Map[T, T2]) ForEach(f func(k T, v T2), reverse ...bool) {
	if len(reverse) > 0 && reverse[0] {
		for i := len(u.order) - 1; i >= 0; i-- {
			f(u.order[i], u.values[u.order[i]])
		}
		return
	}
	for _, orderedKey := range u.order {
		f(orderedKey, u.values[orderedKey])
	}
}

// Get the underlying map of Map[T, T2]
func (u *Map[T, T2]) Map() map[T]T2 {
	return u.values
}

// Length of the underlying map of Map[T, T2]
func (u *Map[T, T2]) Len() int {
	if u == nil {
		return 0
	}
	return len(u.values)
}

// Get the underlying slice of ordered keys
func (u *Map[T, T2]) Keys() []T {
	return u.order
}

func (u *Map[T, T2]) Exists(key T) bool {
	_, ok := u.values[key]
	return ok
}

// Copy the map
func (u *Map[T, T2]) Copy() *Map[T, T2] {
	var newMap = New[T, T2](len(u.order))
	for _, v := range u.order {
		newMap.Set(v, u.values[v])
	}
	return newMap
}

type mapData[T comparable, T2 any] struct {
	Key   T  `json:"key"`
	Value T2 `json:"value"`
}

// Jsonify the map
func (u *Map[T, T2]) Json(indent int) ([]byte, error) {
	var newList = make([]mapData[T, T2], len(u.order))
	for i, v := range u.order {
		newList[i] = mapData[T, T2]{
			Key:   v,
			Value: u.values[v],
		}
	}
	return httputils.Jsonify(newList, indent)
}

// UnJsonify the map
func (u *Map[T, T2]) UnJson(data []byte) error {
	var newList = make([]mapData[T, T2], 0)
	err := httputils.UnJsonify(data, &newList)
	if err != nil {
		return err
	}
	for _, v := range newList {
		u.Set(v.Key, v.Value)
	}
	return nil
}

// Scan implements the Scanner interface.
func (u *Map[T, T2]) Scan(src interface{}) error {
	var srcString string
	switch src := src.(type) {
	case string:
		srcString = src
	case []byte:
		srcString = string(src)
	default:
		//lint:ignore ST1005 This is a generic error message
		return errors.New("Unknown type for Map[T, T2]")
	}

	return u.UnJson([]byte(srcString))
}

// Value implements the driver Valuer interface.
func (u *Map[T, T2]) Value() (driver.Value, error) {
	var b, err = u.Json(0)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
