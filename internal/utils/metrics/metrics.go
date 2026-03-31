package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var DbMetricsIns = NewDbMetrics()

type DbMetrics struct {
	DbSum    *prometheus.HistogramVec
	CacheSum *prometheus.HistogramVec
	ApiSum   *prometheus.HistogramVec
}

func NewDbMetrics() *DbMetrics {
	this := &DbMetrics{}
	this.DbSum = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name: "db_latency",
		Help: "Latency of DB",
	}, []string{"query"})
	this.CacheSum = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name: "cache_latency",
		Help: "Latency of Cache",
	}, []string{"query"})
	this.ApiSum = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name: "api_latency",
		Help: "Latency of API",
	}, []string{"query"})
	return this
}
