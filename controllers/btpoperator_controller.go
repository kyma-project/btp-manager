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
	"crypto/x509"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"github.com/kyma-project/btp-manager/api/v1alpha1"
	"github.com/kyma-project/btp-manager/internal/certs"
	"github.com/kyma-project/btp-manager/internal/conditions"
	"github.com/kyma-project/btp-manager/internal/manifest"
	"github.com/kyma-project/btp-manager/internal/metrics"
	"github.com/kyma-project/btp-manager/internal/ymlutils"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
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

// Configuration options that can be overwritten either by CLI parameter or ConfigMap
var (
	ChartNamespace                 = "kyma-system"
	SecretName                     = "sap-btp-manager"
	ConfigName                     = "sap-btp-manager"
	DeploymentName                 = "sap-btp-operator-controller-manager"
	ProcessingStateRequeueInterval = time.Minute * 5
	ReadyStateRequeueInterval      = time.Minute * 15
	ReadyTimeout                   = time.Minute * 1
	ReadyCheckInterval             = time.Second * 2
	HardDeleteTimeout              = time.Minute * 20
	HardDeleteCheckInterval        = time.Second * 10
	DeleteRequestTimeout           = time.Minute * 5
	StatusUpdateTimeout            = time.Second * 10
	StatusUpdateCheckInterval      = time.Millisecond * 500
	ChartPath                      = "./module-chart/chart"
	ResourcesPath                  = "./module-resources"
)

const (
	chartVersionKey                    = "chart-version"
	secretKind                         = "Secret"
	configMapKind                      = "ConfigMap"
	deploymentKind                     = "Deployment"
	deploymentAvailableConditionType   = "Available"
	deploymentProgressingConditionType = "Progressing"
	operatorName                       = "btp-manager"
	deletionFinalizer                  = "operator.kyma-project.io/btp-manager"
	managedByLabelKey                  = "app.kubernetes.io/managed-by"
	btpServiceOperatorConfigMap        = "sap-btp-operator-config"
	btpServiceOperatorSecret           = "sap-btp-service-operator"
	mutatingWebhookName                = "sap-btp-operator-mutating-webhook-configuration"
	validatingWebhookName              = "sap-btp-operator-validating-webhook-configuration"
	forceDeleteLabelKey                = "force-delete"
)

// debug, test1
const (
	btpOperatorGroup           = "services.cloud.sap.com"
	btpOperatorApiVer          = "v1"
	btpOperatorServiceInstance = "ServiceInstance"
	btpOperatorServiceBinding  = "ServiceBinding"
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
	managedByLabelFilter = client.MatchingLabels{managedByLabelKey: operatorName}
)

var (
	CaSecret                       = "ca-server-cert"
	WebhookSecret                  = "webhook-server-cert"
	CaCertificateExpiration        = time.Hour * 87600 // 10 years
	WebhookCertificateExpiration   = time.Hour * 8760  // 1 year
	ExpirationBoundary             = time.Hour * -168  // 1 week
	CaSecretDataPrefix             = "ca"
	WebhookSecretDataPrefix        = "tls"
	CertificatePostfix             = "crt"
	RsaKeyPostfix                  = "key"
	MutatingWebhookConfiguration   = "MutatingWebhookConfiguration"
	ValidatingWebhookConfiguration = "ValidatingWebhookConfiguration"
)

type InstanceBindingSerivce interface {
	DisableSISBController()
	EnableSISBController()
}

// BtpOperatorReconciler reconciles a BtpOperator object
type BtpOperatorReconciler struct {
	client.Client
	*rest.Config
	Scheme                 *runtime.Scheme
	manifestHandler        *manifest.Handler
	workqueueSize          int
	metrics                *metrics.Metrics
	instanceBindingService InstanceBindingSerivce
}

func NewBtpOperatorReconciler(client client.Client, scheme *runtime.Scheme, instanceBindingSerivice InstanceBindingSerivce, metrics *metrics.Metrics) *BtpOperatorReconciler {
	return &BtpOperatorReconciler{
		Client:                 client,
		Scheme:                 scheme,
		manifestHandler:        &manifest.Handler{Scheme: scheme},
		instanceBindingService: instanceBindingSerivice,
		metrics:                metrics,
	}
}

// RBAC neccessary for the operator itself
//+kubebuilder:rbac:groups="operator.kyma-project.io",resources="btpoperators",verbs="*"
//+kubebuilder:rbac:groups="operator.kyma-project.io",resources="btpoperators/status",verbs="*"
//+kubebuilder:rbac:groups="",resources="namespaces",verbs=get;list;watch
//+kubebuilder:rbac:groups="services.cloud.sap.com",resources=serviceinstances;servicebindings,verbs="*"

// Autogenerated RBAC from the btp-operator chart
//+kubebuilder:rbac:groups="",resources="configmaps",verbs="*"
//+kubebuilder:rbac:groups="",resources="secrets",verbs="*"
//+kubebuilder:rbac:groups="",resources="serviceaccounts",verbs="*"
//+kubebuilder:rbac:groups="",resources="services",verbs="*"
//+kubebuilder:rbac:groups="admissionregistration.k8s.io",resources="mutatingwebhookconfigurations",verbs="*"
//+kubebuilder:rbac:groups="admissionregistration.k8s.io",resources="validatingwebhookconfigurations",verbs="*"
//+kubebuilder:rbac:groups="apiextensions.k8s.io",resources="customresourcedefinitions",verbs="*"
//+kubebuilder:rbac:groups="apps",resources="deployments",verbs="*"
//+kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources="clusterrolebindings",verbs="*"
//+kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources="clusterroles",verbs="*"
//+kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources="rolebindings",verbs="*"
//+kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources="roles",verbs="*"

func (r *BtpOperatorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.workqueueSize += 1
	defer func() { r.workqueueSize -= 1 }()

	logger := log.FromContext(ctx)

	reconcileCr := &v1alpha1.BtpOperator{}
	if err := r.Get(ctx, req.NamespacedName, reconcileCr); err != nil {
		if k8serrors.IsNotFound(err) {
			logger.Info(fmt.Sprintf("%s BtpOperator CR not found. Ignoring it since object has been deleted.", req.Name))
			existingBtpOperators := &v1alpha1.BtpOperatorList{}
			if err := r.List(ctx, existingBtpOperators); err != nil {
				logger.Error(err, "unable to get existing BtpOperator CRs")
				return ctrl.Result{}, nil
			}
			if len(existingBtpOperators.Items) > 0 {
				return r.setNewLeader(ctx, existingBtpOperators)
			}

			return ctrl.Result{}, nil
		}
		logger.Error(err, "unable to get BtpOperator CR")
		return ctrl.Result{}, err
	}

	existingBtpOperators := &v1alpha1.BtpOperatorList{}
	if err := r.List(ctx, existingBtpOperators); err != nil {
		logger.Error(err, "unable to get existing BtpOperator CRs")
		return ctrl.Result{}, err
	}

	if len(existingBtpOperators.Items) > 1 {
		oldestCr := r.getOldestCR(existingBtpOperators)
		if reconcileCr.GetUID() == oldestCr.GetUID() {
			reconcileCr.Status.Conditions = nil
		} else {
			return ctrl.Result{}, r.HandleRedundantCR(ctx, oldestCr, reconcileCr)
		}
	}

	if ctrlutil.AddFinalizer(reconcileCr, deletionFinalizer) {
		return ctrl.Result{}, r.Update(ctx, reconcileCr)
	}

	if !reconcileCr.ObjectMeta.DeletionTimestamp.IsZero() && reconcileCr.Status.State != v1alpha1.StateDeleting {
		return ctrl.Result{}, r.UpdateBtpOperatorStatus(ctx, reconcileCr, v1alpha1.StateDeleting, conditions.HardDeleting, "BtpOperator is to be deleted")
	}

	switch reconcileCr.Status.State {
	case "":
		return ctrl.Result{}, r.HandleInitialState(ctx, reconcileCr)
	case v1alpha1.StateProcessing:
		return ctrl.Result{RequeueAfter: ProcessingStateRequeueInterval}, r.HandleProcessingState(ctx, reconcileCr)
	case v1alpha1.StateError, v1alpha1.StateWarning:
		return ctrl.Result{}, r.HandleErrorState(ctx, reconcileCr)
	case v1alpha1.StateDeleting:
		err := r.HandleDeletingState(ctx, reconcileCr)
		if reconcileCr.IsReasonStringEqual(string(conditions.ServiceInstancesAndBindingsNotCleaned)) {
			return ctrl.Result{RequeueAfter: ReadyStateRequeueInterval}, err
		}
		return ctrl.Result{}, err
	case v1alpha1.StateReady:
		return ctrl.Result{RequeueAfter: ReadyStateRequeueInterval}, r.HandleReadyState(ctx, reconcileCr)
	}

	return ctrl.Result{}, nil
}

