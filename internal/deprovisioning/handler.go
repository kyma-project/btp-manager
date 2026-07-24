package deprovisioning

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/kyma-project/btp-manager/api/v1alpha1"
	"github.com/kyma-project/btp-manager/controllers/config"
	"github.com/kyma-project/btp-manager/internal/conditions"
	"github.com/kyma-project/btp-manager/internal/credentials/drift"
	"github.com/kyma-project/btp-manager/internal/k8s/networkpolicy"
	"github.com/kyma-project/btp-manager/internal/manager/moduleresource"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlutil "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	btpOperatorGroup           = "services.cloud.sap.com"
	btpOperatorApiVer          = "v1"
	btpOperatorServiceBinding  = "ServiceBinding"
	btpOperatorServiceInstance = "ServiceInstance"

	operatorName = "btp-manager"
	operandName  = "sap-btp-operator"

	mutatingWebhookName   = operandName + "-mutating-webhook-configuration"
	validatingWebhookName = operandName + "-validating-webhook-configuration"

	managedByLabelKey   = "app.kubernetes.io/managed-by"
	forceDeleteLabelKey = "force-delete"

	operatorLabelPrefix = "operator.kyma-project.io/"
	deletionFinalizer   = operatorLabelPrefix + operatorName
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

// StatusUpdater allows the handler to update the BtpOperator CR status.
type StatusUpdater interface {
	UpdateBtpOperatorStatus(ctx context.Context, cr *v1alpha1.BtpOperator, newState v1alpha1.State, reason conditions.Reason, message string) error
}

// ResourceReconciler is used to restore resources when deprovisioning fails.
type ResourceReconciler interface {
	ReconcileResourcesWithoutStatusChange(ctx context.Context, cr *v1alpha1.BtpOperator)
}

// InstanceBindingService controls the ServiceInstance/ServiceBinding cleanup controller lifecycle.
type InstanceBindingService interface {
	DisableSISBController()
}

// Handler runs the deprovisioning flow.
type Handler interface {
	Deprovision(ctx context.Context, cr *v1alpha1.BtpOperator) error
}

type handler struct {
	client                 client.Client
	apiServerClient        client.Client
	statusUpdater          StatusUpdater
	resourceReconciler     ResourceReconciler
	instanceBindingService InstanceBindingService
	driftDetector          drift.Detector
	moduleResourceManager  moduleresource.ResourceManager
	networkPolicyManager   networkpolicy.NetworkPolicyManager
}

func NewHandler(
	c client.Client,
	apiServerClient client.Client,
	statusUpdater StatusUpdater,
	resourceReconciler ResourceReconciler,
	instanceBindingService InstanceBindingService,
	driftDetector drift.Detector,
	moduleResourceManager moduleresource.ResourceManager,
	networkPolicyManager networkpolicy.NetworkPolicyManager,
) Handler {
	return &handler{
		client:                 c,
		apiServerClient:        apiServerClient,
		statusUpdater:          statusUpdater,
		resourceReconciler:     resourceReconciler,
		instanceBindingService: instanceBindingService,
		driftDetector:          driftDetector,
		moduleResourceManager:  moduleResourceManager,
		networkPolicyManager:   networkPolicyManager,
	}
}

var _ Handler = (*handler)(nil)

func (h *handler) Deprovision(ctx context.Context, cr *v1alpha1.BtpOperator) error {
	logger := log.FromContext(ctx)

	requiredSecret, err := h.getSecretByNameAndNamespace(ctx, config.SecretName, config.ChartNamespace)
	if err != nil {
		logger.Error(err, fmt.Sprintf("while getting %s secret in %s namespace", config.SecretName, config.ChartNamespace))
		return fmt.Errorf("failed to get the required secret: %w", err)
	}

	h.driftDetector.InitializeFromSecret(requiredSecret)

	if len(cr.GetFinalizers()) == 0 {
		logger.Info("BtpOperator CR without finalizers - nothing to do, waiting for deletion")
		return nil
	}

	if err = h.handleDeprovisioning(ctx, cr); err != nil {
		logger.Error(err, "deprovisioning failed. Restoring resources")
		h.resourceReconciler.ReconcileResourcesWithoutStatusChange(ctx, cr)
		return err
	}
	if cr.IsReasonStringEqual(string(conditions.ServiceInstancesAndBindingsNotCleaned)) {
		h.resourceReconciler.ReconcileResourcesWithoutStatusChange(ctx, cr)

		numberOfBindings, err := h.numberOfResources(ctx, bindingGvk)
		if err != nil {
			return err
		}
		numberOfInstances, err := h.numberOfResources(ctx, instanceGvk)
		if err != nil {
			return err
		}
		if numberOfBindings > 0 || numberOfInstances > 0 {
			logger.Info(fmt.Sprintf("%d instances, %d bindings - leaving deletion", numberOfInstances, numberOfBindings))
			return nil
		}
	}

	if h.instanceBindingService != nil {
		h.instanceBindingService.DisableSISBController()
	}

	logger.Info("Deprovisioning success. Removing finalizers in CR")
	ctrlutil.RemoveFinalizer(cr, deletionFinalizer)
	if err = h.client.Update(ctx, cr); err != nil {
		return err
	}

	return nil
}

