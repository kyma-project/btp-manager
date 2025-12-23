/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kyma-project/btp-manager/api/v1alpha1"
	"github.com/kyma-project/btp-manager/controllers/config"
	"github.com/kyma-project/btp-manager/internal/certs"
	btpmanagermetrics "github.com/kyma-project/btp-manager/internal/metrics"

	. "github.com/onsi/ginkgo/v2"
	ginkgotypes "github.com/onsi/ginkgo/v2/types"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"
	// +kubebuilder:scaffold:imports
)

const (
	fakeSapBtpOperatorImage = "local.test/sap/sap-btp-service-operator/controller:v0.0.1"
	fakeKubeRbacProxyImage  = "local.test/brancz/kube-rbac-proxy:v0.0.1"
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var logger = logf.Log.WithName("suite_test")

const (
	hardDeleteTimeoutForAllTests         = time.Second * 1
	deleteRequestTimeoutForAllTests      = time.Millisecond * 200
	statusUpdateTimeoutForAllTests       = time.Millisecond * 200
	statusUpdateCheckIntervalForAllTests = time.Millisecond * 20
	testRsaKeyBits                       = 1024
	resourceAdded                        = "added"
	resourceUpdated                      = "updated"
	resourceDeleted                      = "deleted"
	kymaNamespace                        = "kyma-system"
	defaultChartPath                     = "./testdata/test-module-chart"
	defaultResourcesPath                 = "./testdata/test-module-resources"
	defaultManagerResourcesPath          = "./testdata/test-manager-resources"
	chartUpdatePath                      = "./testdata/module-chart-update"
	resourcesUpdatePath                  = "./testdata/module-resources-update"
)

var (
	cfg                        *rest.Config
	k8sClient                  client.Client
	k8sClientFromManager       client.Client
	k8sManager                 manager.Manager
	testEnv                    *envtest.Environment
	ctx                        context.Context
	ctxForDeploymentController context.Context
	cancel                     context.CancelFunc
	cancelDeploymentController context.CancelFunc
	reconciler                 *BtpOperatorReconciler
	updateCh                   chan resourceUpdate = make(chan resourceUpdate, 1000)
)

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	suiteCfg, reporterCfg := GinkgoConfiguration()
	ReconfigureGinkgo(&reporterCfg, &suiteCfg)
	singleTestTimeout := os.Getenv("SINGLE_TEST_TIMEOUT")
	if singleTestTimeout != "" {
		timeout, err := time.ParseDuration(singleTestTimeout)
		require.NoError(t, err)
		SetDefaultEventuallyTimeout(timeout)
	} else {
		SetDefaultEventuallyTimeout(time.Second * 5)
	}

	RunSpecs(t, "Controller Suite", suiteCfg, reporterCfg)
}

func ReconfigureGinkgo(reporterCfg *ginkgotypes.ReporterConfig, suiteCfg *ginkgotypes.SuiteConfig) {
	verbosity := os.Getenv("GINKGO_VERBOSE_FLAG")
	// If not override Ginkgo verbosity then "Normal" option will be used.
	switch {
	case verbosity == "ginkgo.v":
		reporterCfg.Verbose = true
	case verbosity == "ginkgo.vv":
		reporterCfg.VeryVerbose = true
	case verbosity == "ginkgo.succinct":
		reporterCfg.Succinct = true
	}

	setTrace := os.Getenv("GINKGO_TRACE")
	if setTrace == "trace" {
		reporterCfg.FullTrace = true
	}

	suiteCfg.LabelFilter = os.Getenv("GINKGO_LABEL_FILTER")

	fmt.Printf("Labels [%s]\n", suiteCfg.LabelFilter)
}

