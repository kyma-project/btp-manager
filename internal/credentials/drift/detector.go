package drift

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/go-logr/logr"
	"github.com/kyma-project/btp-manager/controllers/config"
	"github.com/kyma-project/btp-manager/internal/conditions"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	ClusterIdConfigMapKey = "CLUSTER_ID"

	SapBtpServiceOperatorSecretName          = "sap-btp-service-operator"
	SapBtpServiceOperatorClusterIdSecretName = operandName + "-clusterid"
	SapBtpServiceOperatorConfigMapName       = operandName + "-config"

	clusterIdSecretKey            = "cluster_id"
	credentialsNamespaceSecretKey = "credentials_namespace"
	initialClusterIdSecretKey     = "INITIAL_CLUSTER_ID"

	operatorName = "btp-manager"
	operandName  = "sap-btp-operator"

	sapBtpServiceOperatorSecretName          = "sap-btp-service-operator"
	sapBtpServiceOperatorClusterIdSecretName = operandName + "-clusterid"
	sapBtpServiceOperatorConfigMapName       = operandName + "-config"

	operatorLabelPrefix                       = "operator.kyma-project.io/"
	previousClusterIdAnnotationKey            = operatorLabelPrefix + "previous-cluster-id"
	previousCredentialsNamespaceAnnotationKey = operatorLabelPrefix + "previous-credentials-namespace"

	managedByLabelKey = "app.kubernetes.io/managed-by"
	instanceLabelKey  = "app.kubernetes.io/instance"
)

type Detector interface {
	InitializeFromSecret(s *corev1.Secret)
	CredentialsNamespaceFromManager() string
	CredentialsNamespaceFromOperator() string
	ClusterIdFromManager() string
	ClusterIdFromOperatorConfigMap() string
	ClusterIdFromOperatorClusterIdSecret() string
	PreviousCredentialsNamespace() string
	CheckCredentialsNamespaceDrift(ctx context.Context, requiredSecret *corev1.Secret) *conditions.ErrorWithReason
	CheckClusterIdConfigMapDrift(ctx context.Context, requiredSecret *corev1.Secret) *conditions.ErrorWithReason
	ResolveClusterIdSecretDrift(ctx context.Context, requiredSecret *corev1.Secret) *conditions.ErrorWithReason
	GetDefaultCredentialsSecret(ctx context.Context) (*corev1.Secret, error)
	GetSapBtpServiceOperatorConfigMap(ctx context.Context) (*corev1.ConfigMap, error)
	DeleteClusterIdSecret(ctx context.Context) error
	// DeleteChangedResources must be called after InitializeFromSecret and the Check* methods;
	// it operates on state accumulated by those calls.
	DeleteChangedResources(ctx context.Context) error
}

type DriftDetector struct {
	client          client.Client
	apiServerClient client.Client

	previousCredentialsNamespace                        string
	clusterIdFromSapBtpManagerSecret                    string
	clusterIdFromSapBtpServiceOperatorConfigMap         string
	clusterIdFromSapBtpServiceOperatorClusterIdSecret   string
	credentialsNamespaceFromSapBtpManagerSecret         string
	credentialsNamespaceFromSapBtpServiceOperatorSecret string
}

func NewDetector(k8sClient client.Client, apiServerClient client.Client) *DriftDetector {
	return &DriftDetector{
		client:          k8sClient,
		apiServerClient: apiServerClient,
	}
}

var _ Detector = (*DriftDetector)(nil)

func (d *DriftDetector) InitializeFromSecret(s *corev1.Secret) {
	credentialsNamespace := config.ChartNamespace
	if s != nil {
		if v, ok := s.Data[credentialsNamespaceSecretKey]; ok && len(v) > 0 {
			credentialsNamespace = string(v)
		}
		d.clusterIdFromSapBtpManagerSecret = string(s.Data[clusterIdSecretKey])
		d.previousCredentialsNamespace = s.Annotations[previousCredentialsNamespaceAnnotationKey]
	}
	d.credentialsNamespaceFromSapBtpManagerSecret = credentialsNamespace
	d.credentialsNamespaceFromSapBtpServiceOperatorSecret = credentialsNamespace
}

