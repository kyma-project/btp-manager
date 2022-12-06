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
	"reflect"
	"strings"
	"time"

	"github.com/kyma-project/btp-manager/operator/api/v1alpha1"
	ymlutils "github.com/kyma-project/btp-manager/operator/internal"
	"github.com/kyma-project/module-manager/operator/pkg/manifest"
	"github.com/kyma-project/module-manager/operator/pkg/types"
	"gopkg.in/yaml.v2"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
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
	commonNamespace                = "kyma-system"
	operatorName                   = "btp-manager"
	labelKeyForChart               = "app.kubernetes.io/managed-by"
	secretName                     = "sap-btp-manager"
	deletionFinalizer              = "custom-deletion-finalizer"
	deploymentName                 = "sap-btp-operator-controller-manager"
	mutatingWebhookName            = "sap-btp-operator-mutating-webhook-configuration"
	validatingWebhookName          = "sap-btp-operator-validating-webhook-configuration"
	processingStateRequeueInterval = time.Minute * 5
	readyStateRequeueInterval      = time.Hour * 1
	readyTimeout                   = time.Minute * 1
	chartVersionKey                = "app.kubernetes.io/chart-version"
)

const (
	btpOperatorGroup           = "services.cloud.sap.com"
	btpOperatorApiVer          = "v1"
	btpOperatorServiceInstance = "ServiceInstance"
	btpOperatorServiceBinding  = "ServiceBinding"
	retryInterval              = time.Second * 10
)

