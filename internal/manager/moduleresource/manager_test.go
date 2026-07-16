package moduleresource

import (
	"context"
	"encoding/base64"
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

	moduleResourcesPath         = "./testdata"
	moduleResourcesPathToApply  = moduleResourcesPath + "/apply"
	moduleResourcesPathToDelete = moduleResourcesPath + "/delete"

	requiredSecretName      = "sap-btp-manager"
	requiredSecretNamespace = "kyma-system"
)

var (
	fakeClient          client.Client
	scheme              *runtime.Scheme
	defaultStubDetector *stubCredentialsProvider
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

	defaultStubDetector = &stubCredentialsProvider{
		credentialsNamespace: kymaNamespace,
		clusterId:            "",
	}
})

var _ = Describe("Module Resource Manager", func() {
	var manager *Manager

	BeforeEach(func() {
		fakeClient = fake.NewClientBuilder().
			WithScheme(scheme).
			Build()

		manager = NewManager(fakeClient, scheme, defaultStubDetector)
	})

	Describe("create unstructured objects from manifests directory", func() {
		It("should load and convert manifests to unstructured objects", func() {
			objects, err := manager.CreateUnstructuredObjectsFromManifestsDir(moduleResourcesPathToApply)

			Expect(err).NotTo(HaveOccurred())
			Expect(objects).To(HaveLen(3))

			configmap := findByKindAndName(objects, configmapKind, configmapName)
			Expect(configmap).NotTo(BeNil())
			Expect(configmap.GetNamespace()).To(Equal(testNamespace))

			deployment := findByKindAndName(objects, DeploymentKind, deploymentName)
			Expect(deployment).NotTo(BeNil())
			Expect(deployment.GetNamespace()).To(Equal(testNamespace))

			secret := findByKindAndName(objects, secretKind, secretName)
			Expect(secret).NotTo(BeNil())
			Expect(secret.GetNamespace()).To(Equal(testNamespace))
		})

		It("should return error for non-existent directory", func() {
			_, err := manager.CreateUnstructuredObjectsFromManifestsDir("./non-existent")

			Expect(err).To(HaveOccurred())
		})
	})

	Describe("add labels", func() {
		It("should add managed-by, chart version, and module labels to all resources", func() {
			objects, err := manager.CreateUnstructuredObjectsFromManifestsDir(moduleResourcesPathToApply)
			Expect(err).NotTo(HaveOccurred())

			chartVersion := "0.0.1"
			Expect(manager.AddLabels(chartVersion, objects...)).NotTo(HaveOccurred())

			for _, obj := range objects {
				labels := obj.GetLabels()
				Expect(labels).To(HaveKey(ManagedByLabelKey))
				Expect(labels[ManagedByLabelKey]).To(Equal(OperatorName))
				Expect(labels[KymaProjectModuleLabelKey]).To(Equal(ModuleName))
				Expect(labels[ChartVersionLabelKey]).To(Equal(chartVersion))
			}

			deployment := findByKindAndName(objects, DeploymentKind, deploymentName)
			Expect(deployment).NotTo(BeNil())
			spec, found, err := unstructured.NestedMap(deployment.Object, "spec", "template", "metadata", "labels")
			Expect(err).NotTo(HaveOccurred())
			Expect(found).To(BeTrue())
			Expect(spec[KymaProjectModuleLabelKey]).To(Equal(ModuleName))
		})
	})

	Describe("set namespace", func() {
		It("should set namespace in all resources", func() {
			objects, err := manager.CreateUnstructuredObjectsFromManifestsDir(moduleResourcesPathToApply)
			Expect(err).NotTo(HaveOccurred())

			manager.SetNamespace(objects)

			for _, obj := range objects {
				Expect(obj.GetNamespace()).To(Equal(kymaNamespace))
			}
		})
	})

	Describe("set ConfigMap values", func() {
		It("should set ConfigMap values from the credentials provider", func() {
			const (
				expectedClusterId               = "test-cluster-123"
				expectedCredentialsNamespace    = "test-creds-ns"
				expectedEnableLimitedCacheValue = "true"
			)

			stub := &stubCredentialsProvider{
				credentialsNamespace: expectedCredentialsNamespace,
				clusterId:            expectedClusterId,
			}
			manager = NewManager(fakeClient, scheme, stub)

			restoreEnableLimitedCache := config.EnableLimitedCache
			config.EnableLimitedCache = expectedEnableLimitedCacheValue
			defer func() {
				config.EnableLimitedCache = restoreEnableLimitedCache
			}()

			objects, err := manager.CreateUnstructuredObjectsFromManifestsDir(moduleResourcesPathToApply)
			Expect(err).NotTo(HaveOccurred())

			configmap := findByKindAndName(objects, configmapKind, configmapName)
			Expect(configmap).NotTo(BeNil())

			err = manager.SetConfigMapValues(configmap)
			Expect(err).NotTo(HaveOccurred())

			data, found, err := unstructured.NestedStringMap(configmap.Object, "data")
			Expect(err).NotTo(HaveOccurred())
			Expect(found).To(BeTrue())
			Expect(data[clusterIdConfigMapKey]).To(Equal(expectedClusterId))
			Expect(data[releaseNamespaceConfigMapKey]).To(Equal(expectedCredentialsNamespace))
			Expect(data[managementNamespaceConfigMapKey]).To(Equal(expectedCredentialsNamespace))
			Expect(data[enableLimitedCacheConfigMapKey]).To(Equal(expectedEnableLimitedCacheValue))
		})
	})

	Describe("set default credentials secret values", func() {
		It("should copy data with base64 encoding excluding cluster_id and credentials_namespace", func() {
			const expectedCredentialsNamespace = "credentials-namespace"
			secret := requiredSecret()

			stub := &stubCredentialsProvider{credentialsNamespace: expectedCredentialsNamespace}
			manager = NewManager(fakeClient, scheme, stub)

			secretObj := &unstructured.Unstructured{}
			secretObj.SetKind(secretKind)
			secretObj.SetName(SapBtpServiceOperatorName)
			secretObj.SetNamespace(kymaNamespace)

			Expect(manager.SetSecretValues(secret, secretObj)).NotTo(HaveOccurred())
			Expect(secretObj.GetNamespace()).To(Equal(expectedCredentialsNamespace))

			data, found, err := unstructured.NestedStringMap(secretObj.Object, "data")
			Expect(err).NotTo(HaveOccurred())
			Expect(found).To(BeTrue())

			Expect(data).To(HaveKeyWithValue(ClientIdSecretKey, base64.StdEncoding.EncodeToString(secret.Data[ClientIdSecretKey])))
			Expect(data).To(HaveKeyWithValue(ClientSecretKey, base64.StdEncoding.EncodeToString(secret.Data[ClientSecretKey])))
			Expect(data).To(HaveKeyWithValue(SmUrlSecretKey, base64.StdEncoding.EncodeToString(secret.Data[SmUrlSecretKey])))
			Expect(data).To(HaveKeyWithValue(TokenUrlSecretKey, base64.StdEncoding.EncodeToString(secret.Data[TokenUrlSecretKey])))
			Expect(data).NotTo(HaveKey(ClusterIdSecretKey))
			Expect(data).NotTo(HaveKey(CredentialsNamespaceSecretKey))
		})
	})

	Describe("set Deployment images", func() {
		const (
			sapBtpOperatorImage = "local.test/kyma-project/sap-btp-operator:v0.0.1"
		)

		BeforeEach(func() {
			Expect(os.Setenv(SapBtpServiceOperatorEnv, sapBtpOperatorImage)).To(Succeed())
		})

		AfterEach(func() {
			Expect(os.Unsetenv(SapBtpServiceOperatorEnv)).To(Succeed())
		})

		It("should set container image for sap-btp-service-operator", func() {
			objects, err := manager.CreateUnstructuredObjectsFromManifestsDir(moduleResourcesPathToApply)
			Expect(err).NotTo(HaveOccurred())

			deployment := findByKindAndName(objects, DeploymentKind, deploymentName)
			Expect(deployment).NotTo(BeNil())

			err = manager.SetDeploymentImages(deployment)
			Expect(err).NotTo(HaveOccurred())

			containers, found, err := unstructured.NestedSlice(deployment.Object, "spec", "template", "spec", "containers")
			Expect(err).NotTo(HaveOccurred())
			Expect(found).To(BeTrue())

			managerContainer, ok := containers[0].(map[string]interface{})
			Expect(ok).To(BeTrue())
			Expect(managerContainer["name"]).To(Equal("manager"))
			Expect(managerContainer["image"]).To(Equal(sapBtpOperatorImage))
		})

		It("should return error if container not found", func() {
			deployment := unstructuredDeployment(false, false)
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

			err := manager.SetDeploymentImages(deployment)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("container manager not found"))
		})
	})

	Describe("module resources management", func() {
		var ctx context.Context
		var savedModuleResourcesPath string

		BeforeEach(func() {
			ctx = context.Background()
			savedModuleResourcesPath = config.ResourcesPath
			config.ResourcesPath = moduleResourcesPath
		})

		AfterEach(func() {
			config.ResourcesPath = savedModuleResourcesPath
		})

		Describe("apply or update resources", func() {

			It("should create new resources", func() {
				objects, err := manager.CreateUnstructuredObjectsFromManifestsDir(manager.GetResourcesToApplyPath())
				Expect(err).NotTo(HaveOccurred())
				err = manager.ApplyOrUpdateResources(ctx, objects)
				Expect(err).NotTo(HaveOccurred())

				configmap := &corev1.ConfigMap{}
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

				objects, err := manager.CreateUnstructuredObjectsFromManifestsDir(manager.GetResourcesToApplyPath())
				Expect(err).NotTo(HaveOccurred())
				err = manager.ApplyOrUpdateResources(ctx, objects)
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

				Expect(manager.DeleteOutdatedResources(context.Background())).Should(Succeed())

				err = fakeClient.Get(ctx, client.ObjectKey{
					Name:      configmapName,
					Namespace: testNamespace,
				}, configmap)
				Expect(k8serrors.IsNotFound(err)).To(BeTrue())
			})

			It("should not error when deleting non-existent resources", func() {
				Expect(manager.DeleteOutdatedResources(context.Background())).Should(Succeed())
			})

			It("should return error when unable to delete resources", func() {
				const (
					expectedName1 = "configmap1"
					expectedName2 = "configmap2"
				)
				client := &errorOnDeleteClient{fakeClient}
				manager = NewManager(client, scheme, defaultStubDetector)

				us1 := &unstructured.Unstructured{}
				us1.SetKind(configmapKind)
				us1.SetName(expectedName1)
				us1.SetNamespace(testNamespace)

				us2 := &unstructured.Unstructured{}
				us2.SetKind(configmapKind)
				us2.SetName(expectedName2)
				us2.SetNamespace(testNamespace)

				objects := []*unstructured.Unstructured{us1, us2}

				err := manager.DeleteResources(ctx, objects)
				Expect(err).To(HaveOccurred())

				const errorFormat = "failed to delete %s %s"
				Expect(err.Error()).To(ContainSubstring(fmt.Sprintf(errorFormat, expectedName1, configmapKind)))
				Expect(err.Error()).To(ContainSubstring(fmt.Sprintf(errorFormat, expectedName2, configmapKind)))
			})
		})
	})

	Describe("wait for resources readiness", func() {
		var savedReadyTimeout, savedReadyCheckInterval time.Duration

		BeforeEach(func() {
			savedReadyTimeout, savedReadyCheckInterval = config.ReadyTimeout, config.ReadyCheckInterval
			config.ReadyTimeout = 1 * time.Second
			config.ReadyCheckInterval = 100 * time.Millisecond
		})

		AfterEach(func() {
			config.ReadyTimeout = savedReadyTimeout
			config.ReadyCheckInterval = savedReadyCheckInterval
		})

		It("should successfully wait for Deployment readiness", func() {
			ctx := context.Background()
			deployment := unstructuredDeployment(true, true)

			err := fakeClient.Create(ctx, deployment)
			Expect(err).NotTo(HaveOccurred())

			objects := []*unstructured.Unstructured{deployment}
			err = manager.WaitForResourcesReadiness(ctx, objects)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should timeout when Deployment is not ready", func() {
			ctx := context.Background()
			deployment := unstructuredDeployment(false, false)

			err := fakeClient.Create(ctx, deployment)
			Expect(err).NotTo(HaveOccurred())

			objects := []*unstructured.Unstructured{deployment}
			err = manager.WaitForResourcesReadiness(ctx, objects)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("timeout"))
		})

		It("should consider ConfigMap as immediately ready", func() {
			ctx := context.Background()
			configmap := unstructuredConfigmap()

			err := fakeClient.Create(ctx, configmap)
			Expect(err).NotTo(HaveOccurred())

			objects := []*unstructured.Unstructured{configmap}
			err = manager.WaitForResourcesReadiness(ctx, objects)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should handle multiple resources concurrently", func() {
			ctx := context.Background()
			deployment := unstructuredDeployment(true, true)
			configmap := unstructuredConfigmap()

			err := fakeClient.Create(ctx, deployment)
			Expect(err).NotTo(HaveOccurred())
			err = fakeClient.Create(ctx, configmap)
			Expect(err).NotTo(HaveOccurred())

			objects := []*unstructured.Unstructured{deployment, configmap}
			err = manager.WaitForResourcesReadiness(ctx, objects)
			Expect(err).NotTo(HaveOccurred())
		})
	})
})

