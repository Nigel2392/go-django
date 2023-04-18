package cache_test

import (
	"testing"
	"time"

	"github.com/Nigel2392/go-django/core/cache"
)

func TestCache(t *testing.T) {
	var c = cache.NewInMemoryCache(1 * time.Second)
	c.Connect()
	c.Set("test", "test", cache.NoExpiration)
	var item, err = c.Get("test")
	if err != nil {
		t.Error(err)
	}
	if item.Value() != "test" {
		t.Error("item.Value() != \"test\"")
	}
	if item.TTL() != cache.NoExpiration {
		t.Error("item.TTL() != cache.NoExpiration")
	}
	time.Sleep(2 * time.Second)
	_, err = c.Get("test")
	if err != nil {
		t.Error("err != time.Sleep(2 * time.Second)")
	}

	c.Set("test1", "test", int64(time.Duration(1*time.Second)))
	c.Set("test2", "test", int64(time.Duration(1*time.Second)))
	c.Set("test3", "test", int64(time.Duration(1*time.Second)))
	c.Set("test4", "test", int64(time.Duration(1*time.Second)))
	c.Set("test5", "test", int64(time.Duration(1*time.Second)))
	c.Set("test6", "test", int64(time.Duration(1*time.Second)))
	time.Sleep(2 * time.Second)
	_, err = c.Get("test1")
	if err != cache.ErrNotFound {
		t.Error("err != cache.ErrNotFound int64(time.Duration(1*time.Second))")
	}

	if c.(*cache.InMemoryCache).Tree.Len() != 1 {
		t.Error("c.(*cache.InMemoryCache).Tree.Len() != 1: ", c.(*cache.InMemoryCache).Tree.Len())
	}
}
