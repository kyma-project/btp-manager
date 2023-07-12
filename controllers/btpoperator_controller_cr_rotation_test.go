package controllers

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/kyma-project/btp-manager/api/v1alpha1"
	"github.com/kyma-project/btp-manager/internal/conditions"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	firstBtpOperator  = "first"
	secondBtpOperator = "second"
)

var _ = Describe("BTP Operator CR leader replacement", Label("debug"), func() {
	Describe("Two CRs (1st-Ready, 2nd-Error) and SI/SB exist", func() {
		When("First CR is under deletion, and finalizer is removed", func() {
			It("should reconcile second CR as primary", func() {
				var err error
				ctx := context.Background()

				secret, err := createCorrectSecretFromYaml()
				Expect(k8sClient.Create(ctx, secret)).To(Succeed())

				btpOperator1 := createBtpOperator(firstBtpOperator)
				Eventually(func() error { return k8sClient.Create(ctx, btpOperator1) }).WithTimeout(k8sOpsTimeout).WithPolling(k8sOpsPollingInterval).Should(Succeed())
				Eventually(updateCh).Should(Receive(matchState(v1alpha1.StateReady)))

				_ = createResource(instanceGvk, kymaNamespace, instanceName)
				ensureResourceExists(instanceGvk)

				_ = createResource(bindingGvk, kymaNamespace, bindingName)
				ensureResourceExists(bindingGvk)

				btpOperator2 := createBtpOperator(secondBtpOperator)
				Eventually(func() error { return k8sClient.Create(ctx, btpOperator2) }).WithTimeout(k8sOpsTimeout).WithPolling(k8sOpsPollingInterval).Should(Succeed())
				Eventually(updateCh).Should(Receive(matchState(v1alpha1.StateError)))

				btpOperators := &v1alpha1.BtpOperatorList{}
				err = k8sClient.List(ctx, btpOperators)
				Expect(err).To(BeNil())
				Expect(len(btpOperators.Items)).To(BeEquivalentTo(2))

				btpOperatorWithReadyState, btpOperatorWithErrorState := &v1alpha1.BtpOperator{}, &v1alpha1.BtpOperator{}
				Eventually(func() (bool, error) {
					err := k8sClient.Get(ctx, client.ObjectKey{Namespace: defaultNamespace, Name: firstBtpOperator}, btpOperatorWithReadyState)
					return btpOperatorWithReadyState.Status.State == v1alpha1.StateReady, err
				}).WithTimeout(k8sOpsTimeout).WithPolling(k8sOpsPollingInterval).Should(BeTrue())
				Eventually(func() (bool, error) {
					err := k8sClient.Get(ctx, client.ObjectKey{Namespace: defaultNamespace, Name: secondBtpOperator}, btpOperatorWithErrorState)
					return btpOperatorWithErrorState.Status.State == v1alpha1.StateError && btpOperatorWithErrorState.Status.Conditions[0].Reason == string(conditions.OlderCRExists), err
				}).WithTimeout(k8sOpsTimeout).WithPolling(k8sOpsPollingInterval).Should(BeTrue())

				Expect(k8sClient.Delete(ctx, btpOperator1)).To(Succeed())
				Eventually(updateCh).Should(Receive(matchReadyCondition(v1alpha1.StateDeleting, metav1.ConditionFalse, conditions.ServiceInstancesAndBindingsNotCleaned)))
				Expect(k8sClient.Get(ctx, client.ObjectKey{Namespace: btpOperator1.GetNamespace(), Name: btpOperator1.GetName()}, btpOperator1)).To(Succeed())
				btpOperator1.SetFinalizers([]string{})
				Expect(k8sClient.Update(ctx, btpOperator1)).To(Succeed())
				Eventually(updateCh).Should(Receive(matchDeleted()))

				Expect(k8sClient.List(ctx, btpOperators)).To(Succeed())
				Expect(len(btpOperators.Items)).To(BeEquivalentTo(1))
				Expect(btpOperators.Items[0].Name).To(BeEquivalentTo(secondBtpOperator))

				Eventually(func() (bool, error) {
					err := k8sClient.Get(ctx, client.ObjectKey{Namespace: defaultNamespace, Name: secondBtpOperator}, btpOperator2)
					return btpOperator2.Status.State == v1alpha1.StateReady, err
				}).WithTimeout(k8sOpsTimeout).WithPolling(k8sOpsPollingInterval).Should(BeTrue())
				Expect(btpOperator2.Status.Conditions[0].Reason).To(BeEquivalentTo(conditions.ReconcileSucceeded))
			})
		})
	})
})
