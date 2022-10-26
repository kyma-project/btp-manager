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
	v1 "k8s.io/api/apps/v1"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
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
	operatorName      = "btp-operator"
	labelKey          = "managed-by"
	deletionFinalizer = "custom-deletion-finalizer"
	requeueInterval   = time.Second * 3
)

const (
	btpOperatorGroup           = "services.cloud.sap.com"
	btpOperatorApiVer          = "v1"
	btpOperatorServiceInstance = "ServiceInstance"
	btpOperatorServiceBinding  = "ServiceBinding"
	retryInterval              = time.Second * 10
)

var (
	bindingGvk = schema.GroupVersionKind{
		Group:   btpOperatorGroup,
		Version: btpOperatorApiVer,
		Kind:    btpOperatorServiceBinding,
	}
	instanceGvk = schema.GroupVersionKind{
		Group:   btpOperatorGroup,
		Version: btpOperatorApiVer,
		Kind:    btpOperatorServiceInstance,
	}
	labelFilter = client.MatchingLabels{labelKey: operatorName}
)

type ReconcilerConfig struct {
	timeout            time.Duration
	hardDeleteTestMode bool
}

func NewReconcileConfig(timeout time.Duration, hardDeleteTestMode bool) ReconcilerConfig {
	return ReconcilerConfig{
		timeout:            timeout,
		hardDeleteTestMode: hardDeleteTestMode,
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
	logger := log.FromContext(ctx)
	if err := r.handleDeprovisioning(ctx); err != nil {
		logger.Error(err, "deprovisioning failed")
		return r.SetStatus(types.StateError, ctx, cr)
	} else {
		logger.Info("deprovisioning success. clearing finalizers for btp manager")
		cr.SetFinalizers([]string{})
		if err := r.Update(ctx, cr); err != nil {
			return err
		}
		return r.SetStatus(types.StateReady, ctx, cr)
	}
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

func (r *BtpOperatorReconciler) handleDeprovisioning(ctx context.Context) error {
	logger := log.FromContext(ctx)
	logger.Info("btp operator is under deletion")

	namespaces := &corev1.NamespaceList{}
	if err := r.List(ctx, namespaces); err != nil {
		return err
	}
	if err := r.handlePreDelete(ctx); err != nil {
		return err
	}

	hardDeleteChannel := make(chan bool)
	go r.handleHardDelete(ctx, namespaces, hardDeleteChannel)

	select {
	case hardDeleteOk := <-hardDeleteChannel:
		if hardDeleteOk {
			logger.Info("hard delete success")
			if err := r.cleanUpAllBtpOperatorResources(ctx, namespaces); err != nil {
				logger.Error(err, "failed to remove related installed resources")
				return err
			}
		} else {
			if err := r.handleSoftDelete(ctx, namespaces); err != nil {
				return err
			}
		}
	case <-time.After(r.reconcilerConfig.timeout):
		if err := r.handleSoftDelete(ctx, namespaces); err != nil {
			return err
		}
	}

	return nil
}

func (r *BtpOperatorReconciler) handleHardDelete(ctx context.Context, namespaces *corev1.NamespaceList, success chan bool) {
	defer close(success)
	logger := log.FromContext(ctx)
	anyErr := false

	if err := r.hardDelete(ctx, bindingGvk, namespaces); err != nil {
		anyErr = true
	}

	if err := r.hardDelete(ctx, instanceGvk, namespaces); err != nil {
		anyErr = true
	}

	if r.reconcilerConfig.hardDeleteTestMode {
		anyErr = true
	}

	if anyErr {
		success <- false
		return
	}

	for {
		err, resourcesLeft := r.checkIfAnyResourcesLeft(ctx, namespaces)
		if err != nil {
			logger.Error(err, "Checking failed")
			success <- false
			return
		}
		if !resourcesLeft {
			success <- true
			return
		}
		time.Sleep(retryInterval)
	}
}

func (r *BtpOperatorReconciler) checkIfAnyResourcesLeft(ctx context.Context, namespaces *corev1.NamespaceList) (error, bool) {
	list := func(namespace string, gvk schema.GroupVersionKind) (error, bool) {
		list := &unstructured.UnstructuredList{}
		list.SetGroupVersionKind(gvk)
		if err := r.List(ctx, list, client.InNamespace(namespace)); err != nil {
			if !errors.IsNotFound(err) {
				return err, true
			}
		}

		return nil, len(list.Items) > 0
	}

	for _, namespace := range namespaces.Items {
		var err error
		err, instancesLeft := list(namespace.Name, instanceGvk)
		if err != nil {
			return err, true
		}
		err, bindingsLeft := list(namespace.Name, bindingGvk)
		if err != nil {
			return err, true
		}
		if instancesLeft == false && bindingsLeft == false {
			return nil, false
		}
	}

	return nil, true
}

func (r *BtpOperatorReconciler) handleSoftDelete(ctx context.Context, namespaces *corev1.NamespaceList) error {
	logger := log.FromContext(ctx)
	logger.Info("hard delete failed. trying to perform soft delete")

	if err := r.softDelete(ctx, &instanceGvk); err != nil {
		logger.Error(err, "soft deletion of instances failed")
		return err
	}

	if err := r.softDelete(ctx, &bindingGvk); err != nil {
		logger.Error(err, "hard deletion of bindings failed")
		return err
	}

	if err := r.cleanUpAllBtpOperatorResources(ctx, namespaces); err != nil {
		logger.Error(err, "failed to remove related installed resources")
		return err
	}

	return nil
}

func (r *BtpOperatorReconciler) hardDelete(ctx context.Context, gvk schema.GroupVersionKind, namespaces *corev1.NamespaceList) error {
	logger := log.FromContext(ctx)
	object := &unstructured.Unstructured{}
	object.SetGroupVersionKind(gvk)

	for _, namespace := range namespaces.Items {
		if err := r.DeleteAllOf(ctx, object, client.InNamespace(namespace.Name)); err != nil {
			logger.Error(err, "Err while doing delete all of")
			return err
		}
	}

	return nil
}

func (r *BtpOperatorReconciler) softDelete(ctx context.Context, gvk *schema.GroupVersionKind) error {
	listGvk := *gvk
	listGvk.Kind = gvk.Kind + "List"
	list := &unstructured.UnstructuredList{}
	list.SetGroupVersionKind(listGvk)

	if err := r.List(ctx, list); err != nil {
		err = fmt.Errorf("%w; could not list in soft delete", err)
		return err
	}

	for _, item := range list.Items {
		item.SetFinalizers([]string{})
		if err := r.Update(context.Background(), &item); err != nil {
			return err
		}
	}

	return nil
}

func (r *BtpOperatorReconciler) handlePreDelete(ctx context.Context) error {
	deployment := &v1.Deployment{}
	deployment.Name = "sap-btp-operator-controller-manager"
	deployment.Namespace = "kyma-system"
	if err := r.Delete(ctx, deployment); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
	}

	mutatingWebhook := &admissionregistrationv1.MutatingWebhookConfiguration{}
	if err := r.DeleteAllOf(ctx, mutatingWebhook); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
	}

	validatingWebhook := &admissionregistrationv1.ValidatingWebhookConfiguration{}
	if err := r.DeleteAllOf(ctx, validatingWebhook); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
	}

	return nil
}

