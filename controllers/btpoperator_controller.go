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
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/kyma-project/btp-manager/api/v1alpha1"
	"github.com/kyma-project/btp-manager/internal/gvksutils"
	"github.com/kyma-project/btp-manager/internal/ymlutils"
	"github.com/kyma-project/module-manager/pkg/manifest"
	"github.com/kyma-project/module-manager/pkg/types"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
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

var (
	// Configuration options that can be overwritten either by CLI parameter or ConfigMap
	ChartNamespace                 = "kyma-system"
	SecretName                     = "sap-btp-manager"
	ConfigName                     = "sap-btp-manager"
	DeploymentName                 = "sap-btp-operator-controller-manager"
	ProcessingStateRequeueInterval = time.Minute * 5
	ReadyStateRequeueInterval      = time.Hour * 1
	ReadyTimeout                   = time.Minute * 1
	HardDeleteCheckInterval        = time.Second * 10
	HardDeleteTimeout              = time.Minute * 20
	ChartPath                      = "./module-chart"
)

const (
	operatorName          = "btp-manager"
	labelKeyForChart      = "app.kubernetes.io/managed-by"
	deletionFinalizer     = "custom-deletion-finalizer"
	mutatingWebhookName   = "sap-btp-operator-mutating-webhook-configuration"
	validatingWebhookName = "sap-btp-operator-validating-webhook-configuration"
)

const (
	btpOperatorGroup           = "services.cloud.sap.com"
	btpOperatorApiVer          = "v1"
	btpOperatorServiceInstance = "ServiceInstance"
	btpOperatorServiceBinding  = "ServiceBinding"
)

