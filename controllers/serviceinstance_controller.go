package controllers

import (
	"context"

	"github.com/kyma-project/btp-manager/api/v1alpha1"
	"github.com/kyma-project/btp-manager/internal/conditions"
	"github.com/kyma-project/module-manager/pkg/types"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// ServiceInstanceReconciler reconciles a BtpOperator object in case of service instance changes
type ServiceInstanceReconciler struct {
	client.Client
	*rest.Config
	Scheme *runtime.Scheme
}

func NewServiceInstanceReconciler(client client.Client, scheme *runtime.Scheme) *ServiceInstanceReconciler {
	return &ServiceInstanceReconciler{
		Client: client,
		Scheme: scheme,
	}
}

func (r *ServiceInstanceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	list := &unstructured.UnstructuredList{}
	list.SetGroupVersionKind(instanceGvk)
	err := r.List(ctx, list, client.InNamespace(corev1.NamespaceAll))
	if err != nil {
		return ctrl.Result{}, err
	}
	if len(list.Items) != 0 {
		return ctrl.Result{}, nil
	}

	list.SetGroupVersionKind(bindingGvk)
	err = r.List(ctx, list, client.InNamespace(corev1.NamespaceAll))
	if err != nil {
		return ctrl.Result{}, err
	}
	if len(list.Items) != 0 {
		return ctrl.Result{}, nil
	}

	btpOperator, err := r.getOldestBtpOperator(ctx)
	if err != nil {
		return ctrl.Result{}, err
	}
	// btp operator already removed
	if btpOperator == nil {
		return ctrl.Result{}, nil
	}
	if btpOperator.IsReasonStringEqual(string(conditions.ServiceInstancesAndBindingsNotCleaned)) {
		return ctrl.Result{}, r.UpdateBtpOperatorStatus(ctx, btpOperator, types.StateDeleting, conditions.HardDeleting, "BtpOperator is to be deleted")
	}

	return ctrl.Result{}, nil
}

func (r *ServiceInstanceReconciler) UpdateBtpOperatorStatus(ctx context.Context, cr *v1alpha1.BtpOperator, newState types.State, reason conditions.Reason, message string) error {
	cr.Status.WithState(newState)
	newCondition := conditions.ConditionFromExistingReason(reason, message)
	if newCondition != nil {
		conditions.SetStatusCondition(&cr.Status.Conditions, *newCondition)
	}
	return r.Status().Update(ctx, cr)
}

// SetupWithManager sets up the controller with the Manager.
func (r *ServiceInstanceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.Config = mgr.GetConfig()

	si := &unstructured.Unstructured{}
	si.SetGroupVersionKind(instanceGvk)
	sb := &unstructured.Unstructured{}
	sb.SetGroupVersionKind(bindingGvk)

	return ctrl.NewControllerManagedBy(mgr).
		For(si,
			builder.WithPredicates(r.deletionPredicate())).
		Watches(&source.Kind{Type: sb},
			&handler.EnqueueRequestForObject{},
			builder.WithPredicates(r.deletionPredicate())).
		Complete(r)
}

func (r *ServiceInstanceReconciler) deletionPredicate() predicate.Predicate {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return false
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return true
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			return false
		},
		GenericFunc: func(e event.GenericEvent) bool {
			return false
		},
	}
}

func (r *ServiceInstanceReconciler) getOldestBtpOperator(ctx context.Context) (*v1alpha1.BtpOperator, error) {
	logger := log.FromContext(ctx)
	existingBtpOperators := &v1alpha1.BtpOperatorList{}
	if err := r.List(ctx, existingBtpOperators); err != nil {
		logger.Error(err, "unable to get existing BtpOperator CRs")
		return nil, err
	}

	if len(existingBtpOperators.Items) == 0 {
		return nil, nil
	}

	oldestCr := existingBtpOperators.Items[0]
	for _, item := range existingBtpOperators.Items {
		itemCreationTimestamp := &item.CreationTimestamp
		if !(oldestCr.CreationTimestamp.Before(itemCreationTimestamp)) {
			oldestCr = item
		}
	}
	return &oldestCr, nil
}
