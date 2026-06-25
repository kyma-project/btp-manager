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
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"github.com/kyma-project/btp-manager/api/v1alpha1"
	"github.com/kyma-project/btp-manager/controllers/config"
	"github.com/kyma-project/btp-manager/internal/certs"
	"github.com/kyma-project/btp-manager/internal/conditions"
	"github.com/kyma-project/btp-manager/internal/credentials/drift"
	"github.com/kyma-project/btp-manager/internal/k8s/networkpolicy"
	"github.com/kyma-project/btp-manager/internal/manager/moduleresource"
	"github.com/kyma-project/btp-manager/internal/manifest"
	"github.com/kyma-project/btp-manager/internal/metrics"
	"github.com/kyma-project/btp-manager/internal/webhook/certificate"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
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
)

const (
	ClusterIdSecretKey            = "cluster_id"
	CredentialsNamespaceSecretKey = "credentials_namespace"

	ClusterIdConfigMapKey           = "CLUSTER_ID"
	ReleaseNamespaceConfigMapKey    = "RELEASE_NAMESPACE"
	ManagementNamespaceConfigMapKey = "MANAGEMENT_NAMESPACE"
	InitialClusterIdSecretKey       = "INITIAL_CLUSTER_ID"
	EnableLimitedCacheConfigMapKey  = "ENABLE_LIMITED_CACHE"
)

const (
	SapBtpServiceOperatorName = "sap-btp-service-operator"
	SapBtpServiceOperatorEnv  = "SAP_BTP_SERVICE_OPERATOR"

	moduleName   = "btp-operator"
	operatorName = "btp-manager"
	operandName  = "sap-btp-operator"

	sapBtpServiceOperatorSecretName          = SapBtpServiceOperatorName
	sapBtpServiceOperatorClusterIdSecretName = operandName + "-clusterid"
	sapBtpServiceOperatorConfigMapName       = operandName + "-config"

	caCertSecretName      = "ca-server-cert"
	webhookCertSecretName = "webhook-server-cert"
	mutatingWebhookName   = operandName + "-mutating-webhook-configuration"
	validatingWebhookName = operandName + "-validating-webhook-configuration"

	sapBtpServiceOperatorContainerName = "manager"

	forceDeleteLabelKey       = "force-delete"
	chartVersionLabelKey      = "chart-version"
	kymaProjectModuleLabelKey = "kyma-project.io/module"

	operatorLabelPrefix                       = "operator.kyma-project.io/"
	previousClusterIdAnnotationKey            = operatorLabelPrefix + "previous-cluster-id"
	previousCredentialsNamespaceAnnotationKey = operatorLabelPrefix + "previous-credentials-namespace"
	deletionFinalizer                         = operatorLabelPrefix + operatorName

	kubernetesAppLabelPrefix = "app.kubernetes.io/"
	managedByLabelKey        = kubernetesAppLabelPrefix + "managed-by"
	instanceLabelKey         = kubernetesAppLabelPrefix + "instance"
)

const (
	caCertSecretCertField      = "ca.crt"
	caCertSecretKeyField       = "ca.key"
	webhookCertSecretCertField = "tls.crt"
	webhookCertSecretKeyField  = "tls.key"
)

const (
	secretKind                         = "Secret"
	configMapKind                      = "ConfigMap"
	deploymentKind                     = "Deployment"
	mutatingWebhookConfigurationKind   = "MutatingWebhookConfiguration"
	validatingWebhookConfigurationKind = "ValidatingWebhookConfiguration"

	deploymentAvailableConditionType   = "Available"
	deploymentProgressingConditionType = "Progressing"

	// TODO: remove old webhook network policy name after some time
	oldWebhookNetworkPolicyName = "kyma-project.io--btp-operator-allow-to-webhook"
)

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

type InstanceBindingSerivce interface {
	DisableSISBController()
	EnableSISBController()
}

// BtpOperatorReconciler reconciles a BtpOperator object
type BtpOperatorReconciler struct {
	client.Client
	*rest.Config
	apiServerClient        client.Client
	Scheme                 *runtime.Scheme
	manifestHandler        *manifest.Handler
	webhookMetrics         *metrics.WebhookMetrics
	instanceBindingService InstanceBindingSerivce
	workqueueSize          int
	driftDetector          drift.Detector
	moduleResourceManager  moduleresource.ResourceManager
	networkPolicyManager   networkpolicy.NetworkPolicyManager
	certManager            certificate.CertificateManager
	watchHandlers          []config.WatchHandler
}

type ResourceReadiness struct {
	Name      string
	Namespace string
	Kind      string
	Ready     bool
}

func NewBtpOperatorReconciler(
	k8sClient client.Client,
	apiServerClient client.Client,
	scheme *runtime.Scheme,
	instanceBindingService InstanceBindingSerivce,
	webhookMetrics *metrics.WebhookMetrics,
	watchHandlers []config.WatchHandler,
	driftDetector drift.Detector,
	moduleResourceManager moduleresource.ResourceManager,
	networkPolicyManager networkpolicy.NetworkPolicyManager,
	certManager certificate.CertificateManager,
) *BtpOperatorReconciler {
	return &BtpOperatorReconciler{
		Client:                 k8sClient,
		apiServerClient:        apiServerClient,
		Scheme:                 scheme,
		manifestHandler:        &manifest.Handler{Scheme: scheme},
		instanceBindingService: instanceBindingService,
		webhookMetrics:         webhookMetrics,
		watchHandlers:          watchHandlers,
		driftDetector:          driftDetector,
		moduleResourceManager:  moduleResourceManager,
		networkPolicyManager:   networkPolicyManager,
		certManager:            certManager,
	}
}

