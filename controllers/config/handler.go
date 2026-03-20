package config

import (
	"context"
	"strconv"
	"time"

	"github.com/kyma-project/btp-manager/internal/certs"
	"github.com/kyma-project/btp-manager/internal/metrics"

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

	EnableLimitedCache = "true"
)

type WatchHandler interface {
	Object() client.Object
	Predicates() predicate.Funcs
	Reconcile(ctx context.Context, obj client.Object) []reconcile.Request
}

type Handler struct {
	client.Client
	Scheme        *runtime.Scheme
	configMetrics *metrics.ConfigMetrics
}

func NewHandler(client client.Client, scheme *runtime.Scheme, configMetrics *metrics.ConfigMetrics) *Handler {
	return &Handler{
		Client:        client,
		Scheme:        scheme,
		configMetrics: configMetrics,
	}
}

func (r *Handler) Object() client.Object {
	return &corev1.ConfigMap{}
}

func (r *Handler) Predicates() predicate.Funcs {
	nameMatches := func(o client.Object) bool {
		return o.GetName() == ConfigName && o.GetNamespace() == ChartNamespace
	}
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			matches := nameMatches(e.Object)
			if matches {
				r.configMetrics.ConfigMapApplied()
			}
			return matches
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			matches := nameMatches(e.Object)
			if matches {
				r.configMetrics.ConfigMapNotApplied()
			}
			return matches
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			matches := nameMatches(e.ObjectNew)
			if matches {
				r.configMetrics.ConfigMapApplied()
			}
			return matches
		},
	}
}

func (r *Handler) Start(ctx context.Context) error {
	logger := log.FromContext(ctx)
	cm := &corev1.ConfigMap{}
	err := r.Get(ctx, types.NamespacedName{Name: ConfigName, Namespace: ChartNamespace}, cm)
	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			return err
		}
		logger.Info("custom configuration ConfigMap not found at startup")
		return nil
	}
	logger.Info("custom configuration ConfigMap found at startup, setting metric to 1")
	r.configMetrics.ConfigMapApplied()
	return nil
}

func (r *Handler) Reconcile(ctx context.Context, obj client.Object) []reconcile.Request {
	logger := log.FromContext(ctx)
	parseDuration := func(raw string, defaultValue time.Duration, key string) time.Duration {
		parsed, err := time.ParseDuration(raw)
		if err != nil {
			logger.Info("failed to parse configuration update", key, err)
			return defaultValue
		}
		return parsed
	}

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
			ProcessingStateRequeueInterval = parseDuration(v, ProcessingStateRequeueInterval, k)
		case "ReadyStateRequeueInterval":
			ReadyStateRequeueInterval = parseDuration(v, ReadyStateRequeueInterval, k)
		case "ReadyTimeout":
			ReadyTimeout = parseDuration(v, ReadyTimeout, k)
		case "HardDeleteCheckInterval":
			HardDeleteCheckInterval = parseDuration(v, HardDeleteCheckInterval, k)
		case "HardDeleteTimeout":
			HardDeleteTimeout = parseDuration(v, HardDeleteTimeout, k)
		case "ResourcesPath":
			ResourcesPath = v
		case "ReadyCheckInterval":
			ReadyCheckInterval = parseDuration(v, ReadyCheckInterval, k)
		case "DeleteRequestTimeout":
			DeleteRequestTimeout = parseDuration(v, DeleteRequestTimeout, k)
		case "CaCertificateExpiration":
			CaCertificateExpiration = parseDuration(v, CaCertificateExpiration, k)
		case "WebhookCertificateExpiration":
			WebhookCertificateExpiration = parseDuration(v, WebhookCertificateExpiration, k)
		case "ExpirationBoundary":
			ExpirationBoundary = parseDuration(v, ExpirationBoundary, k)
		case "RsaKeyBits":
			var bits int
			bits, err = strconv.Atoi(v)
			if err == nil {
				certs.SetRsaKeyBits(bits)
			}
		case "EnableLimitedCache":
			EnableLimitedCache = v
		case "StatusUpdateTimeout":
			StatusUpdateTimeout = parseDuration(v, StatusUpdateTimeout, k)
		case "StatusUpdateCheckInterval":
			StatusUpdateCheckInterval = parseDuration(v, StatusUpdateCheckInterval, k)
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
