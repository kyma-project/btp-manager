package controllers

import (
	"context"

	"github.com/kyma-project/btp-manager/api/v1alpha1"
	"github.com/kyma-project/btp-manager/internal/conditions"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("BTP Operator controller - provisioning", func() {
	var cr *v1alpha1.BtpOperator

	BeforeEach(func() {
		GinkgoWriter.Println("--- PROCESS:", GinkgoParallelProcess(), "---")
		ctx = context.Background()
		cr = createDefaultBtpOperator()
		cr.SetLabels(map[string]string{forceDeleteLabelKey: "true"})
		Eventually(func() error { return k8sClient.Create(ctx, cr) }).WithTimeout(k8sOpsTimeout).WithPolling(k8sOpsPollingInterval).Should(Succeed())
	})

	AfterEach(func() {
		cr = &v1alpha1.BtpOperator{}
		Expect(k8sClient.Get(ctx, client.ObjectKey{Namespace: defaultNamespace, Name: btpOperatorName}, cr)).Should(Succeed())
		Expect(k8sClient.Delete(ctx, cr)).Should(Succeed())
		Eventually(updateCh).Should(Receive(matchDeleted()))
		Expect(isCrNotFound()).To(BeTrue())
	})

	When("The required Secret is missing", func() {
		It("should return warning while getting the required Secret", func() {
			Eventually(updateCh).Should(Receive(matchReadyCondition(v1alpha1.StateProcessing, metav1.ConditionFalse, conditions.Initialized)))
			Eventually(updateCh).Should(Receive(matchReadyCondition(v1alpha1.StateWarning, metav1.ConditionFalse, conditions.MissingSecret)))
		})
	})

	Describe("The required Secret exists", func() {
		AfterEach(func() {
			deleteSecret := &corev1.Secret{}
			Expect(k8sClient.Get(ctx, client.ObjectKey{Namespace: kymaNamespace, Name: SecretName}, deleteSecret)).To(Succeed())
			Expect(k8sClient.Delete(ctx, deleteSecret)).To(Succeed())
			Eventually(updateCh).Should(Receive(matchReadyCondition(v1alpha1.StateWarning, metav1.ConditionFalse, conditions.MissingSecret)))
		})

		When("the required Secret does not have all required keys", func() {
			It("should return error while verifying keys", func() {
				secret, err := createSecretWithoutKeys()
				Expect(err).To(BeNil())
				Expect(k8sClient.Create(ctx, secret)).To(Succeed())
				Eventually(updateCh).Should(Receive(matchReadyCondition(v1alpha1.StateWarning, metav1.ConditionFalse, conditions.InvalidSecret)))
			})
		})

		When("the required Secret's keys do not have all values", func() {
			It("should return error while verifying values", func() {
				secret, err := createSecretWithoutValues()
				Expect(err).To(BeNil())
				Expect(k8sClient.Create(ctx, secret)).To(Succeed())
				Eventually(updateCh).Should(Receive(matchReadyCondition(v1alpha1.StateWarning, metav1.ConditionFalse, conditions.InvalidSecret)))
			})
		})

		When("the required Secret is correct", func() {
			It("should install chart successfully", func() {
				secret, err := createCorrectSecretFromYaml()
				Expect(err).To(BeNil())
				Expect(k8sClient.Patch(ctx, secret, client.Apply, client.ForceOwnership, client.FieldOwner(operatorName))).To(Succeed())
				Eventually(updateCh).Should(Receive(matchReadyCondition(v1alpha1.StateReady, metav1.ConditionTrue, conditions.ReconcileSucceeded)))
				btpServiceOperatorDeployment := &appsv1.Deployment{}
				Expect(k8sClient.Get(ctx, client.ObjectKey{Name: DeploymentName, Namespace: kymaNamespace}, btpServiceOperatorDeployment)).To(Succeed())
			})
		})
	})
})
