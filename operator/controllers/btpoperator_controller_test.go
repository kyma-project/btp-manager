package controllers

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/kyma-project/btp-manager/operator/api/v1alpha1"
	"github.com/kyma-project/module-manager/operator/pkg/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	schedulingv1 "k8s.io/api/scheduling/v1"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	btpOperatorKind         = "BtpOperator"
	btpOperatorApiVersion   = `operator.kyma-project.io\v1alpha1`
	btpOperatorName         = "btp-operator-test"
	defaultNamespace        = "default"
	kymaNamespace           = "kyma-system"
	instanceName            = "my-service-instance"
	bindingName             = "my-service-binding"
	secretYamlPath          = "testdata/test-secret.yaml"
	priorityClassYamlPath   = "testdata/test-priorityclass.yaml"
	serviceBindingYamlPath  = "testdata/test-servicebinding.yaml"
	serviceInstanceYamlPath = "testdata/test-serviceinstance.yaml"
	k8sOpsTimeout           = time.Second * 3
	k8sOpsPollingInterval   = time.Millisecond * 200
	crStateChangeTimeout    = time.Second * 2
	crStatePollingIntevral  = time.Microsecond * 100
	crDeprovisioningTimeout = time.Second * 10
)

type timeoutK8sClient struct {
	client.Client
}

func newTimeoutK8sClient(c client.Client) *timeoutK8sClient {
	return &timeoutK8sClient{c}
}

func (c *timeoutK8sClient) DeleteAllOf(ctx context.Context, obj client.Object, opts ...client.DeleteAllOfOption) error {
	kind := obj.GetObjectKind().GroupVersionKind().Kind
	if kind == instanceGvk.Kind || kind == bindingGvk.Kind {
		deleteAllOfCtx, cancel := context.WithTimeout(ctx, time.Millisecond*1)
		defer cancel()
		return c.Client.DeleteAllOf(deleteAllOfCtx, obj, opts...)
	}

	return c.Client.DeleteAllOf(ctx, obj, opts...)
}

type errorK8sClient struct {
	client.Client
}

func newErrorK8sClient(c client.Client) *errorK8sClient {
	return &errorK8sClient{c}
}

func (c *errorK8sClient) DeleteAllOf(ctx context.Context, obj client.Object, opts ...client.DeleteAllOfOption) error {
	kind := obj.GetObjectKind().GroupVersionKind().Kind
	if kind == instanceGvk.Kind || kind == bindingGvk.Kind {
		deleteAllOfCtx, cancel := context.WithTimeout(ctx, time.Millisecond*1)
		defer cancel()
		_ = c.Client.DeleteAllOf(deleteAllOfCtx, obj, opts...)
		return errors.New("expected DeleteAllOf error")
	}

	return c.Client.DeleteAllOf(ctx, obj, opts...)
}

