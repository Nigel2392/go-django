package cache

import "time"

type CacheItem struct {
	key        string
	value      interface{}
	ttl        time.Duration
	keepAround bool
}

func NewCacheItem(key string, value interface{}, ttl int64) *CacheItem {
	return &CacheItem{
		key:        key,
		value:      value,
		ttl:        time.Duration(ttl),
		keepAround: ttl == NoExpiration,
	}
}

func (c *CacheItem) Value() interface{} {
	return c.value
}

func (c *CacheItem) TTL() int64 {
	return int64(c.ttl)
}

func (c *CacheItem) Key() string {
	return c.key
}

func (c *CacheItem) Eq(other *CacheItem) bool {
	return c.key == other.key
}

func (c *CacheItem) Lt(other *CacheItem) bool {
	return c.key < other.key
}

func (c *CacheItem) Gt(other *CacheItem) bool {
	return c.key > other.key
}
