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

type ConfigMetrics struct {
	configMapAppliedGauge prometheus.Gauge
}

func NewConfigMetrics(r prometheus.Registerer) *ConfigMetrics {
	gauge := promauto.With(r).NewGauge(prometheus.GaugeOpts{
		Name: buildMetricName("", "custom_config_applied"),
		Help: "Indicates if the custom configuration ConfigMap is applied (1) or not (0)",
	})
	gauge.Set(0)

	m := &ConfigMetrics{
		configMapAppliedGauge: gauge,
	}
	return m
}

func (m *ConfigMetrics) ConfigMapApplied() {
	m.configMapAppliedGauge.Set(1)
}

func (m *ConfigMetrics) ConfigMapNotApplied() {
	m.configMapAppliedGauge.Set(0)
}
