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
	. "github.com/onsi/gomega/gstruct"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	schedulingv1 "k8s.io/api/scheduling/v1"
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
	btpOperatorKind                 = "BtpOperator"
	btpOperatorApiVersion           = `operator.kyma-project.io\v1alpha1`
	btpOperatorName                 = "btp-operator-test"
	defaultNamespace                = "default"
	kymaNamespace                   = "kyma-system"
	instanceName                    = "my-service-instance"
	bindingName                     = "my-service-binding"
	secretYamlPath                  = "testdata/test-secret.yaml"
	priorityClassYamlPath           = "testdata/test-priorityclass.yaml"
	serviceBindingYamlPath          = "testdata/test-servicebinding.yaml"
	serviceInstanceYamlPath         = "testdata/test-serviceinstance.yaml"
	k8sOpsTimeout                   = time.Second * 3
	k8sOpsPollingInterval           = time.Millisecond * 200
	crStateChangeTimeout            = time.Second * 2
	crStatePollingInterval          = time.Millisecond * 10
	crDeprovisioningPollingInterval = time.Second * 1
	crDeprovisioningTimeout         = time.Second * 30
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
		deleteAllOfCtx, cancel := context.WithTimeout(ctx, time.Millisecond*100)
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
		deleteAllOfCtx, cancel := context.WithTimeout(ctx, time.Millisecond*100)
		defer cancel()
		_ = c.Client.DeleteAllOf(deleteAllOfCtx, obj, opts...)
		return errors.New("expected DeleteAllOf error")
	}

	return c.Client.DeleteAllOf(ctx, obj, opts...)
}