func (r *BtpOperatorReconciler) setNewLeader(ctx context.Context, existingBtpOperators *v1alpha1.BtpOperatorList) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info(fmt.Sprintf("Found %d existing BtpOperators", len(existingBtpOperators.Items)))
	oldestCr := r.getOldestCR(existingBtpOperators)
	if oldestCr.GetStatus().State == v1alpha1.StateError && oldestCr.IsReasonStringEqual(string(conditions.OlderCRExists)) {
		if err := r.UpdateBtpOperatorStatus(ctx, oldestCr, v1alpha1.StateProcessing, conditions.Processing,
			fmt.Sprintf("%s is the new leader", oldestCr.GetName())); err != nil {
			logger.Error(err, fmt.Sprintf("unable to set %s BtpOperator CR as the new leader", oldestCr.GetName()))
			return ctrl.Result{}, err
		}
	}
	logger.Info(fmt.Sprintf("%s BtpOperator is the new leader", oldestCr.GetName()))
	for _, cr := range existingBtpOperators.Items {
		if cr.GetUID() == oldestCr.GetUID() {
			continue
		}
		redundantCR := cr.DeepCopy()
		if err := r.HandleRedundantCR(ctx, oldestCr, redundantCR); err != nil {
			logger.Info(fmt.Sprintf("unable to update %s BtpOperator CR Status", redundantCR.GetName()))
		}
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
	return r.UpdateBtpOperatorStatus(ctx, cr, v1alpha1.StateError, conditions.OlderCRExists, fmt.Sprintf("'%s' BtpOperator CR in '%s' namespace reconciles the module",
		oldestCr.GetName(), oldestCr.GetNamespace()))
}

func (r *BtpOperatorReconciler) UpdateBtpOperatorStatus(ctx context.Context, cr *v1alpha1.BtpOperator, newState v1alpha1.State, reason conditions.Reason, message string) error {
	logger := log.FromContext(ctx)
	timeout := time.Now().Add(StatusUpdateTimeout)

	var err error
	for now := time.Now(); now.Before(timeout); now = time.Now() {
		if err = r.Get(ctx, client.ObjectKeyFromObject(cr), cr); err != nil {
			if k8serrors.IsNotFound(err) {
				return nil
			}
			logger.Error(err, fmt.Sprintf("cannot get the BtpOperator to update the status. Retrying in %s...", StatusUpdateCheckInterval.String()))
			time.Sleep(StatusUpdateCheckInterval)
			continue
		}
		if cr.Status.State == newState && cr.IsMsgForGivenReasonEqual(string(reason), message) {
			return nil
		}
		cr.Status.WithState(newState)
		newCondition := conditions.ConditionFromExistingReason(reason, message)
		if newCondition != nil {
			conditions.SetStatusCondition(&cr.Status.Conditions, *newCondition)
		}
		if err = r.Status().Update(ctx, cr); err != nil {
			logger.Error(err, fmt.Sprintf("cannot update the status of the BtpOperator. Retrying in %s...", StatusUpdateCheckInterval.String()))
			time.Sleep(StatusUpdateCheckInterval)
			continue
		}
		time.Sleep(StatusUpdateCheckInterval)
	}
	logger.Error(err, fmt.Sprintf("timed out while waiting %s for the BtpOperator status change.", StatusUpdateTimeout.String()))

	return err
}

func (r *BtpOperatorReconciler) HandleInitialState(ctx context.Context, cr *v1alpha1.BtpOperator) error {
	logger := log.FromContext(ctx)
	logger.Info("Handling Initial state")
	return r.UpdateBtpOperatorStatus(ctx, cr, v1alpha1.StateProcessing, conditions.Initialized, "Initialized")
}

func (r *BtpOperatorReconciler) HandleProcessingState(ctx context.Context, cr *v1alpha1.BtpOperator) error {
	logger := log.FromContext(ctx)
	logger.Info("Handling Processing state")

	secret, errWithReason := r.getAndVerifyRequiredSecret(ctx)
	if errWithReason != nil {
		return r.handleMissingSecret(ctx, cr, logger, errWithReason)
	}

	if err := r.deleteOutdatedResources(ctx); err != nil {
		return r.UpdateBtpOperatorStatus(ctx, cr, v1alpha1.StateError, conditions.ProvisioningFailed, err.Error())
	}

	if err := r.reconcileResources(ctx, secret); err != nil {
		return r.UpdateBtpOperatorStatus(ctx, cr, v1alpha1.StateError, conditions.ProvisioningFailed, err.Error())
	}

	r.instanceBindingService.EnableSISBController()

	logger.Info("provisioning succeeded")
	return r.UpdateBtpOperatorStatus(ctx, cr, v1alpha1.StateReady, conditions.ReconcileSucceeded, "Module provisioning succeeded")
}

func (r *BtpOperatorReconciler) handleMissingSecret(ctx context.Context, cr *v1alpha1.BtpOperator, logger logr.Logger, errWithReason *ErrorWithReason) error {
	logger.Info("secret verification failed: " + errWithReason.Error())
	return r.UpdateBtpOperatorStatus(ctx, cr, v1alpha1.StateWarning, errWithReason.reason, errWithReason.message)
}

func (r *BtpOperatorReconciler) getAndVerifyRequiredSecret(ctx context.Context) (*corev1.Secret, *ErrorWithReason) {
	logger := log.FromContext(ctx)

	logger.Info("getting the required Secret")
	secret, err := r.getRequiredSecret(ctx)
	if err != nil {
		logger.Error(err, "while getting the required Secret")
		return nil, NewErrorWithReason(conditions.MissingSecret, "Secret resource not found")
	}

	logger.Info("verifying the required Secret")
	if err = r.verifySecret(secret); err != nil {
		logger.Error(err, "while verifying the required Secret")
		return nil, NewErrorWithReason(conditions.InvalidSecret, "Secret validation failed")
	}
	return secret, nil
}

