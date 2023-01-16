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

	"go.uber.org/zap/zapcore"

	. "github.com/onsi/ginkgo/v2"
	ginkgotypes "github.com/onsi/ginkgo/v2/types"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	gomegatypes "github.com/onsi/gomega/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/kyma-project/btp-manager/api/v1alpha1"
	"github.com/kyma-project/module-manager/pkg/types"
	//+kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var logger = logf.Log.WithName("suite_test")

const (
	hardDeleteTimeout = time.Millisecond * 200
	resourceAdded     = "added"
	resourceUpdated   = "updated"
	resourceDeleted   = "deleted"
)

var (
	cfg                  *rest.Config
	k8sClient            client.Client
	k8sClientFromManager client.Client
	k8sManager           manager.Manager
	testEnv              *envtest.Environment
	ctx                  context.Context
	cancel               context.CancelFunc
	reconciler           *BtpOperatorReconciler
	updateCh             chan resourceUpdate = make(chan resourceUpdate, 1000)
)

type resourceUpdate struct {
	Cr     *v1alpha1.BtpOperator
	Action string
}

func resourceUpdateHandler(obj any, t string) {
	if cr, ok := obj.(*v1alpha1.BtpOperator); ok {
		logger.V(1).Info("Triggered update handler for BTPOperator CR", "name", cr.Name, "action", t, "state", cr.Status.State, "conditions", cr.Status.Conditions)
		updateCh <- resourceUpdate{Cr: cr, Action: t}
	}
}

func matchState(state types.State) gomegatypes.GomegaMatcher {
	return MatchFields(IgnoreExtras, Fields{
		"Action": Equal(resourceUpdated),
		"Cr": PointTo(MatchFields(IgnoreExtras, Fields{
			"Status": MatchFields(IgnoreExtras, Fields{
				"State": Equal(state),
			}),
		})),
	})
}

func matchReadyCondition(state types.State, status metav1.ConditionStatus, reason Reason) gomegatypes.GomegaMatcher {
	return MatchFields(IgnoreExtras, Fields{
		"Action": Equal(resourceUpdated),
		"Cr": PointTo(MatchFields(IgnoreExtras, Fields{
			"Status": MatchFields(IgnoreExtras, Fields{
				"State": Equal(state),
				"Conditions": ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":   Equal(ReadyType),
					"Reason": Equal(string(reason)),
					"Status": Equal(status),
				}))),
			}),
		})),
	})
}

func matchDeleted() gomegatypes.GomegaMatcher {
	return MatchFields(IgnoreExtras, Fields{"Action": Equal(resourceDeleted)})
}

func TestAPIs(t *testing.T) {

	RegisterFailHandler(Fail)

	suiteCfg, reporterCfg := GinkgoConfiguration()
	ReconfigureGinkgo(&reporterCfg, &suiteCfg)
	SetDefaultEventuallyTimeout(time.Second * 5)
	reporterCfg.Verbose = true
	RunSpecs(t, "Controller Suite", suiteCfg, reporterCfg)
}

func ReconfigureGinkgo(reporterCfg *ginkgotypes.ReporterConfig, suiteCfg *ginkgotypes.SuiteConfig) {
	verbosity := os.Getenv("GINKGO_VERBOSE_FLAG")
	switch {
	case verbosity == "ginkgo.v":
		reporterCfg.Verbose = true
	case verbosity == "ginkgo.vv":
		reporterCfg.VeryVerbose = true
	case verbosity == "ginkgo.succinct":
		reporterCfg.Succinct = true
	default:
		reporterCfg.Verbose = true
	}
	suiteCfg.LabelFilter = os.Getenv("GINKGO_LABEL_FILTER")
	fmt.Printf("Labels [%s]\n", suiteCfg.LabelFilter)
}

var _ = BeforeSuite(func() {

	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), func(o *zap.Options) {
		o.Development = true
		o.TimeEncoder = zapcore.ISO8601TimeEncoder
	}))

	ctx, cancel = context.WithCancel(context.TODO())

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

	//+kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
	})
	Expect(err).ToNot(HaveOccurred())

	reconciler = &BtpOperatorReconciler{
		Client:                k8sManager.GetClient(),
		Scheme:                k8sManager.GetScheme(),
		WaitForChartReadiness: false,
	}
	k8sClientFromManager = k8sManager.GetClient()
	HardDeleteTimeout = hardDeleteTimeout
	HardDeleteCheckInterval = hardDeleteTimeout / 20
	ChartPath = "../module-chart"

	err = reconciler.SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	informer, err := k8sManager.GetCache().GetInformer(ctx, &v1alpha1.BtpOperator{})
	Expect(err).ToNot(HaveOccurred())
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    func(o any) { resourceUpdateHandler(o, resourceAdded) },
		UpdateFunc: func(o, n any) { resourceUpdateHandler(n, resourceUpdated) },
		DeleteFunc: func(o any) { resourceUpdateHandler(o, resourceDeleted) },
	})

	go func() {
		defer GinkgoRecover()
		err = k8sManager.Start(ctx)
		Expect(err).ToNot(HaveOccurred(), "failed to run manager")
	}()

	k8sManager.GetCache().WaitForCacheSync(ctx)
})

var _ = AfterSuite(func() {
	Eventually(func() int { return reconciler.workqueueSize }).Should(Equal(0))
	cancel()
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})
