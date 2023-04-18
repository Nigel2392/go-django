package cache

import (
	"fmt"
	"time"

	"github.com/Nigel2392/go-django/core/cache/binarytree"
)

const (
	LongExpiration    int64 = 86400000000000 // 1 day
	DefaultExpiration int64 = 600000000000   // 10 minutes
	ShortExpiration   int64 = 30000000000    // 5 minutes
	NoExpiration      int64 = 1<<63 - 1
)

type InMemoryCache struct {
	Tree       *binarytree.BST[*CacheItem]
	ticker     *time.Ticker
	tickerTime time.Duration
}

func NewInMemoryCache(tickerTime time.Duration) Cache {
	return &InMemoryCache{
		Tree:       &binarytree.BST[*CacheItem]{},
		ticker:     time.NewTicker(tickerTime),
		tickerTime: tickerTime,
	}
}

func (c *InMemoryCache) Connect() error {
	go func() {
		for range c.ticker.C {
			c.Tree.Traverse(func(item *CacheItem) {
				item.ttl -= c.tickerTime
			})
			c.Tree.DeleteIf(func(item *CacheItem) bool {
				fmt.Println(item.key, item.ttl <= 0 && !item.keepAround)
				return item.ttl <= 0 && !item.keepAround
			})
		}
	}()
	return nil
}

func (c *InMemoryCache) Get(key string) (Item, error) {
	var item, found = c.Tree.Search(&CacheItem{key: key})
	if !found {
		return nil, ErrNotFound
	}
	return item, nil
}

func (c *InMemoryCache) Set(key string, value any, ttl int64) error {
	c.Tree.Insert(&CacheItem{
		key:        key,
		value:      value,
		ttl:        time.Duration(ttl),
		keepAround: ttl == NoExpiration,
	})
	return nil
}

func (c *InMemoryCache) Delete(key string) error {
	c.Tree.Delete(&CacheItem{key: key})
	return nil
}

func (c *InMemoryCache) Clear() error {
	c.Tree.Clear()
	return nil
}

func (c *InMemoryCache) Has(key string) bool {
	var _, found = c.Tree.Search(&CacheItem{key: key})
	return found
}
