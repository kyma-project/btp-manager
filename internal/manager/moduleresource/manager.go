package moduleresource

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/kyma-project/btp-manager/controllers/config"
	"github.com/kyma-project/btp-manager/internal/manifest"
	"github.com/kyma-project/btp-manager/internal/ymlutils"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	OperatorName = "btp-manager"
	ModuleName   = "btp-operator"

	ManagedByLabelKey         = "app.kubernetes.io/managed-by"
	ChartVersionLabelKey      = "chart-version"
	KymaProjectModuleLabelKey = "kyma-project.io/module"

	ClientIdSecretKey             = "clientid"
	ClientSecretKey               = "clientsecret"
	SmUrlSecretKey                = "sm_url"
	TokenUrlSecretKey             = "tokenurl"
	ClusterIdSecretKey            = "cluster_id"
	CredentialsNamespaceSecretKey = "credentials_namespace"

	SapBtpServiceOperatorName = "sap-btp-service-operator"
	SapBtpServiceOperatorEnv  = "SAP_BTP_SERVICE_OPERATOR"

	DeploymentKind = "Deployment"
)

const (
	clusterIdConfigMapKey           = "CLUSTER_ID"
	releaseNamespaceConfigMapKey    = "RELEASE_NAMESPACE"
	managementNamespaceConfigMapKey = "MANAGEMENT_NAMESPACE"
	enableLimitedCacheConfigMapKey  = "ENABLE_LIMITED_CACHE"

	sapBtpServiceOperatorContainerName = "manager"
)

// CredentialsProvider gives the module resource manager the authoritative credential
// values it needs to configure the operand's ConfigMap and Secret.
// drift.DriftDetector satisfies this interface via Go's structural typing.
type CredentialsProvider interface {
	CredentialsNamespaceFromManager() string
	ClusterIdFromManager() string
}

type Metadata struct {
	Kind string
	Name string
}

type ResourceManager interface {
	CreateUnstructuredObjectsFromManifestsDir(manifestsDir string) ([]*unstructured.Unstructured, error)
	ResourcesOfKinds(resources []*unstructured.Unstructured, kinds ...string) (matching, rest []*unstructured.Unstructured)
	PrepareModuleResources(ctx context.Context, resourcesToApply []*unstructured.Unstructured, s *corev1.Secret) error
	ApplyOrUpdateResources(ctx context.Context, us []*unstructured.Unstructured) error
	WaitForResourcesReadiness(ctx context.Context, us []*unstructured.Unstructured) error
	DeleteOutdatedResources(ctx context.Context) error
	DeleteResources(ctx context.Context, us []*unstructured.Unstructured) error
	DeleteCreationTimestamp(us ...*unstructured.Unstructured)
	GetResourcesToApplyPath() string
	GetResourcesToDeletePath() string
}

type Manager struct {
	client          client.Client
	scheme          *runtime.Scheme
	manifestHandler *manifest.Handler
	resourceIndices map[Metadata]int
	driftDetector   CredentialsProvider
}

func NewManager(client client.Client, scheme *runtime.Scheme, driftDetector CredentialsProvider) *Manager {
	return &Manager{
		client:          client,
		scheme:          scheme,
		manifestHandler: &manifest.Handler{Scheme: scheme},
		resourceIndices: make(map[Metadata]int),
		driftDetector:   driftDetector,
	}
}

var _ ResourceManager = (*Manager)(nil)

func (m *Manager) CreateUnstructuredObjectsFromManifestsDir(manifestsDir string) ([]*unstructured.Unstructured, error) {
	objects, err := m.manifestHandler.CollectObjectsFromDir(manifestsDir)
	if err != nil {
		return nil, fmt.Errorf("while collecting objects from directory %s: %w", manifestsDir, err)
	}

	unstructuredObjects, err := m.manifestHandler.ObjectsToUnstructured(objects)
	if err != nil {
		return nil, fmt.Errorf("while converting to unstructured: %w", err)
	}

	m.indexModuleResources(unstructuredObjects)

	return unstructuredObjects, nil
}

func (m *Manager) indexModuleResources(unstructuredObjects []*unstructured.Unstructured) {
	m.resourceIndices = make(map[Metadata]int, len(unstructuredObjects))
	for i, u := range unstructuredObjects {
		resource := Metadata{
			Kind: u.GetKind(),
			Name: u.GetName(),
		}
		m.resourceIndices[resource] = i
	}
}

