package controllers

import (
	"fmt"
	"os"
	"time"

	"github.com/kyma-project/btp-manager/api/v1alpha1"
	"github.com/kyma-project/btp-manager/internal/manifest"
	"github.com/kyma-project/btp-manager/internal/ymlutils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	apimachienerytypes "k8s.io/apimachinery/pkg/types"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	suffix          = "-updated"
	newChartVersion = "9.9.9"
)

var _ = Describe("BTP Operator controller - updating", func() {
	var cr *v1alpha1.BtpOperator
	var initChartVersion, chartUpdatePathForProcess, resourcesUpdatePathForProcess string
	var manifestHandler *manifest.Handler
	var initApplyObjs []runtime.Object
	var gvks []schema.GroupVersionKind
	var initResourcesNum int
	var actualWorkqueueSize func() int
	var err error

	BeforeEach(func() {
		GinkgoWriter.Println("--- PROCESS:", GinkgoParallelProcess(), "---")

		secret, err := createCorrectSecretFromYaml()
		Expect(err).To(BeNil())
		Expect(k8sClient.Patch(ctx, secret, client.Apply, client.ForceOwnership, client.FieldOwner(operatorName))).To(Succeed())

		manifestHandler = &manifest.Handler{Scheme: k8sManager.GetScheme()}
		actualWorkqueueSize = func() int { return reconciler.workqueueSize }

		cr = createDefaultBtpOperator()
		Expect(k8sClient.Create(ctx, cr)).To(Succeed())
		Eventually(updateCh).Should(Receive(matchState(v1alpha1.StateReady)))

		initChartVersion, err = ymlutils.ExtractStringValueFromYamlForGivenKey(fmt.Sprintf("%s/Chart.yaml", ChartPath), "version")
		Expect(err).To(BeNil())
		_ = initChartVersion

		initApplyObjs, err = manifestHandler.CollectObjectsFromDir(getApplyPath())
		Expect(err).To(BeNil())

		gvks = getUniqueGvksFromObjects(initApplyObjs)

		initResourcesNum, err = countResourcesForGivenChartVer(gvks, initChartVersion)
		Expect(err).To(BeNil())

		chartUpdatePathForProcess = fmt.Sprintf("%s%d", chartUpdatePath, GinkgoParallelProcess())
		resourcesUpdatePathForProcess = fmt.Sprintf("%s%d", resourcesUpdatePath, GinkgoParallelProcess())
		copyDirRecursively(ChartPath, chartUpdatePathForProcess)
		copyDirRecursively(ResourcesPath, resourcesUpdatePathForProcess)
		ChartPath = chartUpdatePathForProcess
		ResourcesPath = resourcesUpdatePathForProcess
	})

	AfterEach(func() {
		cr = &v1alpha1.BtpOperator{}
		Expect(k8sClient.Get(ctx, client.ObjectKey{Namespace: defaultNamespace, Name: btpOperatorName}, cr)).Should(Succeed())
		Expect(k8sClient.Delete(ctx, cr)).Should(Succeed())
		Eventually(updateCh).Should(Receive(matchDeleted()))
		Expect(isCrNotFound()).To(BeTrue())

		deleteSecret := &corev1.Secret{}
		Expect(k8sClient.Get(ctx, client.ObjectKey{Namespace: kymaNamespace, Name: SecretName}, deleteSecret)).To(Succeed())
		Expect(k8sClient.Delete(ctx, deleteSecret)).To(Succeed())

		Expect(os.RemoveAll(chartUpdatePathForProcess)).To(Succeed())
		Expect(os.RemoveAll(resourcesUpdatePathForProcess)).To(Succeed())

		ChartPath = defaultChartPath
		ResourcesPath = defaultResourcesPath
	})

	When("update all resources names and bump chart version", Label("test-update"), func() {
		It("new resources (with new names) should be created and old ones removed", func() {
			err := ymlutils.CopyManifestsFromYamlsIntoOneYaml(getApplyPath(), getToDeleteYamlPath())
			Expect(err).To(BeNil())

			err = ymlutils.AddSuffixToNameInManifests(getApplyPath(), suffix)
			Expect(err).To(BeNil())

			err = ymlutils.UpdateChartVersion(chartUpdatePathForProcess, newChartVersion)
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

			err = ymlutils.UpdateChartVersion(chartUpdatePathForProcess, newChartVersion)
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

			err = ymlutils.UpdateChartVersion(chartUpdatePathForProcess, newChartVersion)
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
			err = ymlutils.UpdateChartVersion(chartUpdatePathForProcess, newChartVersion)
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
