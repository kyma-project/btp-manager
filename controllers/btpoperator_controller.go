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
	"strings"
	"time"

	"github.com/kyma-project/btp-manager/api/v1alpha1"
	"github.com/kyma-project/btp-manager/controllers/config"
	"github.com/kyma-project/btp-manager/internal/certs"
	"github.com/kyma-project/btp-manager/internal/conditions"
	"github.com/kyma-project/btp-manager/internal/manifest"
	"github.com/kyma-project/btp-manager/internal/metrics"
	"github.com/kyma-project/btp-manager/internal/ymlutils"

	"github.com/go-logr/logr"
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
	KubeRbacProxyName         = "kube-rbac-proxy"
	KubeRbacProxyEnv          = "KUBE_RBAC_PROXY"
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

	kubeRbacProxyContainerName         = KubeRbacProxyName
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
	apiServerClient                                     client.Client
	Scheme                                              *runtime.Scheme
	manifestHandler                                     *manifest.Handler
	metrics                                             *metrics.Metrics
	instanceBindingService                              InstanceBindingSerivce
	workqueueSize                                       int
	previousCredentialsNamespace                        string
	clusterIdFromSapBtpManagerSecret                    string
	clusterIdFromSapBtpServiceOperatorConfigMap         string
	clusterIdFromSapBtpServiceOperatorClusterIdSecret   string
	credentialsNamespaceFromSapBtpManagerSecret         string
	credentialsNamespaceFromSapBtpServiceOperatorSecret string
	watchHandlers                                       []config.WatchHandler
}

type ResourceReadiness struct {
	Name      string
	Namespace string
	Kind      string
	Ready     bool
}

func NewBtpOperatorReconciler(client client.Client, apiServerClient client.Client, scheme *runtime.Scheme, instanceBindingSerivice InstanceBindingSerivce, metrics *metrics.Metrics, watchHandlers []config.WatchHandler) *BtpOperatorReconciler {
	return &BtpOperatorReconciler{
		Client:                 client,
		apiServerClient:        apiServerClient,
		Scheme:                 scheme,
		manifestHandler:        &manifest.Handler{Scheme: scheme},
		instanceBindingService: instanceBindingSerivice,
		metrics:                metrics,
		watchHandlers:          watchHandlers,
	}
}

// RBAC neccessary for the operator itself
//+kubebuilder:rbac:groups="operator.kyma-project.io",resources="btpoperators",verbs="*"
//+kubebuilder:rbac:groups="operator.kyma-project.io",resources="btpoperators/status",verbs="*"
//+kubebuilder:rbac:groups="services.cloud.sap.com",resources=serviceinstances;servicebindings,verbs="*"
//+kubebuilder:rbac:groups="",resources="namespaces",verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources="pods",verbs="*"

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

	r.setCredentialsNamespacesAndClusterId(requiredSecret)

	if errWithReason := r.checkDefaultCredentialsSecretNamespace(ctx, logger, requiredSecret); errWithReason != nil {
		return r.UpdateBtpOperatorStatus(ctx, cr, v1alpha1.StateError, errWithReason.reason, errWithReason.message)
	}

	if errWithReason := r.checkSapBtpServiceOperatorClusterIdConfigMap(ctx, logger, requiredSecret); errWithReason != nil {
		return r.UpdateBtpOperatorStatus(ctx, cr, v1alpha1.StateError, errWithReason.reason, errWithReason.message)
	}

	if errWithReason := r.checkSapBtpServiceOperatorClusterIdSecret(ctx, logger, requiredSecret); errWithReason != nil {
		return r.UpdateBtpOperatorStatus(ctx, cr, v1alpha1.StateError, errWithReason.reason, errWithReason.message)
	}

	if err := r.deleteOutdatedResources(ctx); err != nil {
		return r.UpdateBtpOperatorStatus(ctx, cr, v1alpha1.StateError, conditions.ProvisioningFailed, err.Error())
	}

	if err := r.reconcileResources(ctx, cr, requiredSecret); err != nil {
		return r.UpdateBtpOperatorStatus(ctx, cr, v1alpha1.StateError, conditions.ProvisioningFailed, err.Error())
	}

	if err := r.deleteResourcesIfBtpManagerSecretChanged(ctx); err != nil {
		logger.Error(err, "while deleting resources")
		return r.UpdateBtpOperatorStatus(ctx, cr, v1alpha1.StateError, conditions.ResourceRemovalFailed, err.Error())
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
	return fmt.Sprintf("%s%cdelete", config.ResourcesPath, os.PathSeparator)
}

func (r *BtpOperatorReconciler) getNetworkPoliciesPath() string {
	return fmt.Sprintf("%s%cnetwork-policies", config.ManagerResourcesPath, os.PathSeparator)
}

func (r *BtpOperatorReconciler) loadNetworkPolicies() ([]*unstructured.Unstructured, error) {
	return r.createUnstructuredObjectsFromManifestsDir(r.getNetworkPoliciesPath())
}

