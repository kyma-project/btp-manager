package controllers

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/kyma-project/btp-manager/operator/api/v1alpha1"
	ymlutils "github.com/kyma-project/btp-manager/operator/internal"
	"github.com/kyma-project/module-manager/operator/pkg/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	cp "github.com/otiai10/copy"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	schedulingv1 "k8s.io/api/scheduling/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	btpOperatorKind          = "BtpOperator"
	btpOperatorApiVersion    = `operator.kyma-project.io\v1alpha1`
	btpOperatorName          = "btp-operator-test"
	testNamespace            = "default"
	instanceName             = "my-service-instance"
	bindingName              = "my-binding"
	kymaNamespace            = "kyma-system"
	secretYamlPath           = "testdata/test-secret.yaml"
	priorityClassYamlPath    = "testdata/test-priorityclass.yaml"
	testTimeout              = time.Second * 10
	stateChangeTimeout       = time.Second * 1
	deleteTimeout            = time.Second * 30
	crStatePollingIntevral   = time.Microsecond * 1
	operationPollingInterval = time.Second * 1
	updatePath               = "./testdata/module-chart-update"
	suffix                   = "updated"
)

type fakeK8s struct {
	client.Client
}

func newFakeK8s(c client.Client) *fakeK8s {
	return &fakeK8s{c}
}

func (f *fakeK8s) DeleteAllOf(ctx context.Context, obj client.Object, opts ...client.DeleteAllOfOption) error {
	deleteAllOfCtx, cancel := context.WithTimeout(ctx, time.Second*1)
	defer cancel()
	if err := f.Client.DeleteAllOf(deleteAllOfCtx, obj, opts...); err != nil {
		return err
	}
	return nil
}

/*
func (f *fakeK8s) DeleteAllOf(ctx context.Context, obj client.Object, opts ...client.DeleteAllOfOption) error {
	kind := obj.GetObjectKind().GroupVersionKind().Kind
	if kind == instanceGvk.Kind || kind == bindingGvk.Kind {
		return errors.New("expected DeleteAllOf error")
	}

	return f.Client.DeleteAllOf(ctx, obj, opts...)
}
*/

