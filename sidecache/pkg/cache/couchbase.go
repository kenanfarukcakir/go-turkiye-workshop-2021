package cache
//
//import (
//	"github.com/Trendyol/sidecache/pkg/metric"
//	"os"
//	"time"
//
//	"go.uber.org/zap"
//
//	"gopkg.in/couchbase/gocb.v1"
//)
//
//var (
//	FiveMinute = time.Minute * 5
//	lastLoggedTimeStamp time.Time
//)
//
//type CouchbaseRepository struct {
//	bucket  *gocb.Bucket
//	logger  *zap.Logger
//	cluster *gocb.Cluster
//	metrics *metric.Prometheus
//}
//
//func NewCouchbaseRepository(logger *zap.Logger, metrics *metric.Prometheus) *CouchbaseRepository {
//	lastLoggedTimeStamp = time.Now().Add(-FiveMinute)
//	couchbaseHost := os.Getenv("COUCHBASE_HOST")
//	cluster, err := gocb.Connect("couchbase://" + couchbaseHost)
//	if err != nil {
//		logger.Error("Couchbase connection error:", zap.Error(err))
//		return nil
//	}
//
//	err = cluster.Authenticate(gocb.PasswordAuthenticator{
//		Username: os.Getenv("COUCHBASE_USERNAME"),
//		Password: os.Getenv("COUCHBASE_PASSWORD"),
//	})
//
//	if err != nil {
//		logger.Error("Couchbase authentication error:", zap.Error(err))
//		return nil
//	}
//
//	cacheBucket, err := cluster.OpenBucket(os.Getenv("BUCKET_NAME"), "")
//	if err != nil {
//		logger.Error("Couchbase could not open bucket error:", zap.Error(err))
//		return nil
//	}
//	cacheBucket.SetOperationTimeout(100 * time.Millisecond)
//
//	return &CouchbaseRepository{bucket: cacheBucket, logger: logger, cluster: cluster, metrics: metrics}
//}
//
//func (repository *CouchbaseRepository) SetKey(key string, value []byte, ttl int) {
//	_, err := repository.bucket.Upsert(key, value, uint32(ttl))
//	if err != nil {
//		repository.logIfInTime(func() {repository.logger.Warn("Error occurred when Upsert", zap.String("key", key))})
//	}
//}
//
//func (repository *CouchbaseRepository) Get(key string) []byte {
//	var data []byte
//	_, err := repository.bucket.Get(key, &data)
//
//	if err != nil && err != gocb.ErrKeyNotFound {
//		repository.logIfInTime(func() {repository.logger.Warn("Error occurred when Get", zap.String("key", key), zap.Error(err))})
//	}
//
//	return data
//}
//
//func (repository *CouchbaseRepository) Remove(key string) error {
//	_, err := repository.bucket.Remove(key, 0)
//	if err != nil {
//		repository.logIfInTime(func() {repository.logger.Warn("Error occurred when Remove", zap.String("key", key), zap.Error(err))})
//		if err == gocb.ErrKeyNotFound {
//			return nil
//		}
//		return err
//	}
//
//	return nil
//}
//
//func (repository *CouchbaseRepository) Close() {
//	if err := repository.cluster.Close(); err != nil {
//		repository.logger.Error("error while closing couchbase cluster", zap.Error(err))
//	}
//}
//
//func (repository CouchbaseRepository) logIfInTime(logFunc func()) {
//	repository.metrics.CacheWarnCounter.Inc()
//	allowed := time.Since(lastLoggedTimeStamp) > FiveMinute
//	if allowed {
//		logFunc()
//		lastLoggedTimeStamp = time.Now()
//	}
//}
