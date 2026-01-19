package moduleresource

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kyma-project/btp-manager/internal/manifest"
)

const (
	OperatorName = "btp-manager"
	ModuleName   = "btp-operator"

	ManagedByLabelKey             = "app.kubernetes.io/managed-by"
	ChartVersionLabelKey          = "chart-version"
	KymaProjectModuleLabelKey     = "kyma-project.io/module"
	ClusterIdSecretKey            = "cluster_id"
	CredentialsNamespaceSecretKey = "credentials_namespace"

	DeploymentKind = "Deployment"
)

const (
	clusterIdConfigMapKey           = "CLUSTER_ID"
	releaseNamespaceConfigMapKey    = "RELEASE_NAMESPACE"
	managementNamespaceConfigMapKey = "MANAGEMENT_NAMESPACE"
	enableLimitedCacheConfigMapKey  = "ENABLE_LIMITED_CACHE"

	sapBtpServiceOperatorContainerName = "manager"
	kubeRbacProxyContainerName         = "kube-rbac-proxy"
)

type ModuleResource struct {
	Kind string
	Name string
}

type Manager struct {
	client          client.Client
	scheme          *runtime.Scheme
	manifestHandler *manifest.Handler
	config          Config

	resourceIndices map[ModuleResource]int

	clusterID            string
	credentialsNamespace string
}

type Config struct {
	ChartNamespace       string
	ResourcesPath        string
	ManagerResourcesPath string
	SapBtpOperatorImage  string
	KubeRbacProxyImage   string
	EnableLimitedCache   string
}

type State struct {
	ClusterID            string
	CredentialsNamespace string
}

func NewManager(client client.Client, scheme *runtime.Scheme, config Config) *Manager {
	return &Manager{
		client:          client,
		scheme:          scheme,
		manifestHandler: &manifest.Handler{Scheme: scheme},
		config:          config,
		resourceIndices: make(map[ModuleResource]int),
	}
}

func (m *Manager) UpdateState(state State) {
	m.clusterID = state.ClusterID
	m.credentialsNamespace = state.CredentialsNamespace
}

func (m *Manager) createUnstructuredObjectsFromManifestsDir(manifestsDir string) ([]*unstructured.Unstructured, error) {
	objects, err := m.manifestHandler.CollectObjectsFromDir(manifestsDir)
	if err != nil {
		return nil, fmt.Errorf("while collecting objects from directory %s: %w", manifestsDir, err)
	}

	unstructuredObjects, err := m.manifestHandler.ObjectsToUnstructured(objects)
	if err != nil {
		return nil, fmt.Errorf("while converting to unstructured: %w", err)
	}

	for i, u := range unstructuredObjects {
		resource := ModuleResource{
			Kind: u.GetKind(),
			Name: u.GetName(),
		}
		m.resourceIndices[resource] = i
	}

	return unstructuredObjects, nil
}

func (m *Manager) addLabels(chartVersion string, us ...*unstructured.Unstructured) error {
	for _, u := range us {
		labels := u.GetLabels()
		if len(labels) == 0 {
			labels = make(map[string]string)
		}
		labels[ManagedByLabelKey] = OperatorName
		labels[ChartVersionLabelKey] = chartVersion
		labels[KymaProjectModuleLabelKey] = ModuleName
		u.SetLabels(labels)

		if u.GetKind() == DeploymentKind {
			if err := m.addLabelsInPodTemplate(u); err != nil {
				return err
			}
		}
	}
	return nil
}

func (m *Manager) addLabelsInPodTemplate(u *unstructured.Unstructured) error {
	tplLabels, found, err := unstructured.NestedStringMap(u.Object, "spec", "template", "metadata", "labels")
	if err != nil {
		return fmt.Errorf("failed to get pod template labels for deployment %s: %w", u.GetName(), err)
	}
	if !found || tplLabels == nil {
		tplLabels = make(map[string]string)
	}
	tplLabels[KymaProjectModuleLabelKey] = ModuleName
	if err := unstructured.SetNestedStringMap(u.Object, tplLabels, "spec", "template", "metadata", "labels"); err != nil {
		return fmt.Errorf("failed to set pod template labels for deployment %s: %w", u.GetName(), err)
	}
	return nil
}

func (m *Manager) setNamespace(us []*unstructured.Unstructured) {
	for _, u := range us {
		u.SetNamespace(m.config.ChartNamespace)
	}
}

func (m *Manager) setConfigMapValues(secret *corev1.Secret, u *unstructured.Unstructured) error {
	if err := unstructured.SetNestedField(u.Object, string(secret.Data[ClusterIdSecretKey]), "data", clusterIdConfigMapKey); err != nil {
		return fmt.Errorf("failed to set cluster_id: %w", err)
	}

	if err := unstructured.SetNestedField(u.Object, m.credentialsNamespace, "data", releaseNamespaceConfigMapKey); err != nil {
		return fmt.Errorf("failed to set release namespace: %w", err)
	}

	if err := unstructured.SetNestedField(u.Object, m.credentialsNamespace, "data", managementNamespaceConfigMapKey); err != nil {
		return fmt.Errorf("failed to set management namespace: %w", err)
	}

	if err := unstructured.SetNestedField(u.Object, m.config.EnableLimitedCache, "data", enableLimitedCacheConfigMapKey); err != nil {
		return fmt.Errorf("failed to set enable limited cache: %w", err)
	}

	return nil
}

