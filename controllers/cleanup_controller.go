package controllers

import (
	"context"
	"fmt"
	"github.com/kyma-project/btp-manager/api/v1alpha1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"strings"
	"sync"
	"time"
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

func NewCleanupReconciler(client client.Client, scheme *runtime.Scheme, cfg *rest.Config) *CleanupReconciler {
	return &CleanupReconciler{
		Client: client,
		Scheme: scheme,
		cfg:    cfg,
	}
}

func (r *CleanupReconciler) crdExists(ctx context.Context, gvk schema.GroupVersionKind) (bool, error) {
	crdName := fmt.Sprintf("%ss.%s", strings.ToLower(gvk.Kind), gvk.Group)
	crd := &apiextensionsv1.CustomResourceDefinition{}

	if err := r.Get(ctx, client.ObjectKey{Name: crdName}, crd); err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		} else {
			return false, err
		}
	}
	return true, nil
}

func (r *CleanupReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Reconcile Cleanup1")

	sbCrdExists, err := r.crdExists(ctx, bindingGvk)
	if err != nil {
		logger.Error(err, "while checking SB CRD existence", "GVK", bindingGvk.String())
		return ctrl.Result{}, err
	}
	if !sbCrdExists {
		return ctrl.Result{}, nil
	}
	siCrdExists, err := r.crdExists(ctx, instanceGvk)
	if err != nil {
		logger.Error(err, "while checking SI CRD existence", "GVK", instanceGvk.String())
		return ctrl.Result{}, err
	}
	if !siCrdExists {
		return ctrl.Result{}, nil
	}

	existingBtpOperators := &v1alpha1.BtpOperatorList{}
	if err := r.List(ctx, existingBtpOperators); err != nil {
		return ctrl.Result{}, err
	}

	if len(existingBtpOperators.Items) > 0 {
		logger.Info(" - enabling SISB controller")
		r.EnableSISBController()
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *CleanupReconciler) SetupWithManager(mgr ctrl.Manager, cfg *rest.Config) error {
	r.cfg = cfg
	return ctrl.NewControllerManagedBy(mgr).
		For(&apiextensionsv1.CustomResourceDefinition{}, builder.WithPredicates(r.predicate())).
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

func (r *CleanupReconciler) EnableSISBController() {
	fmt.Println("EnableSISBController")
	r.sisbControllerMu.Lock()
	defer r.sisbControllerMu.Unlock()

	if r.enabled {
		return
	}

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
		time.Sleep(500 * time.Millisecond)
		err = r.mgr.Start(ctx) // todo: handle error
	}()

}

//func (r *CleanupReconciler) disableSISBController() {
//	r.sisbControllerMu.Lock()
//	defer r.sisbControllerMu.Unlock()
//	if !r.enabled {
//		return
//	}
//
//	r.enabled = false
//	fmt.Println("DISABLING")
//	if r.sisbReconcilerStopper != nil {
//		r.sisbReconcilerStopper()
//	}
//}

func (r *CleanupReconciler) DisableSISBController() {

	r.sisbControllerMu.Lock()
	defer r.sisbControllerMu.Unlock()
	fmt.Println("*** DISABLE ***")
	if !r.enabled {
		return
	}

	r.enabled = false
	fmt.Println("DISABLING")
	if r.sisbReconcilerStopper != nil {
		r.sisbReconcilerStopper()
	}

}