var _ = Describe("BTP Operator controller", Ordered, func() {
	var cr *v1alpha1.BtpOperator

	BeforeEach(func() {
		ctx = context.Background()
	})

	Describe("Provisioning", func() {
		BeforeAll(func() {
			pClass := &schedulingv1.PriorityClass{}
			Expect(createK8sResourceFromYaml(pClass, priorityClassYamlPath)).To(Succeed())
			Expect(k8sClient.Create(ctx, pClass)).To(Succeed())

			Expect(k8sClient.Create(ctx, &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: kymaNamespace,
				},
			})).To(Succeed())

			cr = createBtpOperator()
			Expect(k8sClient.Create(ctx, cr)).To(Succeed())
			Eventually(getCurrentCrState).WithTimeout(crStateChangeTimeout).WithPolling(crStatePollingIntevral).Should(Equal(types.StateProcessing))
		})

		AfterAll(func() {
			Expect(k8sClient.Get(ctx, client.ObjectKey{Namespace: defaultNamespace, Name: btpOperatorName}, cr)).To(Succeed())
			Expect(k8sClient.Delete(ctx, cr)).To(Succeed())
			Eventually(getCurrentCrState).WithTimeout(crStateChangeTimeout).WithPolling(crStatePollingIntevral).Should(Equal(types.StateDeleting))
			Eventually(isCrNotFound).WithTimeout(crDeprovisioningTimeout).WithPolling(crStatePollingIntevral).Should(BeTrue())
		})

		When("The required Secret is missing", func() {
			It("should return error while getting the required Secret", func() {
				Eventually(getCurrentCrState).WithTimeout(crStateChangeTimeout).WithPolling(crStatePollingIntevral).Should(Equal(types.StateError))
			})
		})

		Describe("The required Secret exists", func() {
			AfterEach(func() {
				deleteSecret := &corev1.Secret{}
				Expect(k8sClient.Get(ctx, client.ObjectKey{Namespace: kymaNamespace, Name: SecretName}, deleteSecret)).To(Succeed())
				Expect(k8sClient.Delete(ctx, deleteSecret)).To(Succeed())
				Eventually(getCurrentCrState).WithTimeout(crStateChangeTimeout).WithPolling(crStatePollingIntevral).Should(Equal(types.StateError))
			})

			When("the required Secret does not have all required keys", func() {
				It("should return error while verifying keys", func() {
					secret, err := createSecretWithoutKeys()
					Expect(err).To(BeNil())
					Expect(k8sClient.Create(ctx, secret)).To(Succeed())
					Eventually(getCurrentCrState).WithTimeout(crStateChangeTimeout).WithPolling(crStatePollingIntevral).Should(Equal(types.StateProcessing))
					Eventually(getCurrentCrState).WithTimeout(crStateChangeTimeout).WithPolling(crStatePollingIntevral).Should(Equal(types.StateError))
				})
			})

			When("the required Secret's keys do not have all values", func() {
				It("should return error while verifying values", func() {
					secret, err := createSecretWithoutValues()
					Expect(err).To(BeNil())
					Expect(k8sClient.Create(ctx, secret)).Should(Succeed())
					Eventually(getCurrentCrState).WithTimeout(crStateChangeTimeout).WithPolling(crStatePollingIntevral).Should(Equal(types.StateProcessing))
					Eventually(getCurrentCrState).WithTimeout(crStateChangeTimeout).WithPolling(crStatePollingIntevral).Should(Equal(types.StateError))
				})
			})

			When("the required Secret is correct", func() {
				It("should install chart successfully", func() {
					// requires real cluster, envtest doesn't start kube-controller-manager
					// see: https://book.kubebuilder.io/reference/envtest.html#configuring-envtest-for-integration-tests
					//      https://book.kubebuilder.io/reference/envtest.html#testing-considerations
					secret, err := createCorrectSecretFromYaml()
					Expect(err).To(BeNil())
					Eventually(k8sClient.Create(ctx, secret)).Should(Succeed())
					Eventually(getCurrentCrState).WithTimeout(crStateChangeTimeout).WithPolling(crStatePollingIntevral).Should(Equal(types.StateReady))
					btpServiceOperatorDeployment := &appsv1.Deployment{}
					Eventually(k8sClient.Get(ctx, client.ObjectKey{Name: DeploymentName, Namespace: kymaNamespace}, btpServiceOperatorDeployment)).
						WithTimeout(k8sOpsTimeout).
						WithPolling(k8sOpsPollingInterval).
						Should(Succeed())
				})
			})

		})
	})

	Describe("Configurability", func() {
		Context("When the ConfigMap is present", func() {
			It("should adjust configuration settings in the operator accordingly", func() {
				cm := initConfig(map[string]string{"ProcessingStateRequeueInterval": "10s"})
				reconciler.reconcileConfig(cm)
				Expect(ProcessingStateRequeueInterval).To(Equal(time.Second * 10))
			})
		})
	})

	Describe("Deprovisioning", func() {
		var siUnstructured, sbUnstructured *unstructured.Unstructured

		BeforeAll(func() {
			createSecret()
		})

		BeforeEach(func() {
			cr := createBtpOperator()
			Expect(k8sClient.Create(ctx, cr)).To(Succeed())
			Eventually(getCurrentCrState).WithTimeout(crStateChangeTimeout).WithPolling(crStatePollingIntevral).Should(Equal(types.StateReady))

			time.Sleep(time.Millisecond * 500)
			err := clearWebhooks()
			Expect(err).To(BeNil())

			siUnstructured = createResource(instanceGvk, kymaNamespace, instanceName)
			ensureResourceExists(instanceGvk)

			sbUnstructured = createResource(bindingGvk, kymaNamespace, bindingName)
			ensureResourceExists(bindingGvk)
		})

		It("soft delete (after timeout) should succeed", func() {
			reconciler.Client = newTimeoutK8sClient(reconciler.Client)
			setFinalizers(siUnstructured)
			setFinalizers(sbUnstructured)
			Expect(k8sClient.Get(ctx, client.ObjectKey{Namespace: defaultNamespace, Name: btpOperatorName}, cr)).To(Succeed())
			Expect(k8sClient.Delete(ctx, cr)).To(Succeed())
			Eventually(getCurrentCrState).WithTimeout(crStateChangeTimeout).WithPolling(crStatePollingIntevral).Should(Equal(types.StateDeleting))
			Eventually(isCrNotFound).WithTimeout(crDeprovisioningTimeout).WithPolling(crStatePollingIntevral).Should(BeTrue())
			doChecks()
		})

		It("soft delete (after hard deletion fail) should succeed", func() {
			reconciler.Client = newErrorK8sClient(reconciler.Client)
			setFinalizers(siUnstructured)
			setFinalizers(sbUnstructured)
			Expect(k8sClient.Get(ctx, client.ObjectKey{Namespace: defaultNamespace, Name: btpOperatorName}, cr)).To(Succeed())
			Expect(k8sClient.Delete(ctx, cr)).To(Succeed())
			Eventually(getCurrentCrState).WithTimeout(crStateChangeTimeout).WithPolling(crStatePollingIntevral).Should(Equal(types.StateDeleting))
			Eventually(isCrNotFound).WithTimeout(crDeprovisioningTimeout).WithPolling(crStatePollingIntevral).Should(BeTrue())
			doChecks()
		})

		It("hard delete should succeed", func() {
			Expect(k8sClient.Get(ctx, client.ObjectKey{Namespace: defaultNamespace, Name: btpOperatorName}, cr)).To(Succeed())
			Expect(k8sClient.Delete(ctx, cr)).To(Succeed())
			Eventually(getCurrentCrState).WithTimeout(crStateChangeTimeout).WithPolling(crStatePollingIntevral).Should(Equal(types.StateDeleting))
			Eventually(isCrNotFound).WithTimeout(crDeprovisioningTimeout).WithPolling(crStatePollingIntevral).Should(BeTrue())
			doChecks()
		})
	})
})