var _ = Describe("BTP Operator controller", Ordered, func() {
	var cr *v1alpha1.BtpOperator

	BeforeAll(func() {
		err := createPrereqs()
		Expect(err).To(BeNil())
	})

	BeforeEach(func() {
		ctx = context.Background()
	})

	Describe("Provisioning", func() {
		BeforeAll(func() {
			cr = createBtpOperator()
			Eventually(k8sClient.Create(ctx, cr)).WithTimeout(k8sOpsTimeout).WithPolling(k8sOpsPollingInterval).Should(Succeed())
			Eventually(getCurrentCrStatus).
				WithTimeout(crStateChangeTimeout).
				WithPolling(crStatePollingInterval).
				Should(
					SatisfyAll(
						HaveField("State", types.StateProcessing),
						HaveField("Conditions", HaveLen(1)),
						HaveField("Conditions",
							ContainElements(
								PointTo(
									MatchFields(IgnoreExtras, Fields{"Type": Equal(ReadyType), "Reason": Equal(string(Initialized)), "Status": Equal(metav1.ConditionFalse)}),
								)))))
		})

		AfterAll(func() {
			Eventually(k8sClient.Get(ctx, client.ObjectKey{Namespace: defaultNamespace, Name: btpOperatorName}, cr)).
				WithTimeout(k8sOpsTimeout).
				WithPolling(k8sOpsPollingInterval).
				Should(Succeed())
			Eventually(k8sClient.Delete(ctx, cr)).WithTimeout(k8sOpsTimeout).WithPolling(k8sOpsPollingInterval).Should(Succeed())
			Eventually(getCurrentCrStatus).
				WithTimeout(crStateChangeTimeout).
				WithPolling(crStatePollingInterval).
				Should(SatisfyAll(HaveField("State", types.StateDeleting), HaveField("Conditions", HaveLen(1))))
			Eventually(isCrNotFound).WithTimeout(crDeprovisioningTimeout).WithPolling(crDeprovisioningPollingInterval).Should(BeTrue())
		})

		When("The required Secret is missing", func() {
			It("should return error while getting the required Secret", func() {
				Eventually(getCurrentCrState).
					WithTimeout(crStateChangeTimeout).
					WithPolling(crStatePollingInterval).
					Should(
						SatisfyAll(
							HaveField("State", types.StateError),
							HaveField("Conditions", HaveLen(1)),
							HaveField("Conditions",
								ContainElements(
									PointTo(
										MatchFields(IgnoreExtras, Fields{"Type": Equal(ReadyType), "Reason": Equal(string(MissingSecret)), "Status": Equal(metav1.ConditionFalse)}),
									))),
						))
			})
		})

		Describe("The required Secret exists", func() {
			AfterEach(func() {
				deleteSecret := &corev1.Secret{}
				Eventually(k8sClient.Get(ctx, client.ObjectKey{Namespace: kymaNamespace, Name: SecretName}, deleteSecret)).
					WithTimeout(k8sOpsTimeout).
					WithPolling(k8sOpsPollingInterval).
					Should(Succeed())
				Eventually(k8sClient.Delete(ctx, deleteSecret)).WithTimeout(k8sOpsTimeout).WithPolling(k8sOpsPollingInterval).Should(Succeed())
				Eventually(getCurrentCrState).
					WithTimeout(crStateChangeTimeout).
					WithPolling(crStatePollingInterval).
					Should(
						SatisfyAll(
							HaveField("State", types.StateError),
							HaveField("Conditions", HaveLen(1)),
							HaveField("Conditions",
								ContainElements(
									PointTo(
										MatchFields(IgnoreExtras, Fields{"Type": Equal(ReadyType), "Reason": Equal(string(MissingSecret)), "Status": Equal(metav1.ConditionFalse)}),
									))),
						))
			})

			When("the required Secret does not have all required keys", func() {
				It("should return error while verifying keys", func() {
					secret, err := createSecretWithoutKeys()
					Expect(err).To(BeNil())
					Eventually(k8sClient.Create(ctx, secret)).WithTimeout(k8sOpsTimeout).WithPolling(k8sOpsPollingInterval).Should(Succeed())
					Eventually(getCurrentCrStatus).
						WithTimeout(crStateChangeTimeout).
						WithPolling(crStatePollingInterval).
						Should(SatisfyAll(HaveField("State", types.StateProcessing), HaveField("Conditions", HaveLen(1))))
					Eventually(getCurrentCrStatus).
						WithTimeout(crStateChangeTimeout).
						WithPolling(crStatePollingInterval).
						Should(
							SatisfyAll(
								HaveField("State", types.StateError),
								HaveField("Conditions", HaveLen(1)),
								HaveField("Conditions",
									ContainElements(
										PointTo(
											MatchFields(IgnoreExtras, Fields{"Type": Equal(ReadyType), "Reason": Equal(string(InvalidSecret)), "Status": Equal(metav1.ConditionFalse)}),
										))),
							))
				})
			})

			When("the required Secret's keys do not have all values", func() {
				It("should return error while verifying values", func() {
					secret, err := createSecretWithoutValues()
					Expect(err).To(BeNil())
					Eventually(k8sClient.Create(ctx, secret)).WithTimeout(k8sOpsTimeout).WithPolling(k8sOpsPollingInterval).Should(Succeed())
					Eventually(getCurrentCrStatus).
						WithTimeout(crStateChangeTimeout).
						WithPolling(crStatePollingInterval).
						Should(SatisfyAll(HaveField("State", types.StateProcessing), HaveField("Conditions", HaveLen(1))))
					Eventually(getCurrentCrState).
						WithTimeout(crStateChangeTimeout).
						WithPolling(crStatePollingInterval).
						Should(
							SatisfyAll(
								HaveField("State", types.StateError),
								HaveField("Conditions", HaveLen(1)),
								HaveField("Conditions",
									ContainElements(
										PointTo(
											MatchFields(IgnoreExtras, Fields{"Type": Equal(ReadyType), "Reason": Equal(string(InvalidSecret)), "Status": Equal(metav1.ConditionFalse)}),
										))),
							))
				})
			})

			When("the required Secret is correct", func() {
				It("should install chart successfully", func() {
					// requires real cluster, envtest doesn't start kube-controller-manager
					// see: https://book.kubebuilder.io/reference/envtest.html#configuring-envtest-for-integration-tests
					//      https://book.kubebuilder.io/reference/envtest.html#testing-considerations
					secret, err := createCorrectSecretFromYaml()
					Expect(err).To(BeNil())
					Eventually(k8sClient.Create(ctx, secret)).WithTimeout(k8sOpsTimeout).WithPolling(k8sOpsPollingInterval).Should(Succeed())
					Eventually(getCurrentCrState).
						WithTimeout(crStateChangeTimeout).
						WithPolling(crStatePollingInterval).
						Should(
							SatisfyAll(
								HaveField("State", types.StateReady),
								HaveField("Conditions", HaveLen(1)),
								HaveField("Conditions",
									ContainElements(
										PointTo(
											MatchFields(IgnoreExtras, Fields{"Type": Equal(ReadyType), "Reason": Equal(string(ReconcileSucceeded)), "Status": Equal(metav1.ConditionTrue)}),
										))),
							))
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
			secret, err := createCorrectSecretFromYaml()
			Expect(err).To(BeNil())
			Eventually(k8sClient.Create(ctx, secret)).WithTimeout(k8sOpsTimeout).WithPolling(k8sOpsPollingInterval).Should(Succeed())
		})

		AfterAll(func() {
			deleteSecret := &corev1.Secret{}
			Eventually(k8sClient.Get(ctx, client.ObjectKey{Namespace: kymaNamespace, Name: SecretName}, deleteSecret)).
				WithTimeout(k8sOpsTimeout).
				WithPolling(k8sOpsPollingInterval).
				Should(Succeed())
			Eventually(k8sClient.Delete(ctx, deleteSecret)).WithTimeout(k8sOpsTimeout).WithPolling(k8sOpsPollingInterval).Should(Succeed())
		})

		BeforeEach(func() {
			cr := createBtpOperator()
			Eventually(k8sClient.Create(ctx, cr)).WithTimeout(k8sOpsTimeout).WithPolling(k8sOpsPollingInterval).Should(Succeed())
			Eventually(getCurrentCrState).WithTimeout(crStateChangeTimeout).WithPolling(crStatePollingInterval).Should(Equal(types.StateReady))
			btpServiceOperatorDeployment := &appsv1.Deployment{}
			Eventually(k8sClient.Get(ctx, client.ObjectKey{Name: DeploymentName, Namespace: kymaNamespace}, btpServiceOperatorDeployment)).
				WithTimeout(k8sOpsTimeout).
				WithPolling(k8sOpsPollingInterval).
				Should(Succeed())

			time.Sleep(time.Second * 1)
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
			Eventually(k8sClient.Get(ctx, client.ObjectKey{Namespace: defaultNamespace, Name: btpOperatorName}, cr)).
				WithTimeout(k8sOpsTimeout).
				WithPolling(k8sOpsPollingInterval).
				Should(Succeed())
			Eventually(k8sClient.Delete(ctx, cr)).WithTimeout(k8sOpsTimeout).WithPolling(k8sOpsPollingInterval).Should(Succeed())
			Eventually(getCurrentCrState).
				WithTimeout(crStateChangeTimeout).
				WithPolling(crStatePollingInterval).
				Should(SatisfyAll(HaveField("State", types.StateDeleting), HaveField("Conditions", HaveLen(1))))
			Eventually(isCrNotFound).WithTimeout(crDeprovisioningTimeout).WithPolling(crDeprovisioningPollingInterval).Should(BeTrue())
			doChecks()
		})

		It("soft delete (after hard deletion fail) should succeed", func() {
			reconciler.Client = newErrorK8sClient(reconciler.Client)
			setFinalizers(siUnstructured)
			setFinalizers(sbUnstructured)
			Eventually(k8sClient.Get(ctx, client.ObjectKey{Namespace: defaultNamespace, Name: btpOperatorName}, cr)).
				WithTimeout(k8sOpsTimeout).
				WithPolling(k8sOpsPollingInterval).
				Should(Succeed())
			Eventually(k8sClient.Delete(ctx, cr)).WithTimeout(k8sOpsTimeout).WithPolling(k8sOpsPollingInterval).Should(Succeed())
			Eventually(getCurrentCrState).
				WithTimeout(crStateChangeTimeout).
				WithPolling(crStatePollingInterval).
				Should(SatisfyAll(HaveField("State", types.StateDeleting), HaveField("Conditions", HaveLen(1))))
			Eventually(isCrNotFound).WithTimeout(crDeprovisioningTimeout).WithPolling(crDeprovisioningPollingInterval).Should(BeTrue())
			doChecks()
		})

		It("hard delete should succeed", func() {
			reconciler.Client = k8sClientFromManager
			Eventually(k8sClient.Get(ctx, client.ObjectKey{Namespace: defaultNamespace, Name: btpOperatorName}, cr)).
				WithTimeout(k8sOpsTimeout).
				WithPolling(k8sOpsPollingInterval).
				Should(Succeed())
			Eventually(k8sClient.Delete(ctx, cr)).WithTimeout(k8sOpsTimeout).WithPolling(k8sOpsPollingInterval).Should(Succeed())
			Eventually(getCurrentCrState).
				WithTimeout(crStateChangeTimeout).
				WithPolling(crStatePollingInterval).
				Should(SatisfyAll(HaveField("State", types.StateDeleting), HaveField("Conditions", HaveLen(1))))
			Eventually(isCrNotFound).WithTimeout(crDeprovisioningTimeout).WithPolling(crDeprovisioningPollingInterval).Should(BeTrue())
			doChecks()
		})
	})
})

func createPrereqs() error {
	pClass := &schedulingv1.PriorityClass{}
	Expect(createK8sResourceFromYaml(pClass, priorityClassYamlPath)).To(Succeed())
	if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(pClass), pClass); err != nil {
		if k8serrors.IsNotFound(err) {
			Eventually(k8sClient.Create(ctx, pClass)).WithTimeout(k8sOpsTimeout).WithPolling(k8sOpsPollingInterval).Should(Succeed())
		} else {
			return err
		}
	}

	kymaNs := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: kymaNamespace}}
	if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(kymaNs), kymaNs); err != nil {
		if k8serrors.IsNotFound(err) {
			Eventually(k8sClient.Create(ctx, kymaNs)).WithTimeout(k8sOpsTimeout).WithPolling(k8sOpsPollingInterval).Should(Succeed())
		} else {
			return err
		}
	}

	return nil
}

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

func getCurrentCrStatus() types.Status {
	cr := &v1alpha1.BtpOperator{}
	if err := k8sClient.Get(ctx, client.ObjectKey{Namespace: defaultNamespace, Name: btpOperatorName}, cr); err != nil {
		return types.Status{}
	}
	return cr.GetStatus()
}

func isCrNotFound() bool {
	cr := &v1alpha1.BtpOperator{}
	err := k8sClient.Get(ctx, client.ObjectKey{Namespace: defaultNamespace, Name: btpOperatorName}, cr)
	return k8serrors.IsNotFound(err)
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
	gvks, err := reconciler.gatherChartGvks()
	Expect(err).To(BeNil())

	found := false
	for _, gvk := range gvks {
		list := &unstructured.UnstructuredList{}
		list.SetGroupVersionKind(schema.GroupVersionKind{
			Version: gvk.Version,
			Group:   gvk.Group,
			Kind:    gvk.Kind,
		})
		if err := k8sClient.List(ctx, list, labelFilter); err != nil {
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
	Expect(found).To(BeFalse())
}