const (
	chartVersionKey       = "chart-version"
	btpManagerConfigMap   = "btp-manager-versions"
	oldChartVersionKey    = "oldChartVersion"
	oldGvksKey            = "oldGvks"
	currentCharVersionKey = "currentChartVersion"
	currentGvksKey        = "currentGvks"
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

type ErrorWithReason struct {
	message string
	reason  Reason
}

func NewErrorWithReason(reason Reason, message string) *ErrorWithReason {
	return &ErrorWithReason{
		message: message,
		reason:  reason,
	}
}

func (e *ErrorWithReason) Error() string {
	return e.message
}

// BtpOperatorReconciler reconciles a BtpOperator object
type BtpOperatorReconciler struct {
	client.Client
	*rest.Config
	Scheme                *runtime.Scheme
	currentVersion        string
	updateCheckDone       bool
	WaitForChartReadiness bool
	workqueueSize         int
}

func (r *BtpOperatorReconciler) handleUpdate(ctx context.Context, cr *v1alpha1.BtpOperator, configMap *corev1.ConfigMap) *ErrorWithReason {
	errorWithReason := r.chartConsistencyCheck(ctx, cr)
	if errorWithReason != nil {
		return errorWithReason
	}

	if err := r.deleteOrphanedResources(ctx, configMap); err != nil {
		return NewErrorWithReason(DeletionOfOrphanedResourcesFailed, "Deletion of orphaned resources failed")
	}
	return nil
}

func (r *BtpOperatorReconciler) createChartNamespaceIfNeeded(ctx context.Context) error {
	namespace := &corev1.Namespace{}
	namespace.Name = ChartNamespace
	err := r.Get(ctx, client.ObjectKeyFromObject(namespace), namespace)
	if k8serrors.IsNotFound(err) {
		err = r.Create(ctx, namespace)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	return nil
}

func (r *BtpOperatorReconciler) buildBtpManagerConfigMap() *corev1.ConfigMap {
	configMap := &corev1.ConfigMap{}
	configMap.Namespace = ChartNamespace
	configMap.Name = btpManagerConfigMap
	return configMap
}

func (r *BtpOperatorReconciler) getBtpManagerConfigMap(ctx context.Context) (*corev1.ConfigMap, error) {

	configMap := r.buildBtpManagerConfigMap()
	err := r.Get(ctx, client.ObjectKeyFromObject(configMap), configMap)

	if err != nil && !k8serrors.IsNotFound(err) {
		return nil, err
	}
	return configMap, nil
}

func (r *BtpOperatorReconciler) storeChartDetails(ctx context.Context, configMap *corev1.ConfigMap) (bool, error) {
	newChartVersion, err := ymlutils.ExtractStringValueFromYamlForGivenKey(fmt.Sprintf("%s/Chart.yaml", ChartPath), "version")
	if err != nil {
		return false, err
	}

	newGvks, err := ymlutils.GatherChartGvks(ChartPath)
	if err != nil {
		return false, err
	}

	err = r.createChartNamespaceIfNeeded(ctx)
	if err != nil {
		return false, err
	}

	if configMap.Data == nil {
		configMap = r.buildBtpManagerConfigMap()
		err = r.handleInitialConfigMap(ctx, configMap, newChartVersion, newGvks)
		if err != nil {
			return false, err
		}
		return false, nil
	} else {
		return r.handleExistingConfigMap(ctx, configMap, newChartVersion, newGvks)
	}
}

func (r *BtpOperatorReconciler) handleInitialConfigMap(ctx context.Context, configMap *corev1.ConfigMap, newChartVersion string, newGvks []schema.GroupVersionKind) error {
	logger := log.FromContext(ctx)
	logger.Info(fmt.Sprintf("%s dosent exists, new will be created", btpManagerConfigMap))

	configMap.Data = make(map[string]string)

	newGvksAsStr, err := gvksutils.GvksToStr(newGvks)
	if err != nil {
		return err
	}

	r.setBtpManagerConfigMap(ctx, configMap, newChartVersion, newChartVersion, newGvksAsStr, newGvksAsStr)

	if err := r.Create(ctx, configMap); err != nil {
		return err
	}

	r.currentVersion = newChartVersion

	return nil
}

func (r *BtpOperatorReconciler) handleExistingConfigMap(ctx context.Context, configMap *corev1.ConfigMap, newVersion string, newGvks []schema.GroupVersionKind) (bool, error) {
	logger := log.FromContext(ctx)
	logger.Info(fmt.Sprintf("%s exists", btpManagerConfigMap))

	currentVersion, ok := configMap.Data[currentCharVersionKey]
	if !ok {
		return false, fmt.Errorf("required key %s is missing in configmap", currentCharVersionKey)
	}

	if newVersion != currentVersion {
		logger.Info(fmt.Sprintf("detected version change: {%s} -> {%s}", currentVersion, newVersion))

		currentGvksAsText, ok := configMap.Data[currentGvksKey]
		if !ok {
			return false, fmt.Errorf("required key %s is missing in configmap", currentGvksAsText)
		}

		newGvksAsText, err := gvksutils.GvksToStr(newGvks)
		if err != nil {
			return false, err
		}

		r.setBtpManagerConfigMap(ctx, configMap, currentVersion, newVersion, currentGvksAsText, newGvksAsText)

		if err := r.Update(ctx, configMap); err != nil {
			return false, err
		}

		logger.Info(fmt.Sprintf("%s has been updated", btpManagerConfigMap))

		r.currentVersion = newVersion

		return true, nil
	}

	r.currentVersion = currentVersion

	return false, nil
}

func (r *BtpOperatorReconciler) setBtpManagerConfigMap(ctx context.Context, configMap *corev1.ConfigMap, oldChartVersion, currentCharVersion, oldGvksStr, currentGvks string) {
	if configMap == nil {
		return
	}

	configMap.Data[oldChartVersionKey] = oldChartVersion
	configMap.Data[oldGvksKey] = oldGvksStr
	configMap.Data[currentCharVersionKey] = currentCharVersion
	configMap.Data[currentGvksKey] = currentGvks
}

func (r *BtpOperatorReconciler) deleteOrphanedResources(ctx context.Context, configMap *corev1.ConfigMap) error {
	logger := log.FromContext(ctx)
	logger.Info("deletion of orphaned resources started")

	oldVersion, ok := configMap.Data[oldChartVersionKey]
	if !ok {
		return fmt.Errorf("%s should be present in configmap but it is not", oldChartVersionKey)
	}

	oldGvksText, ok := configMap.Data[oldGvksKey]
	if !ok {
		return fmt.Errorf("%s should be present in configmap but it is not", oldGvksKey)
	}

	oldGvks, err := gvksutils.StrToGvks(oldGvksText)
	if err != nil {
		return err
	}

	oldVersionLabel := client.MatchingLabels{chartVersionKey: oldVersion}

	numberOfDeletedItems := 0
	for _, gvk := range oldGvks {
		list := &unstructured.UnstructuredList{}
		list.SetGroupVersionKind(gvk)

		if err := r.List(ctx, list, client.InNamespace(ChartNamespace), oldVersionLabel); err != nil {
			if !k8serrors.IsNotFound(err) && !k8serrors.IsMethodNotSupported(err) && !meta.IsNoMatchError(err) {
				return err
			}
		}

		for _, item := range list.Items {
			if err := r.Delete(ctx, &item); err != nil {
				return err
			} else {
				logger.Info(fmt.Sprintf("deleted resource %s of type %s with version = %s", item.GetName(), gvk.Kind, oldVersion))
			}
			numberOfDeletedItems++
		}
	}

	logger.Info(fmt.Sprintf("deleted %d orphaned chart resources after version update", numberOfDeletedItems))

	return nil
}

//+kubebuilder:rbac:groups="*",resources="*",verbs="*"

func (r *BtpOperatorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.workqueueSize += 1
	defer func() { r.workqueueSize -= 1 }()
	logger := log.FromContext(ctx)

	cr := &v1alpha1.BtpOperator{}
	if err := r.Get(ctx, req.NamespacedName, cr); err != nil {
		if k8serrors.IsNotFound(err) {
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
		return ctrl.Result{}, r.UpdateBtpOperatorStatus(ctx, cr, types.StateDeleting, HardDeleting, "BtpOperator is to be deleted")
	}

	if !r.updateCheckDone && (cr.Status.State == types.StateReady || cr.Status.State == types.StateError) {
		return ctrl.Result{}, r.UpdateBtpOperatorStatus(ctx, cr, types.StateProcessing, UpdateCheck, "Checking for updates")
	}

	switch cr.Status.State {
	case "":
		return ctrl.Result{}, r.HandleInitialState(ctx, cr)
	case types.StateProcessing:
		return ctrl.Result{RequeueAfter: ProcessingStateRequeueInterval}, r.HandleProcessingState(ctx, cr)
	case types.StateError:
		return ctrl.Result{}, r.HandleErrorState(ctx, cr)
	case types.StateDeleting:
		return ctrl.Result{}, r.HandleDeletingState(ctx, cr)
	case types.StateReady:
		return ctrl.Result{RequeueAfter: ReadyStateRequeueInterval}, r.HandleReadyState(ctx, cr)
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
	return r.UpdateBtpOperatorStatus(ctx, cr, types.StateError, OlderCRExists, fmt.Sprintf("'%s' BtpOperator CR in '%s' namespace reconciles the operand",
		oldestCr.GetName(), oldestCr.GetNamespace()))
}

func (r *BtpOperatorReconciler) UpdateBtpOperatorStatus(ctx context.Context, cr *v1alpha1.BtpOperator, newState types.State, reason Reason, message string) error {
	cr.Status.WithState(newState)
	newCondition := ConditionFromExistingReason(reason, message)
	if newCondition != nil {
		SetStatusCondition(&cr.Status.Conditions, *newCondition)
	}
	return r.Status().Update(ctx, cr)
}

func (r *BtpOperatorReconciler) HandleInitialState(ctx context.Context, cr *v1alpha1.BtpOperator) error {
	logger := log.FromContext(ctx)
	logger.Info("Handling Initial state")
	return r.UpdateBtpOperatorStatus(ctx, cr, types.StateProcessing, Initialized, "Initialized")
}

func (r *BtpOperatorReconciler) HandleProcessingState(ctx context.Context, cr *v1alpha1.BtpOperator) error {
	logger := log.FromContext(ctx)
	logger.Info("Handling Processing state")

	switch {
	case cr.IsReasonStringEqual(string(UpdateCheck)):
		logger.Info("performing update check")

		configMap, err := r.getBtpManagerConfigMap(ctx)
		if err != nil {
			return r.UpdateBtpOperatorStatus(ctx, cr, types.StateError, GettingConfigMapFailed, "Getting config map failed")
		}

		versionChanged, err := r.storeChartDetails(ctx, configMap)
		if err != nil {
			return r.UpdateBtpOperatorStatus(ctx, cr, types.StateError, StoringChartDetailsFailed, "Failure of storing chart details")
		}

		if versionChanged {
			if errorWithReason := r.handleUpdate(ctx, cr, configMap); err != nil {
				return r.UpdateBtpOperatorStatus(ctx, cr, types.StateError, errorWithReason.reason, errorWithReason.message)
			}
			r.updateCheckDone = true
			return r.UpdateBtpOperatorStatus(ctx, cr, types.StateReady, UpdateDone, "Updated")
		} else {
			r.updateCheckDone = true
			return r.UpdateBtpOperatorStatus(ctx, cr, types.StateReady, UpdateCheckSucceeded, "Update not required")
		}
	default:
		logger.Info("performing provisioning")

		installInfo, errorWithReason := r.prepareInstallInfo(ctx, cr)
		if errorWithReason != nil {
			return r.UpdateBtpOperatorStatus(ctx, cr, types.StateError, errorWithReason.reason, errorWithReason.message)
		}

		logger.Info(fmt.Sprintf("calling InstallChart, with path = %s", installInfo.ChartPath))
		ready, err := manifest.InstallChart(manifest.OperationOptions{
			Logger:             logger,
			InstallInfo:        installInfo,
			ResourceTransforms: []types.ObjectTransform{r.labelTransform},
			PostRuns:           nil,
			Cache:              nil,
		})
		if err != nil {
			logger.Error(err, fmt.Sprintf("error while installing resource %s", client.ObjectKeyFromObject(cr)))
			return r.UpdateBtpOperatorStatus(ctx, cr, types.StateError, ChartInstallFailed, fmt.Sprintf("error while installing resource %s", client.ObjectKeyFromObject(cr)))
		}
		if ready {
			return r.UpdateBtpOperatorStatus(ctx, cr, types.StateReady, ReconcileSucceeded, "Reconcile succeeded")
		}
	}

	return nil
}

func (r *BtpOperatorReconciler) HandleDeletingState(ctx context.Context, cr *v1alpha1.BtpOperator) error {
	logger := log.FromContext(ctx)
	logger.Info("Handling Deleting state")

	if len(cr.GetFinalizers()) == 0 {
		logger.Info("BtpOperator CR without finalizers - nothing to do, waiting for deletion")
		return nil
	}

	if err := r.handleDeprovisioning(ctx, cr); err != nil {
		logger.Error(err, "deprovisioning failed")
		return err
	}
	logger.Info("Deprovisioning success. Removing finalizers in CR")
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
		remainingCr := item
		if err := r.UpdateBtpOperatorStatus(ctx, &remainingCr, types.StateProcessing, Processing, "After deprovisioning"); err != nil {
			logger.Error(err, "unable to set \"Processing\" state")
		}
	}

	return nil
}

func (r *BtpOperatorReconciler) getRequiredSecret(ctx context.Context) (*corev1.Secret, error) {
	secret := &corev1.Secret{}
	objKey := client.ObjectKey{Namespace: ChartNamespace, Name: SecretName}
	if err := r.Get(ctx, objKey, secret); err != nil {
		if k8serrors.IsNotFound(err) {
			return nil, fmt.Errorf("%s Secret in %s namespace not found", SecretName, ChartNamespace)
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
	cr.Labels[chartVersionKey] = r.currentVersion
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
			ChartPath:   ChartPath,
			ReleaseName: cr.GetName(),
			Flags: types.ChartFlags{
				ConfigFlags: types.Flags{
					"Namespace":       ChartNamespace,
					"CreateNamespace": true,
					"Wait":            r.WaitForChartReadiness,
					"Timeout":         ReadyTimeout,
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
						"id": string(secret.Data["cluster_id"]),
					},
				},
			},
		},
		ResourceInfo: &types.ResourceInfo{
			BaseResource: unstructuredObj,
		},
		ClusterInfo: &types.ClusterInfo{
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
		itemLabels[chartVersionKey] = baseLabels[chartVersionKey]
		item.SetLabels(itemLabels)
	}

	return nil
}

func (r *BtpOperatorReconciler) HandleErrorState(ctx context.Context, cr *v1alpha1.BtpOperator) error {
	logger := log.FromContext(ctx)
	logger.Info("Handling Error state")

	return r.UpdateBtpOperatorStatus(ctx, cr, types.StateProcessing, Updated, "Resource has been updated")
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
		Watches(
			&source.Kind{Type: &corev1.ConfigMap{}},
			handler.EnqueueRequestsFromMapFunc(r.reconcileConfig),
			builder.WithPredicates(r.watchConfigPredicates()),
		).
		Complete(r)
}

func (r *BtpOperatorReconciler) reconcileConfig(object client.Object) []reconcile.Request {
	logger := log.FromContext(nil, "name", object.GetName(), "namespace", object.GetNamespace())
	cm, ok := object.(*corev1.ConfigMap)
	if !ok {
		return []reconcile.Request{}
	}
	logger.Info("reconciling config update", "config", cm.Data)
	for k, v := range cm.Data {
		var err error
		switch k {
		case "ChartNamespace":
			ChartNamespace = v
		case "ChartPath":
			ChartPath = v
		case "SecretName":
			SecretName = v
		case "ConfigName":
			ConfigName = v
		case "DeploymentName":
			DeploymentName = v
		case "ProcessingStateRequeueInterval":
			ProcessingStateRequeueInterval, err = time.ParseDuration(v)
		case "ReadyStateRequeueInterval":
			ReadyStateRequeueInterval, err = time.ParseDuration(v)
		case "ReadyTimeout":
			ReadyTimeout, err = time.ParseDuration(v)
		case "HardDeleteCheckInterval":
			HardDeleteCheckInterval, err = time.ParseDuration(v)
		case "HardDeleteTimeout":
			HardDeleteTimeout, err = time.ParseDuration(v)
		default:
			logger.Info("unknown config update key", k, v)
		}
		if err != nil {
			logger.Info("failed to parse config update", k, err)
		}
	}

	return r.enqueueOldestBtpOperator()
}

func (r *BtpOperatorReconciler) enqueueOldestBtpOperator() []reconcile.Request {
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

func (r *BtpOperatorReconciler) reconcileRequestForOldestBtpOperator(secret client.Object) []reconcile.Request {
	return r.enqueueOldestBtpOperator()
}

func (r *BtpOperatorReconciler) watchConfigPredicates() predicate.Funcs {
	nameMatches := func(o client.Object) bool { return o.GetName() == ConfigName && o.GetNamespace() == ChartNamespace }
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool { return nameMatches(e.Object) },
		DeleteFunc: func(e event.DeleteEvent) bool { return nameMatches(e.Object) },
		UpdateFunc: func(e event.UpdateEvent) bool { return nameMatches(e.ObjectNew) },
	}
}

func (r *BtpOperatorReconciler) watchSecretPredicates() predicate.Funcs {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			secret, ok := e.Object.(*corev1.Secret)
			if !ok {
				return false
			}
			return secret.Name == SecretName && secret.Namespace == ChartNamespace
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			secret, ok := e.Object.(*corev1.Secret)
			if !ok {
				return false
			}
			return secret.Name == SecretName && secret.Namespace == ChartNamespace
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			oldSecret, ok := e.ObjectOld.(*corev1.Secret)
			if !ok {
				return false
			}
			if oldSecret.Name == SecretName && oldSecret.Namespace == ChartNamespace {
				return true
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

func (r *BtpOperatorReconciler) handleDeprovisioning(ctx context.Context, cr *v1alpha1.BtpOperator) error {
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
				if updateStatusErr := r.UpdateBtpOperatorStatus(ctx, cr, types.StateError, ResourceRemovalFailed, "Unable to remove installed resources"); updateStatusErr != nil {
					logger.Error(updateStatusErr, "failed to update status")
					return updateStatusErr
				}
				return err
			}
		} else {
			logger.Info("Service Instances and Service Bindings hard delete failed")
			if err := r.UpdateBtpOperatorStatus(ctx, cr, types.StateDeleting, SoftDeleting, "Being soft deleted"); err != nil {
				logger.Error(err, "failed to update status")
				return err
			}
			if err := r.handleSoftDelete(ctx, namespaces); err != nil {
				logger.Error(err, "failed to soft delete")
				return err
			}
		}
	case <-time.After(HardDeleteTimeout):
		logger.Info("hard delete timeout reached", "duration", HardDeleteTimeout)
		timeoutChannel <- true
		if err := r.UpdateBtpOperatorStatus(ctx, cr, types.StateDeleting, SoftDeleting, "Being soft deleted"); err != nil {
			logger.Error(err, "failed to update status")
			return err
		}
		if err := r.handleSoftDelete(ctx, namespaces); err != nil {
			logger.Error(err, "failed to soft delete")
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
			if !errors.Is(err, context.DeadlineExceeded) {
				errs = append(errs, err)
			}
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
			if !errors.Is(err, context.DeadlineExceeded) {
				errs = append(errs, err)
			}
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

		time.Sleep(HardDeleteCheckInterval)
	}
}

func (r *BtpOperatorReconciler) hardDelete(ctx context.Context, gvk schema.GroupVersionKind, namespaces *corev1.NamespaceList) error {
	object := &unstructured.Unstructured{}
	object.SetGroupVersionKind(gvk)
	deleteCtx, cancel := context.WithTimeout(ctx, HardDeleteTimeout/2)
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
		if k8serrors.IsNotFound(err) {
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
			if !k8serrors.IsNotFound(err) {
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

	logger.Info("Deleting module deployment and webhooks")
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
		logger.Info("Removing finalizers in Service Bindings and deleting connected Secrets")
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
		logger.Info("Removing finalizers in Service Instances")
		if err := r.softDelete(ctx, instanceGvk); err != nil {
			logger.Error(err, "while deleting Service Instances")
			return err
		}
		if err := r.ensureResourcesDontExist(ctx, instanceGvk); err != nil {
			logger.Error(err, "Service Instances still exist")
			return err
		}
	}

	logger.Info("Deleting chart resources")
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
		if !k8serrors.IsNotFound(err) {
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
		if item.GetDeletionTimestamp().IsZero() {
			if err := r.Delete(ctx, &item); err != nil {
				return err
			}
		}
		item.SetFinalizers([]string{})
		if err := r.Update(ctx, &item); err != nil {
			return err
		}

		if isBinding {
			secret := &corev1.Secret{}
			secret.Name = item.GetName()
			secret.Namespace = item.GetNamespace()
			if err := r.Delete(ctx, secret); err != nil && !k8serrors.IsNotFound(err) {
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
	if err := r.Get(ctx, client.ObjectKey{Name: DeploymentName, Namespace: ChartNamespace}, deployment); err != nil {
		if !k8serrors.IsNotFound(err) {
			return err
		}
	} else {
		if err := r.Delete(ctx, deployment); client.IgnoreNotFound(err) != nil {
			return err
		}
	}

	mutatingWebhook := &admissionregistrationv1.MutatingWebhookConfiguration{}
	if err := r.Get(ctx, client.ObjectKey{Name: mutatingWebhookName, Namespace: ChartNamespace}, mutatingWebhook); err != nil {
		if !k8serrors.IsNotFound(err) {
			return err
		}
	} else {
		if err := r.Delete(ctx, mutatingWebhook); client.IgnoreNotFound(err) != nil {
			return err
		}
	}

	validatingWebhook := &admissionregistrationv1.ValidatingWebhookConfiguration{}
	if err := r.Get(ctx, client.ObjectKey{Name: validatingWebhookName, Namespace: ChartNamespace}, validatingWebhook); err != nil {
		if !k8serrors.IsNotFound(err) {
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
	gvks, err := ymlutils.GatherChartGvks(ChartPath)
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
		if err := r.DeleteAllOf(ctx, obj, client.InNamespace(ChartNamespace), labelFilter); err != nil {
			if !k8serrors.IsNotFound(err) && !k8serrors.IsMethodNotSupported(err) && !meta.IsNoMatchError(err) {
				return err
			}
		}
	}
	return nil
}

func (r *BtpOperatorReconciler) prepareInstallInfo(ctx context.Context, cr *v1alpha1.BtpOperator) (*types.InstallInfo, *ErrorWithReason) {
	logger := log.FromContext(ctx)

	secret, err := r.getRequiredSecret(ctx)
	if err != nil {
		logger.Error(err, "while getting the required Secret")
		return nil, NewErrorWithReason(MissingSecret, "Secret resource not found")
	}

	if err = r.verifySecret(secret); err != nil {
		logger.Error(err, "while verifying the required Secret")
		return nil, NewErrorWithReason(InvalidSecret, "Secret validation failed")
	}

	r.addTempLabelsToCr(cr)

	installInfo, err := r.getInstallInfo(ctx, cr, secret)
	if err != nil {
		logger.Error(err, "Error while preparing InstallInfo")
		return nil, NewErrorWithReason(PreparingInstallInfoFailed, "Error while preparing InstallInfo")
	}
	if installInfo.ChartPath == "" {
		return nil, NewErrorWithReason(ChartPathEmpty, "No chart path available for processing")
	}

	return &installInfo, nil
}

func (r *BtpOperatorReconciler) chartConsistencyCheck(ctx context.Context, cr *v1alpha1.BtpOperator) *ErrorWithReason {
	logger := log.FromContext(ctx)
	logger.Info("chart consistency check")

	installInfo, errorWithReason := r.prepareInstallInfo(ctx, cr)
	if errorWithReason != nil {
		return errorWithReason
	}

	ready, err := manifest.ConsistencyCheck(manifest.OperationOptions{
		Logger:             logger,
		InstallInfo:        installInfo,
		ResourceTransforms: []types.ObjectTransform{r.labelTransform},
		PostRuns:           nil,
		Cache:              nil,
	})
	if err != nil {
		logger.Error(err, "while doing ConsistencyCheck")
		return NewErrorWithReason(ConsistencyCheckFailed, fmt.Sprintf("Checking consistency of resource %s failed", client.ObjectKeyFromObject(cr)))
	} else if !ready {
		return NewErrorWithReason(InconsistentChart, fmt.Sprintf("Chart is inconsistent. Reconciliation initialized %s", client.ObjectKeyFromObject(cr)))
	}

	return nil
}

func (r *BtpOperatorReconciler) HandleReadyState(ctx context.Context, cr *v1alpha1.BtpOperator) error {
	logger := log.FromContext(ctx)
	logger.Info("Handling Ready state")

	errorWithReason := r.chartConsistencyCheck(ctx, cr)
	if errorWithReason != nil {
		return r.UpdateBtpOperatorStatus(ctx, cr, types.StateError, errorWithReason.reason, errorWithReason.message)
	}

	return nil
}
