package tests

import (
	"crypto/md5"
	"crypto/sha1"
	"github.com/Trendyol/sidecache/pkg/server"
	"github.com/minio/highwayhash"
	"os"
	"testing"
)

var keyToHash = []byte("sidecache-user/users?name=john")

func BenchmarkServerHash(b *testing.B) {
	os.Setenv("CACHE_KEY_PREFIX", "test")
	var cacheServer = new(server.CacheServer)
	for n := 0; n < b.N; n++ {
		cacheServer.HashURL("adsfadsdfasdfas")
	}
}

func BenchmarkMD5(b *testing.B) {
	for i := 0; i < b.N; i++ {
		md5.Sum(keyToHash)
	}
}

func BenchmarkSHA1(b *testing.B) {
	for i := 0; i < b.N; i++ {
		sha1.Sum(keyToHash)
	}
}

func BenchmarkHighway(b *testing.B) {
	key := []byte("000102030405060708090A0B0C0D0E0F")
	for i := 0; i < b.N; i++ {
		highwayhash.Sum(keyToHash, key)
	}
}