func (r *BtpOperatorReconciler) addNetworkPoliciesToResources(ctx context.Context, resourcesToApply *[]*unstructured.Unstructured) error {
	logger := log.FromContext(ctx)
	networkPolicies, err := r.loadNetworkPolicies()
	if err != nil {
		logger.Error(err, "while loading network policies")
		return fmt.Errorf("failed to load network policies: %w", err)
	}
	*resourcesToApply = append(*resourcesToApply, networkPolicies...)
	logger.Info(fmt.Sprintf("added %d network policies to resources to apply", len(networkPolicies)))

	return nil
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
		if err := r.cleanupNetworkPolicies(ctx); err != nil {
			logger.Error(err, "while cleaning up network policies")
			return fmt.Errorf("failed to cleanup network policies: %w", err)
		}
	} else {
		logger.Info("network policies enabled, loading and adding them to resources")
		if err := r.addNetworkPoliciesToResources(ctx, &resourcesToApply); err != nil {
			return err
		}
	}

	if err = r.prepareModuleResourcesFromManifests(ctx, resourcesToApply, s); err != nil {
		logger.Error(err, "while preparing objects to apply")
		return fmt.Errorf("failed to prepare objects to apply: %w", err)
	}

	if err = r.prepareAdmissionWebhooks(ctx, &resourcesToApply); err != nil {
		logger.Error(err, "while preparing admission webhooks")
		return fmt.Errorf("failed to prepare admission webhooks: %w", err)
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

func (r *BtpOperatorReconciler) restartSapBtpServiceOperatorPodIfNotReady(ctx context.Context, logger logr.Logger) error {
	pod, err := r.getSapBtpServiceOperatorPod(ctx)
	if err != nil {
		logger.Error(err, "while getting SAP BTP service operator pod")
		return err
	}
	if pod != nil {
		for _, condition := range pod.Status.Conditions {
			if condition.Type == corev1.PodReady && condition.Status == corev1.ConditionFalse {
				if err := r.deleteObject(ctx, pod); err != nil {
					logger.Error(err, fmt.Sprintf("while deleting not ready %s pod", pod.Name))
					return err
				}
			}
		}
	}

	return nil
}

func (r *BtpOperatorReconciler) getResourcesToApplyPath() string {
	return fmt.Sprintf("%s%capply", config.ResourcesPath, os.PathSeparator)
}

func (r *BtpOperatorReconciler) prepareModuleResourcesFromManifests(ctx context.Context, resourcesToApply []*unstructured.Unstructured, s *corev1.Secret) error {
	logger := log.FromContext(ctx)
	logger.Info("preparing module resources to apply")

	var configMapIndex, secretIndex, deploymentIndex int
	for i, u := range resourcesToApply {
		if u.GetName() == sapBtpServiceOperatorConfigMapName && u.GetKind() == configMapKind {
			configMapIndex = i
			continue
		}
		if u.GetName() == sapBtpServiceOperatorSecretName && u.GetKind() == secretKind {
			secretIndex = i
			continue
		}
		if u.GetName() == config.DeploymentName && u.GetKind() == deploymentKind {
			deploymentIndex = i
			continue
		}
	}

	chartVer, err := ymlutils.ExtractStringValueFromYamlForGivenKey(fmt.Sprintf("%s/Chart.yaml", config.ChartPath), "version")
	if err != nil {
		logger.Error(err, "while getting module chart version")
		return fmt.Errorf("failed to get module chart version: %w", err)
	}

	if err := r.addLabels(chartVer, resourcesToApply...); err != nil {
		logger.Error(err, "while adding labels to resources")
		return fmt.Errorf("failed to add labels to resources: %w", err)
	}
	r.setNamespace(resourcesToApply...)

	if err := r.setConfigMapValues(s, (resourcesToApply)[configMapIndex]); err != nil {
		logger.Error(err, "while setting ConfigMap values")
		return fmt.Errorf("failed to set ConfigMap values: %w", err)
	}
	if err := r.setSecretValues(s, (resourcesToApply)[secretIndex]); err != nil {
		logger.Error(err, "while setting Secret values")
		return fmt.Errorf("failed to set Secret values: %w", err)
	}
	if err := r.setDeploymentImages(resourcesToApply[deploymentIndex]); err != nil {
		logger.Error(err, "while setting container images in Deployment")
		return fmt.Errorf("failed to set container images in Deployment: %w", err)
	}

	return nil
}

func (r *BtpOperatorReconciler) cleanupNetworkPolicies(ctx context.Context) error {
	logger := log.FromContext(ctx)
	logger.Info("deleting all managed network policies")
	if err := r.DeleteAllOf(ctx, &networkingv1.NetworkPolicy{}, client.InNamespace(config.ChartNamespace), managedByLabelFilter); err != nil {
		if !(k8serrors.IsNotFound(err) || k8serrors.IsMethodNotSupported(err) || meta.IsNoMatchError(err)) {
			return fmt.Errorf("failed to delete network policies: %w", err)
		}
	}

	return nil
}

func (r *BtpOperatorReconciler) addLabels(chartVer string, us ...*unstructured.Unstructured) error {

	for _, u := range us {
		labels := u.GetLabels()
		if len(labels) == 0 {
			labels = make(map[string]string)
		}
		labels[managedByLabelKey] = operatorName
		labels[chartVersionLabelKey] = chartVer
		labels[kymaProjectModuleLabelKey] = moduleName
		u.SetLabels(labels)
		if u.GetKind() == deploymentKind {
			tplLabels, found, err := unstructured.NestedStringMap(u.Object, "spec", "template", "metadata", "labels")
			if err == nil {
				if !found || tplLabels == nil {
					tplLabels = make(map[string]string)
				}
				tplLabels[kymaProjectModuleLabelKey] = moduleName
				if err := unstructured.SetNestedStringMap(u.Object, tplLabels, "spec", "template", "metadata", "labels"); err != nil {
					return fmt.Errorf("failed to set pod template labels for deployment %s: %w", u.GetName(), err)
				}
			} else {
				return fmt.Errorf("failed to get pod template labels for deployment %s: %w", u.GetName(), err)
			}
		}
	}
	return nil
}

func (r *BtpOperatorReconciler) setNamespace(us ...*unstructured.Unstructured) {
	for _, u := range us {
		u.SetNamespace(config.ChartNamespace)
	}
}

func (r *BtpOperatorReconciler) deleteCreationTimestamp(us ...*unstructured.Unstructured) {
	for _, u := range us {
		unstructured.RemoveNestedField(u.Object, "metadata", "creationTimestamp")
	}
}

func (r *BtpOperatorReconciler) setConfigMapValues(secret *corev1.Secret, u *unstructured.Unstructured) error {
	if err := unstructured.SetNestedField(u.Object, string(secret.Data[ClusterIdSecretKey]), "data", ClusterIdConfigMapKey); err != nil {
		return err
	}

	if err := unstructured.SetNestedField(u.Object, r.credentialsNamespaceFromSapBtpManagerSecret, "data", ReleaseNamespaceConfigMapKey); err != nil {
		return err
	}

	if err := unstructured.SetNestedField(u.Object, r.credentialsNamespaceFromSapBtpManagerSecret, "data", ManagementNamespaceConfigMapKey); err != nil {
		return err
	}

	if err := unstructured.SetNestedField(u.Object, config.EnableLimitedCache, "data", EnableLimitedCacheConfigMapKey); err != nil {
		return err
	}

	return nil
}

func (r *BtpOperatorReconciler) setSecretValues(secret *corev1.Secret, u *unstructured.Unstructured) error {
	u.SetNamespace(r.credentialsNamespaceFromSapBtpManagerSecret)
	for k := range secret.Data {
		if k == ClusterIdSecretKey || k == CredentialsNamespaceSecretKey {
			continue
		}
		if err := unstructured.SetNestedField(u.Object, base64.StdEncoding.EncodeToString(secret.Data[k]), "data", k); err != nil {
			return err
		}
	}
	return nil
}

func (r *BtpOperatorReconciler) setDeploymentImages(u *unstructured.Unstructured) error {
	sapBtpServiceOperatorImage := os.Getenv(SapBtpServiceOperatorEnv)
	kubeRbacProxyImage := os.Getenv(KubeRbacProxyEnv)
	if err := r.setContainerImage(u, sapBtpServiceOperatorContainerName, sapBtpServiceOperatorImage); err != nil {
		return fmt.Errorf("failed to set container image for %s: %w", SapBtpServiceOperatorName, err)
	}
	if err := r.setContainerImage(u, kubeRbacProxyContainerName, kubeRbacProxyImage); err != nil {
		return fmt.Errorf("failed to set container image for %s: %w", kubeRbacProxyContainerName, err)
	}

	return nil
}

func (r *BtpOperatorReconciler) setContainerImage(u *unstructured.Unstructured, containerName, image string) error {
	containers, found, err := unstructured.NestedSlice(u.Object, "spec", "template", "spec", "containers")
	if err != nil {
		return fmt.Errorf("failed to get containers from %s %s: %w", u.GetKind(), u.GetName(), err)
	}
	if !found {
		return fmt.Errorf("containers not found in %s %s", u.GetKind(), u.GetName())
	}
	for i, c := range containers {
		container, ok := c.(map[string]interface{})
		if !ok {
			return fmt.Errorf("cannot cast container field to map[string]interface{}: %v", c)
		}
		if container["name"] == containerName {
			container["image"] = image
			containers[i] = container
			break
		}
	}

	return unstructured.SetNestedSlice(u.Object, containers, "spec", "template", "spec", "containers")
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
	resourcesReadinessInformer := make(chan ResourceReadiness, numOfResources)
	for _, u := range us {
		if u.GetKind() == deploymentKind {
			go r.checkDeploymentReadiness(ctx, u, resourcesReadinessInformer)
			continue
		}
		go r.checkResourceExistence(ctx, u, resourcesReadinessInformer)
	}

	for i := 0; i < numOfResources; i++ {
		if resourceReady := <-resourcesReadinessInformer; !resourceReady.Ready {
			return fmt.Errorf("%s %s in namespace %s readiness timeout reached", resourceReady.Kind, resourceReady.Name, resourceReady.Namespace)
		}
	}
	return nil
}

func (r *BtpOperatorReconciler) checkDeploymentReadiness(ctx context.Context, u *unstructured.Unstructured, c chan<- ResourceReadiness) {
	logger := log.FromContext(ctx)
	ctxWithTimeout, cancel := context.WithTimeout(ctx, config.ReadyCheckInterval)
	defer cancel()

	var err error
	var availableConditionStatus, progressingConditionStatus string
	got := &appsv1.Deployment{}
	now := time.Now()
	for {
		if time.Since(now) >= config.ReadyTimeout {
			logger.Error(err, fmt.Sprintf("timed out while checking %s %s readiness", u.GetName(), u.GetKind()))
			c <- ResourceReadiness{
				Name:      u.GetName(),
				Namespace: u.GetNamespace(),
				Kind:      u.GetKind(),
				Ready:     false,
			}
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
				c <- ResourceReadiness{Ready: true}
				return
			}
		}
	}
}

