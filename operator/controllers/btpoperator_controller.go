/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/kyma-project/btp-manager/operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// BtpOperatorReconciler reconciles a BtpOperator object
type BtpOperatorReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

const (
	btpOperatorGroup           = "services.cloud.sap.com"
	btpOperatorApiVer          = "v1"
	btpOperatorServiceInstance = "ServiceInstance"
	btpOperatorServiceBinding  = "ServiceBinding"
)

//+kubebuilder:rbac:groups=operator.kyma-project.io,resources=btpoperators,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=operator.kyma-project.io,resources=btpoperators/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=operator.kyma-project.io,resources=btpoperators/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the BtpOperator object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *BtpOperatorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("reconcile of deprovisioning started.")

	btpManager := v1alpha1.BtpOperator{}
	if err := r.Get(ctx, req.NamespacedName, &btpManager); err != nil {
		logger.Error(err, "failed to get btp manager")
		if errors.IsNotFound(err) {
			//It means that btp manager was deleted in current Reconcile occurrence.
			//We can use this, because DeletionMode is saved on BTP Managaer Spec (which is reset now)
			return ctrl.Result{}, nil
		}

		return ctrl.Result{}, err
	}

	underDeletion := !btpManager.ObjectMeta.DeletionTimestamp.IsZero()
	if underDeletion {
		logger.Info("btp operator is under deletion")
	}

	// TODO(user): your logic here

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *BtpOperatorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.BtpOperator{}).
		Complete(r)
}

func (r *BtpOperatorReconciler) Deprovision(ctx *context.Context) error {
	bindingGvk, ok := r.GetBtpGvk(ctx, btpOperatorServiceBinding)
	if !ok {
		return nil
	}

	instanceGvk, ok := r.GetBtpGvk(ctx, btpOperatorServiceInstance)
	if !ok {
		return nil
	}

	allDeletionsOk := true

	if errors := r.HardDelete(ctx, bindingGvk); len(errors) > 0 {
		allDeletionsOk = false
	}

	if errors := r.HardDelete(ctx, instanceGvk); len(errors) > 0 {
		allDeletionsOk = false
	}

	if allDeletionsOk {
		//Remove Resources
		if errors := r.SoftDelete(ctx, bindingGvk); len(errors) > 0 {

		}

		if errors := r.SoftDelete(ctx, instanceGvk); len(errors) > 0 {

		}
	} else {
		//Soft Delete

		//Remove Resources
	}

	return nil
}

func (r *BtpOperatorReconciler) GetBtpGvk(ctx *context.Context, kind string) (schema.GroupVersionKind, bool) {
	logger := log.FromContext(*ctx)
	if kind != btpOperatorServiceBinding && kind != btpOperatorServiceInstance {
		logger.Error(nil, "%s as kind not supported.", kind)
		return schema.GroupVersionKind{}, false
	}
	return schema.GroupVersionKind{Version: btpOperatorApiVer, Group: btpOperatorGroup, Kind: kind}, true
}

func (r *BtpOperatorReconciler) SoftDelete(ctx *context.Context, gvk schema.GroupVersionKind) bool {
	logger := log.FromContext(*ctx)
	ok := true

	list := &unstructured.UnstructuredList{}
	list.SetGroupVersionKind(gvk)
	if err := r.List(*ctx, list); err != nil {
		logger.Error(err, "error occurred in soft delete when listing btp operator resources")
		return false
	}

	for _, item := range list.Items {
		item.SetFinalizers([]string{})
		if err := r.Update(*ctx, &item); err != nil {
			logger.Error(err, "error occurred in soft delete when updating btp operator resources")
			ok = false
		}
	}

	return ok
}

func (r *BtpOperatorReconciler) HardDelete(ctx *context.Context, gvk schema.GroupVersionKind) []error {
	logger := log.FromContext(*ctx)
	errors = make([]error, 0)

	if ctx == nil {
		logger.Error(nil, "ctx not set")
		return false
	}

	obj := unstructured.Unstructured{}
	obj.SetGroupVersionKind(gvk)
	if err := r.DeleteAllOf(*ctx, &obj); err != nil {
		logger.Error(err, "delete all of gvk failed")
		return false
	}

	return true
}