func (r *BtpOperatorReconciler) cleanUpAllBtpOperatorResources(ctx context.Context, namespaces *corev1.NamespaceList) error {
	time.Sleep(time.Second * 10)

	err, gvks := r.discoverDeletableGvks()
	if err != nil {
		return err
	}

	if err := r.deleteAllOfinstalledResources(ctx, namespaces, gvks); err != nil {
		return err
	}

	return nil
}

func (r *BtpOperatorReconciler) discoverDeletableGvks() (error, []schema.GroupVersionKind) {
	var err error
	cs, err := clientset.NewForConfig(r.Config)
	if err != nil {
		return fmt.Errorf("failed to create clientset from config %w", err), []schema.GroupVersionKind{}
	}

	_, apiResourceList, err := cs.ServerGroupsAndResources()
	if err != nil {
		return fmt.Errorf("failed to get server group and resources %w", err), []schema.GroupVersionKind{}
	}

	gvks := make([]schema.GroupVersionKind, 0)
	for _, resourceMap := range apiResourceList {
		gv, _ := schema.ParseGroupVersion(resourceMap.GroupVersion)
		for _, apiResource := range resourceMap.APIResources {
			for _, verb := range apiResource.Verbs {
				if verb == "delete" || verb == "deletecollection" {
					gvks = append(gvks, schema.GroupVersionKind{
						Version: gv.Version,
						Group:   gv.Group,
						Kind:    apiResource.Kind,
					})
					break
				}
			}
		}
	}

	return nil, gvks
}

func (r *BtpOperatorReconciler) deleteAllOfinstalledResources(ctx context.Context, namespaces *corev1.NamespaceList, gvks []schema.GroupVersionKind) error {
	logger := log.FromContext(ctx)
	for _, gvk := range gvks {
		object := &unstructured.Unstructured{}
		object.SetGroupVersionKind(gvk)
		for _, namespace := range namespaces.Items {
			if err := r.DeleteAllOf(ctx, object, client.InNamespace(namespace.Name), labelFilter); err != nil {
				if !errors.IsNotFound(err) && !errors.IsMethodNotSupported(err) && !meta.IsNoMatchError(err) {
					return err
				} else {
					logger.Error(err, "failed to use deleteallof on given resource")
				}
			}
		}
	}
	return nil
}
