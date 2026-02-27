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
	"errors"
	"flag"
	"os"
	"strings"

	//test

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlmetrics "sigs.k8s.io/controller-runtime/pkg/metrics"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"

	"github.com/kyma-project/btp-manager/api/v1alpha1"
	"github.com/kyma-project/btp-manager/controllers"
	"github.com/kyma-project/btp-manager/controllers/config"
	btpmanagermetrics "github.com/kyma-project/btp-manager/internal/metrics"
	//+kubebuilder:scaffold:imports
)

var (
	scheme   = clientgoscheme.Scheme
	setupLog = ctrl.Log.WithName("setup")
)

func init() {

	utilruntime.Must(v1alpha1.AddToScheme(scheme))
	utilruntime.Must(apiextensionsv1.AddToScheme(scheme))

	//+kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.StringVar(&config.ChartNamespace, "chart-namespace", config.ChartNamespace, "Namespace to install chart resources.")
	flag.StringVar(&config.SecretName, "secret-name", config.SecretName, "Secret name with input values for sap-btp-operator chart templating.")
	flag.StringVar(&config.ConfigName, "config-name", config.ConfigName, "ConfigMap name with configuration knobs for the btp-manager internals.")
	flag.StringVar(&config.DeploymentName, "deployment-name", config.DeploymentName, "Name of the deployment of sap-btp-operator for deprovisioning.")
	flag.StringVar(&config.ChartPath, "chart-path", config.ChartPath, "Path to the root directory inside the chart.")
	flag.StringVar(&config.ResourcesPath, "resources-path", config.ResourcesPath, "Path to the directory with module resources to apply/delete.")
	flag.DurationVar(&config.ProcessingStateRequeueInterval, "processing-state-requeue-interval", config.ProcessingStateRequeueInterval, `Requeue interval for state "processing".`)
	flag.DurationVar(&config.ReadyStateRequeueInterval, "ready-state-requeue-interval", config.ReadyStateRequeueInterval, `Requeue interval for state "ready".`)
	flag.DurationVar(&config.ReadyTimeout, "ready-timeout", config.ReadyTimeout, "Helm chart timeout.")
	flag.DurationVar(&config.ReadyCheckInterval, "ready-check-interval", config.ReadyCheckInterval, "Ready check retry interval.")
	flag.DurationVar(&config.HardDeleteCheckInterval, "hard-delete-check-interval", config.HardDeleteCheckInterval, "Hard delete retry interval.")
	flag.DurationVar(&config.HardDeleteTimeout, "hard-delete-timeout", config.HardDeleteTimeout, "Hard delete timeout.")
	flag.DurationVar(&config.DeleteRequestTimeout, "delete-request-timeout", config.DeleteRequestTimeout, "Delete request timeout in hard delete.")
	flag.StringVar(&config.EnableLimitedCache, "enable-limited-cache", config.EnableLimitedCache, "Enable limited cache for sap-btp-operator.")
	flag.DurationVar(&config.StatusUpdateTimeout, "status-update-timeout", config.StatusUpdateTimeout, "Status update timeout.")
	flag.DurationVar(&config.StatusUpdateCheckInterval, "status-update-check-interval", config.StatusUpdateCheckInterval, "Status update retry interval.")
	flag.StringVar(&config.ManagerResourcesPath, "manager-resources-path", config.ManagerResourcesPath, "Path to the directory with BTP Manager resources.")
	opts := zap.Options{
		Development: false,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	requiredEnvs := []string{
		controllers.SapBtpServiceOperatorEnv,
		controllers.KubeRbacProxyEnv,
	}
	if err := ensureRequiredEnvs(requiredEnvs...); err != nil {
		setupLog.Error(err, "missing required environment variables")
		os.Exit(1)
	}

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	restCfg := ctrl.GetConfigOrDie()
	mgr, err := ctrl.NewManager(restCfg, ctrl.Options{
		Scheme:                 scheme,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "ec023d38.kyma-project.io",
		Metrics:                server.Options{BindAddress: metricsAddr},
		HealthProbeBindAddress: probeAddr,
		NewCache:               controllers.CacheCreator,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	apiServerClient, err := client.New(restCfg, client.Options{})
	if err != nil {
		setupLog.Error(err, "unable to create API server client")
		os.Exit(1)
	}

	signalContext := ctrl.SetupSignalHandler()
	webhookMetrics := btpmanagermetrics.NewWebhookMetrics(ctrlmetrics.Registry)
	configMetrics := btpmanagermetrics.NewConfigMetrics(ctrlmetrics.Registry)
	cleanupReconciler := controllers.NewInstanceBindingControllerManager(signalContext, mgr.GetClient(), mgr.GetScheme(), restCfg)
	reconciler := controllers.NewBtpOperatorReconciler(
		mgr.GetClient(),
		apiServerClient,
		scheme,
		cleanupReconciler,
		webhookMetrics,
		[]config.WatchHandler{
			config.NewHandler(mgr.GetClient(), scheme, configMetrics),
		},
	)

	if err = reconciler.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "BtpOperator")
		os.Exit(1)
	}

	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(signalContext); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

func ensureRequiredEnvs(envs ...string) error {
	missingEnvs := make([]string, 0)
	for _, e := range envs {
		if os.Getenv(e) == "" {
			missingEnvs = append(missingEnvs, e)
		}
	}
	if len(missingEnvs) > 0 {
		return errors.New("required environment variables not set: " + strings.Join(missingEnvs, ", "))
	}

	return nil
}