const (
	btpManagerConfigMap = "btp-manager-config-map"
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

// BtpOperatorReconciler reconciles a BtpOperator object
type BtpOperatorReconciler struct {
	client.Client
	*rest.Config
	Scheme                *runtime.Scheme
	timeout               time.Duration
	chartDetails          *ChartDetails
	WaitForChartReadiness bool
}

type ChartDetails struct {
	chartPath              string
	oldChartVersion        string
	oldGvks                []schema.GroupVersionKind
	currentChartVersion    string
	currentGvks            []schema.GroupVersionKind
	needToCheckConsistency bool
}

func gvksToStr(gvks []schema.GroupVersionKind) (error, string) {
	bytes, err := yaml.Marshal(gvks)
	if err != nil {
		return err, ""
	}
	return nil, string(bytes)
}

func strToGvks(str string) (error, []schema.GroupVersionKind) {
	var out []schema.GroupVersionKind
	err := yaml.Unmarshal([]byte(str), &out)
	if err != nil {
		return err, nil
	}
	return nil, out
}

func (r *BtpOperatorReconciler) CreateNamespaceIfNeeded() error {
	namespace := &corev1.Namespace{}
	namespace.Name = commonNamespace
	err := r.Get(context.Background(), client.ObjectKeyFromObject(namespace), namespace)
	if errors.IsNotFound(err) {
		err = r.Create(context.Background(), namespace)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	return nil
}

const (
	currentCharVersionKey = "currentCharVersion"
	currentGvksKey        = "currentGvks"
	oldChartVersionKey    = "oldChartVersion"
	oldGvksKey            = "oldGvks"
)

func (r *BtpOperatorReconciler) StoreChartDetails(ctx context.Context, chartPath string) error {
	r.chartDetails = &ChartDetails{}

	r.chartDetails.chartPath = chartPath

	newChartVersion, err := ymlutils.ExtractValueFromLine(fmt.Sprintf("%s/Chart.yaml", r.chartDetails.chartPath), "version")
	if err != nil {
		return err
	}

	newGvks, err := ymlutils.GatherChartGvks(r.chartDetails.chartPath)
	if err != nil {
		return err
	}

	err = r.CreateNamespaceIfNeeded()
	if err != nil {
		return err
	}

	configMap := &corev1.ConfigMap{}
	configMap.Namespace = commonNamespace
	configMap.Name = btpManagerConfigMap

	if err := r.Get(ctx, client.ObjectKey{
		Namespace: configMap.Namespace,
		Name:      configMap.Name,
	}, configMap); err != nil && !errors.IsNotFound(err) {
		return err
	} else if err != nil && errors.IsNotFound(err) {
		configMap.Data = make(map[string]string)

		err, newGvksAsStr := gvksToStr(newGvks)
		if err != nil {
			return err
		}

		configMap.Data[oldChartVersionKey] = newChartVersion
		r.chartDetails.oldChartVersion = newChartVersion
		configMap.Data[oldGvksKey] = newGvksAsStr
		r.chartDetails.oldGvks = newGvks

		configMap.Data[currentCharVersionKey] = newChartVersion
		r.chartDetails.currentChartVersion = newChartVersion
		configMap.Data[currentGvksKey] = newGvksAsStr
		r.chartDetails.currentGvks = newGvks

		if err := r.Create(ctx, configMap); err != nil {
			return err
		}

		r.chartDetails.needToCheckConsistency = true
	} else {
		current, ok := configMap.Data[currentCharVersionKey]
		if !ok {
			return fmt.Errorf("'current' should be present in configmap but it is not")
		}

		if newChartVersion != current {
			err, newGvksAsStr := gvksToStr(newGvks)
			if err != nil {
				return err
			}
			currentGvksStr, ok := configMap.Data["currentGvks"]
			if !ok {
				return fmt.Errorf("'current' should be present in configmap but it is not")
			}
			err, currentGvks := strToGvks(currentGvksStr)
			if err != nil {
				return err
			}

			configMap.Data[oldChartVersionKey] = current
			r.chartDetails.oldChartVersion = current
			configMap.Data[oldGvksKey] = currentGvksStr
			r.chartDetails.oldGvks = currentGvks

			configMap.Data[currentCharVersionKey] = newChartVersion
			r.chartDetails.currentChartVersion = newChartVersion
			r.chartDetails.currentGvks = newGvks
			configMap.Data[currentGvksKey] = newGvksAsStr

			if err := r.Update(ctx, configMap); err != nil {
				return nil
			}

			r.chartDetails.needToCheckConsistency = true
		}
	}

	return nil
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

	if r.chartDetails.needToCheckConsistency {
		r.HandleReadyState(ctx, cr)
		r.chartDetails.needToCheckConsistency = false
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

	ready, err := manifest.InstallChart(logger, installInfo, []types.ObjectTransform{r.labelTransform}, nil)
	if err != nil {
		logger.Error(err, fmt.Sprintf("error while installing resource %s", client.ObjectKeyFromObject(cr)))
		return r.UpdateBtpOperatorState(ctx, cr, types.StateError)
	}
	if ready {
		if err := r.DeleteOrphanedResources(ctx); err != nil {
			return r.UpdateBtpOperatorState(ctx, cr, types.StateError)
		}
		return r.UpdateBtpOperatorState(ctx, cr, types.StateReady)
	}

	return nil
}

func (r *BtpOperatorReconciler) DeleteOrphanedResources(ctx context.Context) error {
	logger := log.FromContext(ctx)
	if r.chartDetails.oldChartVersion != r.chartDetails.currentChartVersion {
		oldVersionLabel := client.MatchingLabels{chartVersionKey: r.chartDetails.oldChartVersion}
		for _, gvk := range r.chartDetails.oldGvks {
			list := &unstructured.UnstructuredList{}
			list.SetGroupVersionKind(gvk)

			if err := r.List(ctx, list, oldVersionLabel); err != nil {
				return err
			}

			for _, item := range list.Items {
				if err := r.Delete(ctx, &item); err != nil {
					return err
				} else {
					logger.Info(fmt.Sprintf("deleted resource %s of type %s with version = %s", item.GetName(), gvk.Kind, r.chartDetails.oldChartVersion))
				}
			}
		}
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
	objKey := client.ObjectKey{Namespace: commonNamespace, Name: secretName}
	if err := r.Get(ctx, objKey, secret); err != nil {
		if errors.IsNotFound(err) {
			return nil, fmt.Errorf("%s Secret in %s namespace not found", secretName, commonNamespace)
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
	cr.Labels[chartVersionKey] = r.chartDetails.currentChartVersion
}

func (r *BtpOperatorReconciler) getInstallInfo(ctx context.Context, cr *v1alpha1.BtpOperator, secret *corev1.Secret) (types.InstallInfo, error) {
	unstructuredObj := &unstructured.Unstructured{}
	unstructuredBase, err := runtime.DefaultUnstructuredConverter.ToUnstructured(cr)
	if err != nil {
		return types.InstallInfo{}, err
	}
	unstructuredObj.Object = unstructuredBase

	installInfo := types.InstallInfo{
		ChartInfo: &types.ChartInfo{
			ChartPath:   r.chartDetails.chartPath,
			ReleaseName: cr.GetName(),
			Flags: types.ChartFlags{
				ConfigFlags: types.Flags{
					"Namespace":       commonNamespace,
					"CreateNamespace": true,
					"Wait":            r.WaitForChartReadiness,
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
		ResourceInfo: types.ResourceInfo{
			BaseResource: unstructuredObj,
		},
		ClusterInfo: types.ClusterInfo{
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

func (r *BtpOperatorReconciler) labelTransform(ctx context.Context, cr types.BaseCustomObject, resourcesFromChart *types.ManifestResources) error {
	baseLabels := cr.GetLabels()
	if _, found := baseLabels[labelKeyForChart]; !found {
		return fmt.Errorf("missing %s label in %s cr resource", labelKeyForChart, cr.GetName())
	}
	for _, item := range resourcesFromChart.Items {
		itemLabels := item.GetLabels()
		if len(itemLabels) == 0 {
			itemLabels = make(map[string]string)
		}
		itemLabels[labelKeyForChart] = baseLabels[labelKeyForChart]
		itemLabels[chartVersionKey] = baseLabels[chartVersionKey]
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
			return secret.Name == secretName && secret.Namespace == commonNamespace
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			secret, ok := e.Object.(*corev1.Secret)
			if !ok {
				return false
			}
			return secret.Name == secretName && secret.Namespace == commonNamespace
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
			if (oldSecret.Name == secretName && oldSecret.Namespace == commonNamespace) &&
				(newSecret.Name == secretName && newSecret.Namespace == commonNamespace) {
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
			if newBtpOperator.GetStatus().State == types.StateError && newBtpOperator.ObjectMeta.DeletionTimestamp.IsZero() {
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

	errs := make([]error, 0)

	sbCrdExists, err := r.crdExists(ctx, bindingGvk)
	if err != nil {
		logger.Error(err, "while checking CRD existence", "GVK", bindingGvk.String())
		errs = append(errs, err)
	}
	if sbCrdExists {
		if err := r.hardDelete(ctx, bindingGvk, namespaces); err != nil {
			logger.Error(err, "while deleting Service Bindings")
			errs = append(errs, err)
		}
	}

	siCrdExists, err := r.crdExists(ctx, instanceGvk)
	if err != nil {
		logger.Error(err, "while checking CRD existence", "GVK", instanceGvk.String())
		errs = append(errs, err)
	}
	if siCrdExists {
		if err := r.hardDelete(ctx, instanceGvk, namespaces); err != nil {
			logger.Error(err, "while deleting Service Instances")
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		success <- false
		return
	}

	var sbResourcesLeft, siResourcesLeft bool
	for {
		select {
		case <-timeout:
			return
		default:
		}

		if sbCrdExists {
			sbResourcesLeft, err = r.resourcesExist(ctx, namespaces, bindingGvk)
			if err != nil {
				logger.Error(err, "ServiceBinding leftover resources check failed")
				success <- false
				return
			}
		}

		if siCrdExists {
			siResourcesLeft, err = r.resourcesExist(ctx, namespaces, instanceGvk)
			if err != nil {
				logger.Error(err, "ServiceInstance leftover resources check failed")
				success <- false
				return
			}
		}

		if !sbResourcesLeft && !siResourcesLeft {
			success <- true
			return
		}

		time.Sleep(retryInterval)
	}
}

func (r *BtpOperatorReconciler) hardDelete(ctx context.Context, gvk schema.GroupVersionKind, namespaces *corev1.NamespaceList) error {
	object := &unstructured.Unstructured{}
	object.SetGroupVersionKind(gvk)
	deleteCtx, cancel := context.WithTimeout(ctx, r.timeout/4)
	defer cancel()

	for _, namespace := range namespaces.Items {
		if err := r.DeleteAllOf(deleteCtx, object, client.InNamespace(namespace.Name)); err != nil {
			return err
		}
	}

	return nil
}

func (r *BtpOperatorReconciler) crdExists(ctx context.Context, gvk schema.GroupVersionKind) (bool, error) {
	crdName := fmt.Sprintf("%ss.%s", strings.ToLower(gvk.Kind), gvk.Group)
	crd := &apiextensionsv1.CustomResourceDefinition{}

	if err := r.Get(ctx, client.ObjectKey{Name: crdName}, crd); err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		} else {
			return false, err
		}
	}
	return true, nil
}

func (r *BtpOperatorReconciler) resourcesExist(ctx context.Context, namespaces *corev1.NamespaceList, gvk schema.GroupVersionKind) (bool, error) {
	anyLeft := func(namespace string, gvk schema.GroupVersionKind) (bool, error) {
		list := &unstructured.UnstructuredList{}
		list.SetGroupVersionKind(gvk)
		if err := r.List(ctx, list, client.InNamespace(namespace)); err != nil {
			if !errors.IsNotFound(err) {
				return false, err
			}
		}

		return len(list.Items) > 0, nil
	}

	for _, namespace := range namespaces.Items {
		resourcesExist, err := anyLeft(namespace.Name, gvk)
		if err != nil {
			return false, err
		}
		if resourcesExist {
			return true, nil
		}
	}

	return false, nil
}

func (r *BtpOperatorReconciler) handleSoftDelete(ctx context.Context, namespaces *corev1.NamespaceList) error {
	logger := log.FromContext(ctx)
	logger.Info("Deprovisioning BTP Operator - soft delete")

	if err := r.preSoftDeleteCleanup(ctx); err != nil {
		logger.Error(err, "module deployment and webhooks deletion failed")
		return err
	}

	sbCrdExists, err := r.crdExists(ctx, bindingGvk)
	if err != nil {
		logger.Error(err, "while checking CRD existence", "GVK", bindingGvk.String())
		return err
	}

	siCrdExists, err := r.crdExists(ctx, instanceGvk)
	if err != nil {
		logger.Error(err, "while checking CRD existence", "GVK", instanceGvk.String())
		return err
	}

	if sbCrdExists {
		if err := r.softDelete(ctx, bindingGvk); err != nil {
			logger.Error(err, "while deleting Service Bindings")
			return err
		}
		if err := r.ensureResourcesDontExist(ctx, bindingGvk); err != nil {
			logger.Error(err, "Service Bindings still exist")
			return err
		}
	}

	if siCrdExists {
		if err := r.softDelete(ctx, instanceGvk); err != nil {
			logger.Error(err, "while deleting Service Instances")
			return err
		}
		if err := r.ensureResourcesDontExist(ctx, instanceGvk); err != nil {
			logger.Error(err, "Service Instances still exist")
			return err
		}
	}

	if err := r.cleanUpAllBtpOperatorResources(ctx, namespaces); err != nil {
		logger.Error(err, "failed to remove chart resources")
		return err
	}

	return nil
}

func (r *BtpOperatorReconciler) GvkToList(gvk schema.GroupVersionKind) *unstructured.UnstructuredList {
	listGvk := gvk
	listGvk.Kind = gvk.Kind + "List"
	list := &unstructured.UnstructuredList{}
	list.SetGroupVersionKind(listGvk)
	return list
}

func (r *BtpOperatorReconciler) ensureResourcesDontExist(ctx context.Context, gvk schema.GroupVersionKind) error {
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

func (r *BtpOperatorReconciler) softDelete(ctx context.Context, gvk schema.GroupVersionKind) error {
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
	/*
		r.deleteDeployment(ctx)
		r.deleteMutatingWebhook(ctx)
		r.deleteValidatingWebhook(ctx)
	*/
	deployment := &appsv1.Deployment{}
	if err := r.Get(ctx, client.ObjectKey{Name: deploymentName, Namespace: commonNamespace}, deployment); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
	} else {
		if err := r.Delete(ctx, deployment); client.IgnoreNotFound(err) != nil {
			return err
		}
	}

	mutatingWebhook := &admissionregistrationv1.MutatingWebhookConfiguration{}
	if err := r.Get(ctx, client.ObjectKey{Name: mutatingWebhookName, Namespace: commonNamespace}, mutatingWebhook); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
	} else {
		if err := r.Delete(ctx, mutatingWebhook); client.IgnoreNotFound(err) != nil {
			return err
		}
	}

	validatingWebhook := &admissionregistrationv1.ValidatingWebhookConfiguration{}
	if err := r.Get(ctx, client.ObjectKey{Name: validatingWebhookName, Namespace: commonNamespace}, validatingWebhook); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
	} else {
		if err := r.Delete(ctx, validatingWebhook); client.IgnoreNotFound(err) != nil {
			return err
		}
	}

	return nil
}

func (r *BtpOperatorReconciler) cleanUpAllBtpOperatorResources(ctx context.Context, namespaces *corev1.NamespaceList) error {
	gvks, err := ymlutils.GatherChartGvks(r.chartDetails.chartPath)
	if err != nil {
		return err
	}

	if err := r.deleteAllOfinstalledResources(ctx, namespaces, gvks); err != nil {
		return err
	}

	return nil
}

func (r *BtpOperatorReconciler) deleteAllOfinstalledResources(ctx context.Context, namespaces *corev1.NamespaceList, gvks []schema.GroupVersionKind) error {
	for _, gvk := range gvks {
		obj := &unstructured.Unstructured{}
		obj.SetGroupVersionKind(gvk)
		if err := r.DeleteAllOf(ctx, obj, client.InNamespace(commonNamespace), labelFilter); err != nil {
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

	ready, err := manifest.ConsistencyCheck(logger, installInfo, []types.ObjectTransform{r.labelTransform}, nil)
	if err != nil {
		logger.Error(err, fmt.Sprintf("error while checking consistency of resource %s", client.ObjectKeyFromObject(cr)))
		return r.UpdateBtpOperatorState(ctx, cr, types.StateError)
	} else if !ready {
		return r.UpdateBtpOperatorState(ctx, cr, types.StateProcessing)
	}

	return nil
}