var _ = Describe("BTP Operator controller", Ordered, func() {
	var cr *v1alpha1.BtpOperator
	BeforeEach(func() {
		ctx = context.Background()
		cr = createBtpOperator()
	})

	Describe("Provisioning", func() {
		BeforeAll(func() {
			provisionBtpOperatorWithinNeededResources(cr, true, true)
		})

		AfterAll(func() {
			Expect(k8sClient.Get(ctx, client.ObjectKey{Namespace: testNamespace, Name: btpOperatorName}, cr)).To(Succeed())
			Expect(k8sClient.Delete(ctx, cr)).To(Succeed())
			Eventually(getCurrentCrState).WithTimeout(stateChangeTimeout).WithPolling(crStatePollingIntevral).Should(Equal(types.StateDeleting))
			Eventually(isCrNotFound).WithTimeout(deleteTimeout).WithPolling(crStatePollingIntevral).Should(BeTrue())
		})

		When("The required Secret is missing", func() {
			It("should return error while getting the required Secret", func() {
				Eventually(getCurrentCrState).WithTimeout(stateChangeTimeout).WithPolling(crStatePollingIntevral).Should(Equal(types.StateError))
			})
		})

		Describe("The required Secret exists", func() {
			AfterEach(func() {
				deleteSecret := &corev1.Secret{}
				Expect(k8sClient.Get(ctx, client.ObjectKey{Namespace: kymaNamespace, Name: secretName}, deleteSecret)).To(Succeed())
				Expect(k8sClient.Delete(ctx, deleteSecret)).To(Succeed())
				Eventually(getCurrentCrState).WithTimeout(stateChangeTimeout).WithPolling(crStatePollingIntevral).Should(Equal(types.StateError))
			})

			When("the required Secret does not have all required keys", func() {
				It("should return error while verifying keys", func() {
					secret, err := createSecretWithoutKeys()
					Expect(err).To(BeNil())
					Expect(k8sClient.Create(ctx, secret)).To(Succeed())
					Eventually(getCurrentCrState).WithTimeout(stateChangeTimeout).WithPolling(crStatePollingIntevral).Should(Equal(types.StateProcessing))
					Eventually(getCurrentCrState).WithTimeout(stateChangeTimeout).WithPolling(crStatePollingIntevral).Should(Equal(types.StateError))
				})
			})

			When("the required Secret's keys do not have all values", func() {
				It("should return error while verifying values", func() {
					secret, err := createSecretWithoutValues()
					Expect(err).To(BeNil())
					Expect(k8sClient.Create(ctx, secret)).Should(Succeed())
					Eventually(getCurrentCrState).WithTimeout(stateChangeTimeout).WithPolling(crStatePollingIntevral).Should(Equal(types.StateProcessing))
					Eventually(getCurrentCrState).WithTimeout(stateChangeTimeout).WithPolling(crStatePollingIntevral).Should(Equal(types.StateError))
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
					Eventually(getCurrentCrState).WithTimeout(stateChangeTimeout).WithPolling(crStatePollingIntevral).Should(Equal(types.StateProcessing))
					Eventually(getCurrentCrState).WithTimeout(stateChangeTimeout).WithPolling(crStatePollingIntevral).Should(Equal(types.StateReady))
					btpServiceOperatorDeployment := &appsv1.Deployment{}
					Eventually(k8sClient.Get(ctx, client.ObjectKey{Name: deploymentName, Namespace: kymaNamespace}, btpServiceOperatorDeployment)).
						WithTimeout(testTimeout).
						WithPolling(operationPollingInterval).
						Should(Succeed())
				})
			})

		})
	})

	Describe("Deprovisioning", func() {
		var gvks []schema.GroupVersionKind
		BeforeAll(func() {
			createSecret()
			var err error
			gvks, err = ymlutils.GatherChartGvks(reconciler.chartDetails.chartPath)
			Expect(err).To(BeNil())
		})

		BeforeEach(func() {
			provisionBtpOperatorWithinNeededResources(cr, false, false)
			Eventually(getCurrentCrState).WithTimeout(stateChangeTimeout).WithPolling(crStatePollingIntevral).Should(Equal(types.StateReady))

			// required for correct webhooks deletion
			time.Sleep(time.Second * 1)
			err := clearWebhooks()
			Expect(err).To(BeNil())

			createResource(instanceGvk, testNamespace, instanceName)
			ensureResourceExists(instanceGvk)

			createResource(bindingGvk, testNamespace, bindingName)
			ensureResourceExists(bindingGvk)
		})

		It("soft delete (after timeout) should succeed", func() {
			Expect(k8sClient.Get(ctx, client.ObjectKey{Namespace: testNamespace, Name: btpOperatorName}, cr)).To(Succeed())
			Expect(k8sClient.Delete(ctx, cr)).To(Succeed())
			Eventually(getCurrentCrState).WithTimeout(stateChangeTimeout).WithPolling(crStatePollingIntevral).Should(Equal(types.StateDeleting))
			Eventually(isCrNotFound).WithTimeout(deleteTimeout).WithPolling(crStatePollingIntevral).Should(BeTrue())
			doChecks(gvks)
		})

		It("soft delete (after hard deletion fail) should succeed", func() {
			reconciler.Client = newFakeK8s(reconciler.Client)

			Expect(k8sClient.Get(ctx, client.ObjectKey{Namespace: testNamespace, Name: btpOperatorName}, cr)).To(Succeed())
			Expect(k8sClient.Delete(ctx, cr)).To(Succeed())
			Eventually(getCurrentCrState).WithTimeout(stateChangeTimeout).WithPolling(crStatePollingIntevral).Should(Equal(types.StateDeleting))
			Eventually(isCrNotFound).WithTimeout(deleteTimeout).WithPolling(crStatePollingIntevral).Should(BeTrue())
			doChecks(gvks)
		})

		It("hard delete should succeed", func() {
			Expect(k8sClient.Get(ctx, client.ObjectKey{Namespace: testNamespace, Name: btpOperatorName}, cr)).To(Succeed())
			Expect(k8sClient.Delete(ctx, cr)).To(Succeed())
			Eventually(getCurrentCrState).WithTimeout(stateChangeTimeout).WithPolling(crStatePollingIntevral).Should(Equal(types.StateDeleting))
			Eventually(isCrNotFound).WithTimeout(deleteTimeout).WithPolling(crStatePollingIntevral).Should(BeTrue())
			doChecks(gvks)
		})
	})

	Describe("Update", func() {
		onStart := func() {
			err := cp.Copy(reconciler.chartDetails.chartPath, updatePath)
			Expect(err).To(BeNil())
			reconciler.chartDetails.chartPath = updatePath
		}

		onClose := func() {
			reconciler.chartDetails.chartPath = chartPath
			os.RemoveAll(updatePath)
		}

		BeforeAll(func() {
			onStart()
			createSecret()
			provisionBtpOperatorWithinNeededResources(cr, false, false)
			Eventually(getCurrentCrState).WithTimeout(stateChangeTimeout).WithPolling(crStatePollingIntevral).Should(Equal(types.StateReady))

			btpServiceOperatorDeployment := &appsv1.Deployment{}
			Eventually(k8sClient.Get(ctx, client.ObjectKey{Name: deploymentName, Namespace: kymaNamespace}, btpServiceOperatorDeployment)).
				WithTimeout(testTimeout).
				WithPolling(operationPollingInterval).
				Should(Succeed())
		})

		//Consider using labels
		When("update of all resources names and bump chart version", func() {
			It("new resources (with new name) should be created and old ones removed", func() {
				defer onClose()

				gvks, err := ymlutils.GatherChartGvks(updatePath)
				Expect(err).To(BeNil())

				err = ymlutils.TransformCharts(updatePath, suffix)
				Expect(err).To(BeNil())

				err = k8sClient.Get(ctx, client.ObjectKey{Namespace: testNamespace, Name: btpOperatorName}, cr)
				if cr.Annotations == nil {
					cr.Annotations = make(map[string]string)
				}
				cr.Annotations["kubectl.kubernetes.io/restartedAt"] = time.Now().Format(time.RFC3339)
				err = k8sClient.Update(ctx, cr)

				//This should be uncommented when we implement Update in controller, for now the State is Ready, since there were no change, due to missing update feature
				//Eventually(getCurrentCrState).WithTimeout(stateChangeTimeout).WithPolling(1 * time.Nanosecond).Should(Equal(types.StateProcessing))
				Eventually(getCurrentCrState).WithTimeout(stateChangeTimeout).WithPolling(crStatePollingIntevral).Should(Equal(types.StateReady))

				withSuffixCount := 0
				withoutSuffixCount := 0
				for _, gvk := range gvks {
					list := &unstructured.UnstructuredList{}
					list.SetGroupVersionKind(schema.GroupVersionKind{
						Group:   gvk.Group,
						Version: gvk.Version,
						Kind:    gvk.Kind,
					})

					if err = k8sClient.List(ctx, list, labelFilter); err != nil && !canIgnoreErr(err) {
						Expect(err).To(BeNil())
					}

					for _, item := range list.Items {
						if strings.HasSuffix(item.GetName(), suffix) {
							withSuffixCount++
						} else {
							withoutSuffixCount++
						}
					}
				}

				fmt.Printf("withSuffixCount = {%d}, withoutSuffixCount = {%d} \n", withSuffixCount, withoutSuffixCount)
				Expect(withSuffixCount).To(BeEquivalentTo(0))
				Expect(withoutSuffixCount).To(BeZero())
			})
		})

		//After first tests works:

		//Negative scenario
		//update of all resources names and leave same chart version
		//new resources (with new name) should be created and old ones should stay

		//resources should stay as they are and we bump chart version
		//existing resources has new version set and we delete nothing (check if any resources with old labels exists -> should be 0)

	})
})