// RBAC neccessary for the operator itself
//+kubebuilder:rbac:groups="operator.kyma-project.io",resources="btpoperators",verbs=get;list;watch;update;patch
//+kubebuilder:rbac:groups="operator.kyma-project.io",resources="btpoperators/status",verbs=get;update;patch
//+kubebuilder:rbac:groups="services.cloud.sap.com",resources=serviceinstances;servicebindings,verbs=create;get;list;watch;update;patch;delete
//+kubebuilder:rbac:groups="services.cloud.sap.com",resources=serviceinstances/status;servicebindings/status,verbs=get;update;patch
//+kubebuilder:rbac:groups="coordination.k8s.io",resources=leases,verbs=create;get;list;update
//+kubebuilder:rbac:groups="",resources="namespaces",verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources="pods",verbs=get;list;delete
//+kubebuilder:rbac:groups="",resources="events",verbs=create;delete;get;list;patch;update;watch
//+kubebuilder:rbac:groups="authentication.k8s.io",resources=tokenreviews,verbs=create
//+kubebuilder:rbac:groups="authorization.k8s.io",resources=subjectaccessreviews,verbs=create
//+kubebuilder:rbac:groups="networking.k8s.io",resources="networkpolicies",verbs=get;list;watch;create;update;patch;delete;deletecollection
//+kubebuilder:rbac:groups="",resources="serviceaccounts",verbs=get;create;update;patch;deletecollection
//+kubebuilder:rbac:groups="",resources="services",verbs=get;create;update;patch;deletecollection
//+kubebuilder:rbac:groups="admissionregistration.k8s.io",resources="mutatingwebhookconfigurations",verbs=get;list;watch;create;update;patch;delete;deletecollection
//+kubebuilder:rbac:groups="admissionregistration.k8s.io",resources="validatingwebhookconfigurations",verbs=get;list;watch;create;update;patch;delete;deletecollection
//+kubebuilder:rbac:groups="apiextensions.k8s.io",resources="customresourcedefinitions",verbs=get;list;watch;create;update;patch;deletecollection
//+kubebuilder:rbac:groups="apps",resources="deployments",verbs=get;list;watch;create;update;patch;delete;deletecollection
//+kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources="clusterrolebindings",verbs=get;create;update;patch;deletecollection
//+kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources="clusterroles",verbs=get;list;watch;create;update;patch;delete;deletecollection;bind
//+kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources="rolebindings",verbs=get;create;update;patch;deletecollection
//+kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources="roles",verbs=get;list;watch;create;update;patch;delete;deletecollection;bind
//+kubebuilder:rbac:groups="",resources="configmaps",verbs=deletecollection
//+kubebuilder:rbac:groups="",resources="secrets",verbs=deletecollection
//+kubebuilder:rbac:groups="services.cloud.sap.com",resources="servicebindings",verbs=deletecollection
//+kubebuilder:rbac:groups="services.cloud.sap.com",resources="serviceinstances",verbs=deletecollection

// Autogenerated RBAC from the btp-operator chart
//+kubebuilder:rbac:groups="",resources="configmaps",verbs=create;delete;get;list;patch;update;watch
//+kubebuilder:rbac:groups="",resources="configmaps/status",verbs=get;patch;update
//+kubebuilder:rbac:groups="",resources="events",verbs=create
//+kubebuilder:rbac:groups="",resources="secrets",verbs=create;delete;get;list;patch;update;watch
//+kubebuilder:rbac:groups="coordination.k8s.io",resources="leases",verbs=create;get;list;update
//+kubebuilder:rbac:groups="services.cloud.sap.com",resources="servicebindings",verbs=create;delete;get;list;patch;update;watch
//+kubebuilder:rbac:groups="services.cloud.sap.com",resources="servicebindings/status",verbs=get;patch;update
//+kubebuilder:rbac:groups="services.cloud.sap.com",resources="serviceinstances",verbs=create;delete;get;list;patch;update;watch
//+kubebuilder:rbac:groups="services.cloud.sap.com",resources="serviceinstances/status",verbs=get;patch;update

