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
	"github.com/kyma-project/module-manager/operator/pkg/custom"
	"github.com/kyma-project/module-manager/operator/pkg/manifest"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlutil "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	chartPath         = "/Users/lj/Go/src/github.com/kyma-project/modularization/btp-manager/operator/module-chart"
	chartNamespace    = "kyma-system"
	operatorName      = "btp-manager"
	labelKey          = "app.kubernetes.io/managed-by"
	deletionFinalizer = "custom-deletion-finalizer"
	requeueInterval   = time.Second * 3
)

type ReconcilerConfig struct {
	timeout                        time.Duration
	btpOperatorGroup               string
	btpOperatorApiVer              string
	btpOperatorServiceInstance     string
	btpOperatorServiceInstanceList string
	btpOperatorServiceBinding      string
	btpOperatorServiceBindingList  string
	namespaces                     corev1.NamespaceList
	fakeHardDeletionFailForTest    bool
}

func NewReconcileConfig(apiVer string, operatorGroup string, bindingKind string, instanceKind string, timeout time.Duration, fakeFail bool) ReconcilerConfig {
	return ReconcilerConfig{
		timeout:                        timeout,
		btpOperatorGroup:               operatorGroup,
		btpOperatorApiVer:              apiVer,
		btpOperatorServiceInstance:     instanceKind,
		btpOperatorServiceInstanceList: instanceKind + "List",
		btpOperatorServiceBinding:      bindingKind,
		btpOperatorServiceBindingList:  bindingKind + "List",
		fakeHardDeletionFailForTest:    fakeFail,
	}
}

func (r *BtpOperatorReconciler) SetReconcileConfig(reconcilerConfig ReconcilerConfig) {
	r.reconcilerConfig = reconcilerConfig
}

// BtpOperatorReconciler reconciles a BtpOperator object
type BtpOperatorReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	*rest.Config
	reconcilerConfig ReconcilerConfig
	namespaces       corev1.NamespaceList
}

func (r *BtpOperatorReconciler) GetGvk(kind string) schema.GroupVersionKind {
	return schema.GroupVersionKind{
		Group:   r.reconcilerConfig.btpOperatorGroup,
		Version: r.reconcilerConfig.btpOperatorApiVer,
		Kind:    kind,
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

	btpOperatorCr := &v1alpha1.BtpOperator{}
	if err := r.Get(ctx, req.NamespacedName, btpOperatorCr); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		logger.Error(err, "unable to fetch BtpOperator")
		return ctrl.Result{}, err
	}

	if ctrlutil.AddFinalizer(btpOperatorCr, deletionFinalizer) {
		return ctrl.Result{}, r.Update(ctx, btpOperatorCr)
	}

	if !btpOperatorCr.ObjectMeta.DeletionTimestamp.IsZero() && btpOperatorCr.Status.State != types.StateDeleting {
		return ctrl.Result{}, r.SetDeletingState(ctx, btpOperatorCr)
	}

	switch btpOperatorCr.Status.State {
	case "":
		return ctrl.Result{}, r.HandleInitialState(ctx, btpOperatorCr)
	case types.StateProcessing:
		return ctrl.Result{RequeueAfter: requeueInterval}, r.HandleProcessingState(ctx, btpOperatorCr)
	case types.StateDeleting:
		return ctrl.Result{}, r.HandleDeletingState(ctx, btpOperatorCr)
	}

	/*
		var existingBtpOperators v1alpha1.BtpOperatorList
		if err := r.List(ctx, &existingBtpOperators); err != nil {
			logger.Error(err, "unable to fetch existing BtpOperators")
			return ctrl.Result{}, err
		}
	*/

	return ctrl.Result{}, nil
}

func (r *BtpOperatorReconciler) SetStatus(new types.State, ctx context.Context, cr *v1alpha1.BtpOperator) error {
	status := cr.GetStatus()
	status.WithState(new)
	cr.SetStatus(status)
	return r.Status().Update(ctx, cr)
}

func (r *BtpOperatorReconciler) SetDeletingState(ctx context.Context, cr *v1alpha1.BtpOperator) error {
	return r.SetStatus(types.StateDeleting, ctx, cr)
}

func (r *BtpOperatorReconciler) HandleInitialState(ctx context.Context, cr *v1alpha1.BtpOperator) error {
	return r.SetStatus(types.StateProcessing, ctx, cr)
}