func setFinalizers(resource *unstructured.Unstructured) {
	finalizers := []string{"test-finalizer"}
	Expect(k8sClient.Get(ctx, client.ObjectKey{Namespace: resource.GetNamespace(), Name: resource.GetName()}, resource)).To(Succeed())
	Expect(unstructured.SetNestedStringSlice(resource.Object, finalizers, "metadata", "finalizers")).To(Succeed())
	Expect(k8sClient.Update(ctx, resource)).To(Succeed())
}

func getCurrentCrState() types.State {
	cr := &v1alpha1.BtpOperator{}
	if err := k8sClient.Get(ctx, client.ObjectKey{Namespace: defaultNamespace, Name: btpOperatorName}, cr); err != nil {
		return ""
	}
	return cr.GetStatus().State
}

func isCrNotFound() bool {
	cr := &v1alpha1.BtpOperator{}
	err := k8sClient.Get(ctx, client.ObjectKey{Namespace: defaultNamespace, Name: btpOperatorName}, cr)
	return k8serrors.IsNotFound(err)
}

func createSecret() {
	namespace := &corev1.Namespace{}
	namespace.Name = kymaNamespace
	err := k8sClient.Get(ctx, client.ObjectKeyFromObject(namespace), namespace)
	if k8serrors.IsNotFound(err) {
		err = k8sClient.Create(ctx, namespace)
	}
	Expect(err).To(BeNil())

	secret := &corev1.Secret{}
	secret.Type = corev1.SecretTypeOpaque
	secret.Name = "sap-btp-manager"
	secret.Namespace = kymaNamespace
	err = k8sClient.Get(ctx, client.ObjectKeyFromObject(secret), secret)
	if k8serrors.IsNotFound(err) {
		secret.Data = map[string][]byte{
			"clientid":     []byte("dGVzdF9jbGllbnRpZA=="),
			"clientsecret": []byte("dGVzdF9jbGllbnRzZWNyZXQ="),
			"sm_url":       []byte("dGVzdF9zbV91cmw="),
			"tokenurl":     []byte("dGVzdF90b2tlbnVybA=="),
			"cluster_id":   []byte("dGVzdF9jbHVzdGVyX2lk"),
		}
		err = k8sClient.Create(ctx, secret)
	}

	Expect(err).To(BeNil())
}

