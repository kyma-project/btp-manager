package config_test

import (
	"fmt"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	ioprometheusclient "github.com/prometheus/client_model/go"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/event"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/kyma-project/btp-manager/controllers/config"
	"github.com/kyma-project/btp-manager/internal/metrics"
)

const configAppliedMetricName = "btpmanager_custom_config_applied"

func TestHandler(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Config Handler Suite")
}

var _ = Describe("ConfigMetrics", func() {
	var (
		configMetrics *metrics.ConfigMetrics
		handler       *config.Handler
		testRegistry  *prometheus.Registry
		scheme        *runtime.Scheme
	)

	BeforeEach(func() {
		testRegistry = prometheus.NewRegistry()
		configMetrics = metrics.NewConfigMetrics(testRegistry)

		scheme = runtime.NewScheme()
		Expect(clientgoscheme.AddToScheme(scheme)).To(Succeed())

		fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
		handler = config.NewHandler(fakeClient, scheme, configMetrics)
	})

	Context("gauge metric initialization", func() {
		It("should start with gauge value of 0", func() {
			gauge, err := getGaugeMetricFromRegistryByName(testRegistry, configAppliedMetricName)
			Expect(err).NotTo(HaveOccurred())

			Expect(gauge.GetValue()).To(Equal(0.0))
		})
	})

	Context("ConfigMap lifecycle tracking", func() {
		const chartNamespaceKey = "ChartNamespace"
		const kymaNamespace = "kyma-system"
		var configMap *corev1.ConfigMap

		BeforeEach(func() {
			configMap = &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "sap-btp-manager",
					Namespace: kymaNamespace,
				},
				Data: map[string]string{
					chartNamespaceKey: "test-namespace",
				},
			}
		})

		It("should set gauge to 1 when ConfigMap is created", func() {
			predicates := handler.Predicates()

			createEvent := event.CreateEvent{
				Object: configMap,
			}
			result := predicates.CreateFunc(createEvent)

			Expect(result).To(BeTrue(), "predicate should match the ConfigMap")

			gauge, err := getGaugeMetricFromRegistryByName(testRegistry, configAppliedMetricName)
			Expect(err).NotTo(HaveOccurred())

			Expect(gauge.GetValue()).To(Equal(1.0))
		})

		It("should set gauge to 0 when ConfigMap is deleted", func() {
			predicates := handler.Predicates()

			createEvent := event.CreateEvent{
				Object: configMap,
			}
			predicates.CreateFunc(createEvent)

			deleteEvent := event.DeleteEvent{
				Object: configMap,
			}
			result := predicates.DeleteFunc(deleteEvent)

			Expect(result).To(BeTrue(), "predicate should match the ConfigMap")

			gauge, err := getGaugeMetricFromRegistryByName(testRegistry, configAppliedMetricName)
			Expect(err).NotTo(HaveOccurred())

			Expect(gauge.GetValue()).To(Equal(0.0))
		})

		It("should keep gauge at 1 when ConfigMap is updated", func() {
			predicates := handler.Predicates()

			// First create the ConfigMap
			createEvent := event.CreateEvent{
				Object: configMap,
			}
			predicates.CreateFunc(createEvent)

			// Simulate ConfigMap update
			updatedConfigMap := configMap.DeepCopy()
			updatedConfigMap.Data[chartNamespaceKey] = "updated-namespace"

			updateEvent := event.UpdateEvent{
				ObjectOld: configMap,
				ObjectNew: updatedConfigMap,
			}
			result := predicates.UpdateFunc(updateEvent)

			Expect(result).To(BeTrue(), "predicate should match the ConfigMap")

			gauge, err := getGaugeMetricFromRegistryByName(testRegistry, configAppliedMetricName)
			Expect(err).NotTo(HaveOccurred())

			Expect(gauge.GetValue()).To(Equal(1.0))
		})

		It("should not change gauge when non-matching ConfigMap is created", func() {
			predicates := handler.Predicates()

			otherConfigMap := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "other-config",
					Namespace: kymaNamespace,
				},
			}

			createEvent := event.CreateEvent{
				Object: otherConfigMap,
			}
			result := predicates.CreateFunc(createEvent)

			Expect(result).To(BeFalse(), "predicate should not match different ConfigMap")

			gauge, err := getGaugeMetricFromRegistryByName(testRegistry, configAppliedMetricName)
			Expect(err).NotTo(HaveOccurred())

			Expect(gauge.GetValue()).To(Equal(0.0))
		})
	})
})

func getGaugeMetricFromRegistryByName(reg *prometheus.Registry , metricName string) (*ioprometheusclient.Gauge, error) {
	gauge, err := reg.Gather()
	if err != nil {
		return nil, err
	}
	for _, mf := range gauge {
		if mf.GetName() == metricName {
			ms := mf.GetMetric()
			if len(ms) > 0 {
				return ms[0].GetGauge(), nil
			}
		}
	}
	return nil, fmt.Errorf("metric %s not found", metricName)
}