func (h *handler) handleDeprovisioning(ctx context.Context, cr *v1alpha1.BtpOperator) error {
	logger := log.FromContext(ctx)

	namespaces := &corev1.NamespaceList{}
	if err := h.client.List(ctx, namespaces); err != nil {
		return err
	}

	if !IsForceDelete(cr) {
		numberOfBindings, err := h.numberOfResources(ctx, bindingGvk)
		if err != nil {
			return err
		}
		numberOfInstances, err := h.numberOfResources(ctx, instanceGvk)
		if err != nil {
			return err
		}

		if numberOfBindings > 0 || numberOfInstances > 0 {
			logger.Info(fmt.Sprintf("Existing resources (%d instances and %d bindings) block BTP Operator deletion.", numberOfInstances, numberOfBindings))
			msg := fmt.Sprintf("All service instances and bindings must be removed: %d instance(s) and %d binding(s)", numberOfInstances, numberOfBindings)
			logger.Info(msg)

			if cr.IsReasonStringEqual(string(conditions.ServiceInstancesAndBindingsNotCleaned)) && cr.Status.State == v1alpha1.StateWarning {
				return nil
			}

			if updateStatusErr := h.statusUpdater.UpdateBtpOperatorStatus(ctx, cr,
				v1alpha1.StateWarning, conditions.ServiceInstancesAndBindingsNotCleaned, msg); updateStatusErr != nil {
				return updateStatusErr
			}
			return nil
		}
	}
	if cr.IsReasonStringEqual(string(conditions.ServiceInstancesAndBindingsNotCleaned)) {
		if updateStatusErr := h.statusUpdater.UpdateBtpOperatorStatus(ctx, cr,
			v1alpha1.StateDeleting, conditions.HardDeleting,
			"BtpOperator is to be deleted after cleaning service instance and binding resources"); updateStatusErr != nil {
			return updateStatusErr
		}
	}

	hardDeleteSucceededCh := make(chan bool, 1)
	hardDeleteTimeoutReachedCh := make(chan bool, 1)
	defer close(hardDeleteTimeoutReachedCh)

	go h.handleHardDelete(ctx, namespaces, hardDeleteSucceededCh, hardDeleteTimeoutReachedCh)

	select {
	case hardDeleteSucceeded := <-hardDeleteSucceededCh:
		if hardDeleteSucceeded {
			logger.Info("Service Instances and Service Bindings hard delete succeeded. Removing module resources")
			if err := h.deleteBtpOperatorResources(ctx); err != nil {
				logger.Error(err, "failed to remove module resources")
				if updateStatusErr := h.statusUpdater.UpdateBtpOperatorStatus(ctx, cr, v1alpha1.StateError, conditions.ResourceRemovalFailed, "Unable to remove installed resources"); updateStatusErr != nil {
					logger.Error(updateStatusErr, "failed to update status")
					return updateStatusErr
				}
				return err
			}
		} else {
			logger.Info("Service Instances and Service Bindings hard delete failed")
			if err := h.statusUpdater.UpdateBtpOperatorStatus(ctx, cr, v1alpha1.StateDeleting, conditions.SoftDeleting, "Being soft deleted"); err != nil {
				logger.Error(err, "failed to update status")
				return err
			}
			if err := h.handleSoftDelete(ctx, namespaces); err != nil {
				logger.Error(err, "failed to soft delete")
				return err
			}
		}
	case <-time.After(config.HardDeleteTimeout):
		logger.Info("hard delete timeout reached", "duration", config.HardDeleteTimeout)
		hardDeleteTimeoutReachedCh <- true
		if err := h.statusUpdater.UpdateBtpOperatorStatus(ctx, cr, v1alpha1.StateDeleting, conditions.SoftDeleting, "Being soft deleted"); err != nil {
			logger.Error(err, "failed to update status")
			return err
		}
		if err := h.handleSoftDelete(ctx, namespaces); err != nil {
			logger.Error(err, "failed to soft delete")
			return err
		}
	}

	return nil
}

