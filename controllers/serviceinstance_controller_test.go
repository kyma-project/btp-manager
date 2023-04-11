package controllers

import (
	"context"
	"github.com/kyma-project/module-manager/pkg/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Service Instance and Bindings controller", Ordered, func() {

	Describe("Deletion", Focus, func() {

		BeforeAll(func() {
			ChartPath = "../module-chart/chart"
			ResourcesPath = "../module-resources"
			err := createPrereqs()
			Expect(err).To(BeNil())
			Expect(createChartOrResourcesCopyWithoutWebhooks(ChartPath, defaultChartPath)).To(Succeed())
			Expect(createChartOrResourcesCopyWithoutWebhooks(ResourcesPath, defaultResourcesPath)).To(Succeed())
			ChartPath = defaultChartPath
			ResourcesPath = defaultResourcesPath
			secret, err := createCorrectSecretFromYaml()
			Expect(err).To(BeNil())
			Expect(k8sClient.Create(ctx, secret)).To(Succeed())

		})

		AfterAll(func() {
			Expect(removeAllFromPath(defaultChartPath)).To(Succeed())
			Expect(removeAllFromPath(defaultResourcesPath)).To(Succeed())
		})

		BeforeEach(func() {
			ctx = context.Background()
		})

		When("Last Service Instance is removed", func() {
			It("BTP Operator should be removed", func() {
				// GIVEN
				//  - create BTP operator
				btpOperatorResource := createBtpOperator()
				Expect(k8sClient.Create(ctx, btpOperatorResource)).To(Succeed())
				Eventually(updateCh).Should(Receive(matchState(types.StateReady)))

				//  - create Service Instance
				siUnstructured := createResource(instanceGvk, kymaNamespace, instanceName)
				ensureResourceExists(instanceGvk)

				//  - trigger BTP operator deletion
				Expect(k8sClient.Delete(ctx, btpOperatorResource)).To(Succeed())
				Eventually(updateCh).Should(Receive(matchReadyCondition(types.StateDeleting, metav1.ConditionFalse, ServiceInstancesAndBindingsNotCleaned)))

				// WHEN
				Expect(k8sClient.Delete(ctx, siUnstructured)).To(Succeed())

				// THEN
				Eventually(updateCh).Should(Receive(matchDeleted()))
			})
		})

		When("Last Service Binding is removed", func() {
			It("BTP Operator should be removed", func() {
				sbUnstructured := createResource(bindingGvk, kymaNamespace, bindingName)
				ensureResourceExists(bindingGvk)

				btpOperatorResource := createBtpOperator()
				Expect(k8sClient.Create(ctx, btpOperatorResource)).To(Succeed())
				Eventually(updateCh).Should(Receive(matchState(types.StateReady)))
				Expect(k8sClient.Delete(ctx, btpOperatorResource)).To(Succeed())

				Eventually(updateCh).Should(Receive(matchReadyCondition(types.StateDeleting, metav1.ConditionFalse, ServiceInstancesAndBindingsNotCleaned)))

				Expect(k8sClient.Delete(ctx, sbUnstructured)).To(Succeed())

				Eventually(updateCh).Should(Receive(matchDeleted()))
			})
		})
	})

})
