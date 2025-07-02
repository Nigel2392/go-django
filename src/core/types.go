package core

type Settings interface {
	Set(key string, value interface{})
	Get(key string) (any, bool)
}
