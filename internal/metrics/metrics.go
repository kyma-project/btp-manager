package metrics

import (
	"github.com/kyma-project/btp-manager/internal/conditions"
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
	"strings"
)

const (
	metricsNamespace = "btpmanager"
)

type Metrics struct {
	ReasonCounters map[conditions.Reason]prometheus.Counter
}

func (m *Metrics) registerMetrics() {
	m.ReasonCounters = make(map[conditions.Reason]prometheus.Counter, len(conditions.Reasons))
	for reason, metadata := range conditions.Reasons {
		counter := prometheus.NewCounter(prometheus.CounterOpts{
			Name:        prometheus.BuildFQName(metricsNamespace, "", strings.ToLower(string(reason))),
			ConstLabels: prometheus.Labels{"state": string(metadata.State)},
		})
		m.ReasonCounters[reason] = counter
		metrics.Registry.MustRegister(counter)
	}
}

func NewMetrics() *Metrics {
	metrics := &Metrics{}
	metrics.registerMetrics()
	return metrics
}

func (m *Metrics) IncreaseReasonCounter(reason conditions.Reason) {
	counter, found := m.ReasonCounters[reason]
	if found {
		counter.Inc()
	}
}
