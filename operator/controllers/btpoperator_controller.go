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
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/kyma-project/btp-manager/operator/api/v1alpha1"
	"github.com/kyma-project/module-manager/operator/pkg/custom"
	"github.com/kyma-project/module-manager/operator/pkg/manifest"
	"github.com/kyma-project/module-manager/operator/pkg/types"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	k8sgenerictypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlutil "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	chartPath                      = "./module-chart"
	chartNamespace                 = "kyma-system"
	operatorName                   = "btp-manager"
	labelKeyForChart               = "app.kubernetes.io/managed-by"
	secretName                     = "sap-btp-manager"
	deletionFinalizer              = "custom-deletion-finalizer"
	deploymentName                 = "sap-btp-operator-controller-manager"
	processingStateRequeueInterval = time.Minute * 5
	readyStateRequeueInterval      = time.Hour * 1
	readyTimeout                   = time.Minute * 1
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
	labelFilter = client.MatchingLabels{labelKeyForChart: operatorName}
)

type btpOperatorGvk struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
}

// BtpOperatorReconciler reconciles a BtpOperator object
type BtpOperatorReconciler struct {
	client.Client
	*rest.Config
	Scheme  *runtime.Scheme
	timeout time.Duration
}

func (r *BtpOperatorReconciler) SetTimeout(timeout time.Duration) {
	r.timeout = timeout
}

//+kubebuilder:rbac:groups=operator.kyma-project.io,resources=btpoperators,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=operator.kyma-project.io,resources=btpoperators/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=operator.kyma-project.io,resources=btpoperators/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

func (r *BtpOperatorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	cr := &v1alpha1.BtpOperator{}
	if err := r.Get(ctx, req.NamespacedName, cr); err != nil {
		if errors.IsNotFound(err) {
			logger.Info("BtpOperator resource not found. Ignoring since object must be deleted.")
			return ctrl.Result{}, nil
		}
		logger.Error(err, "unable to fetch BtpOperator resource")
		return ctrl.Result{}, err
	}

	existingBtpOperators := &v1alpha1.BtpOperatorList{}
	if err := r.List(ctx, existingBtpOperators); err != nil {
		logger.Error(err, "unable to fetch existing BtpOperators")
		return ctrl.Result{}, err
	}

	if len(existingBtpOperators.Items) > 1 {
		oldestCr := r.getOldestCR(existingBtpOperators)
		if cr.GetUID() == oldestCr.GetUID() {
			cr.Status.Conditions = nil
		} else {
			return ctrl.Result{}, r.HandleRedundantCR(ctx, oldestCr, cr)
		}
	}

	if ctrlutil.AddFinalizer(cr, deletionFinalizer) {
		return ctrl.Result{}, r.Update(ctx, cr)
	}

	if !cr.ObjectMeta.DeletionTimestamp.IsZero() && cr.Status.State != types.StateDeleting {
		return ctrl.Result{}, r.UpdateBtpOperatorState(ctx, cr, types.StateDeleting)
	}

	switch cr.Status.State {
	case "":
		return ctrl.Result{}, r.HandleInitialState(ctx, cr)
	case types.StateProcessing:
		return ctrl.Result{RequeueAfter: processingStateRequeueInterval}, r.HandleProcessingState(ctx, cr)
	case types.StateError:
		return ctrl.Result{}, r.HandleErrorState(ctx, cr)
	case types.StateDeleting:
		return ctrl.Result{}, r.HandleDeletingState(ctx, cr)
	case types.StateReady:
		return ctrl.Result{RequeueAfter: readyStateRequeueInterval}, r.HandleReadyState(ctx, cr)
	}

	return ctrl.Result{}, nil
}

func (r *BtpOperatorReconciler) getOldestCR(existingBtpOperators *v1alpha1.BtpOperatorList) *v1alpha1.BtpOperator {
	oldestCr := existingBtpOperators.Items[0]
	for _, item := range existingBtpOperators.Items {
		itemCreationTimestamp := &item.CreationTimestamp
		if !(oldestCr.CreationTimestamp.Before(itemCreationTimestamp)) {
			oldestCr = item
		}
	}
	return &oldestCr
}