func findByKindAndName(objects []*unstructured.Unstructured, kind, name string) *unstructured.Unstructured {
	for _, o := range objects {
		if o.GetKind() == kind && o.GetName() == name {
			return o
		}
	}
	return nil
}

func requiredSecret() *corev1.Secret {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      requiredSecretName,
			Namespace: requiredSecretNamespace,
		},
		Data: map[string][]byte{
			ClientIdSecretKey:  []byte("test_clientid"),
			ClientSecretKey:    []byte("test_clientsecret"),
			SmUrlSecretKey:     []byte("test_sm_url"),
			TokenUrlSecretKey:  []byte("test_tokenurl"),
			ClusterIdSecretKey: []byte("test_cluster_id"),
		},
	}
	return secret
}

type stubCredentialsProvider struct {
	credentialsNamespace string
	clusterId            string
}

func (s *stubCredentialsProvider) CredentialsNamespaceFromManager() string {
	return s.credentialsNamespace
}
func (s *stubCredentialsProvider) ClusterIdFromManager() string { return s.clusterId }

type errorOnDeleteClient struct {
	client.Client
}

func (e *errorOnDeleteClient) Delete(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error {
	return fmt.Errorf("expected delete error")
}

func unstructuredDeployment(available, progressing bool) *unstructured.Unstructured {
	condStatus := func(v bool) string {
		if v {
			return "True"
		}
		return "False"
	}
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "apps/v1",
			"kind":       DeploymentKind,
			"metadata": map[string]interface{}{
				"name":      deploymentName,
				"namespace": testNamespace,
			},
			"spec": map[string]interface{}{
				"replicas": int64(1),
			},
			"status": map[string]interface{}{
				"conditions": []interface{}{
					map[string]interface{}{"type": "Available", "status": condStatus(available)},
					map[string]interface{}{"type": "Progressing", "status": condStatus(progressing)},
				},
			},
		},
	}
}

func unstructuredConfigmap() *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       configmapKind,
			"metadata": map[string]interface{}{
				"name":      configmapName,
				"namespace": testNamespace,
			},
		},
	}
}