func createBtpOperator() *v1alpha1.BtpOperator {
	return &v1alpha1.BtpOperator{
		TypeMeta: metav1.TypeMeta{
			Kind:       btpOperatorKind,
			APIVersion: btpOperatorApiVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      btpOperatorName,
			Namespace: defaultNamespace,
		},
	}
}

func initConfig(data map[string]string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ConfigName,
			Namespace: ChartNamespace,
		},
		Data: data,
	}
}

func createCorrectSecretFromYaml() (*corev1.Secret, error) {
	secret := &corev1.Secret{}
	data, err := os.ReadFile(secretYamlPath)
	if err != nil {
		return nil, fmt.Errorf("while reading the required Secret YAML: %w", err)
	}
	err = yaml.Unmarshal(data, secret)
	if err != nil {
		return nil, fmt.Errorf("while unmarshalling Secret YAML to struct: %w", err)
	}

	return secret, nil
}

func createSecretWithoutKeys() (*corev1.Secret, error) {
	secret, err := createCorrectSecretFromYaml()
	if err != nil {
		return nil, fmt.Errorf("while creating Secret from YAML: %w", err)
	}
	delete(secret.Data, "cluster_id")
	delete(secret.Data, "clientsecret")

	return secret, nil
}

func createSecretWithoutValues() (*corev1.Secret, error) {
	secret, err := createCorrectSecretFromYaml()
	if err != nil {
		return nil, fmt.Errorf("while creating Secret from YAML: %w", err)
	}
	secret.Data["cluster_id"] = []byte("")
	secret.Data["clientsecret"] = []byte("")

	return secret, nil
}

func createK8sResourceFromYaml[T runtime.Object](resource T, yamlPath string) error {
	data, err := os.ReadFile(yamlPath)
	if err != nil {
		return fmt.Errorf("while reading YAML: %w", err)
	}
	err = yaml.Unmarshal(data, resource)
	if err != nil {
		return fmt.Errorf("while unmarshalling YAML to struct: %w", err)
	}

	return nil
}

func ensureResourceExists(gvk schema.GroupVersionKind) {
	list := &unstructured.UnstructuredList{}
	list.SetGroupVersionKind(gvk)
	err := k8sClient.List(ctx, list)
	Expect(err).To(BeNil())
	Expect(list.Items).To(HaveLen(1))
}

func createResource(gvk schema.GroupVersionKind, namespace string, name string) *unstructured.Unstructured {
	object := &unstructured.Unstructured{}
	object.SetGroupVersionKind(gvk)
	object.SetNamespace(namespace)
	object.SetName(name)
	kind := object.GetObjectKind().GroupVersionKind().Kind
	if kind == instanceGvk.Kind {
		populateServiceInstanceFields(object)
	} else if kind == bindingGvk.Kind {
		populateServiceBindingFields(object)
	}
	err := k8sClient.Create(ctx, object)
	Expect(err).To(BeNil())

	return object
}

