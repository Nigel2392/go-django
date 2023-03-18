package tags

import "strings"

type TagMap map[string]string

func ParseTags(tag string) TagMap {
	var m = make(TagMap)
	var parts = strings.Split(tag, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		var kv = strings.Split(part, ":")
		if len(kv) == 2 {
			m[kv[0]] = kv[1]
		} else if len(kv) == 1 {
			m[kv[0]] = ""
		}
	}
	return m
}

func (t TagMap) Get(key string, def ...string) string {
	var v, ok = t[key]
	if !ok {
		if len(def) > 0 {
			return def[0]
		}
		return ""
	}
	return v
}

func (t TagMap) Exists(key string) bool {
	_, ok := t[key]
	return ok
}
