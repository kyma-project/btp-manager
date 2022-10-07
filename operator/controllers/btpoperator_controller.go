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
	"github.com/kyma-project/module-manager/operator/pkg/types"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"time"

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

// SetupWithManager sets up the controller with the Manager.
func (r *BtpOperatorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.BtpOperator{}).
		Complete(r)
}

//+kubebuilder:rbac:groups=operator.kyma-project.io,resources=btpoperators,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=operator.kyma-project.io,resources=btpoperators/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=operator.kyma-project.io,resources=btpoperators/finalizers,verbs=update

func (r *BtpOperatorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	btpManager := v1alpha1.BtpOperator{}
	if err := r.Get(ctx, req.NamespacedName, &btpManager); err != nil {
		logger.Error(err, "failed to get btp manager")
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	deletion := !btpManager.ObjectMeta.DeletionTimestamp.IsZero()
	if deletion {
		err := r.HandleDeprovisioning(ctx, &btpManager)
		if err != nil {
			logger.Error(err, "deprovisioning failed")
			btpManager.Status.State = types.StateError
			return ctrl.Result{}, err
		} else {
			btpManager.Status.State = types.StateReady
			return ctrl.Result{}, nil
		}
	}

	return ctrl.Result{}, nil
}

func (r *BtpOperatorReconciler) HandleDeprovisioning(ctx context.Context, btpManager *v1alpha1.BtpOperator) error {
	logger := log.FromContext(ctx)

	bindingGvk, err := r.GetBtpGvk(btpOperatorServiceBinding)
	if err != nil {
		return err
	}

	instanceGvk, err := r.GetBtpGvk(btpOperatorServiceInstance)
	if err != nil {
		return err
	}

	logger.Info("btp operator is under deletion")

	hardDeleteChannel := make(chan bool)
	go func() {
		anyFail := false
		if err := r.HardDelete(&ctx, bindingGvk); err != nil {
			logger.Error(err, "failed to hard delete bindings")
			anyFail = true
		}

		if err := r.HardDelete(&ctx, instanceGvk); err != nil {
			logger.Error(err, "failed to hard delete instances")
			anyFail = true
		}

		hardDeleteChannel <- anyFail
	}()

	select {
	case hardDeleteOk := <-hardDeleteChannel:
		if hardDeleteOk {
			logger.Info("hard delete success")
			if err := r.RemoveInstalledResources(); err != nil {
				logger.Error(err, "failed to remove related installed resources")
				return err
			}
		} else {
			logger.Info("hard delete failed. trying to perform soft delete")

			if err := r.SoftDelete(&ctx, instanceGvk); err != nil {
				logger.Error(err, "soft deletion of instances failed")
			}

			if err := r.SoftDelete(&ctx, bindingGvk); err != nil {
				logger.Error(err, "hard deletion of bindings failed")
			}

			if err := r.RemoveInstalledResources(); err != nil {
				logger.Error(err, "failed to remove related installed resources")
				return err
			}
		}
	case <-time.After(time.Second * 1):
		fmt.Println("TIMEOUT!!!")
	}

	return nil
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

func (r *BtpOperatorReconciler) SoftDelete(ctx *context.Context, gvk schema.GroupVersionKind) error {
	errs := fmt.Errorf("")

	list := &unstructured.UnstructuredList{}
	list.SetGroupVersionKind(gvk)
	if errs := r.List(*ctx, list); errs != nil {
		errs = fmt.Errorf("%w; could not list in soft delete", errs)
		return errs
	}

	for _, item := range list.Items {
		item.SetFinalizers([]string{})
		if err := r.Update(*ctx, &item); err != nil {
			errs = fmt.Errorf("%w; error occurred in soft delete when updating btp operator resources", err)
		}
	}

	if errs != nil && len(errs.Error()) > 0 {
		return errs
	} else {
		return nil
	}
}

func (r *BtpOperatorReconciler) RemoveInstalledResources() error {
	return nil
}
