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
	"bytes"
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/kyma-project/btp-manager/internal/certs"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/kyma-project/btp-manager/api/v1alpha1"
	"github.com/kyma-project/btp-manager/internal/manifest"
	"github.com/kyma-project/btp-manager/internal/ymlutils"
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
	ChartPath                      = "./module-chart/chart"
	HardDeleteCheckInterval        = time.Second * 10
	ResourcesPath                  = "./module-resources"
	RootCertExpiration             = time.Now().Add(time.Minute * 5)
	WebhookCertExpriation          = time.Now().Add(time.Minute * 5)
	ExpirationBoundary             = time.Hour * -72
)

const (
	secretKind                  = "Secret"
	configMapKind               = "ConfigMap"
	operatorName                = "btp-manager"
	deletionFinalizer           = "custom-deletion-finalizer"
	managedByLabelKey           = "app.kubernetes.io/managed-by"
	btpServiceOperatorConfigMap = "sap-btp-operator-config"
	btpServiceOperatorSecret    = "sap-btp-service-operator"
	mutatingWebhookName         = "sap-btp-operator-mutating-webhook-configuration"
	validatingWebhookName       = "sap-btp-operator-validating-webhook-configuration"
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
	managedByLabelFilter = client.MatchingLabels{managedByLabelKey: operatorName}
)

var (
	CaCertSecretName      = "ca-server-cert"
	WebhookCertSecretName = "webhook-server-cert"
	validatingWebhookGvk  = (&admissionregistrationv1.ValidatingWebhookConfiguration{}).GroupVersionKind()
	mutatingWebhookGvk    = (&admissionregistrationv1.MutatingWebhookConfigurationList{}).GroupVersionKind()
)

// BtpOperatorReconciler reconciles a BtpOperator object
type BtpOperatorReconciler struct {
	client.Client
	*rest.Config
	Scheme          *runtime.Scheme
	manifestHandler *manifest.Handler
	workqueueSize   int
}