func (r *BtpOperatorReconciler) checkResourceExistence(ctx context.Context, u *unstructured.Unstructured, c chan<- ResourceReadiness) {
	logger := log.FromContext(ctx)
	ctxWithTimeout, cancel := context.WithTimeout(ctx, config.ReadyCheckInterval)
	defer cancel()

	var err error
	now := time.Now()
	got := &unstructured.Unstructured{}
	got.SetGroupVersionKind(u.GroupVersionKind())
	for {
		if time.Since(now) >= config.ReadyTimeout {
			logger.Error(err, fmt.Sprintf("timed out while checking %s %s existence", u.GetName(), u.GetKind()))
			c <- ResourceReadiness{
				Name:      u.GetName(),
				Namespace: u.GetNamespace(),
				Kind:      u.GetKind(),
				Ready:     false,
			}
			return
		}
		if err = r.Get(ctxWithTimeout, client.ObjectKey{Name: u.GetName(), Namespace: u.GetNamespace()}, got); err == nil {
			c <- ResourceReadiness{Ready: true}
			return
		}
	}
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

	r.setCredentialsNamespacesAndClusterId(requiredSecret)

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

	clusterIdSecret, err := r.getSecretByNameAndNamespace(ctx, sapBtpServiceOperatorClusterIdSecretName, r.credentialsNamespaceFromSapBtpManagerSecret)
	if err != nil {
		logger.Error(err, fmt.Sprintf("while getting %s secret in %s namespace", sapBtpServiceOperatorClusterIdSecretName, r.credentialsNamespaceFromSapBtpManagerSecret))
		return fmt.Errorf("failed to get cluster ID secret: %w", err)
	}
	if clusterIdSecret != nil {
		if err := r.deleteObject(ctx, clusterIdSecret); err != nil {
			logger.Error(err, fmt.Sprintf("while deleting %s secret from %s namespace", sapBtpServiceOperatorClusterIdSecretName, r.credentialsNamespaceFromSapBtpManagerSecret))
			return fmt.Errorf("failed to delete cluster ID secret: %w", err)
		}
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
	deployment := &appsv1.Deployment{}
	if err := r.Get(ctx, client.ObjectKey{Name: config.DeploymentName, Namespace: config.ChartNamespace}, deployment); err != nil {
		if !k8serrors.IsNotFound(err) {
			return err
		}
	} else {
		if err := r.Delete(ctx, deployment); client.IgnoreNotFound(err) != nil {
			return err
		}
	}

	mutatingWebhook := &admissionregistrationv1.MutatingWebhookConfiguration{}
	if err := r.Get(ctx, client.ObjectKey{Name: mutatingWebhookName, Namespace: config.ChartNamespace}, mutatingWebhook); err != nil {
		if !k8serrors.IsNotFound(err) {
			return err
		}
	} else {
		if err := r.Delete(ctx, mutatingWebhook); client.IgnoreNotFound(err) != nil {
			return err
		}
	}

	validatingWebhook := &admissionregistrationv1.ValidatingWebhookConfiguration{}
	if err := r.Get(ctx, client.ObjectKey{Name: validatingWebhookName, Namespace: config.ChartNamespace}, validatingWebhook); err != nil {
		if !k8serrors.IsNotFound(err) {
			return err
		}
	} else {
		if err := r.Delete(ctx, validatingWebhook); client.IgnoreNotFound(err) != nil {
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

	r.setCredentialsNamespacesAndClusterId(requiredSecret)

	defaultCredentialsSecret, err := r.getDefaultCredentialsSecret(ctx)
	if err != nil {
		logger.Error(err, fmt.Sprintf("while getting %s Secret", sapBtpServiceOperatorSecretName))
		return r.UpdateBtpOperatorStatus(ctx, cr, v1alpha1.StateError, conditions.GettingDefaultCredentialsSecretFailed, err.Error())
	}

	if defaultCredentialsSecret != nil {
		r.credentialsNamespaceFromSapBtpServiceOperatorSecret = defaultCredentialsSecret.Namespace
		if r.credentialsNamespaceFromSapBtpManagerSecret != r.credentialsNamespaceFromSapBtpServiceOperatorSecret {
			msg := fmt.Sprintf("credentials namespace changed from %s to %s", r.credentialsNamespaceFromSapBtpServiceOperatorSecret, r.credentialsNamespaceFromSapBtpManagerSecret)
			logger.Info(msg)
			return r.UpdateBtpOperatorStatus(ctx, cr, v1alpha1.StateProcessing, conditions.CredentialsNamespaceChanged, msg)
		}
	}

	sapBtpOperatorConfigMap, err := r.getSapBtpServiceOperatorConfigMap(ctx)
	if err != nil {
		logger.Error(err, fmt.Sprintf("while getting %s ConfigMap", sapBtpServiceOperatorConfigMapName))
		return r.UpdateBtpOperatorStatus(ctx, cr, v1alpha1.StateError, conditions.GettingSapBtpServiceOperatorConfigMapFailed, err.Error())
	}

	if sapBtpOperatorConfigMap != nil {
		r.clusterIdFromSapBtpServiceOperatorConfigMap = sapBtpOperatorConfigMap.Data[strings.ToUpper(ClusterIdSecretKey)]
		if r.clusterIdFromSapBtpManagerSecret != r.clusterIdFromSapBtpServiceOperatorConfigMap {
			msg := fmt.Sprintf("cluster ID changed from %s to %s", r.clusterIdFromSapBtpServiceOperatorConfigMap, r.credentialsNamespaceFromSapBtpManagerSecret)
			logger.Info(msg)
			return r.UpdateBtpOperatorStatus(ctx, cr, v1alpha1.StateProcessing, conditions.ClusterIdChanged, msg)
		}
	}

	if err := r.deleteOutdatedResources(ctx); err != nil {
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

func (r *BtpOperatorReconciler) watchDeploymentPredicates() predicate.Funcs {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			obj := e.Object.(*appsv1.Deployment)
			return obj.Name == config.DeploymentName && obj.Namespace == config.ChartNamespace
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			obj := e.Object.(*appsv1.Deployment)
			return obj.Name == config.DeploymentName && obj.Namespace == config.ChartNamespace
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			newObj := e.ObjectNew.(*appsv1.Deployment)
			oldObj := e.ObjectOld.(*appsv1.Deployment)
			if !(newObj.Name == config.DeploymentName && newObj.Namespace == config.ChartNamespace) {
				return false
			}
			var newAvailableConditionStatus, newProgressingConditionStatus string
			for _, condition := range newObj.Status.Conditions {
				if string(condition.Type) == deploymentProgressingConditionType {
					newProgressingConditionStatus = string(condition.Status)
				} else if string(condition.Type) == deploymentAvailableConditionType {
					newAvailableConditionStatus = string(condition.Status)
				}
			}
			var oldAvailableConditionStatus, oldProgressingConditionStatus string
			for _, condition := range oldObj.Status.Conditions {
				if string(condition.Type) == deploymentProgressingConditionType {
					oldProgressingConditionStatus = string(condition.Status)
				} else if string(condition.Type) == deploymentAvailableConditionType {
					oldAvailableConditionStatus = string(condition.Status)
				}
			}
			return newAvailableConditionStatus != oldAvailableConditionStatus || newProgressingConditionStatus != oldProgressingConditionStatus
		},
	}
}

func (r *BtpOperatorReconciler) getNetworkPolicyNamesFromManifests() (map[string]struct{}, error) {
	names := make(map[string]struct{})
	us, err := r.loadNetworkPolicies()
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

// *[]*unstructured.Unstructured is required because we extend the slice during certificates regeneration adding secrets and webhook configurations,
// so the result of the function execution is in resourcesToApply slice
func (r *BtpOperatorReconciler) prepareAdmissionWebhooks(ctx context.Context, resourcesToApply *[]*unstructured.Unstructured) error {
	logger := log.FromContext(ctx)
	logger.Info("preparing admission webhooks")

	logger.Info("checking CA certificate")
	caCertSecret, err := r.getSecretByNameAndNamespace(ctx, caCertSecretName, config.ChartNamespace)
	if err != nil {
		return err
	}
	if caCertSecret == nil {
		logger.Info("CA cert secret does not exist")
		return r.regenerateCertificates(ctx, resourcesToApply)
	}
	if err := r.validateCert(caCertSecret); err != nil {
		logger.Info(fmt.Sprintf("CA cert is not valid: %s", err))
		return r.regenerateCertificates(ctx, resourcesToApply)
	}

	caBundle := caCertSecret.Data[caCertSecretCertField]

	logger.Info("checking webhook certificate")
	webhookCertSecret, err := r.getSecretByNameAndNamespace(ctx, webhookCertSecretName, config.ChartNamespace)
	if err != nil {
		return err
	}
	if webhookCertSecret == nil {
		logger.Info("webhook cert secret does not exist")
		return r.regenerateWebhookCertificate(ctx, resourcesToApply, caCertSecret.Data)
	}
	if err := r.validateWebhookCert(webhookCertSecret, caBundle); err != nil {
		logger.Info(fmt.Sprintf("webhook cert is not valid: %s", err))
		var certSignErr CertificateSignError
		if errors.As(err, &certSignErr) {
			return r.regenerateCertificates(ctx, resourcesToApply)
		}
		return r.regenerateWebhookCertificate(ctx, resourcesToApply, caCertSecret.Data)
	}

	logger.Info("certificates for admission webhooks are valid")
	return r.prepareWebhooksManifests(ctx, resourcesToApply, caBundle)
}

func (r *BtpOperatorReconciler) regenerateCertificates(ctx context.Context, resourcesToApply *[]*unstructured.Unstructured) error {
	logger := log.FromContext(ctx)
	logger.Info("regenerating CA and webhook certificates")

	caCertificate, caPrivateKey, err := r.generateSelfSignedCert(ctx)
	if err != nil {
		return fmt.Errorf("while generating CA self signed cert: %w", err)
	}

	logger.Info("adding secret with regenerated CA self signed cert to resources to apply")
	err = r.appendCertificateSecretToUnstructured(caCertSecretName, caCertificate, caPrivateKey, resourcesToApply)
	if err != nil {
		return fmt.Errorf("while adding secret with regenerated CA self signed cert to resources to apply: %w", err)
	}

	webhookCertificate, webhookPrivateKey, err := r.generateSignedCert(ctx, caCertificate, caPrivateKey)
	if err != nil {
		return fmt.Errorf("while generating webhook signed cert: %w", err)
	}

	logger.Info("adding secret with regenerated webhook signed cert to resources to apply")
	err = r.appendCertificateSecretToUnstructured(webhookCertSecretName, webhookCertificate, webhookPrivateKey, resourcesToApply)
	if err != nil {
		return fmt.Errorf("while adding regenerated webhook signed cert to resources to apply: %w", err)
	}

	if err = r.prepareWebhooksManifests(ctx, resourcesToApply, caCertificate); err != nil {
		return fmt.Errorf("while preparing webhooks manifests: %w", err)
	}

	logger.Info("certificates regeneration succeeded")
	r.metrics.IncreaseCertsRegenerationsCounter()

	return nil
}

func (r *BtpOperatorReconciler) regenerateWebhookCertificate(ctx context.Context, resourcesToApply *[]*unstructured.Unstructured, caCertSecretData map[string][]byte) error {
	logger := log.FromContext(ctx)
	logger.Info("regenerating webhook certificate")

	webhookCertificate, webhookPrivateKey, err := r.generateSignedCert(ctx, caCertSecretData[caCertSecretCertField], caCertSecretData[caCertSecretKeyField])
	if err != nil {
		return fmt.Errorf("while regenerating webhook signed cert: %w", err)
	}

	logger.Info("adding secret with regenerated webhook signed cert to resources to apply")
	err = r.appendCertificateSecretToUnstructured(webhookCertSecretName, webhookCertificate, webhookPrivateKey, resourcesToApply)
	if err != nil {
		return fmt.Errorf("while adding regenerated webhook signed cert to resources to apply: %w", err)
	}

	if err = r.prepareWebhooksManifests(ctx, resourcesToApply, caCertSecretData[caCertSecretKeyField]); err != nil {
		return err
	}

	logger.Info("webhook certificate regeneration succeeded")
	r.metrics.IncreaseCertsRegenerationsCounter()

	return nil
}

func (r *BtpOperatorReconciler) generateSelfSignedCert(ctx context.Context) ([]byte, []byte, error) {
	logger := log.FromContext(ctx)
	logger.Info("generating self signed cert")

	caCertificate, caPrivateKey, err := certs.GenerateSelfSignedCertificate(time.Now().UTC().Add(config.CaCertificateExpiration))
	if err != nil {
		return nil, nil, fmt.Errorf("while generating self signed cert: %w", err)
	}

	logger.Info("generation of self signed cert succeeded")
	return caCertificate, caPrivateKey, nil
}

func (r *BtpOperatorReconciler) generateSignedCert(ctx context.Context, caCert, caPrivateKey []byte) ([]byte, []byte, error) {
	logger := log.FromContext(ctx)
	logger.Info("generating webhook signed cert")

	webhookCertificate, webhookPrivateKey, err := certs.GenerateSignedCertificate(time.Now().UTC().Add(config.WebhookCertificateExpiration), caCert, caPrivateKey)
	if err != nil {
		return nil, nil, fmt.Errorf("while generating webhook signed cert: %w", err)
	}

	logger.Info("generation of webhook signed cert succeeded")
	return webhookCertificate, webhookPrivateKey, nil
}

func (r *BtpOperatorReconciler) appendCertificateSecretToUnstructured(secretName string, certificate, privateKey []byte, resourcesToApply *[]*unstructured.Unstructured) error {
	certFieldName, err := certFieldFromSecretBySecretName(secretName)
	if err != nil {
		return err
	}
	privateKeyFieldName, err := privateKeyFieldFromSecretBySecretName(secretName)
	if err != nil {
		return err
	}

	data := r.mapCertToSecretData(certificate, privateKey, certFieldName, privateKeyFieldName)
	secret := r.buildSecretWithData(secretName, data)

	unstructuredObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(secret)
	if err != nil {
		return err
	}
	*resourcesToApply = append(*resourcesToApply, &unstructured.Unstructured{Object: unstructuredObj})
	return nil
}

func (r *BtpOperatorReconciler) mapCertToSecretData(certificate, privateKey []byte, certFieldName, privateKeyFieldName string) map[string][]byte {
	return map[string][]byte{
		certFieldName:       certificate,
		privateKeyFieldName: privateKey,
	}
}

func (r *BtpOperatorReconciler) prepareWebhooksManifests(ctx context.Context, resourcesToApply *[]*unstructured.Unstructured, caBundle []byte) error {
	logger := log.FromContext(ctx)
	logger.Info("preparing webhooks manifests")

	for i, resource := range *resourcesToApply {
		if isResourceAdmissionWebhook(resource.GetKind()) {
			webhookManifest, err := r.prepareWebhookManifest(ctx, resource, caBundle)
			if err != nil {
				return err
			}
			(*resourcesToApply)[i] = webhookManifest
		}
	}

	logger.Info("webhooks manifests have been prepared successfully")
	return nil
}

func isResourceAdmissionWebhook(resourceKind string) bool {
	return resourceKind == mutatingWebhookConfigurationKind || resourceKind == validatingWebhookConfigurationKind
}

func (r *BtpOperatorReconciler) prepareWebhookManifest(ctx context.Context, webhookManifest *unstructured.Unstructured, caBundle []byte) (*unstructured.Unstructured, error) {
	const (
		WebhooksKey     = "webhooks"
		ClientConfigKey = "clientConfig"
		CaBundleKey     = "caBundle"
	)
	webhookManifestCopy := webhookManifest.DeepCopy()

	logger := log.FromContext(ctx)
	logger.Info(fmt.Sprintf("setting CA bundle in %s %s", webhookManifestCopy.GetName(), webhookManifestCopy.GetKind()))

	webhooks, exists, err := unstructured.NestedSlice(webhookManifestCopy.Object, WebhooksKey)
	if err != nil {
		return nil, fmt.Errorf("while getting webhooks array from the webhook manifest: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("webhooks array does not exist in the webhook manifest")
	}
	webhookManifestCopy.SetManagedFields(nil)

	for i, webhook := range webhooks {
		genericWebhook, ok := webhook.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("while casting webhook object to map[string]interface{}")
		}
		genericClientConfig, ok := genericWebhook[ClientConfigKey].(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("while casting webhook.clientConfig object to map[string]interface{}")
		}
		genericClientConfig[CaBundleKey] = caBundle
		genericWebhook[ClientConfigKey] = genericClientConfig
		webhooks[i] = genericWebhook
	}
	webhookManifestCopy.Object[WebhooksKey] = webhooks

	logger.Info("CA bundle has been set successfully")
	return webhookManifestCopy, nil
}

func (r *BtpOperatorReconciler) isWebhookCertSignedBySelfSignedCa(ctx context.Context) (bool, error) {
	caCertificate, err := r.getCertificateFromSecret(ctx, caCertSecretName)
	if err != nil {
		return false, err
	}

	webhookCertificate, err := r.getCertificateFromSecret(ctx, webhookCertSecretName)
	if err != nil {
		return false, err
	}

	ok, err := certs.VerifyIfLeafIsSignedByGivenCA(caCertificate, webhookCertificate)
	if err != nil {
		return false, err
	}

	return ok, nil
}

func (r *BtpOperatorReconciler) getDataFromSecret(ctx context.Context, name string) (map[string][]byte, error) {
	secret := &corev1.Secret{}
	if err := r.Get(ctx, client.ObjectKey{Namespace: config.ChartNamespace, Name: name}, secret); err != nil {
		return nil, err
	}
	return secret.Data, nil
}

func (r *BtpOperatorReconciler) getCertificateFromSecret(ctx context.Context, secretName string) ([]byte, error) {
	data, err := r.getDataFromSecret(ctx, secretName)
	if err != nil {
		return nil, err
	}
	key, err := certFieldFromSecretBySecretName(secretName)
	if err != nil {
		return nil, err
	}
	cert, err := r.getSecretDataValueByKey(key, data)
	if err != nil {
		return nil, err
	}
	return cert, nil
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

func (r *BtpOperatorReconciler) buildSecretWithData(name string, data map[string][]byte) *corev1.Secret {
	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       secretKind,
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: config.ChartNamespace,
			Labels: map[string]string{
				managedByLabelKey: operatorName,
			},
		},
		Data: data,
	}
}

func (r *BtpOperatorReconciler) reconcileResourcesWithoutChangingCrState(ctx context.Context, cr *v1alpha1.BtpOperator, logger *logr.Logger) {
	secret, errWithReason := r.getAndVerifyRequiredSecret(ctx)
	if errWithReason != nil {
		logger.Error(errWithReason, "secret verification failed")
	}
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

func (r *BtpOperatorReconciler) setCredentialsNamespacesAndClusterId(s *corev1.Secret) {
	credentialsNamespace := config.ChartNamespace
	if s != nil {
		if v, ok := s.Data[CredentialsNamespaceSecretKey]; ok && len(v) > 0 {
			credentialsNamespace = string(v)
		}
		r.clusterIdFromSapBtpManagerSecret = string(s.Data[ClusterIdSecretKey])
		r.previousCredentialsNamespace = s.Annotations[previousCredentialsNamespaceAnnotationKey]
	}
	r.credentialsNamespaceFromSapBtpManagerSecret = credentialsNamespace
	r.credentialsNamespaceFromSapBtpServiceOperatorSecret = credentialsNamespace
}

func (r *BtpOperatorReconciler) checkDefaultCredentialsSecretNamespace(ctx context.Context, logger logr.Logger, requiredSecret *corev1.Secret) *ErrorWithReason {
	defaultCredentialsSecret, err := r.getDefaultCredentialsSecret(ctx)
	if err != nil {
		logger.Error(err, fmt.Sprintf("while getting %s secret", sapBtpServiceOperatorSecretName))
		return NewErrorWithReason(conditions.GettingDefaultCredentialsSecretFailed, err.Error())
	}

	if defaultCredentialsSecret != nil {
		r.credentialsNamespaceFromSapBtpServiceOperatorSecret = defaultCredentialsSecret.Namespace
		if r.credentialsNamespaceFromSapBtpManagerSecret != r.credentialsNamespaceFromSapBtpServiceOperatorSecret {
			logger.Info(fmt.Sprintf("credentials namespaces between %s secret and %s secret don't match", config.SecretName, sapBtpServiceOperatorSecretName))
			if err := r.annotateSecret(ctx, requiredSecret, previousCredentialsNamespaceAnnotationKey, r.credentialsNamespaceFromSapBtpServiceOperatorSecret); err != nil {
				return NewErrorWithReason(conditions.AnnotatingSecretFailed, err.Error())
			}
		}
	}

	return nil
}

func (r *BtpOperatorReconciler) checkSapBtpServiceOperatorClusterIdConfigMap(ctx context.Context, logger logr.Logger, requiredSecret *corev1.Secret) *ErrorWithReason {
	sapBtpOperatorConfigMap, err := r.getSapBtpServiceOperatorConfigMap(ctx)
	if err != nil {
		logger.Error(err, fmt.Sprintf("while getting %s ConfigMap", sapBtpServiceOperatorConfigMapName))
		return NewErrorWithReason(conditions.GettingSapBtpServiceOperatorConfigMapFailed, err.Error())
	}

	if sapBtpOperatorConfigMap != nil {
		r.clusterIdFromSapBtpServiceOperatorConfigMap = sapBtpOperatorConfigMap.Data[strings.ToUpper(ClusterIdSecretKey)]
		r.clusterIdFromSapBtpServiceOperatorClusterIdSecret = r.clusterIdFromSapBtpServiceOperatorConfigMap //default value in case of missing cluster ID secret
		if r.clusterIdFromSapBtpManagerSecret != r.clusterIdFromSapBtpServiceOperatorConfigMap {
			logger.Info(fmt.Sprintf("cluster IDs between %s secret and %s configmap don't match", config.SecretName, sapBtpServiceOperatorConfigMapName))
			if err := r.annotateSecret(ctx, requiredSecret, previousClusterIdAnnotationKey, r.clusterIdFromSapBtpServiceOperatorConfigMap); err != nil {
				return NewErrorWithReason(conditions.AnnotatingSecretFailed, err.Error())
			}
		}
	}

	return nil
}

func (r *BtpOperatorReconciler) checkSapBtpServiceOperatorClusterIdSecret(ctx context.Context, logger logr.Logger, requiredSecret *corev1.Secret) *ErrorWithReason {
	clusterIdSecret, err := r.getSecretByNameAndNamespace(ctx, sapBtpServiceOperatorClusterIdSecretName, r.credentialsNamespaceFromSapBtpServiceOperatorSecret)
	if err != nil {
		logger.Error(err, fmt.Sprintf("while getting %s secret", sapBtpServiceOperatorClusterIdSecretName))
		return NewErrorWithReason(conditions.GettingSapBtpServiceOperatorClusterIdSecretFailed, err.Error())
	}

	if clusterIdSecret != nil {
		if clusterIdFromSecret, ok := clusterIdSecret.Data[InitialClusterIdSecretKey]; ok && len(clusterIdFromSecret) > 0 {
			r.clusterIdFromSapBtpServiceOperatorClusterIdSecret = string(clusterIdFromSecret)
		}
		if r.clusterIdFromSapBtpServiceOperatorConfigMap != r.clusterIdFromSapBtpServiceOperatorClusterIdSecret {
			logger.Info(fmt.Sprintf("cluster IDs between %s configmap and %s secret don't match", sapBtpServiceOperatorConfigMapName, sapBtpServiceOperatorClusterIdSecretName))
			if err = r.annotateSecret(ctx, requiredSecret, previousClusterIdAnnotationKey, r.clusterIdFromSapBtpServiceOperatorClusterIdSecret); err != nil {
				logger.Error(err, fmt.Sprintf("while annotating %s secret", requiredSecret.Name))
				return NewErrorWithReason(conditions.AnnotatingSecretFailed, err.Error())
			}
			logger.Info(fmt.Sprintf("deleting %s secret from %s namespace due to invalid cluster ID", clusterIdSecret.Name, clusterIdSecret.Namespace))
			if err = r.deleteObject(ctx, clusterIdSecret); err != nil {
				logger.Error(err, fmt.Sprintf("while deleting %s secret", clusterIdSecret.Name))
				return NewErrorWithReason(conditions.DeletionOfOrphanedResourcesFailed, err.Error())
			}
			if err = r.restartSapBtpServiceOperatorPodIfNotReady(ctx, logger); err != nil {
				return NewErrorWithReason(conditions.ResourceRemovalFailed, fmt.Sprintf("while restarting SAP BTP service operator pod: %s", err))
			}
		}
	}

	return nil
}

func (r *BtpOperatorReconciler) getDefaultCredentialsSecret(ctx context.Context) (*corev1.Secret, error) {
	var defaultCredentialsSecret *corev1.Secret
	secrets := &corev1.SecretList{}
	if err := r.List(ctx, secrets, client.MatchingLabels{managedByLabelKey: operatorName}); err != nil {
		return nil, fmt.Errorf("unable to list managed secrets: %w", err)
	}
	if len(secrets.Items) == 0 {
		return nil, nil
	}
	for i, s := range secrets.Items {
		if s.Name == sapBtpServiceOperatorSecretName && s.Namespace == r.previousCredentialsNamespace {
			defaultCredentialsSecret = &secrets.Items[i]
			break
		} else if s.Name == sapBtpServiceOperatorSecretName {
			defaultCredentialsSecret = &secrets.Items[i]
		}
	}
	return defaultCredentialsSecret, nil
}

func (r *BtpOperatorReconciler) annotateSecret(ctx context.Context, s *corev1.Secret, key, value string) error {
	annotations := s.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}
	if annotations[key] == value {
		return nil
	}
	annotations[key] = value
	s.SetAnnotations(annotations)
	return r.Update(ctx, s, client.FieldOwner(operatorName))
}

func (r *BtpOperatorReconciler) getSapBtpServiceOperatorConfigMap(ctx context.Context) (*corev1.ConfigMap, error) {
	cm := &corev1.ConfigMap{}
	if err := r.Get(ctx, client.ObjectKey{Namespace: config.ChartNamespace, Name: sapBtpServiceOperatorConfigMapName}, cm); err != nil {
		if k8serrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return cm, nil
}

func (r *BtpOperatorReconciler) deleteResourcesIfBtpManagerSecretChanged(ctx context.Context) error {
	clusterIdSecret, err := r.getSecretByNameAndNamespace(ctx, sapBtpServiceOperatorClusterIdSecretName, r.credentialsNamespaceFromSapBtpServiceOperatorSecret)
	if err != nil {
		return err
	}
	pod, err := r.getSapBtpServiceOperatorPod(ctx)
	if err != nil {
		return err
	}
	credentialsSecret, err := r.getSecretByNameAndNamespace(ctx, sapBtpServiceOperatorSecretName, r.credentialsNamespaceFromSapBtpServiceOperatorSecret)
	if err != nil {
		return err
	}

	isCredentialsNamespaceChanged := r.credentialsNamespaceFromSapBtpServiceOperatorSecret != "" &&
		r.credentialsNamespaceFromSapBtpManagerSecret != r.credentialsNamespaceFromSapBtpServiceOperatorSecret

	isClusterIdChanged := r.clusterIdFromSapBtpServiceOperatorConfigMap != "" &&
		(r.clusterIdFromSapBtpManagerSecret != r.clusterIdFromSapBtpServiceOperatorConfigMap ||
			r.clusterIdFromSapBtpServiceOperatorConfigMap != r.clusterIdFromSapBtpServiceOperatorClusterIdSecret)

	if isCredentialsNamespaceChanged || isClusterIdChanged {
		if clusterIdSecret != nil {
			if err = r.deleteObject(ctx, clusterIdSecret); err != nil {
				return err
			}
		}
		if pod != nil {
			if err = r.deleteObject(ctx, pod); err != nil {
				return err
			}
		}
	}

	if isCredentialsNamespaceChanged && credentialsSecret != nil {
		if err = r.deleteObject(ctx, credentialsSecret); err != nil {
			return err
		}
	}

	return nil
}

func (r *BtpOperatorReconciler) deleteObject(ctx context.Context, obj client.Object) error {
	if err := r.apiServerClient.Delete(ctx, obj); err != nil {
		return fmt.Errorf("while deleting %s %s from %s namespace: %w", obj.GetName(), obj.GetObjectKind().GroupVersionKind().Kind, obj.GetNamespace(), err)
	}
	return nil
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

func (r *BtpOperatorReconciler) getSapBtpServiceOperatorPod(ctx context.Context) (*corev1.Pod, error) {
	var pod *corev1.Pod
	pods := &corev1.PodList{}
	if err := r.apiServerClient.List(ctx, pods, client.MatchingLabels{instanceLabelKey: operandName}); err != nil {
		return nil, fmt.Errorf("unable to list SAP BTP service operator pods: %w", err)
	}
	if len(pods.Items) == 0 {
		return nil, nil
	}
	for i, p := range pods.Items {
		if strings.HasPrefix(p.Name, operandName) && p.Namespace == config.ChartNamespace {
			pod = &pods.Items[i]
			break
		}
	}
	return pod, nil
}

func (r *BtpOperatorReconciler) validateCert(secret *corev1.Secret) error {
	certFieldName, err := certFieldFromSecretBySecretName(secret.GetName())
	if err != nil {
		return err
	}
	encodedCert, err := r.getSecretDataValueByKey(certFieldName, secret.Data)
	if err != nil {
		return err
	}
	privateKeyFieldName, err := privateKeyFieldFromSecretBySecretName(secret.GetName())
	if err != nil {
		return err
	}
	_, err = r.getSecretDataValueByKey(privateKeyFieldName, secret.Data)
	if err != nil {
		return err
	}
	block, err := certs.DecodeCertificate(encodedCert)
	if err != nil {
		return err
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return err
	}
	if certs.CertificateExpires(cert, config.ExpirationBoundary) {
		return fmt.Errorf("CA cert expires soon")
	}

	return nil
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

func privateKeyFieldFromSecretBySecretName(secretName string) (string, error) {
	switch secretName {
	case caCertSecretName:
		return caCertSecretKeyField, nil
	case webhookCertSecretName:
		return webhookCertSecretKeyField, nil
	}
	return "", fmt.Errorf("unknown secret %q - private key field undefined", secretName)
}

func (r *BtpOperatorReconciler) validateWebhookCert(webhookCertSecret *corev1.Secret, caCert []byte) error {
	if err := r.validateCert(webhookCertSecret); err != nil {
		return err
	}
	return r.verifyCASign(caCert, webhookCertSecret.Data[webhookCertSecretCertField])
}

func (r *BtpOperatorReconciler) verifyCASign(caCert []byte, signedCert []byte) error {
	ok, err := certs.VerifyIfLeafIsSignedByGivenCA(caCert, signedCert)
	if err != nil || !ok {
		return NewCertificateSignError(err.Error())
	}
	return nil
}
