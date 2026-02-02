package generic_test

import (
	"context"
	"testing"

	"github.com/kyma-project/btp-manager/internal/k8s/generic"
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
	namespace     = "kyma-system"
	configmapName = "test-configmap"
)

var (
	fakeClient client.Client
	scheme     *runtime.Scheme
)

func TestObjectManager(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Object Manager Suite")
}

var _ = BeforeSuite(func() {
	scheme = runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	fakeClient = fake.NewClientBuilder().
		WithScheme(scheme).
		Build()
})

var _ = Describe("Object Manager", func() {
	var (
		configmapManager *generic.ObjectManager[*corev1.ConfigMap]
	)

	BeforeEach(func() {
		fakeClient = fake.NewClientBuilder().
			WithScheme(scheme).
			Build()
	})

	Describe("for ConfigMaps", func() {

		BeforeEach(func() {
			configmapManager = generic.NewObjectManager[*corev1.ConfigMap](fakeClient)
		})

		Describe("Create a ConfigMap", func() {
			Context("when the ConfigMap does not exist", func() {
				It("should create the ConfigMap", func() {
					configmap := &corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Name:      configmapName,
							Namespace: namespace,
						},
					}

					Expect(configmapManager.Create(context.Background(), configmap)).To(Succeed())

					got := &corev1.ConfigMap{}
					Expect(fakeClient.Get(context.Background(), client.ObjectKeyFromObject(configmap), got)).To(Succeed())
					Expect(got).To(Equal(configmap))
				})
			})
		})

	})
})
