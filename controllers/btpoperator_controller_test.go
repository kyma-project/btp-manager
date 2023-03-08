package controllers

import (
	"bytes"
	"context"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"github.com/kyma-project/btp-manager/internal/certs"
	"github.com/kyma-project/btp-manager/internal/manifest"
	"io/fs"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"strings"
	"sync"
	"time"

	"github.com/kyma-project/btp-manager/api/v1alpha1"
	"github.com/kyma-project/btp-manager/internal/ymlutils"
	"github.com/kyma-project/module-manager/pkg/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
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
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	btpOperatorKind       = "BtpOperator"
	btpOperatorApiVersion = `operator.kyma-project.io\v1alpha1`
	btpOperatorName       = "btp-operator-test"
	defaultNamespace      = "default"
	kymaNamespace         = "kyma-system"
	instanceName          = "my-service-instance"
	bindingName           = "my-service-binding"
	secretYamlPath        = "testdata/test-secret.yaml"
	priorityClassYamlPath = "testdata/test-priorityclass.yaml"
	k8sOpsTimeout         = time.Second * 3
	k8sOpsPollingInterval = time.Millisecond * 200
	chartUpdatePath       = "./testdata/module-chart-update"
	resourcesUpdatePath   = "./testdata/module-resources-update"
	suffix                = "-updated"
	defaultChartPath      = "./testdata/test-module-chart"
	newChartVersion       = "9.9.9"
	defaultResourcesPath  = "./testdata/test-module-resources"
	testRsaKeyBits        = 512
)

