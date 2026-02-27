package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	metricsNamespace = "btpmanager"
)

func buildMetricName(subsystem, name string) string {
	return prometheus.BuildFQName(metricsNamespace, subsystem, name)
}

type WebhookMetrics struct {
	certsRegenerationCounter prometheus.Counter
}

func NewWebhookMetrics(r prometheus.Registerer) *WebhookMetrics {
	certRegenCounter := promauto.With(r).NewCounter(prometheus.CounterOpts{
		Name: buildMetricName("", "certs_regenerations_total"),
		Help: "Total number of certs regenerations",
	})
	m := &WebhookMetrics{
		certsRegenerationCounter: certRegenCounter,
	}
	return m
}

func (m *WebhookMetrics) IncrementCertsRegenerationCounter() {
	m.certsRegenerationCounter.Inc()
}