func (d *DriftDetector) CredentialsNamespaceFromManager() string {
	return d.credentialsNamespaceFromSapBtpManagerSecret
}

func (d *DriftDetector) CredentialsNamespaceFromOperator() string {
	return d.credentialsNamespaceFromSapBtpServiceOperatorSecret
}

func (d *DriftDetector) ClusterIdFromManager() string {
	return d.clusterIdFromSapBtpManagerSecret
}

func (d *DriftDetector) ClusterIdFromOperatorConfigMap() string {
	return d.clusterIdFromSapBtpServiceOperatorConfigMap
}

func (d *DriftDetector) ClusterIdFromOperatorClusterIdSecret() string {
	return d.clusterIdFromSapBtpServiceOperatorClusterIdSecret
}

func (d *DriftDetector) PreviousCredentialsNamespace() string {
	return d.previousCredentialsNamespace
}

func (d *DriftDetector) SetClusterIdFromOperatorConfigMap(id string) {
	d.clusterIdFromSapBtpServiceOperatorConfigMap = id
}

func (d *DriftDetector) CheckCredentialsNamespaceDrift(ctx context.Context, requiredSecret *corev1.Secret) *conditions.ErrorWithReason {
	logger := log.FromContext(ctx)

	defaultCredentialsSecret, err := d.GetDefaultCredentialsSecret(ctx)
	if err != nil {
		logger.Error(err, fmt.Sprintf("while getting %s secret", sapBtpServiceOperatorSecretName))
		return conditions.NewErrorWithReason(conditions.GettingDefaultCredentialsSecretFailed, err.Error())
	}

	if defaultCredentialsSecret != nil {
		d.credentialsNamespaceFromSapBtpServiceOperatorSecret = defaultCredentialsSecret.Namespace
		if d.credentialsNamespaceFromSapBtpManagerSecret != d.credentialsNamespaceFromSapBtpServiceOperatorSecret {
			logger.Info(fmt.Sprintf("credentials namespaces between %s secret and %s secret don't match", config.SecretName, sapBtpServiceOperatorSecretName))
			if err := d.annotateSecret(ctx, requiredSecret, previousCredentialsNamespaceAnnotationKey, d.credentialsNamespaceFromSapBtpServiceOperatorSecret); err != nil {
				return conditions.NewErrorWithReason(conditions.AnnotatingSecretFailed, err.Error())
			}
		}
	}

	return nil
}

func (d *DriftDetector) CheckClusterIdConfigMapDrift(ctx context.Context, requiredSecret *corev1.Secret) *conditions.ErrorWithReason {
	logger := log.FromContext(ctx)

	sapBtpOperatorConfigMap, err := d.GetSapBtpServiceOperatorConfigMap(ctx)
	if err != nil {
		logger.Error(err, fmt.Sprintf("while getting %s ConfigMap", sapBtpServiceOperatorConfigMapName))
		return conditions.NewErrorWithReason(conditions.GettingSapBtpServiceOperatorConfigMapFailed, err.Error())
	}

	if sapBtpOperatorConfigMap != nil {
		d.clusterIdFromSapBtpServiceOperatorConfigMap = sapBtpOperatorConfigMap.Data[strings.ToUpper(clusterIdSecretKey)]
		d.clusterIdFromSapBtpServiceOperatorClusterIdSecret = d.clusterIdFromSapBtpServiceOperatorConfigMap
		if d.clusterIdFromSapBtpManagerSecret != d.clusterIdFromSapBtpServiceOperatorConfigMap {
			logger.Info(fmt.Sprintf("cluster IDs between %s secret and %s configmap don't match", config.SecretName, sapBtpServiceOperatorConfigMapName))
			if err := d.annotateSecret(ctx, requiredSecret, previousClusterIdAnnotationKey, d.clusterIdFromSapBtpServiceOperatorConfigMap); err != nil {
				return conditions.NewErrorWithReason(conditions.AnnotatingSecretFailed, err.Error())
			}
		}
	}

	return nil
}

