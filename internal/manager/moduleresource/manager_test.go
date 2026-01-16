package moduleresource

import (
	"context"
	"encoding/base64"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/kyma-project/btp-manager/api/v1alpha1"
)

var (
	fakeClient client.Client
	scheme     *runtime.Scheme
)

func TestResources(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Resources Suite")
}

var _ = BeforeSuite(func() {
	scheme = runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(v1alpha1.AddToScheme(scheme))

	fakeClient = fake.NewClientBuilder().
		WithScheme(scheme).
		Build()
})

var _ = Describe("Manager", func() {
	var (
		manager *Manager
	)

	BeforeEach(func() {
		// Rebuild fake client for each test to avoid state pollution
		fakeClient = fake.NewClientBuilder().
			WithScheme(scheme).
			Build()

		manager = NewModuleResourceManager(fakeClient, scheme, Config{
			ChartNamespace:       "kyma-system",
			ResourcesPath:        "./testdata",
			ManagerResourcesPath: "./testdata",
		})
	})

	Describe("create unstructured objects from manifests directory", func() {
		It("should load and convert manifests to unstructured objects", func() {
			objects, err := manager.createUnstructuredObjectsFromManifestsDir("./testdata")

			Expect(err).NotTo(HaveOccurred())
			Expect(objects).To(HaveLen(3))
			Expect(manager.resourceIndices).To(HaveLen(3))

			configMapIndex := manager.resourceIndices[ModuleResource{Kind: "ConfigMap", Name: "test-configmap"}]
			Expect(objects[configMapIndex].GetKind()).To(Equal("ConfigMap"))
			Expect(objects[configMapIndex].GetName()).To(Equal("test-configmap"))
			Expect(objects[configMapIndex].GetNamespace()).To(Equal("test-namespace"))

			deploymentIndex := manager.resourceIndices[ModuleResource{Kind: "Deployment", Name: "test-deployment"}]
			Expect(objects[deploymentIndex].GetKind()).To(Equal("Deployment"))
			Expect(objects[deploymentIndex].GetName()).To(Equal("test-deployment"))
			Expect(objects[deploymentIndex].GetNamespace()).To(Equal("test-namespace"))

			secretIndex := manager.resourceIndices[ModuleResource{Kind: "Secret", Name: "test-secret"}]
			Expect(objects[secretIndex].GetKind()).To(Equal("Secret"))
			Expect(objects[secretIndex].GetName()).To(Equal("test-secret"))
		})

		It("should return error for non-existent directory", func() {
			_, err := manager.createUnstructuredObjectsFromManifestsDir("./non-existent")

			Expect(err).To(HaveOccurred())
		})
	})

	Describe("add labels", func() {
		It("should add managed-by, chart version, and module labels to all resources", func() {
			objects, err := manager.createUnstructuredObjectsFromManifestsDir("./testdata")
			Expect(err).NotTo(HaveOccurred())

			chartVersion := "0.0.1"
			manager.addLabels(chartVersion, objects...)

			for _, obj := range objects {
				labels := obj.GetLabels()
				Expect(labels).To(HaveKey(ManagedByLabelKey))
				Expect(labels[ManagedByLabelKey]).To(Equal(OperatorName))
				Expect(labels[KymaProjectModuleLabelKey]).To(Equal(ModuleName))
				Expect(labels[ChartVersionLabelKey]).To(Equal(chartVersion))
			}

			deployIdx := manager.resourceIndices[ModuleResource{Kind: "Deployment", Name: "test-deployment"}]
			spec, found, err := unstructured.NestedMap(objects[deployIdx].Object, "spec", "template", "metadata", "labels")
			Expect(err).NotTo(HaveOccurred())
			Expect(found).To(BeTrue())
			Expect(spec[KymaProjectModuleLabelKey]).To(Equal(ModuleName))
		})
	})

	Describe("set namespace", func() {
		It("should set namespace in all resources", func() {
			objects, err := manager.createUnstructuredObjectsFromManifestsDir("./testdata")
			Expect(err).NotTo(HaveOccurred())

			manager.setNamespace(objects)

			for _, obj := range objects {
				Expect(obj.GetNamespace()).To(Equal("kyma-system"))
			}
		})
	})

	Describe("set ConfigMap values", func() {
		It("should set ConfigMap values from secret", func() {
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-secret",
					Namespace: "test-namespace",
				},
				Data: map[string][]byte{
					"cluster_id":            []byte("test-cluster-123"),
					"credentials_namespace": []byte("test-creds-ns"),
					"clientid":              []byte("test-client"),
				},
			}

			manager.config.EnableLimitedCache = "true"
			manager.UpdateState(State{
				ClusterID:            "test-cluster-123",
				CredentialsNamespace: "test-creds-ns",
			})

			objects, err := manager.createUnstructuredObjectsFromManifestsDir("./testdata")
			Expect(err).NotTo(HaveOccurred())

			configMapIndex, found := manager.resourceIndices[ModuleResource{Kind: "ConfigMap", Name: "test-configmap"}]
			Expect(found).To(BeTrue())
			configMap := objects[configMapIndex]

			err = manager.setConfigMapValues(secret, configMap)
			Expect(err).NotTo(HaveOccurred())

			data, found, err := unstructured.NestedStringMap(configMap.Object, "data")
			Expect(err).NotTo(HaveOccurred())
			Expect(found).To(BeTrue())
			Expect(data["CLUSTER_ID"]).To(Equal("test-cluster-123"))
			Expect(data["RELEASE_NAMESPACE"]).To(Equal("test-creds-ns"))
			Expect(data["MANAGEMENT_NAMESPACE"]).To(Equal("test-creds-ns"))
			Expect(data["ENABLE_LIMITED_CACHE"]).To(Equal("true"))
		})
	})

	Describe("set Secret values", func() {
		It("should copy secret data with base64 encoding excluding cluster_id and credentials_namespace", func() {
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-secret",
					Namespace: "test-namespace",
				},
				Data: map[string][]byte{
					"cluster_id":            []byte("test-cluster-123"),
					"credentials_namespace": []byte("test-creds-ns"),
					"clientid":              []byte("test-client"),
					"clientsecret":          []byte("test-secret"),
					"sm_url":                []byte("https://test.url"),
				},
			}

			manager.UpdateState(State{
				ClusterID:            "test-cluster-123",
				CredentialsNamespace: "target-namespace",
			})

			secretObj := &unstructured.Unstructured{}
			secretObj.SetKind("Secret")
			secretObj.SetName("sap-btp-service-operator")
			secretObj.SetNamespace("default")

			err := manager.setSecretValues(secret, secretObj)
			Expect(err).NotTo(HaveOccurred())

			Expect(secretObj.GetNamespace()).To(Equal("target-namespace"))

			data, found, err := unstructured.NestedStringMap(secretObj.Object, "data")
			Expect(err).NotTo(HaveOccurred())
			Expect(found).To(BeTrue())

			Expect(data).To(HaveKey("clientid"))
			Expect(data).To(HaveKey("clientsecret"))
			Expect(data).To(HaveKey("sm_url"))
			Expect(data).NotTo(HaveKey("cluster_id"))
			Expect(data).NotTo(HaveKey("credentials_namespace"))

			decoded, err := base64.StdEncoding.DecodeString(data["clientid"])
			Expect(err).NotTo(HaveOccurred())
			Expect(string(decoded)).To(Equal("test-client"))
		})
	})

	Describe("set Deployment images", func() {
		It("should set container images for manager and kube-rbac-proxy", func() {
			const (
				sapBtpOperatorImage = "local.test/kyma-project/sap-btp-operator:v0.0.1"
				kubeRbacProxyImage  = "local.test/kyma-project/kube-rbac-proxy:v0.0.1"
			)
			manager.config.SapBtpOperatorImage = sapBtpOperatorImage
			manager.config.KubeRbacProxyImage = kubeRbacProxyImage

			objects, err := manager.createUnstructuredObjectsFromManifestsDir("./testdata")
			Expect(err).NotTo(HaveOccurred())

			deploymentIndex, found := manager.resourceIndices[ModuleResource{Kind: "Deployment", Name: "test-deployment"}]
			Expect(found).To(BeTrue())
			deployment := objects[deploymentIndex]

			err = manager.setDeploymentImages(deployment)
			Expect(err).NotTo(HaveOccurred())

			containers, found, err := unstructured.NestedSlice(deployment.Object, "spec", "template", "spec", "containers")
			Expect(err).NotTo(HaveOccurred())
			Expect(found).To(BeTrue())
			Expect(containers).To(HaveLen(2))

			managerContainer, ok := containers[0].(map[string]interface{})
			Expect(ok).To(BeTrue())
			Expect(managerContainer["name"]).To(Equal("manager"))
			Expect(managerContainer["image"]).To(Equal(sapBtpOperatorImage))

			// Check kube-rbac-proxy container
			proxyContainer, ok := containers[1].(map[string]interface{})
			Expect(ok).To(BeTrue())
			Expect(proxyContainer["name"]).To(Equal("kube-rbac-proxy"))
			Expect(proxyContainer["image"]).To(Equal(kubeRbacProxyImage))
		})

		It("should return error if container not found", func() {
			manager.config.SapBtpOperatorImage = "test-image"

			deployment := &unstructured.Unstructured{}
			deployment.SetKind("Deployment")
			deployment.SetName("test-deployment")
			deployment.Object["spec"] = map[string]interface{}{
				"template": map[string]interface{}{
					"spec": map[string]interface{}{
						"containers": []interface{}{
							map[string]interface{}{
								"name": "wrong-container-name",
							},
						},
					},
				},
			}

			err := manager.setDeploymentImages(deployment)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("container manager not found"))
		})
	})

	Describe("apply or update resources", func() {
		var (
			ctx context.Context
		)

		BeforeEach(func() {
			ctx = context.Background()
		})

		It("should create new resources using Server-Side Apply", func() {
			objects, err := manager.createUnstructuredObjectsFromManifestsDir("./testdata")
			Expect(err).NotTo(HaveOccurred())

			err = manager.applyOrUpdateResources(ctx, objects)
			Expect(err).NotTo(HaveOccurred())

			configMapIndex := manager.resourceIndices[ModuleResource{Kind: "ConfigMap", Name: "test-configmap"}]
			configMap := &unstructured.Unstructured{}
			configMap.SetGroupVersionKind(objects[configMapIndex].GroupVersionKind())
			err = fakeClient.Get(ctx, client.ObjectKey{
				Name:      "test-configmap",
				Namespace: "test-namespace",
			}, configMap)
			Expect(err).NotTo(HaveOccurred())
			Expect(configMap.GetName()).To(Equal("test-configmap"))
		})

		It("should update existing resources", func() {
			configMap := &unstructured.Unstructured{}
			configMap.SetGroupVersionKind(schema.GroupVersionKind{
				Group:   "",
				Version: "v1",
				Kind:    "ConfigMap",
			})
			configMap.SetName("test-configmap")
			configMap.SetNamespace("test-namespace")
			unstructured.SetNestedField(configMap.Object, "old-value", "data", "key")

			err := fakeClient.Create(ctx, configMap)
			Expect(err).NotTo(HaveOccurred())

			objects, err := manager.createUnstructuredObjectsFromManifestsDir("./testdata")
			Expect(err).NotTo(HaveOccurred())

			err = manager.applyOrUpdateResources(ctx, objects)
			Expect(err).NotTo(HaveOccurred())

			updated := &unstructured.Unstructured{}
			updated.SetGroupVersionKind(schema.GroupVersionKind{
				Group:   "",
				Version: "v1",
				Kind:    "ConfigMap",
			})
			err = fakeClient.Get(ctx, client.ObjectKey{
				Name:      "test-configmap",
				Namespace: "test-namespace",
			}, updated)
			Expect(err).NotTo(HaveOccurred())

			data, found, err := unstructured.NestedString(updated.Object, "data", "key")
			Expect(err).NotTo(HaveOccurred())
			Expect(found).To(BeTrue())
			Expect(data).To(Equal("value"))
		})
	})

	Describe("delete resources", func() {
		var (
			ctx context.Context
		)

		BeforeEach(func() {
			ctx = context.Background()
		})

		It("should delete existing resources", func() {
			configMap := &unstructured.Unstructured{}
			configMap.SetGroupVersionKind(schema.GroupVersionKind{
				Group:   "",
				Version: "v1",
				Kind:    "ConfigMap",
			})
			configMap.SetName("test-configmap")
			configMap.SetNamespace("test-namespace")

			err := fakeClient.Create(ctx, configMap)
			Expect(err).NotTo(HaveOccurred())

			err = fakeClient.Get(ctx, client.ObjectKey{
				Name:      "test-configmap",
				Namespace: "test-namespace",
			}, configMap)
			Expect(err).NotTo(HaveOccurred())

			objects, err := manager.createUnstructuredObjectsFromManifestsDir("./testdata")
			Expect(err).NotTo(HaveOccurred())

			configMapIndex := manager.resourceIndices[ModuleResource{Kind: "ConfigMap", Name: "test-configmap"}]
			err = manager.deleteResources(ctx, []*unstructured.Unstructured{objects[configMapIndex]})
			Expect(err).NotTo(HaveOccurred())

			err = fakeClient.Get(ctx, client.ObjectKey{
				Name:      "test-configmap",
				Namespace: "test-namespace",
			}, configMap)
			Expect(k8serrors.IsNotFound(err)).To(BeTrue())
		})

		It("should not error when deleting non-existent resources", func() {
			objects, err := manager.createUnstructuredObjectsFromManifestsDir("./testdata")
			Expect(err).NotTo(HaveOccurred())

			err = manager.deleteResources(ctx, objects)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("wait for resources readiness", func() {
		It("should successfully wait for ready Deployment", func() {
			ctx := context.Background()
			deployment := &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "apps/v1",
					"kind":       "Deployment",
					"metadata": map[string]interface{}{
						"name":      "test-deployment",
						"namespace": "test-namespace",
					},
					"spec": map[string]interface{}{
						"replicas": 2,
					},
					"status": map[string]interface{}{
						"replicas":      2,
						"readyReplicas": 2,
					},
				},
			}

			err := fakeClient.Create(ctx, deployment)
			Expect(err).NotTo(HaveOccurred())

			objects := []*unstructured.Unstructured{deployment}
			err = manager.waitForResourcesReadiness(ctx, objects, 5*time.Second)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should timeout when Deployment is not ready", func() {
			ctx := context.Background()
			deployment := &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "apps/v1",
					"kind":       "Deployment",
					"metadata": map[string]interface{}{
						"name":      "test-deployment",
						"namespace": "test-namespace",
					},
					"spec": map[string]interface{}{
						"replicas": 2,
					},
					"status": map[string]interface{}{
						"replicas":      2,
						"readyReplicas": 1,
					},
				},
			}

			err := fakeClient.Create(ctx, deployment)
			Expect(err).NotTo(HaveOccurred())

			objects := []*unstructured.Unstructured{deployment}
			err = manager.waitForResourcesReadiness(ctx, objects, 1*time.Second)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("timeout"))
		})

		It("should consider ConfigMap as immediately ready", func() {
			ctx := context.Background()
			configMap := &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "v1",
					"kind":       "ConfigMap",
					"metadata": map[string]interface{}{
						"name":      "test-configmap",
						"namespace": "test-namespace",
					},
				},
			}

			err := fakeClient.Create(ctx, configMap)
			Expect(err).NotTo(HaveOccurred())

			objects := []*unstructured.Unstructured{configMap}
			err = manager.waitForResourcesReadiness(ctx, objects, 5*time.Second)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should handle multiple resources concurrently", func() {
			ctx := context.Background()
			deployment := &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "apps/v1",
					"kind":       "Deployment",
					"metadata": map[string]interface{}{
						"name":      "test-deployment",
						"namespace": "test-namespace",
					},
					"spec": map[string]interface{}{
						"replicas": 1,
					},
					"status": map[string]interface{}{
						"replicas":      1,
						"readyReplicas": 1,
					},
				},
			}

			configMap := &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "v1",
					"kind":       "ConfigMap",
					"metadata": map[string]interface{}{
						"name":      "test-configmap",
						"namespace": "test-namespace",
					},
				},
			}

			err := fakeClient.Create(ctx, deployment)
			Expect(err).NotTo(HaveOccurred())
			err = fakeClient.Create(ctx, configMap)
			Expect(err).NotTo(HaveOccurred())

			objects := []*unstructured.Unstructured{deployment, configMap}
			err = manager.waitForResourcesReadiness(ctx, objects, 5*time.Second)
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
