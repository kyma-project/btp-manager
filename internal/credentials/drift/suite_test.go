package drift_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/kyma-project/btp-manager/internal/credentials/drift"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	kymaNamespace = "kyma-system"

	sapBtpManagerSecretName = "sap-btp-manager"
	operandName             = "sap-btp-operator"

	managedByLabelKey = "app.kubernetes.io/managed-by"
	instanceLabelKey  = "app.kubernetes.io/instance"
	operatorName      = "btp-manager"

	clusterIdSecretKey            = "cluster_id"
	credentialsNamespaceSecretKey = "credentials_namespace"
	initialClusterIdSecretKey     = "INITIAL_CLUSTER_ID"
	clusterIdConfigMapKey         = "CLUSTER_ID"

	previousClusterIdAnnotationKey            = "operator.kyma-project.io/previous-cluster-id"
	previousCredentialsNamespaceAnnotationKey = "operator.kyma-project.io/previous-credentials-namespace"
)

var scheme *runtime.Scheme

func TestDriftDetector(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Drift Detector Suite")
}

var _ = BeforeSuite(func() {
	scheme = runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
})

func newFakeClient(objects ...client.Object) client.Client {
	return fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(objects...).
		Build()
}

func btpManagerSecret(clusterID, credentialsNamespace string, annotations map[string]string) *corev1.Secret {
	s := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:        sapBtpManagerSecretName,
			Namespace:   kymaNamespace,
			Annotations: annotations,
		},
		Data: map[string][]byte{
			clusterIdSecretKey: []byte(clusterID),
		},
	}
	if credentialsNamespace != "" {
		s.Data[credentialsNamespaceSecretKey] = []byte(credentialsNamespace)
	}
	return s
}

func operatorSecret(namespace string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      drift.SapBtpServiceOperatorSecretName,
			Namespace: namespace,
			Labels:    map[string]string{managedByLabelKey: operatorName},
		},
	}
}

func clusterIdSecret(namespace, initialClusterID string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      drift.SapBtpServiceOperatorClusterIdSecretName,
			Namespace: namespace,
		},
		Data: map[string][]byte{
			initialClusterIdSecretKey: []byte(initialClusterID),
		},
	}
}

func operatorConfigMap(clusterID string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      drift.SapBtpServiceOperatorConfigMapName,
			Namespace: kymaNamespace,
		},
		Data: map[string]string{
			clusterIdConfigMapKey: clusterID,
		},
	}
}

func operatorPod(name, namespace string, ready corev1.ConditionStatus) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    map[string]string{instanceLabelKey: operandName},
		},
		Status: corev1.PodStatus{
			Conditions: []corev1.PodCondition{
				{Type: corev1.PodReady, Status: ready},
			},
		},
	}
}

// errorOnGetClient wraps a real client and forces Get calls to fail.
type errorOnGetClient struct {
	client.Client
}

func newErrorOnGetClient(inner client.Client) client.Client {
	return &errorOnGetClient{Client: inner}
}

func (c *errorOnGetClient) Get(_ context.Context, _ client.ObjectKey, _ client.Object, _ ...client.GetOption) error {
	return fmt.Errorf("simulated Get error")
}

// errorOnListClient wraps a real client and forces List calls to fail.
type errorOnListClient struct {
	client.Client
}

func newErrorOnListClient(inner client.Client) client.Client {
	return &errorOnListClient{Client: inner}
}

func (c *errorOnListClient) List(_ context.Context, _ client.ObjectList, _ ...client.ListOption) error {
	return fmt.Errorf("simulated List error")
}

// errorOnUpdateClient wraps a real client and forces Update calls to fail.
type errorOnUpdateClient struct {
	client.Client
}

func newErrorOnUpdateClient(inner client.Client) client.Client {
	return &errorOnUpdateClient{Client: inner}
}

func (c *errorOnUpdateClient) Update(_ context.Context, _ client.Object, _ ...client.UpdateOption) error {
	return fmt.Errorf("simulated Update error")
}

// errorOnDeleteClient wraps a real client and forces Delete calls to fail.
type errorOnDeleteClient struct {
	client.Client
}

func newErrorOnDeleteClient(inner client.Client) client.Client {
	return &errorOnDeleteClient{Client: inner}
}

func (c *errorOnDeleteClient) Delete(_ context.Context, _ client.Object, _ ...client.DeleteOption) error {
	return fmt.Errorf("simulated Delete error")
}