func (r *BtpOperatorReconciler) getRequiredSecret(ctx context.Context) (*corev1.Secret, error) {
	secret := &corev1.Secret{}
	objKey := client.ObjectKey{Namespace: ChartNamespace, Name: SecretName}
	if err := r.Get(ctx, objKey, secret); err != nil {
		if k8serrors.IsNotFound(err) {
			return nil, fmt.Errorf("%s Secret in %s namespace not found", SecretName, ChartNamespace)
		}
		return nil, fmt.Errorf("unable to get Secret: %w", err)
	}

	return secret, nil
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

func (r *BtpOperatorReconciler) deleteOutdatedResources(ctx context.Context) error {
	logger := log.FromContext(ctx)

	logger.Info("getting outdated module resources to delete")
	resourcesToDelete, err := r.createUnstructuredObjectsFromManifestsDir(r.getResourcesToDeletePath())
	if err != nil {
		logger.Error(err, "while getting objects to delete from manifests")
		return fmt.Errorf("Failed to create deletable objects from manifests: %w", err)
	}
	logger.Info(fmt.Sprintf("got %d outdated module resources to delete", len(resourcesToDelete)))

	err = r.deleteResources(ctx, resourcesToDelete)
	if err != nil {
		logger.Error(err, "while deleting outdated resources")
		return fmt.Errorf("Failed to delete outdated resources: %w", err)
	}

	return nil
}

func (r *BtpOperatorReconciler) createUnstructuredObjectsFromManifestsDir(manifestsDir string) ([]*unstructured.Unstructured, error) {
	objs, err := r.manifestHandler.CollectObjectsFromDir(manifestsDir)
	if err != nil {
		return nil, err
	}
	us, err := r.manifestHandler.ObjectsToUnstructured(objs)
	if err != nil {
		return nil, err
	}

	return us, nil
}

func (r *BtpOperatorReconciler) getResourcesToDeletePath() string {
	return fmt.Sprintf("%s%cdelete", ResourcesPath, os.PathSeparator)
}

func (r *BtpOperatorReconciler) deleteResources(ctx context.Context, us []*unstructured.Unstructured) error {
	logger := log.FromContext(ctx)

	var errs []string
	for _, u := range us {
		if err := r.Delete(ctx, u); err != nil {
			if k8serrors.IsNotFound(err) {
				continue
			} else {
				errs = append(errs, fmt.Sprintf("failed to delete %s %s: %s", u.GetName(), u.GetKind(), err))
			}
		}
		logger.Info("deleted resource", "name", u.GetName(), "kind", u.GetKind())
	}

	if errs != nil {
		return errors.New(strings.Join(errs, ", "))
	}

	return nil
}

func (r *BtpOperatorReconciler) reconcileResources(ctx context.Context, s *corev1.Secret) error {
	logger := log.FromContext(ctx)

	logger.Info("getting module resources to apply")
	resourcesToApply, err := r.createUnstructuredObjectsFromManifestsDir(r.getResourcesToApplyPath())
	if err != nil {
		logger.Error(err, "while creating applicable objects from manifests")
		return fmt.Errorf("failed to create applicable objects from manifests: %w", err)
	}
	logger.Info(fmt.Sprintf("got %d module resources to apply based on %s directory", len(resourcesToApply), r.getResourcesToApplyPath()))

	logger.Info("preparing module resources to apply")
	if err = r.prepareModuleResourcesFromManifests(ctx, resourcesToApply, s); err != nil {
		logger.Error(err, "while preparing objects to apply")
		return fmt.Errorf("failed to prepare objects to apply: %w", err)
	}

	if err := r.prepareCertificatesReconciliationData(ctx, &resourcesToApply); err != nil {
		return fmt.Errorf("failed to reconcile webhook certs: %w", err)
	}

	r.deleteCreationTimestamp(resourcesToApply...)

	logger.Info(fmt.Sprintf("applying module resources for %d resources", len(resourcesToApply)))
	if err = r.applyOrUpdateResources(ctx, resourcesToApply); err != nil {
		logger.Error(err, "while applying module resources")
		return fmt.Errorf("failed to apply module resources: %w", err)
	}

	logger.Info("waiting for module resources readiness")
	if err = r.waitForResourcesReadiness(ctx, resourcesToApply); err != nil {
		logger.Error(err, "while waiting for module resources readiness")
		return fmt.Errorf("timed out while waiting for resources readiness: %w", err)
	}

	return nil
}

func (r *BtpOperatorReconciler) getResourcesToApplyPath() string {
	return fmt.Sprintf("%s%capply", ResourcesPath, os.PathSeparator)
}

func (r *BtpOperatorReconciler) prepareModuleResourcesFromManifests(ctx context.Context, resourcesToApply []*unstructured.Unstructured, s *corev1.Secret) error {
	logger := log.FromContext(ctx)

	var configMapIndex, secretIndex int
	for i, u := range resourcesToApply {
		if u.GetName() == btpServiceOperatorConfigMap && u.GetKind() == configMapKind {
			configMapIndex = i
		}
		if u.GetName() == btpServiceOperatorSecret && u.GetKind() == secretKind {
			secretIndex = i
		}
	}

	chartVer, err := ymlutils.ExtractStringValueFromYamlForGivenKey(fmt.Sprintf("%s/Chart.yaml", ChartPath), "version")
	if err != nil {
		logger.Error(err, "while getting module chart version")
		return fmt.Errorf("failed to get module chart version: %w", err)
	}

	r.addLabels(chartVer, resourcesToApply...)
	r.setNamespace(resourcesToApply...)

	if err := r.setConfigMapValues(s, (resourcesToApply)[configMapIndex]); err != nil {
		logger.Error(err, "while setting ConfigMap values")
		return fmt.Errorf("failed to set ConfigMap values: %w", err)
	}
	if err := r.setSecretValues(s, (resourcesToApply)[secretIndex]); err != nil {
		logger.Error(err, "while setting Secret values")
		return fmt.Errorf("failed to set Secret values: %w", err)
	}

	return nil
}

func (r *BtpOperatorReconciler) addLabels(chartVer string, us ...*unstructured.Unstructured) {

	for _, u := range us {
		labels := u.GetLabels()
		if len(labels) == 0 {
			labels = make(map[string]string)
		}
		labels[managedByLabelKey] = operatorName
		labels[chartVersionKey] = chartVer
		u.SetLabels(labels)
	}
}

func (r *BtpOperatorReconciler) setNamespace(us ...*unstructured.Unstructured) {
	for _, u := range us {
		u.SetNamespace(ChartNamespace)
	}
}

func (r *BtpOperatorReconciler) deleteCreationTimestamp(us ...*unstructured.Unstructured) {
	for _, u := range us {
		unstructured.RemoveNestedField(u.Object, "metadata", "creationTimestamp")
	}
}

func (r *BtpOperatorReconciler) setConfigMapValues(secret *corev1.Secret, u *unstructured.Unstructured) error {
	return unstructured.SetNestedField(u.Object, string(secret.Data["cluster_id"]), "data", "CLUSTER_ID")
}

func (r *BtpOperatorReconciler) setSecretValues(secret *corev1.Secret, u *unstructured.Unstructured) error {
	for k := range secret.Data {
		if err := unstructured.SetNestedField(u.Object, base64.StdEncoding.EncodeToString(secret.Data[k]), "data", k); err != nil {
			return err
		}
	}
	return nil
}

func (r *BtpOperatorReconciler) applyOrUpdateResources(ctx context.Context, us []*unstructured.Unstructured) error {
	logger := log.FromContext(ctx)
	for _, u := range us {
		preExistingResource := &unstructured.Unstructured{}
		preExistingResource.SetGroupVersionKind(u.GroupVersionKind())
		if err := r.Get(ctx, client.ObjectKey{Name: u.GetName(), Namespace: u.GetNamespace()}, preExistingResource); err != nil {
			if !k8serrors.IsNotFound(err) {
				return fmt.Errorf("while trying to get %s %s: %w", u.GetName(), u.GetKind(), err)
			}
			logger.Info(fmt.Sprintf("applying %s - %s", u.GetKind(), u.GetName()))
			if err := r.Patch(ctx, u, client.Apply, client.ForceOwnership, client.FieldOwner(operatorName)); err != nil {
				return fmt.Errorf("while applying %s %s: %w", u.GetName(), u.GetKind(), err)
			}
		} else {
			logger.Info(fmt.Sprintf("updating %s - %s", u.GetKind(), u.GetName()))
			u.SetResourceVersion(preExistingResource.GetResourceVersion())
			if err := r.Update(ctx, u, client.FieldOwner(operatorName)); err != nil {
				return fmt.Errorf("while updating %s %s: %w", u.GetName(), u.GetKind(), err)
			}
		}
	}
	return nil
}

func (r *BtpOperatorReconciler) waitForResourcesReadiness(ctx context.Context, us []*unstructured.Unstructured) error {
	numOfResources := len(us)
	resourcesReadinessInformer := make(chan bool, numOfResources)
	allReadyInformer := make(chan bool, 1)
	for _, u := range us {
		go r.checkResourceReadiness(ctx, u, resourcesReadinessInformer)
	}
	go func(c chan bool) {
		timeout := time.After(ReadyTimeout)
		for i := 0; i < numOfResources; i++ {
			select {
			case <-resourcesReadinessInformer:
				continue
			case <-timeout:
				return
			}
		}
		allReadyInformer <- true
	}(resourcesReadinessInformer)
	select {
	case <-allReadyInformer:
		return nil
	case <-time.After(ReadyTimeout):
		return errors.New("resources readiness timeout reached")
	}
}

func (r *BtpOperatorReconciler) checkResourceReadiness(ctx context.Context, u *unstructured.Unstructured, c chan<- bool) {
	switch u.GetKind() {
	case deploymentKind:
		r.checkDeploymentReadiness(ctx, u, c)
	default:
		r.checkResourceExistence(ctx, u, c)
	}
}

func (r *BtpOperatorReconciler) checkDeploymentReadiness(ctx context.Context, u *unstructured.Unstructured, c chan<- bool) {
	logger := log.FromContext(ctx)
	ctxWithTimeout, cancel := context.WithTimeout(ctx, ReadyCheckInterval/2)
	defer cancel()

	var err error
	var availableConditionStatus, progressingConditionStatus string
	got := &appsv1.Deployment{}
	now := time.Now()
	for {
		if time.Since(now) >= ReadyTimeout {
			logger.Error(err, fmt.Sprintf("timed out while checking %s %s readiness", u.GetName(), u.GetKind()))
			return
		}
		if err = r.Get(ctxWithTimeout, client.ObjectKey{Name: u.GetName(), Namespace: u.GetNamespace()}, got); err == nil {
			for _, condition := range got.Status.Conditions {
				if string(condition.Type) == deploymentProgressingConditionType {
					progressingConditionStatus = string(condition.Status)
				} else if string(condition.Type) == deploymentAvailableConditionType {
					availableConditionStatus = string(condition.Status)
				}
			}
			if progressingConditionStatus == "True" && availableConditionStatus == "True" {
				c <- true
				return
			}
		}
		time.Sleep(ReadyCheckInterval)
	}
}

func (r *BtpOperatorReconciler) checkResourceExistence(ctx context.Context, u *unstructured.Unstructured, c chan<- bool) {
	logger := log.FromContext(ctx)
	ctxWithTimeout, cancel := context.WithTimeout(ctx, ReadyCheckInterval/2)
	defer cancel()

	var err error
	now := time.Now()
	got := &unstructured.Unstructured{}
	got.SetGroupVersionKind(u.GroupVersionKind())
	for {
		if time.Since(now) >= ReadyTimeout {
			logger.Error(err, fmt.Sprintf("timed out while checking %s %s existence", u.GetName(), u.GetKind()))
			return
		}
		if err = r.Get(ctxWithTimeout, client.ObjectKey{Name: u.GetName(), Namespace: u.GetNamespace()}, got); err == nil {
			c <- true
			return
		}
		time.Sleep(ReadyCheckInterval)
	}
}

func (r *BtpOperatorReconciler) HandleErrorState(ctx context.Context, cr *v1alpha1.BtpOperator) error {
	logger := log.FromContext(ctx)
	logger.Info("Handling Error state")

	return r.UpdateBtpOperatorStatus(ctx, cr, v1alpha1.StateProcessing, conditions.Updated, "CR has been updated")
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
	if cr.IsReasonStringEqual(string(conditions.ServiceInstancesAndBindingsNotCleaned)) {
		numberOfBindings, err := r.numberOfResources(ctx, bindingGvk)
		if err != nil {
			return err
		}
		numberOfInstances, err := r.numberOfResources(ctx, instanceGvk)
		if err != nil {
			return err
		}
		if numberOfBindings > 0 || numberOfInstances > 0 {
			logger.Info(fmt.Sprintf("%d instances, %d bindings - leaving deletion", numberOfInstances, numberOfBindings))
			return nil
		}
	}

	r.instanceBindingService.DisableSISBController()

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
		if err := r.UpdateBtpOperatorStatus(ctx, &remainingCr, v1alpha1.StateProcessing, conditions.Processing, "After deprovisioning"); err != nil {
			logger.Error(err, "unable to set \"Processing\" state")
		}
	}

	return nil
}

