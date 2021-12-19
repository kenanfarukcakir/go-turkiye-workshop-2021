package metric

import (
	"github.com/prometheus/client_golang/prometheus"
	"strings"
)

var (
	gauge = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "sidecache",
			Name:      "cache_hit",
			Help:      "This is cache hit metric",
		})

	cacheHitCounter = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: "sidecache",
			Name:      "cache_hit_counter",
			Help:      "Cache hit count",
		})

	totalRequestCounter = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: "sidecache",
			Name:      "all_request_hit_counter",
			Help:      "All request hit counter",
		})

	purgeRequestCounter = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: "sidecache",
			Name:      "purge_request_counter",
			Help:      "Purge request counter",
		})

	purgeSuccessCounter = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: "sidecache",
			Name:      "purge_success_counter",
			Help:      "Purge success counter",
		})

	cacheErrorCounter = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: "sidecache",
			Name:      "cache_error_counter",
			Help:      "Cache error counter",
		})

	cacheWarnCounter = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: "sidecache",
			Name:      "cache_warn_counter",
			Help:      "Cache warn counter",
		})

	proxyErrorCounter = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: "sidecache",
			Name:      "proxy_error_counter",
			Help:      "Proxy error counter",
		})

	buildInfoGaugeVec = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "sidecache_admission_build_info",
			Help: "Build info for sidecache admission webhook",
		}, []string{"version"})
)

type Prometheus struct {
	CacheHitCounter     prometheus.Counter
	TotalRequestCounter prometheus.Counter
	PurgeRequestCounter prometheus.Counter
	PurgeSuccessCounter prometheus.Counter
	CacheErrorCounter prometheus.Counter
	CacheWarnCounter prometheus.Counter
	ProxyErrorCounter prometheus.Counter
}

func NewPrometheusClient() *Prometheus {
	prometheus.MustRegister(cacheHitCounter,
		totalRequestCounter,
		purgeRequestCounter,
		purgeSuccessCounter,
		buildInfoGaugeVec,
		cacheErrorCounter,
		cacheWarnCounter,
		proxyErrorCounter)

	return &Prometheus{
		TotalRequestCounter: totalRequestCounter,
		CacheHitCounter:     cacheHitCounter,
		PurgeRequestCounter: purgeRequestCounter,
		PurgeSuccessCounter: purgeSuccessCounter,
		CacheErrorCounter:   cacheErrorCounter,
		CacheWarnCounter:    cacheWarnCounter,
		ProxyErrorCounter: proxyErrorCounter,
	}
}

func BuildInfo(admission string) {
	isNotEmptyAdmissionVersion := len(strings.TrimSpace(admission)) > 0

	if isNotEmptyAdmissionVersion {
		buildInfoGaugeVec.WithLabelValues(admission)
	}
}
