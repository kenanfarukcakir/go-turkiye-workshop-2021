package main

import (
	"fmt"
	"github.com/Trendyol/sidecache/pkg/cache"
	"github.com/Trendyol/sidecache/pkg/server"
	"github.com/valyala/fasthttp"
	"go.uber.org/zap"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var version string

const (
	defaultProxyTimeout          = 5 * time.Second
	defaultMaxConnectionsPerHost = 2048
)

func main() {
	logger, _ := zap.NewProduction()
	logger.Info("Side cache process started...", zap.String("version", version))

	defer logger.Sync()

	couchbaseRepo := cache.NewCouchbaseRepository()

	mainContainerPort := "8080"
	logger.Info("Main container port", zap.String("port", mainContainerPort))

	proxy := &fasthttp.HostClient{
		Addr:                      fmt.Sprintf("127.0.0.1:%s", mainContainerPort),
		MaxIdemponentCallAttempts: 2,
		MaxIdleConnDuration: 2 * time.Second,
		MaxConns:                  defaultMaxConnectionsPerHost,
	}

	cacheServer := server.NewServer(couchbaseRepo, proxy, logger)
	logger.Info("Cache key prefix", zap.String("prefix", cacheServer.CacheKeyPrefix))

	stopChan := make(chan os.Signal)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)

	cacheServer.Start(stopChan)
}
