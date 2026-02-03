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

		Describe("Create ConfigMap", func() {
			Context("when the ConfigMap does not exist", func() {
				It("should create the ConfigMap", func() {
					configmap := configmap()

					Expect(configmapManager.Create(context.Background(), configmap)).To(Succeed())

					got := &corev1.ConfigMap{}
					Expect(fakeClient.Get(context.Background(), client.ObjectKeyFromObject(configmap), got)).To(Succeed())
					Expect(got).To(Equal(configmap))
				})
			})

			Context("when the ConfigMap already exists", func() {
				It("should return error while creating the ConfigMap", func() {
					existingConfigmap := configmap()
					configmap := existingConfigmap.DeepCopy()
					Expect(fakeClient.Create(context.Background(), existingConfigmap)).To(Succeed())

					err := configmapManager.Create(context.Background(), configmap)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("already exists"))
				})
			})
		})

		Describe("Apply ConfigMap", func() {
			const fieldOwner = "object-manager-test"

			Context("when the ConfigMap does not exist", func() {
				It("should apply the ConfigMap", func() {
					configmap := configmap()

					Expect(configmapManager.Apply(context.Background(), configmap, client.FieldOwner(fieldOwner))).To(Succeed())

					got := &corev1.ConfigMap{}
					Expect(fakeClient.Get(context.Background(), client.ObjectKeyFromObject(configmap), got)).To(Succeed())
					Expect(got).To(Equal(configmap))
				})
			})
		})
	})
})

func configmap() *corev1.ConfigMap {
	return &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      configmapName,
			Namespace: namespace,
		},
	}
}
