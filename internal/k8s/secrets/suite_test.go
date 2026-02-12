package secrets_test

import (
	"testing"

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
	requiredSecretName                       = "sap-btp-manager"
	kymaNamespace                            = "kyma-system"
	sapBtpServiceOperatorClusterIdSecretName = "sap-btp-operator-clusterid"
	caServerCertSecretName                   = "ca-server-cert"
	webhookServerCertSecretName              = "webhook-server-cert"
)

var (
	fakeClient client.Client
	scheme     *runtime.Scheme
)

func TestSecretsManager(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Secrets Manager Suite")
}

var _ = BeforeSuite(func() {
	scheme = runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	fakeClient = fake.NewClientBuilder().
		WithScheme(scheme).
		Build()
})

func requiredSecret() *corev1.Secret {
	return credsSecretWithNameAndNamespace(requiredSecretName, kymaNamespace)
}

func credsSecretWithNameAndNamespace(name, namespace string) *corev1.Secret {
	secret := secretWithNameAndNamespaceManagedByBtpManager(name, namespace)
	data := map[string][]byte{
		"clientid":       []byte("dGVzdF9jbGllbnRpZA=="),
		"clientsecret":   []byte("dGVzdF9jbGllbnRzZWNyZXQ="),
		"sm_url":         []byte("dGVzdF9zbV91cmw="),
		"tokenurl":       []byte("dGVzdF90b2tlbnVybA=="),
		"tokenurlsuffix": []byte("L29hdXRoL3Rva2Vu"),
		"cluster_id":     []byte("dGVzdF9jbHVzdGVyX2lk"),
	}
	secret.Data = data
	return secret
}

func secretWithNameAndNamespaceManagedByBtpManager(name, namespace string) *corev1.Secret {
	secret := secretWithNameAndNamespace(name, namespace)
	labels := map[string]string{
		"app.kubernetes.io/managed-by": "btp-manager",
	}
	secret.Labels = labels
	return secret
}

func secretWithNameAndNamespace(name, namespace string) *corev1.Secret {
	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Type: corev1.SecretTypeOpaque,
	}
}