func populateServiceInstanceFields(object *unstructured.Unstructured) {
	Expect(unstructured.SetNestedField(object.Object, "test-service", "spec", "serviceOfferingName")).To(Succeed())
	Expect(unstructured.SetNestedField(object.Object, "test-plan", "spec", "servicePlanName")).To(Succeed())
	Expect(unstructured.SetNestedField(object.Object, "test-service-instance-external", "spec", "externalName")).To(Succeed())
}

func populateServiceBindingFields(object *unstructured.Unstructured) {
	Expect(unstructured.SetNestedField(object.Object, "test-service-instance", "spec", "serviceInstanceName")).To(Succeed())
	Expect(unstructured.SetNestedField(object.Object, "test-binding-external", "spec", "externalName")).To(Succeed())
	Expect(unstructured.SetNestedField(object.Object, "test-service-binding-secret", "spec", "secretName")).To(Succeed())
}

func clearWebhooks() error {
	mutatingWebhook := &admissionregistrationv1.MutatingWebhookConfiguration{}
	if err := k8sClient.DeleteAllOf(ctx, mutatingWebhook, labelFilter); err != nil {
		if !k8serrors.IsNotFound(err) {
			return err
		}
	}
	validatingWebhook := &admissionregistrationv1.ValidatingWebhookConfiguration{}
	if err := k8sClient.DeleteAllOf(ctx, validatingWebhook, labelFilter); err != nil {
		if !k8serrors.IsNotFound(err) {
			return err
		}
	}
	return nil
}

func doChecks() {
	checkIfNoServiceExists(btpOperatorServiceBinding)
	checkIfNoBindingSecretExists()
	checkIfNoServiceExists(btpOperatorServiceInstance)
	checkIfNoBtpResourceExists()
}

func checkIfNoServiceExists(kind string) {
	list := unstructured.UnstructuredList{}
	list.SetGroupVersionKind(schema.GroupVersionKind{Version: btpOperatorApiVer, Group: btpOperatorGroup, Kind: kind})
	err := k8sClient.List(ctx, &list)
	Expect(k8serrors.IsNotFound(err)).To(BeTrue())
	Expect(list.Items).To(HaveLen(0))
}

func checkIfNoBindingSecretExists() {
	secret := &corev1.Secret{}
	err := k8sClient.Get(ctx, client.ObjectKey{Name: bindingName, Namespace: ChartNamespace}, secret)
	Expect(*secret).To(BeEquivalentTo(corev1.Secret{}))
	Expect(k8serrors.IsNotFound(err)).To(BeTrue())
}

func checkIfNoBtpResourceExists() {
	cs, err := clientset.NewForConfig(cfg)
	Expect(err).To(BeNil())

	_, resourceMap, err := cs.ServerGroupsAndResources()
	Expect(err).To(BeNil())

	namespaces := &corev1.NamespaceList{}
	err = k8sClient.List(ctx, namespaces)
	Expect(err).To(BeNil())

	found := false
	for _, resource := range resourceMap {
		gv, _ := schema.ParseGroupVersion(resource.GroupVersion)
		for _, apiResource := range resource.APIResources {
			list := &unstructured.UnstructuredList{}
			list.SetGroupVersionKind(schema.GroupVersionKind{
				Version: gv.Version,
				Group:   gv.Group,
				Kind:    apiResource.Kind,
			})
			for _, namespace := range namespaces.Items {
				if err := k8sClient.List(ctx, list, client.InNamespace(namespace.Name), labelFilter); err != nil {
					ignore := k8serrors.IsNotFound(err) || meta.IsNoMatchError(err) || k8serrors.IsMethodNotSupported(err)
					if !ignore {
						found = true
						break
					}
				} else if len(list.Items) > 0 {
					found = true
					break
				}
			}
		}
	}
	Expect(found).To(BeFalse())
}