func (m *Manager) setSecretValues(secret *corev1.Secret, u *unstructured.Unstructured) error {
	u.SetNamespace(m.credentialsNamespace)

	for k := range secret.Data {
		if k == ClusterIdSecretKey || k == CredentialsNamespaceSecretKey {
			continue
		}
		if err := unstructured.SetNestedField(u.Object, base64.StdEncoding.EncodeToString(secret.Data[k]), "data", k); err != nil {
			return fmt.Errorf("failed to set secret field %s: %w", k, err)
		}
	}

	return nil
}

func (m *Manager) setDeploymentImages(u *unstructured.Unstructured) error {
	if err := m.setContainerImage(u, sapBtpServiceOperatorContainerName, m.config.SapBtpOperatorImage); err != nil {
		return fmt.Errorf("failed to set container image for sap-btp-service-operator: %w", err)
	}

	if err := m.setContainerImage(u, kubeRbacProxyContainerName, m.config.KubeRbacProxyImage); err != nil {
		return fmt.Errorf("failed to set container image for kube-rbac-proxy: %w", err)
	}

	return nil
}

func (m *Manager) setContainerImage(u *unstructured.Unstructured, containerName, image string) error {
	containers, found, err := unstructured.NestedSlice(u.Object, "spec", "template", "spec", "containers")
	if err != nil {
		return fmt.Errorf("failed to get containers from %s %s: %w", u.GetKind(), u.GetName(), err)
	}
	if !found {
		return fmt.Errorf("containers not found in %s %s", u.GetKind(), u.GetName())
	}

	containerFound := false
	for i, c := range containers {
		container, ok := c.(map[string]interface{})
		if !ok {
			return fmt.Errorf("cannot cast container field to map[string]interface{}: %v", c)
		}
		if container["name"] == containerName {
			container["image"] = image
			containers[i] = container
			containerFound = true
			break
		}
	}

	if !containerFound {
		return fmt.Errorf("container %s not found in %s %s", containerName, u.GetKind(), u.GetName())
	}

	return unstructured.SetNestedSlice(u.Object, containers, "spec", "template", "spec", "containers")
}

func (m *Manager) applyOrUpdateResources(ctx context.Context, us []*unstructured.Unstructured) error {
	for _, u := range us {
		preExistingResource := &unstructured.Unstructured{}
		preExistingResource.SetGroupVersionKind(u.GroupVersionKind())

		err := m.client.Get(ctx, client.ObjectKey{Name: u.GetName(), Namespace: u.GetNamespace()}, preExistingResource)
		if err != nil {
			if !k8serrors.IsNotFound(err) {
				return fmt.Errorf("while trying to get %s %s: %w", u.GetKind(), u.GetName(), err)
			}
			if err := m.client.Patch(ctx, u, client.Apply, client.ForceOwnership, client.FieldOwner(OperatorName)); err != nil {
				return fmt.Errorf("while applying %s %s: %w", u.GetKind(), u.GetName(), err)
			}
		} else {
			// Resource exists, update it using Server-Side Apply
			u.SetResourceVersion(preExistingResource.GetResourceVersion())
			if err := m.client.Update(ctx, u, client.FieldOwner(OperatorName)); err != nil {
				return fmt.Errorf("while updating %s %s: %w", u.GetKind(), u.GetName(), err)
			}
		}
	}
	return nil
}

func (m *Manager) deleteResources(ctx context.Context, us []*unstructured.Unstructured) error {
	for _, u := range us {
		if err := m.client.Delete(ctx, u); err != nil {
			if !k8serrors.IsNotFound(err) {
				return fmt.Errorf("while deleting %s %s: %w", u.GetKind(), u.GetName(), err)
			}
		}
	}
	return nil
}

func (m *Manager) DeleteOutdatedResources(ctx context.Context) error {
	deletePath := m.config.ResourcesPath + "/delete"
	objects, err := m.createUnstructuredObjectsFromManifestsDir(deletePath)
	if err != nil {
		return nil
	}

	return m.deleteResources(ctx, objects)
}

func (m *Manager) waitForResourcesReadiness(ctx context.Context, us []*unstructured.Unstructured, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	errChan := make(chan error, len(us))

	for _, u := range us {
		go func(resource *unstructured.Unstructured) {
			errChan <- m.waitForResourceReady(ctx, resource)
		}(u)
	}

	for range us {
		if err := <-errChan; err != nil {
			return err
		}
	}

	return nil
}

func (m *Manager) waitForResourceReady(ctx context.Context, u *unstructured.Unstructured) error {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for %s %s to be ready", u.GetKind(), u.GetName())
		case <-ticker.C:
			current := &unstructured.Unstructured{}
			current.SetGroupVersionKind(u.GroupVersionKind())

			if err := m.client.Get(ctx, client.ObjectKey{Name: u.GetName(), Namespace: u.GetNamespace()}, current); err != nil {
				if k8serrors.IsNotFound(err) {
					continue
				}
				return fmt.Errorf("while checking readiness of %s %s: %w", u.GetKind(), u.GetName(), err)
			}

			if m.isResourceReady(current) {
				return nil
			}
		}
	}
}

func (m *Manager) isResourceReady(u *unstructured.Unstructured) bool {
	kind := u.GetKind()

	switch kind {
	case "Deployment":
		return m.isDeploymentReady(u)
	default:
		return true
	}
}

func (m *Manager) isDeploymentReady(u *unstructured.Unstructured) bool {
	replicas, _, _ := unstructured.NestedInt64(u.Object, "spec", "replicas")
	readyReplicas, _, _ := unstructured.NestedInt64(u.Object, "status", "readyReplicas")
	return replicas == readyReplicas && replicas > 0
}