func (d *DriftDetector) ResolveClusterIdSecretDrift(ctx context.Context, requiredSecret *corev1.Secret) *conditions.ErrorWithReason {
	logger := log.FromContext(ctx)

	clusterIdSecret, err := d.getSecretByNameAndNamespace(ctx, sapBtpServiceOperatorClusterIdSecretName, d.credentialsNamespaceFromSapBtpServiceOperatorSecret)
	if err != nil {
		logger.Error(err, fmt.Sprintf("while getting %s secret", sapBtpServiceOperatorClusterIdSecretName))
		return conditions.NewErrorWithReason(conditions.GettingSapBtpServiceOperatorClusterIdSecretFailed, err.Error())
	}

	if clusterIdSecret != nil {
		if clusterIdFromSecret, ok := clusterIdSecret.Data[initialClusterIdSecretKey]; ok && len(clusterIdFromSecret) > 0 {
			d.clusterIdFromSapBtpServiceOperatorClusterIdSecret = string(clusterIdFromSecret)
		}
		if d.clusterIdFromSapBtpServiceOperatorConfigMap != d.clusterIdFromSapBtpServiceOperatorClusterIdSecret {
			logger.Info(fmt.Sprintf("cluster IDs between %s configmap and %s secret don't match", sapBtpServiceOperatorConfigMapName, sapBtpServiceOperatorClusterIdSecretName))
			if err = d.annotateSecret(ctx, requiredSecret, previousClusterIdAnnotationKey, d.clusterIdFromSapBtpServiceOperatorClusterIdSecret); err != nil {
				logger.Error(err, fmt.Sprintf("while annotating %s secret", requiredSecret.Name))
				return conditions.NewErrorWithReason(conditions.AnnotatingSecretFailed, err.Error())
			}
			logger.Info(fmt.Sprintf("deleting %s secret from %s namespace due to invalid cluster ID", clusterIdSecret.Name, clusterIdSecret.Namespace))
			if err = d.deleteObject(ctx, clusterIdSecret); err != nil {
				logger.Error(err, fmt.Sprintf("while deleting %s secret", clusterIdSecret.Name))
				return conditions.NewErrorWithReason(conditions.DeletionOfOrphanedResourcesFailed, err.Error())
			}
			if err = d.restartSapBtpServiceOperatorPodIfNotReady(ctx, logger); err != nil {
				return conditions.NewErrorWithReason(conditions.ResourceRemovalFailed, fmt.Sprintf("while restarting SAP BTP service operator pod: %s", err))
			}
		}
	}

	return nil
}

func (d *DriftDetector) DeleteChangedResources(ctx context.Context) error {
	clusterIdSecret, err := d.getSecretByNameAndNamespace(ctx, sapBtpServiceOperatorClusterIdSecretName, d.credentialsNamespaceFromSapBtpServiceOperatorSecret)
	if err != nil {
		return err
	}
	pod, err := d.getSapBtpServiceOperatorPod(ctx)
	if err != nil {
		return err
	}
	credentialsSecret, err := d.getSecretByNameAndNamespace(ctx, sapBtpServiceOperatorSecretName, d.credentialsNamespaceFromSapBtpServiceOperatorSecret)
	if err != nil {
		return err
	}

	isCredentialsNamespaceChanged := d.credentialsNamespaceFromSapBtpServiceOperatorSecret != "" &&
		d.credentialsNamespaceFromSapBtpManagerSecret != d.credentialsNamespaceFromSapBtpServiceOperatorSecret

	isClusterIdChanged := d.clusterIdFromSapBtpServiceOperatorConfigMap != "" &&
		(d.clusterIdFromSapBtpManagerSecret != d.clusterIdFromSapBtpServiceOperatorConfigMap ||
			d.clusterIdFromSapBtpServiceOperatorConfigMap != d.clusterIdFromSapBtpServiceOperatorClusterIdSecret)

	if isCredentialsNamespaceChanged || isClusterIdChanged {
		if clusterIdSecret != nil {
			if err = d.deleteObject(ctx, clusterIdSecret); err != nil {
				return err
			}
		}
		if pod != nil {
			if err = d.deleteObject(ctx, pod); err != nil {
				return err
			}
		}
	}

	if isCredentialsNamespaceChanged && credentialsSecret != nil {
		if err = d.deleteObject(ctx, credentialsSecret); err != nil {
			return err
		}
	}

	return nil
}

