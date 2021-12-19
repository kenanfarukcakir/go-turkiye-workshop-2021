package cache

import (
	"time"
)

var (
	FiveMinute = time.Minute * 5
	lastLoggedTimeStamp time.Time
)

type CouchbaseRepository struct {
	bucket  map[string][]byte
}

func NewCouchbaseRepository() *CouchbaseRepository {
	cacheBucket := make(map[string][]byte)

	return &CouchbaseRepository{bucket: cacheBucket}
}

func (repository *CouchbaseRepository) SetKey(key string, value []byte) {
	repository.bucket[key] = value
}

func (repository *CouchbaseRepository) Get(key string) []byte {
	return repository.bucket[key]
}

func (repository *CouchbaseRepository) Remove(key string) error {
	delete(repository.bucket, key)
	return nil
}
