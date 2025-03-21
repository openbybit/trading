package lru

const (
	shardCount = 32
)

// ShardLRU is a shard LRU
type ShardLRU []*Cache

// NewShardLRU create a new shard LRU
func NewShardLRU(size int) (ShardLRU, error) {
	m := make(ShardLRU, shardCount)
	for i := 0; i < shardCount; i++ {
		cache, err := NewLRU(size)
		if err != nil {
			return nil, err
		}
		m[i] = cache
	}
	return m, nil
}

// Get looks up a key's value from the cache.
func (s ShardLRU) Get(key string) (interface{}, bool) {
	shard := s.getShard(key)
	return shard.Get(key)
}

// Set adds a value to the cache.
func (s ShardLRU) Set(key string, val interface{}) {
	shard := s.getShard(key)
	shard.Add(key, val)
}

// Del deletes a key from the cache.
func (s ShardLRU) Del(key string) {
	shard := s.getShard(key)
	shard.Remove(key)
}

func (s ShardLRU) getShard(key string) *Cache {
	return s[uint(fnv32(key))%uint(shardCount)]
}

func fnv32(key string) uint32 {
	hash := uint32(2166136261)
	const prime32 = uint32(16777619)
	for i := 0; i < len(key); i++ {
		hash *= prime32
		hash ^= uint32(key[i])
	}
	return hash
}