// ResourcesOfKinds partitions resources into two slices: those whose kind is
// listed in kinds (matching) and everything else (rest). The partition is based
// on the index built during manifest loading, so resources appended after
// loading (e.g. network policies) are always placed in rest regardless of kind.
func (m *Manager) ResourcesOfKinds(resources []*unstructured.Unstructured, kinds ...string) (matching, rest []*unstructured.Unstructured) {
	kindSet := make(map[string]struct{}, len(kinds))
	for _, k := range kinds {
		kindSet[k] = struct{}{}
	}

	matchingIndices := make(map[int]struct{})
	for meta, idx := range m.resourceIndices {
		if _, ok := kindSet[meta.Kind]; ok {
			matchingIndices[idx] = struct{}{}
		}
	}

	for i, r := range resources {
		if _, ok := matchingIndices[i]; ok {
			matching = append(matching, r)
		} else {
			rest = append(rest, r)
		}
	}
	return
}

func (m *Manager) PrepareModuleResources(ctx context.Context, resourcesToApply []*unstructured.Unstructured, s *corev1.Secret) error {
	configMapIndex, secretIndex, deploymentIndex := -1, -1, -1
	for i, u := range resourcesToApply {
		switch {
		case u.GetName() == "sap-btp-operator-config" && u.GetKind() == "ConfigMap":
			configMapIndex = i
		case u.GetName() == SapBtpServiceOperatorName && u.GetKind() == "Secret":
			secretIndex = i
		case u.GetName() == config.DeploymentName && u.GetKind() == DeploymentKind:
			deploymentIndex = i
		}
	}
	if configMapIndex < 0 || secretIndex < 0 || deploymentIndex < 0 {
		return fmt.Errorf("required module resources not found in manifests (configMapIndex=%d, secretIndex=%d, deploymentIndex=%d)", configMapIndex, secretIndex, deploymentIndex)
	}
	chartVer, err := ymlutils.ExtractStringValueFromYamlForGivenKey(fmt.Sprintf("%s%cChart.yaml", config.ChartPath, os.PathSeparator), "version")
	if err != nil {
		return fmt.Errorf("failed to get module chart version: %w", err)
	}

	if err := m.AddLabels(chartVer, resourcesToApply...); err != nil {
		return fmt.Errorf("failed to add labels to resources: %w", err)
	}
	m.SetNamespace(resourcesToApply)

	if err := m.SetConfigMapValues(resourcesToApply[configMapIndex]); err != nil {
		return fmt.Errorf("failed to set ConfigMap values: %w", err)
	}
	if err := m.SetSecretValues(s, resourcesToApply[secretIndex]); err != nil {
		return fmt.Errorf("failed to set Secret values: %w", err)
	}
	if err := m.SetDeploymentImages(resourcesToApply[deploymentIndex]); err != nil {
		return fmt.Errorf("failed to set container images in Deployment: %w", err)
	}

	return nil
}

func (m *Manager) DeleteCreationTimestamp(us ...*unstructured.Unstructured) {
	for _, u := range us {
		unstructured.RemoveNestedField(u.Object, "metadata", "creationTimestamp")
	}
}

