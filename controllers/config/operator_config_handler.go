package config

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	enableLimitedCacheKey       = "ENABLE_LIMITED_CACHE"
	sapBtpOperatorConfigMapName = "sap-btp-operator-config"
)

// OperatorConfigHandler watches the sap-btp-operator-config ConfigMap and restarts
// sap-btp-operator pods when ENABLE_LIMITED_CACHE changes.
type OperatorConfigHandler struct {
	client.Client
}

func NewOperatorConfigHandler(c client.Client) *OperatorConfigHandler {
	return &OperatorConfigHandler{Client: c}
}

func (h *OperatorConfigHandler) Object() client.Object {
	return &corev1.ConfigMap{}
}

func (h *OperatorConfigHandler) Predicates() predicate.Funcs {
	isOperatorConfig := func(obj client.Object) bool {
		return obj.GetName() == sapBtpOperatorConfigMapName &&
			obj.GetNamespace() == ChartNamespace
	}
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool { return false },
		DeleteFunc: func(e event.DeleteEvent) bool { return false },
		UpdateFunc: func(e event.UpdateEvent) bool {
			if !isOperatorConfig(e.ObjectNew) {
				return false
			}
			oldCM, ok1 := e.ObjectOld.(*corev1.ConfigMap)
			newCM, ok2 := e.ObjectNew.(*corev1.ConfigMap)
			if !ok1 || !ok2 {
				return false
			}
			return oldCM.Data[enableLimitedCacheKey] != newCM.Data[enableLimitedCacheKey]
		},
	}
}

func (h *OperatorConfigHandler) Reconcile(ctx context.Context, _ client.Object) []reconcile.Request {
	logger := log.FromContext(ctx)
	podList := &corev1.PodList{}
	if err := h.List(ctx, podList,
		client.InNamespace(ChartNamespace),
		client.MatchingLabels{"app.kubernetes.io/instance": "sap-btp-operator"},
	); err != nil {
		logger.Error(err, "failed to list sap-btp-operator pods for restart")
		return []reconcile.Request{}
	}
	for i := range podList.Items {
		pod := &podList.Items[i]
		if err := h.Delete(ctx, pod); err != nil && !k8serrors.IsNotFound(err) {
			logger.Error(err, "failed to delete sap-btp-operator pod", "pod", pod.Name)
		}
	}
	return []reconcile.Request{}
}
