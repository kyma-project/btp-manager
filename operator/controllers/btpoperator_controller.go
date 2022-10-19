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
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
	"time"

	"github.com/kyma-project/btp-manager/operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	chartPath                      = "./module-chart"
	chartNamespace                 = "kyma-system"
	operatorName                   = "btp-manager"
	labelKey                       = "app.kubernetes.io/managed-by"
	deletionFinalizer              = "custom-deletion-finalizer"
	requeueInterval                = time.Second * 3
	btpOperatorGroup               = "services.cloud.sap.com"
	btpOperatorApiVer              = "v1"
	btpOperatorServiceInstance     = "ServiceInstance"
	btpOperatorServiceInstanceList = btpOperatorServiceInstance + "List"
	btpOperatorServiceBinding      = "ServiceBinding"
	btpOperatorServiceBindingList  = btpOperatorServiceBinding + "List"
)

var (
	bindingGvk  = schema.GroupVersionKind{Version: btpOperatorApiVer, Group: btpOperatorGroup, Kind: btpOperatorServiceBinding}
	instanceGvk = schema.GroupVersionKind{Version: btpOperatorApiVer, Group: btpOperatorGroup, Kind: btpOperatorServiceInstance}
)

type Cfg struct {
	timeout time.Duration
}

func NewCfg(time time.Duration) *Cfg {
	return &Cfg{
		timeout: time,
	}
}

// BtpOperatorReconciler reconciles a BtpOperator object
type BtpOperatorReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	*rest.Config
	cfg        *Cfg
	namespaces corev1.NamespaceList
}

func (r *BtpOperatorReconciler) SetupCfg(cfg *Cfg) {
	if cfg != nil {
		r.cfg = cfg
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *BtpOperatorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.Config = mgr.GetConfig()
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.BtpOperator{}).
		Complete(r)
}

//+kubebuilder:rbac:groups=operator.kyma-project.io,resources=btpoperators,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=operator.kyma-project.io,resources=btpoperators/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=operator.kyma-project.io,resources=btpoperators/finalizers,verbs=update