func (r *BtpOperatorReconciler) HandleRedundantCR(ctx context.Context, oldestCr *v1alpha1.BtpOperator, cr *v1alpha1.BtpOperator) error {
	logger := log.FromContext(ctx)
	logger.Info("Handling redundant BtpOperator CR")

	status := cr.GetStatus()
	status.Conditions = make([]*metav1.Condition, 0)
	errorCondition := &metav1.Condition{
		Type:               "Ready",
		Status:             metav1.ConditionFalse,
		ObservedGeneration: 0,
		LastTransitionTime: metav1.Time{Time: time.Now()},
		Reason:             "OlderCRExists",
		Message: fmt.Sprintf("\"%s\" BtpOperator CR in \"%s\" namespace reconciles the operand",
			oldestCr.GetName(), oldestCr.GetNamespace()),
	}
	status.Conditions = append(status.Conditions, errorCondition)
	cr.SetStatus(status.WithState(types.StateError))
	return r.Status().Update(ctx, cr)
}

func (r *BtpOperatorReconciler) UpdateBtpOperatorState(ctx context.Context, cr *v1alpha1.BtpOperator, newState types.State) error {
	cr.SetStatus(cr.Status.WithState(newState))
	return r.Status().Update(ctx, cr)
}

func (r *BtpOperatorReconciler) HandleInitialState(ctx context.Context, cr *v1alpha1.BtpOperator) error {
	logger := log.FromContext(ctx)
	logger.Info("Handling Initial state")
	return r.UpdateBtpOperatorState(ctx, cr, types.StateProcessing)
}

func (r *BtpOperatorReconciler) HandleProcessingState(ctx context.Context, cr *v1alpha1.BtpOperator) error {
	logger := log.FromContext(ctx)
	logger.Info("Handling Processing state")

	secret, err := r.getRequiredSecret(ctx)
	if err != nil {
		logger.Error(err, "while getting the required Secret")
		return r.UpdateBtpOperatorState(ctx, cr, types.StateError)
	}

	if err = r.verifySecret(secret); err != nil {
		logger.Error(err, "while verifying the required Secret")
		return r.UpdateBtpOperatorState(ctx, cr, types.StateError)
	}

	r.addTempLabelsToCr(cr)

	installInfo, err := r.getInstallInfo(ctx, cr, secret)
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
		return r.UpdateBtpOperatorState(ctx, cr, types.StateError)
	}
	if ready {
		return r.UpdateBtpOperatorState(ctx, cr, types.StateReady)
	}

	return nil
}

func (r *BtpOperatorReconciler) HandleDeletingState(ctx context.Context, cr *v1alpha1.BtpOperator) error {
	logger := log.FromContext(ctx)
	logger.Info("Handling Deleting state")

	if err := r.handleDeprovisioning(ctx); err != nil {
		logger.Error(err, "deprovisioning failed")
		return r.UpdateBtpOperatorState(ctx, cr, types.StateError)
	}
	logger.Info("Deprovisioning success. Clearing finalizers in CR")
	cr.SetFinalizers([]string{})
	if err := r.Update(ctx, cr); err != nil {
		return err
	}
	existingBtpOperators := &v1alpha1.BtpOperatorList{}
	if err := r.List(ctx, existingBtpOperators); err != nil {
		logger.Error(err, "unable to fetch existing BtpOperators")
		return fmt.Errorf("while getting existing BtpOperators: %w", err)
	}
	for _, item := range existingBtpOperators.Items {
		if item.GetUID() == cr.GetUID() {
			continue
		}
		cr := item
		if err := r.UpdateBtpOperatorState(ctx, &cr, types.StateProcessing); err != nil {
			logger.Error(err, "unable to set \"Processing\" state")
		}
	}

	return nil
}

func (r *BtpOperatorReconciler) getRequiredSecret(ctx context.Context) (*corev1.Secret, error) {
	secret := &corev1.Secret{}
	objKey := client.ObjectKey{Namespace: chartNamespace, Name: secretName}
	if err := r.Get(ctx, objKey, secret); err != nil {
		if errors.IsNotFound(err) {
			return nil, fmt.Errorf("%s Secret in %s namespace not found", secretName, chartNamespace)
		}
		return nil, fmt.Errorf("unable to fetch Secret: %w", err)
	}

	return secret, nil
}

