package tags

import "strings"

type TagMap map[string][]string

func ParseTags(tag string) TagMap {

	// KEY: VALUE, VALUE, VALUE; KEY: VALUE, VALUE, VALUE; KEY: VALUE, VALUE, VALUE

	return ParseWithDelimiter(tag, ";", ":", ",")
}

func ParseWithDelimiter(tag string, delimiterKV string, delimiterK string, delimiterV string) TagMap {
	var m = make(TagMap)
	var parts = strings.Split(tag, delimiterKV)
	for _, part := range parts {
		part = strings.TrimSpace(part)
		var kv = strings.Split(part, delimiterK)
		if len(kv) == 2 {
			var v = strings.Split(kv[1], delimiterV)
			m[kv[0]] = v
		} else if len(kv) == 1 {
			m[kv[0]] = []string{}
		}
	}
	return m
}

func (t TagMap) GetOK(key string) ([]string, bool) {
	var v, ok = t[key]
	if !ok {
		v = make([]string, 0)
	}
	return v, ok
}

func (t TagMap) GetSingle(key string, def ...string) string {
	var v, ok = t[key]
	if !ok || len(v) == 0 {
		if len(def) > 0 {
			return def[0]
		}
		return ""
	}
	return v[0]
}

func (t TagMap) Exists(key string) bool {
	_, ok := t[key]
	return ok
}
