package main

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var userAgents = prometheus.NewCounterVec(prometheus.CounterOpts{
	Name: "user_agents",
	Help: "User Agents",
}, []string{"user_agent"})

var requests = prometheus.NewCounterVec(prometheus.CounterOpts{
	Name: "requests",
	Help: "Requests",
}, []string{"method", "req_uri"})

var responses = prometheus.NewCounterVec(prometheus.CounterOpts{
	Name: "responses",
	Help: "Responses",
}, []string{"code", "info", "content_type", "req_uri"})

func init() {
	prometheus.MustRegister(requests)
	prometheus.MustRegister(responses)
	prometheus.MustRegister(userAgents)
	http.Handle("/metrics", promhttp.Handler())
}
