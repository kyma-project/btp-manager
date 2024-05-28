/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions  and
limitations under the License.
*/

package main

import (
	"context"
	"flag"
	"log/slog"
	"os"

	"github.com/kyma-project/btp-manager/api/v1alpha1"
	"github.com/kyma-project/btp-manager/controllers"
	clusterobject "github.com/kyma-project/btp-manager/internal/cluster-object"
	btpmanagermetrics "github.com/kyma-project/btp-manager/internal/metrics"
	servicemanager "github.com/kyma-project/btp-manager/internal/service-manager"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"
	// +kubebuilder:scaffold:imports
	api2 "github.com/kyma-project/btp-manager/internal/api"
)

type managerWithContext struct {
	context.Context
	manager.Manager
}

func (mgr *managerWithContext) start() {
	setupLog.Info("starting manager")
	if err := mgr.Manager.Start(mgr.Context); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

type serviceManagerClient struct {
	context.Context
	*servicemanager.Client
}

func (c *serviceManagerClient) start() {
	setupLog.Info("starting Service Manager client")
	if err := c.Defaults(c.Context); err != nil {
		setupLog.Error(err, "problem running Service Manager client")
		os.Exit(1)
	}
}

var (
	scheme   = clientgoscheme.Scheme
	setupLog = ctrl.Log.WithName("setup")
)

func init() {

	utilruntime.Must(v1alpha1.AddToScheme(scheme))
	utilruntime.Must(apiextensionsv1.AddToScheme(scheme))

	// +kubebuilder:scaffold:scheme
}

func main() {
	var probeAddr, metricsAddr string
	var enableLeaderElection bool
	parseCmdFlags(&probeAddr, &metricsAddr, &enableLeaderElection)

	restCfg := ctrl.GetConfigOrDie()
	signalContext := ctrl.SetupSignalHandler()

	mgr := setupManager(restCfg, &probeAddr, &metricsAddr, &enableLeaderElection, signalContext)
	sm := setupSMClient(restCfg, signalContext)
	api := api2.NewAPI(sm.Client)
	// start components
	go mgr.start()
	go sm.start()
	go api.Start()

	select {
	case <-signalContext.Done():
		setupLog.Info("shutting down btp-manager")
	}
}

func parseCmdFlags(probeAddr *string, metricsAddr *string, enableLeaderElection *bool) {
	flag.StringVar(probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.StringVar(metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.BoolVar(
		enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.",
	)
	flag.StringVar(
		&controllers.ChartNamespace, "chart-namespace", controllers.ChartNamespace,
		"Namespace to install chart resources.",
	)
	flag.StringVar(
		&controllers.SecretName, "secret-name", controllers.SecretName,
		"Secret name with input values for sap-btp-operator chart templating.",
	)
	flag.StringVar(
		&controllers.ConfigName, "config-name", controllers.ConfigName,
		"ConfigMap name with configuration knobs for the btp-manager internals.",
	)
	flag.StringVar(
		&controllers.DeploymentName, "deployment-name", controllers.DeploymentName,
		"Name of the deployment of sap-btp-operator for deprovisioning.",
	)
	flag.StringVar(
		&controllers.ChartPath, "chart-path", controllers.ChartPath, "Path to the root directory inside the chart.",
	)
	flag.StringVar(
		&controllers.ResourcesPath, "resources-path", controllers.ResourcesPath,
		"Path to the directory with module resources to apply/delete.",
	)
	flag.DurationVar(
		&controllers.ProcessingStateRequeueInterval, "processing-state-requeue-interval",
		controllers.ProcessingStateRequeueInterval, `Requeue interval for state "processing".`,
	)
	flag.DurationVar(
		&controllers.ReadyStateRequeueInterval, "ready-state-requeue-interval", controllers.ReadyStateRequeueInterval,
		`Requeue interval for state "ready".`,
	)
	flag.DurationVar(&controllers.ReadyTimeout, "ready-timeout", controllers.ReadyTimeout, "Helm chart timeout.")
	flag.DurationVar(
		&controllers.ReadyCheckInterval, "ready-check-interval", controllers.ReadyCheckInterval,
		"Ready check retry interval.",
	)
	flag.DurationVar(
		&controllers.HardDeleteCheckInterval, "hard-delete-check-interval", controllers.HardDeleteCheckInterval,
		"Hard delete retry interval.",
	)
	flag.DurationVar(
		&controllers.HardDeleteTimeout, "hard-delete-timeout", controllers.HardDeleteTimeout, "Hard delete timeout.",
	)
	flag.DurationVar(
		&controllers.DeleteRequestTimeout, "delete-request-timeout", controllers.DeleteRequestTimeout,
		"Delete request timeout in hard delete.",
	)
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))
}

func setupManager(
	restCfg *rest.Config, probeAddr *string, metricsAddr *string, enableLeaderElection *bool,
	signalContext context.Context,
) managerWithContext {
	mgr, err := ctrl.NewManager(
		restCfg, ctrl.Options{
			Scheme:                 scheme,
			LeaderElection:         *enableLeaderElection,
			LeaderElectionID:       "ec023d38.kyma-project.io",
			Metrics:                server.Options{BindAddress: *metricsAddr},
			HealthProbeBindAddress: *probeAddr,
			NewCache:               controllers.CacheCreator,
		},
	)
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	metrics := btpmanagermetrics.NewMetrics()
	cleanupReconciler := controllers.NewInstanceBindingControllerManager(
		signalContext, mgr.GetClient(), mgr.GetScheme(), restCfg,
	)
	reconciler := controllers.NewBtpOperatorReconciler(mgr.GetClient(), scheme, cleanupReconciler, metrics)

	if err = reconciler.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "BtpOperator")
		os.Exit(1)
	}
	// +kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	return managerWithContext{
		Manager: mgr,
		Context: signalContext,
	}
}

func setupSMClient(restCfg *rest.Config, signalCtx context.Context) *serviceManagerClient {
	k8sClient, err := client.New(restCfg, client.Options{})
	if err != nil {
		setupLog.Error(err, "unable to create k8s client")
		os.Exit(1)
	}
	slogger := slog.Default()
	namespaceProvider := clusterobject.NewNamespaceProvider(k8sClient, slogger)
	serviceInstanceProvider := clusterobject.NewServiceInstanceProvider(k8sClient, slogger)
	secretProvider := clusterobject.NewSecretProvider(k8sClient, namespaceProvider, serviceInstanceProvider, slogger)
	smClient := servicemanager.NewClient(signalCtx, slogger, secretProvider)

	return &serviceManagerClient{
		Context: signalCtx,
		Client:  smClient,
	}
}
