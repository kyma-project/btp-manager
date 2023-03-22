package controllers

import (
	"context"
	"fmt"
	"github.com/kyma-project/btp-manager/api/v1alpha1"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	v1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sync"
)

// CleanupReconciler xxx
type CleanupReconciler struct {
	client.Client
	*rest.Config
	Scheme *runtime.Scheme
	//manifestHandler *manifest.Handler
	workqueueSize int

	sisbControllerMu      sync.Mutex
	sisbReconciler        *ServiceInstanceReconciler
	mgr                   ctrl.Manager
	sisbReconcilerStopper func()
	enabled               bool
	cfg                   *rest.Config
}

func NewCleanupReconciler(client client.Client, scheme *runtime.Scheme) *CleanupReconciler {
	return &CleanupReconciler{
		Client: client,
		Scheme: scheme,
	}
}

func (r *CleanupReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	fmt.Println("*** CLEANUP RECONCILER ***")

	si := apiextensions.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: "serviceinstances.services.cloud.sap.com",
		},
	}
	err := r.Get(ctx, client.ObjectKey{Name: "serviceinstances.services.cloud.sap.com"}, &si)
	switch {
	case errors.IsNotFound(err):
		fmt.Println("not found, OK")
		return ctrl.Result{}, nil
	case err != nil:
		fmt.Printf("error1: %s", err.Error())
	}

	sb := apiextensions.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: "servicebindings.services.cloud.sap.com",
		},
	}
	err = r.Get(ctx, client.ObjectKey{Name: "servicebindings.services.cloud.sap.com"}, &sb)
	switch {
	case errors.IsNotFound(err):
		fmt.Println("not found, OK")
		return ctrl.Result{}, nil
	case err != nil:
		fmt.Printf("error2: %s", err.Error())
	}

	existingBtpOperators := &v1alpha1.BtpOperatorList{}
	if err := r.List(ctx, existingBtpOperators); err != nil {
		return ctrl.Result{}, err
	}

	if len(existingBtpOperators.Items) > 0 {
		r.enableSISBController()
	} else {
		r.disableSISBController()
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *CleanupReconciler) SetupWithManager(mgr ctrl.Manager, cfg *rest.Config) error {
	r.cfg = cfg
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1.CustomResourceDefinition{}, builder.WithPredicates(r.predicate())).
		Complete(r)
}

func (r *CleanupReconciler) predicate() predicate.Predicate {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return true
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return true
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			return false
		},
	}
}

func (r *CleanupReconciler) enableSISBController() {
	r.sisbControllerMu.Lock()
	defer r.sisbControllerMu.Unlock()

	if r.enabled {
		return
	}

	fmt.Println("ENABLING")
	// todo: handle error
	mgr, err := ctrl.NewManager(r.cfg, ctrl.Options{
		Scheme:                 r.Scheme,
		MetricsBindAddress:     "0",
		HealthProbeBindAddress: "",
		ReadinessEndpointName:  "",
		LivenessEndpointName:   "",
	})
	if err != nil {
		fmt.Println("ERROR:", err.Error())
		return
	}
	r.mgr = mgr
	fmt.Println("ENABLING1")

	r.sisbReconciler = NewServiceInstanceReconciler(r.Client, r.Scheme)
	err = r.sisbReconciler.SetupWithManager(r.mgr)

	if err != nil {
		fmt.Println(err.Error())
		return
	}

	fmt.Println("ENABLING2")

	r.enabled = true

	// todo: signal handling
	ctx, cancel := context.WithCancel(ctrl.SetupSignalHandler())
	r.sisbReconcilerStopper = cancel
	go func() {
		fmt.Println("STARTING")
		err = r.mgr.Start(ctx) // todo: handle error
		fmt.Println(err)
	}()

}

func (r *CleanupReconciler) disableSISBController() {
	r.sisbControllerMu.Lock()
	defer r.sisbControllerMu.Unlock()
	if !r.enabled {
		return
	}

	r.enabled = false
	fmt.Println("DISABLING")
	if r.sisbReconcilerStopper != nil {
		r.sisbReconcilerStopper()
	}
}
