package secrets_test

import (
	"context"
	"testing"

	"github.com/kyma-project/btp-manager/internal/k8s/secrets"
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
	requiredSecretName = "sap-btp-manager"
	kymaNamespace      = "kyma-system"
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

var _ = Describe("Secrets Manager", func() {
	var mgr *secrets.Manager

	BeforeEach(func() {
		fakeClient = fake.NewClientBuilder().
			WithScheme(scheme).
			Build()

		mgr = secrets.NewManager(fakeClient)
	})

	Describe("Required sap-btp-manager secret", func() {
		Context("when the secret exists", func() {
			It("should return the secret", func() {
				expectedSecret := requiredSecret()
				Expect(fakeClient.Create(context.Background(), expectedSecret)).To(Succeed())

				actualSecret, err := mgr.GetRequiredSecret(context.Background())
				Expect(err).ToNot(HaveOccurred())

				Expect(actualSecret).To(Equal(expectedSecret))
			})
		})

		Context("when the secret does not exist", func() {
			It("should return an error", func() {
				_, err := mgr.GetRequiredSecret(context.Background())

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("not found"))
			})
		})
	})
})

func requiredSecret() *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      requiredSecretName,
			Namespace: kymaNamespace,
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"clientid":     []byte("dGVzdF9jbGllbnRpZA=="),
			"clientsecret": []byte("dGVzdF9jbGllbnRzZWNyZXQ="),
			"sm_url":       []byte("dGVzdF9zbV91cmw="),
			"tokenurl":     []byte("dGVzdF90b2tlbnVybA=="),
			"cluster_id":   []byte("dGVzdF9jbHVzdGVyX2lk"),
		},
	}
}