func (r *BtpOperatorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.workqueueSize += 1
	defer func() { r.workqueueSize -= 1 }()

	logger := log.FromContext(ctx)

	reconcileCr := &v1alpha1.BtpOperator{}
	if err := r.Get(ctx, req.NamespacedName, reconcileCr); err != nil {
		if k8serrors.IsNotFound(err) {
			logger.Info(fmt.Sprintf("%s BtpOperator CR not found. Ignoring it since object has been deleted.", req.Name))
			return ctrl.Result{}, nil
		}
		logger.Error(err, "unable to get BtpOperator CR")
		return ctrl.Result{}, err
	}

	if req.Name != config.BtpOperatorCrName || req.Namespace != config.KymaSystemNamespaceName {
		logger.Info(fmt.Sprintf("BtpOperator CR %s/%s is not the one we are looking for. Ignoring it.", req.Namespace, req.Name))
		return ctrl.Result{}, r.HandleWrongNamespaceOrName(ctx, reconcileCr)
	}

	if ctrlutil.AddFinalizer(reconcileCr, deletionFinalizer) {
		return ctrl.Result{}, r.Update(ctx, reconcileCr)
	}

	if !reconcileCr.ObjectMeta.DeletionTimestamp.IsZero() && reconcileCr.Status.State != v1alpha1.StateDeleting && !reconcileCr.IsReasonStringEqual(string(conditions.ServiceInstancesAndBindingsNotCleaned)) {
		return ctrl.Result{}, r.UpdateBtpOperatorStatus(ctx, reconcileCr, v1alpha1.StateDeleting, conditions.HardDeleting, "BtpOperator is to be deleted")
	}

	switch reconcileCr.Status.State {
	case "":
		return ctrl.Result{}, r.HandleInitialState(ctx, reconcileCr)
	case v1alpha1.StateProcessing:
		return ctrl.Result{RequeueAfter: config.ProcessingStateRequeueInterval}, r.HandleProcessingState(ctx, reconcileCr)
	case v1alpha1.StateWarning:
		return r.HandleWarningState(ctx, reconcileCr)
	case v1alpha1.StateError:
		return ctrl.Result{}, r.HandleErrorState(ctx, reconcileCr)
	case v1alpha1.StateDeleting:
		err := r.HandleDeletingState(ctx, reconcileCr)
		if reconcileCr.IsReasonStringEqual(string(conditions.ServiceInstancesAndBindingsNotCleaned)) {
			return ctrl.Result{RequeueAfter: config.ReadyStateRequeueInterval}, err
		}
		return ctrl.Result{}, err
	case v1alpha1.StateReady:
		return ctrl.Result{RequeueAfter: config.ReadyStateRequeueInterval}, r.HandleReadyState(ctx, reconcileCr)
	}

	return ctrl.Result{}, nil
}

func (r *BtpOperatorReconciler) HandleWrongNamespaceOrName(ctx context.Context, cr *v1alpha1.BtpOperator) error {
	return r.UpdateBtpOperatorStatus(ctx, cr, v1alpha1.StateWarning, conditions.WrongNamespaceOrName, "Your resource must be in the kyma-system namespace. The resource's name must be btpoperator.")
}

func (r *BtpOperatorReconciler) UpdateBtpOperatorStatus(ctx context.Context, cr *v1alpha1.BtpOperator, newState v1alpha1.State, reason conditions.Reason, message string) error {
	logger := log.FromContext(ctx)
	timeout := time.Now().Add(config.StatusUpdateTimeout)

	var err error
	for now := time.Now(); now.Before(timeout); now = time.Now() {
		if err = r.Get(ctx, client.ObjectKeyFromObject(cr), cr); err != nil {
			if k8serrors.IsNotFound(err) {
				return nil
			}
			logger.Error(err, fmt.Sprintf("cannot get the BtpOperator to update the status. Retrying in %s...", config.StatusUpdateCheckInterval.String()))
			time.Sleep(config.StatusUpdateCheckInterval)
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
			logger.Error(err, fmt.Sprintf("cannot update the status of the BtpOperator. Retrying in %s...", config.StatusUpdateCheckInterval.String()))
			time.Sleep(config.StatusUpdateCheckInterval)
			continue
		}
		time.Sleep(config.StatusUpdateCheckInterval)
	}
	logger.Error(err, fmt.Sprintf("timed out while waiting %s for the BtpOperator status change.", config.StatusUpdateTimeout.String()))

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

	requiredSecret, errWithReason := r.getAndVerifyRequiredSecret(ctx)
	if errWithReason != nil {
		return r.handleMissingSecret(ctx, cr, logger, errWithReason)
	}

	r.driftDetector.InitializeFromSecret(requiredSecret)

	if errWithReason := r.driftDetector.CheckCredentialsNamespaceDrift(ctx, requiredSecret); errWithReason != nil {
		return r.UpdateBtpOperatorStatus(ctx, cr, v1alpha1.StateError, errWithReason.Reason, errWithReason.Message)
	}

	if errWithReason := r.driftDetector.CheckClusterIdConfigMapDrift(ctx, requiredSecret); errWithReason != nil {
		return r.UpdateBtpOperatorStatus(ctx, cr, v1alpha1.StateError, errWithReason.Reason, errWithReason.Message)
	}

	if errWithReason := r.driftDetector.CheckClusterIdSecretDrift(ctx, requiredSecret); errWithReason != nil {
		return r.UpdateBtpOperatorStatus(ctx, cr, v1alpha1.StateError, errWithReason.Reason, errWithReason.Message)
	}

	if err := r.moduleResourceManager.DeleteOutdatedResources(ctx); err != nil {
		return r.UpdateBtpOperatorStatus(ctx, cr, v1alpha1.StateError, conditions.ProvisioningFailed, err.Error())
	}

	if err := r.reconcileResources(ctx, cr, requiredSecret); err != nil {
		return r.UpdateBtpOperatorStatus(ctx, cr, v1alpha1.StateError, conditions.ProvisioningFailed, err.Error())
	}

	if err := r.driftDetector.DeleteChangedResources(ctx); err != nil {
		logger.Error(err, "while deleting resources")
		return r.UpdateBtpOperatorStatus(ctx, cr, v1alpha1.StateError, conditions.ResourceRemovalFailed, err.Error())
	}

	r.instanceBindingService.EnableSISBController()

	logger.Info("provisioning succeeded")
	return r.UpdateBtpOperatorStatus(ctx, cr, v1alpha1.StateReady, conditions.ReconcileSucceeded, "Module provisioning succeeded")
}

