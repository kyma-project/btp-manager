package config

import (
	"context"
	"strconv"
	"time"

	"github.com/kyma-project/btp-manager/internal/certs"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	BtpOperatorCrName       = "btpoperator"
	KymaSystemNamespaceName = "kyma-system"
)

// Configuration options that can be overwritten either by CLI parameter or ConfigMap
var (
	ChartNamespace = "kyma-system"
	SecretName     = "sap-btp-manager"
	ConfigName     = "sap-btp-manager"
	DeploymentName = "sap-btp-operator-controller-manager"

	ProcessingStateRequeueInterval = time.Minute * 5
	ReadyStateRequeueInterval      = time.Minute * 15
	ReadyTimeout                   = time.Minute * 5
	ReadyCheckInterval             = time.Second * 30
	HardDeleteTimeout              = time.Minute * 20
	HardDeleteCheckInterval        = time.Second * 10
	DeleteRequestTimeout           = time.Minute * 5
	StatusUpdateTimeout            = time.Second * 10
	StatusUpdateCheckInterval      = time.Millisecond * 500

	CaCertificateExpiration      = time.Hour * 87600 // 10 years
	WebhookCertificateExpiration = time.Hour * 8760  // 1 year
	ExpirationBoundary           = time.Hour * -168  // 1 week

	ChartPath            = "./module-chart/chart"
	ResourcesPath        = "./module-resources"
	ManagerResourcesPath = "./manager-resources"

	EnableLimitedCache = "false"
)

type Handler struct {
	client.Client
	Scheme *runtime.Scheme
}

func NewHandler(client client.Client, scheme *runtime.Scheme) *Handler {
	return &Handler{
		Client: client,
		Scheme: scheme,
	}
}

func (r *Handler) Predicates() predicate.Funcs {
	nameMatches := func(o client.Object) bool {
		return o.GetName() == ConfigName && o.GetNamespace() == ChartNamespace
	}
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool { return nameMatches(e.Object) },
		DeleteFunc: func(e event.DeleteEvent) bool { return nameMatches(e.Object) },
		UpdateFunc: func(e event.UpdateEvent) bool { return nameMatches(e.ObjectNew) },
	}
}

func (r *Handler) Reconcile(ctx context.Context, obj client.Object) []reconcile.Request {
	logger := log.FromContext(ctx)

	cm, ok := obj.(*corev1.ConfigMap)
	if !ok {
		return []reconcile.Request{}
	}

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
		case "StatusUpdateTimeout":
			StatusUpdateTimeout, err = time.ParseDuration(v)
		case "StatusUpdateCheckInterval":
			StatusUpdateCheckInterval, err = time.ParseDuration(v)
		case "ManagerResourcesPath":
			ManagerResourcesPath = v
		default:
			logger.Info("unknown configuration update key", k, v)
		}
		if err != nil {
			logger.Info("failed to parse configuration update", k, err)
		}
	}

	return []reconcile.Request{{NamespacedName: types.NamespacedName{Name: BtpOperatorCrName, Namespace: KymaSystemNamespaceName}}}
}