func (r *BtpOperatorReconciler) handleDeprovisioning(ctx context.Context, cr *v1alpha1.BtpOperator) error {
	logger := log.FromContext(ctx)

	namespaces := &corev1.NamespaceList{}
	if err := r.List(ctx, namespaces); err != nil {
		return err
	}

	if !r.IsForceDelete(cr) {
		numberOfBindings, err := r.numberOfResources(ctx, bindingGvk)
		if err != nil {
			return err
		}
		numberOfInstances, err := r.numberOfResources(ctx, instanceGvk)
		if err != nil {
			return err
		}

		if numberOfBindings > 0 || numberOfInstances > 0 {
			logger.Info(fmt.Sprintf("Existing resources (%d instances and %d bindings) block BTP Operator deletion.", numberOfInstances, numberOfBindings))
			msg := fmt.Sprintf("All service instances and bindings must be removed: %d instance(s) and %d binding(s)", numberOfInstances, numberOfBindings)
			logger.Info(msg)

			// if the reason is already set, do nothing
			if cr.IsReasonStringEqual(string(conditions.ServiceInstancesAndBindingsNotCleaned)) {
				return nil
			}

			if updateStatusErr := r.UpdateBtpOperatorStatus(ctx, cr,
				v1alpha1.StateDeleting, conditions.ServiceInstancesAndBindingsNotCleaned, msg); updateStatusErr != nil {
				return updateStatusErr
			}
			return nil
		}
	}
	if cr.IsReasonStringEqual(string(conditions.ServiceInstancesAndBindingsNotCleaned)) {
		// go to a state which starts deleting process
		if updateStatusErr := r.UpdateBtpOperatorStatus(ctx, cr,
			v1alpha1.StateDeleting, conditions.HardDeleting,
			"BtpOperator is to be deleted after cleaning service instance and binding resources"); updateStatusErr != nil {
			return updateStatusErr
		}
	}

	hardDeleteSucceededCh := make(chan bool, 1)
	hardDeleteTimeoutReachedCh := make(chan bool, 1)
	defer close(hardDeleteTimeoutReachedCh)

	go r.handleHardDelete(ctx, namespaces, hardDeleteSucceededCh, hardDeleteTimeoutReachedCh)

	select {
	case hardDeleteSucceeded := <-hardDeleteSucceededCh:
		if hardDeleteSucceeded {
			logger.Info("Service Instances and Service Bindings hard delete succeeded. Removing module resources")
			if err := r.deleteBtpOperatorResources(ctx); err != nil {
				logger.Error(err, "failed to remove module resources")
				if updateStatusErr := r.UpdateBtpOperatorStatus(ctx, cr, v1alpha1.StateError, conditions.ResourceRemovalFailed, "Unable to remove installed resources"); updateStatusErr != nil {
					logger.Error(updateStatusErr, "failed to update status")
					return updateStatusErr
				}
				return err
			}
		} else {
			logger.Info("Service Instances and Service Bindings hard delete failed")
			if err := r.UpdateBtpOperatorStatus(ctx, cr, v1alpha1.StateDeleting, conditions.SoftDeleting, "Being soft deleted"); err != nil {
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
		hardDeleteTimeoutReachedCh <- true
		if err := r.UpdateBtpOperatorStatus(ctx, cr, v1alpha1.StateDeleting, conditions.SoftDeleting, "Being soft deleted"); err != nil {
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

func (r *BtpOperatorReconciler) handleHardDelete(ctx context.Context, namespaces *corev1.NamespaceList, hardDeleteSucceededCh, hardDeleteTimeoutReachedCh chan bool) {
	logger := log.FromContext(ctx)
	logger.Info("Deprovisioning BTP Operator - hard delete")
	defer close(hardDeleteSucceededCh)

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
		hardDeleteSucceededCh <- false
		return
	}

	var sbResourcesLeft, siResourcesLeft bool
	for {
		select {
		case <-hardDeleteTimeoutReachedCh:
			return
		default:
		}

		if sbCrdExists {
			sbResourcesLeft, err = r.resourcesExist(ctx, namespaces, bindingGvk)
			if err != nil {
				logger.Error(err, "ServiceBinding leftover resources check failed")
				hardDeleteSucceededCh <- false
				return
			}
		}

		if siCrdExists {
			siResourcesLeft, err = r.resourcesExist(ctx, namespaces, instanceGvk)
			if err != nil {
				logger.Error(err, "ServiceInstance leftover resources check failed")
				hardDeleteSucceededCh <- false
				return
			}
		}

		if !sbResourcesLeft && !siResourcesLeft {
			hardDeleteSucceededCh <- true
			return
		}

		time.Sleep(HardDeleteCheckInterval)
	}
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

func (r *BtpOperatorReconciler) hardDelete(ctx context.Context, gvk schema.GroupVersionKind, namespaces *corev1.NamespaceList) error {
	object := &unstructured.Unstructured{}
	object.SetGroupVersionKind(gvk)
	deleteCtx, cancel := context.WithTimeout(ctx, DeleteRequestTimeout)
	defer cancel()

	for _, namespace := range namespaces.Items {
		if err := r.DeleteAllOf(deleteCtx, object, client.InNamespace(namespace.Name)); err != nil {
			return err
		}
	}

	return nil
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

func (r *BtpOperatorReconciler) numberOfResources(ctx context.Context, gvk schema.GroupVersionKind) (int, error) {
	exists, err := r.crdExists(ctx, gvk)
	if err != nil {
		return 0, err
	}
	if !exists {
		return 0, nil
	}

	list := &unstructured.UnstructuredList{}
	list.SetGroupVersionKind(gvk)
	err = r.List(ctx, list, client.InNamespace(corev1.NamespaceAll))
	if err != nil {
		return 0, err
	}
	return len(list.Items), nil
}

func (r *BtpOperatorReconciler) deleteBtpOperatorResources(ctx context.Context) error {
	logger := log.FromContext(ctx)

	logger.Info("getting module resources to delete")
	resourcesToDeleteFromApply, err := r.createUnstructuredObjectsFromManifestsDir(r.getResourcesToApplyPath())
	if err != nil {
		logger.Error(err, "while getting objects to delete from manifests")
		return fmt.Errorf("Failed to create deletable objects from manifests: %w", err)
	}
	logger.Info(fmt.Sprintf("got %d module resources to delete from \"apply\" dir", len(resourcesToDeleteFromApply)))

	resourcesToDeleteFromDelete, err := r.createUnstructuredObjectsFromManifestsDir(r.getResourcesToDeletePath())
	if err != nil {
		logger.Error(err, "while getting objects to delete from manifests")
		return fmt.Errorf("Failed to create deletable objects from manifests: %w", err)
	}
	logger.Info(fmt.Sprintf("got %d module resources to delete from \"delete\" dir", len(resourcesToDeleteFromDelete)))

	resourcesToDelete := make([]*unstructured.Unstructured, 0)
	resourcesToDelete = append(resourcesToDelete, resourcesToDeleteFromApply...)
	resourcesToDelete = append(resourcesToDelete, resourcesToDeleteFromDelete...)

	if err = r.deleteAllOfResourcesTypes(ctx, resourcesToDelete...); err != nil {
		logger.Error(err, "while deleting module resources")
		return fmt.Errorf("Failed to delete module resources: %w", err)
	}

	return nil
}

func (r *BtpOperatorReconciler) deleteAllOfResourcesTypes(ctx context.Context, resourcesToDelete ...*unstructured.Unstructured) error {
	logger := log.FromContext(ctx)
	deletedGvks := make(map[string]struct{}, 0)
	for _, u := range resourcesToDelete {
		if _, exists := deletedGvks[u.GroupVersionKind().String()]; exists {
			continue
		}
		logger.Info(fmt.Sprintf("deleting all of %s/%s module resources in %s namespace",
			u.GroupVersionKind().GroupVersion(), u.GetKind(), ChartNamespace))
		if err := r.DeleteAllOf(ctx, u, client.InNamespace(ChartNamespace), managedByLabelFilter); err != nil {
			if !(k8serrors.IsNotFound(err) || k8serrors.IsMethodNotSupported(err) || meta.IsNoMatchError(err)) {
				return err
			}
		}
		deletedGvks[u.GroupVersionKind().String()] = struct{}{}
	}

	return nil
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

	logger.Info("Deleting module resources")
	if err := r.deleteBtpOperatorResources(ctx); err != nil {
		logger.Error(err, "failed to delete module resources")
		return err
	}

	return nil
}

func (r *BtpOperatorReconciler) preSoftDeleteCleanup(ctx context.Context) error {
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

func (r *BtpOperatorReconciler) HandleReadyState(ctx context.Context, cr *v1alpha1.BtpOperator) error {
	logger := log.FromContext(ctx)
	logger.Info("Handling Ready state")

	secret, errWithReason := r.getAndVerifyRequiredSecret(ctx)
	if errWithReason != nil {
		return r.handleMissingSecret(ctx, cr, logger, errWithReason)
	}

	if err := r.deleteOutdatedResources(ctx); err != nil {
		return r.UpdateBtpOperatorStatus(ctx, cr, v1alpha1.StateError, conditions.ReconcileFailed, err.Error())
	}

	if err := r.reconcileResources(ctx, secret); err != nil {
		return r.UpdateBtpOperatorStatus(ctx, cr, v1alpha1.StateError, conditions.ReconcileFailed, err.Error())
	}

	logger.Info("reconciliation succeeded")
	return nil
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
		Watches(
			&source.Kind{Type: &admissionregistrationv1.MutatingWebhookConfiguration{}},
			handler.EnqueueRequestsFromMapFunc(r.reconcileRequestForOldestBtpOperator),
			builder.WithPredicates(r.watchMutatingWebhooksPredicates()),
		).
		Watches(
			&source.Kind{Type: &admissionregistrationv1.ValidatingWebhookConfiguration{}},
			handler.EnqueueRequestsFromMapFunc(r.reconcileRequestForOldestBtpOperator),
			builder.WithPredicates(r.watchValidatingWebhooksPredicates()),
		).
		Complete(r)
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
			state := newBtpOperator.GetStatus().State
			if (state == v1alpha1.StateError || state == v1alpha1.StateWarning) && newBtpOperator.ObjectMeta.DeletionTimestamp.IsZero() {
				return false
			}

			return true
		},
		GenericFunc: func(e event.GenericEvent) bool {
			return true
		},
	}
}

func (r *BtpOperatorReconciler) reconcileRequestForOldestBtpOperator(secret client.Object) []reconcile.Request {
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

func (r *BtpOperatorReconciler) watchSecretPredicates() predicate.Funcs {
	predicateIfReconcile := func(secret *corev1.Secret) bool {
		return secret.Namespace == ChartNamespace && (secret.Name == SecretName || secret.Name == CaSecret || secret.Name == WebhookSecret)
	}

	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			secret, ok := e.Object.(*corev1.Secret)
			if !ok {
				return false
			}
			return predicateIfReconcile(secret)
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			secret, ok := e.Object.(*corev1.Secret)
			if !ok {
				return false
			}
			return predicateIfReconcile(secret)
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			oldSecret, ok := e.ObjectOld.(*corev1.Secret)
			if !ok {
				return false
			}
			return predicateIfReconcile(oldSecret)
		},
	}
}

func (r *BtpOperatorReconciler) watchValidatingWebhooksPredicates() predicate.Funcs {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			obj := e.Object.(*admissionregistrationv1.ValidatingWebhookConfiguration)
			return obj.Name == validatingWebhookName
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			obj := e.Object.(*admissionregistrationv1.ValidatingWebhookConfiguration)
			return obj.Name == validatingWebhookName
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			objectOld := e.ObjectOld.(*admissionregistrationv1.ValidatingWebhookConfiguration)

			if objectOld.Name != validatingWebhookName {
				return false
			}

			existingCaBundles := make([][]byte, 0)
			for _, w := range objectOld.Webhooks {
				existingCaBundles = append(existingCaBundles, w.ClientConfig.CABundle)
			}

			objectNew := e.ObjectNew.(*admissionregistrationv1.ValidatingWebhookConfiguration)
			newCaBundles := make([][]byte, 0)
			for _, w := range objectNew.Webhooks {
				newCaBundles = append(newCaBundles, w.ClientConfig.CABundle)
			}

			if !reflect.DeepEqual(existingCaBundles, newCaBundles) {
				return true
			}

			return false
		},
	}
}

func (r *BtpOperatorReconciler) watchMutatingWebhooksPredicates() predicate.Funcs {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			obj := e.Object.(*admissionregistrationv1.MutatingWebhookConfiguration)
			return obj.Name == mutatingWebhookName
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			obj := e.Object.(*admissionregistrationv1.MutatingWebhookConfiguration)
			return obj.Name == mutatingWebhookName
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			objectOld := e.ObjectOld.(*admissionregistrationv1.MutatingWebhookConfiguration)

			if objectOld.Name != mutatingWebhookName {
				return false
			}

			existingCaBundles := make([][]byte, 0)
			for _, w := range objectOld.Webhooks {
				existingCaBundles = append(existingCaBundles, w.ClientConfig.CABundle)
			}

			objectNew := e.ObjectNew.(*admissionregistrationv1.MutatingWebhookConfiguration)
			newCaBundles := make([][]byte, 0)
			for _, w := range objectNew.Webhooks {
				newCaBundles = append(newCaBundles, w.ClientConfig.CABundle)
			}

			if !reflect.DeepEqual(existingCaBundles, newCaBundles) {
				return true
			}

			return false
		},
	}
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
		case "ResourcesPath":
			ResourcesPath = v
		case "ReadyCheckInterval":
			ReadyCheckInterval, err = time.ParseDuration(v)
		case "DeleteRequestTimeout":
			DeleteRequestTimeout, err = time.ParseDuration(v)
		case "CaCertificateExpiration":
			CaCertificateExpiration, err = time.ParseDuration(v)
		case "WebhookCertificateExpiration":
			WebhookCertificateExpiration, err = time.ParseDuration(v)
		case "ExpirationBoundary":
			ExpirationBoundary, err = time.ParseDuration(v)
		case "RsaKeyBits":
			var bits int
			bits, err = strconv.Atoi(v)
			if err == nil {
				certs.SetRsaKeyBits(bits)
			}
		default:
			logger.Info("unknown config update key", k, v)
		}
		if err != nil {
			logger.Info("failed to parse config update", k, err)
		}
	}

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

