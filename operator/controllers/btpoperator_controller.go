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
	"fmt"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/kyma-project/btp-manager/operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	btpOperatorGroup           = "services.cloud.sap.com"
	btpOperatorApiVer          = "v1"
	btpOperatorServiceInstance = "ServiceInstance"
	btpOperatorServiceBinding  = "ServiceBinding"
)

// BtpOperatorReconciler reconciles a BtpOperator object
type BtpOperatorReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

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
	_ = log.FromContext(ctx)

	_, _ = r.HandleDeprovisioning(ctx, req)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *BtpOperatorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.BtpOperator{}).
		Complete(r)
}

func (r *BtpOperatorReconciler) HandleDeprovisioning(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	btpManager := v1alpha1.BtpOperator{}
	if err := r.Get(ctx, req.NamespacedName, &btpManager); err != nil {
		logger.Error(err, "failed to get btp manager")
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}

		return ctrl.Result{}, err
	}

	markedForDelete := !btpManager.ObjectMeta.DeletionTimestamp.IsZero()
	if markedForDelete {
		logger.Info("btp operator is under deletion")
	}

	return ctrl.Result{}, nil
}

func (r *BtpOperatorReconciler) GetBtpGvk(kind string) (schema.GroupVersionKind, error) {
	if kind != btpOperatorServiceBinding && kind != btpOperatorServiceInstance {
		return schema.GroupVersionKind{}, fmt.Errorf("%s as kind not supported", kind)
	}
	return schema.GroupVersionKind{Version: btpOperatorApiVer, Group: btpOperatorGroup, Kind: kind}, nil
}

func (r *BtpOperatorReconciler) HardDelete(ctx *context.Context, gvk schema.GroupVersionKind) error {
	if ctx == nil {
		return fmt.Errorf("hard delete: ctx not set")
	}

	obj := unstructured.Unstructured{}
	obj.SetGroupVersionKind(gvk)
	if err := r.DeleteAllOf(*ctx, &obj); err != nil {
		return err
	}

	return nil
}

func (r *BtpOperatorReconciler) SoftDelete(ctx *context.Context, gvk schema.GroupVersionKind) []error {
	errs := make([]error, 0)

	list := &unstructured.UnstructuredList{}
	list.SetGroupVersionKind(gvk)
	if err := r.List(*ctx, list); err != nil {
		errs = append(errs, err)
		return errs
	}

	for _, item := range list.Items {
		item.SetFinalizers([]string{})
		if err := r.Update(*ctx, &item); err != nil {
			errs = append(errs, fmt.Errorf("error occurred in soft delete when updating btp operator resources"))
		}
	}

	return nil
}

func (r *BtpOperatorReconciler) RemoveInstalledResources() {

}
