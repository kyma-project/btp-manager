package drift_test

import (
	"context"

	"github.com/kyma-project/btp-manager/controllers/config"
	"github.com/kyma-project/btp-manager/internal/conditions"
	"github.com/kyma-project/btp-manager/internal/credentials/drift"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Drift Detector", func() {
	var (
		detector  *drift.DriftDetector
		k8sClient client.Client
		ctx       context.Context
	)

	BeforeEach(func() {
		config.ChartNamespace = kymaNamespace
		ctx = context.Background()
	})

	Describe("InitializeFromSecret", func() {
		BeforeEach(func() {
			k8sClient = newFakeClient()
			detector = drift.NewDetector(k8sClient, k8sClient)
		})

		It("should extract cluster ID from the secret", func() {
			secret := btpManagerSecret("test-cluster-id", "", nil)

			detector.InitializeFromSecret(secret)

			Expect(detector.ClusterIdFromManager()).To(Equal("test-cluster-id"))
		})

		It("should extract credentials namespace from the secret", func() {
			secret := btpManagerSecret("", "custom-ns", nil)

			detector.InitializeFromSecret(secret)

			Expect(detector.CredentialsNamespaceFromManager()).To(Equal("custom-ns"))
			Expect(detector.CredentialsNamespaceFromOperator()).To(Equal("custom-ns"))
		})

		It("should default credentials namespace to ChartNamespace when not set in the secret", func() {
			secret := btpManagerSecret("", "", nil)

			detector.InitializeFromSecret(secret)

			Expect(detector.CredentialsNamespaceFromManager()).To(Equal(kymaNamespace))
		})

		It("should extract previous credentials namespace from annotation", func() {
			secret := btpManagerSecret("", "", map[string]string{
				previousCredentialsNamespaceAnnotationKey: "old-ns",
			})

			detector.InitializeFromSecret(secret)

			Expect(detector.PreviousCredentialsNamespace()).To(Equal("old-ns"))
		})

		It("should default credentials namespace to ChartNamespace when secret is nil", func() {
			detector.InitializeFromSecret(nil)

			Expect(detector.CredentialsNamespaceFromManager()).To(Equal(kymaNamespace))
			Expect(detector.CredentialsNamespaceFromOperator()).To(Equal(kymaNamespace))
			Expect(detector.ClusterIdFromManager()).To(BeEmpty())
			Expect(detector.PreviousCredentialsNamespace()).To(BeEmpty())
		})
	})

	Describe("GetDefaultCredentialsSecret", func() {
		It("should return nil when no managed secrets exist", func() {
			k8sClient = newFakeClient()
			detector = drift.NewDetector(k8sClient, k8sClient)

			secret, err := detector.GetDefaultCredentialsSecret(ctx)

			Expect(err).NotTo(HaveOccurred())
			Expect(secret).To(BeNil())
		})

		It("should return the operator secret matching the previous credentials namespace", func() {
			secretInDefault := operatorSecret(kymaNamespace)
			secretInPrevious := operatorSecret("previous-ns")
			k8sClient = newFakeClient(secretInDefault, secretInPrevious)
			detector = drift.NewDetector(k8sClient, k8sClient)

			managerSecret := btpManagerSecret("", "", map[string]string{
				previousCredentialsNamespaceAnnotationKey: "previous-ns",
			})
			detector.InitializeFromSecret(managerSecret)

			secret, err := detector.GetDefaultCredentialsSecret(ctx)

			Expect(err).NotTo(HaveOccurred())
			Expect(secret).NotTo(BeNil())
			Expect(secret.Namespace).To(Equal("previous-ns"))
		})

		It("should fall back to any matching secret when previous namespace does not match", func() {
			secretInOther := operatorSecret("other-ns")
			k8sClient = newFakeClient(secretInOther)
			detector = drift.NewDetector(k8sClient, k8sClient)

			detector.InitializeFromSecret(nil)

			secret, err := detector.GetDefaultCredentialsSecret(ctx)

			Expect(err).NotTo(HaveOccurred())
			Expect(secret).NotTo(BeNil())
			Expect(secret.Namespace).To(Equal("other-ns"))
		})

		It("should deterministically pick the lowest namespace when multiple secrets match and none is the previous namespace", func() {
			secretInB := operatorSecret("ns-b")
			secretInA := operatorSecret("ns-a")
			secretInC := operatorSecret("ns-c")
			k8sClient = newFakeClient(secretInB, secretInA, secretInC)
			detector = drift.NewDetector(k8sClient, k8sClient)

			detector.InitializeFromSecret(nil)

			secret, err := detector.GetDefaultCredentialsSecret(ctx)

			Expect(err).NotTo(HaveOccurred())
			Expect(secret).NotTo(BeNil())
			Expect(secret.Namespace).To(Equal("ns-a"))
		})

		It("should ignore secrets with a different name", func() {
			unrelatedSecret := &corev1.Secret{}
			unrelatedSecret.Name = "some-other-secret"
			unrelatedSecret.Namespace = kymaNamespace
			unrelatedSecret.Labels = map[string]string{managedByLabelKey: operatorName}

			k8sClient = newFakeClient(unrelatedSecret)
			detector = drift.NewDetector(k8sClient, k8sClient)

			secret, err := detector.GetDefaultCredentialsSecret(ctx)

			Expect(err).NotTo(HaveOccurred())
			Expect(secret).To(BeNil())
		})
	})

	Describe("GetSapBtpServiceOperatorConfigMap", func() {
		It("should return the ConfigMap when it exists", func() {
			cm := operatorConfigMap("cluster-123")
			k8sClient = newFakeClient(cm)
			detector = drift.NewDetector(k8sClient, k8sClient)

			result, err := detector.GetSapBtpServiceOperatorConfigMap(ctx)

			Expect(err).NotTo(HaveOccurred())
			Expect(result).NotTo(BeNil())
			Expect(result.Data[clusterIdConfigMapKey]).To(Equal("cluster-123"))
		})

		It("should return nil when the ConfigMap does not exist", func() {
			k8sClient = newFakeClient()
			detector = drift.NewDetector(k8sClient, k8sClient)

			result, err := detector.GetSapBtpServiceOperatorConfigMap(ctx)

			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(BeNil())
		})
	})

	Describe("CheckCredentialsNamespaceDrift", func() {
		var requiredSecret *corev1.Secret

		BeforeEach(func() {
			requiredSecret = btpManagerSecret("cluster-1", kymaNamespace, nil)
		})

		Context("when no operator secret exists", func() {
			BeforeEach(func() {
				k8sClient = newFakeClient()
				detector = drift.NewDetector(k8sClient, k8sClient)
				detector.InitializeFromSecret(requiredSecret)
			})

			It("should return nil (no drift)", func() {
				result := detector.CheckCredentialsNamespaceDrift(ctx, requiredSecret)

				Expect(result).To(BeNil())
			})
		})

		Context("when the operator secret is in the same namespace as the manager secret", func() {
			BeforeEach(func() {
				opSecret := operatorSecret(kymaNamespace)
				k8sClient = newFakeClient(opSecret)
				detector = drift.NewDetector(k8sClient, k8sClient)
				detector.InitializeFromSecret(requiredSecret)
			})

			It("should return nil (no drift)", func() {
				result := detector.CheckCredentialsNamespaceDrift(ctx, requiredSecret)

				Expect(result).To(BeNil())
			})

			It("should track the operator's namespace", func() {
				Expect(detector.CheckCredentialsNamespaceDrift(ctx, requiredSecret)).To(Succeed())

				Expect(detector.CredentialsNamespaceFromOperator()).To(Equal(kymaNamespace))
			})
		})

		Context("when the operator secret is in a different namespace", func() {
			BeforeEach(func() {
				opSecret := operatorSecret("old-ns")
				k8sClient = newFakeClient(opSecret, requiredSecret)
				detector = drift.NewDetector(k8sClient, k8sClient)
				detector.InitializeFromSecret(requiredSecret)
			})

			It("should return nil but annotate the required secret with the previous namespace", func() {
				result := detector.CheckCredentialsNamespaceDrift(ctx, requiredSecret)

				Expect(result).To(BeNil())
				Expect(requiredSecret.Annotations).To(HaveKeyWithValue(
					previousCredentialsNamespaceAnnotationKey, "old-ns",
				))
			})

			It("should track the operator's namespace", func() {
				Expect(detector.CheckCredentialsNamespaceDrift(ctx, requiredSecret)).To(Succeed())

				Expect(detector.CredentialsNamespaceFromOperator()).To(Equal("old-ns"))
			})
		})

		Context("error paths", func() {
			It("should return GettingDefaultCredentialsSecretFailed when listing secrets fails", func() {
				k8sClient = newErrorOnListClient(newFakeClient())
				detector = drift.NewDetector(k8sClient, newFakeClient())
				detector.InitializeFromSecret(requiredSecret)

				result := detector.CheckCredentialsNamespaceDrift(ctx, requiredSecret)

				Expect(result).NotTo(BeNil())
				Expect(result.Reason).To(Equal(conditions.GettingDefaultCredentialsSecretFailed))
			})

			It("should return AnnotatingSecretFailed when annotating the secret fails", func() {
				opSecret := operatorSecret("old-ns")
				// List must succeed (to detect drift); Update must fail (to trigger annotation error).
				k8sClient = newErrorOnUpdateClient(newFakeClient(opSecret, requiredSecret))
				detector = drift.NewDetector(k8sClient, newFakeClient())
				detector.InitializeFromSecret(requiredSecret)

				result := detector.CheckCredentialsNamespaceDrift(ctx, requiredSecret)

				Expect(result).NotTo(BeNil())
				Expect(result.Reason).To(Equal(conditions.AnnotatingSecretFailed))
			})
		})
	})

	Describe("CheckClusterIdConfigMapDrift", func() {
		var requiredSecret *corev1.Secret

		BeforeEach(func() {
			requiredSecret = btpManagerSecret("cluster-1", kymaNamespace, nil)
		})

		Context("when no operator ConfigMap exists", func() {
			BeforeEach(func() {
				k8sClient = newFakeClient()
				detector = drift.NewDetector(k8sClient, k8sClient)
				detector.InitializeFromSecret(requiredSecret)
			})

			It("should return nil (no drift)", func() {
				result := detector.CheckClusterIdConfigMapDrift(ctx, requiredSecret)

				Expect(result).To(BeNil())
			})
		})

		Context("when the ConfigMap cluster ID matches the manager secret", func() {
			BeforeEach(func() {
				cm := operatorConfigMap("cluster-1")
				k8sClient = newFakeClient(cm)
				detector = drift.NewDetector(k8sClient, k8sClient)
				detector.InitializeFromSecret(requiredSecret)
			})

			It("should return nil (no drift)", func() {
				result := detector.CheckClusterIdConfigMapDrift(ctx, requiredSecret)

				Expect(result).To(BeNil())
			})

			It("should track the ConfigMap's cluster ID", func() {
				Expect(detector.CheckClusterIdConfigMapDrift(ctx, requiredSecret)).To(Succeed())

				Expect(detector.ClusterIdFromOperatorConfigMap()).To(Equal("cluster-1"))
			})
		})

		Context("when the ConfigMap cluster ID differs from the manager secret", func() {
			BeforeEach(func() {
				cm := operatorConfigMap("old-cluster-id")
				k8sClient = newFakeClient(cm, requiredSecret)
				detector = drift.NewDetector(k8sClient, k8sClient)
				detector.InitializeFromSecret(requiredSecret)
			})

			It("should return nil but annotate the required secret with the old cluster ID", func() {
				result := detector.CheckClusterIdConfigMapDrift(ctx, requiredSecret)

				Expect(result).To(BeNil())
				Expect(requiredSecret.Annotations).To(HaveKeyWithValue(
					previousClusterIdAnnotationKey, "old-cluster-id",
				))
			})

			It("should track the ConfigMap's cluster ID", func() {
				Expect(detector.CheckClusterIdConfigMapDrift(ctx, requiredSecret)).To(Succeed())

				Expect(detector.ClusterIdFromOperatorConfigMap()).To(Equal("old-cluster-id"))
			})
		})

		Context("error paths", func() {
			It("should return GettingSapBtpServiceOperatorConfigMapFailed when the ConfigMap fetch fails", func() {
				badClient := newErrorOnGetClient(newFakeClient())
				detector = drift.NewDetector(badClient, badClient)
				detector.InitializeFromSecret(requiredSecret)

				result := detector.CheckClusterIdConfigMapDrift(ctx, requiredSecret)

				Expect(result).NotTo(BeNil())
				Expect(result.Reason).To(Equal(conditions.GettingSapBtpServiceOperatorConfigMapFailed))
			})

			It("should return AnnotatingSecretFailed when annotating the secret fails", func() {
				cm := operatorConfigMap("old-cluster-id")
				// Get must succeed (to return ConfigMap); Update must fail (to trigger annotation error).
				k8sClient = newErrorOnUpdateClient(newFakeClient(cm, requiredSecret))
				detector = drift.NewDetector(k8sClient, newFakeClient())
				detector.InitializeFromSecret(requiredSecret)

				result := detector.CheckClusterIdConfigMapDrift(ctx, requiredSecret)

				Expect(result).NotTo(BeNil())
				Expect(result.Reason).To(Equal(conditions.AnnotatingSecretFailed))
			})
		})
	})

	Describe("ResolveClusterIdSecretDrift", func() {
		var requiredSecret *corev1.Secret

		BeforeEach(func() {
			requiredSecret = btpManagerSecret("cluster-1", kymaNamespace, nil)
		})

		Context("when no cluster ID secret exists", func() {
			BeforeEach(func() {
				k8sClient = newFakeClient()
				detector = drift.NewDetector(k8sClient, k8sClient)
				detector.InitializeFromSecret(requiredSecret)
			})

			It("should return nil (no drift)", func() {
				result := detector.ResolveClusterIdSecretDrift(ctx, requiredSecret)

				Expect(result).To(BeNil())
			})
		})

		Context("when the cluster ID secret matches the ConfigMap value", func() {
			BeforeEach(func() {
				cidSecret := clusterIdSecret(kymaNamespace, "cluster-1")
				k8sClient = newFakeClient(cidSecret)
				detector = drift.NewDetector(k8sClient, k8sClient)
				detector.InitializeFromSecret(requiredSecret)
				detector.SetClusterIdFromOperatorConfigMap("cluster-1")
			})

			It("should return nil (no drift)", func() {
				result := detector.ResolveClusterIdSecretDrift(ctx, requiredSecret)

				Expect(result).To(BeNil())
			})

			It("should track the cluster ID secret's value", func() {
				Expect(detector.ResolveClusterIdSecretDrift(ctx, requiredSecret)).To(Succeed())

				Expect(detector.ClusterIdFromOperatorClusterIdSecret()).To(Equal("cluster-1"))
			})
		})

		Context("when the cluster ID secret differs from the ConfigMap value", func() {
			var cidSecret *corev1.Secret

			BeforeEach(func() {
				cidSecret = clusterIdSecret(kymaNamespace, "stale-cluster-id")
				pod := operatorPod(operandName+"-controller-manager-abc", kymaNamespace, corev1.ConditionFalse)
				k8sClient = newFakeClient(cidSecret, pod, requiredSecret)
				detector = drift.NewDetector(k8sClient, k8sClient)
				detector.InitializeFromSecret(requiredSecret)
				detector.SetClusterIdFromOperatorConfigMap("cluster-1")
			})

			It("should annotate the required secret with the stale cluster ID", func() {
				Expect(detector.ResolveClusterIdSecretDrift(ctx, requiredSecret)).To(Succeed())

				Expect(requiredSecret.Annotations).To(HaveKeyWithValue(
					previousClusterIdAnnotationKey, "stale-cluster-id",
				))
			})

			It("should delete the stale cluster ID secret", func() {
				result := detector.ResolveClusterIdSecretDrift(ctx, requiredSecret)

				Expect(result).To(BeNil())

				secret := &corev1.Secret{}
				err := k8sClient.Get(ctx, client.ObjectKeyFromObject(cidSecret), secret)
				Expect(err).To(HaveOccurred())
			})

			It("should restart the not-ready operator pod", func() {
				result := detector.ResolveClusterIdSecretDrift(ctx, requiredSecret)

				Expect(result).To(BeNil())

				pods := &corev1.PodList{}
				Expect(k8sClient.List(ctx, pods, client.MatchingLabels{instanceLabelKey: operandName})).To(Succeed())
				Expect(pods.Items).To(BeEmpty())
			})
		})

		Context("when the cluster ID secret differs but the operator pod is ready", func() {
			BeforeEach(func() {
				cidSecret := clusterIdSecret(kymaNamespace, "stale-cluster-id")
				pod := operatorPod(operandName+"-controller-manager-abc", kymaNamespace, corev1.ConditionTrue)
				k8sClient = newFakeClient(cidSecret, pod, requiredSecret)
				detector = drift.NewDetector(k8sClient, k8sClient)
				detector.InitializeFromSecret(requiredSecret)
				detector.SetClusterIdFromOperatorConfigMap("cluster-1")
			})

			It("should not delete the ready pod", func() {
				Expect(detector.ResolveClusterIdSecretDrift(ctx, requiredSecret)).To(Succeed())

				pods := &corev1.PodList{}
				Expect(k8sClient.List(ctx, pods, client.MatchingLabels{instanceLabelKey: operandName})).To(Succeed())
				Expect(pods.Items).To(HaveLen(1))
			})
		})

		Context("when the cluster ID secret differs but the labelled pod is in a different namespace", func() {
			BeforeEach(func() {
				cidSecret := clusterIdSecret(kymaNamespace, "stale-cluster-id")
				podInWrongNs := operatorPod(operandName+"-abc", "other-namespace", corev1.ConditionFalse)
				k8sClient = newFakeClient(cidSecret, podInWrongNs, requiredSecret)
				detector = drift.NewDetector(k8sClient, k8sClient)
				detector.InitializeFromSecret(requiredSecret)
				detector.SetClusterIdFromOperatorConfigMap("cluster-1")
			})

			It("should not restart a pod outside ChartNamespace", func() {
				Expect(detector.ResolveClusterIdSecretDrift(ctx, requiredSecret)).To(Succeed())

				pods := &corev1.PodList{}
				Expect(k8sClient.List(ctx, pods, client.MatchingLabels{instanceLabelKey: operandName})).To(Succeed())
				Expect(pods.Items).To(HaveLen(1))
			})
		})

		Context("error paths", func() {
			It("should return GettingSapBtpServiceOperatorClusterIdSecretFailed when the cluster ID secret fetch fails", func() {
				badApiClient := newErrorOnGetClient(newFakeClient())
				detector = drift.NewDetector(k8sClient, badApiClient)
				detector.InitializeFromSecret(requiredSecret)

				result := detector.ResolveClusterIdSecretDrift(ctx, requiredSecret)

				Expect(result).NotTo(BeNil())
				Expect(result.Reason).To(Equal(conditions.GettingSapBtpServiceOperatorClusterIdSecretFailed))
			})

			It("should return AnnotatingSecretFailed when annotating the required secret fails", func() {
				cidSecret := clusterIdSecret(kymaNamespace, "stale-cluster-id")
				// Get must succeed (apiServerClient); Update must fail (cached client).
				apiServerClient := newFakeClient(cidSecret)
				cachedClient := newErrorOnUpdateClient(newFakeClient(requiredSecret))
				detector = drift.NewDetector(cachedClient, apiServerClient)
				detector.InitializeFromSecret(requiredSecret)
				detector.SetClusterIdFromOperatorConfigMap("cluster-1")

				result := detector.ResolveClusterIdSecretDrift(ctx, requiredSecret)

				Expect(result).NotTo(BeNil())
				Expect(result.Reason).To(Equal(conditions.AnnotatingSecretFailed))
			})

			It("should return DeletionOfOrphanedResourcesFailed when deleting the cluster ID secret fails", func() {
				cidSecret := clusterIdSecret(kymaNamespace, "stale-cluster-id")
				// Get must succeed; Update must succeed; Delete must fail.
				apiServerClient := newErrorOnDeleteClient(newFakeClient(cidSecret))
				cachedClient := newFakeClient(requiredSecret)
				detector = drift.NewDetector(cachedClient, apiServerClient)
				detector.InitializeFromSecret(requiredSecret)
				detector.SetClusterIdFromOperatorConfigMap("cluster-1")

				result := detector.ResolveClusterIdSecretDrift(ctx, requiredSecret)

				Expect(result).NotTo(BeNil())
				Expect(result.Reason).To(Equal(conditions.DeletionOfOrphanedResourcesFailed))
			})

			It("should return ResourceRemovalFailed when restarting the operator pod fails", func() {
				cidSecret := clusterIdSecret(kymaNamespace, "stale-cluster-id")
				notReadyPod := operatorPod(operandName+"-abc", kymaNamespace, corev1.ConditionFalse)
				// Get/Delete must succeed; List must fail to simulate pod listing failure during restart.
				apiServerClient := newErrorOnListClient(newFakeClient(cidSecret, notReadyPod))
				cachedClient := newFakeClient(requiredSecret)
				detector = drift.NewDetector(cachedClient, apiServerClient)
				detector.InitializeFromSecret(requiredSecret)
				detector.SetClusterIdFromOperatorConfigMap("cluster-1")

				result := detector.ResolveClusterIdSecretDrift(ctx, requiredSecret)

				Expect(result).NotTo(BeNil())
				Expect(result.Reason).To(Equal(conditions.ResourceRemovalFailed))
			})
		})
	})

	Describe("DeleteChangedResources", func() {
		// DeleteChangedResources relies on internal state populated by the
		// Check* methods. These tests simulate the real reconciler workflow:
		// InitializeFromSecret → Check* → DeleteChangedResources.

		Context("when nothing changed", func() {
			BeforeEach(func() {
				cidSecret := clusterIdSecret(kymaNamespace, "cluster-1")
				cm := operatorConfigMap("cluster-1")
				opSecret := operatorSecret(kymaNamespace)
				pod := operatorPod(operandName+"-abc", kymaNamespace, corev1.ConditionTrue)
				requiredSecret := btpManagerSecret("cluster-1", kymaNamespace, nil)
				k8sClient = newFakeClient(cidSecret, cm, opSecret, pod, requiredSecret)
				detector = drift.NewDetector(k8sClient, k8sClient)

				detector.InitializeFromSecret(requiredSecret)
				Expect(detector.CheckCredentialsNamespaceDrift(ctx, requiredSecret)).To(Succeed())
				Expect(detector.CheckClusterIdConfigMapDrift(ctx, requiredSecret)).To(Succeed())
				Expect(detector.ResolveClusterIdSecretDrift(ctx, requiredSecret)).To(Succeed())
			})

			It("should not delete any resources", func() {
				err := detector.DeleteChangedResources(ctx)

				Expect(err).NotTo(HaveOccurred())

				secrets := &corev1.SecretList{}
				Expect(k8sClient.List(ctx, secrets)).To(Succeed())
				secretNames := extractSecretNames(secrets.Items)
				Expect(secretNames).To(ContainElements(
					sapBtpServiceOperatorSecretName,
					sapBtpServiceOperatorClusterIdSecretName,
				))

				pods := &corev1.PodList{}
				Expect(k8sClient.List(ctx, pods)).To(Succeed())
				Expect(pods.Items).To(HaveLen(1))
			})
		})

		Context("when cluster ID changed (manager secret differs from ConfigMap)", func() {
			BeforeEach(func() {
				cidSecret := clusterIdSecret(kymaNamespace, "old-cluster")
				cm := operatorConfigMap("old-cluster")
				opSecret := operatorSecret(kymaNamespace)
				pod := operatorPod(operandName+"-abc", kymaNamespace, corev1.ConditionTrue)
				requiredSecret := btpManagerSecret("new-cluster", kymaNamespace, nil)
				k8sClient = newFakeClient(cidSecret, cm, opSecret, pod, requiredSecret)
				detector = drift.NewDetector(k8sClient, k8sClient)

				detector.InitializeFromSecret(requiredSecret)
				Expect(detector.CheckCredentialsNamespaceDrift(ctx, requiredSecret)).To(Succeed())
				Expect(detector.CheckClusterIdConfigMapDrift(ctx, requiredSecret)).To(Succeed())
				Expect(detector.ResolveClusterIdSecretDrift(ctx, requiredSecret)).To(Succeed())
			})

			It("should delete the cluster ID secret and the operator pod", func() {
				err := detector.DeleteChangedResources(ctx)

				Expect(err).NotTo(HaveOccurred())

				pods := &corev1.PodList{}
				Expect(k8sClient.List(ctx, pods)).To(Succeed())
				Expect(pods.Items).To(BeEmpty())

				secrets := &corev1.SecretList{}
				Expect(k8sClient.List(ctx, secrets)).To(Succeed())
				secretNames := extractSecretNames(secrets.Items)
				Expect(secretNames).NotTo(ContainElement(sapBtpServiceOperatorClusterIdSecretName))
			})

			It("should not delete the operator credentials secret", func() {
				err := detector.DeleteChangedResources(ctx)

				Expect(err).NotTo(HaveOccurred())

				secrets := &corev1.SecretList{}
				Expect(k8sClient.List(ctx, secrets)).To(Succeed())
				secretNames := extractSecretNames(secrets.Items)
				Expect(secretNames).To(ContainElement(sapBtpServiceOperatorSecretName))
			})
		})

		Context("when credentials namespace changed", func() {
			BeforeEach(func() {
				cidSecret := clusterIdSecret("old-ns", "cluster-1")
				opSecret := operatorSecret("old-ns")
				pod := operatorPod(operandName+"-abc", kymaNamespace, corev1.ConditionTrue)
				requiredSecret := btpManagerSecret("cluster-1", "new-ns", nil)
				k8sClient = newFakeClient(cidSecret, opSecret, pod, requiredSecret)
				detector = drift.NewDetector(k8sClient, k8sClient)

				detector.InitializeFromSecret(requiredSecret)
				Expect(detector.CheckCredentialsNamespaceDrift(ctx, requiredSecret)).To(Succeed())
			})

			It("should delete the cluster ID secret, the operator pod, and the credentials secret", func() {
				err := detector.DeleteChangedResources(ctx)

				Expect(err).NotTo(HaveOccurred())

				pods := &corev1.PodList{}
				Expect(k8sClient.List(ctx, pods)).To(Succeed())
				Expect(pods.Items).To(BeEmpty())

				secrets := &corev1.SecretList{}
				Expect(k8sClient.List(ctx, secrets)).To(Succeed())
				secretNames := extractSecretNames(secrets.Items)
				Expect(secretNames).NotTo(ContainElement(sapBtpServiceOperatorClusterIdSecretName))
				Expect(secretNames).NotTo(ContainElement(sapBtpServiceOperatorSecretName))
			})
		})

		Context("when resources targeted for deletion are already absent", func() {
			BeforeEach(func() {
				cm := operatorConfigMap("old-cluster")
				requiredSecret := btpManagerSecret("new-cluster", kymaNamespace, nil)
				k8sClient = newFakeClient(cm, requiredSecret)
				detector = drift.NewDetector(k8sClient, k8sClient)

				detector.InitializeFromSecret(requiredSecret)
				Expect(detector.CheckClusterIdConfigMapDrift(ctx, requiredSecret)).To(Succeed())
			})

			It("should succeed without errors", func() {
				err := detector.DeleteChangedResources(ctx)

				Expect(err).NotTo(HaveOccurred())
			})
		})
	})

	Describe("DeleteClusterIdSecret", func() {
		Context("when the cluster ID secret exists", func() {
			BeforeEach(func() {
				cidSecret := clusterIdSecret(kymaNamespace, "cluster-1")
				k8sClient = newFakeClient(cidSecret)
				detector = drift.NewDetector(k8sClient, k8sClient)

				managerSecret := btpManagerSecret("cluster-1", kymaNamespace, nil)
				detector.InitializeFromSecret(managerSecret)
			})

			It("should delete it", func() {
				err := detector.DeleteClusterIdSecret(ctx)

				Expect(err).NotTo(HaveOccurred())

				secrets := &corev1.SecretList{}
				Expect(k8sClient.List(ctx, secrets)).To(Succeed())
				Expect(secrets.Items).To(BeEmpty())
			})
		})

		Context("when the cluster ID secret does not exist", func() {
			BeforeEach(func() {
				k8sClient = newFakeClient()
				detector = drift.NewDetector(k8sClient, k8sClient)

				managerSecret := btpManagerSecret("cluster-1", kymaNamespace, nil)
				detector.InitializeFromSecret(managerSecret)
			})

			It("should succeed without errors", func() {
				err := detector.DeleteClusterIdSecret(ctx)

				Expect(err).NotTo(HaveOccurred())
			})
		})
	})
})

func extractSecretNames(secrets []corev1.Secret) []string {
	names := make([]string, 0, len(secrets))
	for _, s := range secrets {
		names = append(names, s.Name)
	}
	return names
}
