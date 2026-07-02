package utils

import (
	"sync"
	"time"
)

// cacheEntry 缓存条目包装，记录缓存值及过期时间
type cacheEntry[V any] struct {
	value     V
	expiredAt time.Time
}

// TTLCache 并发安全的泛型 TTL 缓存包装
type TTLCache[V any] struct {
	cache sync.Map
	ttl   time.Duration
}

// NewTTLCache 创建一个 TTLCache 实例，如果 ttl <= 0 代表永不过期
func NewTTLCache[V any](ttl time.Duration) *TTLCache[V] {
	return &TTLCache[V]{
		ttl: ttl,
	}
}

// Get 从缓存中获取值。如果已过期，则从缓存中删除并返回零值与 false
func (c *TTLCache[V]) Get(key string) (V, bool) {
	var zero V
	if val, ok := c.cache.Load(key); ok {
		if entry, ok := val.(cacheEntry[V]); ok {
			if c.ttl <= 0 || time.Now().Before(entry.expiredAt) {
				return entry.value, true
			}
			c.cache.Delete(key)
		}
	}
	return zero, false
}

// Set 将值放入缓存中，根据当前配置的 TTL 计算其过期时间
func (c *TTLCache[V]) Set(key string, value V) {
	var expiredAt time.Time
	if c.ttl > 0 {
		expiredAt = time.Now().Add(c.ttl)
	}
	c.cache.Store(key, cacheEntry[V]{
		value:     value,
		expiredAt: expiredAt,
	})
}

// Delete 从缓存中移除指定键的值
func (c *TTLCache[V]) Delete(key string) {
	c.cache.Delete(key)
}

// SetTTL 动态调整缓存的生存时间 (TTL)
func (c *TTLCache[V]) SetTTL(ttl time.Duration) {
	c.ttl = ttl
}