func (r *BtpOperatorReconciler) IsForceDelete(cr *v1alpha1.BtpOperator) bool {
	if _, exists := cr.Labels[forceDeleteLabelKey]; !exists {
		return false
	}
	return cr.Labels[forceDeleteLabelKey] == "true"
}

// *[]*unstructured.Unstructured is required because we extend the slice during certificates regeneration adding secrets and webhook configurations,
// so the result of the function execution is in resourcesToApply slice
func (r *BtpOperatorReconciler) prepareCertificatesReconciliationData(ctx context.Context, resourcesToApply *[]*unstructured.Unstructured) error {
	logger := log.FromContext(ctx)
	logger.Info("preparation of certificates reconciliation data started")

	certificatesRegenerationDone, err := r.ensureCertificatesExists(ctx, resourcesToApply)
	if err != nil {
		return err
	}
	if certificatesRegenerationDone {
		r.metrics.IncreaseCertsRegenerationsCounter()
		return nil
	}

	certificatesRegenerationDone, err = r.ensureSecretsDataIsSet(ctx, resourcesToApply)
	if err != nil {
		return err
	}
	if certificatesRegenerationDone {
		r.metrics.IncreaseCertsRegenerationsCounter()
		return nil
	}

	certificatesRegenerationDone, err = r.ensureCertificatesAreCorrectlyStructured(ctx, resourcesToApply)
	if err != nil {
		return err
	}
	if certificatesRegenerationDone {
		r.metrics.IncreaseCertsRegenerationsCounter()
		return nil
	}

	certificatesRegenerationDone, err = r.ensureCertificatesHaveValidExpiration(ctx, resourcesToApply)
	if err != nil {
		return err
	}
	if certificatesRegenerationDone {
		r.metrics.IncreaseCertsRegenerationsCounter()
		return nil
	}

	certificatesRegenerationDone, err = r.ensureCertificatesAreCorrectSigned(ctx, resourcesToApply)
	if err != nil {
		return err
	}
	if certificatesRegenerationDone {
		r.metrics.IncreaseCertsRegenerationsCounter()
		return nil
	}

	if err := r.prepareWebhooksConfigurationsReconciliationData(ctx, resourcesToApply, nil); err != nil {
		return err
	}

	return nil
}

