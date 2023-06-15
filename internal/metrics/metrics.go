package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

const (
	metricsNamespace = "btpmanager"
)

type Metrics struct{}

func (m *Metrics) registerMetrics() {
	//register new custom metrics here, for example:
	//counter := prometheus.NewCounter(....)
	//metrics.Registry.MustRegister(counter)
}

func NewMetrics() *Metrics {
	metrics := &Metrics{}
	metrics.registerMetrics()
	return metrics
}

func buildMetricName(subsystem, name string) string {
	return prometheus.BuildFQName(metricsNamespace, subsystem, name)
}