func (r *BtpOperatorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(context.Background())

	btpManager := v1alpha1.BtpOperator{}
	if err := r.Get(context.Background(), req.NamespacedName, &btpManager); err != nil {
		logger.Error(err, "failed to get btp manager")
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	deletionTimestampSet := btpManager.ObjectMeta.DeletionTimestamp.IsZero() == false
	if deletionTimestampSet && btpManager.Status.State != types.StateDeleting {
		btpManager.Status.State = types.StateDeleting

		r.namespaces = corev1.NamespaceList{}
		if err := r.List(context.Background(), &(r.namespaces)); err != nil {
			logger.Error(err, "cannot list namespaces")
			return ctrl.Result{}, err
		}

		err := r.handleDeprovisioning()
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

func (r *BtpOperatorReconciler) handleDeprovisioning() error {
	logger := log.FromContext(context.Background())

	logger.Info("btp operator is under deletion")

	hardDeleteChannel := make(chan bool)
	go func(success chan bool) {
		anyErr := false
		if err := r.hardDelete(bindingGvk); err != nil {
			logger.Error(err, "failed to hard delete bindings")
			anyErr = true
		}

		if err := r.hardDelete(instanceGvk); err != nil {
			logger.Error(err, "failed to hard delete instances")
			anyErr = true
		}

		if anyErr {
			success <- false
		}

		for {
			resourcesLeft := true
			for _, namespace := range r.namespaces.Items {

				bindings := &unstructured.UnstructuredList{}
				bindings.SetGroupVersionKind(bindingGvk)
				if err := r.List(context.Background(), bindings, client.InNamespace(namespace.Name)); err != nil {
					if !errors.IsNotFound(err) {
						logger.Error(err, "failed to list bindings")
					}
				}

				instances := &unstructured.UnstructuredList{}
				instances.SetGroupVersionKind(instanceGvk)
				if err := r.List(context.Background(), instances, client.InNamespace(namespace.Name)); err != nil {
					if !errors.IsNotFound(err) {
						logger.Error(err, "failed to list instances")
					}
				}

				resourcesLeft = len(bindings.Items) > 0 || len(instances.Items) > 0
			}

			if !resourcesLeft {
				success <- true
				return
			}

			time.Sleep(time.Second * 10)
		}
	}(hardDeleteChannel)

	select {
	case hardDeleteOk := <-hardDeleteChannel:
		if hardDeleteOk {
			logger.Info("hard delete success")
			if err := r.removeInstalledResources(); err != nil {
				logger.Error(err, "failed to remove related installed resources")
				return err
			}
		} else {
			if err := r.handleHardDeleteFail(); err != nil {
				return err
			}
		}
	case <-time.After(r.cfg.timeout):
		if err := r.handleHardDeleteFail(); err != nil {
			return err
		}
	}

	return nil
}

func (r *BtpOperatorReconciler) deleteDeployment() error {
	deployment := &appsv1.Deployment{}
	deployment.Name = "sap-btp-operator-controller-manager"
	deployment.Namespace = "default"
	if err := r.Delete(context.Background(), deployment); err != nil {
		return err
	}
	return nil
}

func (r *BtpOperatorReconciler) handleHardDeleteFail() error {
	logger := log.FromContext(context.Background())

	logger.Info("hard delete failed. trying to perform soft delete")

	if err := r.softDelete(instanceGvk); err != nil {
		logger.Error(err, "soft deletion of instances failed")
	}

	if err := r.softDelete(bindingGvk); err != nil {
		logger.Error(err, "hard deletion of bindings failed")
	}

	if err := r.removeInstalledResources(); err != nil {
		logger.Error(err, "failed to remove related installed resources")
		return err
	}

	return nil
}

func (r *BtpOperatorReconciler) hardDelete(gvk schema.GroupVersionKind) error {
	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(gvk)

	for _, namespace := range r.namespaces.Items {
		if err := r.DeleteAllOf(context.Background(), obj, client.InNamespace(namespace.Name)); err != nil {
			return err
		}
	}

	return nil
}

func (r *BtpOperatorReconciler) softDelete(gvk schema.GroupVersionKind) error {
	errs := fmt.Errorf("")
	listGvk := gvk
	listGvk.Kind = gvk.Kind + "List"
	list := &unstructured.UnstructuredList{}
	list.SetGroupVersionKind(listGvk)
	if errs := r.List(context.Background(), list); errs != nil {
		errs = fmt.Errorf("%w; could not list in soft delete", errs)
		return errs
	}

	for _, item := range list.Items {
		item.SetFinalizers([]string{})
		if err := r.Update(context.Background(), &item); err != nil {
			errs = fmt.Errorf("%w; error occurred in soft delete when updating btp operator resources", err)
		}
	}

	if errs != nil && len(errs.Error()) > 0 {
		return errs
	} else {
		return nil
	}
}

func (r *BtpOperatorReconciler) removeInstalledResources() error {
	time.Sleep(time.Second * 30)
	c, cerr := clientset.NewForConfig(r.Config)
	if cerr != nil {
		return cerr
	}

	_, apiResourceList, err := c.ServerGroupsAndResources()
	if err != nil {
		return err
	}

	errs := fmt.Errorf("")
	//var edeb []error
	for _, apiResource := range apiResourceList {
		gv, _ := schema.ParseGroupVersion(apiResource.GroupVersion)
		for _, apiResourceNested := range apiResource.APIResources {
			gvk := schema.GroupVersionKind{
				Version: gv.Version,
				Group:   gv.Group,
				Kind:    apiResourceNested.Kind,
			}

			var hasDeleteVerb bool = false
			for _, verb := range apiResourceNested.Verbs {
				if verb == "delete" || verb == "deletecollection" {
					hasDeleteVerb = true
					break
				}
			}

			if !hasDeleteVerb {
				continue
			}

			obj := &unstructured.Unstructured{}
			obj.SetGroupVersionKind(gvk)

			if apiResourceNested.Namespaced {
				for _, namespace := range r.namespaces.Items {
					if err := r.DeleteAllOf(context.Background(), obj, client.InNamespace(namespace.Name), client.MatchingLabels{"managed-by": "btp-operator"}); err != nil {
						if !errors.IsNotFound(err) {
							errs = fmt.Errorf("%w; error occurred in soft delete when updating btp operator resources", err)
							//edeb = append(edeb, err)
						}
					}
				}
			} else {
				if err := r.DeleteAllOf(context.Background(), obj, client.MatchingLabels{"managed-by": "btp-operator"}); err != nil {
					if !errors.IsNotFound(err) {
						errs = fmt.Errorf("%w; error occurred in soft delete when updating btp operator resources", err)
						//edeb = append(edeb, err)
					}
				}
			}
		}
	}

	fmt.Print(errs)
	return nil
}