func (r *BtpOperatorReconciler) ensureCertificatesExists(ctx context.Context, resourcesToApply *[]*unstructured.Unstructured) (bool, error) {
	logger := log.FromContext(ctx)
	caSecretExists, err := r.checkIfSecretExists(ctx, CaSecret)
	if err != nil {
		return false, err
	}
	if !caSecretExists {
		logger.Info("CA secret with cert doesn't exists")
		if err := r.doFullCertificatesRegeneration(ctx, resourcesToApply); err != nil {
			return false, err
		}
		return true, nil
	}
	logger.Info("CA secret exists")
	webhookSecretExists, err := r.checkIfSecretExists(ctx, WebhookSecret)

	if err != nil {
		return false, err
	}
	if !webhookSecretExists {
		logger.Info("webhook secret with cert does not exists")
		if err := r.doPartialCertificatesRegeneration(ctx, resourcesToApply); err != nil {
			return false, err
		}
		return true, nil
	}
	logger.Info("webhook secret exists")
	return false, nil
}

func (r *BtpOperatorReconciler) ensureSecretsDataIsSet(ctx context.Context, resourcesToApply *[]*unstructured.Unstructured) (bool, error) {
	caSecretData, err := r.getDataFromSecret(ctx, CaSecret)
	_, err = r.getValueByKey(r.buildKeyNameWithExtension(CaSecretDataPrefix, CertificatePostfix), caSecretData)
	caSecretDataIncorrect := err != nil

	_, err = r.getValueByKey(r.buildKeyNameWithExtension(CaSecretDataPrefix, RsaKeyPostfix), caSecretData)
	caSecretDataIncorrect = caSecretDataIncorrect || err != nil

	if caSecretDataIncorrect {
		if err := r.doFullCertificatesRegeneration(ctx, resourcesToApply); err != nil {
			return false, err
		}
		return true, nil
	}

	webhookSecretData, err := r.getDataFromSecret(ctx, WebhookSecret)
	_, err = r.getValueByKey(r.buildKeyNameWithExtension(WebhookSecretDataPrefix, CertificatePostfix), webhookSecretData)
	webhookSecretDataIncorrect := err != nil

	_, err = r.getValueByKey(r.buildKeyNameWithExtension(WebhookSecretDataPrefix, RsaKeyPostfix), webhookSecretData)
	webhookSecretDataIncorrect = webhookSecretDataIncorrect || err != nil

	if webhookSecretDataIncorrect {
		if err := r.doPartialCertificatesRegeneration(ctx, resourcesToApply); err != nil {
			return false, err
		}
		return true, nil
	}

	return false, nil
}

