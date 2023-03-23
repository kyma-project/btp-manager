package controllers

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"reflect"
	"time"

	"github.com/kyma-project/btp-manager/api/v1alpha1"
	"github.com/kyma-project/btp-manager/internal/certs"
	"github.com/kyma-project/btp-manager/internal/manifest"
	"github.com/kyma-project/btp-manager/internal/ymlutils"
	"github.com/kyma-project/module-manager/pkg/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	apimachienerytypes "k8s.io/apimachinery/pkg/types"
	controllerruntime "sigs.k8s.io/controller-runtime"
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

var _ = Describe("BTP Operator controller - provisioning", func() {
	var cr *v1alpha1.BtpOperator

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
		Expect(createPrereqs()).To(Succeed())
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

	//var _ = Describe("BTP Operator controller - configuration", func() {}

	Describe("Configurability", func() {
		Context("When the ConfigMap is present", func() {
			It("should adjust configuration settings in the operator accordingly", func() {
				cm := initConfig(map[string]string{"ProcessingStateRequeueInterval": "10s"})
				reconciler.reconcileConfig(cm)
				Expect(ProcessingStateRequeueInterval).To(Equal(time.Second * 10))
			})
		})
	})

	//var _ = Describe("BTP Operator controller - deprovisioning", func() {
	Describe("Deprovisioning without force-delete label", func() {

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
			Expect(k8sClient.Get(ctx, client.ObjectKey{Namespace: defaultNamespace, Name: btpOperatorName}, cr)).To(Succeed())
			cr.SetLabels(map[string]string{forceDeleteLabelKey: "true"})
			Expect(k8sClient.Update(ctx, cr)).To(Succeed())
			Eventually(func() (bool, error) {
				err := k8sClient.Get(ctx, client.ObjectKey{Namespace: defaultNamespace, Name: btpOperatorName}, cr)
				return cr.Labels[forceDeleteLabelKey] == "true", err
			}).WithTimeout(k8sOpsTimeout).WithPolling(k8sOpsPollingInterval).Should(BeTrue())
			Eventually(updateCh).Should(Receive(matchDeleted()))
		})

		It("Delete should fail because of existing instances and bindings", func() {
			_ = createResource(instanceGvk, kymaNamespace, instanceName)
			ensureResourceExists(instanceGvk)

			_ = createResource(bindingGvk, kymaNamespace, bindingName)
			ensureResourceExists(bindingGvk)

			Expect(k8sClient.Get(ctx, client.ObjectKey{Namespace: defaultNamespace, Name: btpOperatorName}, cr)).To(Succeed())
			Expect(k8sClient.Delete(ctx, cr)).To(Succeed())

			Eventually(updateCh).Should(Receive(matchReadyCondition(types.StateDeleting, metav1.ConditionFalse, ServiceInstancesAndBindingsNotCleaned)))
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

	// var _ = Describe("BTP Operator controller - updating", func() {
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

	// var _ = Describe("BTP Operator controller - certificates", func() {
	Describe("Certification management", func() {
		orgCaCertificateExpiration := CaCertificateExpiration
		orgWebhookCertExpiration := WebhookCertificateExpiration
		orgExpirationBoundary := ExpirationBoundary

		BeforeAll(func() {
			Expect(removeAllFromPath(defaultChartPath)).To(Succeed())
			Expect(removeAllFromPath(defaultResourcesPath)).To(Succeed())

			Expect(os.Setenv("DISABLE_WEBHOOK_FILTER_FOR_TESTS", "true")).To(BeNil())
			Expect(createChartOrResourcesCopyWithoutWebhooks("../module-chart/chart", defaultChartPath)).To(Succeed())
			Expect(createChartOrResourcesCopyWithoutWebhooks("../module-resources", defaultResourcesPath)).To(Succeed())
			ChartPath = defaultChartPath
			ResourcesPath = defaultResourcesPath
		})

		restoreOriginalCertificateTimes := func() {
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
				restoreOriginalCertificateTimes()
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
			restoreOriginalCertificateTimes()
		}

		ensureReconciliationQueueIsEmpty := func() {
			Eventually(func() int { return reconciler.workqueueSize }).WithTimeout(time.Second * 5).WithPolling(time.Millisecond * 100).Should(Equal(0))
		}

		ensureCorrectState := func() {
			ensureReconciliationQueueIsEmpty()
			ok, err := reconciler.isWebhookSecretCertSignedByCaSecretCert(ctx)
			Expect(err).To(BeNil())
			Expect(ok).To(BeTrue())
			ensureAllWebhooksManagedByBtpOperatorHaveCorrectCABundles()
		}

		Context("certs created with default expiration times", func() {
			BeforeEach(func() {
				certBeforeEach(nil)
			})

			AfterEach(func() {
				certAfterEach()
			})

			When("certs don't exist in the cluster prior provisioning", func() {
				It("should generate correct certs pair", func() {
					ensureCorrectState()
				})
			})

			When("CA certificate changes", func() {
				It("should do fully regenerate of CA certificate and webhook certificate", func() {
					newCaCertificate, newCaPrivateKey, err := certs.GenerateSelfSignedCertificate(time.Now().Add(CaCertificateExpiration))
					newCaPrivateKeyStructured, err := structToByteArray(newCaPrivateKey)
					Expect(err).To(BeNil())

					caSecret := getSecret(CaSecret)
					replaceSecretData(caSecret, reconciler.buildKeyNameWithExtension(CaSecretDataPrefix, CertificatePostfix), newCaCertificate, reconciler.buildKeyNameWithExtension(CaSecretDataPrefix, RsaKeyPostfix), newCaPrivateKeyStructured)
					ensureReconciliationQueueIsEmpty()
					updatedCaSecret := getSecret(CaSecret)

					caCertificateAfterUpdate, ok := updatedCaSecret.Data[reconciler.buildKeyNameWithExtension(CaSecretDataPrefix, CertificatePostfix)]
					Expect(ok).To(BeTrue())
					Expect(!bytes.Equal(caCertificateAfterUpdate, newCaCertificate)).To(BeTrue())

					caCertificateOriginal, ok := caSecret.Data[reconciler.buildKeyNameWithExtension(CaSecretDataPrefix, CertificatePostfix)]
					Expect(ok).To(BeTrue())
					Expect(!bytes.Equal(caCertificateAfterUpdate, caCertificateOriginal)).To(BeTrue())

					caPrivateKeyAfterUpdate, ok := updatedCaSecret.Data[reconciler.buildKeyNameWithExtension(CaSecretDataPrefix, RsaKeyPostfix)]
					Expect(ok).To(BeTrue())
					Expect(!bytes.Equal(caPrivateKeyAfterUpdate, newCaPrivateKeyStructured)).To(BeTrue())

					caPrivateKeyOriginal, ok := caSecret.Data[reconciler.buildKeyNameWithExtension(CaSecretDataPrefix, RsaKeyPostfix)]
					Expect(ok).To(BeTrue())
					Expect(!bytes.Equal(caPrivateKeyAfterUpdate, caPrivateKeyOriginal)).To(BeTrue())

					ensureCorrectState()
				})
			})

			When("webhook certificate changes and is signed by same CA certificate", func() {
				It("CA certificate is not changed, webhook certificate is regenerated", func() {
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
					newWebhookPrivateKeyStructured, err := structToByteArray(newWebhookPrivateKey)
					Expect(err).To(BeNil())

					webhookCert := getSecret(WebhookSecret)
					replaceSecretData(webhookCert, reconciler.buildKeyNameWithExtension(WebhookSecretDataPrefix, CertificatePostfix), newWebhookCertificate, reconciler.buildKeyNameWithExtension(WebhookSecretDataPrefix, RsaKeyPostfix), newWebhookPrivateKeyStructured)
					ensureReconciliationQueueIsEmpty()

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

			When("webhook certificate is signed by different CA certificate", func() {
				It("CA certificate and webhook certificate are fully regenerated", func() {
					newCaCertificate, newCaPrivateKey, err := certs.GenerateSelfSignedCertificate(time.Now().Add(CaCertificateExpiration))
					Expect(err).To(BeNil())

					newWebhookCertificate, newWebhookPrivateKey, err := certs.GenerateSignedCertificate(time.Now().Add(WebhookCertificateExpiration), newCaCertificate, newCaPrivateKey)
					newWebhookCertificateStructured, err := structToByteArray(newWebhookPrivateKey)
					Expect(err).To(BeNil())

					beforeCaSecret := getSecret(CaSecret)
					beforeWebhookSecret := getSecret(WebhookSecret)

					webhookCertSecret := getSecret(WebhookSecret)
					replaceSecretData(webhookCertSecret, reconciler.buildKeyNameWithExtension(WebhookSecretDataPrefix, CertificatePostfix), newWebhookCertificate, reconciler.buildKeyNameWithExtension(WebhookSecretDataPrefix, RsaKeyPostfix), newWebhookCertificateStructured)
					ensureReconciliationQueueIsEmpty()

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

			When("webhook caBundle modified with new CA certificate", func() {
				It("should be reconciled to existing CA certificate", func() {
					newCaCertificate, _, err := certs.GenerateSelfSignedCertificate(time.Now().Add(CaCertificateExpiration))
					Expect(err).To(BeNil())
					updated := replaceCaBundleInMutatingWebhooks(newCaCertificate)
					if !updated {
						updated = replaceCaBundleInValidatingWebhooks(newCaCertificate)
					}
					Expect(updated).To(BeTrue())
					ensureCorrectState()
				})
			})

			When("webhook caBundle modified with some dummy text", func() {
				It("should be reconciled to existing CA certificate", func() {
					dummy := []byte("dummy")
					updated := replaceCaBundleInMutatingWebhooks(dummy)
					if !updated {
						updated = replaceCaBundleInValidatingWebhooks(dummy)
					}
					Expect(updated).To(BeTrue())
					ensureCorrectState()
				})
			})
		})

		Context("certs created with custom expiration times", func() {
			fakeSeconds := 30.0
			fakeExpiration := 10.0

			AfterEach(func() {
				certAfterEach()
			})

			When("webhook certificate expires", func() {
				BeforeEach(func() {
					timeOpts := &certificationsTimeOpts{
						CaCertificateExpiration: CaCertificateExpiration,
						WebhookCertExpiration:   time.Second * time.Duration(fakeSeconds),
						ExpirationBoundary:      time.Second * time.Duration(fakeExpiration),
					}
					certBeforeEach(timeOpts)
				})

				It("CA certificate is not changed, webhook certificate is regenerated", func() {
					caSecretBeforeExpiration := getSecret(CaSecret)
					webhookSecretBeforeExpiration := getSecret(WebhookSecret)
					Expect(checkHowManySecondsToExpiration(WebhookSecret)).Should(BeNumerically("<=", fakeSeconds))

					restoreOriginalCertificateTimes()
					ensureReconciliationQueueIsEmpty()
					_, err := reconciler.Reconcile(ctx, controllerruntime.Request{NamespacedName: apimachienerytypes.NamespacedName{
						Namespace: cr.Namespace,
						Name:      cr.Name,
					}})
					Expect(err).To(BeNil())
					ensureReconciliationQueueIsEmpty()
					caSecretAfterExpiration := getSecret(CaSecret)
					webhookSecretAfterExpiration := getSecret(WebhookSecret)
					Expect(reflect.DeepEqual(caSecretBeforeExpiration.Data, caSecretAfterExpiration.Data)).To(BeTrue())
					Expect(reflect.DeepEqual(webhookSecretBeforeExpiration.Data, webhookSecretAfterExpiration.Data)).To(BeFalse())
					Expect(checkHowManySecondsToExpiration(WebhookSecret)).Should(BeNumerically(">=", fakeSeconds))

					ensureCorrectState()
				})
			})

			When("CA certificate expires", func() {
				BeforeEach(func() {
					timeOpts := &certificationsTimeOpts{
						CaCertificateExpiration: time.Second * time.Duration(fakeSeconds),
						WebhookCertExpiration:   orgWebhookCertExpiration,
						ExpirationBoundary:      time.Second * time.Duration(fakeExpiration),
					}
					certBeforeEach(timeOpts)
				})

				It("fully regenerate of CA certificate and webhook certificate", func() {
					caSecretBeforeExpiration := getSecret(CaSecret)
					webhookSecretBeforeExpiration := getSecret(WebhookSecret)
					Expect(checkHowManySecondsToExpiration(CaSecret)).Should(BeNumerically("<=", fakeSeconds))
					restoreOriginalCertificateTimes()
					ensureReconciliationQueueIsEmpty()
					_, err := reconciler.Reconcile(ctx, controllerruntime.Request{NamespacedName: apimachienerytypes.NamespacedName{
						Namespace: cr.Namespace,
						Name:      cr.Name,
					}})
					Expect(err).To(BeNil())
					ensureReconciliationQueueIsEmpty()
					caSecretAfterExpiration := getSecret(CaSecret)
					webhookSecretAfterExpiration := getSecret(WebhookSecret)
					Expect(reflect.DeepEqual(caSecretBeforeExpiration.Data, caSecretAfterExpiration.Data)).To(BeFalse())
					Expect(reflect.DeepEqual(webhookSecretBeforeExpiration.Data, webhookSecretAfterExpiration.Data)).To(BeFalse())
					Expect(checkHowManySecondsToExpiration(WebhookSecret)).Should(BeNumerically(">=", fakeSeconds))
					Expect(checkHowManySecondsToExpiration(CaSecret)).Should(BeNumerically(">=", fakeSeconds))
					ensureCorrectState()
				})
			})
		})
	})
})