func (m *Manager) AddLabels(chartVersion string, us ...*unstructured.Unstructured) error {
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

func (m *Manager) SetNamespace(us []*unstructured.Unstructured) {
	for _, u := range us {
		u.SetNamespace(config.ChartNamespace)
	}
}

func (m *Manager) SetConfigMapValues(u *unstructured.Unstructured) error {
	credentialsNamespace := m.driftDetector.CredentialsNamespaceFromManager()
	clusterId := m.driftDetector.ClusterIdFromManager()

	if err := unstructured.SetNestedField(u.Object, clusterId, "data", clusterIdConfigMapKey); err != nil {
		return fmt.Errorf("failed to set cluster_id: %w", err)
	}

	if err := unstructured.SetNestedField(u.Object, credentialsNamespace, "data", releaseNamespaceConfigMapKey); err != nil {
		return fmt.Errorf("failed to set release namespace: %w", err)
	}

	if err := unstructured.SetNestedField(u.Object, credentialsNamespace, "data", managementNamespaceConfigMapKey); err != nil {
		return fmt.Errorf("failed to set management namespace: %w", err)
	}

	if err := unstructured.SetNestedField(u.Object, config.EnableLimitedCache, "data", enableLimitedCacheConfigMapKey); err != nil {
		return fmt.Errorf("failed to set enable limited cache: %w", err)
	}

	return nil
}

func (m *Manager) SetSecretValues(secret *corev1.Secret, u *unstructured.Unstructured) error {
	u.SetNamespace(m.driftDetector.CredentialsNamespaceFromManager())

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

func (m *Manager) SetDeploymentImages(u *unstructured.Unstructured) error {
	sapBtpServiceOperatorImage := os.Getenv(SapBtpServiceOperatorEnv)
	if err := m.setContainerImage(u, sapBtpServiceOperatorContainerName, sapBtpServiceOperatorImage); err != nil {
		return fmt.Errorf("failed to set container image for %s: %w", SapBtpServiceOperatorName, err)
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

func (m *Manager) GetResourcesToApplyPath() string {
	return fmt.Sprintf("%s%capply", config.ResourcesPath, os.PathSeparator)
}

func (m *Manager) GetResourcesToDeletePath() string {
	return fmt.Sprintf("%s%cdelete", config.ResourcesPath, os.PathSeparator)
}

func (m *Manager) ApplyOrUpdateResources(ctx context.Context, us []*unstructured.Unstructured) error {
	for _, u := range us {
		if err := m.applyOrUpdateResource(ctx, u); err != nil {
			return err
		}
	}
	return nil
}

func (m *Manager) applyOrUpdateResource(ctx context.Context, u *unstructured.Unstructured) error {
	preExistingResource := &unstructured.Unstructured{}
	preExistingResource.SetGroupVersionKind(u.GroupVersionKind())
	if err := m.client.Get(ctx, client.ObjectKey{Name: u.GetName(), Namespace: u.GetNamespace()}, preExistingResource); err != nil {
		if k8serrors.IsNotFound(err) {
			return m.applyResource(ctx, u)
		}
		return fmt.Errorf("while trying to get %s %s: %w", u.GetName(), u.GetKind(), err)
	}
	u.SetResourceVersion(preExistingResource.GetResourceVersion())
	return m.updateResource(ctx, u)
}

func (m *Manager) applyResource(ctx context.Context, u *unstructured.Unstructured) error {
	if err := m.client.Create(ctx, u, client.FieldOwner(OperatorName)); err != nil {
		return fmt.Errorf("while creating %s %s: %w", u.GetName(), u.GetKind(), err)
	}
	return nil
}

func (m *Manager) updateResource(ctx context.Context, u *unstructured.Unstructured) error {
	if err := m.client.Update(ctx, u, client.FieldOwner(OperatorName)); err != nil {
		return fmt.Errorf("while updating %s %s: %w", u.GetName(), u.GetKind(), err)
	}
	return nil
}

func (m *Manager) DeleteOutdatedResources(ctx context.Context) error {
	objects, err := m.CreateUnstructuredObjectsFromManifestsDir(m.GetResourcesToDeletePath())
	if err != nil {
		return fmt.Errorf("failed to create deletable objects from manifests: %w", err)
	}

	return m.DeleteResources(ctx, objects)
}

func (m *Manager) DeleteResources(ctx context.Context, us []*unstructured.Unstructured) error {
	var errs []string
	for _, u := range us {
		if err := m.client.Delete(ctx, u); err != nil {
			if !k8serrors.IsNotFound(err) {
				errs = append(errs, fmt.Sprintf("failed to delete %s %s: %s", u.GetName(), u.GetKind(), err))
			}
		}
	}
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, ", "))
	}
	return nil
}

func (m *Manager) WaitForResourcesReadiness(ctx context.Context, us []*unstructured.Unstructured) error {
	errChan := make(chan error, len(us))

	for _, u := range us {
		go func(resource *unstructured.Unstructured) {
			errChan <- m.waitForResource(ctx, resource)
		}(u)
	}

	var firstErr error
	for range us {
		if err := <-errChan; err != nil && firstErr == nil {
			firstErr = err
		}
	}

	return firstErr
}

func (m *Manager) waitForResource(ctx context.Context, u *unstructured.Unstructured) error {
	now := time.Now()
	for {
		if time.Since(now) >= config.ReadyTimeout {
			return fmt.Errorf("timeout waiting for %s %s to be ready", u.GetName(), u.GetKind())
		}

		ctxWithTimeout, cancel := context.WithTimeout(ctx, config.ReadyCheckInterval)
		current := &unstructured.Unstructured{}
		current.SetGroupVersionKind(u.GroupVersionKind())
		err := m.client.Get(ctxWithTimeout, client.ObjectKey{Name: u.GetName(), Namespace: u.GetNamespace()}, current)
		cancel()
		if err == nil && m.isResourceReady(current) {
			return nil
		}
		if err != nil && !k8serrors.IsNotFound(err) && ctxWithTimeout.Err() == nil && ctx.Err() == nil {
			return fmt.Errorf("while checking readiness of %s %s: %w", u.GetName(), u.GetKind(), err)
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for %s %s to be ready", u.GetName(), u.GetKind())
		case <-time.After(config.ReadyCheckInterval):
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
	conditions, found, err := unstructured.NestedSlice(u.Object, "status", "conditions")
	if err != nil || !found {
		return false
	}
	var available, progressing bool
	for _, c := range conditions {
		cond, ok := c.(map[string]interface{})
		if !ok {
			continue
		}
		condType, _ := cond["type"].(string)
		condStatus, _ := cond["status"].(string)
		switch condType {
		case "Available":
			available = condStatus == "True"
		case "Progressing":
			progressing = condStatus == "True"
		}
	}
	return available && progressing
}