func NewBtpOperatorReconciler(client client.Client, scheme *runtime.Scheme) *BtpOperatorReconciler {
	return &BtpOperatorReconciler{
		Client:          client,
		Scheme:          scheme,
		manifestHandler: &manifest.Handler{Scheme: scheme},
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
//+kubebuilder:rbac:groups="batch",resources="jobs",verbs="*"
//+kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources="clusterrolebindings",verbs="*"
//+kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources="clusterroles",verbs="*"
//+kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources="rolebindings",verbs="*"
//+kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources="roles",verbs="*"

func (r *BtpOperatorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.workqueueSize += 1
	defer func() { r.workqueueSize -= 1 }()
	logger := log.FromContext(ctx)
	cr := &v1alpha1.BtpOperator{}
	if err := r.Get(ctx, req.NamespacedName, cr); err != nil {
		if k8serrors.IsNotFound(err) {
			logger.Info("BtpOperator CR not found. Ignoring since object has been deleted.")
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
	return r.UpdateBtpOperatorStatus(ctx, cr, types.StateError, OlderCRExists, fmt.Sprintf("'%s' BtpOperator CR in '%s' namespace reconciles the module",
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

	secret, errWithReason := r.getAndVerifyRequiredSecret(ctx)
	if errWithReason != nil {
		return r.UpdateBtpOperatorStatus(ctx, cr, types.StateError, errWithReason.reason, errWithReason.message)
	}

	if err := r.deleteOutdatedResources(ctx); err != nil {
		return r.UpdateBtpOperatorStatus(ctx, cr, types.StateError, ProvisioningFailed, err.Error())
	}

	if err := r.reconcileResources(ctx, secret); err != nil {
		return r.UpdateBtpOperatorStatus(ctx, cr, types.StateError, ProvisioningFailed, err.Error())
	}

	logger.Info("provisioning succeeded")
	return r.UpdateBtpOperatorStatus(ctx, cr, types.StateReady, ReconcileSucceeded, "Module provisioning succeeded")
}

func (r *BtpOperatorReconciler) getAndVerifyRequiredSecret(ctx context.Context) (*corev1.Secret, *ErrorWithReason) {
	logger := log.FromContext(ctx)

	logger.Info("getting the required Secret")
	secret, err := r.getRequiredSecret(ctx)
	if err != nil {
		logger.Error(err, "while getting the required Secret")
		return nil, NewErrorWithReason(MissingSecret, "Secret resource not found")
	}

	logger.Info("verifying the required Secret")
	if err = r.verifySecret(secret); err != nil {
		logger.Error(err, "while verifying the required Secret")
		return nil, NewErrorWithReason(InvalidSecret, "Secret validation failed")
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
		return fmt.Errorf("Failed to certtifactGenerator deletable objects from manifests: %w", err)
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
		return fmt.Errorf("Failed to certtifactGenerator applicable objects from manifests: %w", err)
	}
	logger.Info(fmt.Sprintf("got %d module resources to apply", len(resourcesToApply)))

	logger.Info("preparing module resources to apply")
	if err = r.prepareModuleResources(ctx, &resourcesToApply, s); err != nil {
		logger.Error(err, "while preparing objects to apply")
		return fmt.Errorf("Failed to prepare objects to apply: %w", err)
	}

	logger.Info("applying module resources")
	if err = r.applyResources(ctx, resourcesToApply); err != nil {
		logger.Error(err, "while applying module resources")
		return fmt.Errorf("Failed to apply module resources: %w", err)
	}

	logger.Info("waiting for module resources readiness")
	if err = r.waitForResourcesReadiness(ctx, resourcesToApply); err != nil {
		logger.Error(err, "while waiting for module resources readiness")
		return fmt.Errorf("Timed out while waiting for resources readiness: %w", err)
	}

	return nil
}

func (r *BtpOperatorReconciler) getResourcesToApplyPath() string {
	return fmt.Sprintf("%s%capply", ResourcesPath, os.PathSeparator)
}

func (r *BtpOperatorReconciler) reconcileWebhooks() error {

}

func (r *BtpOperatorReconciler) prepareModuleResources(ctx context.Context, resourcesToApply *[]*unstructured.Unstructured, s *corev1.Secret) error {
	logger := log.FromContext(ctx)

	var configMapIndex, secretIndex int
	var webhooksIndex []int
	for i, u := range *resourcesToApply {
		if u.GetName() == btpServiceOperatorConfigMap && u.GetKind() == configMapKind {
			configMapIndex = i
		}
		if u.GetName() == btpServiceOperatorSecret && u.GetKind() == secretKind {
			secretIndex = i
		}
		if u.GetName() == "" && u.GetKind() == "" {
			webhooksIndex = append(webhooksIndex, i)
		}
	}

	chartVer, err := ymlutils.ExtractStringValueFromYamlForGivenKey(fmt.Sprintf("%s/Chart.yaml", ChartPath), "version")
	if err != nil {
		logger.Error(err, "while getting module chart version")
		return fmt.Errorf("failed to get module chart version: %w", err)
	}

	r.addLabels(chartVer, *resourcesToApply...)
	r.setNamespace(*resourcesToApply...)
	r.deleteCreationTimestamp(*resourcesToApply...)
	if err := r.setConfigMapValues(s, (*resourcesToApply)[configMapIndex]); err != nil {
		logger.Error(err, "while setting ConfigMap values")
		return fmt.Errorf("failed to set ConfigMap values: %w", err)
	}
	if err := r.setSecretValues(s, (*resourcesToApply)[secretIndex]); err != nil {
		logger.Error(err, "while setting Secret values")
		return fmt.Errorf("failed to set Secret values: %w", err)
	}

	if err := r.checkWebhooksCertifications(resourcesToApply); err != nil {
		return fmt.Errorf("failed to reconcile webhook certs: %w", err)
	}

	return nil
}

func (r *BtpOperatorReconciler) checkWebhooksCertifications(resourcesToApply *[]*unstructured.Unstructured) error {
	rootCertExists, err := r.doSecretExists(CaCertSecretName)
	if err != nil {
		return err
	}
	if !rootCertExists {
		if err := r.fullyRegenerate(resourcesToApply); err != nil {
			return err
		}
		return nil
	}

	webhookCertExists, err := r.doSecretExists(WebhookCertSecretName)
	if err != nil {
		return err
	}
	if !webhookCertExists {
		if err := r.partialRegenerate(resourcesToApply); err != nil {
			return err
		}
	}

	if err := r.reconcileWebhookCABundles(resourcesToApply, nil); err != nil {
		return err
	}

	signOk, err := r.isWebhookCertSignedByRoot()
	if err != nil {
		return err
	}
	if !signOk {
		if err := r.fullyRegenerate(resourcesToApply); err != nil {
			return nil
		}
		return nil
	}

	rootCertValid, err := r.checkIfCertExpire(CaCertSecretName)
	if err != nil {
		return err
	}
	if !rootCertValid {
		if err := r.fullyRegenerate(resourcesToApply); err != nil {
			return err
		}
		return nil
	}

	caWebhookCertValid, err := r.checkIfCertExpire(WebhookCertSecretName)
	if err != nil {
		return err
	}
	if !caWebhookCertValid {
		if err := r.partialRegenerate(resourcesToApply); err != nil {
			return err
		}
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

func (r *BtpOperatorReconciler) applyResources(ctx context.Context, us []*unstructured.Unstructured) error {
	for _, u := range us {
		if err := r.Patch(ctx, u, client.Apply, client.ForceOwnership, client.FieldOwner(operatorName)); err != nil {
			return fmt.Errorf("while applying %s %s: %w", u.GetName(), u.GetKind(), err)
		}
	}

	return nil
}

func (r *BtpOperatorReconciler) waitForResourcesReadiness(ctx context.Context, us []*unstructured.Unstructured) error {
	numOfResources := len(us)
	resourcesReadinessInformer := make(chan bool, numOfResources)
	allReadyInformer := make(chan bool)
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
	logger := log.FromContext(ctx)
	ctxWithTimeout, cancel := context.WithTimeout(ctx, ReadyCheckInterval/2)
	defer cancel()

	var err error
	now := time.Now()
	got := &unstructured.Unstructured{}
	got.SetGroupVersionKind(u.GroupVersionKind())
	for {
		if time.Since(now) >= ReadyTimeout {
			logger.Error(err, fmt.Sprintf("failed to get %s %s from the cluster", u.GetName(), u.GetKind()))
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

	return r.UpdateBtpOperatorStatus(ctx, cr, types.StateProcessing, Updated, "CR has been updated")
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
			logger.Info("Service Instances and Service Bindings hard delete succeeded. Removing module resources")
			if err := r.deleteBtpOperatorResources(ctx); err != nil {
				logger.Error(err, "failed to remove module resources")
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
	deleteCtx, cancel := context.WithTimeout(ctx, HardDeleteTimeout/2)
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
		return r.UpdateBtpOperatorStatus(ctx, cr, types.StateError, errWithReason.reason, errWithReason.message)
	}

	if err := r.deleteOutdatedResources(ctx); err != nil {
		return r.UpdateBtpOperatorStatus(ctx, cr, types.StateError, ReconcileFailed, err.Error())
	}

	if err := r.reconcileResources(ctx, secret); err != nil {
		return r.UpdateBtpOperatorStatus(ctx, cr, types.StateError, ReconcileFailed, err.Error())
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

func (r *BtpOperatorReconciler) mapCertToSecretData(cert []byte, privateKey *rsa.PrivateKey, keyNameForCert, keyNameForPrivateKey string) (error, map[string][]byte) {
	b, err := json.Marshal(privateKey)
	if err != nil {
		return err, nil
	}

	return nil, map[string][]byte{
		keyNameForCert:       cert,
		keyNameForPrivateKey: b,
	}
}

func (r *BtpOperatorReconciler) buildSecretWithData(name string, data map[string][]byte) *corev1.Secret {
	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "kyma-system",
			Labels: map[string]string{
				managedByLabelKey: operatorName,
			},
		},
		Data: data,
	}
}

func (r *BtpOperatorReconciler) genereateSelfSignedCertAndAddToResources(resourcesToApply *[]*unstructured.Unstructured) ([]byte, *rsa.PrivateKey, error) {
	CA, CAPrivateKey, err := certs.GenerateSelfSignedCert(RootCertExpiration)
	if err != nil {
		return []byte{}, nil, err
	}
	err = r.applyUnstructuredCertAsSecret(resourcesToApply, CaCertSecretName, CA, CAPrivateKey, "CA")
	if err != nil {
		return []byte{}, nil, err
	}
	return CA, CAPrivateKey, nil
}

func (r *BtpOperatorReconciler) genereateSignedCertAndAddToResources(resourcesToApply *[]*unstructured.Unstructured, CA []byte, CAPrivateKey *rsa.PrivateKey) error {
	cert, certPk, err := r.generateSignedCert(WebhookCertExpriation, CA, CAPrivateKey)
	if err != nil {
		return err
	}
	err = r.applyUnstructuredCertAsSecret(resourcesToApply, WebhookCertSecretName, cert, certPk, "tls")
	if err != nil {
		return err
	}
	return nil
}

func (r *BtpOperatorReconciler) generateSignedCert(expiration time.Time, ca []byte, pk *rsa.PrivateKey) ([]byte, *rsa.PrivateKey, error) {
	if ca == nil || pk == nil {
		data, err := r.getDataFromSecret(CaCertSecretName)
		if err != nil {
			return []byte{}, nil, err
		}

		ca, err = r.getValueByKey("ca.crt", data)
		if err != nil {
			return []byte{}, nil, err
		}

		pkb, err := r.getValueByKey("ca.key", data)
		if err != nil {
			return []byte{}, nil, err
		}

		err = binary.Read(bytes.NewReader(pkb), binary.LittleEndian, &pk)
		if err != nil {
			return []byte{}, nil, err
		}
	}

	cert, certPk, err := certs.GenerateSignedCert(expiration, ca, pk)
	if err != nil {
		return []byte{}, nil, err
	}

	return cert, certPk, err
}

func (r *BtpOperatorReconciler) getDataFromSecret(name string) (map[string][]byte, error) {
	secret := &corev1.Secret{}
	if err := r.Get(context.TODO(), client.ObjectKey{Namespace: ChartNamespace, Name: name}, secret); err != nil {
		return nil, err
	}
	return secret.Data, nil
}

func (r *BtpOperatorReconciler) getValueByKey(key string, m map[string][]byte) ([]byte, error) {
	value, ok := m[key]
	if !ok {
		return nil, fmt.Errorf("")
	}
	return value, nil
}

func (r *BtpOperatorReconciler) isResourceWebhook(kind string) bool {
	if kind == "MutatingWebhookConfiguration" || kind == "ValidatingWebhookConfiguration" {
		return true
	}
	return false
}

func (r *BtpOperatorReconciler) applyUnstructuredCertAsSecret(us *[]*unstructured.Unstructured, certName string, cert []byte, pk *rsa.PrivateKey, prefix string) error {
	certSecretExists, err := r.doSecretExists(certName)
	if err != nil {
		return err
	} else if certSecretExists {
		return nil
	}

	if err != nil {
		return err
	}
	err, data := r.MapCertToSecretData(cert, pk, fmt.Sprintf("%s.crt", prefix), fmt.Sprintf("%s.key", prefix))
	if err != nil {
		return err
	}
	secret := r.buildSecretWithData(certName, data)

	unstructuredObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(secret)
	if err != nil {
		return err
	}
	*us = append(*us, &unstructured.Unstructured{Object: unstructuredObj})
	return nil
}

func (r *BtpOperatorReconciler) doSecretExists(name string) (bool, error) {
	ca := &corev1.Secret{}
	err := r.Get(context.TODO(), client.ObjectKey{Namespace: ChartNamespace, Name: name}, ca)
	if err != nil {
		return false, nil
	}

	return true, nil
}

func (r *BtpOperatorReconciler) reconcileWebhookCABundles(us *[]*unstructured.Unstructured, value []byte) error {
	for _, webhook := range *us {
		webhookValue := reflect.ValueOf(webhook)
		attachedWebhooks := webhookValue.FieldByName("Webhooks")
		for i := 0; i < attachedWebhooks.NumField(); i++ {
			admissionReviewVersion := reflect.ValueOf(attachedWebhooks.Field(i).Interface())
			clientConfig := admissionReviewVersion.FieldByName("clientConfig")
			caBundle := clientConfig.FieldByName("caBundle")
			caBundle.SetBytes(value)
		}
	}
	return nil
}

func (r *BtpOperatorReconciler) mapSecretNameToKeyPrefix(secretName string) (string, error) {
	switch secretName {
	case CaCertSecretName:
		return "ca", nil
	case WebhookCertSecretName:
		return "tls", nil
	default:
		return "", fmt.Errorf("")
	}
}

func (r *BtpOperatorReconciler) buildFilenameWithExtension(filename, extension string) string {
	return fmt.Sprintf("%s.%s", filename, extension)
}

func (r *BtpOperatorReconciler) checkIfCertExpire(secretName string) (bool, error) {
	caData, err := r.getDataFromSecret(secretName)
	if err != nil {
		return true, err
	}

	prefix, err := r.mapSecretNameToKeyPrefix(secretName)
	if err != nil {
		return false, err
	}

	rr, err := r.GetValueByKey(r.buildFilenameWithExtension(prefix, "crt"), caData)
	if err != nil {
		return true, err
	}

	caTemplate, err := x509.ParseCertificate(rr)
	_, err = caTemplate.Verify(x509.VerifyOptions{})
	if err != nil {
		return true, err
	}

	expirationTriggerBound := caTemplate.NotAfter.Add(ExpirationBoundary) //NotAfter is expiration day
	expiresSoon := time.Now().After(expirationTriggerBound)
	return expiresSoon, nil
}

func (*BtpOperatorReconciler) findResourceInUnstructured() *unstructured.Unstructured {
	return nil
}

func (r *BtpOperatorReconciler) isWebhookCertSignedByRoot() (bool, error) {
	return true, nil
}

func (r *BtpOperatorReconciler) fullyRegenerate(resourcesToApply *[]*unstructured.Unstructured) error {
	CA, CAPk, err := r.GenereateSelfSignedCertAndAddToResources(resourcesToApply)
	if err != nil {
		return err
	}
	err = r.genereateSignedCertAndAddToResources(resourcesToApply, CA, CAPk)
	if err != nil {
		return err
	}
	return nil
}

func (r *BtpOperatorReconciler) partialRegenerate(resourceToApply *[]*unstructured.Unstructured) error {
	err := r.genereateSignedCertAndAddToResources(resourceToApply, nil, nil)
	if err != nil {
		return err
	}
	return nil
}