func (r *BtpOperatorReconciler) ensureCertificatesAreCorrectlyStructured(ctx context.Context, resourcesToApply *[]*unstructured.Unstructured) (bool, error) {
	logger := log.FromContext(ctx)
	logger.Info("checking structure of certificates")

	caCertificate, err := r.getCertificateFromSecret(ctx, CaSecret)
	if err != nil {
		return false, err
	}
	_, err = certs.TryDecodeCertificate(caCertificate)
	if err != nil {
		logger.Info("CA cert is structured incorrectly")
		if err := r.doFullCertificatesRegeneration(ctx, resourcesToApply); err != nil {
			return false, err
		}
		logger.Info("full regeneration done due to CA cert being structured incorrectly")
		return true, nil
	}

	webhookCertificate, err := r.getCertificateFromSecret(ctx, WebhookSecret)
	if err != nil {
		return false, err
	}
	_, err = certs.TryDecodeCertificate(webhookCertificate)
	if err != nil {
		logger.Info("webhook cert is structured incorrectly")
		if err := r.doPartialCertificatesRegeneration(ctx, resourcesToApply); err != nil {
			return false, err
		}
		logger.Info("partial regeneration done due to webhook cert being structured incorrectly")
		return true, nil
	}

	logger.Info("checking structure of certificates succeeded. no work need to be done.")
	return false, nil
}

func (r *BtpOperatorReconciler) ensureCertificatesHaveValidExpiration(ctx context.Context, resourcesToApply *[]*unstructured.Unstructured) (bool, error) {
	logger := log.FromContext(ctx)
	doCaCertificateExpiresSoon, err := r.doesCertificateExpireSoon(ctx, CaSecret)
	if err != nil {
		logger.Error(err, "CA cert is invalid")
		return false, err
	}
	if doCaCertificateExpiresSoon {
		logger.Error(nil, "CA cert expires soon")
		if err := r.doFullCertificatesRegeneration(ctx, resourcesToApply); err != nil {
			return false, err
		}
		return true, nil
	}
	logger.Info("CA certificate is valid")

	doWebhookCertificateExpiresSoon, err := r.doesCertificateExpireSoon(ctx, WebhookSecret)
	if err != nil {
		logger.Error(err, "webhook cert is invalid")
		return false, err
	}
	if doWebhookCertificateExpiresSoon {
		logger.Error(nil, "webhook cert expires soon")
		if err := r.doPartialCertificatesRegeneration(ctx, resourcesToApply); err != nil {
			return false, err
		}
		return true, nil
	}
	logger.Info("webhook certificate is valid")
	return false, nil
}

func (r *BtpOperatorReconciler) ensureCertificatesAreCorrectSigned(ctx context.Context, resourcesToApply *[]*unstructured.Unstructured) (bool, error) {
	logger := log.FromContext(ctx)
	signOk, err := r.isWebhookSecretCertSignedByCaSecretCert(ctx)
	logger.Info("checking if webhook is signed by correct CA")

	if err != nil {
		logger.Error(err, "while checking if webhook is signed by correct CA")
		if err := r.doFullCertificatesRegeneration(ctx, resourcesToApply); err != nil {
			return false, err
		}
		return true, nil
	}
	if !signOk {
		logger.Error(nil, "webhook cert is not signed by correct CA")
		if err := r.doFullCertificatesRegeneration(ctx, resourcesToApply); err != nil {
			return false, err
		}
		return true, nil
	}
	logger.Info("webhook certificate is signed by correct root CA")

	return false, nil
}