func (r *BtpOperatorReconciler) HandleProcessingState(ctx context.Context, cr *v1alpha1.BtpOperator) error {
	logger := log.FromContext(ctx)

	r.addTempLabelsToCr(cr)

	installInfo, err := r.getInstallInfo(ctx, cr)
	if err != nil {
		logger.Error(err, "while preparing InstallInfo")
		return err
	}
	if installInfo.ChartPath == "" {
		return fmt.Errorf("no chart path available for processing")
	}

	ready, err := manifest.InstallChart(&logger, installInfo, []types.ObjectTransform{r.labelTransform})
	if err != nil {
		logger.Error(err, fmt.Sprintf("error while installing resource %s", client.ObjectKeyFromObject(cr)))
		return r.SetStatus(types.StateError, ctx, cr)
	}
	if ready {
		return r.SetStatus(types.StateReady, ctx, cr)
	}

	return nil
}

func (r *BtpOperatorReconciler) HandleDeletingState(ctx context.Context, cr *v1alpha1.BtpOperator) error {
	logger := log.FromContext(context.Background())

	r.namespaces = corev1.NamespaceList{}
	if err := r.List(ctx, &(r.namespaces)); err != nil {
		logger.Error(err, "cannot list namespaces")
		return err
	}

	err := r.handleDeprovisioning()
	if err != nil {
		logger.Error(err, "deprovisioning failed")
		return r.SetStatus(types.StateError, ctx, cr)
	} else {
		cr.SetFinalizers([]string{})
		if err := r.Update(ctx, cr); err != nil {
			return err
		}
		return r.SetStatus(types.StateReady, ctx, cr)
	}

	return nil
}

func (r *BtpOperatorReconciler) addTempLabelsToCr(cr *v1alpha1.BtpOperator) {
	if len(cr.Labels) == 0 {
		cr.Labels = make(map[string]string)
	}
	cr.Labels[labelKey] = operatorName
}

func (r *BtpOperatorReconciler) getInstallInfo(ctx context.Context, cr *v1alpha1.BtpOperator) (manifest.InstallInfo, error) {
	unstructuredObj := &unstructured.Unstructured{}
	unstructuredBase, err := runtime.DefaultUnstructuredConverter.ToUnstructured(cr)
	if err != nil {
		return manifest.InstallInfo{}, err
	}
	unstructuredObj.Object = unstructuredBase

	installInfo := manifest.InstallInfo{
		ChartInfo: &manifest.ChartInfo{
			ChartPath:   chartPath,
			ReleaseName: cr.GetName(),
			Flags: types.ChartFlags{
				ConfigFlags: types.Flags{
					"Namespace":       chartNamespace,
					"CreateNamespace": true,
					"Wait":            true,
				},
			},
		},
		ResourceInfo: manifest.ResourceInfo{
			BaseResource: unstructuredObj,
		},
		ClusterInfo: custom.ClusterInfo{
			Config: r.Config,
			Client: r.Client,
		},
		Ctx: ctx,
	}

	return installInfo, nil
}

func (r *BtpOperatorReconciler) labelTransform(ctx context.Context, base types.BaseCustomObject, res *types.ManifestResources) error {
	baseLabels := base.GetLabels()
	if _, found := baseLabels[labelKey]; !found {
		return fmt.Errorf("missing %s label in %s base resource", labelKey, base.GetName())
	}
	for _, item := range res.Items {
		itemLabels := item.GetLabels()
		if len(itemLabels) == 0 {
			itemLabels = make(map[string]string)
		}
		itemLabels[labelKey] = baseLabels[labelKey]
		item.SetLabels(itemLabels)
	}
	return nil
}

func (r *BtpOperatorReconciler) handleDeprovisioning() error {
	logger := log.FromContext(context.Background())

	bindingGvk := r.GetGvk(r.reconcilerConfig.btpOperatorServiceBinding)
	instanceGvk := r.GetGvk(r.reconcilerConfig.btpOperatorServiceInstance)

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
			return
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
			if err := r.handleHardDeleteFail(&instanceGvk, &bindingGvk); err != nil {
				return err
			}
		}
	case <-time.After(r.reconcilerConfig.timeout):
		if err := r.handleHardDeleteFail(&instanceGvk, &bindingGvk); err != nil {
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

func (r *BtpOperatorReconciler) handleHardDeleteFail(instanceGvk *schema.GroupVersionKind, bindingGvk *schema.GroupVersionKind) error {
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

	if r.reconcilerConfig.fakeHardDeletionFailForTest {
		return errors.NewServiceUnavailable("Not avaiable due to test mode.")
	}

	return nil
}

func (r *BtpOperatorReconciler) softDelete(gvk *schema.GroupVersionKind) error {
	errs := fmt.Errorf("")
	listGvk := gvk
	listGvk.Kind = gvk.Kind + "List"
	list := &unstructured.UnstructuredList{}
	list.SetGroupVersionKind(*listGvk)
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