func (r *BtpOperatorReconciler) handleMissingSecret(ctx context.Context, cr *v1alpha1.BtpOperator, logger logr.Logger, errWithReason *ErrorWithReason) error {
	logger.Info("secret verification failed: " + errWithReason.Error())
	return r.UpdateBtpOperatorStatus(ctx, cr, v1alpha1.StateWarning, errWithReason.Reason, errWithReason.Message)
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
	objKey := client.ObjectKey{Namespace: config.ChartNamespace, Name: config.SecretName}
	if err := r.Get(ctx, objKey, secret); err != nil {
		if k8serrors.IsNotFound(err) {
			return nil, fmt.Errorf("%s Secret in %s namespace not found", config.SecretName, config.ChartNamespace)
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
	return r.moduleResourceManager.DeleteOutdatedResources(ctx)
}

func (r *BtpOperatorReconciler) createUnstructuredObjectsFromManifestsDir(manifestsDir string) ([]*unstructured.Unstructured, error) {
	return r.moduleResourceManager.CreateUnstructuredObjectsFromManifestsDir(manifestsDir)
}

func (r *BtpOperatorReconciler) getResourcesToDeletePath() string {
	return r.moduleResourceManager.GetResourcesToDeletePath()
}

func (r *BtpOperatorReconciler) getNetworkPoliciesPath() string {
	return fmt.Sprintf("%s%cnetwork-policies", config.ManagerResourcesPath, os.PathSeparator)
}

func (r *BtpOperatorReconciler) addNetworkPoliciesToResources(ctx context.Context, resourcesToApply *[]*unstructured.Unstructured) error {
	logger := log.FromContext(ctx)
	networkPolicies, err := r.networkPolicyManager.LoadNetworkPolicies()
	if err != nil {
		logger.Error(err, "while loading network policies")
		return fmt.Errorf("failed to load network policies: %w", err)
	}
	*resourcesToApply = append(*resourcesToApply, networkPolicies...)
	logger.Info(fmt.Sprintf("added %d network policies to resources to apply", len(networkPolicies)))
	return nil
}

func (r *BtpOperatorReconciler) deleteResources(ctx context.Context, us []*unstructured.Unstructured) error {
	return r.moduleResourceManager.DeleteResources(ctx, us)
}

func (r *BtpOperatorReconciler) reconcileResources(ctx context.Context, cr *v1alpha1.BtpOperator, s *corev1.Secret) error {
	logger := log.FromContext(ctx)

	logger.Info("getting module resources to apply")
	resourcesToApply, err := r.createUnstructuredObjectsFromManifestsDir(r.getResourcesToApplyPath())
	if err != nil {
		logger.Error(err, "while creating applicable objects from manifests")
		return fmt.Errorf("failed to create applicable objects from manifests: %w", err)
	}
	logger.Info(fmt.Sprintf("got %d module resources to apply based on %s directory", len(resourcesToApply), r.getResourcesToApplyPath()))

	if cr.IsNetworkPoliciesDisabled() {
		logger.Info("network policies disabled, cleaning up existing ones")
		if err := r.networkPolicyManager.CleanupNetworkPolicies(ctx); err != nil {
			logger.Error(err, "while cleaning up network policies")
			return fmt.Errorf("failed to cleanup network policies: %w", err)
		}
	} else {
		logger.Info("network policies enabled, loading and adding them to resources")
		if err := r.addNetworkPoliciesToResources(ctx, &resourcesToApply); err != nil {
			return err
		}
	}
	if err := r.networkPolicyManager.DeleteOldWebhookNetworkPolicy(ctx); err != nil {
		logger.Error(err, "while deleting old webhook network policy")
		return fmt.Errorf("failed to delete old webhook network policy: %w", err)
	}

	if err = r.moduleResourceManager.PrepareModuleResources(ctx, resourcesToApply, s); err != nil {
		logger.Error(err, "while preparing objects to apply")
		return fmt.Errorf("failed to prepare objects to apply: %w", err)
	}

	var webhookResources, nonWebhookResources []*unstructured.Unstructured
	for _, u := range resourcesToApply {
		if certificate.IsAdmissionWebhook(u.GetKind()) {
			webhookResources = append(webhookResources, u)
		} else {
			nonWebhookResources = append(nonWebhookResources, u)
		}
	}
	preparedWebhooks, err := r.certManager.PrepareAdmissionWebhooks(ctx, webhookResources)
	if err != nil {
		logger.Error(err, "while preparing admission webhooks")
		return fmt.Errorf("failed to prepare admission webhooks: %w", err)
	}
	resourcesToApply = append(nonWebhookResources, preparedWebhooks...)

	r.moduleResourceManager.DeleteCreationTimestamp(resourcesToApply...)

	logger.Info(fmt.Sprintf("applying module resources for %d resources", len(resourcesToApply)))
	if err = r.moduleResourceManager.ApplyOrUpdateResources(ctx, resourcesToApply); err != nil {
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
	return r.moduleResourceManager.GetResourcesToApplyPath()
}

func (r *BtpOperatorReconciler) cleanupNetworkPolicies(ctx context.Context) error {
	return r.networkPolicyManager.CleanupNetworkPolicies(ctx)
}

func (r *BtpOperatorReconciler) deleteOldWebhookNetworkPolicy(ctx context.Context) error {
	return r.networkPolicyManager.DeleteOldWebhookNetworkPolicy(ctx)
}

func (r *BtpOperatorReconciler) HandleWarningState(ctx context.Context, cr *v1alpha1.BtpOperator) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Handling Warning state")

	if cr.IsReasonStringEqual(string(conditions.ServiceInstancesAndBindingsNotCleaned)) {
		err := r.handleDeleting(ctx, cr)
		if cr.IsReasonStringEqual(string(conditions.ServiceInstancesAndBindingsNotCleaned)) {
			return ctrl.Result{RequeueAfter: config.ReadyStateRequeueInterval}, err
		}
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, r.UpdateBtpOperatorStatus(ctx, cr, v1alpha1.StateProcessing, conditions.Updated, "CR has been updated")
}

func (r *BtpOperatorReconciler) HandleErrorState(ctx context.Context, cr *v1alpha1.BtpOperator) error {
	logger := log.FromContext(ctx)
	logger.Info("Handling Error state")

	return r.UpdateBtpOperatorStatus(ctx, cr, v1alpha1.StateProcessing, conditions.Updated, "CR has been updated")
}

func (r *BtpOperatorReconciler) HandleDeletingState(ctx context.Context, cr *v1alpha1.BtpOperator) error {
	logger := log.FromContext(ctx)
	logger.Info("Handling Deleting state")

	return r.handleDeleting(ctx, cr)
}

func (r *BtpOperatorReconciler) handleDeleting(ctx context.Context, cr *v1alpha1.BtpOperator) error {
	logger := log.FromContext(ctx)

	requiredSecret, err := r.getSecretByNameAndNamespace(ctx, config.SecretName, config.ChartNamespace)
	if err != nil {
		logger.Error(err, fmt.Sprintf("while getting %s secret in %s namespace", config.SecretName, config.ChartNamespace))
		return fmt.Errorf("failed to get the required secret: %w", err)
	}

	r.driftDetector.InitializeFromSecret(requiredSecret)

	if len(cr.GetFinalizers()) == 0 {
		logger.Info("BtpOperator CR without finalizers - nothing to do, waiting for deletion")
		return nil
	}

	if err = r.handleDeprovisioning(ctx, cr); err != nil {
		logger.Error(err, "deprovisioning failed. Restoring resources")
		r.reconcileResourcesWithoutChangingCrState(ctx, cr, &logger)
		return err
	}
	if cr.IsReasonStringEqual(string(conditions.ServiceInstancesAndBindingsNotCleaned)) {
		r.reconcileResourcesWithoutChangingCrState(ctx, cr, &logger)

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
	if err = r.Update(ctx, cr); err != nil {
		return err
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
			if cr.IsReasonStringEqual(string(conditions.ServiceInstancesAndBindingsNotCleaned)) && cr.Status.State == v1alpha1.StateWarning {
				return nil
			}

			if updateStatusErr := r.UpdateBtpOperatorStatus(ctx, cr,
				v1alpha1.StateWarning, conditions.ServiceInstancesAndBindingsNotCleaned, msg); updateStatusErr != nil {
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
	case <-time.After(config.HardDeleteTimeout):
		logger.Info("hard delete timeout reached", "duration", config.HardDeleteTimeout)
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

		time.Sleep(config.HardDeleteCheckInterval)
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
	deleteCtx, cancel := context.WithTimeout(ctx, config.DeleteRequestTimeout)
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
		return fmt.Errorf("failed to delete module resources: %w", err)
	}

	if err := r.cleanupNetworkPolicies(ctx); err != nil {
		logger.Error(err, "while cleaning up network policies during hard delete")
		return fmt.Errorf("failed to cleanup network policies during hard delete: %w", err)
	}

	if err := r.driftDetector.DeleteClusterIdSecret(ctx); err != nil {
		logger.Error(err, "while deleting cluster ID secret")
		return fmt.Errorf("failed to delete cluster ID secret: %w", err)
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
			u.GroupVersionKind().GroupVersion(), u.GetKind(), config.ChartNamespace))
		if err := r.DeleteAllOf(ctx, u, client.InNamespace(config.ChartNamespace), managedByLabelFilter); err != nil {
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
	toDelete := []client.Object{
		&appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: config.DeploymentName, Namespace: config.ChartNamespace}},
		&admissionregistrationv1.MutatingWebhookConfiguration{ObjectMeta: metav1.ObjectMeta{Name: mutatingWebhookName}},
		&admissionregistrationv1.ValidatingWebhookConfiguration{ObjectMeta: metav1.ObjectMeta{Name: validatingWebhookName}},
	}
	for _, obj := range toDelete {
		if err := r.Delete(ctx, obj); client.IgnoreNotFound(err) != nil {
			return err
		}
	}

	if err := r.cleanupNetworkPolicies(ctx); err != nil {
		return fmt.Errorf("failed to cleanup network policies during soft delete: %w", err)
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

	requiredSecret, errWithReason := r.getAndVerifyRequiredSecret(ctx)
	if errWithReason != nil {
		return r.handleMissingSecret(ctx, cr, logger, errWithReason)
	}

	r.driftDetector.InitializeFromSecret(requiredSecret)

	defaultCredentialsSecret, err := r.driftDetector.GetDefaultCredentialsSecret(ctx)
	if err != nil {
		logger.Error(err, fmt.Sprintf("while getting %s Secret", sapBtpServiceOperatorSecretName))
		return r.UpdateBtpOperatorStatus(ctx, cr, v1alpha1.StateError, conditions.GettingDefaultCredentialsSecretFailed, err.Error())
	}

	if defaultCredentialsSecret != nil {
		managerNs := r.driftDetector.CredentialsNamespaceFromManager()
		if managerNs != defaultCredentialsSecret.Namespace {
			msg := fmt.Sprintf("credentials namespace changed from %s to %s", defaultCredentialsSecret.Namespace, managerNs)
			logger.Info(msg)
			return r.UpdateBtpOperatorStatus(ctx, cr, v1alpha1.StateProcessing, conditions.CredentialsNamespaceChanged, msg)
		}
	}

	sapBtpOperatorConfigMap, err := r.driftDetector.GetSapBtpServiceOperatorConfigMap(ctx)
	if err != nil {
		logger.Error(err, fmt.Sprintf("while getting %s ConfigMap", sapBtpServiceOperatorConfigMapName))
		return r.UpdateBtpOperatorStatus(ctx, cr, v1alpha1.StateError, conditions.GettingSapBtpServiceOperatorConfigMapFailed, err.Error())
	}

	if sapBtpOperatorConfigMap != nil {
		clusterIdFromCM := sapBtpOperatorConfigMap.Data[strings.ToUpper(ClusterIdSecretKey)]
		if r.driftDetector.ClusterIdFromManager() != clusterIdFromCM {
			msg := fmt.Sprintf("cluster ID changed from %s to %s", clusterIdFromCM, r.driftDetector.ClusterIdFromManager())
			logger.Info(msg)
			return r.UpdateBtpOperatorStatus(ctx, cr, v1alpha1.StateProcessing, conditions.ClusterIdChanged, msg)
		}
	}

	if err := r.moduleResourceManager.DeleteOutdatedResources(ctx); err != nil {
		return r.UpdateBtpOperatorStatus(ctx, cr, v1alpha1.StateError, conditions.ReconcileFailed, err.Error())
	}

	if err := r.reconcileResources(ctx, cr, requiredSecret); err != nil {
		return r.UpdateBtpOperatorStatus(ctx, cr, v1alpha1.StateError, conditions.ReconcileFailed, err.Error())
	}

	logger.Info("reconciliation succeeded")
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *BtpOperatorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.Config = mgr.GetConfig()
	controllerBuilder := ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.BtpOperator{},
			builder.WithPredicates(r.watchBtpOperatorUpdatePredicate())).
		Watches(
			&corev1.Secret{},
			handler.EnqueueRequestsFromMapFunc(r.reconcileRequestForPrimaryBtpOperator),
			builder.WithPredicates(r.watchSecretPredicates()),
		).
		Watches(
			&admissionregistrationv1.MutatingWebhookConfiguration{},
			handler.EnqueueRequestsFromMapFunc(r.reconcileRequestForPrimaryBtpOperator),
			builder.WithPredicates(r.watchMutatingWebhooksPredicates()),
		).
		Watches(
			&admissionregistrationv1.ValidatingWebhookConfiguration{},
			handler.EnqueueRequestsFromMapFunc(r.reconcileRequestForPrimaryBtpOperator),
			builder.WithPredicates(r.watchValidatingWebhooksPredicates()),
		).
		Watches(
			&appsv1.Deployment{},
			handler.EnqueueRequestsFromMapFunc(r.reconcileRequestForPrimaryBtpOperator),
			builder.WithPredicates(r.watchDeploymentPredicates()),
		).
		Watches(
			&networkingv1.NetworkPolicy{},
			handler.EnqueueRequestsFromMapFunc(r.reconcileRequestForPrimaryBtpOperator),
			builder.WithPredicates(r.watchNetworkPolicyPredicates()),
		)

	for _, watchHandler := range r.watchHandlers {
		controllerBuilder.Watches(
			watchHandler.Object(),
			handler.EnqueueRequestsFromMapFunc(watchHandler.Reconcile),
			builder.WithPredicates(watchHandler.Predicates()),
		)
	}

	return controllerBuilder.Complete(r)
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

func (r *BtpOperatorReconciler) reconcileRequestForPrimaryBtpOperator(ctx context.Context, obj client.Object) []reconcile.Request {
	return []reconcile.Request{{NamespacedName: k8sgenerictypes.NamespacedName{Name: config.BtpOperatorCrName, Namespace: config.KymaSystemNamespaceName}}}
}

func (r *BtpOperatorReconciler) watchSecretPredicates() predicate.TypedPredicate[client.Object] {
	return predicate.TypedFuncs[client.Object]{
		CreateFunc: func(e event.CreateEvent) bool {
			secret, ok := e.Object.(*corev1.Secret)
			if !ok {
				return false
			}
			return r.isManagedSecret(secret)
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			secret, ok := e.Object.(*corev1.Secret)
			if !ok {
				return false
			}
			return r.isManagedSecret(secret)
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			oldSecret, ok := e.ObjectOld.(*corev1.Secret)
			if !ok {
				return false
			}
			return r.isManagedSecret(oldSecret)
		},
	}
}

func deploymentConditionStatuses(d *appsv1.Deployment) (available, progressing string) {
	for _, c := range d.Status.Conditions {
		switch string(c.Type) {
		case deploymentAvailableConditionType:
			available = string(c.Status)
		case deploymentProgressingConditionType:
			progressing = string(c.Status)
		}
	}
	return
}

func (r *BtpOperatorReconciler) watchDeploymentPredicates() predicate.Funcs {
	isManagedDeployment := func(obj client.Object) bool {
		return obj.GetName() == config.DeploymentName && obj.GetNamespace() == config.ChartNamespace
	}
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return isManagedDeployment(e.Object)
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return isManagedDeployment(e.Object)
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			if !isManagedDeployment(e.ObjectNew) {
				return false
			}
			newAvail, newProg := deploymentConditionStatuses(e.ObjectNew.(*appsv1.Deployment))
			oldAvail, oldProg := deploymentConditionStatuses(e.ObjectOld.(*appsv1.Deployment))
			return newAvail != oldAvail || newProg != oldProg
		},
	}
}

func (r *BtpOperatorReconciler) getNetworkPolicyNamesFromManifests() (map[string]struct{}, error) {
	names := make(map[string]struct{})
	us, err := r.networkPolicyManager.LoadNetworkPolicies()
	if err != nil {
		return names, err
	}
	for _, u := range us {
		if n := u.GetName(); n != "" {
			names[n] = struct{}{}
		}
	}
	return names, nil
}

func (r *BtpOperatorReconciler) watchNetworkPolicyPredicates() predicate.Funcs {
	nameSet, _ := r.getNetworkPolicyNamesFromManifests()
	isManaged := func(obj *networkingv1.NetworkPolicy) bool {
		labels := obj.GetLabels()
		if labels != nil {
			if labels[managedByLabelKey] == operatorName && labels[kymaProjectModuleLabelKey] == moduleName {
				return true
			}
		}
		if _, ok := nameSet[obj.GetName()]; ok {
			return true
		}
		return false
	}

	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			obj := e.Object.(*networkingv1.NetworkPolicy)
			return isManaged(obj)
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			obj := e.Object.(*networkingv1.NetworkPolicy)
			return isManaged(obj)
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			objOld := e.ObjectOld.(*networkingv1.NetworkPolicy)
			objNew := e.ObjectNew.(*networkingv1.NetworkPolicy)

			return isManaged(objOld) || isManaged(objNew)
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

func (r *BtpOperatorReconciler) IsForceDelete(cr *v1alpha1.BtpOperator) bool {
	if _, exists := cr.Labels[forceDeleteLabelKey]; !exists {
		return false
	}
	return cr.Labels[forceDeleteLabelKey] == "true"
}

func (r *BtpOperatorReconciler) getSecretByNameAndNamespace(ctx context.Context, name, namespace string) (*corev1.Secret, error) {
	secret := &corev1.Secret{}
	if err := r.apiServerClient.Get(ctx, client.ObjectKey{Name: name, Namespace: namespace}, secret); err != nil {
		if k8serrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("while getting %s secret from %s namespace: %w", name, namespace, err)
	}
	return secret, nil
}

func (r *BtpOperatorReconciler) deleteObject(ctx context.Context, obj client.Object) error {
	if err := r.apiServerClient.Delete(ctx, obj); err != nil {
		return fmt.Errorf("while deleting %s %s from %s namespace: %w", obj.GetName(), obj.GetObjectKind().GroupVersionKind().Kind, obj.GetNamespace(), err)
	}
	return nil
}

func (r *BtpOperatorReconciler) reconcileResourcesWithoutChangingCrState(ctx context.Context, cr *v1alpha1.BtpOperator, logger *logr.Logger) {
	secret, errWithReason := r.getAndVerifyRequiredSecret(ctx)
	if errWithReason != nil {
		logger.Error(errWithReason, "secret verification failed")
		return
	}
	r.driftDetector.InitializeFromSecret(secret)
	if err := r.deleteOutdatedResources(ctx); err != nil {
		logger.Error(err, "outdated resources deletion failed")
	}
	if err := r.reconcileResources(ctx, cr, secret); err != nil {
		logger.Error(err, "resources reconciliation failed")
	}
}

func (r *BtpOperatorReconciler) isManagedSecret(s *corev1.Secret) bool {
	return r.isCredentialsSecret(s) || r.isCertSecret(s)
}

func (r *BtpOperatorReconciler) isCredentialsSecret(s *corev1.Secret) bool {
	return s.Namespace == config.ChartNamespace && s.Name == config.SecretName
}

func (r *BtpOperatorReconciler) isCertSecret(s *corev1.Secret) bool {
	return s.Namespace == config.ChartNamespace && (s.Name == caCertSecretName || s.Name == webhookCertSecretName)
}

func (r *BtpOperatorReconciler) isWebhookCertSignedBySelfSignedCa(ctx context.Context) (bool, error) {
	getSecretCert := func(secretName, fieldName string) ([]byte, error) {
		secret := &corev1.Secret{}
		if err := r.Get(ctx, client.ObjectKey{Namespace: config.ChartNamespace, Name: secretName}, secret); err != nil {
			return nil, err
		}
		val, ok := secret.Data[fieldName]
		if !ok || len(val) == 0 {
			return nil, fmt.Errorf("key %s missing in secret %s", fieldName, secretName)
		}
		return val, nil
	}
	caCert, err := getSecretCert(caCertSecretName, caCertSecretCertField)
	if err != nil {
		return false, err
	}
	webhookCert, err := getSecretCert(webhookCertSecretName, webhookCertSecretCertField)
	if err != nil {
		return false, err
	}
	return certs.VerifyIfLeafIsSignedByGivenCA(caCert, webhookCert)
}

func (r *BtpOperatorReconciler) getDataFromSecret(ctx context.Context, name string) (map[string][]byte, error) {
	secret := &corev1.Secret{}
	if err := r.Get(ctx, client.ObjectKey{Namespace: config.ChartNamespace, Name: name}, secret); err != nil {
		return nil, err
	}
	return secret.Data, nil
}

func (r *BtpOperatorReconciler) getSecretDataValueByKey(key string, data map[string][]byte) ([]byte, error) {
	value, ok := data[key]
	if !ok {
		return nil, fmt.Errorf("missing key: %s", key)
	}
	if len(value) == 0 {
		return nil, fmt.Errorf("empty value for key: %s", key)
	}
	return value, nil
}

func certFieldFromSecretBySecretName(secretName string) (string, error) {
	switch secretName {
	case caCertSecretName:
		return caCertSecretCertField, nil
	case webhookCertSecretName:
		return webhookCertSecretCertField, nil
	}
	return "", fmt.Errorf("unknown secret %q - cert field undefined", secretName)
}

func (r *BtpOperatorReconciler) waitForResourcesReadiness(ctx context.Context, us []*unstructured.Unstructured) error {
	numOfResources := len(us)
	readinessCh := make(chan ResourceReadiness, numOfResources)
	for _, u := range us {
		if u.GetKind() == deploymentKind {
			go r.checkDeploymentReadiness(ctx, u, readinessCh)
			continue
		}
		go r.checkResourceExistence(ctx, u, readinessCh)
	}
	for i := 0; i < numOfResources; i++ {
		if rr := <-readinessCh; !rr.Ready {
			return fmt.Errorf("%s %s in namespace %s readiness timeout reached", rr.Kind, rr.Name, rr.Namespace)
		}
	}
	return nil
}

func (r *BtpOperatorReconciler) checkDeploymentReadiness(ctx context.Context, u *unstructured.Unstructured, c chan<- ResourceReadiness) {
	logger := log.FromContext(ctx)

	var err error
	var availableStatus, progressingStatus string
	got := &appsv1.Deployment{}
	now := time.Now()
	for {
		if time.Since(now) >= config.ReadyTimeout {
			logger.Error(err, fmt.Sprintf("timed out while checking %s %s readiness", u.GetName(), u.GetKind()))
			c <- ResourceReadiness{Name: u.GetName(), Namespace: u.GetNamespace(), Kind: u.GetKind(), Ready: false}
			return
		}
		ctxWithTimeout, cancel := context.WithTimeout(ctx, config.ReadyCheckInterval)
		err = r.Get(ctxWithTimeout, client.ObjectKey{Name: u.GetName(), Namespace: u.GetNamespace()}, got)
		cancel()
		if err == nil {
			for _, cond := range got.Status.Conditions {
				if string(cond.Type) == deploymentProgressingConditionType {
					progressingStatus = string(cond.Status)
				} else if string(cond.Type) == deploymentAvailableConditionType {
					availableStatus = string(cond.Status)
				}
			}
			if progressingStatus == "True" && availableStatus == "True" {
				c <- ResourceReadiness{Ready: true}
				return
			}
		}
	}
}

func (r *BtpOperatorReconciler) checkResourceExistence(ctx context.Context, u *unstructured.Unstructured, c chan<- ResourceReadiness) {
	logger := log.FromContext(ctx)

	var err error
	now := time.Now()
	got := &unstructured.Unstructured{}
	got.SetGroupVersionKind(u.GroupVersionKind())
	for {
		if time.Since(now) >= config.ReadyTimeout {
			logger.Error(err, fmt.Sprintf("timed out while checking %s %s existence", u.GetName(), u.GetKind()))
			c <- ResourceReadiness{Name: u.GetName(), Namespace: u.GetNamespace(), Kind: u.GetKind(), Ready: false}
			return
		}
		ctxWithTimeout, cancel := context.WithTimeout(ctx, config.ReadyCheckInterval)
		err = r.Get(ctxWithTimeout, client.ObjectKey{Name: u.GetName(), Namespace: u.GetNamespace()}, got)
		cancel()
		if err == nil {
			c <- ResourceReadiness{Ready: true}
			return
		}
	}
}