func (r *BtpOperatorReconciler) addTempLabelsToCr(cr *v1alpha1.BtpOperator) {
	if len(cr.Labels) == 0 {
		cr.Labels = make(map[string]string)
	}
	cr.Labels[labelKeyForChart] = operatorName
}

func (r *BtpOperatorReconciler) getInstallInfo(ctx context.Context, cr *v1alpha1.BtpOperator, secret *corev1.Secret) (manifest.InstallInfo, error) {
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
					"Timeout":         readyTimeout,
				},
				SetFlags: types.Flags{
					"manager": map[string]interface{}{
						"secret": map[string]interface{}{
							"clientid":     string(secret.Data["clientid"]),
							"clientsecret": string(secret.Data["clientsecret"]),
							"sm_url":       string(secret.Data["sm_url"]),
							"tokenurl":     string(secret.Data["tokenurl"]),
						},
					},
					"cluster": map[string]interface{}{
						"id": string(secret.Data["clientid"]),
					},
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

func (r *BtpOperatorReconciler) verifySecret(secret *corev1.Secret) error {
	missingKeys := make([]string, 0)
	missingValues := make([]string, 0)
	errs := make([]string, 0)
	requiredKeys := []string{"clientid", "clientsecret", "sm_url", "tokenurl", "cluster_id"}
	for _, key := range requiredKeys {
		value, exists := secret.Data[key]
		if !exists {
			missingKeys = append(missingKeys, key)
			continue
		}
		if len(value) == 0 {
			missingValues = append(missingValues, key)
		}
	}
	if len(missingKeys) > 0 {
		missingKeysMsg := fmt.Sprintf("key(s) %s not found", strings.Join(missingKeys, ", "))
		errs = append(errs, missingKeysMsg)
	}
	if len(missingValues) > 0 {
		missingValuesMsg := fmt.Sprintf("missing value(s) for %s key(s)", strings.Join(missingValues, ", "))
		errs = append(errs, missingValuesMsg)
	}
	if len(errs) > 0 {
		return fmt.Errorf("%s", strings.Join(errs, ", "))
	}

	return nil
}

func (r *BtpOperatorReconciler) labelTransform(ctx context.Context, base types.BaseCustomObject, res *types.ManifestResources) error {
	baseLabels := base.GetLabels()
	if _, found := baseLabels[labelKeyForChart]; !found {
		return fmt.Errorf("missing %s label in %s base resource", labelKeyForChart, base.GetName())
	}
	for _, item := range res.Items {
		itemLabels := item.GetLabels()
		if len(itemLabels) == 0 {
			itemLabels = make(map[string]string)
		}
		itemLabels[labelKeyForChart] = baseLabels[labelKeyForChart]
		item.SetLabels(itemLabels)
	}

	return nil
}

func (r *BtpOperatorReconciler) HandleErrorState(ctx context.Context, cr *v1alpha1.BtpOperator) error {
	logger := log.FromContext(ctx)
	logger.Info("Handling Error state")

	return r.UpdateBtpOperatorState(ctx, cr, types.StateProcessing)
}

// SetupWithManager sets up the controller with the Manager.
func (r *BtpOperatorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.Config = mgr.GetConfig()

	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.BtpOperator{},
			builder.WithPredicates(r.watchBtpOperatorUpdatePredicate())).
		Watches(
			&source.Kind{Type: &corev1.Secret{}},
			handler.EnqueueRequestsFromMapFunc(r.reconcileRequestForOldestBtpOperator),
			builder.WithPredicates(r.watchSecretPredicates()),
		).
		Complete(r)
}

func (r *BtpOperatorReconciler) reconcileRequestForOldestBtpOperator(secret client.Object) []reconcile.Request {
	btpOperators := &v1alpha1.BtpOperatorList{}
	err := r.List(context.Background(), btpOperators)
	if err != nil {
		return []reconcile.Request{}
	}
	if len(btpOperators.Items) == 0 {
		return nil
	}
	requests := make([]reconcile.Request, 0)
	oldestCr := r.getOldestCR(btpOperators)
	requests = append(requests, reconcile.Request{NamespacedName: k8sgenerictypes.NamespacedName{Name: oldestCr.GetName(), Namespace: oldestCr.GetNamespace()}})

	return requests
}

