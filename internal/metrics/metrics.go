package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

type Metrics struct {
	WorkqueueSizeGauge prometheus.Gauge
}

func (m *Metrics) registerMetrics() {
	m.WorkqueueSizeGauge = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "workqueue_size",
			Help: "Size of work queue",
		},
	)
	metrics.Registry.MustRegister(m.WorkqueueSizeGauge)
}

func InitMetrics() *Metrics {
	metrics := &Metrics{}
	metrics.registerMetrics()
	return metrics
}
