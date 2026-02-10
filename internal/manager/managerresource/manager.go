package managerresource

import (
	"context"
	"fmt"

	"github.com/kyma-project/btp-manager/api/v1alpha1"
	"github.com/kyma-project/btp-manager/controllers/config"
	"github.com/kyma-project/btp-manager/internal/manager/moduleresource"
	"github.com/kyma-project/btp-manager/internal/manifest"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	managedByLabelFilter = client.MatchingLabels{moduleresource.ManagedByLabelKey: moduleresource.OperatorName}
)

type Resource interface {
	Name() string
	Enabled(cr *v1alpha1.BtpOperator) bool
	ManifestsPath() string
	Object() client.Object
}

type Manager struct {
	client          client.Client
	scheme          *runtime.Scheme
	manifestHandler *manifest.Handler
	resources       []Resource
}

func NewManager(client client.Client, scheme *runtime.Scheme, resources []Resource) *Manager {
	return &Manager{
		client:          client,
		scheme:          scheme,
		manifestHandler: &manifest.Handler{Scheme: scheme},
		resources:       resources,
	}
}

func (m *Manager) createManagerResources(ctx context.Context) error {
	logger := log.FromContext(ctx)

	btpOperator := &v1alpha1.BtpOperator{}
	if err := m.client.Get(ctx, client.ObjectKey{Namespace: config.KymaSystemNamespaceName, Name: config.BtpOperatorCrName}, btpOperator); err != nil {
		return fmt.Errorf("failed to get BTP Operator CR: %w", err)
	}

	for _, resource := range m.resources {
		if resource.Enabled(btpOperator) {
			logger.Info(fmt.Sprintf("loading %s", resource.Name()))

			objects, err := m.manifestHandler.CollectObjectsFromDir(resource.ManifestsPath())
			if err != nil {
				return fmt.Errorf("while collecting objects from directory %s: %w", resource.ManifestsPath(), err)
			}

			unstructuredObjects, err := m.manifestHandler.ObjectsToUnstructured(objects)
			if err != nil {
				return fmt.Errorf("while converting to unstructured: %w", err)
			}

			logger.Info(fmt.Sprintf("creating %d %s", len(objects), resource.Name()))
			err = m.createOrUpdateResources(ctx, unstructuredObjects)
			if err != nil {
				return fmt.Errorf("failed to create %s: %w", resource.Name(), err)
			}
		}
	}

	return nil
}

func (m *Manager) deleteManagerResources(ctx context.Context) error {
	logger := log.FromContext(ctx)

	btpOperator := &v1alpha1.BtpOperator{}
	if err := m.client.Get(ctx, client.ObjectKey{Namespace: config.KymaSystemNamespaceName, Name: config.BtpOperatorCrName}, btpOperator); err != nil {
		return fmt.Errorf("failed to get BTP Operator CR: %w", err)
	}

	for _, resource := range m.resources {
		if !resource.Enabled(btpOperator) {
			logger.Info(fmt.Sprintf("%s disabled, cleaning up existing ones", resource.Name()))
			if err := m.client.DeleteAllOf(ctx, resource.Object(), client.InNamespace(config.ChartNamespace), managedByLabelFilter); err != nil {
				if !(k8serrors.IsNotFound(err) || k8serrors.IsMethodNotSupported(err) || meta.IsNoMatchError(err)) {
					return fmt.Errorf("failed to delete %s: %w", resource.Name(), err)
				}
			}
		}
	}

	return nil
}

func (m *Manager) createOrUpdateResources(ctx context.Context, us []*unstructured.Unstructured) error {
	for _, u := range us {
		if err := m.createOrUpdateResource(ctx, u); err != nil {
			return err
		}
	}
	return nil
}

func (m *Manager) createOrUpdateResource(ctx context.Context, u *unstructured.Unstructured) error {
	preExistingResource := &unstructured.Unstructured{}
	preExistingResource.SetGroupVersionKind(u.GroupVersionKind())
	if err := m.client.Get(ctx, client.ObjectKey{Name: u.GetName(), Namespace: u.GetNamespace()}, preExistingResource); err != nil {
		if k8serrors.IsNotFound(err) {
			return m.createResource(ctx, u)
		}
		return fmt.Errorf("while trying to get %s %s: %w", u.GetName(), u.GetKind(), err)
	}
	u.SetResourceVersion(preExistingResource.GetResourceVersion())
	return m.updateResource(ctx, u)
}

func (m *Manager) createResource(ctx context.Context, u *unstructured.Unstructured) error {
	if err := m.client.Create(ctx, u, client.FieldOwner(moduleresource.OperatorName)); err != nil {
		return fmt.Errorf("while creating %s %s: %w", u.GetName(), u.GetKind(), err)
	}
	return nil
}

func (m *Manager) updateResource(ctx context.Context, u *unstructured.Unstructured) error {
	if err := m.client.Update(ctx, u, client.FieldOwner(moduleresource.OperatorName)); err != nil {
		return fmt.Errorf("while updating %s %s: %w", u.GetName(), u.GetKind(), err)
	}
	return nil
}