func (r *BtpOperatorReconciler) watchSecretPredicates() predicate.Funcs {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			secret, ok := e.Object.(*corev1.Secret)
			if !ok {
				return false
			}
			return secret.Name == secretName && secret.Namespace == chartNamespace
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			secret, ok := e.Object.(*corev1.Secret)
			if !ok {
				return false
			}
			return secret.Name == secretName && secret.Namespace == chartNamespace
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			oldSecret, ok := e.ObjectOld.(*corev1.Secret)
			if !ok {
				return false
			}
			newSecret, ok := e.ObjectNew.(*corev1.Secret)
			if !ok {
				return false
			}
			if (oldSecret.Name == secretName && oldSecret.Namespace == chartNamespace) &&
				(newSecret.Name == secretName && newSecret.Namespace == chartNamespace) {
				return !reflect.DeepEqual(oldSecret.Data, newSecret.Data)
			}
			return false
		},
	}
}

func (r *BtpOperatorReconciler) watchBtpOperatorUpdatePredicate() predicate.Funcs {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return true
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return true
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			newBtpOperator, ok := e.ObjectNew.(*v1alpha1.BtpOperator)
			if !ok {
				return false
			}
			if newBtpOperator.GetStatus().State == types.StateError {
				return false
			}
			return true
		},
		GenericFunc: func(e event.GenericEvent) bool {
			return true
		},
	}
}

func (r *BtpOperatorReconciler) handleDeprovisioning(ctx context.Context) error {
	logger := log.FromContext(ctx)

	namespaces := &corev1.NamespaceList{}
	if err := r.List(ctx, namespaces); err != nil {
		return err
	}

	hardDeleteChannel := make(chan bool)
	timeoutChannel := make(chan bool)
	go r.handleHardDelete(ctx, namespaces, hardDeleteChannel, timeoutChannel)

	select {
	case hardDeleteOk := <-hardDeleteChannel:
		if hardDeleteOk {
			logger.Info("Service Instances and Service Bindings hard delete succeeded. Removing chart resources")
			if err := r.cleanUpAllBtpOperatorResources(ctx, namespaces); err != nil {
				logger.Error(err, "failed to remove chart resources")
				return err
			}
		} else {
			logger.Info("Service Instances and Service Bindings hard delete failed.")
			if err := r.handleSoftDelete(ctx, namespaces); err != nil {
				return err
			}
		}
	case <-time.After(r.timeout):
		logger.Info("hard delete timeout reached", "duration", r.timeout)
		timeoutChannel <- true
		if err := r.handleSoftDelete(ctx, namespaces); err != nil {
			return err
		}
	}

	return nil
}