var _ = SynchronizedBeforeSuite(func() {
	// runs only on process #1
	config.ChartPath = "../module-chart/chart"
	config.ResourcesPath = "../module-resources"
	config.ManagerResourcesPath = "../manager-resources"
	Expect(createChartOrResourcesCopyWithoutWebhooksByConfig(config.ChartPath, defaultChartPath)).To(Succeed())
	Expect(createChartOrResourcesCopyWithoutWebhooksByConfig(config.ResourcesPath, defaultResourcesPath)).To(Succeed())
	Expect(createChartOrResourcesCopyWithoutWebhooksByConfig(config.ManagerResourcesPath, defaultManagerResourcesPath)).To(Succeed())
}, func() {
	// runs on all processes
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), func(o *zap.Options) {
		o.Development = true
		o.TimeEncoder = zapcore.ISO8601TimeEncoder
	}))
	ctx, cancel = context.WithCancel(context.TODO())
	ctxForDeploymentController, cancelDeploymentController = ctx, cancel

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
	}

	var err error
	// cfg is defined in this file globally.
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = v1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	// +kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	k8sManager, err = ctrl.NewManager(cfg, ctrl.Options{
		Scheme:                 scheme.Scheme,
		Metrics:                server.Options{BindAddress: "0"},
		HealthProbeBindAddress: "0",
		NewCache:               CacheCreator,
	})
	Expect(err).ToNot(HaveOccurred())

	ctx, cancel = context.WithCancel(ctrl.SetupSignalHandler())

	metrics := btpmanagermetrics.NewMetrics()
	cleanupReconciler := NewInstanceBindingControllerManager(ctx, k8sManager.GetClient(), k8sManager.GetScheme(), cfg)
	reconciler = NewBtpOperatorReconciler(
		k8sManager.GetClient(),
		k8sClient,
		k8sManager.GetScheme(),
		cleanupReconciler,
		metrics,
		[]config.WatchHandler{
			config.NewHandler(k8sManager.GetClient(), k8sManager.GetScheme()),
		},
	)

	k8sClientFromManager = k8sManager.GetClient()

	if hardDeleteTimeoutFromEnv := os.Getenv("HARD_DELETE_TIMEOUT"); hardDeleteTimeoutFromEnv != "" {
		config.HardDeleteTimeout, err = time.ParseDuration(hardDeleteTimeoutFromEnv)
		Expect(err).NotTo(HaveOccurred())
	} else {
		config.HardDeleteTimeout = hardDeleteTimeoutForAllTests
	}
	if hardDeleteCheckIntervalFromEnv := os.Getenv("HARD_DELETE_CHECK_INTERVAL"); hardDeleteCheckIntervalFromEnv != "" {
		config.HardDeleteCheckInterval, err = time.ParseDuration(hardDeleteCheckIntervalFromEnv)
		Expect(err).NotTo(HaveOccurred())
	} else {
		config.HardDeleteCheckInterval = hardDeleteTimeoutForAllTests / 20
	}
	if deleteRequestTimeoutFromEnv := os.Getenv("DELETE_REQUEST_TIMEOUT"); deleteRequestTimeoutFromEnv != "" {
		config.DeleteRequestTimeout, err = time.ParseDuration(deleteRequestTimeoutFromEnv)
		Expect(err).NotTo(HaveOccurred())
	} else {
		config.DeleteRequestTimeout = deleteRequestTimeoutForAllTests
	}
	config.ChartPath = defaultChartPath
	config.ResourcesPath = defaultResourcesPath
	certs.SetRsaKeyBits(testRsaKeyBits)

	useExistingClusterEnv := os.Getenv("USE_EXISTING_CLUSTER")
	if useExistingClusterEnv != "true" {
		config.StatusUpdateTimeout = statusUpdateTimeoutForAllTests
		config.StatusUpdateCheckInterval = statusUpdateCheckIntervalForAllTests
	}

	if os.Getenv(SapBtpServiceOperatorEnv) == "" {
		Expect(os.Setenv(SapBtpServiceOperatorEnv, fakeSapBtpOperatorImage)).To(Succeed())
	}
	if os.Getenv(KubeRbacProxyEnv) == "" {
		Expect(os.Setenv(KubeRbacProxyEnv, fakeKubeRbacProxyImage)).To(Succeed())
	}

	err = reconciler.SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	informer, err := k8sManager.GetCache().GetInformer(ctx, &v1alpha1.BtpOperator{})
	Expect(err).ToNot(HaveOccurred())
	_, err = informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    func(o any) { resourceAddDeleteHandler(o, resourceAdded) },
		UpdateFunc: func(o, n any) { resourceUpdateHandler(o, n, resourceUpdated) },
		DeleteFunc: func(o any) { resourceAddDeleteHandler(o, resourceDeleted) },
	})
	Expect(err).ToNot(HaveOccurred())

	Expect(createPrereqs()).To(Succeed())

	go func() {
		defer GinkgoRecover()
		err = k8sManager.Start(ctx)
		Expect(err).ToNot(HaveOccurred(), "failed to run manager")
	}()

	if useExistingClusterEnv != "true" {
		ctxForDeploymentController, cancelDeploymentController = context.WithCancel(ctx)
		go func() {
			deploymentController := newDeploymentController(cfg, k8sManager)
			defer GinkgoRecover()
			<-k8sManager.Elected()
			err := deploymentController.Start(ctxForDeploymentController)
			Expect(err).ToNot(HaveOccurred(), "failed to run deployment controller")
		}()

		apiServerAddressAndPort := fmt.Sprintf("%s:%s", testEnv.ControlPlane.APIServer.Address, testEnv.ControlPlane.APIServer.Port)
		etcdAddressAndPort := testEnv.ControlPlane.Etcd.URL.Host
		ginkgoProcessInfoMsg := fmt.Sprintf("Process: %d, ApiServer: %s, etcd: %s", GinkgoParallelProcess(), apiServerAddressAndPort, etcdAddressAndPort)
		GinkgoWriter.Println(ginkgoProcessInfoMsg)
	}

	k8sManager.GetCache().WaitForCacheSync(ctx)
})

var _ = SynchronizedAfterSuite(func() {
	// runs on all processes
	Eventually(func() int { return reconciler.workqueueSize }).Should(Equal(0))
	cancelDeploymentController()
	cancel()
	By("tearing down the test environment")
	Expect(testEnv.Stop()).To(Succeed())
}, func() {
	// runs only on process #1
	Expect(os.RemoveAll(defaultChartPath)).To(Succeed())
	Expect(os.RemoveAll(defaultResourcesPath)).To(Succeed())
})
