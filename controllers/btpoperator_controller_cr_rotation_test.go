package controllers

import (
	"context"
	"time"

	"github.com/kyma-project/btp-manager/api/v1alpha1"
	"github.com/kyma-project/btp-manager/internal/conditions"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	firstBtpOperator  = "first"
	secondBtpOperator = "second"
	thirdBtpOperator  = "third"
)

var _ = Describe("BTP Operator CR leader replacement", func() {

	BeforeEach(func() {
		GinkgoWriter.Println("--- PROCESS:", GinkgoParallelProcess(), "---")
		ctx = context.Background()
		secret, err := createCorrectSecretFromYaml()
		Expect(err).To(BeNil())
		Expect(k8sClient.Patch(ctx, secret, client.Apply, client.ForceOwnership, client.FieldOwner(operatorName))).To(Succeed())
	})

	AfterEach(func() {
		btpOperators := &v1alpha1.BtpOperatorList{}
		Expect(k8sClient.List(ctx, btpOperators)).To(Succeed())
		if len(btpOperators.Items) > 0 {
			for _, cr := range btpOperators.Items {
				btpOp := cr.DeepCopy()
				Expect(k8sClient.Delete(ctx, btpOp)).To(Succeed())
				Eventually(updateCh).Should(Receive(matchDeleted()))
			}
		}
		Expect(k8sClient.List(ctx, btpOperators)).To(Succeed())
		Expect(len(btpOperators.Items)).To(BeEquivalentTo(0))
		deleteSecret := &corev1.Secret{}
		Expect(k8sClient.Get(ctx, client.ObjectKey{Namespace: kymaNamespace, Name: SecretName}, deleteSecret)).To(Succeed())
		Expect(k8sClient.Delete(ctx, deleteSecret)).To(Succeed())
	})

	Describe("Three CRs (1st-Ready, 2nd-Error, 3rd-Error) exist", func() {
		When("First CR is deleted", func() {
			It("should set second CR as leader and update third CR status", func() {
				btpOperator1 := createBtpOperator(firstBtpOperator)
				Eventually(func() error { return k8sClient.Create(ctx, btpOperator1) }).WithTimeout(k8sOpsTimeout).WithPolling(k8sOpsPollingInterval).Should(Succeed())
				Eventually(updateCh).Should(Receive(matchState(v1alpha1.StateReady)))

				// required for later CreationTimestamp than btpOperator1
				time.Sleep(1 * time.Second)
				btpOperator2 := createBtpOperator(secondBtpOperator)
				Eventually(func() error { return k8sClient.Create(ctx, btpOperator2) }).WithTimeout(k8sOpsTimeout).WithPolling(k8sOpsPollingInterval).Should(Succeed())
				Eventually(updateCh).Should(Receive(matchState(v1alpha1.StateError)))

				// required for later CreationTimestamp than btpOperator2
				time.Sleep(1 * time.Second)
				btpOperator3 := createBtpOperator(thirdBtpOperator)
				Eventually(func() error { return k8sClient.Create(ctx, btpOperator3) }).WithTimeout(k8sOpsTimeout).WithPolling(k8sOpsPollingInterval).Should(Succeed())
				Eventually(updateCh).Should(Receive(matchState(v1alpha1.StateError)))

				btpOperators := &v1alpha1.BtpOperatorList{}
				Expect(k8sClient.List(ctx, btpOperators)).To(Succeed())
				Expect(len(btpOperators.Items)).To(BeEquivalentTo(3))

				btpOperatorWithCurrentState := &v1alpha1.BtpOperator{}
				Eventually(func() (bool, error) {
					err := k8sClient.Get(ctx, client.ObjectKey{Namespace: defaultNamespace, Name: firstBtpOperator}, btpOperatorWithCurrentState)
					return btpOperatorWithCurrentState.Status.State == v1alpha1.StateReady, err
				}).WithTimeout(k8sOpsTimeout).WithPolling(k8sOpsPollingInterval).Should(BeTrue())
				Eventually(func() (bool, error) {
					err := k8sClient.Get(ctx, client.ObjectKey{Namespace: defaultNamespace, Name: secondBtpOperator}, btpOperatorWithCurrentState)
					return btpOperatorWithCurrentState.Status.State == v1alpha1.StateError && btpOperatorWithCurrentState.Status.Conditions[0].Reason == string(conditions.OlderCRExists), err
				}).WithTimeout(k8sOpsTimeout).WithPolling(k8sOpsPollingInterval).Should(BeTrue())
				Eventually(func() (bool, error) {
					err := k8sClient.Get(ctx, client.ObjectKey{Namespace: defaultNamespace, Name: thirdBtpOperator}, btpOperatorWithCurrentState)
					return btpOperatorWithCurrentState.Status.State == v1alpha1.StateError && btpOperatorWithCurrentState.Status.Conditions[0].Reason == string(conditions.OlderCRExists), err
				}).WithTimeout(k8sOpsTimeout).WithPolling(k8sOpsPollingInterval).Should(BeTrue())

				Expect(k8sClient.Delete(ctx, btpOperator1)).To(Succeed())
				Eventually(updateCh).Should(Receive(matchDeleted()))

				Expect(k8sClient.List(ctx, btpOperators)).To(Succeed())
				Expect(len(btpOperators.Items)).To(BeEquivalentTo(2))

				Eventually(func() (bool, error) {
					err := k8sClient.Get(ctx, client.ObjectKey{Namespace: defaultNamespace, Name: secondBtpOperator}, btpOperatorWithCurrentState)
					return btpOperatorWithCurrentState.Status.State == v1alpha1.StateReady, err
				}).WithTimeout(k8sOpsTimeout).WithPolling(k8sOpsPollingInterval).Should(BeTrue())
				Expect(btpOperatorWithCurrentState.Status.Conditions[0].Reason).To(BeEquivalentTo(conditions.ReconcileSucceeded))

				Eventually(func() (bool, error) {
					err := k8sClient.Get(ctx, client.ObjectKey{Namespace: defaultNamespace, Name: thirdBtpOperator}, btpOperatorWithCurrentState)
					return btpOperatorWithCurrentState.Status.State == v1alpha1.StateError && btpOperatorWithCurrentState.Status.Conditions[0].Reason == string(conditions.OlderCRExists), err
				}).WithTimeout(k8sOpsTimeout).WithPolling(k8sOpsPollingInterval).Should(BeTrue())
			})
		})
	})

	Describe("Two CRs (1st-Ready, 2nd-Error) and SI/SB exist", func() {
		When("First CR is under deletion, and finalizer is removed", func() {
			It("should set second CR as leader", func() {
				btpOperator1 := createBtpOperator(firstBtpOperator)
				Eventually(func() error { return k8sClient.Create(ctx, btpOperator1) }).WithTimeout(k8sOpsTimeout).WithPolling(k8sOpsPollingInterval).Should(Succeed())
				Eventually(updateCh).Should(Receive(matchState(v1alpha1.StateReady)))

				siUnstructured := createResource(InstanceGvk, kymaNamespace, instanceName)
				ensureResourceExists(InstanceGvk)

				sbUnstructured := createResource(bindingGvk, kymaNamespace, bindingName)
				ensureResourceExists(bindingGvk)

				// required for later CreationTimestamp than btpOperator1
				time.Sleep(1 * time.Second)
				btpOperator2 := createBtpOperator(secondBtpOperator)
				Eventually(func() error { return k8sClient.Create(ctx, btpOperator2) }).WithTimeout(k8sOpsTimeout).WithPolling(k8sOpsPollingInterval).Should(Succeed())
				Eventually(updateCh).Should(Receive(matchState(v1alpha1.StateError)))

				btpOperators := &v1alpha1.BtpOperatorList{}
				Expect(k8sClient.List(ctx, btpOperators)).To(Succeed())
				Expect(len(btpOperators.Items)).To(BeEquivalentTo(2))

				btpOperatorWithCurrentState := &v1alpha1.BtpOperator{}
				Eventually(func() (bool, error) {
					err := k8sClient.Get(ctx, client.ObjectKey{Namespace: defaultNamespace, Name: firstBtpOperator}, btpOperatorWithCurrentState)
					return btpOperatorWithCurrentState.Status.State == v1alpha1.StateReady, err
				}).WithTimeout(k8sOpsTimeout).WithPolling(k8sOpsPollingInterval).Should(BeTrue())
				Eventually(func() (bool, error) {
					err := k8sClient.Get(ctx, client.ObjectKey{Namespace: defaultNamespace, Name: secondBtpOperator}, btpOperatorWithCurrentState)
					return btpOperatorWithCurrentState.Status.State == v1alpha1.StateError && btpOperatorWithCurrentState.Status.Conditions[0].Reason == string(conditions.OlderCRExists), err
				}).WithTimeout(k8sOpsTimeout).WithPolling(k8sOpsPollingInterval).Should(BeTrue())

				Expect(k8sClient.Delete(ctx, btpOperator1)).To(Succeed())
				Eventually(updateCh).Should(Receive(matchReadyCondition(v1alpha1.StateWarning, metav1.ConditionFalse, conditions.ServiceInstancesAndBindingsNotCleaned)))
				Expect(k8sClient.Get(ctx, client.ObjectKey{Namespace: btpOperator1.GetNamespace(), Name: btpOperator1.GetName()}, btpOperator1)).To(Succeed())
				btpOperator1.SetFinalizers([]string{})
				Expect(k8sClient.Update(ctx, btpOperator1)).To(Succeed())
				Eventually(updateCh).Should(Receive(matchDeleted()))

				Expect(k8sClient.List(ctx, btpOperators)).To(Succeed())
				Expect(len(btpOperators.Items)).To(BeEquivalentTo(1))
				Expect(btpOperators.Items[0].Name).To(BeEquivalentTo(secondBtpOperator))

				Eventually(func() (bool, error) {
					err := k8sClient.Get(ctx, client.ObjectKey{Namespace: defaultNamespace, Name: secondBtpOperator}, btpOperatorWithCurrentState)
					return btpOperatorWithCurrentState.Status.State == v1alpha1.StateReady, err
				}).WithTimeout(k8sOpsTimeout).WithPolling(k8sOpsPollingInterval).Should(BeTrue())
				Expect(btpOperatorWithCurrentState.Status.Conditions[0].Reason).To(BeEquivalentTo(conditions.ReconcileSucceeded))

				Expect(k8sClient.Delete(ctx, siUnstructured)).To(Succeed())
				Expect(k8sClient.Delete(ctx, sbUnstructured)).To(Succeed())
			})
		})
	})
})
