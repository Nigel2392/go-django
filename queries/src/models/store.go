package models

import (
	"fmt"
	"strings"
)

type MapDataStore map[string]interface{}

func (m MapDataStore) String() string {
	var sb strings.Builder
	sb.WriteString("[")
	var i = 0
	for k, v := range m {
		if i > 0 {
			sb.WriteString(", ")
		}
		fmt.Fprintf(&sb, "%q: %v", k, v)
		i++
	}
	sb.WriteString("]")
	return sb.String()
}

func (m MapDataStore) HasValue(key string) bool {
	_, ok := m[key]
	return ok
}

func (m MapDataStore) SetValue(key string, value any) error {
	m[key] = value
	return nil
}

func (m MapDataStore) GetValue(key string) (any, bool) {
	if v, ok := m[key]; ok {
		return v, true
	}
	return nil, false
}

func (m MapDataStore) DeleteValue(key string) error {
	delete(m, key)
	return nil
}
