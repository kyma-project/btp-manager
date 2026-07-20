package provisioning

import (
	"context"
	"fmt"
	"strings"

	"github.com/kyma-project/btp-manager/api/v1alpha1"
	"github.com/kyma-project/btp-manager/controllers/config"
	"github.com/kyma-project/btp-manager/internal/conditions"
	"github.com/kyma-project/btp-manager/internal/credentials/drift"
	"github.com/kyma-project/btp-manager/internal/k8s/networkpolicy"
	"github.com/kyma-project/btp-manager/internal/manager/moduleresource"
	"github.com/kyma-project/btp-manager/internal/webhook/certificate"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// InstanceBindingService controls the ServiceInstance/ServiceBinding cleanup controller lifecycle.
type InstanceBindingService interface {
	EnableSISBController()
}

// ProvisionResult communicates the outcome of Provision to the caller.
// Both fields nil means success.
type ProvisionResult struct {
	WarningReason *conditions.ErrorWithReason
	ErrorReason   *conditions.ErrorWithReason
}

// Handler runs the provisioning flow and exposes helpers used by the Ready state path.
type Handler interface {
	Provision(ctx context.Context, cr *v1alpha1.BtpOperator) ProvisionResult
	GetAndVerifyRequiredSecret(ctx context.Context) (*corev1.Secret, *conditions.ErrorWithReason)
	ReconcileResources(ctx context.Context, cr *v1alpha1.BtpOperator, secret *corev1.Secret) error
	ReconcileResourcesWithoutStatusChange(ctx context.Context, cr *v1alpha1.BtpOperator)
}

type handler struct {
	client                 client.Client
	driftDetector          drift.Detector
	moduleResourceManager  moduleresource.ResourceManager
	networkPolicyManager   networkpolicy.NetworkPolicyManager
	certManager            certificate.CertificateManager
	instanceBindingService InstanceBindingService
}

func NewHandler(
	c client.Client,
	driftDetector drift.Detector,
	moduleResourceManager moduleresource.ResourceManager,
	networkPolicyManager networkpolicy.NetworkPolicyManager,
	certManager certificate.CertificateManager,
	instanceBindingService InstanceBindingService,
) Handler {
	return &handler{
		client:                 c,
		driftDetector:          driftDetector,
		moduleResourceManager:  moduleResourceManager,
		networkPolicyManager:   networkPolicyManager,
		certManager:            certManager,
		instanceBindingService: instanceBindingService,
	}
}

var _ Handler = (*handler)(nil)

func (h *handler) Provision(ctx context.Context, cr *v1alpha1.BtpOperator) ProvisionResult {
	logger := log.FromContext(ctx)

	requiredSecret, errWithReason := h.GetAndVerifyRequiredSecret(ctx)
	if errWithReason != nil {
		logger.Info("secret verification failed: " + errWithReason.Error())
		if errWithReason.Reason == conditions.InvalidSecret {
			return ProvisionResult{ErrorReason: errWithReason}
		}
		return ProvisionResult{WarningReason: errWithReason}
	}

	h.driftDetector.InitializeFromSecret(requiredSecret)

	if errWithReason := h.driftDetector.CheckCredentialsNamespaceDrift(ctx, requiredSecret); errWithReason != nil {
		return ProvisionResult{ErrorReason: errWithReason}
	}

	if errWithReason := h.driftDetector.CheckClusterIdConfigMapDrift(ctx, requiredSecret); errWithReason != nil {
		return ProvisionResult{ErrorReason: errWithReason}
	}

	if errWithReason := h.driftDetector.ResolveClusterIdSecretDrift(ctx, requiredSecret); errWithReason != nil {
		return ProvisionResult{ErrorReason: errWithReason}
	}

	if err := h.moduleResourceManager.DeleteOutdatedResources(ctx); err != nil {
		return ProvisionResult{ErrorReason: conditions.NewErrorWithReason(conditions.ProvisioningFailed, err.Error())}
	}

	if err := h.ReconcileResources(ctx, cr, requiredSecret); err != nil {
		return ProvisionResult{ErrorReason: conditions.NewErrorWithReason(conditions.ProvisioningFailed, err.Error())}
	}

	if err := h.driftDetector.DeleteChangedResources(ctx); err != nil {
		logger.Error(err, "while deleting resources")
		return ProvisionResult{ErrorReason: conditions.NewErrorWithReason(conditions.ResourceRemovalFailed, err.Error())}
	}

	h.instanceBindingService.EnableSISBController()
	logger.Info("provisioning succeeded")
	return ProvisionResult{}
}

func (h *handler) GetAndVerifyRequiredSecret(ctx context.Context) (*corev1.Secret, *conditions.ErrorWithReason) {
	logger := log.FromContext(ctx)

	logger.Info("getting the required Secret")
	secret, err := h.getRequiredSecret(ctx)
	if err != nil {
		logger.Error(err, "while getting the required Secret")
		return nil, conditions.NewErrorWithReason(conditions.MissingSecret, "Secret resource not found")
	}

	logger.Info("verifying the required Secret")
	if err = verifySecret(secret); err != nil {
		logger.Error(err, "while verifying the required Secret")
		return nil, conditions.NewErrorWithReason(conditions.InvalidSecret, "Secret validation failed")
	}
	return secret, nil
}

func (h *handler) getRequiredSecret(ctx context.Context) (*corev1.Secret, error) {
	secret := &corev1.Secret{}
	objKey := client.ObjectKey{Namespace: config.ChartNamespace, Name: config.SecretName}
	if err := h.client.Get(ctx, objKey, secret); err != nil {
		if k8serrors.IsNotFound(err) {
			return nil, fmt.Errorf("%s Secret in %s namespace not found", config.SecretName, config.ChartNamespace)
		}
		return nil, fmt.Errorf("unable to get Secret: %w", err)
	}
	return secret, nil
}

func (h *handler) ReconcileResources(ctx context.Context, cr *v1alpha1.BtpOperator, s *corev1.Secret) error {
	logger := log.FromContext(ctx)

	logger.Info("getting module resources to apply")
	resourcesToApply, err := h.moduleResourceManager.CreateUnstructuredObjectsFromManifestsDir(h.moduleResourceManager.GetResourcesToApplyPath())
	if err != nil {
		logger.Error(err, "while creating applicable objects from manifests")
		return fmt.Errorf("failed to create applicable objects from manifests: %w", err)
	}
	logger.Info(fmt.Sprintf("got %d module resources to apply based on %s directory", len(resourcesToApply), h.moduleResourceManager.GetResourcesToApplyPath()))

	if cr.IsNetworkPoliciesDisabled() {
		logger.Info("network policies disabled, cleaning up existing ones")
		if err := h.cleanupNetworkPolicies(ctx); err != nil {
			logger.Error(err, "while cleaning up network policies")
			return fmt.Errorf("failed to cleanup network policies: %w", err)
		}
	} else {
		logger.Info("network policies enabled, loading and adding them to resources")
		if err := h.addNetworkPoliciesToResources(ctx, &resourcesToApply); err != nil {
			return err
		}
	}

	if err := h.deleteOldWebhookNetworkPolicy(ctx); err != nil {
		logger.Error(err, "while deleting old webhook network policy")
		return fmt.Errorf("failed to delete old webhook network policy: %w", err)
	}

	if err = h.moduleResourceManager.PrepareModuleResources(ctx, resourcesToApply, s); err != nil {
		logger.Error(err, "while preparing objects to apply")
		return fmt.Errorf("failed to prepare objects to apply: %w", err)
	}

	webhookResources, nonWebhookResources := certificate.PartitionWebhooks(resourcesToApply)
	preparedWebhooks, err := h.certManager.PrepareAdmissionWebhooks(ctx, webhookResources)
	if err != nil {
		logger.Error(err, "while preparing admission webhooks")
		return fmt.Errorf("failed to prepare admission webhooks: %w", err)
	}
	resourcesToApply = append(nonWebhookResources, preparedWebhooks...)

	h.moduleResourceManager.DeleteCreationTimestamp(resourcesToApply...)

	logger.Info(fmt.Sprintf("applying module resources for %d resources", len(resourcesToApply)))
	if err = h.moduleResourceManager.ApplyOrUpdateResources(ctx, resourcesToApply); err != nil {
		logger.Error(err, "while applying module resources")
		return fmt.Errorf("failed to apply module resources: %w", err)
	}

	logger.Info("waiting for module resources readiness")
	if err = h.moduleResourceManager.WaitForResourcesReadiness(ctx, resourcesToApply); err != nil {
		logger.Error(err, "while waiting for module resources readiness")
		return fmt.Errorf("timed out while waiting for resources readiness: %w", err)
	}

	return nil
}

func (h *handler) ReconcileResourcesWithoutStatusChange(ctx context.Context, cr *v1alpha1.BtpOperator) {
	logger := log.FromContext(ctx)
	secret, errWithReason := h.GetAndVerifyRequiredSecret(ctx)
	if errWithReason != nil {
		logger.Error(errWithReason, "secret verification failed")
		return
	}
	if err := h.moduleResourceManager.DeleteOutdatedResources(ctx); err != nil {
		logger.Error(err, "outdated resources deletion failed")
	}
	if err := h.ReconcileResources(ctx, cr, secret); err != nil {
		logger.Error(err, "resources reconciliation failed")
	}
}

func (h *handler) addNetworkPoliciesToResources(ctx context.Context, resourcesToApply *[]*unstructured.Unstructured) error {
	if h.networkPolicyManager == nil {
		return nil
	}
	logger := log.FromContext(ctx)
	networkPolicies, err := h.networkPolicyManager.LoadNetworkPolicies()
	if err != nil {
		logger.Error(err, "while loading network policies")
		return fmt.Errorf("failed to load network policies: %w", err)
	}
	*resourcesToApply = append(*resourcesToApply, networkPolicies...)
	logger.Info(fmt.Sprintf("added %d network policies to resources to apply", len(networkPolicies)))
	return nil
}

func (h *handler) cleanupNetworkPolicies(ctx context.Context) error {
	if h.networkPolicyManager == nil {
		return nil
	}
	return h.networkPolicyManager.CleanupNetworkPolicies(ctx)
}

func (h *handler) deleteOldWebhookNetworkPolicy(ctx context.Context) error {
	if h.networkPolicyManager == nil {
		return nil
	}
	return h.networkPolicyManager.DeleteOldWebhookNetworkPolicy(ctx)
}

func verifySecret(secret *corev1.Secret) error {
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
		errs = append(errs, fmt.Sprintf("key(s) %s not found", strings.Join(missingKeys, ", ")))
	}
	if len(missingValues) > 0 {
		errs = append(errs, fmt.Sprintf("missing value(s) for %s key(s)", strings.Join(missingValues, ", ")))
	}
	if len(errs) > 0 {
		return fmt.Errorf("%s", strings.Join(errs, ", "))
	}
	return nil
}
