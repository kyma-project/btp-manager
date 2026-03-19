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
	"github.com/kyma-project/btp-manager/internal/k8s/secrets"
	"github.com/kyma-project/btp-manager/internal/manifest"
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

	KubeRbacProxyName         = "kube-rbac-proxy"
	KubeRbacProxyEnv          = "KUBE_RBAC_PROXY"
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
	kubeRbacProxyContainerName         = KubeRbacProxyName

	operatorLabelPrefix                       = "operator.kyma-project.io/"
	previousCredentialsNamespaceAnnotationKey = operatorLabelPrefix + "previous-credentials-namespace"
)

type Metadata struct {
	Kind string
	Name string
}

type credentialsContext struct {
	previousCredentialsNamespace                        string
	clusterIdFromSapBtpManagerSecret                    string
	clusterIdFromSapBtpServiceOperatorConfigMap         string
	clusterIdFromSapBtpServiceOperatorClusterIdSecret   string
	credentialsNamespaceFromSapBtpManagerSecret         string
	credentialsNamespaceFromSapBtpServiceOperatorSecret string
}

type Manager struct {
	client             client.Client
	scheme             *runtime.Scheme
	manifestHandler    *manifest.Handler
	resourceIndices    map[Metadata]int
	credentialsContext credentialsContext
	secretsManager     secrets.Manager
}

func NewManager(client client.Client, scheme *runtime.Scheme, secretsManager secrets.Manager) *Manager {
	return &Manager{
		client:          client,
		scheme:          scheme,
		manifestHandler: &manifest.Handler{Scheme: scheme},
		resourceIndices: make(map[Metadata]int),
		secretsManager:  secretsManager,
	}
}

func (m *Manager) GetResourceIndexByMetadata(metadata Metadata) (int, error) {
	index, ok := m.resourceIndices[metadata]
	if !ok {
		return 0, fmt.Errorf(fmt.Sprintf("%s/%s resource index does not exist", metadata.Kind, metadata.Name))
	}
	return index, nil
}

func (m *Manager) SetCredentialsContext(s *corev1.Secret) {
	m.SetClusterID(s)
	m.SetCredentialsNamespace(s)
}

func (m *Manager) SetCredentialsNamespace(s *corev1.Secret) {
	credentialsNamespace := config.ChartNamespace
	if s != nil {
		if v, ok := s.Data[CredentialsNamespaceSecretKey]; ok && len(v) > 0 {
			credentialsNamespace = string(v)
		}
		m.credentialsContext.previousCredentialsNamespace = s.Annotations[previousCredentialsNamespaceAnnotationKey]
	}
	m.credentialsContext.credentialsNamespaceFromSapBtpManagerSecret = credentialsNamespace
}

func (m *Manager) SetClusterID(s *corev1.Secret) {
	m.credentialsContext.clusterIdFromSapBtpManagerSecret = string(s.Data[ClusterIdSecretKey])
}

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
	for i, u := range unstructuredObjects {
		resource := Metadata{
			Kind: u.GetKind(),
			Name: u.GetName(),
		}
		m.resourceIndices[resource] = i
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

func (m *Manager) SetConfigMapValues(secret *corev1.Secret, u *unstructured.Unstructured) error {
	if err := unstructured.SetNestedField(u.Object, string(secret.Data[ClusterIdSecretKey]), "data", clusterIdConfigMapKey); err != nil {
		return fmt.Errorf("failed to set cluster_id: %w", err)
	}

	if err := unstructured.SetNestedField(u.Object, m.credentialsContext.credentialsNamespaceFromSapBtpManagerSecret, "data", releaseNamespaceConfigMapKey); err != nil {
		return fmt.Errorf("failed to set release namespace: %w", err)
	}

	if err := unstructured.SetNestedField(u.Object, m.credentialsContext.credentialsNamespaceFromSapBtpManagerSecret, "data", managementNamespaceConfigMapKey); err != nil {
		return fmt.Errorf("failed to set management namespace: %w", err)
	}

	if err := unstructured.SetNestedField(u.Object, config.EnableLimitedCache, "data", enableLimitedCacheConfigMapKey); err != nil {
		return fmt.Errorf("failed to set enable limited cache: %w", err)
	}

	return nil
}

func (m *Manager) SetSecretValues(secret *corev1.Secret, u *unstructured.Unstructured) error {
	u.SetNamespace(m.credentialsContext.credentialsNamespaceFromSapBtpManagerSecret)

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
	sapBtpServiceOperatorImage, exists := os.LookupEnv(SapBtpServiceOperatorEnv)
	if exists {
		if err := m.setContainerImage(u, sapBtpServiceOperatorContainerName, sapBtpServiceOperatorImage); err != nil {
			return fmt.Errorf("failed to set container image for %s: %w", SapBtpServiceOperatorName, err)
		}
	}
	kubeRbacProxyImage, exists := os.LookupEnv(KubeRbacProxyEnv)
	if exists {
		if err := m.setContainerImage(u, kubeRbacProxyContainerName, kubeRbacProxyImage); err != nil {
			return fmt.Errorf("failed to set container image for %s: %w", kubeRbacProxyContainerName, err)
		}
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

func (m *Manager) applyModuleResources(ctx context.Context) error {
	objects, err := m.CreateUnstructuredObjectsFromManifestsDir(resourcesToApplyPath())
	if err != nil {
		return nil
	}

	return m.ApplyOrUpdateResources(ctx, objects)
}

func resourcesToApplyPath() string {
	return fmt.Sprintf("%s%capply", config.ResourcesPath, os.PathSeparator)
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
	if err := m.client.Patch(ctx, u, client.Apply, client.ForceOwnership, client.FieldOwner(OperatorName)); err != nil {
		return fmt.Errorf("while applying %s %s: %w", u.GetName(), u.GetKind(), err)
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
	objects, err := m.CreateUnstructuredObjectsFromManifestsDir(resourcesToDeletePath())
	if err != nil {
		return nil
	}

	return m.deleteResources(ctx, objects)
}

func resourcesToDeletePath() string {
	return fmt.Sprintf("%s%cdelete", config.ResourcesPath, os.PathSeparator)
}

func (m *Manager) deleteResources(ctx context.Context, us []*unstructured.Unstructured) error {
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
	ctx, cancel := context.WithTimeout(ctx, config.ReadyTimeout)
	defer cancel()

	errChan := make(chan error, len(us))

	for _, u := range us {
		go func(resource *unstructured.Unstructured) {
			errChan <- m.waitForResource(ctx, resource)
		}(u)
	}

	for range us {
		if err := <-errChan; err != nil {
			return err
		}
	}

	return nil
}

func (m *Manager) waitForResource(ctx context.Context, u *unstructured.Unstructured) error {
	ticker := time.NewTicker(config.ReadyCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for %s %s to be ready", u.GetName(), u.GetKind())
		case <-ticker.C:
			current := &unstructured.Unstructured{}
			current.SetGroupVersionKind(u.GroupVersionKind())

			if err := m.client.Get(ctx, client.ObjectKey{Name: u.GetName(), Namespace: u.GetNamespace()}, current); err != nil {
				if k8serrors.IsNotFound(err) {
					continue
				}
				return fmt.Errorf("while checking readiness of %s %s: %w", u.GetName(), u.GetKind(), err)
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
