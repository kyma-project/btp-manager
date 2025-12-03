package controllers

import (
	"context"
	"strconv"
	"time"

	"github.com/kyma-project/btp-manager/internal/certs"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

type ConfigReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func NewConfigReconciler(client client.Client, scheme *runtime.Scheme) *ConfigReconciler {
	return &ConfigReconciler{
		Client: client,
		Scheme: scheme,
	}
}

func (r *ConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.ConfigMap{}).
		WithEventFilter(r.predicates()).
		Complete(r)
}

func (r *ConfigReconciler) predicates() predicate.Funcs {
	nameMatches := func(o client.Object) bool { return o.GetName() == ConfigName && o.GetNamespace() == ChartNamespace }
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool { return nameMatches(e.Object) },
		DeleteFunc: func(e event.DeleteEvent) bool { return nameMatches(e.Object) },
		UpdateFunc: func(e event.UpdateEvent) bool { return nameMatches(e.ObjectNew) },
	}
}

func (r *ConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	cm := &corev1.ConfigMap{}
	if err := r.Get(ctx, req.NamespacedName, cm); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	logger := log.FromContext(ctx, "name", cm.GetName(), "namespace", cm.GetNamespace())
	logger.Info("reconciling configuration update", "config", cm.Data)

	for k, v := range cm.Data {
		var err error
		switch k {
		case "ChartNamespace":
			ChartNamespace = v
		case "ChartPath":
			ChartPath = v
		case "SecretName":
			SecretName = v
		case "ConfigName":
			ConfigName = v
		case "DeploymentName":
			DeploymentName = v
		case "ProcessingStateRequeueInterval":
			ProcessingStateRequeueInterval, err = time.ParseDuration(v)
		case "ReadyStateRequeueInterval":
			ReadyStateRequeueInterval, err = time.ParseDuration(v)
		case "ReadyTimeout":
			ReadyTimeout, err = time.ParseDuration(v)
		case "HardDeleteCheckInterval":
			HardDeleteCheckInterval, err = time.ParseDuration(v)
		case "HardDeleteTimeout":
			HardDeleteTimeout, err = time.ParseDuration(v)
		case "ResourcesPath":
			ResourcesPath = v
		case "ReadyCheckInterval":
			ReadyCheckInterval, err = time.ParseDuration(v)
		case "DeleteRequestTimeout":
			DeleteRequestTimeout, err = time.ParseDuration(v)
		case "CaCertificateExpiration":
			CaCertificateExpiration, err = time.ParseDuration(v)
		case "WebhookCertificateExpiration":
			WebhookCertificateExpiration, err = time.ParseDuration(v)
		case "ExpirationBoundary":
			ExpirationBoundary, err = time.ParseDuration(v)
		case "RsaKeyBits":
			var bits int
			bits, err = strconv.Atoi(v)
			if err == nil {
				certs.SetRsaKeyBits(bits)
			}
		case "EnableLimitedCache":
			EnableLimitedCache = v
		default:
			logger.Info("unknown configuration update key", k, v)
		}
		if err != nil {
			logger.Info("failed to parse configuration update", k, err)
		}
	}

	return ctrl.Result{}, nil
}