func (r *BtpOperatorReconciler) handleHardDelete(ctx context.Context, namespaces *corev1.NamespaceList, success chan bool, timeout chan bool) {
	defer close(success)
	defer close(timeout)
	logger := log.FromContext(ctx)
	logger.Info("Deprovisioning BTP Operator - hard delete")

	anyErr := false
	if err := r.hardDelete(ctx, bindingGvk, namespaces); err != nil {
		logger.Error(err, "while deleting bindings")
		anyErr = true
	}
	if err := r.hardDelete(ctx, instanceGvk, namespaces); err != nil {
		logger.Error(err, "while deleting instances")
		anyErr = true
	}

	if anyErr {
		success <- false
		return
	}

	for {
		select {
		case <-timeout:
			return
		default:
		}

		err, resourcesLeft := r.checkIfAnyResourcesLeft(ctx, namespaces)
		if err != nil {
			logger.Error(err, "leftover resources check failed")
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

func (r *BtpOperatorReconciler) hardDelete(ctx context.Context, gvk schema.GroupVersionKind, namespaces *corev1.NamespaceList) error {
	logger := log.FromContext(ctx)
	object := &unstructured.Unstructured{}
	object.SetGroupVersionKind(gvk)

	for _, namespace := range namespaces.Items {
		if err := r.DeleteAllOf(ctx, object, client.InNamespace(namespace.Name)); err != nil {
			logger.Error(err, "while deleting all resources", "kind", object.GetKind())
			return err
		}
	}

	return nil
}

func (r *BtpOperatorReconciler) checkIfAnyResourcesLeft(ctx context.Context, namespaces *corev1.NamespaceList) (error, bool) {
	anyLeft := func(namespace string, gvk schema.GroupVersionKind) (error, bool) {
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
		err, bindingsLeft := anyLeft(namespace.Name, bindingGvk)
		if err != nil {
			return err, true
		}
		if bindingsLeft {
			return nil, true
		}
		err, instancesLeft := anyLeft(namespace.Name, instanceGvk)
		if err != nil {
			return err, true
		}
		if instancesLeft {
			return nil, true
		}
	}

	return nil, false
}

func (r *BtpOperatorReconciler) handleSoftDelete(ctx context.Context, namespaces *corev1.NamespaceList) error {
	logger := log.FromContext(ctx)
	logger.Info("Deprovisioning BTP Operator - soft delete")

	if err := r.preSoftDeleteCleanup(ctx); err != nil {
		return err
	}

	if err := r.softDelete(ctx, &bindingGvk); err != nil {
		logger.Error(err, "while deleting bindings")
		return err
	}
	if err := r.ensureResourcesDontExist(ctx, &bindingGvk); err != nil {
		logger.Error(err, "bindings still exist")
		return err
	}

	if err := r.softDelete(ctx, &instanceGvk); err != nil {
		logger.Error(err, "while deleting instances")
		return err
	}
	if err := r.ensureResourcesDontExist(ctx, &instanceGvk); err != nil {
		logger.Error(err, "instances still exist")
		return err
	}

	if err := r.cleanUpAllBtpOperatorResources(ctx, namespaces); err != nil {
		logger.Error(err, "failed to remove chart resources")
		return err
	}

	return nil
}

func (r *BtpOperatorReconciler) GvkToList(gvk *schema.GroupVersionKind) *unstructured.UnstructuredList {
	listGvk := *gvk
	listGvk.Kind = gvk.Kind + "List"
	list := &unstructured.UnstructuredList{}
	list.SetGroupVersionKind(listGvk)
	return list
}

func (r *BtpOperatorReconciler) ensureResourcesDontExist(ctx context.Context, gvk *schema.GroupVersionKind) error {
	list := r.GvkToList(gvk)

	if err := r.List(ctx, list); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
	} else if len(list.Items) > 0 {
		return fmt.Errorf("list returned %d records", len(list.Items))
	}

	return nil
}

func (r *BtpOperatorReconciler) softDelete(ctx context.Context, gvk *schema.GroupVersionKind) error {
	list := r.GvkToList(gvk)

	if err := r.List(ctx, list); err != nil {
		return fmt.Errorf("%w; could not list in soft delete", err)
	}

	isBinding := gvk.Kind == btpOperatorServiceBinding
	for _, item := range list.Items {
		item.SetFinalizers([]string{})
		if err := r.Update(ctx, &item); err != nil {
			return err
		}

		if isBinding {
			secret := &corev1.Secret{}
			secret.Name = item.GetName()
			secret.Namespace = item.GetNamespace()
			if err := r.Delete(ctx, secret); err != nil && !errors.IsNotFound(err) {
				return err
			}
		}
	}

	return nil
}

func (r *BtpOperatorReconciler) preSoftDeleteCleanup(ctx context.Context) error {
	deployment := &appsv1.Deployment{}
	if err := r.DeleteAllOf(ctx, deployment, labelFilter); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
	}

	mutatingWebhook := &admissionregistrationv1.MutatingWebhookConfiguration{}
	if err := r.DeleteAllOf(ctx, mutatingWebhook, labelFilter); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
	}

	validatingWebhook := &admissionregistrationv1.ValidatingWebhookConfiguration{}
	if err := r.DeleteAllOf(ctx, validatingWebhook, labelFilter); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
	}

	return nil
}

func (r *BtpOperatorReconciler) cleanUpAllBtpOperatorResources(ctx context.Context, namespaces *corev1.NamespaceList) error {
	time.Sleep(time.Second * 10)

	gvks, err := r.gatherChartGvks()
	if err != nil {
		return err
	}

	if err := r.deleteAllOfinstalledResources(ctx, namespaces, gvks); err != nil {
		return err
	}

	return nil
}

