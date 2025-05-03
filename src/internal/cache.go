package internal

type Cache[K comparable, V any] struct {
	m map[K]V
}

func NewCache[K comparable, V any]() *Cache[K, V] {
	return &Cache[K, V]{
		m: make(map[K]V),
	}
}

func (c *Cache[K, V]) Get(k K) (V, bool) {
	v, ok := c.m[k]
	return v, ok
}

func (c *Cache[K, V]) GetAll() map[K]V {
	return c.m
}

func (c *Cache[K, V]) Set(k K, v V) {
	c.m[k] = v
}