func (h *handler) handleHardDelete(ctx context.Context, namespaces *corev1.NamespaceList, hardDeleteSucceededCh, hardDeleteTimeoutReachedCh chan bool) {
	logger := log.FromContext(ctx)
	logger.Info("Deprovisioning BTP Operator - hard delete")
	defer close(hardDeleteSucceededCh)

	errs := make([]error, 0)

	sbCrdExists, err := h.crdExists(ctx, bindingGvk)
	if err != nil {
		logger.Error(err, "while checking CRD existence", "GVK", bindingGvk.String())
		errs = append(errs, err)
	}
	if sbCrdExists {
		if err := h.hardDelete(ctx, bindingGvk, namespaces); err != nil {
			logger.Error(err, "while deleting Service Bindings")
			if !isDeadlineExceeded(err) {
				errs = append(errs, err)
			}
		}
	}

	siCrdExists, err := h.crdExists(ctx, instanceGvk)
	if err != nil {
		logger.Error(err, "while checking CRD existence", "GVK", instanceGvk.String())
		errs = append(errs, err)
	}
	if siCrdExists {
		if err := h.hardDelete(ctx, instanceGvk, namespaces); err != nil {
			logger.Error(err, "while deleting Service Instances")
			if !isDeadlineExceeded(err) {
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
			sbResourcesLeft, err = h.resourcesExist(ctx, namespaces, bindingGvk)
			if err != nil {
				logger.Error(err, "ServiceBinding leftover resources check failed")
				hardDeleteSucceededCh <- false
				return
			}
		}

		if siCrdExists {
			siResourcesLeft, err = h.resourcesExist(ctx, namespaces, instanceGvk)
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

func (h *handler) crdExists(ctx context.Context, gvk schema.GroupVersionKind) (bool, error) {
	crdName := fmt.Sprintf("%ss.%s", strings.ToLower(gvk.Kind), gvk.Group)
	crd := &apiextensionsv1.CustomResourceDefinition{}

	if err := h.client.Get(ctx, client.ObjectKey{Name: crdName}, crd); err != nil {
		if k8serrors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (h *handler) hardDelete(ctx context.Context, gvk schema.GroupVersionKind, namespaces *corev1.NamespaceList) error {
	object := &unstructured.Unstructured{}
	object.SetGroupVersionKind(gvk)
	deleteCtx, cancel := context.WithTimeout(ctx, config.DeleteRequestTimeout)
	defer cancel()

	for _, namespace := range namespaces.Items {
		if err := h.client.DeleteAllOf(deleteCtx, object, client.InNamespace(namespace.Name)); err != nil {
			return err
		}
	}

	return nil
}

func (h *handler) resourcesExist(ctx context.Context, namespaces *corev1.NamespaceList, gvk schema.GroupVersionKind) (bool, error) {
	anyLeft := func(namespace string, gvk schema.GroupVersionKind) (bool, error) {
		list := &unstructured.UnstructuredList{}
		list.SetGroupVersionKind(gvk)
		if err := h.client.List(ctx, list, client.InNamespace(namespace)); err != nil {
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

func (h *handler) numberOfResources(ctx context.Context, gvk schema.GroupVersionKind) (int, error) {
	exists, err := h.crdExists(ctx, gvk)
	if err != nil {
		return 0, err
	}
	if !exists {
		return 0, nil
	}

	list := &unstructured.UnstructuredList{}
	list.SetGroupVersionKind(gvk)
	err = h.client.List(ctx, list, client.InNamespace(corev1.NamespaceAll))
	if err != nil {
		return 0, err
	}
	return len(list.Items), nil
}

func (h *handler) deleteBtpOperatorResources(ctx context.Context) error {
	logger := log.FromContext(ctx)

	logger.Info("getting module resources to delete")
	resourcesToDeleteFromApply, err := h.moduleResourceManager.CreateUnstructuredObjectsFromManifestsDir(h.moduleResourceManager.GetResourcesToApplyPath())
	if err != nil {
		logger.Error(err, "while getting objects to delete from manifests")
		return fmt.Errorf("failed to create deletable objects from manifests: %w", err)
	}
	logger.Info(fmt.Sprintf("got %d module resources to delete from \"apply\" dir", len(resourcesToDeleteFromApply)))

	resourcesToDeleteFromDelete, err := h.moduleResourceManager.CreateUnstructuredObjectsFromManifestsDir(h.moduleResourceManager.GetResourcesToDeletePath())
	if err != nil {
		logger.Error(err, "while getting objects to delete from manifests")
		return fmt.Errorf("failed to create deletable objects from manifests: %w", err)
	}
	logger.Info(fmt.Sprintf("got %d module resources to delete from \"delete\" dir", len(resourcesToDeleteFromDelete)))

	resourcesToDelete := make([]*unstructured.Unstructured, 0)
	resourcesToDelete = append(resourcesToDelete, resourcesToDeleteFromApply...)
	resourcesToDelete = append(resourcesToDelete, resourcesToDeleteFromDelete...)

	if err = h.deleteAllOfResourcesTypes(ctx, resourcesToDelete...); err != nil {
		logger.Error(err, "while deleting module resources")
		return fmt.Errorf("failed to delete module resources: %w", err)
	}

	if err := h.cleanupNetworkPolicies(ctx); err != nil {
		logger.Error(err, "while cleaning up network policies during hard delete")
		return fmt.Errorf("failed to cleanup network policies during hard delete: %w", err)
	}

	if err := h.driftDetector.DeleteClusterIdSecret(ctx); err != nil {
		logger.Error(err, "while deleting cluster ID secret")
		return fmt.Errorf("failed to delete cluster ID secret: %w", err)
	}

	return nil
}

func (h *handler) deleteAllOfResourcesTypes(ctx context.Context, resourcesToDelete ...*unstructured.Unstructured) error {
	logger := log.FromContext(ctx)
	deletedGvks := make(map[string]struct{}, 0)
	for _, u := range resourcesToDelete {
		if _, exists := deletedGvks[u.GroupVersionKind().String()]; exists {
			continue
		}
		logger.Info(fmt.Sprintf("deleting all of %s/%s module resources in %s namespace",
			u.GroupVersionKind().GroupVersion(), u.GetKind(), config.ChartNamespace))
		if err := h.client.DeleteAllOf(ctx, u, client.InNamespace(config.ChartNamespace), managedByLabelFilter); err != nil {
			if !(k8serrors.IsNotFound(err) || k8serrors.IsMethodNotSupported(err) || meta.IsNoMatchError(err)) {
				return err
			}
		}
		deletedGvks[u.GroupVersionKind().String()] = struct{}{}
	}

	return nil
}

func (h *handler) handleSoftDelete(ctx context.Context, namespaces *corev1.NamespaceList) error {
	logger := log.FromContext(ctx)
	logger.Info("Deprovisioning BTP Operator - soft delete")

	logger.Info("Deleting module deployment and webhooks")
	if err := h.preSoftDeleteCleanup(ctx); err != nil {
		logger.Error(err, "module deployment and webhooks deletion failed")
		return err
	}

	sbCrdExists, err := h.crdExists(ctx, bindingGvk)
	if err != nil {
		logger.Error(err, "while checking CRD existence", "GVK", bindingGvk.String())
		return err
	}

	siCrdExists, err := h.crdExists(ctx, instanceGvk)
	if err != nil {
		logger.Error(err, "while checking CRD existence", "GVK", instanceGvk.String())
		return err
	}

	if sbCrdExists {
		logger.Info("Removing finalizers in Service Bindings and deleting connected Secrets")
		if err := h.softDelete(ctx, bindingGvk); err != nil {
			logger.Error(err, "while deleting Service Bindings")
			return err
		}
		if err := h.ensureResourcesDontExist(ctx, bindingGvk); err != nil {
			logger.Error(err, "Service Bindings still exist")
			return err
		}
	}

	if siCrdExists {
		logger.Info("Removing finalizers in Service Instances")
		if err := h.softDelete(ctx, instanceGvk); err != nil {
			logger.Error(err, "while deleting Service Instances")
			return err
		}
		if err := h.ensureResourcesDontExist(ctx, instanceGvk); err != nil {
			logger.Error(err, "Service Instances still exist")
			return err
		}
	}

	logger.Info("Deleting module resources")
	if err := h.deleteBtpOperatorResources(ctx); err != nil {
		logger.Error(err, "failed to delete module resources")
		return err
	}

	return nil
}

func (h *handler) preSoftDeleteCleanup(ctx context.Context) error {
	toDelete := []client.Object{
		&appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: config.DeploymentName, Namespace: config.ChartNamespace}},
		&admissionregistrationv1.MutatingWebhookConfiguration{ObjectMeta: metav1.ObjectMeta{Name: mutatingWebhookName}},
		&admissionregistrationv1.ValidatingWebhookConfiguration{ObjectMeta: metav1.ObjectMeta{Name: validatingWebhookName}},
	}
	for _, obj := range toDelete {
		if err := h.client.Delete(ctx, obj); client.IgnoreNotFound(err) != nil {
			return err
		}
	}

	if err := h.cleanupNetworkPolicies(ctx); err != nil {
		return fmt.Errorf("failed to cleanup network policies during soft delete: %w", err)
	}

	return nil
}

func (h *handler) softDelete(ctx context.Context, gvk schema.GroupVersionKind) error {
	list := GvkToList(gvk)

	if err := h.client.List(ctx, list); err != nil {
		return fmt.Errorf("%w; could not list in soft delete", err)
	}

	isBinding := gvk.Kind == btpOperatorServiceBinding
	for _, item := range list.Items {
		if item.GetDeletionTimestamp().IsZero() {
			if err := h.client.Delete(ctx, &item); err != nil {
				return err
			}
		}
		item.SetFinalizers([]string{})
		if err := h.client.Update(ctx, &item); err != nil {
			return err
		}

		if isBinding {
			secret := &corev1.Secret{}
			secret.Name = item.GetName()
			secret.Namespace = item.GetNamespace()
			if err := h.client.Delete(ctx, secret); err != nil && !k8serrors.IsNotFound(err) {
				return err
			}
		}
	}

	return nil
}

func (h *handler) ensureResourcesDontExist(ctx context.Context, gvk schema.GroupVersionKind) error {
	list := GvkToList(gvk)

	if err := h.client.List(ctx, list); err != nil {
		if !k8serrors.IsNotFound(err) {
			return err
		}
	} else if len(list.Items) > 0 {
		return fmt.Errorf("list returned %d records", len(list.Items))
	}

	return nil
}

func (h *handler) cleanupNetworkPolicies(ctx context.Context) error {
	if h.networkPolicyManager == nil {
		return nil
	}
	return h.networkPolicyManager.CleanupNetworkPolicies(ctx)
}

func (h *handler) getSecretByNameAndNamespace(ctx context.Context, name, namespace string) (*corev1.Secret, error) {
	secret := &corev1.Secret{}
	if err := h.apiServerClient.Get(ctx, client.ObjectKey{Name: name, Namespace: namespace}, secret); err != nil {
		if k8serrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("while getting %s secret from %s namespace: %w", name, namespace, err)
	}
	return secret, nil
}

// IsForceDelete reports whether the CR has the force-delete label set to "true".
func IsForceDelete(cr *v1alpha1.BtpOperator) bool {
	if _, exists := cr.Labels[forceDeleteLabelKey]; !exists {
		return false
	}
	return cr.Labels[forceDeleteLabelKey] == "true"
}

// GvkToList converts a GVK to its corresponding list GVK.
func GvkToList(gvk schema.GroupVersionKind) *unstructured.UnstructuredList {
	listGvk := gvk
	listGvk.Kind = gvk.Kind + "List"
	list := &unstructured.UnstructuredList{}
	list.SetGroupVersionKind(listGvk)
	return list
}

func isDeadlineExceeded(err error) bool {
	return errors.Is(err, context.DeadlineExceeded)
}