func provisionBtpOperatorWithinNeededResources(cr *v1alpha1.BtpOperator, withinPriorityClass bool, withinNamespace bool) {
	if withinPriorityClass {
		pClass, err := createPriorityClassFromYaml()
		Expect(err).To(BeNil())
		Expect(k8sClient.Create(ctx, pClass)).To(Succeed())
	}

	if withinNamespace {
		Expect(k8sClient.Create(ctx, &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: kymaNamespace,
			},
		})).To(Succeed())
	}

	cr = createBtpOperator()
	Expect(k8sClient.Create(ctx, cr)).To(Succeed())
	Eventually(getCurrentCrState).WithTimeout(stateChangeTimeout).WithPolling(crStatePollingIntevral).Should(Equal(types.StateProcessing))
}

func getCurrentCrState() types.State {
	cr := &v1alpha1.BtpOperator{}
	if err := k8sClient.Get(ctx, client.ObjectKey{Namespace: testNamespace, Name: btpOperatorName}, cr); err != nil {
		return ""
	}
	return cr.GetStatus().State
}

func isCrNotFound() bool {
	cr := &v1alpha1.BtpOperator{}
	err := k8sClient.Get(ctx, client.ObjectKey{Namespace: testNamespace, Name: btpOperatorName}, cr)
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
			Namespace: testNamespace,
		},
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