func (r *BtpOperatorReconciler) checkIfSecretExists(ctx context.Context, name string) (bool, error) {
	secret := &corev1.Secret{}
	err := r.Get(ctx, client.ObjectKey{Namespace: ChartNamespace, Name: name}, secret)
	if k8serrors.IsNotFound(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (r *BtpOperatorReconciler) doFullCertificatesRegeneration(ctx context.Context, resourcesToApply *[]*unstructured.Unstructured) error {
	logger := log.FromContext(ctx)
	logger.Info("full regeneration of certificates started")

	caCertificate, caPrivateKey, err := r.generateSelfSignedCertAndAddToApplyList(ctx, resourcesToApply)
	if err != nil {
		return fmt.Errorf("error while generating self signed cert in full regeneration proccess. %w", err)
	}

	err = r.generateSignedCertAndAddToApplyList(ctx, resourcesToApply, caCertificate, caPrivateKey)
	if err != nil {
		return fmt.Errorf("error while generating signed cert in full regeneration proccess. %w", err)
	}

	if err := r.prepareWebhooksConfigurationsReconciliationData(ctx, resourcesToApply, caCertificate); err != nil {
		return fmt.Errorf("error while reconciling webhooks. %w", err)
	}

	logger.Info("full regeneration success")
	return nil
}

func (r *BtpOperatorReconciler) doPartialCertificatesRegeneration(ctx context.Context, resourceToApply *[]*unstructured.Unstructured) error {
	logger := log.FromContext(ctx)
	logger.Info("partial regeneration started")

	err := r.generateSignedCertAndAddToApplyList(ctx, resourceToApply, nil, nil)
	if err != nil {
		return fmt.Errorf("error while generating signed cert in partial regeneration proccess. %w", err)
	}
	if err := r.prepareWebhooksConfigurationsReconciliationData(ctx, resourceToApply, nil); err != nil {
		return err
	}
	logger.Info("partial regeneration succeeded")
	return nil
}

func (r *BtpOperatorReconciler) generateSelfSignedCertAndAddToApplyList(ctx context.Context, resourcesToApply *[]*unstructured.Unstructured) ([]byte, []byte, error) {
	logger := log.FromContext(ctx)
	logger.Info("generation of self signed cert started")

	caCertificate, caPrivateKey, err := certs.GenerateSelfSignedCertificate(time.Now().UTC().Add(CaCertificateExpiration))
	if err != nil {
		return nil, nil, fmt.Errorf("while generating self signed cert: %w", err)
	}

	logger.Info("adding secret with newly generated self signed cert to list of resources to apply")
	err = r.appendCertificationDataToUnstructured(CaSecret, caCertificate, caPrivateKey, CaSecretDataPrefix, resourcesToApply)
	if err != nil {
		return nil, nil, fmt.Errorf("while adding newly generated self signed cert to list of resources to apply: %w", err)
	}

	logger.Info("generation of self signed cert succeeded")
	return caCertificate, caPrivateKey, nil
}

func (r *BtpOperatorReconciler) generateSignedCertAndAddToApplyList(ctx context.Context, resourcesToApply *[]*unstructured.Unstructured, ca, caPrivateKey []byte) error {
	logger := log.FromContext(ctx)
	logger.Info("generation of signed webhook certificate started")

	webhookCertificate, webhookPrivateKey, err := r.generateSignedCert(ctx, time.Now().UTC().Add(WebhookCertificateExpiration), ca, caPrivateKey)
	if err != nil {
		return fmt.Errorf("while generating signed webhook certificate: %w", err)
	}
	logger.Info("adding secret with newly generated signed webhook certificate to list of resources to apply")
	err = r.appendCertificationDataToUnstructured(WebhookSecret, webhookCertificate, webhookPrivateKey, WebhookSecretDataPrefix, resourcesToApply)
	if err != nil {
		return fmt.Errorf("while adding newly generated signed webhook certificate to list of resources to apply: %w", err)
	}
	logger.Info("generation of signed webhook certificate success")
	return nil
}

func (r *BtpOperatorReconciler) generateSignedCert(ctx context.Context, expiration time.Time, caCertificate, caPrivateKey []byte) ([]byte, []byte, error) {
	if caCertificate == nil || caPrivateKey == nil {
		data, err := r.getDataFromSecret(ctx, CaSecret)
		if err != nil {
			return nil, nil, err
		}

		caCertificate, err = r.getValueByKey(r.buildKeyNameWithExtension(CaSecretDataPrefix, CertificatePostfix), data)
		if err != nil {
			return nil, nil, err
		}

		caPrivateKey, err = r.getValueByKey(r.buildKeyNameWithExtension(CaSecretDataPrefix, RsaKeyPostfix), data)
		if err != nil {
			return nil, nil, err
		}
	}

	webhookCertificate, webhookPrivateKey, err := certs.GenerateSignedCertificate(expiration, caCertificate, caPrivateKey)
	if err != nil {
		return nil, nil, err
	}

	return webhookCertificate, webhookPrivateKey, err
}

func (r *BtpOperatorReconciler) appendCertificationDataToUnstructured(certName string, certificate, privateKey []byte, prefix string, resourcesToApply *[]*unstructured.Unstructured) error {
	data := r.mapCertToSecretData(certificate, privateKey, r.buildKeyNameWithExtension(prefix, CertificatePostfix), r.buildKeyNameWithExtension(prefix, RsaKeyPostfix))

	secret := r.buildSecretWithDataAndLabels(certName, data, map[string]string{managedByLabelKey: operatorName})

	unstructuredObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(secret)
	if err != nil {
		return err
	}
	*resourcesToApply = append(*resourcesToApply, &unstructured.Unstructured{Object: unstructuredObj})
	return nil
}

func (r *BtpOperatorReconciler) mapCertToSecretData(certificate, privateKey []byte, keyNameForCert, keyNameForPrivateKey string) map[string][]byte {
	return map[string][]byte{
		keyNameForCert:       certificate,
		keyNameForPrivateKey: privateKey,
	}
}

func (r *BtpOperatorReconciler) prepareWebhooksConfigurationsReconciliationData(ctx context.Context, resourcesToApply *[]*unstructured.Unstructured, expectedCa []byte) error {
	logger := log.FromContext(ctx)
	logger.Info("starting reconciliation of webhooks")
	if expectedCa == nil {
		secret := &corev1.Secret{}
		if err := r.Get(ctx, client.ObjectKey{Namespace: ChartNamespace, Name: CaSecret}, secret); err != nil {
			return err
		}
		ca, ok := secret.Data[r.buildKeyNameWithExtension(CaSecretDataPrefix, CertificatePostfix)]
		if !ok || ca == nil {
			return fmt.Errorf("while receiving certificate data from CA secret in reconcilation webhook")
		}
		expectedCa = ca
	}

	for _, resource := range *resourcesToApply {
		kind := resource.GetKind()
		if kind == MutatingWebhookConfiguration || kind == ValidatingWebhookConfiguration {
			err := r.prepareWebhookReconciliationData(ctx, resource, expectedCa)
			if err != nil {
				return err
			}
		}
	}

	logger.Info("webhooks cert bundles check success")
	return nil
}

func (r *BtpOperatorReconciler) prepareWebhookReconciliationData(ctx context.Context, webhook *unstructured.Unstructured, expectedCa []byte) error {
	const (
		WebhooksKey     = "webhooks"
		ClientConfigKey = "clientConfig"
		CaBundleKey     = "caBundle"
	)

	logger := log.FromContext(ctx)
	webhooksValue, ok := webhook.Object[WebhooksKey]
	if !ok {
		return fmt.Errorf("while geting webhooks in reconcileCaBundle")
	}
	webhooks, ok := webhooksValue.([]interface{})
	if !ok {
		return fmt.Errorf("while casting webhooks in reconcileCaBundle")
	}
	webhook.SetManagedFields(nil)

	for i, w := range webhooks {
		webhookAsMap, ok := w.(map[string]interface{})
		if !ok {
			return fmt.Errorf("could not get webhookAsMap from unstructured")
		}
		clientConfigValue, ok := webhookAsMap[ClientConfigKey]
		if !ok {
			return fmt.Errorf("while geting client config in reconcileCaBundle")
		}
		clientConfigAsMap, ok := clientConfigValue.(map[string]interface{})
		if !ok {
			return fmt.Errorf("while casting client config in reconcileCaBundle")
		}

		clientConfigAsMap[CaBundleKey] = expectedCa
		webhookAsMap[ClientConfigKey] = clientConfigAsMap
		webhooks[i] = webhookAsMap
		logger.Info("CA bundle replaced with success")
	}
	webhook.Object[WebhooksKey] = webhooks
	return nil
}

func (r *BtpOperatorReconciler) isWebhookSecretCertSignedByCaSecretCert(ctx context.Context) (bool, error) {
	//logger := log.FromContext(ctx)
	//logger.Info("CA bundle replaced with success")

	caCertificate, err := r.getCertificateFromSecret(ctx, CaSecret)
	//logger.Info("CASecret", CaSecret)
	if err != nil {
		return false, err
	}

	webhookCertificate, err := r.getCertificateFromSecret(ctx, WebhookSecret)
	//logger.Info("WebhookSecret", CaSecret)
	if err != nil {
		return false, err
	}

	ok, err := certs.VerifyIfLeafIsSignedByGivenCA(caCertificate, webhookCertificate)
	if err != nil {
		return false, err
	}

	return ok, nil
}

func (r *BtpOperatorReconciler) doesCertificateExpireSoon(ctx context.Context, secretName string) (bool, error) {
	certificate, err := r.getCertificateFromSecret(ctx, secretName)

	if err != nil {
		return false, err
	}
	certificateDecoded, err := certs.TryDecodeCertificate(certificate)
	if err != nil {
		return true, err
	}
	certificateTemplate, err := x509.ParseCertificate(certificateDecoded.Bytes)
	if err != nil {
		return false, err
	}

	expirationTriggerBound := certificateTemplate.NotAfter.UTC().Add(ExpirationBoundary)
	expiresSoon := time.Now().UTC().After(expirationTriggerBound)
	return expiresSoon, nil
}

func (r *BtpOperatorReconciler) getDataFromSecret(ctx context.Context, name string) (map[string][]byte, error) {
	secret := &corev1.Secret{}
	if err := r.Get(ctx, client.ObjectKey{Namespace: ChartNamespace, Name: name}, secret); err != nil {
		return nil, err
	}
	return secret.Data, nil
}

func (r *BtpOperatorReconciler) getCertificateFromSecret(ctx context.Context, secretName string) ([]byte, error) {
	data, err := r.getDataFromSecret(ctx, secretName)
	if err != nil {
		return nil, err
	}
	key, err := r.mapSecretNameToSecretDataKey(secretName)
	if err != nil {
		return nil, err
	}
	cert, err := r.getValueByKey(r.buildKeyNameWithExtension(key, CertificatePostfix), data)
	if err != nil {
		return nil, err
	}
	return cert, nil
}

func (r *BtpOperatorReconciler) mapSecretNameToSecretDataKey(secretName string) (string, error) {
	switch secretName {
	case CaSecret:
		return CaSecretDataPrefix, nil
	case WebhookSecret:
		return WebhookSecretDataPrefix, nil
	default:
		return "", fmt.Errorf("not found secret data key for secret name: %s", secretName)
	}
}

func (r *BtpOperatorReconciler) buildKeyNameWithExtension(filename, extension string) string {
	return fmt.Sprintf("%s.%s", filename, extension)
}

func (r *BtpOperatorReconciler) getValueByKey(key string, data map[string][]byte) ([]byte, error) {
	value, ok := data[key]
	if !ok {
		return nil, fmt.Errorf("while getting data for key: %s", key)
	}
	if value == nil || len(value) == 0 {
		return nil, fmt.Errorf("empty data for key: %s", key)
	}
	return value, nil
}

func (r *BtpOperatorReconciler) buildSecretWithDataAndLabels(name string, data map[string][]byte, labels map[string]string) *corev1.Secret {
	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ChartNamespace,
			Labels:    labels,
		},
		Data: data,
	}
}
