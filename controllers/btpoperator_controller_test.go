package controllers

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	appsv1 "k8s.io/api/apps/v1"

	"github.com/kyma-project/btp-manager/api/v1alpha1"
	"github.com/kyma-project/btp-manager/internal/gvksutils"
	"github.com/kyma-project/btp-manager/internal/ymlutils"
	"github.com/kyma-project/module-manager/pkg/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	corev1 "k8s.io/api/core/v1"
	schedulingv1 "k8s.io/api/scheduling/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	apimachienerytypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/yaml"
	controllerruntime "sigs.k8s.io/controller-runtime"
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
	k8sOpsTimeout                   = time.Second * 3
	k8sOpsPollingInterval           = time.Millisecond * 200
	crStateChangeTimeout            = time.Second * 5
	crStateUpdatedTimeout           = time.Second
	crStatePollingInterval          = time.Millisecond * 10
	crStateUpdatedPollingInterval   = time.Millisecond
	crDeprovisioningPollingInterval = time.Second * 1
	crDeprovisioningTimeout         = time.Second * 30
	updatePath                      = "./testdata/module-chart-update"
	suffix                          = "updated"
	defaultChartPath                = "./testdata/test-module-chart"
	newChartVersion                 = "v99"
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
	HardDeleteCheckInterval = 10 * time.Millisecond
	HardDeleteTimeout = 1 * time.Second

	BeforeAll(func() {
		err := createPrereqs()
		Expect(err).To(BeNil())
		Expect(testChartPreparation()).To(Succeed())
		ChartPath = defaultChartPath
		reconciler.updateCheckDone = true
	})

	AfterAll(func() {
		Expect(testChartCleanup()).To(Succeed())
	})

	BeforeEach(func() {
		ctx = context.Background()
	})

	Describe("Provisioning", func() {
		BeforeEach(func() {
			cr = createBtpOperator()
			Eventually(func() error { return k8sClient.Create(ctx, cr) }).WithTimeout(k8sOpsTimeout).WithPolling(k8sOpsPollingInterval).Should(Succeed())
		})

		AfterEach(func() {
			cr = &v1alpha1.BtpOperator{}
			Eventually(func() error {
				return k8sClient.Get(ctx, client.ObjectKey{Namespace: defaultNamespace, Name: btpOperatorName}, cr)
			}).
				WithTimeout(k8sOpsTimeout).
				WithPolling(k8sOpsPollingInterval).
				Should(Succeed())
			Eventually(func() error { return k8sClient.Delete(ctx, cr) }).WithTimeout(k8sOpsTimeout).WithPolling(k8sOpsPollingInterval).Should(Succeed())
			Eventually(isCrNotFound).WithTimeout(crDeprovisioningTimeout).WithPolling(crDeprovisioningPollingInterval).Should(BeTrue())
		})

		When("The required Secret is missing", func() {
			It("should return error while getting the required Secret", func() {
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
										MatchFields(IgnoreExtras, Fields{"Type": Equal(ReadyType), "Reason": Equal(string(MissingSecret)), "Status": Equal(metav1.ConditionFalse)}),
									))),
						))
			})
		})

		Describe("The required Secret exists", func() {
			AfterEach(func() {
				deleteSecret := &corev1.Secret{}
				Eventually(func() error {
					return k8sClient.Get(ctx, client.ObjectKey{Namespace: kymaNamespace, Name: SecretName}, deleteSecret)
				}).
					WithTimeout(k8sOpsTimeout).
					WithPolling(k8sOpsPollingInterval).
					Should(Succeed())
				Eventually(func() error { return k8sClient.Delete(ctx, deleteSecret) }).WithTimeout(k8sOpsTimeout).WithPolling(k8sOpsPollingInterval).Should(Succeed())
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
										MatchFields(IgnoreExtras, Fields{"Type": Equal(ReadyType), "Reason": Equal(string(MissingSecret)), "Status": Equal(metav1.ConditionFalse)}),
									))),
						))
			})

			When("the required Secret does not have all required keys", func() {
				It("should return error while verifying keys", func() {
					secret, err := createSecretWithoutKeys()
					Expect(err).To(BeNil())
					Eventually(func() error { return k8sClient.Create(ctx, secret) }).WithTimeout(k8sOpsTimeout).WithPolling(k8sOpsPollingInterval).Should(Succeed())
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
					Eventually(func() error { return k8sClient.Create(ctx, secret) }).WithTimeout(k8sOpsTimeout).WithPolling(k8sOpsPollingInterval).Should(Succeed())
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

			When("the required Secret is correct", func() {
				It("should install chart successfully", func() {
					// requires real cluster, envtest doesn't start kube-controller-manager
					// see: https://book.kubebuilder.io/reference/envtest.html#configuring-envtest-for-integration-tests
					//      https://book.kubebuilder.io/reference/envtest.html#testing-considerations
					secret, err := createCorrectSecretFromYaml()
					Expect(err).To(BeNil())
					Eventually(func() error { return k8sClient.Create(ctx, secret) }).WithTimeout(k8sOpsTimeout).WithPolling(k8sOpsPollingInterval).Should(Succeed())
					Eventually(getCurrentCrStatus).
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
					Eventually(func() error {
						return k8sClient.Get(ctx, client.ObjectKey{Name: DeploymentName, Namespace: kymaNamespace}, btpServiceOperatorDeployment)
					}).
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

		BeforeEach(func() {
			secret, err := createCorrectSecretFromYaml()
			Expect(err).To(BeNil())
			Eventually(func() error { return k8sClient.Create(ctx, secret) }).WithTimeout(k8sOpsTimeout).WithPolling(k8sOpsPollingInterval).Should(Succeed())
			cr := createBtpOperator()
			Eventually(func() error { return k8sClient.Create(ctx, cr) }).WithTimeout(k8sOpsTimeout).WithPolling(k8sOpsPollingInterval).Should(Succeed())
			Eventually(getCurrentCrState).WithTimeout(crStateChangeTimeout).WithPolling(crStatePollingInterval).Should(Equal(types.StateReady))
			btpServiceOperatorDeployment := &appsv1.Deployment{}
			Eventually(func() error {
				return k8sClient.Get(ctx, client.ObjectKey{Name: DeploymentName, Namespace: kymaNamespace}, btpServiceOperatorDeployment)
			}).
				WithTimeout(k8sOpsTimeout).
				WithPolling(k8sOpsPollingInterval).
				Should(Succeed())

			siUnstructured = createResource(instanceGvk, kymaNamespace, instanceName)
			ensureResourceExists(instanceGvk)

			sbUnstructured = createResource(bindingGvk, kymaNamespace, bindingName)
			ensureResourceExists(bindingGvk)
		})

		AfterEach(func() {
			deleteSecret := &corev1.Secret{}
			Eventually(func() error {
				return k8sClient.Get(ctx, client.ObjectKey{Namespace: kymaNamespace, Name: SecretName}, deleteSecret)
			}).
				WithTimeout(k8sOpsTimeout).
				WithPolling(k8sOpsPollingInterval).
				Should(Succeed())
			Eventually(func() error { return k8sClient.Delete(ctx, deleteSecret) }).WithTimeout(k8sOpsTimeout).WithPolling(k8sOpsPollingInterval).Should(Succeed())
		})

		It("soft delete (after timeout) should succeed", func() {
			reconciler.Client = newTimeoutK8sClient(reconciler.Client)
			setFinalizers(siUnstructured)
			setFinalizers(sbUnstructured)
			cr = &v1alpha1.BtpOperator{}
			Eventually(func() error {
				return k8sClient.Get(ctx, client.ObjectKey{Namespace: defaultNamespace, Name: btpOperatorName}, cr)
			}).
				WithTimeout(k8sOpsTimeout).
				WithPolling(k8sOpsPollingInterval).
				Should(Succeed())
			Eventually(func() error { return k8sClient.Delete(ctx, cr) }).WithTimeout(k8sOpsTimeout).WithPolling(k8sOpsPollingInterval).Should(Succeed())
			Eventually(getCurrentCrStatus).
				WithTimeout(crStateChangeTimeout).
				WithPolling(crStatePollingInterval).
				Should(
					SatisfyAll(
						HaveField("State", types.StateDeleting),
						HaveField("Conditions", HaveLen(1)),
						HaveField("Conditions",
							ContainElements(
								PointTo(
									MatchFields(IgnoreExtras, Fields{"Type": Equal(ReadyType), "Reason": Equal(string(HardDeleting)), "Status": Equal(metav1.ConditionFalse)}),
								))),
					))
			Eventually(getCurrentCrStatus).
				WithTimeout(crStateChangeTimeout).
				WithPolling(crStatePollingInterval).
				Should(
					SatisfyAll(
						HaveField("State", types.StateDeleting),
						HaveField("Conditions", HaveLen(1)),
						HaveField("Conditions",
							ContainElements(
								PointTo(
									MatchFields(IgnoreExtras, Fields{"Type": Equal(ReadyType), "Reason": Equal(string(SoftDeleting)), "Status": Equal(metav1.ConditionFalse)}),
								))),
					))
			Eventually(isCrNotFound).WithTimeout(crDeprovisioningTimeout).WithPolling(crDeprovisioningPollingInterval).Should(BeTrue())
			doChecks()
		})

		It("soft delete (after hard deletion fail) should succeed", func() {
			reconciler.Client = newErrorK8sClient(reconciler.Client)
			setFinalizers(siUnstructured)
			setFinalizers(sbUnstructured)
			Eventually(func() error {
				return k8sClient.Get(ctx, client.ObjectKey{Namespace: defaultNamespace, Name: btpOperatorName}, cr)
			}).
				WithTimeout(k8sOpsTimeout).
				WithPolling(k8sOpsPollingInterval).
				Should(Succeed())
			Eventually(func() error { return k8sClient.Delete(ctx, cr) }).WithTimeout(k8sOpsTimeout).WithPolling(k8sOpsPollingInterval).Should(Succeed())
			Eventually(getCurrentCrStatus).
				WithTimeout(crStateChangeTimeout).
				WithPolling(crStatePollingInterval).
				Should(
					SatisfyAll(
						HaveField("State", types.StateDeleting),
						HaveField("Conditions", HaveLen(1)),
						HaveField("Conditions",
							ContainElements(
								PointTo(
									MatchFields(IgnoreExtras, Fields{"Type": Equal(ReadyType), "Reason": Equal(string(SoftDeleting)), "Status": Equal(metav1.ConditionFalse)}),
								))),
					))
			Eventually(isCrNotFound).WithTimeout(crDeprovisioningTimeout).WithPolling(crDeprovisioningPollingInterval).Should(BeTrue())
			doChecks()
		})

		It("hard delete should succeed", func() {
			reconciler.Client = k8sClientFromManager
			Eventually(func() error {
				return k8sClient.Get(ctx, client.ObjectKey{Namespace: defaultNamespace, Name: btpOperatorName}, cr)
			}).
				WithTimeout(k8sOpsTimeout).
				WithPolling(k8sOpsPollingInterval).
				Should(Succeed())
			Eventually(func() error { return k8sClient.Delete(ctx, cr) }).WithTimeout(k8sOpsTimeout).WithPolling(k8sOpsPollingInterval).Should(Succeed())
			Eventually(getCurrentCrStatus).
				WithTimeout(crStateChangeTimeout).
				WithPolling(crStatePollingInterval).
				Should(
					SatisfyAll(
						HaveField("State", types.StateDeleting),
						HaveField("Conditions", HaveLen(1)),
						HaveField("Conditions",
							ContainElements(
								PointTo(
									MatchFields(IgnoreExtras, Fields{"Type": Equal(ReadyType), "Reason": Equal(string(HardDeleting)), "Status": Equal(metav1.ConditionFalse)}),
								))),
					))
			Eventually(isCrNotFound).WithTimeout(crDeprovisioningTimeout).WithPolling(crDeprovisioningPollingInterval).Should(BeTrue())
			doChecks()
		})
	})

	Describe("Update", func() {
		var initChartVersion string
		var minimalExpectedElementsCount int
		var gvks []schema.GroupVersionKind

		BeforeAll(func() {
			copyChartAndSetPath()

			err := createPrereqs()
			Expect(err).To(BeNil())

			secret, err := createCorrectSecretFromYaml()
			Expect(err).To(BeNil())
			Eventually(func() error { return k8sClient.Create(ctx, secret) }).WithTimeout(k8sOpsTimeout).WithPolling(k8sOpsPollingInterval).Should(Succeed())

			cr = createBtpOperator()
			Expect(k8sClient.Create(ctx, cr)).To(Succeed())
			Eventually(getCurrentCrState).WithTimeout(crStateChangeTimeout).WithPolling(crStatePollingInterval).Should(Equal(types.StateProcessing))
			Eventually(getCurrentCrState).WithTimeout(crStateChangeTimeout).WithPolling(crStatePollingInterval).Should(Equal(types.StateReady))

			initChartVersion, err = ymlutils.ExtractStringValueFromYamlForGivenKey(fmt.Sprintf("%s/Chart.yaml", ChartPath), "version")
			Expect(err).To(BeNil())

			gvks, err = ymlutils.GatherChartGvks(defaultChartPath)
			Expect(err).To(BeNil())
			minimalExpectedElementsCount = len(gvks)

			revertToOriginalChart()
		})

		BeforeEach(func() {
			copyChartAndSetPath()

			simulateRestart(ctx, cr)

			Expect(pullFromBtpManagerConfigMap(oldChartVersionKey)).To(Equal(initChartVersion))
			Expect(pullFromBtpManagerConfigMap(currentCharVersionKey)).To(Equal(initChartVersion))
		})

		When("update of all resources names and bump chart version", Label("test-update"), func() {
			It("new resources (with new name) should be created and old ones removed", func() {
				err := ymlutils.TransformCharts(updatePath, suffix)
				Expect(err).To(BeNil())

				err = ymlutils.UpdateVersion(updatePath, newChartVersion)
				Expect(err).To(BeNil())

				simulateRestart(ctx, cr)

				Expect(pullFromBtpManagerConfigMap(oldChartVersionKey)).To(Equal(initChartVersion))
				Expect(pullFromBtpManagerConfigMap(currentCharVersionKey)).To(Equal(newChartVersion))

				oldCount, newCount := countResources()
				Expect(oldCount).To(BeZero())
				Expect(newCount >= minimalExpectedElementsCount).To(BeTrue())

				Eventually(getCurrentCrStatus).
					WithTimeout(crStateChangeTimeout).
					WithPolling(crStatePollingInterval).
					Should(
						SatisfyAll(
							HaveField("State", types.StateReady),
							HaveField("Conditions", HaveLen(1)),
							HaveField("Conditions",
								ContainElements(
									PointTo(
										MatchFields(IgnoreExtras, Fields{"Type": Equal(ReadyType), "Reason": Equal(string(UpdateDone)), "Status": Equal(metav1.ConditionTrue)}),
									))),
						))
			})
		})

		When("update of all resources names and leave same chart version", Label("test-update"), func() {
			It("new ones not created because version is not changed, old ones should stay", Label("test-update"), func() {
				err := ymlutils.TransformCharts(updatePath, suffix)
				Expect(err).To(BeNil())

				simulateRestart(ctx, cr)

				Expect(pullFromBtpManagerConfigMap(oldChartVersionKey)).To(Equal(initChartVersion))
				Expect(pullFromBtpManagerConfigMap(currentCharVersionKey)).To(Equal(initChartVersion))

				oldCount, newCount := countResources()
				Expect(pullFromBtpManagerConfigMap(currentCharVersionKey)).To(Equal(pullFromBtpManagerConfigMap(oldChartVersionKey)))
				Expect(newCount).To(BeEquivalentTo(oldCount))
				Expect(oldCount >= minimalExpectedElementsCount).To(BeTrue())

				Eventually(getCurrentCrStatus).
					WithTimeout(crStateChangeTimeout).
					WithPolling(crStatePollingInterval).
					Should(
						SatisfyAll(
							HaveField("State", types.StateReady),
							HaveField("Conditions", HaveLen(1)),
							HaveField("Conditions",
								ContainElements(
									PointTo(
										MatchFields(IgnoreExtras, Fields{"Type": Equal(ReadyType), "Reason": Equal(string(UpdateCheckSucceeded)), "Status": Equal(metav1.ConditionTrue)}),
									))),
						))
			})
		})

		When("resources should stay as they are and we bump chart version", Label("test-update"), func() {
			It("existing resources has new version set and we delete nothing (check if any resources with old labels exists -> should be 0)", func() {
				err := ymlutils.UpdateVersion(updatePath, newChartVersion)
				Expect(err).To(BeNil())

				oldCount, newCount := countResources()
				Expect(newCount).To(BeEquivalentTo(oldCount))
				Expect(oldCount >= minimalExpectedElementsCount).To(BeTrue())

				simulateRestart(ctx, cr)

				Expect(pullFromBtpManagerConfigMap(oldChartVersionKey)).To(Equal(initChartVersion))
				Expect(pullFromBtpManagerConfigMap(currentCharVersionKey)).To(Equal(newChartVersion))

				oldCount, newCount = countResources()
				Expect(oldCount).To(BeEquivalentTo(0))
				Expect(newCount >= minimalExpectedElementsCount).To(BeTrue())

				Eventually(getCurrentCrStatus).
					WithTimeout(crStateChangeTimeout).
					WithPolling(crStatePollingInterval).
					Should(
						SatisfyAll(
							HaveField("State", types.StateReady),
							HaveField("Conditions", HaveLen(1)),
							HaveField("Conditions",
								ContainElements(
									PointTo(
										MatchFields(IgnoreExtras, Fields{"Type": Equal(ReadyType), "Reason": Equal(string(UpdateDone)), "Status": Equal(metav1.ConditionTrue)}),
									))),
						))
			})
		})

		AfterEach(func() {

			revertToOriginalChart()

			reconciler.updateCheckDone = false
			reconciler.currentVersion = ""

			configMap := reconciler.buildBtpManagerConfigMap()
			err := k8sClient.Delete(ctx, configMap)
			Expect(err).To(BeNil())
		})
	})
})

func copyChartAndSetPath() {
	cmd := exec.Command("cp", "-r", ChartPath, updatePath)
	err := cmd.Run()
	Expect(err).To(BeNil())

	ChartPath = updatePath
}

func revertToOriginalChart() {
	ChartPath = defaultChartPath

	err := os.RemoveAll(updatePath)
	Expect(err).To(BeNil())
}

func pullFromBtpManagerConfigMap(key string) string {
	ctx := context.Background()
	configMap := reconciler.buildBtpManagerConfigMap()
	err := k8sClient.Get(ctx, client.ObjectKeyFromObject(configMap), configMap)
	Expect(err).To(BeNil())
	Expect(configMap).ToNot(BeNil())
	value, ok := configMap.Data[key]
	Expect(ok).To(BeTrue())
	return value
}

func simulateRestart(ctx context.Context, cr *v1alpha1.BtpOperator) {
	reconciler.updateCheckDone = false
	reconciler.currentVersion = ""
	_, err := reconciler.Reconcile(ctx, controllerruntime.Request{NamespacedName: apimachienerytypes.NamespacedName{
		Namespace: cr.Namespace,
		Name:      cr.Name,
	}})
	Expect(err).To(BeNil())
	Eventually(getCurrentCrState).WithTimeout(crStateChangeTimeout).WithPolling(crStatePollingInterval).Should(Equal(types.StateReady))
}

func countResources() (int, int) {
	oldVersion := pullFromBtpManagerConfigMap(oldChartVersionKey)
	oldVersionLabel := client.MatchingLabels{chartVersionKey: oldVersion}
	oldGvksText := pullFromBtpManagerConfigMap(oldGvksKey)
	oldGvks, err := gvksutils.StrToGvks(oldGvksText)
	Expect(err).To(BeNil())
	oldCount := countWithLabel(oldVersionLabel, oldGvks)

	currentVersion := pullFromBtpManagerConfigMap(currentCharVersionKey)
	currentVersionLabel := client.MatchingLabels{chartVersionKey: currentVersion}
	currentGvksText := pullFromBtpManagerConfigMap(currentGvksKey)
	currentGvks, err := gvksutils.StrToGvks(currentGvksText)
	Expect(err).To(BeNil())
	newCount := countWithLabel(currentVersionLabel, currentGvks)

	if oldVersion == currentVersion {
		fmt.Printf("versions are equal (%s), with count = {%d} \n", oldVersion, oldCount)
	} else {
		fmt.Printf("oldCount = {%d}, newCount = {%d} \n", oldCount, newCount)
	}

	return oldCount, newCount
}

func countWithLabel(label client.MatchingLabels, gvks []schema.GroupVersionKind) int {
	count := 0
	for _, gvk := range gvks {
		list := &unstructured.UnstructuredList{}
		list.SetGroupVersionKind(schema.GroupVersionKind{
			Group:   gvk.Group,
			Version: gvk.Version,
			Kind:    gvk.Kind,
		})
		if err := k8sClient.List(ctx, list, label); err != nil && !canIgnoreErr(err) {
			Expect(err).To(BeNil())
		} else {
			count += len(list.Items)
		}
	}
	return count
}

func createPrereqs() error {
	pClass := &schedulingv1.PriorityClass{}
	Expect(createK8sResourceFromYaml(pClass, priorityClassYamlPath)).To(Succeed())
	if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(pClass), pClass); err != nil {
		if k8serrors.IsNotFound(err) {
			Eventually(func() error { return k8sClient.Create(ctx, pClass) }).WithTimeout(k8sOpsTimeout).WithPolling(k8sOpsPollingInterval).Should(Succeed())
		} else {
			return err
		}
	}

	kymaNs := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: kymaNamespace}}
	if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(kymaNs), kymaNs); err != nil {
		if k8serrors.IsNotFound(err) {
			Eventually(func() error { return k8sClient.Create(ctx, kymaNs) }).WithTimeout(k8sOpsTimeout).WithPolling(k8sOpsPollingInterval).Should(Succeed())
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
	Expect(k8sClient.Create(ctx, object)).To(BeNil())

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

func filterWebhooks(file []byte) (filtered []byte, hasWebhook bool) {
	lines := strings.Split(string(file), "\n")
	var buffer []byte
	isWebhook := false
	for _, l := range lines {
		buffer = append(buffer, []byte(l+"\n")...)
		if l == "---" && len(buffer) != 0 {
			if isWebhook {
				split := strings.Split(string(buffer), "\n")
				// hack for one case where helm templating block spans across two adjacent documents
				for _, spl := range split {
					splTrunc := strings.ReplaceAll(spl, " ", "")
					splTrunc = strings.ReplaceAll(splTrunc, "-", "")
					if splTrunc != "{{end}}" {
						break
					}
					filtered = append(filtered, []byte(spl+"\n")...)
				}
			} else {
				filtered = append(filtered, buffer...)
			}
			buffer = []byte{}
			isWebhook = false
		}
		if strings.HasPrefix(l, "kind: ") {
			split := strings.Split(l, ":")
			if len(split) != 2 {
				continue
			}
			kind := strings.TrimLeft(split[1], " ")
			if kind == "MutatingWebhookConfiguration" || kind == "ValidatingWebhookConfiguration" {
				isWebhook, hasWebhook = true, true
			}
		}
	}
	if !isWebhook {
		filtered = append(filtered, buffer...)
	}
	return
}

func testChartPreparation() error {
	Expect(os.MkdirAll(defaultChartPath, 0700)).To(Succeed())
	src := fmt.Sprintf("%v/.", ChartPath)
	cmd := exec.Command("cp", "-r", src, defaultChartPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("copying: %v -> %v\n\nout: %v\nerr: %v", src, defaultChartPath, string(out), err)
	}
	filterWebhooksDisabled := os.Getenv("DISABLE_WEBHOOK_FILTER_FOR_TESTS")

	return filepath.WalkDir(defaultChartPath, func(path string, de fs.DirEntry, err error) error {
		if filterWebhooksDisabled == "true" {
			return nil
		}
		if de.IsDir() {
			return nil
		}
		if err != nil {
			return err
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".yaml" && ext != ".yml" {
			return nil
		}
		dat, err := os.ReadFile(path)
		Expect(err).To(BeNil())

		documents, filtered := filterWebhooks(dat)
		if len(documents) == 0 {
			Expect(os.Remove(path)).To(Succeed())
		}
		if filtered {
			Expect(os.WriteFile(path, documents, 0700)).To(Succeed())
		}
		return nil
	})
}

func testChartCleanup() error {
	return os.RemoveAll(defaultChartPath)
}

func doChecks() {
	var wg sync.WaitGroup
	wg.Add(4)

	go func() {
		defer wg.Done()
		checkIfNoServiceExists(btpOperatorServiceBinding)
	}()
	go func() {
		defer wg.Done()
		checkIfNoBindingSecretExists()
	}()
	go func() {
		defer wg.Done()
		checkIfNoServiceExists(btpOperatorServiceInstance)
	}()
	go func() {
		defer wg.Done()
		checkIfNoBtpResourceExists()
	}()
	wg.Wait()
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
	gvks, err := ymlutils.GatherChartGvks(ChartPath)
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
			if !canIgnoreErr(err) {
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

func canIgnoreErr(err error) bool {
	return k8serrors.IsNotFound(err) || meta.IsNoMatchError(err) || k8serrors.IsMethodNotSupported(err)
}
