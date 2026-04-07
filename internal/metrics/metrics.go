package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
	CacheHits = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "caches_hits",
		Help: "Total number of cache hits."},
	)
	CacheMisses = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "caches_misses",
		Help: "Total number of cache misses.",
	})
)

func Init() {
	prometheus.MustRegister(CacheHits, CacheMisses)
}
