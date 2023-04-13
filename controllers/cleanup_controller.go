package controllers

import (
	"context"
	"fmt"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sync"
	"time"
)

// InstanceBindingControllerManager xxx
type InstanceBindingControllerManager struct {
	client.Client
	*rest.Config
	Scheme *runtime.Scheme
	//manifestHandler *manifest.Handler
	workqueueSize int

	sisbControllerMu sync.Mutex
	sisbReconciler   *ServiceInstanceReconciler
	mgr              ctrl.Manager
	stopper          func()
	enabled          bool
	cfg              *rest.Config
	ctx              context.Context
}

func NewInstanceBindingControllerManager(ctx context.Context, client client.Client, scheme *runtime.Scheme, cfg *rest.Config) *InstanceBindingControllerManager {
	return &InstanceBindingControllerManager{
		Client: client,
		Scheme: scheme,
		cfg:    cfg,
		ctx:    ctx,
	}
}

func (r *InstanceBindingControllerManager) EnableSISBController() {
	r.sisbControllerMu.Lock()
	defer r.sisbControllerMu.Unlock()

	if r.enabled {
		return
	}
	mgr, err := ctrl.NewManager(r.cfg, ctrl.Options{
		Scheme:             r.Scheme,
		MetricsBindAddress: "0",
	})
	if err != nil {
		return
	}
	r.mgr = mgr

	r.sisbReconciler = NewServiceInstanceReconciler(r.Client, r.Scheme)
	err = r.sisbReconciler.SetupWithManager(r.mgr)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	r.enabled = true

	_, cancel := context.WithCancel(r.ctx)
	r.stopper = cancel
	go func() {
		time.Sleep(5 * time.Millisecond)
		err = r.mgr.Start(r.ctx) // todo: handle error
	}()

}

func (r *InstanceBindingControllerManager) DisableSISBController() {
	r.sisbControllerMu.Lock()
	defer r.sisbControllerMu.Unlock()

	if !r.enabled {
		return
	}

	r.enabled = false
	if r.stopper != nil {
		go func() {
			r.stopper()
		}()
	}

}
