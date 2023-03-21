package controllers

import (
	"github.com/kyma-project/module-manager/pkg/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Service Instance and Bindings controller", Ordered, func() {
	Describe("Deleting when other exists", func() {

		When("xxx", func() {
			It("ccc", func() {
				btpOperatorResource := createBtpOperator()
				Expect(k8sClient.Create(ctx, btpOperatorResource)).To(Succeed())
				Eventually(updateCh).Should(Receive(matchState(types.StateReady)))

				siUnstructured := createResource(instanceGvk, kymaNamespace, instanceName)
				ensureResourceExists(instanceGvk)

				sbUnstructured := createResource(bindingGvk, kymaNamespace, bindingName)
				ensureResourceExists(bindingGvk)

				setFinalizers(siUnstructured)
				setFinalizers(sbUnstructured)

				Expect(k8sClient.Delete(ctx, btpOperatorResource)).To(Succeed())

				//Eventually(updateCh).Should(Receive(matchReadyCondition(types.StateDeleting, metav1.ConditionFalse, ServiceInstancesAndBindingsNotCleaned)))

			})
		})

	})
})
