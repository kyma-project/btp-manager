package moduleresource

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/kyma-project/btp-manager/controllers/config"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/kyma-project/btp-manager/api/v1alpha1"
)

const (
	testNamespace = "test-namespace"
	kymaNamespace = "kyma-system"

	configmapKind = "ConfigMap"
	secretKind    = "Secret"

	configmapName  = "test-configmap"
	secretName     = "test-secret"
	deploymentName = "test-deployment"

	moduleResourcesPath = "./testdata"

	requiredSecretName      = "sap-btp-manager"
	requiredSecretNamespace = "kyma-system"
)

var (
	fakeClient client.Client
	scheme     *runtime.Scheme
)

func TestModuleResource(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Module Resource Suite")
}

var _ = BeforeSuite(func() {
	scheme = runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(v1alpha1.AddToScheme(scheme))

	fakeClient = fake.NewClientBuilder().
		WithScheme(scheme).
		Build()
})

var _ = Describe("Module Resource Manager", func() {
	var manager *Manager

	BeforeEach(func() {
		fakeClient = fake.NewClientBuilder().
			WithScheme(scheme).
			Build()

		manager = NewManager(fakeClient, scheme)
	})

	Describe("read required secret", func() {
		It("should return error when required secret does not exist", func() {
			secret, err := manager.getRequiredSecret(context.Background())
			Expect(k8serrors.IsNotFound(err)).To(BeTrue())
			Expect(secret).To(BeNil())
		})

		It("should get required secret", func() {
			Expect(createRequiredSecret(fakeClient)).To(Succeed())
			secret, err := manager.getRequiredSecret(context.Background())

			Expect(err).To(BeNil())
			Expect(secret.Name).To(Equal(requiredSecretName))
			Expect(secret.Namespace).To(Equal(requiredSecretNamespace))
		})

		It("should successfully verify required secret", func() {
			Expect(createRequiredSecret(fakeClient)).To(Succeed())
			secret, err := manager.getRequiredSecret(context.Background())
			Expect(err).To(BeNil())

			Expect(manager.verifySecret(secret)).To(Succeed())
		})

		It("should return error when required secret does not contain required key", func() {
			Expect(createRequiredSecret(fakeClient)).To(Succeed())
			secret, err := manager.getRequiredSecret(context.Background())
			Expect(err).To(BeNil())

			delete(secret.Data, ClientIdSecretKey)
			err = manager.verifySecret(secret)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(ClientIdSecretKey))
		})

		It("should return error when one of keys in required secret does not contain value", func() {
			Expect(createRequiredSecret(fakeClient)).To(Succeed())
			secret, err := manager.getRequiredSecret(context.Background())
			Expect(err).To(BeNil())

			secret.Data[ClientSecretKey] = []byte{}
			err = manager.verifySecret(secret)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(ClientSecretKey))
		})
	})

	Describe("setup credentials context", func() {
		var secret *corev1.Secret

		BeforeEach(func() {
			secret = requiredSecret()
		})

		It("should set default credentials namespace when required secret does not contain credentials_namespace", func() {
			manager.setCredentialsNamespace(secret)

			Expect(manager.credentialsContext.credentialsNamespaceFromSapBtpManagerSecret).To(Equal(kymaNamespace))
		})

		It("should set credentials namespace from required secret", func() {
			const expectedCredentialsNamespace = "new-credentials-namespace"
			secret.Data[CredentialsNamespaceSecretKey] = []byte(expectedCredentialsNamespace)
			manager.setCredentialsNamespace(secret)

			Expect(manager.credentialsContext.credentialsNamespaceFromSapBtpManagerSecret).To(Equal(expectedCredentialsNamespace))
		})

		It("should set credentials ID from required secret", func() {
			const expectedClusterID = "new-credentials-id"
			secret.Data[ClusterIdSecretKey] = []byte(expectedClusterID)
			manager.setClusterID(secret)

			Expect(manager.credentialsContext.clusterIdFromSapBtpManagerSecret).To(Equal(expectedClusterID))
		})
	})

	Describe("create unstructured objects from manifests directory", func() {
		It("should load and convert manifests to unstructured objects", func() {
			objects, err := manager.createUnstructuredObjectsFromManifestsDir(moduleResourcesPath)

			Expect(err).NotTo(HaveOccurred())
			Expect(objects).To(HaveLen(3))
			Expect(manager.resourceIndices).To(HaveLen(3))

			configmapIndex := manager.resourceIndices[Metadata{Kind: configmapKind, Name: configmapName}]
			configmap := objects[configmapIndex]
			Expect(configmap.GetKind()).To(Equal(configmapKind))
			Expect(configmap.GetName()).To(Equal(configmapName))
			Expect(configmap.GetNamespace()).To(Equal(testNamespace))

			deploymentIndex := manager.resourceIndices[Metadata{Kind: DeploymentKind, Name: deploymentName}]
			deployment := objects[deploymentIndex]
			Expect(deployment.GetKind()).To(Equal(DeploymentKind))
			Expect(deployment.GetName()).To(Equal(deploymentName))
			Expect(deployment.GetNamespace()).To(Equal(testNamespace))

			secretIndex := manager.resourceIndices[Metadata{Kind: secretKind, Name: secretName}]
			secret := objects[secretIndex]
			Expect(secret.GetKind()).To(Equal(secretKind))
			Expect(secret.GetName()).To(Equal(secretName))
			Expect(secret.GetNamespace()).To(Equal(testNamespace))
		})

		It("should return error for non-existent directory", func() {
			_, err := manager.createUnstructuredObjectsFromManifestsDir("./non-existent")

			Expect(err).To(HaveOccurred())
		})
	})

	Describe("add labels", func() {
		It("should add managed-by, chart version, and module labels to all resources", func() {
			objects, err := manager.createUnstructuredObjectsFromManifestsDir(moduleResourcesPath)
			Expect(err).NotTo(HaveOccurred())

			chartVersion := "0.0.1"
			Expect(manager.addLabels(chartVersion, objects...)).NotTo(HaveOccurred())

			for _, obj := range objects {
				labels := obj.GetLabels()
				Expect(labels).To(HaveKey(ManagedByLabelKey))
				Expect(labels[ManagedByLabelKey]).To(Equal(OperatorName))
				Expect(labels[KymaProjectModuleLabelKey]).To(Equal(ModuleName))
				Expect(labels[ChartVersionLabelKey]).To(Equal(chartVersion))
			}

			deploymentIndex := manager.resourceIndices[Metadata{Kind: DeploymentKind, Name: deploymentName}]
			spec, found, err := unstructured.NestedMap(objects[deploymentIndex].Object, "spec", "template", "metadata", "labels")
			Expect(err).NotTo(HaveOccurred())
			Expect(found).To(BeTrue())
			Expect(spec[KymaProjectModuleLabelKey]).To(Equal(ModuleName))
		})
	})

	Describe("set namespace", func() {
		It("should set namespace in all resources", func() {
			objects, err := manager.createUnstructuredObjectsFromManifestsDir(moduleResourcesPath)
			Expect(err).NotTo(HaveOccurred())

			manager.setNamespace(objects)

			for _, obj := range objects {
				Expect(obj.GetNamespace()).To(Equal(kymaNamespace))
			}
		})
	})

	Describe("set ConfigMap values", func() {
		It("should set ConfigMap values from Secret", func() {
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secretName,
					Namespace: testNamespace,
				},
				Data: map[string][]byte{
					ClientIdSecretKey:             []byte("test-client"),
					ClusterIdSecretKey:            []byte("test-cluster-123"),
					CredentialsNamespaceSecretKey: []byte("test-creds-ns"),
				},
			}

			restoreEnableLimitedCache := config.EnableLimitedCache
			config.EnableLimitedCache = "true"
			defer func() {
				config.EnableLimitedCache = restoreEnableLimitedCache
			}()

			objects, err := manager.createUnstructuredObjectsFromManifestsDir(moduleResourcesPath)
			Expect(err).NotTo(HaveOccurred())

			configmapIndex, found := manager.resourceIndices[Metadata{Kind: configmapKind, Name: configmapName}]
			Expect(found).To(BeTrue())
			configmap := objects[configmapIndex]

			manager.setCredentialsContext(secret)
			err = manager.setConfigMapValues(secret, configmap)
			Expect(err).NotTo(HaveOccurred())

			data, found, err := unstructured.NestedStringMap(configmap.Object, "data")
			Expect(err).NotTo(HaveOccurred())
			Expect(found).To(BeTrue())
			Expect(data["CLUSTER_ID"]).To(Equal("test-cluster-123"))
			Expect(data["RELEASE_NAMESPACE"]).To(Equal("test-creds-ns"))
			Expect(data["MANAGEMENT_NAMESPACE"]).To(Equal("test-creds-ns"))
			Expect(data["ENABLE_LIMITED_CACHE"]).To(Equal("true"))
		})
	})

	Describe("set default credentials secret values", func() {
		It("should copy data with base64 encoding excluding cluster_id and credentials_namespace", func() {
			const expectedCredentialsNamespace = "credentials-namespace"
			secret := requiredSecret()
			secret.Data[CredentialsNamespaceSecretKey] = []byte(expectedCredentialsNamespace)

			secretObj := &unstructured.Unstructured{}
			secretObj.SetKind(secretKind)
			secretObj.SetName(SapBtpServiceOperatorName)
			secretObj.SetNamespace(kymaNamespace)

			manager.setCredentialsContext(secret)

			Expect(manager.setSecretValues(secret, secretObj)).NotTo(HaveOccurred())
			Expect(secretObj.GetNamespace()).To(Equal(expectedCredentialsNamespace))

			data, found, err := unstructured.NestedStringMap(secretObj.Object, "data")
			Expect(err).NotTo(HaveOccurred())
			Expect(found).To(BeTrue())

			Expect(data).To(HaveKey(ClientIdSecretKey))
			Expect(data).To(HaveKey(ClientSecretKey))
			Expect(data).To(HaveKey(SmUrlSecretKey))
			Expect(data).NotTo(HaveKey(ClusterIdSecretKey))
			Expect(data).NotTo(HaveKey(CredentialsNamespaceSecretKey))
		})
	})

	Describe("set Deployment images", func() {
		const (
			sapBtpOperatorImage = "local.test/kyma-project/sap-btp-operator:v0.0.1"
			kubeRbacProxyImage  = "local.test/kyma-project/kube-rbac-proxy:v0.0.1"
		)

		BeforeEach(func() {
			Expect(os.Setenv(KubeRbacProxyEnv, kubeRbacProxyImage)).To(Succeed())
			Expect(os.Setenv(SapBtpServiceOperatorEnv, sapBtpOperatorImage)).To(Succeed())
		})

		AfterEach(func() {
			Expect(os.Unsetenv(KubeRbacProxyEnv)).To(Succeed())
			Expect(os.Unsetenv(SapBtpServiceOperatorEnv)).To(Succeed())
		})

		It("should set container images for manager and kube-rbac-proxy", func() {
			objects, err := manager.createUnstructuredObjectsFromManifestsDir(moduleResourcesPath)
			Expect(err).NotTo(HaveOccurred())

			deploymentIndex, found := manager.resourceIndices[Metadata{Kind: DeploymentKind, Name: deploymentName}]
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

			proxyContainer, ok := containers[1].(map[string]interface{})
			Expect(ok).To(BeTrue())
			Expect(proxyContainer["name"]).To(Equal("kube-rbac-proxy"))
			Expect(proxyContainer["image"]).To(Equal(kubeRbacProxyImage))
		})

		It("should return error if container not found", func() {
			deployment := &unstructured.Unstructured{}
			deployment.SetKind(DeploymentKind)
			deployment.SetName(deploymentName)
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
		var ctx context.Context

		BeforeEach(func() {
			ctx = context.Background()
		})

		It("should create new resources using Server-Side Apply", func() {
			objects, err := manager.createUnstructuredObjectsFromManifestsDir(moduleResourcesPath)
			Expect(err).NotTo(HaveOccurred())

			err = manager.applyOrUpdateResources(ctx, objects)
			Expect(err).NotTo(HaveOccurred())

			configmapIndex := manager.resourceIndices[Metadata{Kind: configmapKind, Name: configmapName}]
			configmap := &unstructured.Unstructured{}
			configmap.SetGroupVersionKind(objects[configmapIndex].GroupVersionKind())
			err = fakeClient.Get(ctx, client.ObjectKey{
				Name:      configmapName,
				Namespace: testNamespace,
			}, configmap)
			Expect(err).NotTo(HaveOccurred())
			Expect(configmap.GetName()).To(Equal(configmapName))
		})

		It("should update existing resources", func() {
			const (
				expectedKey = "key"
				expectedVal = "value"
			)
			configmap := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      configmapName,
					Namespace: testNamespace,
				},
				Data: map[string]string{
					"old-key": "old-value",
				},
			}

			err := fakeClient.Create(ctx, configmap)
			Expect(err).NotTo(HaveOccurred())

			objects, err := manager.createUnstructuredObjectsFromManifestsDir(moduleResourcesPath)
			Expect(err).NotTo(HaveOccurred())

			err = manager.applyOrUpdateResources(ctx, objects)
			Expect(err).NotTo(HaveOccurred())

			updated := &corev1.ConfigMap{}
			err = fakeClient.Get(ctx, client.ObjectKey{
				Name:      configmapName,
				Namespace: testNamespace,
			}, updated)

			Expect(err).NotTo(HaveOccurred())
			Expect(updated.Data[expectedKey]).To(Equal(expectedVal))
		})
	})

	Describe("delete resources", func() {
		var ctx context.Context

		BeforeEach(func() {
			ctx = context.Background()
		})

		It("should delete existing resources", func() {
			configmap := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      configmapName,
					Namespace: testNamespace,
				},
				Data: map[string]string{
					"foo": "bar",
				},
			}

			err := fakeClient.Create(ctx, configmap)
			Expect(err).NotTo(HaveOccurred())

			err = fakeClient.Get(ctx, client.ObjectKey{
				Name:      configmapName,
				Namespace: testNamespace,
			}, configmap)
			Expect(err).NotTo(HaveOccurred())

			objects, err := manager.createUnstructuredObjectsFromManifestsDir(moduleResourcesPath)
			Expect(err).NotTo(HaveOccurred())

			configmapIndex := manager.resourceIndices[Metadata{Kind: configmapKind, Name: configmapName}]
			err = manager.deleteResources(ctx, []*unstructured.Unstructured{objects[configmapIndex]})
			Expect(err).NotTo(HaveOccurred())

			err = fakeClient.Get(ctx, client.ObjectKey{
				Name:      configmapName,
				Namespace: testNamespace,
			}, configmap)
			Expect(k8serrors.IsNotFound(err)).To(BeTrue())
		})

		It("should not error when deleting non-existent resources", func() {
			objects, err := manager.createUnstructuredObjectsFromManifestsDir(moduleResourcesPath)
			Expect(err).NotTo(HaveOccurred())

			err = manager.deleteResources(ctx, objects)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should return error when unable to delete resources", func() {
			const (
				expectedName1 = "configmap1"
				expectedName2 = "configmap2"
			)
			client := &errorOnDeleteClient{fakeClient}
			manager = NewManager(client, scheme)

			us1 := &unstructured.Unstructured{}
			us1.SetKind(configmapKind)
			us1.SetName(expectedName1)
			us1.SetNamespace(testNamespace)

			us2 := &unstructured.Unstructured{}
			us2.SetKind(configmapKind)
			us2.SetName(expectedName2)
			us2.SetNamespace(testNamespace)

			objects := []*unstructured.Unstructured{us1, us2}

			err := manager.deleteResources(ctx, objects)
			Expect(err).To(HaveOccurred())

			const errorFormat = "failed to delete %s %s"
			Expect(err.Error()).To(ContainSubstring(fmt.Sprintf(errorFormat, expectedName1, configmapKind)))
			Expect(err.Error()).To(ContainSubstring(fmt.Sprintf(errorFormat, expectedName2, configmapKind)))
		})
	})

	Describe("wait for resources readiness", func() {
		It("should successfully wait for Deployment readiness", func() {
			ctx := context.Background()
			deployment := &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "apps/v1",
					"kind":       DeploymentKind,
					"metadata": map[string]interface{}{
						"name":      deploymentName,
						"namespace": testNamespace,
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
					"kind":       DeploymentKind,
					"metadata": map[string]interface{}{
						"name":      deploymentName,
						"namespace": testNamespace,
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
			configmap := &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "v1",
					"kind":       configmapKind,
					"metadata": map[string]interface{}{
						"name":      configmapName,
						"namespace": testNamespace,
					},
				},
			}

			err := fakeClient.Create(ctx, configmap)
			Expect(err).NotTo(HaveOccurred())

			objects := []*unstructured.Unstructured{configmap}
			err = manager.waitForResourcesReadiness(ctx, objects, 5*time.Second)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should handle multiple resources concurrently", func() {
			ctx := context.Background()
			deployment := &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "apps/v1",
					"kind":       DeploymentKind,
					"metadata": map[string]interface{}{
						"name":      deploymentName,
						"namespace": testNamespace,
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

			configmap := &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "v1",
					"kind":       configmapKind,
					"metadata": map[string]interface{}{
						"name":      configmapName,
						"namespace": testNamespace,
					},
				},
			}

			err := fakeClient.Create(ctx, deployment)
			Expect(err).NotTo(HaveOccurred())
			err = fakeClient.Create(ctx, configmap)
			Expect(err).NotTo(HaveOccurred())

			objects := []*unstructured.Unstructured{deployment, configmap}
			err = manager.waitForResourcesReadiness(ctx, objects, 5*time.Second)
			Expect(err).NotTo(HaveOccurred())
		})
	})
})

func createRequiredSecret(k8sClient client.Client) error {
	return k8sClient.Create(context.Background(), requiredSecret())
}

func requiredSecret() *corev1.Secret {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      requiredSecretName,
			Namespace: requiredSecretNamespace,
		},
		Data: map[string][]byte{
			ClientIdSecretKey:  []byte("dGVzdF9jbGllbnRpZA=="),
			ClientSecretKey:    []byte("dGVzdF9jbGllbnRzZWNyZXQ="),
			SmUrlSecretKey:     []byte("dGVzdF9zbV91cmw="),
			TokenUrlSecretKey:  []byte("dGVzdF90b2tlbnVybA=="),
			ClusterIdSecretKey: []byte("dGVzdF9jbHVzdGVyX2lk"),
		},
	}
	return secret
}

type errorOnDeleteClient struct {
	client.Client
}

func (e *errorOnDeleteClient) Delete(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error {
	return fmt.Errorf("expected delete error")
}
