package controllers

import (
	"context"
	"time"

	"github.com/kyma-project/btp-manager/api/v1alpha1"
	"github.com/kyma-project/btp-manager/internal/conditions"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	firstBtpOperator  = "f"
	secondBtpOperator = "s"
)

var _ = Describe("BTP Operator CR leader replacement", Label("debug"), func() {
	It("2 CRs(1-Ready,2-Error) & SI/SB exists, first is under deletion, and finalizer is removed ", func() {
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

		btpOperatorInErrorExists := false
		btpOperatorInReadyExists := false
		for _, btpOperator := range btpOperators.Items {
			if btpOperator.Status.State == v1alpha1.StateError && btpOperator.Status.Conditions[0].Reason == string(conditions.OlderCRExists) {
				btpOperatorInErrorExists = true
				continue
			}

			if btpOperator.Status.State == v1alpha1.StateReady {
				btpOperatorInReadyExists = true
				continue
			}
		}
		Expect(btpOperatorInErrorExists).To(BeTrue())
		Expect(btpOperatorInReadyExists).To(BeTrue())

		Expect(k8sClient.Delete(ctx, btpOperator1)).To(Succeed())
		Eventually(updateCh).Should(Receive(matchReadyCondition(v1alpha1.StateDeleting, metav1.ConditionFalse, conditions.ServiceInstancesAndBindingsNotCleaned)))
		time.Sleep(time.Second * 5)
		Expect(k8sClient.Get(ctx, client.ObjectKey{Namespace: btpOperator1.GetNamespace(), Name: btpOperator1.GetName()}, btpOperator1)).To(Succeed())
		btpOperator1.SetFinalizers([]string{})
		Eventually(func() error { return k8sClient.Update(ctx, btpOperator1) }).WithTimeout(k8sOpsTimeout).WithPolling(k8sOpsPollingInterval).Should(Succeed())
		time.Sleep(time.Second * 5)
		err = k8sClient.List(ctx, btpOperators)
		Expect(err).To(BeNil())
		Expect(len(btpOperators.Items)).To(BeEquivalentTo(1))

		Expect(btpOperators.Items[0].Name).To(BeEquivalentTo(secondBtpOperator))
		Expect(btpOperators.Items[0].Status.State).To(BeEquivalentTo(v1alpha1.StateReady))
		Expect(btpOperators.Items[0].Status.Conditions[0].Reason).To(BeEquivalentTo(conditions.ReconcileSucceeded))

	})
})
