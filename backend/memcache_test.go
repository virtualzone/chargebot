package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMemcache_CRUD(t *testing.T) {
	cache := NewInMemoryCache(5 * time.Minute)

	assert.Nil(t, cache.Get("key1"))
	assert.Nil(t, cache.Get("key2"))

	cache.Set("key1", "value1")
	assert.NotNil(t, cache.Get("key1"))
	assert.Equal(t, "value1", cache.Get("key1"))
	assert.Nil(t, cache.Get("key2"))

	cache.Set("key2", "value2")
	assert.NotNil(t, cache.Get("key1"))
	assert.NotNil(t, cache.Get("key2"))
	assert.Equal(t, "value1", cache.Get("key1"))
	assert.Equal(t, "value2", cache.Get("key2"))

	cache.Set("key2", "value2.2")
	assert.Equal(t, "value1", cache.Get("key1"))
	assert.Equal(t, "value2.2", cache.Get("key2"))

	cache.Delete("key1")
	assert.Nil(t, cache.Get("key1"))
	assert.NotNil(t, cache.Get("key2"))
}

func TestMemcache_Expiry(t *testing.T) {
	cache := NewInMemoryCache(2 * time.Second)
	cache.Set("key1", "value1")
	assert.NotNil(t, cache.Get("key1"))
	time.Sleep(2 * time.Second)
	assert.Nil(t, cache.Get("key1"))
}