func (r *BtpOperatorReconciler) gatherChartGvks() ([]schema.GroupVersionKind, error) {
	var allGvks []schema.GroupVersionKind
	appendToSlice := func(gvk schema.GroupVersionKind) {
		if reflect.DeepEqual(gvk, schema.GroupVersionKind{}) {
			return
		}
		for _, v := range allGvks {
			if reflect.DeepEqual(gvk, v) {
				return
			}
		}
		allGvks = append(allGvks, gvk)
	}

	root := fmt.Sprintf("%s/templates/", chartPath)
	if err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !strings.HasSuffix(info.Name(), ".yml") {
			return nil
		}

		bytes, err := os.ReadFile(fmt.Sprintf("%s/%s", root, info.Name()))
		if err != nil {
			return err
		}

		fileGvks, err := r.extractGvkFromYml(string(bytes))
		if err != nil {
			return err
		}

		for _, gvk := range fileGvks {
			appendToSlice(gvk)
		}

		return nil
	}); err != nil {
		return []schema.GroupVersionKind{}, err
	}

	return allGvks, nil
}

func (r *BtpOperatorReconciler) extractGvkFromYml(wholeFile string) ([]schema.GroupVersionKind, error) {
	var gvks []schema.GroupVersionKind
	parts := strings.Split(wholeFile, "---\n")
	for _, part := range parts {
		if part == "" {
			continue
		}
		var yamlGvk btpOperatorGvk
		lines := strings.Split(part, "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "apiVersion:") {
				yamlGvk.APIVersion = strings.TrimSpace(strings.Split(line, ":")[1])
			}

			if strings.HasPrefix(line, "kind:") {
				yamlGvk.Kind = strings.TrimSpace(strings.Split(line, ":")[1])
			}
		}
		if yamlGvk.Kind != "" && yamlGvk.APIVersion != "" {
			apiVersion := strings.Split(yamlGvk.APIVersion, "/")
			if len(apiVersion) == 1 {
				gvks = append(gvks, schema.GroupVersionKind{
					Kind:    yamlGvk.Kind,
					Version: apiVersion[0],
					Group:   "",
				})
			} else if len(apiVersion) == 2 {
				gvks = append(gvks, schema.GroupVersionKind{
					Kind:    yamlGvk.Kind,
					Version: apiVersion[1],
					Group:   apiVersion[0],
				})
			} else {
				return nil, fmt.Errorf("incorrect split of apiVersion")
			}
		}
	}

	return gvks, nil
}

func (r *BtpOperatorReconciler) deleteAllOfinstalledResources(ctx context.Context, namespaces *corev1.NamespaceList, gvks []schema.GroupVersionKind) error {
	for _, gvk := range gvks {
		obj := &unstructured.Unstructured{}
		obj.SetGroupVersionKind(gvk)
		if err := r.DeleteAllOf(ctx, obj, client.InNamespace(chartNamespace), labelFilter); err != nil {
			if !errors.IsNotFound(err) && !errors.IsMethodNotSupported(err) && !meta.IsNoMatchError(err) {
				return err
			}
		}
	}
	return nil
}

func (r *BtpOperatorReconciler) HandleReadyState(ctx context.Context, cr *v1alpha1.BtpOperator) error {
	logger := log.FromContext(ctx)
	logger.Info("Handling Ready state")

	secret, err := r.getRequiredSecret(ctx)
	if err != nil {
		logger.Error(err, "while getting the required Secret")
		return r.UpdateBtpOperatorState(ctx, cr, types.StateError)
	}

	if err = r.verifySecret(secret); err != nil {
		logger.Error(err, "while verifying the required Secret")
		return r.UpdateBtpOperatorState(ctx, cr, types.StateError)
	}

	r.addTempLabelsToCr(cr)

	installInfo, err := r.getInstallInfo(ctx, cr, secret)
	if err != nil {
		logger.Error(err, "while preparing InstallInfo")
		return err
	}
	if installInfo.ChartPath == "" {
		return fmt.Errorf("no chart path available for processing")
	}

	ready, err := manifest.ConsistencyCheck(&logger, installInfo, []types.ObjectTransform{r.labelTransform})
	if err != nil {
		logger.Error(err, fmt.Sprintf("error while checking consistency of resource %s", client.ObjectKeyFromObject(cr)))
		return r.UpdateBtpOperatorState(ctx, cr, types.StateError)
	} else if !ready {
		return r.UpdateBtpOperatorState(ctx, cr, types.StateProcessing)
	}

	return nil
}
