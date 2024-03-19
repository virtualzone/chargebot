package main

import (
	"sync"
	"time"
)

type InMemoryCache struct {
	mutex       sync.Mutex
	store       map[string]*InMemoryCacheEntry
	lifetime    time.Duration
	cleanTicker *time.Ticker
}

type InMemoryCacheEntry struct {
	expiry int64
	value  interface{}
}

func NewInMemoryCache(lifetime time.Duration) *InMemoryCache {
	c := &InMemoryCache{
		store:       make(map[string]*InMemoryCacheEntry),
		lifetime:    lifetime,
		cleanTicker: time.NewTicker(time.Minute * 5),
	}
	go func() {
		for {
			c.cleanup()
			<-c.cleanTicker.C
		}
	}()
	return c
}

func (c *InMemoryCache) Set(key string, value interface{}) {
	entry := &InMemoryCacheEntry{
		expiry: time.Now().UTC().Add(c.lifetime).Unix(),
		value:  value,
	}
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.store[key] = entry
}

func (c *InMemoryCache) Get(key string) interface{} {
	val, ok := c.store[key]
	if !ok {
		return nil
	}
	if val.expiry <= time.Now().UTC().Unix() {
		c.mutex.Lock()
		defer c.mutex.Unlock()
		return nil
	}
	return val.value
}

func (c *InMemoryCache) Delete(key string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	delete(c.store, key)
}

func (c *InMemoryCache) cleanup() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	now := time.Now().UTC().Unix()
	for k, e := range c.store {
		if e.expiry <= now {
			delete(c.store, k)
		}
	}
}
