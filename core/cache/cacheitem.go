package cache

import "time"

type CacheItem struct {
	value interface{}
	ttl   time.Duration
}

func NewCacheItem(key string, value interface{}, ttl time.Duration) *CacheItem {
	return &CacheItem{
		value: value,
		ttl:   time.Duration(ttl),
	}
}

func (c *CacheItem) Value() interface{} {
	return c.value
}

func (c *CacheItem) TTL() time.Duration {
	return c.ttl
}
