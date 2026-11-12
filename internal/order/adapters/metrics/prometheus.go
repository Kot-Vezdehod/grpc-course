package metrics

import "github.com/prometheus/client_golang/prometheus"

type PrometheusMetrics struct {
	ordersCreated prometheus.Counter
}

func NewOrderMetrics(counter prometheus.Counter) *PrometheusMetrics {
	return &PrometheusMetrics{
		ordersCreated: counter,
	}
}

func (p *PrometheusMetrics) OrderCreated() {
	p.ordersCreated.Inc()
}