func createPriorityClassFromYaml() (*schedulingv1.PriorityClass, error) {
	pClass := &schedulingv1.PriorityClass{}
	data, err := os.ReadFile(priorityClassYamlPath)
	if err != nil {
		return nil, fmt.Errorf("while reading the required PriorityClass YAML: %w", err)
	}
	err = yaml.Unmarshal(data, pClass)
	if err != nil {
		return nil, fmt.Errorf("while unmarshalling PriorityClass YAML to struct: %w", err)
	}

	return pClass, nil
}

func ensureResourceExists(gvk schema.GroupVersionKind) {
	list := &unstructured.UnstructuredList{}
	list.SetGroupVersionKind(gvk)
	err := k8sClient.List(ctx, list)
	Expect(err).To(BeNil())
	Expect(list.Items).To(HaveLen(1))
}

func createResource(gvk schema.GroupVersionKind, namespace string, name string) {
	object := &unstructured.Unstructured{}
	object.SetGroupVersionKind(gvk)
	object.SetNamespace(namespace)
	object.SetName(name)
	err := k8sClient.Create(ctx, object)
	Expect(err).To(BeNil())
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

func doChecks(gvks []schema.GroupVersionKind) {
	checkIfNoServicesExists(btpOperatorServiceBinding)
	checkIfNoBindingSecretExists()
	checkIfNoServicesExists(btpOperatorServiceInstance)
	checkIfNoBtpResourceExists(gvks)
}

func checkIfNoServicesExists(kind string) {
	list := unstructured.UnstructuredList{}
	list.SetGroupVersionKind(schema.GroupVersionKind{Version: btpOperatorApiVer, Group: btpOperatorGroup, Kind: kind})
	err := k8sClient.List(ctx, &list)
	Expect(k8serrors.IsNotFound(err)).To(BeTrue())
	Expect(list.Items).To(HaveLen(0))
}

func checkIfNoBindingSecretExists() {
	secret := &corev1.Secret{}
	err := k8sClient.Get(ctx, client.ObjectKey{Name: bindingName, Namespace: testNamespace}, secret)
	Expect(*secret).To(BeEquivalentTo(corev1.Secret{}))
	Expect(k8serrors.IsNotFound(err)).To(BeTrue())
}

func checkIfNoBtpResourceExists(gvks []schema.GroupVersionKind) {
	found := false
	for _, gvk := range gvks {
		list := &unstructured.UnstructuredList{}
		list.SetGroupVersionKind(schema.GroupVersionKind{
			Version: gvk.Version,
			Group:   gvk.Group,
			Kind:    gvk.Kind,
		})
		if err := k8sClient.List(ctx, list, labelFilter); err != nil {
			if !canIgnoreErr(err) {
				found = true
				break
			} else if len(list.Items) > 0 {
				found = true
				break
			}
		}
	}
	Expect(found).To(BeFalse())
}

func canIgnoreErr(err error) bool {
	return k8serrors.IsNotFound(err) || meta.IsNoMatchError(err) || k8serrors.IsMethodNotSupported(err)
}
