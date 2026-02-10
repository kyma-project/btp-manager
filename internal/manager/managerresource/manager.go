package managerresource

import (
	"context"
	"fmt"

	"github.com/kyma-project/btp-manager/internal/manifest"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type Resource interface {
	Name() string
	ManifestsPath() string
	Object() client.Object
}

type Manager struct {
	resources       []Resource
	manifestHandler *manifest.Handler
}

func NewManager(resources []Resource, manifestHandler *manifest.Handler) *Manager {
	return &Manager{
		resources:       resources,
		manifestHandler: manifestHandler,
	}
}

func (m *Manager) ResourcesToCreate(ctx context.Context) ([]*unstructured.Unstructured, error) {
	logger := log.FromContext(ctx)
	var resources []*unstructured.Unstructured

	for _, resource := range m.resources {
		logger.Info(fmt.Sprintf("Loading %s", resource.Name()))
		objects, err := m.manifestHandler.CollectObjectsFromDir(resource.ManifestsPath())
		if err != nil {
			return nil, fmt.Errorf("while collecting objects from directory %s: %w", resource.ManifestsPath(), err)
		}

		unstructuredObjects, err := m.manifestHandler.ObjectsToUnstructured(objects)
		if err != nil {
			return nil, fmt.Errorf("while converting to unstructured: %w", err)
		}
		logger.Info(fmt.Sprintf("Found %d objects", len(unstructuredObjects)))

		resources = append(resources, unstructuredObjects...)
	}

	return resources, nil
}

func (m *Manager) ResourcesToDelete(ctx context.Context) ([]client.Object, error) {
	logger := log.FromContext(ctx)
	var resources []client.Object

	for _, resource := range m.resources {
		logger.Info(fmt.Sprintf("%s disabled, preparing existing resources for removal", resource.Name()))
		resources = append(resources, resource.Object())
	}

	return resources, nil
}