type certificationsTimeOpts struct {
	CaCertificateExpiration time.Duration
	WebhookCertExpiration   time.Duration
	ExpirationBoundary      time.Duration
}

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
		certs.SetRsaKeyBits(testRsaKeyBits)
		err := createPrereqs()
		Expect(err).To(BeNil())
		Expect(createChartOrResourcesCopyWithoutWebhooks(ChartPath, defaultChartPath)).To(Succeed())
		Expect(createChartOrResourcesCopyWithoutWebhooks(ResourcesPath, defaultResourcesPath)).To(Succeed())
		ChartPath = defaultChartPath
		ResourcesPath = defaultResourcesPath
	})

	AfterAll(func() {
		Expect(removeAllFromPath(defaultChartPath)).To(Succeed())
		Expect(removeAllFromPath(defaultResourcesPath)).To(Succeed())
	})

	BeforeEach(func() {
		ctx = context.Background()
	})

	Describe("Provisioning", func() {

		BeforeEach(func() {
			cr = createBtpOperator()
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
			It("should return error while getting the required Secret", func() {
				Eventually(updateCh).Should(Receive(matchReadyCondition(types.StateProcessing, metav1.ConditionFalse, Initialized)))
				Eventually(updateCh).Should(Receive(matchReadyCondition(types.StateError, metav1.ConditionFalse, MissingSecret)))
			})
		})

		Describe("The required Secret exists", func() {
			AfterEach(func() {
				deleteSecret := &corev1.Secret{}
				Expect(k8sClient.Get(ctx, client.ObjectKey{Namespace: kymaNamespace, Name: SecretName}, deleteSecret)).To(Succeed())
				Expect(k8sClient.Delete(ctx, deleteSecret)).To(Succeed())
				Eventually(updateCh).Should(Receive(matchReadyCondition(types.StateError, metav1.ConditionFalse, MissingSecret)))
			})

			When("the required Secret does not have all required keys", func() {
				It("should return error while verifying keys", func() {
					secret, err := createSecretWithoutKeys()
					Expect(err).To(BeNil())
					Expect(k8sClient.Create(ctx, secret)).To(Succeed())
					Eventually(updateCh).Should(Receive(matchReadyCondition(types.StateError, metav1.ConditionFalse, InvalidSecret)))
				})
			})

			When("the required Secret's keys do not have all values", func() {
				It("should return error while verifying values", func() {
					secret, err := createSecretWithoutValues()
					Expect(err).To(BeNil())
					Expect(k8sClient.Create(ctx, secret)).To(Succeed())
					Eventually(updateCh).Should(Receive(matchReadyCondition(types.StateError, metav1.ConditionFalse, InvalidSecret)))
				})
			})

			When("the required Secret is correct", func() {
				It("should install chart successfully", func() {
					// requires real cluster, envtest doesn't start kube-controller-manager
					// see: https://book.kubebuilder.io/reference/envtest.html#configuring-envtest-for-integration-tests
					//      https://book.kubebuilder.io/reference/envtest.html#testing-considerations
					secret, err := createCorrectSecretFromYaml()
					Expect(err).To(BeNil())
					Expect(k8sClient.Create(ctx, secret)).To(Succeed())
					Eventually(updateCh).Should(Receive(matchReadyCondition(types.StateReady, metav1.ConditionTrue, ReconcileSucceeded)))
					btpServiceOperatorDeployment := &appsv1.Deployment{}
					Expect(k8sClient.Get(ctx, client.ObjectKey{Name: DeploymentName, Namespace: kymaNamespace}, btpServiceOperatorDeployment)).To(Succeed())
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

	Describe("Deprovisioning without force-delete label", func() {
		var siUnstructured, sbUnstructured *unstructured.Unstructured

		BeforeEach(func() {
			secret, err := createCorrectSecretFromYaml()
			Expect(err).To(BeNil())
			Expect(k8sClient.Create(ctx, secret)).To(Succeed())
			cr = createBtpOperator()
			Expect(k8sClient.Create(ctx, cr)).To(Succeed())
			Eventually(updateCh).Should(Receive(matchState(types.StateReady)))
			btpServiceOperatorDeployment := &appsv1.Deployment{}
			Expect(k8sClient.Get(ctx, client.ObjectKey{Name: DeploymentName, Namespace: kymaNamespace}, btpServiceOperatorDeployment)).Should(Succeed())
		})

		AfterEach(func() {
			deleteSecret := &corev1.Secret{}
			Expect(k8sClient.Get(ctx, client.ObjectKey{Namespace: kymaNamespace, Name: SecretName}, deleteSecret)).To(Succeed())
			Expect(k8sClient.Delete(ctx, deleteSecret)).To(Succeed())
			k8sClient.Get(ctx, client.ObjectKey{Namespace: defaultNamespace, Name: btpOperatorName}, cr)
			cr.SetLabels(map[string]string{forceDeleteLabelKey: "true"})
			k8sClient.Update(ctx, cr)
			time.Sleep(time.Second)
		})

		It("Delete should fail because of existing instances and bindings", func() {
			siUnstructured = createResource(instanceGvk, kymaNamespace, instanceName)
			ensureResourceExists(instanceGvk)

			sbUnstructured = createResource(bindingGvk, kymaNamespace, bindingName)
			ensureResourceExists(bindingGvk)

			reconciler.Client = newTimeoutK8sClient(reconciler.Client)
			setFinalizers(siUnstructured)
			setFinalizers(sbUnstructured)
			Expect(k8sClient.Get(ctx, client.ObjectKey{Namespace: defaultNamespace, Name: btpOperatorName}, cr)).To(Succeed())
			Expect(k8sClient.Delete(ctx, cr)).To(Succeed())

			Eventually(updateCh).Should(Receive(matchReadyCondition(types.StateDeleting, metav1.ConditionFalse, ServiceInstancesAndBindingsNotCleaned)))

			Expect(k8sClient.Delete(ctx, sbUnstructured)).To(Succeed())
			Expect(k8sClient.Delete(ctx, siUnstructured)).To(Succeed())
		})

	})

	Describe("Deprovisioning with force-delete label", func() {
		var siUnstructured, sbUnstructured *unstructured.Unstructured

		BeforeEach(func() {
			secret, err := createCorrectSecretFromYaml()
			Expect(err).To(BeNil())
			Expect(k8sClient.Create(ctx, secret)).To(Succeed())
			cr = createBtpOperator()
			cr.SetLabels(map[string]string{forceDeleteLabelKey: "true"})
			Expect(k8sClient.Create(ctx, cr)).To(Succeed())
			Eventually(updateCh).Should(Receive(matchState(types.StateReady)))
			btpServiceOperatorDeployment := &appsv1.Deployment{}
			Expect(k8sClient.Get(ctx, client.ObjectKey{Name: DeploymentName, Namespace: kymaNamespace}, btpServiceOperatorDeployment)).Should(Succeed())

			siUnstructured = createResource(instanceGvk, kymaNamespace, instanceName)
			ensureResourceExists(instanceGvk)

			sbUnstructured = createResource(bindingGvk, kymaNamespace, bindingName)
			ensureResourceExists(bindingGvk)
		})

		AfterEach(func() {
			deleteSecret := &corev1.Secret{}
			Expect(k8sClient.Get(ctx, client.ObjectKey{Namespace: kymaNamespace, Name: SecretName}, deleteSecret)).To(Succeed())
			Expect(k8sClient.Delete(ctx, deleteSecret)).To(Succeed())
		})

		It("soft delete (after timeout) should succeed", func() {
			reconciler.Client = newTimeoutK8sClient(reconciler.Client)
			setFinalizers(siUnstructured)
			setFinalizers(sbUnstructured)
			Expect(k8sClient.Get(ctx, client.ObjectKey{Namespace: defaultNamespace, Name: btpOperatorName}, cr)).To(Succeed())
			Expect(k8sClient.Delete(ctx, cr)).To(Succeed())
			Eventually(updateCh).Should(Receive(matchReadyCondition(types.StateDeleting, metav1.ConditionFalse, HardDeleting)))
			Eventually(updateCh).Should(Receive(matchReadyCondition(types.StateDeleting, metav1.ConditionFalse, SoftDeleting)))
			Eventually(updateCh).Should(Receive(matchDeleted()))
			doChecks()
		})

		It("soft delete (after hard deletion fail) should succeed", func() {
			reconciler.Client = newErrorK8sClient(reconciler.Client)
			setFinalizers(siUnstructured)
			setFinalizers(sbUnstructured)
			Expect(k8sClient.Get(ctx, client.ObjectKey{Namespace: defaultNamespace, Name: btpOperatorName}, cr)).To(Succeed())
			Expect(k8sClient.Delete(ctx, cr)).To(Succeed())
			Eventually(updateCh).Should(Receive(matchReadyCondition(types.StateDeleting, metav1.ConditionFalse, SoftDeleting)))
			Eventually(updateCh).Should(Receive(matchDeleted()))
			doChecks()
		})

		It("hard delete should succeed", func() {
			reconciler.Client = k8sClientFromManager
			Expect(k8sClient.Get(ctx, client.ObjectKey{Namespace: defaultNamespace, Name: btpOperatorName}, cr)).To(Succeed())
			Expect(k8sClient.Delete(ctx, cr)).Should(Succeed())
			Eventually(updateCh).Should(Receive(matchReadyCondition(types.StateDeleting, metav1.ConditionFalse, HardDeleting)))
			Eventually(updateCh).Should(Receive(matchDeleted()))
			doChecks()
		})
	})

	Describe("Update", func() {
		var initChartVersion string
		var manifestHandler *manifest.Handler
		var initApplyObjs []runtime.Object
		var gvks []schema.GroupVersionKind
		var initResourcesNum int
		var actualWorkqueueSize func() int
		var err error

		BeforeAll(func() {
			err := createPrereqs()
			Expect(err).To(BeNil())

			secret, err := createCorrectSecretFromYaml()
			Expect(err).To(BeNil())
			Expect(k8sClient.Create(ctx, secret)).To(Succeed())

			manifestHandler = &manifest.Handler{Scheme: k8sManager.GetScheme()}
			actualWorkqueueSize = func() int { return reconciler.workqueueSize }
		})

		AfterAll(func() {
			err := os.RemoveAll(chartUpdatePath)
			Expect(err).To(BeNil())
			err = os.RemoveAll(resourcesUpdatePath)
			Expect(err).To(BeNil())

			ChartPath = defaultChartPath
			ResourcesPath = defaultResourcesPath
		})

		BeforeEach(func() {
			cr = createBtpOperator()
			Expect(k8sClient.Create(ctx, cr)).To(Succeed())
			Eventually(updateCh).Should(Receive(matchState(types.StateProcessing)))
			Eventually(updateCh).Should(Receive(matchState(types.StateReady)))

			initChartVersion, err = ymlutils.ExtractStringValueFromYamlForGivenKey(fmt.Sprintf("%s/Chart.yaml", ChartPath), "version")
			Expect(err).To(BeNil())
			_ = initChartVersion

			initApplyObjs, err = manifestHandler.CollectObjectsFromDir(getApplyPath())
			Expect(err).To(BeNil())

			gvks = getUniqueGvksFromObjects(initApplyObjs)

			initResourcesNum, err = countResourcesForGivenChartVer(gvks, initChartVersion)
			Expect(err).To(BeNil())

			copyDirRecursively(ChartPath, chartUpdatePath)
			copyDirRecursively(ResourcesPath, resourcesUpdatePath)
			ChartPath = chartUpdatePath
			ResourcesPath = resourcesUpdatePath
		})

		AfterEach(func() {
			cr = &v1alpha1.BtpOperator{}
			Expect(k8sClient.Get(ctx, client.ObjectKey{Namespace: defaultNamespace, Name: btpOperatorName}, cr)).Should(Succeed())
			Expect(k8sClient.Delete(ctx, cr)).Should(Succeed())
			Eventually(updateCh).Should(Receive(matchState(types.StateReady)))
			Eventually(updateCh).Should(Receive(matchDeleted()))
			Expect(isCrNotFound()).To(BeTrue())

			err := os.RemoveAll(chartUpdatePath)
			Expect(err).To(BeNil())
			err = os.RemoveAll(resourcesUpdatePath)
			Expect(err).To(BeNil())

			ChartPath = defaultChartPath
			ResourcesPath = defaultResourcesPath
		})

		When("update all resources names and bump chart version", Label("test-update"), func() {
			It("new resources (with new names) should be created and old ones removed", func() {
				err := ymlutils.CopyManifestsFromYamlsIntoOneYaml(getApplyPath(), getToDeleteYamlPath())
				Expect(err).To(BeNil())

				err = ymlutils.AddSuffixToNameInManifests(getApplyPath(), suffix)
				Expect(err).To(BeNil())

				err = ymlutils.UpdateChartVersion(chartUpdatePath, newChartVersion)
				Expect(err).To(BeNil())

				Eventually(actualWorkqueueSize).WithTimeout(time.Second * 5).WithPolling(time.Millisecond * 100).Should(Equal(0))
				_, err = reconciler.Reconcile(ctx, controllerruntime.Request{NamespacedName: apimachienerytypes.NamespacedName{
					Namespace: cr.Namespace,
					Name:      cr.Name,
				}})
				Expect(err).To(BeNil())

				actualNumOfOldResources, err := countResourcesForGivenChartVer(gvks, initChartVersion)
				Expect(err).To(BeNil())
				Expect(actualNumOfOldResources).To(Equal(0))
				actualNumOfNewResources, err := countResourcesForGivenChartVer(gvks, newChartVersion)
				Expect(err).To(BeNil())
				Expect(actualNumOfNewResources).To(Equal(initResourcesNum))
			})
		})

		When("update some resources names and bump chart version", Label("test-update"), func() {
			It("all applied resources should receive new chart version, resources with new names should replace the ones with old names", func() {
				updateManifestsNum := 3
				err := moveOrCopyNFilesFromDirToDir(updateManifestsNum, false, getApplyPath(), getTempPath())
				Expect(err).To(BeNil())

				oldObjs, err := manifestHandler.CollectObjectsFromDir(getTempPath())
				Expect(err).To(BeNil())
				oldUns, err := manifestHandler.ObjectsToUnstructured(oldObjs)
				Expect(err).To(BeNil())

				err = ymlutils.CopyManifestsFromYamlsIntoOneYaml(getTempPath(), getToDeleteYamlPath())
				Expect(err).To(BeNil())

				err = ymlutils.AddSuffixToNameInManifests(getTempPath(), suffix)
				Expect(err).To(BeNil())

				err = moveOrCopyNFilesFromDirToDir(updateManifestsNum, true, getTempPath(), getApplyPath())
				Expect(err).To(BeNil())

				err = ymlutils.UpdateChartVersion(chartUpdatePath, newChartVersion)
				Expect(err).To(BeNil())

				Eventually(actualWorkqueueSize).WithTimeout(time.Second * 5).WithPolling(time.Millisecond * 100).Should(Equal(0))
				_, err = reconciler.Reconcile(ctx, controllerruntime.Request{NamespacedName: apimachienerytypes.NamespacedName{
					Namespace: cr.Namespace,
					Name:      cr.Name,
				}})
				Expect(err).To(BeNil())

				actualNumOfOldResources, err := countResourcesForGivenChartVer(gvks, initChartVersion)
				Expect(err).To(BeNil())
				Expect(actualNumOfOldResources).To(Equal(0))
				actualNumOfNewResources, err := countResourcesForGivenChartVer(gvks, newChartVersion)
				Expect(err).To(BeNil())
				Expect(actualNumOfNewResources).To(Equal(initResourcesNum))
				assertResourcesRemoval(oldUns...)
			})
		})

		When("remove some manifests and bump chart version", Label("test-update"), func() {
			It("resources without manifests should be removed, unchanged resources should stay and receive new chart version", func() {
				allManifests, err := manifestHandler.GetManifestsFromDir(getApplyPath())
				Expect(err).To(BeNil())
				err = moveOrCopyNFilesFromDirToDir(len(allManifests), true, getApplyPath(), getTempPath())
				Expect(err).To(BeNil())

				remainingManifestsNum := 4
				err = moveOrCopyNFilesFromDirToDir(remainingManifestsNum, true, getTempPath(), getApplyPath())
				Expect(err).To(BeNil())

				expectedDeleteObjs, err := manifestHandler.CollectObjectsFromDir(getTempPath())
				Expect(err).To(BeNil())
				unexpectedUns, err := manifestHandler.ObjectsToUnstructured(expectedDeleteObjs)
				Expect(err).To(BeNil())

				expectedApplyObjs, err := manifestHandler.CollectObjectsFromDir(getApplyPath())
				Expect(err).To(BeNil())
				expectedUns, err := manifestHandler.ObjectsToUnstructured(expectedApplyObjs)
				Expect(err).To(BeNil())

				err = ymlutils.CopyManifestsFromYamlsIntoOneYaml(getTempPath(), getToDeleteYamlPath())
				Expect(err).To(BeNil())

				err = ymlutils.UpdateChartVersion(chartUpdatePath, newChartVersion)
				Expect(err).To(BeNil())

				Eventually(actualWorkqueueSize).WithTimeout(time.Second * 5).WithPolling(time.Millisecond * 100).Should(Equal(0))
				_, err = reconciler.Reconcile(ctx, controllerruntime.Request{NamespacedName: apimachienerytypes.NamespacedName{
					Namespace: cr.Namespace,
					Name:      cr.Name,
				}})
				Expect(err).To(BeNil())

				actualNumOfOldResources, err := countResourcesForGivenChartVer(gvks, initChartVersion)
				Expect(err).To(BeNil())
				Expect(actualNumOfOldResources).To(Equal(0))
				actualNumOfNewResources, err := countResourcesForGivenChartVer(gvks, newChartVersion)
				Expect(err).To(BeNil())
				Expect(actualNumOfNewResources).To(Equal(len(expectedApplyObjs)))
				assertResourcesExistence(expectedUns...)
				assertResourcesRemoval(unexpectedUns...)
			})
		})

		When("bump chart version only", Label("test-update"), func() {
			It("resources should stay and receive new chart version", func() {
				err = ymlutils.UpdateChartVersion(chartUpdatePath, newChartVersion)
				Expect(err).To(BeNil())

				Eventually(actualWorkqueueSize).WithTimeout(time.Second * 5).WithPolling(time.Millisecond * 100).Should(Equal(0))
				_, err = reconciler.Reconcile(ctx, controllerruntime.Request{NamespacedName: apimachienerytypes.NamespacedName{
					Namespace: cr.Namespace,
					Name:      cr.Name,
				}})
				Expect(err).To(BeNil())

				actualNumOfResourcesWithOldChartVer, err := countResourcesForGivenChartVer(gvks, initChartVersion)
				Expect(err).To(BeNil())
				Expect(actualNumOfResourcesWithOldChartVer).To(Equal(0))
				actualNumOfResourcesWithNewChartVer, err := countResourcesForGivenChartVer(gvks, newChartVersion)
				Expect(err).To(BeNil())
				Expect(actualNumOfResourcesWithNewChartVer).To(Equal(initResourcesNum))
			})
		})
	})

	Describe("Certification management", func() {
		orgCaCertificateExpiration := CaCertificateExpiration
		orgWebhookCertExpiration := WebhookCertificateExpiration
		orgExpirationBoundary := ExpirationBoundary

		var actualWorkqueueSize func() int

		BeforeAll(func() {
			Expect(removeAllFromPath(defaultChartPath)).To(Succeed())
			Expect(removeAllFromPath(defaultResourcesPath)).To(Succeed())

			os.Setenv("DISABLE_WEBHOOK_FILTER_FOR_TESTS", "true")
			Expect(createChartOrResourcesCopyWithoutWebhooks("../module-chart/chart", defaultChartPath)).To(Succeed())
			Expect(createChartOrResourcesCopyWithoutWebhooks("../module-resources", defaultResourcesPath)).To(Succeed())
			ChartPath = defaultChartPath
			ResourcesPath = defaultResourcesPath

			actualWorkqueueSize = func() int { return reconciler.workqueueSize }
		})

		restoreOriginalCertificateTiems := func() {
			CaCertificateExpiration = orgCaCertificateExpiration
			WebhookCertificateExpiration = orgWebhookCertExpiration
			ExpirationBoundary = orgExpirationBoundary
		}

		certBeforeEach := func(opts *certificationsTimeOpts) {
			if opts != nil {
				CaCertificateExpiration = opts.CaCertificateExpiration
				WebhookCertificateExpiration = opts.WebhookCertExpiration
				ExpirationBoundary = opts.ExpirationBoundary
			} else {
				restoreOriginalCertificateTiems()
			}

			cr = createBtpOperator()
			Expect(k8sClient.Create(ctx, cr)).To(Succeed())
			Eventually(updateCh).Should(Receive(matchState(types.StateProcessing)))
			Eventually(updateCh).Should(Receive(matchState(types.StateReady)))
		}

		certAfterEach := func() {
			cr = &v1alpha1.BtpOperator{}
			Expect(k8sClient.Get(ctx, client.ObjectKey{Namespace: defaultNamespace, Name: btpOperatorName}, cr)).Should(Succeed())
			Expect(k8sClient.Delete(ctx, cr)).Should(Succeed())
			Eventually(updateCh).Should(Receive(matchState(types.StateReady)))
			Eventually(updateCh).Should(Receive(matchDeleted()))
			Expect(isCrNotFound()).To(BeTrue())
			restoreOriginalCertificateTiems()
		}

		ensureReconcilationQueueIsEmpty := func() {
			Eventually(actualWorkqueueSize).WithTimeout(time.Second * 5).WithPolling(time.Millisecond * 100).Should(Equal(0))
		}

		ensureCorrectState := func() {
			ensureReconcilationQueueIsEmpty()
			ok, err := reconciler.isWebhookSecretCertSignedByCaSecretCert(ctx)
			Expect(err).To(BeNil())
			Expect(ok).To(BeTrue())
			ensureAllWebhooksManagedByBtpOperatorHaveCorrectCABundles()
		}

		When("doing provisioning", func() {
			BeforeAll(func() {
				certBeforeEach(nil)
			})
			AfterAll(func() {
				certAfterEach()
			})
			It("should generate correct certs pair", func() {
				ensureCorrectState()
			})
		})

		When("ca cert changes", func() {
			BeforeAll(func() {
				certBeforeEach(nil)
			})
			AfterAll(func() {
				certAfterEach()
			})
			It("should regenerate CA and webhook certs", func() {
				newCaCertificate, newCaPrivateKey, err := certs.GenerateSelfSignedCertificate(time.Now().Add(CaCertificateExpiration))
				newCaPrivateKeyStructured, err := reconciler.structToByteArray(newCaPrivateKey)
				Expect(err).To(BeNil())

				caSecret := getSecret(CaSecret)
				orgCaSecret := caSecret
				replaceSecretData(caSecret, reconciler.buildKeyNameWithExtension(CaSecretDataPrefix, CertificatePostfix), newCaCertificate, reconciler.buildKeyNameWithExtension(CaSecretDataPrefix, RsaKeyPostfix), newCaPrivateKeyStructured)
				ensureReconcilationQueueIsEmpty()
				updatedCaSecret := getSecret(CaSecret)

				caCertificateAfterUpdate, ok := updatedCaSecret.Data[reconciler.buildKeyNameWithExtension(CaSecretDataPrefix, CertificatePostfix)]
				Expect(ok).To(BeTrue())
				Expect(!bytes.Equal(caCertificateAfterUpdate, newCaCertificate)).To(BeTrue())

				caCertificateOriginal, ok := orgCaSecret.Data[reconciler.buildKeyNameWithExtension(CaSecretDataPrefix, CertificatePostfix)]
				Expect(ok).To(BeTrue())
				Expect(!bytes.Equal(caCertificateAfterUpdate, caCertificateOriginal)).To(BeTrue())

				caPrivateKeyAfterUpdate, ok := updatedCaSecret.Data[reconciler.buildKeyNameWithExtension(CaSecretDataPrefix, RsaKeyPostfix)]
				Expect(ok).To(BeTrue())
				Expect(!bytes.Equal(caPrivateKeyAfterUpdate, newCaPrivateKeyStructured)).To(BeTrue())

				caPrivateKeyOriginal, ok := orgCaSecret.Data[reconciler.buildKeyNameWithExtension(CaSecretDataPrefix, RsaKeyPostfix)]
				Expect(ok).To(BeTrue())
				Expect(!bytes.Equal(caPrivateKeyAfterUpdate, caPrivateKeyOriginal)).To(BeTrue())

				ensureCorrectState()
			})
		})

		When("webhook cert changes, signed by same CA", func() {
			BeforeAll(func() {
				certBeforeEach(nil)
			})
			AfterAll(func() {
				certAfterEach()
			})
			It("CA certificate stay same, but Webhook certificate is change (signed by same CA)", func() {
				beforeCaSecret := getSecret(CaSecret)

				currentCa, err := reconciler.getDataFromSecret(ctx, CaSecret)
				Expect(err).To(BeNil())
				ca, err := reconciler.getValueByKey(reconciler.buildKeyNameWithExtension(CaSecretDataPrefix, CertificatePostfix), currentCa)
				Expect(err).To(BeNil())
				pk, err := reconciler.getValueByKey(reconciler.buildKeyNameWithExtension(CaSecretDataPrefix, RsaKeyPostfix), currentCa)
				Expect(err).To(BeNil())
				currentWebhookSecret := getSecret(WebhookSecret)
				originalWebhookSecret := currentWebhookSecret

				newWebhookCertificate, newWebhookPrivateKey, err := certs.GenerateSignedCertificate(time.Now().Add(WebhookCertificateExpiration), ca, pk)
				Expect(err).To(BeNil())
				newWebhookPrivateKeyStructured, err := reconciler.structToByteArray(newWebhookPrivateKey)
				Expect(err).To(BeNil())

				webhookCert := getSecret(CaSecret)
				replaceSecretData(webhookCert, reconciler.buildKeyNameWithExtension(WebhookSecretDataPrefix, CertificatePostfix), newWebhookCertificate, reconciler.buildKeyNameWithExtension(WebhookSecretDataPrefix, RsaKeyPostfix), newWebhookPrivateKeyStructured)
				ensureReconcilationQueueIsEmpty()

				originalWebhookCert, ok := originalWebhookSecret.Data[reconciler.buildKeyNameWithExtension(WebhookSecretDataPrefix, CertificatePostfix)]
				Expect(!bytes.Equal(originalWebhookCert, newWebhookCertificate))

				currentWebhookSecret = getSecret(WebhookSecret)
				currentWebhookCert, ok := currentWebhookSecret.Data[reconciler.buildKeyNameWithExtension(WebhookSecretDataPrefix, CertificatePostfix)]
				Expect(ok).To(BeTrue())
				Expect(bytes.Equal(currentWebhookCert, newWebhookCertificate))

				afterCaSecret := getSecret(CaSecret)
				afterCaSecretCert, ok := afterCaSecret.Data[reconciler.buildKeyNameWithExtension(CaSecretDataPrefix, CertificatePostfix)]
				beforeCaSecretCert, ok := beforeCaSecret.Data[reconciler.buildKeyNameWithExtension(CaSecretDataPrefix, CertificatePostfix)]
				Expect(bytes.Equal(afterCaSecretCert, beforeCaSecretCert))
				ensureCorrectState()
			})
		})

		When("webhook cert changes, signed by different CA", func() {
			BeforeAll(func() {
				certBeforeEach(nil)
			})
			AfterAll(func() {
				certAfterEach()
			})
			It("CA and Webhook certificate is regenerated", func() {
				newCaCertificate, newCaPrivateKey, err := certs.GenerateSelfSignedCertificate(time.Now().Add(CaCertificateExpiration))
				Expect(err).To(BeNil())

				newWebhookCertificate, newWebhookPrivateKey, err := certs.GenerateSignedCertificate(time.Now().Add(WebhookCertificateExpiration), newCaCertificate, newCaPrivateKey)
				newWebhookCertificateStructured, err := reconciler.structToByteArray(newWebhookPrivateKey)
				Expect(err).To(BeNil())

				beforeCaSecret := getSecret(CaSecret)
				beforeWebhookSecret := getSecret(WebhookSecret)

				webhookCertSecret := getSecret(WebhookSecret)
				replaceSecretData(webhookCertSecret, reconciler.buildKeyNameWithExtension(WebhookSecretDataPrefix, CertificatePostfix), newWebhookCertificate, reconciler.buildKeyNameWithExtension(WebhookSecretDataPrefix, RsaKeyPostfix), newWebhookCertificateStructured)
				ensureReconcilationQueueIsEmpty()

				currentCaSecret := getSecret(CaSecret)
				currentCaCert, ok := currentCaSecret.Data[reconciler.buildKeyNameWithExtension(CaSecretDataPrefix, CertificatePostfix)]
				Expect(ok).To(BeTrue())
				beforeCaCert, ok := beforeCaSecret.Data[reconciler.buildKeyNameWithExtension(CaSecretDataPrefix, CertificatePostfix)]
				Expect(ok).To(BeTrue())
				Expect(!bytes.Equal(currentCaCert, beforeCaCert))

				currentWebhookSecret := getSecret(WebhookSecret)
				currentWebhookCert, ok := currentWebhookSecret.Data[reconciler.buildKeyNameWithExtension(WebhookSecretDataPrefix, CertificatePostfix)]
				Expect(ok).To(BeTrue())
				beforeWebhookCert, ok := beforeWebhookSecret.Data[reconciler.buildKeyNameWithExtension(WebhookSecretDataPrefix, CertificatePostfix)]
				Expect(ok).To(BeTrue())
				Expect(!bytes.Equal(currentWebhookCert, beforeWebhookCert))
				Expect(!bytes.Equal(currentWebhookCert, newWebhookCertificate))

				ensureCorrectState()
			})
		})

		When("webhook expired", func() {
			fakeTime := 30.0
			fakeExpiration := 10.0
			BeforeAll(func() {
				timeOpts := &certificationsTimeOpts{
					CaCertificateExpiration: CaCertificateExpiration,
					WebhookCertExpiration:   time.Second * time.Duration(fakeTime),
					ExpirationBoundary:      time.Second * time.Duration(fakeExpiration),
				}
				certBeforeEach(timeOpts)
			})
			AfterAll(func() {
				certAfterEach()
			})

			It("should generate new webhook cert, CA should stay as is", func() {
				caSecretBeforeExpiration := getSecret(CaSecret)
				webhookSecretBeforeExpiration := getSecret(WebhookSecret)
				Expect(checkHowManySecondsToExpiration(WebhookSecret) <= fakeTime).To(BeTrue())
				restoreOriginalCertificateTiems()
				ensureReconcilationQueueIsEmpty()
				_, err := reconciler.Reconcile(ctx, controllerruntime.Request{NamespacedName: apimachienerytypes.NamespacedName{
					Namespace: cr.Namespace,
					Name:      cr.Name,
				}})
				Expect(err).To(BeNil())
				ensureReconcilationQueueIsEmpty()
				caSecretAfterExpiration := getSecret(CaSecret)
				webhookSecretAfterExpiration := getSecret(WebhookSecret)
				Expect(reflect.DeepEqual(caSecretBeforeExpiration.Data, caSecretAfterExpiration.Data)).To(BeTrue())
				Expect(reflect.DeepEqual(webhookSecretBeforeExpiration.Data, webhookSecretAfterExpiration.Data)).To(BeFalse())
				Expect(checkHowManySecondsToExpiration(WebhookSecret) >= fakeTime).To(BeTrue())

				ensureCorrectState()
			})
		})

		When("ca expired", func() {
			fakeSeconds := 30.0
			fakeExpiration := 10.0
			BeforeAll(func() {
				timeOpts := &certificationsTimeOpts{
					CaCertificateExpiration: time.Second * time.Duration(fakeSeconds),
					WebhookCertExpiration:   orgWebhookCertExpiration,
					ExpirationBoundary:      time.Second * time.Duration(fakeExpiration),
				}
				certBeforeEach(timeOpts)
			})
			AfterAll(func() {
				certAfterEach()
			})

			It("should generate new webhook cert, CA should stay as is", func() {
				caSecretBeforeExpiration := getSecret(CaSecret)
				webhookSecretBeforeExpiration := getSecret(WebhookSecret)
				Expect(checkHowManySecondsToExpiration(CaSecret) <= fakeSeconds).To(BeTrue())
				time.Sleep(time.Second * 10)
				Expect(checkHowManySecondsToExpiration(CaSecret) <= fakeSeconds).To(BeTrue())
				restoreOriginalCertificateTiems()
				ensureReconcilationQueueIsEmpty()
				_, err := reconciler.Reconcile(ctx, controllerruntime.Request{NamespacedName: apimachienerytypes.NamespacedName{
					Namespace: cr.Namespace,
					Name:      cr.Name,
				}})
				Expect(err).To(BeNil())
				ensureReconcilationQueueIsEmpty()
				caSecretAfterExpiration := getSecret(CaSecret)
				webhookSecretAfterExpiration := getSecret(WebhookSecret)
				Expect(reflect.DeepEqual(caSecretBeforeExpiration.Data, caSecretAfterExpiration.Data)).To(BeFalse())
				Expect(reflect.DeepEqual(webhookSecretBeforeExpiration.Data, webhookSecretAfterExpiration.Data)).To(BeFalse())
				Expect(checkHowManySecondsToExpiration(WebhookSecret) >= fakeSeconds).To(BeTrue())
				Expect(checkHowManySecondsToExpiration(CaSecret) >= fakeSeconds).To(BeTrue())

				ensureCorrectState()
			})
		})

		When("webhook ca bundle modified, from different CA", func() {
			BeforeAll(func() {
				certBeforeEach(nil)
			})
			AfterAll(func() {
				certAfterEach()
			})
			It("should be restored to existing CA", func() {
				newCaCertificate, _, err := certs.GenerateSelfSignedCertificate(time.Now().Add(CaCertificateExpiration))
				Expect(err).To(BeNil())
				updated := replaceCaBundleInMutatingWebhooks(newCaCertificate)
				updated = !updated && replaceCaBundleInValidatingWebhooks(newCaCertificate)
				ensureCorrectState()
			})
		})

		When("webhook ca bundle modified, with some dummy text", func() {
			BeforeAll(func() {
				certBeforeEach(nil)
			})
			AfterAll(func() {
				certAfterEach()
			})
			It("should be restored to existing CA", func() {
				dummy := []byte("dummy")
				updated := replaceCaBundleInMutatingWebhooks(dummy)
				updated = !updated && replaceCaBundleInValidatingWebhooks(dummy)
				ensureCorrectState()
			})
		})
	})
})

func getApplyPath() string {
	return fmt.Sprintf("%s%capply", ResourcesPath, os.PathSeparator)
}

func getDeletePath() string {
	return fmt.Sprintf("%s%cdelete", ResourcesPath, os.PathSeparator)
}

func getToDeleteYamlPath() string {
	return fmt.Sprintf("%s%cto-delete.yml", getDeletePath(), os.PathSeparator)
}

func getTempPath() string {
	return fmt.Sprintf("%s%ctemp", ResourcesPath, os.PathSeparator)
}

func assertResourcesExistence(uns ...*unstructured.Unstructured) {
	for _, u := range uns {
		gvk := u.GroupVersionKind()
		temp := &unstructured.Unstructured{}
		temp.SetGroupVersionKind(gvk)
		Expect(k8sClient.Get(ctx, client.ObjectKey{Namespace: u.GetNamespace(), Name: u.GetName()}, temp)).To(Succeed())
	}
}

func assertResourcesRemoval(uns ...*unstructured.Unstructured) {
	for _, u := range uns {
		gvk := u.GroupVersionKind()
		temp := &unstructured.Unstructured{}
		temp.SetGroupVersionKind(gvk)
		gr := schema.GroupResource{
			Group:    gvk.Group,
			Resource: fmt.Sprintf("%ss", strings.ToLower(gvk.Kind)),
		}
		expectedErr := k8serrors.NewNotFound(gr, u.GetName())
		expectedErr.ErrStatus.TypeMeta.APIVersion = "v1"
		expectedErr.ErrStatus.TypeMeta.Kind = "Status"
		Expect(k8sClient.Get(ctx, client.ObjectKey{Namespace: u.GetNamespace(), Name: u.GetName()}, temp)).To(MatchError(expectedErr))
	}
}

func moveOrCopyNFilesFromDirToDir(filesNum int, deleteFiles bool, srcDir, targetDir string) error {
	if err := os.Mkdir(targetDir, 0700); err != nil && !os.IsExist(err) {
		return err
	}
	files, err := os.ReadDir(srcDir)
	if err != nil {
		return err
	}
	for i, f := range files {
		if i >= filesNum {
			break
		}
		input, err := os.ReadFile(fmt.Sprintf("%s%c%s", srcDir, os.PathSeparator, f.Name()))
		if err != nil {
			return err
		}
		if err := os.WriteFile(fmt.Sprintf("%s%c%s", targetDir, os.PathSeparator, f.Name()), input, 0700); err != nil {
			return err
		}
		if deleteFiles {
			if err := os.Remove(fmt.Sprintf("%s%c%s", srcDir, os.PathSeparator, f.Name())); err != nil {
				return err
			}
		}
	}

	return nil
}

func getUniqueGvksFromObjects(objs []runtime.Object) []schema.GroupVersionKind {
	gvks := make([]schema.GroupVersionKind, 0)
	helper := make(map[string]struct{}, 0)
	for _, o := range objs {
		gvk := o.GetObjectKind().GroupVersionKind()
		gvkString := gvk.String()
		if _, exists := helper[gvkString]; exists {
			continue
		}
		helper[gvkString] = struct{}{}
		gvks = append(gvks, gvk)
	}

	return gvks
}

func countResourcesForGivenChartVer(gvks []schema.GroupVersionKind, version string) (int, error) {
	var foundResources int
	var ul *unstructured.UnstructuredList
	for _, gvk := range gvks {
		ul = &unstructured.UnstructuredList{}
		ul.SetGroupVersionKind(gvk)
		if err := k8sClient.List(ctx, ul, client.MatchingLabels{chartVersionKey: version}); err != nil {
			return 0, err
		}
		foundResources += len(ul.Items)
	}

	return foundResources, nil
}

func copyDirRecursively(src, target string) {
	cmd := exec.Command("cp", "-r", src, target)
	err := cmd.Run()
	Expect(err).To(BeNil())
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
	GinkgoLogr.Info(fmt.Sprintf("Got CR status: %s\n", cr.Status.State))
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

func createChartOrResourcesCopyWithoutWebhooks(src, dst string) error {
	Expect(os.MkdirAll(dst, 0700)).To(Succeed())
	src = fmt.Sprintf("%v/.", src)
	cmd := exec.Command("cp", "-r", src, dst)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("copying: %v -> %v\n\nout: %v\nerr: %v", src, dst, string(out), err)
	}
	filterWebhooksDisabled := os.Getenv("DISABLE_WEBHOOK_FILTER_FOR_TESTS")

	return filepath.WalkDir(dst, func(path string, de fs.DirEntry, err error) error {
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

func removeAllFromPath(path string) error {
	return os.RemoveAll(path)
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
		if err := k8sClient.List(ctx, list, managedByLabelFilter); err != nil {
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

func replaceCaBundleInValidatingWebhooks(newCaBundle []byte) bool {
	webhookConfig := &admissionregistrationv1.ValidatingWebhookConfigurationList{}
	err := k8sClient.List(ctx, webhookConfig, managedByLabelFilter)
	Expect(err).To(BeNil())
	Expect(len(webhookConfig.Items) > 0).To(BeTrue())
	if len(webhookConfig.Items) > 0 {
		webhook := webhookConfig.Items[0]
		if len(webhook.Webhooks) > 0 {
			webhook.Webhooks[0].ClientConfig.CABundle = newCaBundle
			err := k8sClient.Update(ctx, &webhook)
			Expect(err).To(BeNil())
			//time.Sleep(time.Second * 10)
			return true
		}
	}
	return false
}

func replaceCaBundleInMutatingWebhooks(newCaBundle []byte) bool {
	webhookConfig := &admissionregistrationv1.MutatingWebhookConfigurationList{}
	err := k8sClient.List(ctx, webhookConfig, managedByLabelFilter)
	Expect(err).To(BeNil())
	Expect(len(webhookConfig.Items) > 0).To(BeTrue())
	if len(webhookConfig.Items) > 0 {
		webhook := webhookConfig.Items[0]
		if len(webhook.Webhooks) > 0 {
			webhook.Webhooks[0].ClientConfig.CABundle = newCaBundle
			err := k8sClient.Update(ctx, &webhook)
			Expect(err).To(BeNil())
			//time.Sleep(time.Second * 10)
			return true
		}
	}
	return false
}

func replaceSecretData(secret *corev1.Secret, key string, value []byte, key2 string, value2 []byte) {
	data := secret.Data
	if key != "" && value != nil {
		data[key] = value
	}

	if key2 != "" && value2 != nil {
		data[key2] = value2
	}
	secret.Data = data
	err := k8sClient.Update(ctx, secret)
	Expect(err).To(BeNil())
}

func getSecret(name string) *corev1.Secret {
	secret := &corev1.Secret{}
	err := k8sClient.Get(ctx, client.ObjectKey{Namespace: ChartNamespace, Name: name}, secret)
	Expect(err).To(BeNil())
	return secret
}

func checkHowManySecondsToExpiration(name string) float64 {
	data, err := reconciler.getDataFromSecret(ctx, name)
	Expect(err).To(BeNil())
	key, err := reconciler.mapSecretNameToSecretDataKey(name)
	Expect(err).To(BeNil())
	value, err := reconciler.getValueByKey(reconciler.buildKeyNameWithExtension(key, CertificatePostfix), data)
	Expect(err).To(BeNil())
	decoded, _ := pem.Decode(value)
	cert, err := x509.ParseCertificate(decoded.Bytes)
	Expect(err).To(BeNil())
	diff := cert.NotAfter.Sub(time.Now())
	return diff.Seconds()
}

func ensureAllWebhooksManagedByBtpOperatorHaveCorrectCABundles() {
	secret := getSecret(CaSecret)
	ca, ok := secret.Data[reconciler.buildKeyNameWithExtension(CaSecretDataPrefix, CertificatePostfix)]
	Expect(ok).To(BeTrue())
	Expect(ca).To(Not(BeNil()))
	vw := &admissionregistrationv1.ValidatingWebhookConfigurationList{}
	err := k8sClient.List(ctx, vw, managedByLabelFilter)
	Expect(err).To(BeNil())
	for _, w := range vw.Items {
		for _, n := range w.Webhooks {
			Expect(bytes.Equal(n.ClientConfig.CABundle, ca)).To(BeTrue())
		}
	}
	mw := &admissionregistrationv1.MutatingWebhookConfigurationList{}
	err = k8sClient.List(ctx, mw, managedByLabelFilter)
	Expect(err).To(BeNil())
	for _, w := range mw.Items {
		for _, n := range w.Webhooks {
			Expect(bytes.Equal(n.ClientConfig.CABundle, ca)).To(BeTrue())
		}
	}
}
