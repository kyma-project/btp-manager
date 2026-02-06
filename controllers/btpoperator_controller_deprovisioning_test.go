package controllers

import (
	"github.com/kyma-project/btp-manager/api/v1alpha1"
	"github.com/kyma-project/btp-manager/controllers/config"
	"github.com/kyma-project/btp-manager/internal/conditions"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	instanceName = "my-service-instance"
	bindingName  = "my-service-binding"
)

var _ = Describe("BTP Operator controller - deprovisioning", func() {
	var cr *v1alpha1.BtpOperator

	Describe("Deprovisioning without force-delete label", func() {

		BeforeEach(func() {
			GinkgoWriter.Println("--- PROCESS:", GinkgoParallelProcess(), "---")
			secret, err := createCorrectSecretFromYaml()
			Expect(err).To(BeNil())
			Expect(k8sClient.Patch(ctx, secret, client.Apply, client.ForceOwnership, client.FieldOwner(operatorName))).To(Succeed())
			cr = createDefaultBtpOperator()
			Expect(k8sClient.Create(ctx, cr)).To(Succeed())
			Eventually(updateCh).Should(Receive(matchState(v1alpha1.StateReady)))
			btpServiceOperatorDeployment := &appsv1.Deployment{}
			Expect(k8sClient.Get(ctx, client.ObjectKey{Name: config.DeploymentName, Namespace: kymaNamespace}, btpServiceOperatorDeployment)).Should(Succeed())
		})

		AfterEach(func() {
			Expect(k8sClient.Get(ctx, client.ObjectKey{Namespace: kymaNamespace, Name: btpOperatorName}, cr)).To(Succeed())
			cr.SetLabels(map[string]string{forceDeleteLabelKey: "true"})
			Expect(k8sClient.Update(ctx, cr)).To(Succeed())
			Eventually(func() (bool, error) {
				err := k8sClient.Get(ctx, client.ObjectKey{Namespace: kymaNamespace, Name: btpOperatorName}, cr)
				return cr.Labels[forceDeleteLabelKey] == "true", err
			}).WithTimeout(k8sOpsTimeout).WithPolling(k8sOpsPollingInterval).Should(BeTrue())
			Eventually(updateCh).Should(Receive(matchDeleted()))
			deleteSecret := &corev1.Secret{}
			Expect(k8sClient.Get(ctx, client.ObjectKey{Namespace: kymaNamespace, Name: config.SecretName}, deleteSecret)).To(Succeed())
			Expect(k8sClient.Delete(ctx, deleteSecret)).To(Succeed())
		})

		It("Delete should fail because of existing instances and bindings", func() {
			_ = createResource(instanceGvk, kymaNamespace, instanceName)
			ensureResourceExists(instanceGvk)

			_ = createResource(bindingGvk, kymaNamespace, bindingName)
			ensureResourceExists(bindingGvk)

			Expect(k8sClient.Get(ctx, client.ObjectKey{Namespace: kymaNamespace, Name: btpOperatorName}, cr)).To(Succeed())
			Expect(k8sClient.Delete(ctx, cr)).To(Succeed())

			Eventually(updateCh).Should(Receive(matchReadyCondition(v1alpha1.StateWarning, metav1.ConditionFalse, conditions.ServiceInstancesAndBindingsNotCleaned)))
		})

	})

	Describe("Deprovisioning with force-delete label", func() {
		var siUnstructured, sbUnstructured *unstructured.Unstructured

		BeforeEach(func() {
			GinkgoWriter.Println("--- PROCESS:", GinkgoParallelProcess(), "---")
			secret, err := createCorrectSecretFromYaml()
			Expect(err).To(BeNil())
			Expect(k8sClient.Patch(ctx, secret, client.Apply, client.ForceOwnership, client.FieldOwner(operatorName))).To(Succeed())
			cr = createDefaultBtpOperator()
			cr.SetLabels(map[string]string{forceDeleteLabelKey: "true"})
			Expect(k8sClient.Create(ctx, cr)).To(Succeed())
			Eventually(updateCh).Should(Receive(matchState(v1alpha1.StateReady)))
			btpServiceOperatorDeployment := &appsv1.Deployment{}
			Expect(k8sClient.Get(ctx, client.ObjectKey{Name: config.DeploymentName, Namespace: kymaNamespace}, btpServiceOperatorDeployment)).Should(Succeed())

			siUnstructured = createResource(instanceGvk, kymaNamespace, instanceName)
			ensureResourceExists(instanceGvk)

			sbUnstructured = createResource(bindingGvk, kymaNamespace, bindingName)
			ensureResourceExists(bindingGvk)
		})

		AfterEach(func() {
			deleteSecret := &corev1.Secret{}
			Expect(k8sClient.Get(ctx, client.ObjectKey{Namespace: kymaNamespace, Name: config.SecretName}, deleteSecret)).To(Succeed())
			Expect(k8sClient.Delete(ctx, deleteSecret)).To(Succeed())
		})

		It("soft delete (after timeout) should succeed", func() {
			reconciler.Client = newTimeoutK8sClient(reconciler.Client)
			setFinalizers(siUnstructured)
			setFinalizers(sbUnstructured)
			Expect(k8sClient.Get(ctx, client.ObjectKey{Namespace: kymaNamespace, Name: btpOperatorName}, cr)).To(Succeed())
			Expect(k8sClient.Delete(ctx, cr)).To(Succeed())
			Eventually(updateCh).Should(Receive(matchReadyCondition(v1alpha1.StateDeleting, metav1.ConditionFalse, conditions.HardDeleting)))
			Eventually(updateCh).Should(Receive(matchReadyCondition(v1alpha1.StateDeleting, metav1.ConditionFalse, conditions.SoftDeleting)))
			Eventually(updateCh).Should(Receive(matchDeleted()))
			doChecks()
		})

		It("soft delete (after hard deletion fail) should succeed", func() {
			reconciler.Client = newErrorK8sClient(reconciler.Client)
			setFinalizers(siUnstructured)
			setFinalizers(sbUnstructured)
			Expect(k8sClient.Get(ctx, client.ObjectKey{Namespace: kymaNamespace, Name: btpOperatorName}, cr)).To(Succeed())
			Expect(k8sClient.Delete(ctx, cr)).To(Succeed())
			Eventually(updateCh).Should(Receive(matchReadyCondition(v1alpha1.StateDeleting, metav1.ConditionFalse, conditions.SoftDeleting)))
			Eventually(updateCh).Should(Receive(matchDeleted()))
			doChecks()
		})

		It("hard delete should succeed", func() {
			reconciler.Client = k8sClientFromManager
			Expect(k8sClient.Get(ctx, client.ObjectKey{Namespace: kymaNamespace, Name: btpOperatorName}, cr)).To(Succeed())
			Expect(k8sClient.Delete(ctx, cr)).Should(Succeed())
			Eventually(updateCh).Should(Receive(matchReadyCondition(v1alpha1.StateDeleting, metav1.ConditionFalse, conditions.HardDeleting)))
			Eventually(updateCh).Should(Receive(matchDeleted()))
			doChecks()
		})
	})

	Describe("Deprovisioning with network policies", func() {
		var networkPolicyNames = []string{
			"kyma-project.io--btp-operator-allow-to-apiserver",
			"kyma-project.io--btp-operator-to-dns",
			"kyma-project.io--allow-btp-operator-metrics",
			"kyma-project.io--allow-btp-operator-webhook",
		}

		BeforeEach(func() {
			GinkgoWriter.Println("--- PROCESS:", GinkgoParallelProcess(), "---")
			secret, err := createCorrectSecretFromYaml()
			Expect(err).To(BeNil())
			Expect(k8sClient.Patch(ctx, secret, client.Apply, client.ForceOwnership, client.FieldOwner(operatorName))).To(Succeed())
			cr = createDefaultBtpOperator()
			cr.SetLabels(map[string]string{forceDeleteLabelKey: "true"})
			Expect(k8sClient.Create(ctx, cr)).To(Succeed())
			Eventually(updateCh).Should(Receive(matchState(v1alpha1.StateReady)))
			Eventually(func() bool {
				for _, policyName := range networkPolicyNames {
					policy := &unstructured.Unstructured{}
					policy.SetAPIVersion("networking.k8s.io/v1")
					policy.SetKind("NetworkPolicy")
					err := k8sClient.Get(ctx, client.ObjectKey{Name: policyName, Namespace: "kyma-system"}, policy)
					if err != nil {
						return false
					}
					labels := policy.GetLabels()
					if labels == nil || labels[managedByLabelKey] != operatorName {
						return false
					}
				}
				return true
			}).Should(BeTrue(), "All network policies should exist after provisioning")
		})

		AfterEach(func() {
			deleteSecret := &corev1.Secret{}
			Expect(k8sClient.Get(ctx, client.ObjectKey{Namespace: kymaNamespace, Name: config.SecretName}, deleteSecret)).To(Succeed())
			Expect(k8sClient.Delete(ctx, deleteSecret)).To(Succeed())
		})

		It("Should delete network policies during deprovisioning", func() {
			Expect(k8sClient.Get(ctx, client.ObjectKey{Namespace: kymaNamespace, Name: btpOperatorName}, cr)).To(Succeed())
			Expect(k8sClient.Delete(ctx, cr)).Should(Succeed())
			Eventually(updateCh).Should(Receive(matchReadyCondition(v1alpha1.StateDeleting, metav1.ConditionFalse, conditions.HardDeleting)))
			Eventually(updateCh).Should(Receive(matchDeleted()))
			Eventually(func() bool {
				for _, policyName := range networkPolicyNames {
					policy := &unstructured.Unstructured{}
					policy.SetAPIVersion("networking.k8s.io/v1")
					policy.SetKind("NetworkPolicy")
					err := k8sClient.Get(ctx, client.ObjectKey{Name: policyName, Namespace: "kyma-system"}, policy)
					if err == nil {
						return false
					}
				}
				return true
			}).Should(BeTrue(), "All network policies should be deleted during deprovisioning")
		})
	})
})
