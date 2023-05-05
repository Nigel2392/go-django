package cache

import (
	"errors"
	"reflect"
	"time"

	"github.com/Nigel2392/netcache/src/cache"
	"github.com/Nigel2392/netcache/src/client"
)

const (
	ShortExpiration   = time.Second * 10
	MedExpiration     = time.Second * 60
	LongExpiration    = time.Hour
	DefaultExpiration = MedExpiration
)

func init() {
	var _ = client.Cache(&InMemoryCache{})
}

type InMemoryCache struct {
	cache      *cache.MemoryCache[any]
	tickerTime time.Duration
}

func NewInMemoryCache(cleanInterval time.Duration) *InMemoryCache {
	var c = &InMemoryCache{
		cache:      cache.NewGenericMemoryCache[any](),
		tickerTime: cleanInterval,
	}
	return c
}

// Connect to the cache.
func (c *InMemoryCache) Connect() error {
	c.cache.Run(c.tickerTime)
	return nil
}

// Get an item from the cache.
func (c *InMemoryCache) Get(key string, dst any) (client.Item, error) {
	var item, ttl, err = c.cache.Get(key)
	if err != nil {
		return nil, err
	}

	if dst != nil {
		err = reflectSetDst(dst, item)
		if err != nil {
			return nil, err
		}
	}

	return NewCacheItem(key, item, ttl), nil
}

func reflectSetDst(dst, src any) error {
	var dstValue = reflect.ValueOf(dst)
	var srcValue = reflect.ValueOf(src)

	if dstValue.Kind() != reflect.Ptr {
		return errors.New("dst must be a pointer")
	}

	if dstValue.IsNil() {
		return errors.New("dst must not be nil")
	}

	if dstValue.Elem().Kind() != srcValue.Kind() {
		return errors.New("dst and src must be the same type")
	}

	if srcValue.Kind() == reflect.Ptr {
		if srcValue.IsNil() {
			return errors.New("src must not be nil")
		}
	}

	dstValue.Elem().Set(srcValue)
	return nil
}

// Set an item in the cache.
func (c *InMemoryCache) Set(key string, value any, ttl time.Duration) error {
	var _, err = c.cache.Set(key, value, ttl)
	return err
}

// Delete an item from the cache.
func (c *InMemoryCache) Delete(key string) error {
	var _, err = c.cache.Delete(key)
	return err
}

// Clear the cache.
func (c *InMemoryCache) Clear() error {
	c.cache.Clear()
	return nil
}

// Check if the cache has an item.
func (c *InMemoryCache) Has(key string) (bool, error) {
	var _, has = c.cache.Has(key)
	return has, nil
}

// Keys returns all keys in the cache.
func (c *InMemoryCache) Keys() ([]string, error) {
	var keys = c.cache.Keys()
	return keys, nil
}
