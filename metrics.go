package main

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var userAgents = prometheus.NewCounterVec(prometheus.CounterOpts{
	Name: "jasmin_user_agents",
	Help: "HTTP User Agents",
}, []string{"user_agent"})

var requests = prometheus.NewCounterVec(prometheus.CounterOpts{
	Name: "jasmin_requests",
	Help: "HTTP Requests",
}, []string{"method", "req_uri"})

var responses = prometheus.NewCounterVec(prometheus.CounterOpts{
	Name: "jasmin_responses",
	Help: "HTTP Responses",
}, []string{"code", "info", "content_type", "req_uri"})

func init() {
	prometheus.MustRegister(requests)
	prometheus.MustRegister(responses)
	prometheus.MustRegister(userAgents)
	http.Handle("/metrics", promhttp.Handler())
}