func (d *DriftDetector) DeleteClusterIdSecret(ctx context.Context) error {
	clusterIdSecret, err := d.getSecretByNameAndNamespace(ctx, sapBtpServiceOperatorClusterIdSecretName, d.credentialsNamespaceFromSapBtpManagerSecret)
	if err != nil {
		return fmt.Errorf("failed to get cluster ID secret: %w", err)
	}
	if clusterIdSecret != nil {
		if err := d.deleteObject(ctx, clusterIdSecret); err != nil {
			return fmt.Errorf("failed to delete cluster ID secret: %w", err)
		}
	}
	return nil
}

func (d *DriftDetector) GetDefaultCredentialsSecret(ctx context.Context) (*corev1.Secret, error) {
	var defaultCredentialsSecret *corev1.Secret
	secrets := &corev1.SecretList{}
	if err := d.client.List(ctx, secrets, client.MatchingLabels{managedByLabelKey: operatorName}); err != nil {
		return nil, fmt.Errorf("unable to list managed secrets: %w", err)
	}
	if len(secrets.Items) == 0 {
		return nil, nil
	}
	// Sort by namespace so that, when more than one operator secret exists, the
	// fallback pick is deterministic rather than dependent on List ordering.
	sort.Slice(secrets.Items, func(i, j int) bool {
		return secrets.Items[i].Namespace < secrets.Items[j].Namespace
	})
	for i, s := range secrets.Items {
		if s.Name != sapBtpServiceOperatorSecretName {
			continue
		}
		if s.Namespace == d.previousCredentialsNamespace {
			return &secrets.Items[i], nil
		}
		if defaultCredentialsSecret == nil {
			defaultCredentialsSecret = &secrets.Items[i]
		}
	}
	return defaultCredentialsSecret, nil
}

func (d *DriftDetector) GetSapBtpServiceOperatorConfigMap(ctx context.Context) (*corev1.ConfigMap, error) {
	cm := &corev1.ConfigMap{}
	if err := d.client.Get(ctx, client.ObjectKey{Namespace: config.ChartNamespace, Name: sapBtpServiceOperatorConfigMapName}, cm); err != nil {
		if k8serrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return cm, nil
}

func (d *DriftDetector) annotateSecret(ctx context.Context, s *corev1.Secret, key, value string) error {
	annotations := s.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}
	if annotations[key] == value {
		return nil
	}
	annotations[key] = value
	s.SetAnnotations(annotations)
	return d.client.Update(ctx, s, client.FieldOwner(operatorName))
}

func (d *DriftDetector) getSecretByNameAndNamespace(ctx context.Context, name, namespace string) (*corev1.Secret, error) {
	secret := &corev1.Secret{}
	if err := d.apiServerClient.Get(ctx, client.ObjectKey{Name: name, Namespace: namespace}, secret); err != nil {
		if k8serrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("while getting %s secret from %s namespace: %w", name, namespace, err)
	}
	return secret, nil
}

func (d *DriftDetector) getSapBtpServiceOperatorPod(ctx context.Context) (*corev1.Pod, error) {
	var pod *corev1.Pod
	pods := &corev1.PodList{}
	if err := d.apiServerClient.List(ctx, pods, client.MatchingLabels{instanceLabelKey: operandName}); err != nil {
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

func (d *DriftDetector) deleteObject(ctx context.Context, obj client.Object) error {
	if err := d.apiServerClient.Delete(ctx, obj); err != nil {
		return fmt.Errorf("while deleting %s %s from %s namespace: %w", obj.GetName(), obj.GetObjectKind().GroupVersionKind().Kind, obj.GetNamespace(), err)
	}
	return nil
}

func (d *DriftDetector) restartSapBtpServiceOperatorPodIfNotReady(ctx context.Context, logger logr.Logger) error {
	pod, err := d.getSapBtpServiceOperatorPod(ctx)
	if err != nil {
		logger.Error(err, "while getting SAP BTP service operator pod")
		return err
	}
	if pod != nil {
		for _, condition := range pod.Status.Conditions {
			if condition.Type == corev1.PodReady && condition.Status == corev1.ConditionFalse {
				if err := d.deleteObject(ctx, pod); err != nil {
					logger.Error(err, fmt.Sprintf("while deleting not ready %s pod", pod.Name))
					return err
				}
				break
			}
		}
	}
	return nil
}
