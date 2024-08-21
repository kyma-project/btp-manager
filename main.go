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
	"fmt"
	"log/slog"
	"os"
	"strings"

	//test

	clusterobject "github.com/kyma-project/btp-manager/internal/cluster-object"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/manager"

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

type managerWithContext struct {
	manager.Manager
	context.Context
}

func (mgr *managerWithContext) start() {
	setupLog.Info("starting manager")
	if err := mgr.Manager.Start(mgr.Context); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

type smClient struct {
	context.Context
	k8sReader      client.Reader
	secretProvider *clusterobject.SecretProvider
}

func (c *smClient) start() {
	setupLog.Info("starting SM client")
	siCrdExists, err := c.crdExists(c.Context, controllers.InstanceGvk)
	if err != nil {
		ctrl.Log.Error(err, "failed to check if ServiceInstance CRD exists")
		os.Exit(1)
	}
	if !siCrdExists {
		ctrl.Log.Info("cannot fetch all existing SAP BTP service operator secrets, required ServiceInstance CRD does not exist")
		return
	}
	secrets, err := c.secretProvider.All(c.Context)
	if err != nil {
		ctrl.Log.Error(err, "failed to fetch all SAP BTP service operator secrets")
		os.Exit(1)
	}
	ctrl.Log.Info("number of existing SAP BTP service operator secrets", "count", len(secrets.Items))
}

func (c *smClient) crdExists(ctx context.Context, gvk schema.GroupVersionKind) (bool, error) {
	crdName := fmt.Sprintf("%ss.%s", strings.ToLower(gvk.Kind), gvk.Group)
	crd := &apiextensionsv1.CustomResourceDefinition{}

	ctrl.Log.Info("checking if CRD exists", "name", crdName)

	if err := c.k8sReader.Get(ctx, client.ObjectKey{Name: crdName}, crd); err != nil {
		if k8serrors.IsNotFound(err) {
			ctrl.Log.Info("CRD does not exist", "name", crdName)
			return false, nil
		} else {
			ctrl.Log.Error(err, "failed to get CRD", "name", crdName)
			return false, err
		}
	}
	ctrl.Log.Info("CRD exists", "name", crdName)
	return true, nil
}

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
	var probeAddr, metricsAddr string
	var enableLeaderElection bool
	parseCmdFlags(&probeAddr, &metricsAddr, &enableLeaderElection)

	restCfg := ctrl.GetConfigOrDie()
	signalContext := ctrl.SetupSignalHandler()

	mgr := setupManager(restCfg, &probeAddr, &metricsAddr, &enableLeaderElection, signalContext)
	sm := setupSMClient(restCfg, signalContext)

	// start components
	go mgr.start()
	go sm.start()

	select {
	case <-signalContext.Done():
		setupLog.Info("shutting down btp-manager")
	}
}

func parseCmdFlags(probeAddr *string, metricsAddr *string, enableLeaderElection *bool) {
	flag.StringVar(probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.StringVar(metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.BoolVar(enableLeaderElection, "leader-elect", false,
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
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))
}

func setupManager(restCfg *rest.Config, probeAddr *string, metricsAddr *string, enableLeaderElection *bool, signalContext context.Context) managerWithContext {
	mgr, err := ctrl.NewManager(restCfg, ctrl.Options{
		Scheme:                 scheme,
		LeaderElection:         *enableLeaderElection,
		LeaderElectionID:       "ec023d38.kyma-project.io",
		Metrics:                server.Options{BindAddress: *metricsAddr},
		HealthProbeBindAddress: *probeAddr,
		NewCache:               controllers.CacheCreator,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	metrics := btpmanagermetrics.NewMetrics()
	cleanupReconciler := controllers.NewInstanceBindingControllerManager(signalContext, mgr.GetClient(), mgr.GetScheme(), restCfg)
	reconciler := controllers.NewBtpOperatorReconciler(mgr.GetClient(), scheme, cleanupReconciler, metrics)

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

	return managerWithContext{
		Manager: mgr,
		Context: signalContext,
	}
}

func setupSMClient(restCfg *rest.Config, signalCtx context.Context) smClient {
	k8sClient, err := client.New(restCfg, client.Options{})
	if err != nil {
		setupLog.Error(err, "unable to create k8s client")
		os.Exit(1)
	}
	slogger := slog.Default()
	namespaceProvider := clusterobject.NewNamespaceProvider(k8sClient, slogger)
	serviceInstanceProvider := clusterobject.NewServiceInstanceProvider(k8sClient, slogger)
	secretProvider := clusterobject.NewSecretProvider(k8sClient, namespaceProvider, serviceInstanceProvider, slogger)

	return smClient{
		Context:        signalCtx,
		k8sReader:      k8sClient,
		secretProvider: secretProvider,
	}
}
