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

	"github.com/go-logr/logr"
	"github.com/kyma-project/btp-manager/api/v1alpha1"
	"github.com/kyma-project/btp-manager/controllers/config"
	"github.com/kyma-project/btp-manager/internal/conditions"
	"github.com/kyma-project/btp-manager/internal/credentials/drift"
	"github.com/kyma-project/btp-manager/internal/deprovisioning"
	"github.com/kyma-project/btp-manager/internal/k8s/networkpolicy"
	"github.com/kyma-project/btp-manager/internal/manager/moduleresource"
	"github.com/kyma-project/btp-manager/internal/metrics"
	"github.com/kyma-project/btp-manager/internal/provisioning"
	"github.com/kyma-project/btp-manager/internal/webhook/certificate"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"

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
	webhookMetrics         *metrics.WebhookMetrics
	instanceBindingService InstanceBindingSerivce
	workqueueSize          int
	networkPolicyManager   networkpolicy.NetworkPolicyManager
	driftDetector          drift.Detector
	moduleResourceManager  moduleresource.ResourceManager
	certManager            certificate.CertificateManager
	provisioningHandler    provisioning.Handler
	watchHandlers          []config.WatchHandler
	deprovisioningHandler  deprovisioning.Handler
}

func NewBtpOperatorReconciler(client client.Client, apiServerClient client.Client, scheme *runtime.Scheme, instanceBindingSerivice InstanceBindingSerivce, metrics *metrics.WebhookMetrics, watchHandlers []config.WatchHandler, networkPolicyManager networkpolicy.NetworkPolicyManager, driftDetector drift.Detector, moduleResourceManager moduleresource.ResourceManager, certManager certificate.CertificateManager, provisioningHandler provisioning.Handler) *BtpOperatorReconciler {
	return &BtpOperatorReconciler{
		Client:                 client,
		apiServerClient:        apiServerClient,
		Scheme:                 scheme,
		instanceBindingService: instanceBindingSerivice,
		webhookMetrics:         metrics,
		watchHandlers:          watchHandlers,
		networkPolicyManager:   networkPolicyManager,
		driftDetector:          driftDetector,
		moduleResourceManager:  moduleResourceManager,
		certManager:            certManager,
		provisioningHandler:    provisioningHandler,
	}
}

func (r *BtpOperatorReconciler) SetDeprovisioningHandler(h deprovisioning.Handler) {
	r.deprovisioningHandler = h
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
	log.FromContext(ctx).Info("Handling Processing state")
	result := r.provisioningHandler.Provision(ctx, cr)
	if result.WarningReason != nil {
		return r.UpdateBtpOperatorStatus(ctx, cr, v1alpha1.StateWarning, result.WarningReason.Reason, result.WarningReason.Message)
	}
	if result.ErrorReason != nil {
		return r.UpdateBtpOperatorStatus(ctx, cr, v1alpha1.StateError, result.ErrorReason.Reason, result.ErrorReason.Message)
	}
	return r.UpdateBtpOperatorStatus(ctx, cr, v1alpha1.StateReady, conditions.ReconcileSucceeded, "Module provisioning succeeded")
}

func (r *BtpOperatorReconciler) handleSecretVerificationFailure(ctx context.Context, cr *v1alpha1.BtpOperator, logger logr.Logger, errWithReason *ErrorWithReason) error {
	logger.Info("secret verification failed: " + errWithReason.Error())
	if errWithReason.Reason == conditions.InvalidSecret {
		return r.UpdateBtpOperatorStatus(ctx, cr, v1alpha1.StateError, errWithReason.Reason, errWithReason.Message)
	}
	return r.UpdateBtpOperatorStatus(ctx, cr, v1alpha1.StateWarning, errWithReason.Reason, errWithReason.Message)
}

func (r *BtpOperatorReconciler) getAndVerifyRequiredSecret(ctx context.Context) (*corev1.Secret, *ErrorWithReason) {
	return r.provisioningHandler.GetAndVerifyRequiredSecret(ctx)
}

func (r *BtpOperatorReconciler) reconcileResources(ctx context.Context, cr *v1alpha1.BtpOperator, s *corev1.Secret) error {
	return r.provisioningHandler.ReconcileResources(ctx, cr, s)
}

func (r *BtpOperatorReconciler) HandleWarningState(ctx context.Context, cr *v1alpha1.BtpOperator) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Handling Warning state")

	if cr.IsReasonStringEqual(string(conditions.ServiceInstancesAndBindingsNotCleaned)) {
		if r.deprovisioningHandler == nil {
			return ctrl.Result{}, fmt.Errorf("deprovisioningHandler is not set; call SetDeprovisioningHandler before starting the manager")
		}
		err := r.deprovisioningHandler.Deprovision(ctx, cr)
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

	if r.deprovisioningHandler == nil {
		return fmt.Errorf("deprovisioningHandler is not set; call SetDeprovisioningHandler before starting the manager")
	}
	return r.deprovisioningHandler.Deprovision(ctx, cr)
}

// ReconcileResourcesWithoutStatusChange satisfies deprovisioning.ResourceReconciler.
func (r *BtpOperatorReconciler) ReconcileResourcesWithoutStatusChange(ctx context.Context, cr *v1alpha1.BtpOperator) {
	r.provisioningHandler.ReconcileResourcesWithoutStatusChange(ctx, cr)
}

func (r *BtpOperatorReconciler) HandleReadyState(ctx context.Context, cr *v1alpha1.BtpOperator) error {
	logger := log.FromContext(ctx)
	logger.Info("Handling Ready state")

	requiredSecret, errWithReason := r.getAndVerifyRequiredSecret(ctx)
	if errWithReason != nil {
		return r.handleSecretVerificationFailure(ctx, cr, logger, errWithReason)
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
	if r.deprovisioningHandler == nil {
		return fmt.Errorf("deprovisioningHandler is not set; call SetDeprovisioningHandler before SetupWithManager")
	}
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

func (r *BtpOperatorReconciler) watchNetworkPolicyPredicates() predicate.Funcs {
	nameSet := make(map[string]struct{})
	if r.networkPolicyManager != nil {
		if policies, err := r.networkPolicyManager.LoadNetworkPolicies(); err == nil {
			for _, p := range policies {
				if n := p.GetName(); n != "" {
					nameSet[n] = struct{}{}
				}
			}
		}
	}
	isManaged := func(obj *networkingv1.NetworkPolicy) bool {
		labels := obj.GetLabels()
		if labels != nil && labels[managedByLabelKey] == operatorName && labels[kymaProjectModuleLabelKey] == moduleName {
			return true
		}
		_, ok := nameSet[obj.GetName()]
		return ok
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

func (r *BtpOperatorReconciler) isManagedSecret(s *corev1.Secret) bool {
	return r.isCredentialsSecret(s) || r.isCertSecret(s)
}

func (r *BtpOperatorReconciler) isCredentialsSecret(s *corev1.Secret) bool {
	return s.Namespace == config.ChartNamespace && s.Name == config.SecretName
}

func (r *BtpOperatorReconciler) isCertSecret(s *corev1.Secret) bool {
	return s.Namespace == config.ChartNamespace && (s.Name == caCertSecretName || s.Name == webhookCertSecretName)
}
