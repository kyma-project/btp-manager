package controllers

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"
)

// InstanceBindingControllerManager runs and stops the ServiceInstance controller
type InstanceBindingControllerManager struct {
	client.Client
	*rest.Config
	Scheme *runtime.Scheme

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
		Scheme:                 r.Scheme,
		Metrics:                server.Options{BindAddress: "0"},
		HealthProbeBindAddress: "0",
	})
	if err != nil {
		slog.Error(fmt.Sprintf("unable to create controller manager: %v", err))
		return
	}
	r.mgr = mgr

	if r.sisbReconciler != nil {
		r.sisbReconciler = NewServiceInstanceReconciler(r.Client, r.Scheme)
		err = r.sisbReconciler.SetupWithManager(r.mgr)
		if err != nil {
			slog.Error(fmt.Sprintf("unable to create SI SB controller: %v", err))
			return
		}
	}
	r.enabled = true

	contextWithCancel, cancel := context.WithCancel(r.ctx)
	r.stopper = cancel
	go func() {
		err := r.mgr.Start(contextWithCancel)
		if err != nil {
			slog.Error(fmt.Sprintf("unable to start SI SB controller: %v", err))
		} else {
			slog.Info("SI SB controller goroutine stopped")
		}
	}()

}

func (r *InstanceBindingControllerManager) DisableSISBController() {
	r.sisbControllerMu.Lock()
	defer r.sisbControllerMu.Unlock()
	if !r.enabled {
		return
	}

	if r.stopper != nil {
		r.stopper()
	}
	r.enabled = false

}
