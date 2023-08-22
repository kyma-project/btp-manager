package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

const (
	metricsNamespace = "btpmanager"
)

type Metrics struct {
	certsRegenerationsCounter prometheus.Counter
}

func (m *Metrics) registerMetrics() {
	//register new custom metrics here, for example:
	//counter := prometheus.NewCounter(....)
	//metrics.Registry.MustRegister(counter)
	certRegenCounter := prometheus.NewCounter(prometheus.CounterOpts{
		Name: buildMetricName("", "certs_regenerations_total"),
		Help: "Total number of certs regenerations",
	})
	m.certsRegenerationsCounter = certRegenCounter
	metrics.Registry.MustRegister(certRegenCounter)
}

func (m *Metrics) IncreaseCertsRegenerationsCounter() {
	m.certsRegenerationsCounter.Inc()
}

func NewMetrics() *Metrics {
	metrics := &Metrics{}
	metrics.registerMetrics()
	return metrics
}

func buildMetricName(subsystem, name string) string {
	return prometheus.BuildFQName(metricsNamespace, subsystem, name)
}
