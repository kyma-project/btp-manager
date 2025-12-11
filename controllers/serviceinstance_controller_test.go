package controllers

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/kyma-project/btp-manager/api/v1alpha1"
	"github.com/kyma-project/btp-manager/controllers/config"
	"github.com/kyma-project/btp-manager/internal/conditions"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Service Instance and Bindings controller", Ordered, func() {

	Describe("Deletion", func() {

		var resourcesPathForProcess, chartPathForProcess string
		var serviceInstanceName = fmt.Sprintf("testing-instance-%s", rand.String(4))
		var serviceBindingName = fmt.Sprintf("testing-binding-%s", rand.String(4))

		BeforeEach(func() {
			config.ChartPath = "../module-chart/chart"
			config.ResourcesPath = "../module-resources"
			err := createPrereqs()
			Expect(err).To(BeNil())
			secret, err := createCorrectSecretFromYaml()
			Expect(err).To(BeNil())
			Expect(k8sClient.Create(ctx, secret)).To(Succeed())

			chartPathForProcess = fmt.Sprintf("%s-%d-%d", defaultChartPath, GinkgoParallelProcess(), rand.Intn(999))
			resourcesPathForProcess = fmt.Sprintf("%s-%d-%d", defaultResourcesPath, GinkgoParallelProcess(), rand.Intn(999))
			err = createChartOrResourcesCopyWithoutWebhooksByConfig(config.ChartPath, chartPathForProcess)
			Expect(err).To(BeNil())
			err = createChartOrResourcesCopyWithoutWebhooksByConfig(config.ResourcesPath, resourcesPathForProcess)
			Expect(err).To(BeNil())
			config.ChartPath = chartPathForProcess
			config.ResourcesPath = resourcesPathForProcess

			ctx = context.Background()
		})

		AfterEach(func() {
			Expect(os.RemoveAll(chartPathForProcess)).To(Succeed())
			Expect(os.RemoveAll(resourcesPathForProcess)).To(Succeed())

			config.ChartPath = defaultChartPath
			config.ResourcesPath = defaultResourcesPath

			deleteSecret := &corev1.Secret{}
			Expect(k8sClient.Get(ctx, client.ObjectKey{Namespace: kymaNamespace, Name: config.SecretName}, deleteSecret)).To(Succeed())
			Expect(k8sClient.Delete(ctx, deleteSecret)).To(Succeed())
		})

		When("Last Service Instance is removed", func() {
			It("BTP Operator should be removed", func() {
				// GIVEN
				//  - create BTP operator
				btpOperatorResource := createDefaultBtpOperator()
				Expect(k8sClient.Create(ctx, btpOperatorResource)).To(Succeed())
				Eventually(updateCh).Should(Receive(matchState(v1alpha1.StateReady)))

				//  - create Service Instance
				siUnstructured := createResource(instanceGvk, kymaNamespace, serviceInstanceName)
				ensureResourceExists(instanceGvk)

				//  - trigger BTP operator deletion
				Expect(k8sClient.Delete(ctx, btpOperatorResource)).To(Succeed())
				Eventually(updateCh).Should(Receive(matchReadyCondition(v1alpha1.StateWarning, metav1.ConditionFalse, conditions.ServiceInstancesAndBindingsNotCleaned)))

				// WHEN
				Expect(k8sClient.Delete(ctx, siUnstructured)).To(Succeed())

				// THEN
				Eventually(updateCh).Should(Receive(matchDeleted()))
			})
		})

		When("Last Service Binding is removed", func() {
			It("BTP Operator should be removed", func() {
				// GIVEN
				//  - create BTP operator
				btpOperatorResource := createDefaultBtpOperator()
				Expect(k8sClient.Create(ctx, btpOperatorResource)).To(Succeed())
				Eventually(updateCh).Should(Receive(matchState(v1alpha1.StateReady)))
				//  - create Service Binding
				sbUnstructured := createResource(bindingGvk, kymaNamespace, serviceBindingName)
				ensureResourceExists(bindingGvk)

				//  - trigger BTP operator deletion
				Expect(k8sClient.Delete(ctx, btpOperatorResource)).To(Succeed())
				Eventually(updateCh).Should(Receive(matchReadyCondition(v1alpha1.StateWarning, metav1.ConditionFalse, conditions.ServiceInstancesAndBindingsNotCleaned)))

				// WHEN
				Expect(k8sClient.Delete(ctx, sbUnstructured)).To(Succeed())

				// THEN
				Eventually(updateCh).Should(Receive(matchDeleted()))
			})
		})
		// NOTE: Without this sleep, the tests are flaky when run sequentially.
		// However, if only a single test from this file is run (without the second one), it always passes. Even if the sleep is removed.
		AfterEach(func() {
			time.Sleep(1 * time.Second)
		})
	})

})
