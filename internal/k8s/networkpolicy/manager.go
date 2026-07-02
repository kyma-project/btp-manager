package networkpolicy

import (
	"context"
	"fmt"
	"os"

	"github.com/kyma-project/btp-manager/controllers/config"
	"github.com/kyma-project/btp-manager/internal/manifest"
	networkingv1 "k8s.io/api/networking/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	operatorName = "btp-manager"
	moduleName   = "btp-operator"

	managedByLabelKey         = "app.kubernetes.io/managed-by"
	kymaProjectModuleLabelKey = "kyma-project.io/module"

	oldWebhookNetworkPolicyName = "kyma-project.io--btp-operator-allow-to-webhook"
)

var managedLabelsFilter = client.MatchingLabels{managedByLabelKey: operatorName, kymaProjectModuleLabelKey: moduleName}

type NetworkPolicyManager interface {
	CleanupNetworkPolicies(ctx context.Context) error
	DeleteOldWebhookNetworkPolicy(ctx context.Context) error
	IsManaged(obj *networkingv1.NetworkPolicy) (bool, error)
	LoadNetworkPolicies() ([]*unstructured.Unstructured, error)
}

type Manager struct {
	client          client.Client
	manifestHandler *manifest.Handler
}

func NewManager(k8sClient client.Client, manifestHandler *manifest.Handler) *Manager {
	return &Manager{
		client:          k8sClient,
		manifestHandler: manifestHandler,
	}
}

var _ NetworkPolicyManager = (*Manager)(nil)

func (m *Manager) getNetworkPoliciesPath() string {
	return fmt.Sprintf("%s%cnetwork-policies", config.ManagerResourcesPath, os.PathSeparator)
}

func (m *Manager) LoadNetworkPolicies() ([]*unstructured.Unstructured, error) {
	objects, err := m.manifestHandler.CollectObjectsFromDir(m.getNetworkPoliciesPath())
	if err != nil {
		return nil, err
	}
	return m.manifestHandler.ObjectsToUnstructured(objects)
}

func (m *Manager) CleanupNetworkPolicies(ctx context.Context) error {
	logger := log.FromContext(ctx)
	logger.Info("deleting all managed network policies")
	if err := m.client.DeleteAllOf(ctx, &networkingv1.NetworkPolicy{}, client.InNamespace(config.ChartNamespace), managedLabelsFilter); err != nil {
		if !(k8serrors.IsNotFound(err) || k8serrors.IsMethodNotSupported(err) || meta.IsNoMatchError(err)) {
			return fmt.Errorf("failed to delete network policies: %w", err)
		}
	}
	return nil
}

func (m *Manager) DeleteOldWebhookNetworkPolicy(ctx context.Context) error {
	logger := log.FromContext(ctx)
	oldPolicy := &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      oldWebhookNetworkPolicyName,
			Namespace: config.ChartNamespace,
		},
	}
	if err := m.client.Delete(ctx, oldPolicy); err != nil {
		if !k8serrors.IsNotFound(err) {
			logger.Error(err, "failed to delete old webhook network policy during migration", "policyName", oldWebhookNetworkPolicyName)
			return fmt.Errorf("failed to delete old webhook network policy: %w", err)
		}
		logger.Info("old webhook network policy not found, skipping deletion", "policyName", oldWebhookNetworkPolicyName)
	}
	return nil
}

func (m *Manager) IsManaged(obj *networkingv1.NetworkPolicy) (bool, error) {
	labels := obj.GetLabels()
	if labels != nil {
		if labels[managedByLabelKey] == operatorName && labels[kymaProjectModuleLabelKey] == moduleName {
			return true, nil
		}
	}
	nameSet, err := m.getNetworkPolicyNamesFromManifests()
	if err != nil {
		return false, err
	}
	_, ok := nameSet[obj.GetName()]
	return ok, nil
}

func (m *Manager) getNetworkPolicyNamesFromManifests() (map[string]struct{}, error) {
	names := make(map[string]struct{})
	us, err := m.LoadNetworkPolicies()
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
