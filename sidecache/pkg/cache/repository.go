package cache

type CacheRepository interface {
	SetKey(key string, value []byte)
	Get(key string) []byte
	Remove(key string) error
}
