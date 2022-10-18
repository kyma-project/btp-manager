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
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
	"time"

	"github.com/kyma-project/btp-manager/operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	btpOperatorGroup               = "services.cloud.sap.com"
	btpOperatorApiVer              = "v1"
	btpOperatorServiceInstance     = "ServiceInstance"
	btpOperatorServiceInstanceList = btpOperatorServiceInstance + "List"
	btpOperatorServiceBinding      = "ServiceBinding"
	btpOperatorServiceBindingList  = btpOperatorServiceBinding + "List"
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
	namespace  string
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

		err := r.HandleDeprovisioning()
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

func (r *BtpOperatorReconciler) HandleDeprovisioning() error {
	logger := log.FromContext(context.Background())

	bindingGvk := schema.GroupVersionKind{Version: btpOperatorApiVer, Group: btpOperatorGroup, Kind: btpOperatorServiceBinding}
	instanceGvk := schema.GroupVersionKind{Version: btpOperatorApiVer, Group: btpOperatorGroup, Kind: btpOperatorServiceInstance}

	logger.Info("btp operator is under deletion")

	if err := r.DeleteDeployment(); err != nil {
		logger.Error(err, "could not delete deployment")
		return err
	}

	if err := r.DeleteWebhooks(); err != nil {
		logger.Error(err, "could not delete webhooks")
		return err
	}

	hardDeleteChannel := make(chan bool)
	go func() {
		anyFail := false
		if err := r.HardDelete(bindingGvk); err != nil {
			logger.Error(err, "failed to hard delete bindings")
			anyFail = true
		}

		if err := r.HardDelete(instanceGvk); err != nil {
			logger.Error(err, "failed to hard delete instances")
			anyFail = true
		}

		//anyFail = true
		hardDeleteChannel <- anyFail
	}()

	select {
	case hardDeleteFail := <-hardDeleteChannel:
		if hardDeleteFail {
			if err := r.HandleHardDeleteFail(&instanceGvk, &bindingGvk); err != nil {
				return err
			}
		} else {
			logger.Info("hard delete success")
			if err := r.RemoveInstalledResources(); err != nil {
				logger.Error(err, "failed to remove related installed resources")
				return err
			}
		}
	case <-time.After(r.cfg.timeout):
		if err := r.HandleHardDeleteFail(&instanceGvk, &bindingGvk); err != nil {
			return err
		}
	}

	return nil
}

func (r *BtpOperatorReconciler) HandleHardDeleteFail(instanceGvk *schema.GroupVersionKind, bindingGvk *schema.GroupVersionKind) error {
	logger := log.FromContext(context.Background())

	logger.Info("hard delete failed. trying to perform soft delete")

	if err := r.SoftDelete(*instanceGvk); err != nil {
		logger.Error(err, "soft deletion of instances failed")
	}

	if err := r.SoftDelete(*bindingGvk); err != nil {
		logger.Error(err, "hard deletion of bindings failed")
	}

	if err := r.RemoveInstalledResources(); err != nil {
		logger.Error(err, "failed to remove related installed resources")
		return err
	}

	return nil
}

func (r *BtpOperatorReconciler) HardDelete(gvk schema.GroupVersionKind) error {
	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(gvk)

	for _, namespace := range r.namespaces.Items {
		listGvk := gvk
		listGvk.Kind = gvk.Kind + "List"

		lst := &unstructured.UnstructuredList{}
		lst.SetGroupVersionKind(listGvk)

		if err := r.List(context.Background(), lst, client.InNamespace(namespace.Name)); err != nil {
			return err
		}

		if err := r.DeleteAllOf(context.Background(), obj, client.InNamespace(namespace.Name)); err != nil {
			return err
		}
	}

	return nil
}

func (r *BtpOperatorReconciler) SoftDelete(gvk schema.GroupVersionKind) error {
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

func (r *BtpOperatorReconciler) RemoveInstalledResources() error {
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

	/*if errs != nil && len(errs.Error()) > 0 {
		return errs
	} else {
		return nil
	}*/
}

func (r *BtpOperatorReconciler) DeleteDeployment() error {
	deployment := &appsv1.Deployment{}
	deployment.Name = "sap-btp-operator-controller-manager"
	deployment.Namespace = "default"
	if err := r.Delete(context.Background(), deployment); err != nil {
		return err
	}
	return nil
}

func (r *BtpOperatorReconciler) DeleteWebhooks() error {
	mutatingWebHook := &admissionregistrationv1.MutatingWebhookConfiguration{}
	mutatingWebHook.Name = "sap-btp-operator-mutating-webhook-configuration"

	if err := r.Delete(context.Background(), mutatingWebHook); err != nil {
		return err
	}

	validatingWebhook := &admissionregistrationv1.ValidatingWebhookConfiguration{}
	validatingWebhook.Name = "sap-btp-operator-validating-webhook-configuration"

	if err := r.Delete(context.Background(), validatingWebhook); err != nil {
		return err
	}

	return nil
}
