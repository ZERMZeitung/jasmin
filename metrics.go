package main

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var queries = prometheus.NewCounterVec(prometheus.CounterOpts{
	Name: "queries",
	Help: "Queries",
}, []string{"code", "info", "content_type", "host", "method", "req_uri", "user_agent"})

func init() {
	prometheus.MustRegister(queries)
	http.Handle("/metrics", promhttp.Handler())
}
