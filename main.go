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

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"

	"github.com/kyma-project/btp-manager/api/v1alpha1"
	"github.com/kyma-project/btp-manager/controllers"
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
	flag.StringVar(&controllers.ChartNamespace, "chart-namespace", controllers.ChartNamespace, "Namespace to install chart resources.")
	flag.StringVar(&controllers.SecretName, "secret-name", controllers.SecretName, "Secret name with input values for sap-btp-operator chart templating.")
	flag.StringVar(&controllers.ConfigName, "config-name", controllers.ConfigName, "ConfigMap name with configuration knobs for the btp-manager internals.")
	flag.StringVar(&controllers.DeploymentName, "deployment-name", controllers.DeploymentName, "Name of the deployment of sap-btp-operator for deprovisioning.")
	flag.StringVar(&controllers.ChartPath, "chart-path", controllers.ChartPath, "Path to the root directory inside the chart.")
	flag.StringVar(&controllers.ResourcesPath, "resources-path", controllers.ResourcesPath, "Path to the directory with module resources to apply/delete.")
	flag.DurationVar(&controllers.ProcessingStateRequeueInterval, "processing-state-requeue-interval", controllers.ProcessingStateRequeueInterval, `Requeue interval for state "processing".`)
	flag.DurationVar(&controllers.ReadyStateRequeueInterval, "ready-state-requeue-interval", controllers.ReadyStateRequeueInterval, `Requeue interval for state "ready".`)
	flag.DurationVar(&controllers.ReadyTimeout, "ready-timeout", controllers.ReadyTimeout, "Helm chart timeout.")
	flag.DurationVar(&controllers.ReadyCheckInterval, "ready-check-interval", controllers.ReadyCheckInterval, "Ready check retry interval.")
	flag.DurationVar(&controllers.HardDeleteCheckInterval, "hard-delete-check-interval", controllers.HardDeleteCheckInterval, "Hard delete retry interval.")
	flag.DurationVar(&controllers.HardDeleteTimeout, "hard-delete-timeout", controllers.HardDeleteTimeout, "Hard delete timeout.")
	flag.DurationVar(&controllers.DeleteRequestTimeout, "delete-request-timeout", controllers.DeleteRequestTimeout, "Delete request timeout in hard delete.")
	flag.StringVar(&controllers.EnableLimitedCache, "enable-limited-cache", controllers.EnableLimitedCache, "Enable limited cache for sap-btp-operator.")
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
	metrics := btpmanagermetrics.NewMetrics()
	cleanupReconciler := controllers.NewInstanceBindingControllerManager(signalContext, mgr.GetClient(), mgr.GetScheme(), restCfg)
	reconciler := controllers.NewBtpOperatorReconciler(mgr.GetClient(), apiServerClient, scheme, cleanupReconciler, metrics)
	configReconciler := controllers.NewConfigReconciler(mgr.GetClient(), scheme, reconciler)

	if err = reconciler.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "BtpOperator")
		os.Exit(1)
	}

	if err = configReconciler.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Configuration")
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
